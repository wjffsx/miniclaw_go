# MiniClaw Go

MiniClaw Go 是 Go 语言实现的AI助手，旨在将 AI 助手 应用在 树莓派平台，提高系统性能、增强扩展性、改善可维护性。

## 项目概述

MiniClaw Go 是一个基于树莓派5和 Go 语言开发的个人 AI 助手，通过 Telegram、WebSocket 或命令行界面与用户交互，使用 Anthropic 的 Claude 模型或 OpenAI 的 GPT 模型进行智能处理，并通过本地存储实现记忆功能。

## 特性

- **多通道通信**：支持 Telegram、WebSocket 和命令行界面
- **本地记忆存储**：支持长期记忆和每日笔记管理
- **工具调用机制**：支持网络搜索、获取时间、计算、文件操作等工具
- **多模型支持**：支持 Anthropic、OpenAI 等多个 LLM 提供商，可根据需求动态切换
- **本地模型运行**：支持在树莓派5上运行小型本地 LLM（通过 llama.cpp）
- **ReAct 循环**：支持多轮对话和智能工具调用
- **HTTP 代理支持**：支持通过 HTTP 代理访问外部服务
- **性能优化**：连接池、请求重试、速率限制和性能监控
- **完整测试**：单元测试和集成测试覆盖所有核心模块

## 项目结构

```
miniclaw/
├── cmd/                    # 主程序入口
├── configs/               # 配置文件目录
│   └── config.example.yaml
├── internal/              # 内部包
│   ├── agent/            # Agent 服务（ReAct循环实现）
│   ├── bus/              # 消息总线（事件驱动架构）
│   ├── communication/     # 通信模块
│   │   ├── telegram/     # Telegram 机器人
│   │   ├── websocket/    # WebSocket 服务
│   │   └── cli/         # 命令行界面
│   ├── config/           # 配置服务
│   ├── context/          # 上下文构建器（系统提示、记忆、工具文档）
│   ├── filetools/        # 文件操作工具
│   ├── integration/       # 集成测试
│   ├── llm/            # LLM 服务
│   │   ├── anthropic.go  # Anthropic Claude 集成
│   │   ├── openai.go    # OpenAI GPT 集成
│   │   ├── local.go     # 本地模型支持（llama.cpp）
│   │   ├── multi.go     # 多模型管理器
│   │   ├── monitor.go   # 性能监控
│   │   ├── ratelimit.go # 速率限制
│   │   └── scanner.go   # 模型扫描器
│   ├── search/          # 搜索服务
│   │   └── brave.go     # Brave Search API 集成
│   ├── storage/         # 存储服务
│   │   ├── 文件系统存储
│   │   ├── 会话存储
│   │   └── 记忆存储
│   └── tools/           # 工具模块
│       ├── base.go       # 工具基础接口
│       ├── builtin.go    # 内置工具（时间、计算、回显）
│       ├── file.go       # 文件操作工具
│       └── tools.go     # 工具注册表和执行器
├── Dockerfile          # Docker 配置
├── Makefile           # 构建脚本
├── go.mod            # Go 模块文件
└── README.md         # 项目文档
```

## 快速开始

### 前置要求

- 树莓派5（4GB 或 8GB 内存配置）
- Go 1.22 或更高版本
- Raspbian OS（最新版）

### 安装

1. 克隆仓库：
```bash
git clone https://github.com/wjffsx/miniclaw_go.git
cd miniclaw_go
```

2. 安装依赖：
```bash
go mod download
```

3. 配置系统：
```bash
cp configs/config.example.yaml configs/config.yaml
# 编辑 configs/config.yaml，填入你的配置信息
```

4. 构建并运行：
```bash
go build -o bin/miniclaw_go cmd/main.go
./bin/miniclaw_go
```

或使用 Makefile：
```bash
make build
make run
```

### Docker 部署

```bash
docker build -t miniclaw_go .
docker run -d --name miniclaw_go -p 8080:8080 miniclaw_go
```

## 配置

配置文件位于 `configs/config.yaml`，包含以下主要配置项：

- **通信配置**：Telegram Bot、WebSocket 服务、CLI 配置
- **LLM 配置**：API 密钥、模型选择、温度、最大令牌数
- **存储配置**：数据目录、会话存储、记忆存储
- **工具配置**：网络搜索 API、文件操作路径
- **代理配置**：HTTP 代理地址和端口
- **性能配置**：连接池大小、重试次数、速率限制

### 配置示例

```yaml
communication:
  telegram:
    enabled: true
    bot_token: "your_telegram_bot_token"
  websocket:
    enabled: true
    port: 8080
  cli:
    enabled: true

llm:
  provider: "anthropic"
  models:
    - name: "claude"
      provider: "anthropic"
      api_key: "your_anthropic_api_key"
      model: "claude-3-5-sonnet"
      max_tokens: 4096
      temperature: 0.7
    - name: "gpt"
      provider: "openai"
      api_key: "your_openai_api_key"
      model: "gpt-4"
      max_tokens: 4096
      temperature: 0.7
  default_model: "claude"

storage:
  data_dir: "./data"
  memory_file: "memory/MEMORY.md"
  daily_notes_dir: "memory/notes"

tools:
  web_search:
    enabled: true
    api_key: "your_brave_search_api_key"
  file_operations:
    enabled: true
    base_path: "./data/files"

proxy:
  enabled: false
  http_proxy: "http://proxy.example.com:8080"
```

