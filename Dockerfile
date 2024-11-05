# 构建阶段
FROM golang:1.23.2-bullseye AS builder

WORKDIR /app

# 设置 Go 环境变量
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOPROXY=https://goproxy.io,direct

# 复制依赖文件并下载
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
ARG VERSION
ARG BUILD_TIME

RUN go build -trimpath \
    -ldflags="-s -w -extldflags '-static' -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
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
COPY --from=builder /app/cloud_model.json ./

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
