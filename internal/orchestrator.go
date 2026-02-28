package internal

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/google/uuid"
)

// Orchestrator 消息编排器
type Orchestrator struct {
	db       *DB
	queue    *GroupQueue
	agent    *Agent
	cfg      *Config
	program  *tea.Program
	onReply  func(ChatJID, string)
}

// NewOrchestrator 创建编排器
func NewOrchestrator(db *DB, queue *GroupQueue, agent *Agent, cfg *Config) *Orchestrator {
	return &Orchestrator{
		db:    db,
		queue: queue,
		agent: agent,
		cfg:   cfg,
	}
}

// SetProgram 设置TUI程序（用于发送消息）
func (o *Orchestrator) SetProgram(p *tea.Program) {
	o.program = p
}

// SetOnReply 设置回复回调
func (o *Orchestrator) SetOnReply(fn func(ChatJID, string)) {
	o.onReply = fn
}

// HandleMessage 处理用户消息
func (o *Orchestrator) HandleMessage(chatJID ChatJID, sender, content string) {
	// 保存消息
	msg := &Message{
		ID:         MessageID(uuid.New().String()),
		ChatJID:    chatJID,
		Sender:     sender,
		SenderName: sender,
		Content:    content,
		Timestamp:  time.Now(),
	}
	if err := o.db.SaveMessage(msg); err != nil {
		slog.Error("save message", "err", err)
		return
	}

	// 发送给TUI
	if o.program != nil {
		o.program.Send(TUIMsg{ChatJID: chatJID, Message: *msg})
	}

	// 检查触发词
	if o.cfg.App.TriggerPattern.MatchString(content) {
		o.enqueueAgent(context.Background(), chatJID)
	}
}

// enqueueAgent 将Agent任务加入队列
func (o *Orchestrator) enqueueAgent(ctx context.Context, chatJID ChatJID) {
	// 发送思考中状态
	if o.program != nil {
		o.program.Send(ThinkingMsg{ChatJID: chatJID, Thinking: true})
	}

	if err := o.queue.Enqueue(ctx, chatJID, func() {
		o.runAgent(chatJID)
	}); err != nil {
		slog.Error("enqueue agent", "err", err)
		if o.program != nil {
			o.program.Send(ThinkingMsg{ChatJID: chatJID, Thinking: false})
		}
	}
}

// runAgent 运行Agent
func (o *Orchestrator) runAgent(chatJID ChatJID) {
	defer func() {
		if o.program != nil {
			o.program.Send(ThinkingMsg{ChatJID: chatJID, Thinking: false})
		}
	}()

	// 获取历史消息
	messages, err := o.db.GetMessages(chatJID, 20)
	if err != nil {
		slog.Error("get messages", "err", err)
		o.sendReply(chatJID, fmt.Sprintf("Error: %v", err))
		return
	}

	// 获取群组信息
	group, err := o.db.GetGroup(chatJID)
	if err != nil {
		// 使用默认群组
		group = &Group{Folder: "main"}
	}

	// 调用Agent
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	resp, err := o.agent.Run(ctx, group.Folder, messages)
	if err != nil {
		slog.Error("agent run", "err", err)
		o.sendReply(chatJID, fmt.Sprintf("Error: %v", err))
		return
	}

	// 保存回复
	botMsg := &Message{
		ID:           MessageID(uuid.New().String()),
		ChatJID:      chatJID,
		Sender:       o.cfg.App.Name,
		SenderName:   o.cfg.App.Name,
		Content:      resp,
		Timestamp:    time.Now(),
		IsBotMessage: true,
	}
	if err := o.db.SaveMessage(botMsg); err != nil {
		slog.Error("save bot message", "err", err)
	}

	// 发送给TUI
	if o.program != nil {
		o.program.Send(TUIMsg{ChatJID: chatJID, Message: *botMsg})
	}

	// 回调
	o.sendReply(chatJID, resp)
}

func (o *Orchestrator) sendReply(chatJID ChatJID, content string) {
	if o.onReply != nil {
		o.onReply(chatJID, content)
	}
}