## 核心功能

### Agent 服务

Agent 服务实现了 ReAct（Reasoning + Acting）循环，支持：

- 多轮对话管理
- 智能工具调用
- 上下文感知响应
- 迭代优化

### 工具系统

内置工具：

- **get_time**：获取当前时间
- **echo**：回显输入内容
- **calculate**：执行数学计算
- **web_search**：网络搜索（Brave Search）
- **read_file**：读取文件内容
- **write_file**：写入文件
- **list_dir**：列出目录内容
- **delete_file**：删除文件或目录
- **add_memory**：添加长期记忆
- **search_memory**：搜索记忆
- **add_daily_note**：添加每日笔记
- **get_recent_notes**：获取最近的笔记

自定义工具：

可以通过实现 `Tool` 接口来添加自定义工具：

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() json.RawMessage
    Execute(ctx context.Context, params map[string]interface{}) (string, error)
}
```

### 多模型管理

支持多个 LLM 提供商和模型：

- **Anthropic Claude**：claude-3-5-sonnet、claude-3-haiku
- **OpenAI GPT**：gpt-4、gpt-3.5-turbo
- **本地模型**：通过 llama.cpp 运行小型模型

动态切换模型：

```go
err := multiModelManager.SwitchModel("gpt")
```

### 性能优化

- **连接池**：复用 HTTP 连接
- **请求重试**：自动重试失败的请求
- **速率限制**：防止 API 限流
- **性能监控**：跟踪请求延迟和成功率

## 开发

### 运行测试

运行所有测试：
```bash
go test ./...
```

运行特定包的测试：
```bash
go test ./internal/agent/...
go test ./internal/tools/...
```

运行测试并显示覆盖率：
```bash
go test -cover ./...
```

### 代码风格

项目遵循 Go 语言的代码风格规范，使用 `gofmt` 格式化代码：

```bash
gofmt -w .
```

使用 `golint` 检查代码：
```bash
golint ./...
```

### 代码审查

所有代码必须经过代码审查才能合并到主分支。

## 测试

项目包含完整的测试覆盖：

- **单元测试**：每个模块都有对应的单元测试
- **集成测试**：测试各模块之间的交互
- **端到端测试**：测试完整的用户流程

运行集成测试：
```bash
go test ./internal/integration/...
```

## 部署

### 树莓派部署

1. 交叉编译：
```bash
GOARCH=arm64 GOOS=linux go build -o bin/miniclaw_go cmd/main.go
```

2. 复制到树莓派：
```bash
scp bin/miniclaw_go pi@raspberrypi:/home/pi/
```

3. 在树莓派上运行：
```bash
chmod +x miniclaw_go
./miniclaw_go
```

### Docker 部署

```bash
docker build -t miniclaw_go .
docker run -d --name miniclaw_go \
  -v $(pwd)/data:/app/data \
  -p 8080:8080 \
  miniclaw_go
```

## 故障排除

### 常见问题

1. **API 密钥错误**：检查配置文件中的 API 密钥是否正确
2. **连接超时**：检查网络连接和代理配置
3. **内存不足**：减少最大令牌数或使用更小的模型
4. **文件权限**：确保数据目录有读写权限

### 日志

应用日志输出到标准输出，可以通过以下方式查看：

```bash
./miniclaw_go 2>&1 | tee app.log
```

## 贡献

欢迎贡献！请阅读 [CONTRIBUTING.md](CONTRIBUTING.md) 了解如何参与项目。

### 贡献流程

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件。

## 致谢

MiniClaw Go 项目受到以下项目的启发：

- [MimiClaw](https://github.com/memovai/mimiclaw) - 原始 ESP32-S3 版本
- [OpenClaw](https://github.com/openclaw/openclaw) - AI 助手架构
- [Nanobot](https://github.com/HKUDS/nanobot) - Python AI 代理实现
- [llama.cpp](https://github.com/ggerganov/llama.cpp) - 本地 LLM 推理引擎

## 联系方式

- 项目主页：[https://github.com/wjffsx/miniclaw_go](https://github.com/wjffsx/miniclaw_go)
- 问题反馈：[Issues](https://github.com/wjffsx/miniclaw_go/issues)
- 讨论区：[Discussions](https://github.com/wjffsx/miniclaw_go/discussions)

## 路线图

### v1.1.0（计划中）

- [ ] 支持更多 LLM 提供商（Google Gemini、Mistral）
- [ ] 添加语音输入/输出支持
- [ ] 实现插件系统
- [ ] 添加 Web 管理界面
- [ ] 支持分布式部署

### v1.2.0（规划中）

- [ ] 多用户支持
- [ ] 权限管理
- [ ] 数据加密
- [ ] 备份和恢复功能
