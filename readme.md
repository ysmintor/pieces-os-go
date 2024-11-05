# 项目简介
![img](https://raw.githubusercontent.com/pieces-app/pieces-os-client-sdk-for-csharp/main/assets/pieces-logo.png)

逆向Pieces-OS GRPC流并转换为标准OpenAI接口的项目

所有模型均由 Pieces-OS 提供

本项目基于GPLV3协议开源

如果帮助到了你，能否给一个Star呢？ 

# 免责声明
本项目仅供学习交流使用，不得用于商业用途，如有侵权请联系删除

# DEMO站
**请善待公共服务，尽量自己搭建**

[Vercel](https://pieces.nekomoon.cc)

[Cloudflare worker反代koyeb](https://pieces.464888.xyz)

# 一键部署
[![Deploy on Vercel](https://vercel.com/button)](https://vercel.com/new/clone?repository-url=https://github.com/Nekohy/pieces-os&project-name=Pieces-OS&repository-name=Pieces-OS)

[![Deploy to Koyeb](https://www.koyeb.com/static/images/deploy/button.svg)](https://app.koyeb.com/deploy?name=pieces-os&type=docker&image=chb2024%2Fpieces-os%3Alatest&regions=was&env%5B%5D=&ports=8787%3Bhttp%3B%2F)

请注意下列环境变量！私人使用请添加API_KEY！

cloudflare work反代koyeb
```javascript
export default {
  async fetch(request, env) {
    // 创建目标 URL 改成你自己的部署地址，不带协议头和/
    const url = new URL(request.url);
    url.hostname = 'abcdefg.koyeb.app';
    
    // 创建新的请求对象
    const newRequest = new Request(url, {
      method: request.method,
      headers: request.headers,
      body: request.method === 'POST' ? request.body : null,
      redirect: 'follow'
    });

    // 转发请求并返回响应
    return fetch(newRequest);
  }
}
```

# todo
- [x] 流式实现
- [x] Serverless部署
- [x] Docker支持
- [x] Go语言重构
- [x] 实现同时将日志写入文件(V8)
- [x] 实现GPT和Claude的Tokens计算(V9)

# 项目结构
```
pieces-os-go/                          # 项目根目录
  cmd/                                 # 可执行文件目录
    server/                           
      main.go                         # 主程序入口
  internal/                           # 内部包目录
    config/                           # 配置相关
      config.go                       # 配置结构和加载逻辑
    handler/                          # HTTP处理器
      chat.go                         # 聊天相关接口处理
      health.go                       # 健康检查接口
      models.go                       # 模型相关接口处理
    middleware/                       # 中间件
      auth.go                         # 认证中间件
      cors.go                         # 跨域处理
      logger.go                       # 日志中间件
    model/                            # 数据模型
      chat.go                         # 聊天相关数据结构
      error.go                        # 错误定义
      models.go                       # 模型相关数据结构
    service/                          # 业务逻辑层
      chat.go                         # 聊天业务逻辑
      grpc.go                         # GRPC客户端实现
  pkg/                                # 公共包目录
    proto/                            # 协议定义和生成的代码
      gpt/                           # GPT相关协议
        gpt.pb.go                    # 生成的GPT协议代码
        gpt.proto                    # GPT协议定义
        gpt_grpc.pb.go              # 生成的GPT GRPC代码
      vertex/                        # Vertex AI相关协议
        vertex.pb.go                # 生成的Vertex协议代码
        vertex.proto                # Vertex协议定义
        vertex_grpc.pb.go           # 生成的Vertex GRPC代码
    tiktoken_loader/                 # GPT分词器加载器
      assets/                       
        assets.go                   # 资源文件
        cl100k_base.tiktoken       # GPT3.5分词器数据
        o200k_base.tiktoken        # GPT4分词器数据
        tokenizer.json             # 分词器配置
      offline_loader.go             # 离线加载实现
    tokenizer/                      # 通用分词器接口
      claude.go                     # Claude分词实现
      errors.go                     # 错误定义
      models.go                     # 分词器模型
      num.go                        # Token计数实现
  protos/                           # 原始协议定义
    GPTInferenceService.proto      # GPT服务协议
    VertexInferenceService.proto   # Vertex服务协议
  cloud_model.json                  # 云端模型配置
  go.mod                            # Go模块定义
  go.sum                            # 依赖版本锁定
  readme.md                         # 项目说明文档

# 测试可用模型

## Claude 系列(Nextchat可将@换为-)
- **claude-3-5-sonnet@20240620**
- **claude-3-haiku@20240307**
- **claude-3-sonnet@20240229**
- **claude-3-opus@20240229**

## GPT 系列
- **gpt-3.5-turbo**
- **gpt-4**
- **gpt-4-turbo**
- **gpt-4o-mini**
- **gpt-4o**

## Gemini 系列
- **gemini-pro**
- **gemini-1.5-flash**
- **gemini-1.5-pro**

## 其他
- **chat-bison**
- **codechat-bison**

# 手动部署
1. 克隆项目
```bash
git clone https://github.com/wisdgod/pieces-os-go.git
cd pieces-os-go
```

2. 安装依赖
```bash
go mod download
```

3. 运行项目
```bash
go run cmd/server/main.go
```

# 测试命令
```bash
# 获取模型列表
curl --request GET 'http://127.0.0.1:8787/v1/models' \
  --header 'Content-Type: application/json'
```

```bash
# 发送聊天请求
curl --request POST 'http://127.0.0.1:8787/v1/chat/completions' \
  --header 'Content-Type: application/json' \
  --data '{
    "messages": [
      {
        "role": "user",
        "content": "你好！"
      }
    ],
    "model": "gpt-4o",
    "stream": true
  }'
```

# 环境变量
## `API_PREFIX`
- **描述**: API 请求的前缀路径
- **默认值**: `'/v1/'`
- **环境变量**: `API_PREFIX`

## `API_KEY`
- **描述**: API 请求的密钥
- **默认值**: `''`
- **环境变量**: `API_KEY`

## `MAX_RETRIES`
- **描述**: 最大重试次数
- **默认值**: `3`
- **环境变量**: `MAX_RETRIES`

## `TIMEOUT`
- **描述**: 请求超时时间(秒)
- **默认值**: `30`
- **环境变量**: `TIMEOUT`

## `PORT`
- **描述**: 服务监听的端口
- **默认值**: `8787`
- **环境变量**: `PORT`

## `DEBUG`
- **描述**: 是否启用调试模式
- **默认值**: `false`
- **环境变量**: `DEBUG`

## `DEFAULT_MODEL`
- **描述**: 当请求的模型不存在时重定向到的默认模型
- **默认值**: `''`（空字符串，表示拒绝不存在的模型请求）
- **环境变量**: `DEFAULT_MODEL`

## `LOG_FILE`
- **描述**: 日志文件路径，同时将日志写入该文件（v8版本新增）
- **默认值**: `''`（空字符串，表示仅输出到控制台）
- **环境变量**: `LOG_FILE`
- **示例值**: `/var/log/pieces-os.log` 或 `pieces-os.log`

## `MIN_POOL_SIZE`
- **描述**: gRPC连接池最小连接数
- **默认值**: `5`
- **环境变量**: `MIN_POOL_SIZE`

## `MAX_POOL_SIZE`
- **描述**: gRPC连接池最大连接数
- **默认值**: `20`
- **环境变量**: `MAX_POOL_SIZE`

## `SCALE_INTERVAL`
- **描述**: 连接池扩缩容检查间隔(秒)
- **默认值**: `30`
- **环境变量**: `SCALE_INTERVAL`

## `ENABLE_MODEL_ROUTE`
- **描述**: 是否启用模型路由功能（v19版本新增）
- **默认值**: `false`
- **环境变量**: `ENABLE_MODEL_ROUTE`
- **说明**: 启用后可通过 `/{model_name}/v1/chat/completions` 格式直接访问指定模型

# Docker 部署说明

### 使用 Docker Compose（推荐）

1. 创建部署目录：
```bash
mkdir pieces-os-go && cd pieces-os-go
```

2. 下载配置文件：
```bash
# 下载 docker-compose.yml
curl -O https://raw.githubusercontent.com/wisdgod/pieces-os-go/main/docker-compose.yml

# 创建环境变量文件
cat > .env << EOF
API_KEY=your_api_key_here
PORT=8787
API_PREFIX=/v1/
MIN_POOL_SIZE=5
MAX_POOL_SIZE=20
SCALE_INTERVAL=30
DEBUG=false
DEFAULT_MODEL=
MAX_RETRIES=3
TIMEOUT=30
EOF
```

3. 启动服务：
```bash
docker-compose up -d
```

4. 查看日志：
```bash
docker-compose logs -f
```

5. 停止服务：
```bash
docker-compose down
```

### 使用 Docker 命令

1. 拉取镜像：
```bash
docker pull wisdgod/pieces-os-go:latest
```

2. 运行容器：
```bash
docker run -d \
  --name pieces-os \
  -p 8787:8787 \
  -e API_KEY=your_api_key_here \
  -e TZ=Asia/Shanghai \
  -v pieces_logs:/app/logs \
  --restart unless-stopped \
  --cpus=1 \
  --memory=1g \
  --security-opt no-new-privileges:true \
  --log-driver json-file \
  --log-opt max-size=10m \
  --log-opt max-file=3 \
  wisdgod/pieces-os-go:latest
```

3. 管理容器：
```bash
# 查看日志
docker logs -f pieces-os

# 停止容器
docker stop pieces-os

# 启动容器
docker start pieces-os

# 重启容器
docker restart pieces-os

# 删除容器
docker rm -f pieces-os
```

### 构建自定义镜像

如果需要自定义构建，可以使用项目提供的 Dockerfile：

```bash
# 克隆项目
git clone https://github.com/wisdgod/pieces-os-go.git
cd pieces-os-go

# 构建镜像
docker build -t pieces-os-go \
  --build-arg VERSION=$(git describe --tags --always) \
  --build-arg BUILD_TIME=$(date -u +'%Y-%m-%d_%H:%M:%S') \
  .

# 运行自构建镜像
docker run -d \
  --name pieces-os \
  -p 8787:8787 \
  -e API_KEY=your_api_key_here \
  pieces-os-go
```

### 容器资源限制
- CPU: 默认限制 1 核心，最小保证 0.25 核心
- 内存: 默认限制 1GB，最小保证 256MB
- 日志: 自动轮转，单个文件最大 10MB，保留 3 个文件

### 安全特性
- 使用非 root 用户运行
- 禁止容器获取新权限
- 启用健康检查
- 自动重启策略

### 支持的架构
- linux/amd64
- linux/arm64
- freebsd/amd64 (实验性支持)

# V16 更新内容
- 实现版本信息
  - 启动时打印版本信息和构建时间
- 实现gRPC连接池管理
  - 动态扩缩容
  - 自动清理失效连接
  - 连接复用提升性能
  - 可配置连接池参数
- 优化日志处理
  - 使用缓冲写入和互斥锁
- 增强健康检查接口
  - 返回每分钟和每秒请求数

# V17 更新内容
- 新增 Docker 支持
  - 提供官方 Docker 镜像
  - 支持多架构部署（amd64/arm64）
  - 添加 Dockerfile 和 docker-compose.yml
  - 优化容器运行环境
- 完善部署文档
  - 添加 Docker 部署指南
  - 更新环境变量说明
  - 补充容器管理命令

# V19 更新内容
- 新增模型路由功能
  - 支持通过URL直接指定模型
  - 格式: `/{model_name}/v1/chat/completions`
  - Claude模型支持 `-` 和 `@` 两种分隔符
  - 可通过环境变量控制是否启用
- 优化错误处理
  - 添加404路由不存在提示
  - 完善错误信息本地化
- 代码结构优化
  - 重构路由配置逻辑
  - 提升代码可维护性
