# 构建阶段
FROM golang:1.23.2-bookworm AS builder

# 添加架构参数
ARG TARGETARCH

WORKDIR /app

# 设置 Go 环境变量
ENV GO111MODULE=on \
    CGO_ENABLED=1 \
    GOOS=linux \
    GOPROXY=direct \
    GOPRIVATE=github.com/wisdgod/grpc-go

# 安装构建依赖
RUN apt-get update && apt-get install -y --no-install-recommends \
    pkg-config \
    libssl-dev \
    && rm -rf /var/lib/apt/lists/*

# 下载并安装 tokenizers
RUN if [ "$TARGETARCH" = "arm64" ]; then \
      TOKENIZERS_URL="https://github.com/daulet/tokenizers/releases/latest/download/libtokenizers.linux-arm64.tar.gz"; \
      # arm64 需要安装交叉编译工具 \
      apt-get update && apt-get install -y gcc-aarch64-linux-gnu g++-aarch64-linux-gnu; \
      export CC=aarch64-linux-gnu-gcc; \
      export CXX=aarch64-linux-gnu-g++; \
    else \
      TOKENIZERS_URL="https://github.com/daulet/tokenizers/releases/latest/download/libtokenizers.linux-amd64.tar.gz"; \
    fi && \
    wget $TOKENIZERS_URL -O tokenizers.tar.gz && \
    tar xzf tokenizers.tar.gz && \
    cp *.{so,a} /usr/local/lib/ 2>/dev/null || true && \
    ldconfig && \
    rm -f tokenizers.tar.gz

# 设置 tokenizers 环境变量
ENV TOKENIZERS_LIB_DIR=/usr/local/lib \
    CGO_LDFLAGS="-L/usr/local/lib -ltokenizers" \
    LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH \
    LIBRARY_PATH=/usr/local/lib:$LIBRARY_PATH

# 复制依赖文件并下载
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
ARG VERSION
ARG BUILD_TIME

RUN go build -trimpath \
    -ldflags="-s -w  -extldflags '-static' -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
    -o pieces-os-go ./cmd/server/.

# 运行阶段
FROM debian:bullseye-slim

# 安装基本工具和运行时依赖
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /app/logs \
    && chown -R 1001:root /app/logs \
    && chmod 755 /app/logs

# 设置时区
ENV TZ=Asia/Shanghai

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/pieces-os-go .

# 创建非 root 用户
RUN useradd -r -u 1001 -g root pieces
RUN chown -R pieces:root /app
USER pieces

# 设置环境变量
ENV PORT=8787 \
    API_PREFIX=/v1/ \
    MIN_POOL_SIZE=5 \
    MAX_POOL_SIZE=20 \
    SCALE_INTERVAL=30 \
    LOG_FILE=/app/logs/pieces-os.log

# 声明卷
VOLUME ["/app/logs"]

# 暴露端口
EXPOSE 8787

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:${PORT}/ping || exit 1

# 运行应用
CMD ["./pieces-os-go"]
