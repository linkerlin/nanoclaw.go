# nanoclaw.go

Go语言复刻 [NanoClaw](https://github.com/qwibitai/nanoclaw/) - 极简AI助手

## 特性

- **极简架构**: ~1700行代码，11个Go文件
- **TUI界面**: Bubbletea v2 三栏布局
- **LLM支持**: OpenAI兼容API（OpenAI/Groq/DeepSeek/Ollama）
- **Skills系统**: Claude SKILL格式 + Gopher-Lua脚本
- **并发控制**: Google官方semaphore
- **用户隔离**: 低权限用户 + Unix Socket
- **定时任务**: Cron/Interval/Once调度

## 快速开始

### 环境变量配置

```bash
# OpenAI官方
export OPENAI_API_KEY="sk-..."
export OPENAI_BASE_URL="https://api.openai.com/v1"
export OPENAI_MODEL="gpt-4o-mini"

# 或 Groq（高速推理）
export OPENAI_API_KEY="gsk-..."
export OPENAI_BASE_URL="https://api.groq.com/openai/v1"
export OPENAI_MODEL="llama-3.3-70b-versatile"

# 或本地Ollama
export OPENAI_API_KEY="ollama"
export OPENAI_BASE_URL="http://localhost:11434/v1"
export OPENAI_MODEL="qwen2.5:14b"
```

### 运行

```bash
go mod tidy
go run ./cmd/nanoclaw
```

### 构建

```bash
make build
./build/nanoclaw
```

### 设置隔离用户（可选）

```bash
make setup
sudo -u nanoclaw ./nanoclaw
```

## 使用

在TUI中：
- `Tab`: 切换面板
- `Enter`: 发送消息
- `Ctrl+C`: 退出

触发词：`@Andy <message>`

## 项目结构

```
nanoclaw.go/
├── cmd/nanoclaw/main.go    # 入口
├── internal/
│   ├── domain.go           # 领域模型
│   ├── config.go           # 配置（含LLM环境变量）
│   ├── db.go               # SQLite
│   ├── queue.go            # Semaphore队列
│   ├── agent.go            # OpenAI Agent
│   ├── scheduler.go        # 定时任务
│   ├── orchestrator.go     # 消息编排
│   ├── tui.go              # Bubbletea v2
│   ├── skills.go           # Skills + Lua
│   └── ipc.go              # Unix Socket
├── skills/builtin/         # 内置Skills
├── groups/main/            # 群组数据
└── data/                   # SQLite数据库
```

## 许可证

MIT
