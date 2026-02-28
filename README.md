# nanoclaw.go

Go语言 1:1 复刻 https://github.com/qwibitai/nanoclaw/

TUI采用 [bubbletea / bubbles](https://github.com/charmbracelet/bubbles) · Agent层采用 [ADK-GO](https://github.com/google/adk-go)

---

## 架构

```
TUI (bubbletea) --> SQLite --> Message loop --> ADK-GO Agent (Gemini) --> Response
```

单 Go 进程，几个源文件，无微服务。Agent 层通过 Google ADK-GO 调用 Gemini 模型，支持每个群组独立的 `CLAUDE.md` 系统指令文件。

## 项目结构

| 文件 | 说明 |
|------|------|
| `cmd/nanoclaw/main.go` | 入口：初始化数据库、TUI、Orchestrator、Scheduler |
| `internal/config/config.go` | 配置（`ASSISTANT_NAME`、路径、触发词正则等） |
| `internal/db/db.go` | SQLite 数据库层（与原版 schema 一致） |
| `internal/types/types.go` | 核心类型定义 |
| `internal/router/router.go` | XML 消息格式化 / 出站内容清理 |
| `internal/queue/queue.go` | 每群组 FIFO 队列 + 全局并发限制 |
| `internal/agent/agent.go` | ADK-GO LLM Agent（Gemini 后端） |
| `internal/scheduler/scheduler.go` | 定时任务调度（cron / interval / once） |
| `internal/tui/tui.go` | 三栏 bubbletea TUI（群组列表 / 消息视图 / 输入框） |
| `internal/orchestrator/orchestrator.go` | 消息循环、触发词检测、Agent 分发 |
| `groups/*/CLAUDE.md` | 每个群组的系统指令文件 |

## 快速开始

```bash
git clone https://github.com/linkerlin/nanoclaw.go
cd nanoclaw.go
export GOOGLE_API_KEY=your_api_key
go run ./cmd/nanoclaw
```

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `GOOGLE_API_KEY` | *(必填)* | Google AI API 密钥 |
| `ASSISTANT_NAME` | `Andy` | 触发词（`@Andy`） |
| `GEMINI_MODEL` | `gemini-2.0-flash` | Gemini 模型名称 |

## TUI 快捷键

| 按键 | 功能 |
|------|------|
| `Tab` | 切换面板（群组列表 → 消息视图 → 输入框） |
| `Enter` | 发送消息（在输入框面板时） |
| `Ctrl+C` | 退出 |

## 消息流

1. 用户在 TUI 输入框中输入消息并按 Enter
2. 消息存入 SQLite
3. 若消息以触发词（如 `@Andy`）开头，Orchestrator 将群组加入 Agent 队列
4. Agent Runner 读取该群组的 `groups/{folder}/CLAUDE.md` 作为系统指令
5. ADK-GO 创建 Gemini LLM Agent 并运行
6. Agent 回复存入 SQLite，TUI 实时刷新显示

## 许可证

MIT

