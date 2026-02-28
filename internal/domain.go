package internal

import (
	"regexp"
	"strings"
	"time"
)

// MessageID 消息ID
type MessageID string

// ChatJID 聊天会话ID
type ChatJID string

// Message 消息领域模型
type Message struct {
	ID           MessageID
	ChatJID      ChatJID
	Sender       string
	SenderName   string
	Content      string
	Timestamp    time.Time
	IsFromMe     bool
	IsBotMessage bool
	Metadata     map[string]string
}

// HasTrigger 检查消息是否包含触发词
func (m *Message) HasTrigger(pattern *regexp.Regexp) bool {
	return pattern.MatchString(m.Content)
}

// IsSystem 检查是否为系统命令
func (m *Message) IsSystem() bool {
	return strings.HasPrefix(m.Content, "/")
}

// Group 群组/会话模型
type Group struct {
	JID             ChatJID
	Name            string
	Folder          string
	TriggerPattern  string
	RequiresTrigger bool
	AddedAt         time.Time
}

// Session 会话状态
type Session struct {
	GroupFolder string
	SessionID   string
	UpdatedAt   time.Time
}

// Task 定时任务
type Task struct {
	ID            string
	GroupFolder   string
	ChatJID       ChatJID
	Prompt        string
	ScheduleType  string // cron/interval/once
	ScheduleValue string
	NextRun       *time.Time
	LastRun       *time.Time
	LastResult    string
	Status        string // active/paused/completed
	CreatedAt     time.Time
}

// IsDue 检查任务是否到期
func (t *Task) IsDue(now time.Time) bool {
	if t.NextRun == nil || t.Status != "active" {
		return false
	}
	return !now.Before(*t.NextRun)
}

// StreamEvent 流式响应事件
type StreamEvent struct {
	Content string
	Err     error
	Done    bool
}
