# 项目简介
![img](https://raw.githubusercontent.com/pieces-app/pieces-os-client-sdk-for-csharp/main/assets/pieces-logo.png)

逆向Pieces-OS GRPC流并转换为标准OpenAI接口的项目

所有模型均由 Pieces-OS 提供

# 许可证

本项目基于 GNU General Public License v3.0 (GPLv3) 开源。

## 主要权利和限制

- ✅ 商业使用（需遵守开源要求）
- ✅ 修改源代码
- ✅ 分发
- ✅ 私人使用
- ⚠️ 必须开源：任何基于本项目的衍生作品必须以相同的许可证（GPLv3）开源
- ⚠️ 声明变更：必须标注修改过的文件
- ⚠️ 许可证和版权声明：必须包含原始许可证和版权声明
- ⚠️ 相同许可：衍生作品必须使用相同的许可证（GPLv3）

## 版权声明

在每个源代码文件的开头添加：

```
Copyright (C) [年份] [作者名]

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
```

# 免责声明
本项目仅供学习交流使用，不得用于商业用途，如有侵权请联系删除

# DEMO站
**请善待公共服务，尽量自己搭建**

[Koyeb](https://calcpi.wisdgod.com/)


# todo
- [x] 流式实现
- [x] Docker支持
- [x] Go语言重构
- [x] 实现Tokens计算

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
export GOPROXY=direct
export GOPRIVATE=github.com/wisdgod/grpc-go
go mod download
```

3. 运行项目
```bash
# Linux / MacOS
export CGO_ENABLED=1
export CGO_LDFLAGS=-L/path/to/tokenizers -ltokenizers -static
export GOOS=linux
export GOARCH=amd64
go run cmd/server/main.go

# Windows bash
export CGO_ENABLED=1
export CGO_LDFLAGS=-L/path/to/tokenizers -ltokenizers -static -lws2_32 -lbcrypt -luserenv -lntdll
export GOOS=windows
export GOARCH=amd64
go run cmd/server/main.go
```

# 测试命令
```bash
# 获取模型列表
curl --request GET 'http://localhost:8787/v1/models' \
  --header 'Content-Type: application/json'
```

```bash
# 发送聊天请求
curl --request POST 'http://localhost:8787/v1/chat/completions' \
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
- **默认值**: `'/v1'`
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
- **描述**: 是否启用模型路由功能
- **默认值**: `false`
- **环境变量**: `ENABLE_MODEL_ROUTE`
- **说明**: 启用后可通过 `/{model_name}{API_PREFIX}/chat/completions` 格式直接访问指定模型

## `ENABLE_FOOLPROOF_ROUTE`
- **描述**: 是否启用防呆路由功能
- **默认值**: `false`
- **环境变量**: `ENABLE_FOOLPROOF_ROUTE`
- **说明**: 启用后可通过非标准格式访问 `/chat/completions`

## 超时配置
### `REQUEST_TIMEOUT`
- **描述**: 普通请求的超时时间(秒)
- **默认值**: `30`
- **环境变量**: `REQUEST_TIMEOUT`
- **说明**: 非流式请求的最大处理时间，超过此时间将返回超时错误

### `STREAM_TIMEOUT`
- **描述**: 流式请求的超时时间(秒)
- **默认值**: `300`
- **环境变量**: `STREAM_TIMEOUT`
- **说明**: 流式(SSE)请求的最大处理时间，设置为0表示不限制

## 限流配置
### 默认限流器 (default)
- **RATE_LIMIT**: 每个IP在时间窗口内允许的最大请求数（默认: 60）
- **RATE_LIMIT_WINDOW**: 限流时间窗口(秒)（默认: 60）
- **RATE_LIMIT_ENABLED**: 是否启用（默认: true）

### 严格限流器 (strict)
- **STRICT_RATE_LIMIT**: 严格模式下的请求限制（默认: 10）
- **STRICT_RATE_LIMIT_WINDOW**: 严格模式的时间窗口(秒)（默认: 60）
- **STRICT_RATE_LIMIT_ENABLED**: 是否启用严格模式（默认: false）

### 突发限流器 (burst)
- **BURST_RATE_LIMIT**: 突发模式下的请求限制（默认: 100）
- **BURST_RATE_LIMIT_WINDOW**: 突发模式的时间窗口(秒)（默认: 1）
- **BURST_RATE_LIMIT_ENABLED**: 是否启用突发模式（默认: false）

### IP白名单
- **IP_WHITELIST**: IP白名单，多个IP用逗号分隔（默认: 空）
- **示例**: `127.0.0.1,192.168.1.100`
- **说明**: 白名单中的IP不受任何限流规则限制

### 限流说明
1. 系统支持同时启用多个限流规则，请求需要同时满足所有启用的规则才能通过
2. 默认限流器适用于一般场景，每分钟限制60个请求
3. 严格限流器适用于需要更严格控制的场景，每分钟限制10个请求
4. 突发限流器用于防止突发流量，每秒限制100个请求
5. 可以通过环境变量分别控制每个限流器的启用状态

## 黑名单配置
### `BLACKLIST_MODE`
- **描述**: 黑名单模式
- **默认值**: `single`
- **可选值**: 
  - `off`: 关闭自动拉黑
  - `single`: 单个IP拉黑模式
  - `subnet`: IP段拉黑模式（IPv4使用/24，IPv6使用/48）
- **环境变量**: `BLACKLIST_MODE`

### `BLACKLIST_THRESHOLD`
- **描述**: 触发自动拉黑的违规阈值
- **默认值**: `100`
- **环境变量**: `BLACKLIST_THRESHOLD`
- **说明**: 当IP违规次数达到此阈值时会被自动拉黑

### `BLACKLIST_FILE`
- **描述**: 黑名单持久化文件路径
- **默认值**: `blacklist.txt`
- **环境变量**: `BLACKLIST_FILE`
- **说明**: 自动生成的黑名单将保存在此文件中

### `IP_BLACKLIST`
- **描述**: 配置的永久黑名单
- **默认值**: 空
- **环境变量**: `IP_BLACKLIST`
- **格式**: 多个IP用逗号分隔
- **示例**: `1.2.3.4,2.3.4.5,2001:db8::1`
- **说明**: 支持IPv4和IPv6地址

### `IPV4_MASK`
- **描述**: IPv4子网掩码长度
- **默认值**: `24`
- **范围**: `8-32`
- **环境变量**: `IPV4_MASK`
- **说明**: 
  - 在subnet模式下用于确定封禁范围
  - 不建议设置过小的值以避免误封
  - /24对应一个C类网段（256个地址）

### `IPV6_MASK`
- **描述**: IPv6子网掩码长度
- **默认值**: `48`
- **范围**: `32-128`
- **环境变量**: `IPV6_MASK`
- **说明**: 
  - 在subnet模式下用于确定封禁范围
  - /48通常对应一个组织的地址分配
  - 建议不要小于/48以避免过度封禁

### 管理接口配置
#### `ADMIN_KEY`
- **描述**: 管理接口访问密钥
- **默认值**: 自动生成的32-64位随机字符串
- **环境变量**: `ADMIN_KEY`
- **说明**: 
  - 用于访问管理接口的专用密钥，与API_KEY分开管理
  - 如果未设置，系统会自动生成一个随机密钥并在启动时打印到日志
  - 建议在生产环境中手动设置固定值

### 黑名单管理
1. 系统支持两种黑名单：
   - 配置黑名单：通过环境变量配置的永久黑名单
   - 自动黑名单：根据违规次数自动生成的黑名单

2. 黑名单模式：
   - `single`模式：单独封禁违规IP
   - `subnet`模式：封禁违规IP所在的网段（IPv4封禁/24网段，IPv6封禁/48网段）

3. 黑名单文件下载：
   - 端点：`/admin/blacklist`
   - 方法：GET
   - 权限要求：
     - 需要管理密钥（ADMIN_KEY）
     - 在请求头中使用 Bearer 认证
   - 响应：直接下载黑名单文件
   - 安全限制：
     - 需要有效的管理密钥
     - 黑名单IP无法访问此接口
     - 使用Content-Disposition确保文件下载而不是显示

4. 黑名单持久化：
   - 自动生成的黑名单会保存到文件
   - 服务重启时会自动加载保存的黑名单
   - 配置的黑名单优先级高于自动生成的黑名单

### 配置示例
```env
# API访问密钥
API_KEY=your_api_key_here

# 管理接口密钥（如果不设置会自动生成）
ADMIN_KEY=your_admin_key_here

# 黑名单配置
BLACKLIST_MODE=single
BLACKLIST_THRESHOLD=100
BLACKLIST_FILE=blacklist.txt
IP_BLACKLIST=1.2.3.4,2001:db8::1
```

### 使用示例
```bash
# 下载黑名单文件
curl -O -H "Authorization: Bearer your_admin_key_here" http://localhost:8787/admin/blacklist
```

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
API_PREFIX=/v1
MIN_POOL_SIZE=5
MAX_POOL_SIZE=20
SCALE_INTERVAL=30
DEBUG=false
DEFAULT_MODEL=
MAX_RETRIES=3
TIMEOUT=30
ENABLE_MODEL_ROUTE=false
ENABLE_FOOLPROOF_ROUTE=false
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
