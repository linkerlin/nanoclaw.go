package internal

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/bubbles/v2/textarea"
	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
)

// TUIMsg TUI消息
type TUIMsg struct {
	ChatJID ChatJID
	Message Message
}

// ThinkingMsg 思考中状态
type ThinkingMsg struct {
	ChatJID  ChatJID
	Thinking bool
}

// FocusPane 焦点面板
type FocusPane int

const (
	FocusSidebar FocusPane = iota
	FocusMain
	FocusInput
)

// TUI Bubbletea v2 TUI模型
type TUI struct {
	width, height int
	db            *DB
	queue         *GroupQueue
	agent         *Agent
	cfg           *Config

	groups      []Group
	groupList   list.Model
	messages    map[ChatJID][]Message
	thinking    map[ChatJID]bool
	viewports   map[ChatJID]viewport.Model
	input       textarea.Model
	focus       FocusPane
	onSend      func(ChatJID, string)
}

// NewTUI 创建TUI
func NewTUI(db *DB, queue *GroupQueue, agent *Agent, cfg *Config) *TUI {
	// 加载群组
	groups := []Group{
		{JID: "main@nanoclaw", Name: "Main", Folder: "main", RequiresTrigger: true},
	}

	// 创建群组列表
	items := make([]list.Item, len(groups))
	for i, g := range groups {
		items[i] = groupItem{group: g}
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Groups"
	l.SetShowHelp(false)

	// 创建输入框
	ta := textarea.New()
	ta.Placeholder = "Type a message... (Enter to send, Ctrl+C to quit)"
	ta.Focus()
	ta.CharLimit = 2000
	ta.SetWidth(60)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false

	return &TUI{
		db:        db,
		queue:     queue,
		agent:     agent,
		cfg:       cfg,
		groups:    groups,
		groupList: l,
		messages:  make(map[ChatJID][]Message),
		thinking:  make(map[ChatJID]bool),
		viewports: make(map[ChatJID]viewport.Model),
		input:     ta,
		focus:     FocusInput,
	}
}

// SetOnSend 设置发送回调
func (t *TUI) SetOnSend(fn func(ChatJID, string)) {
	t.onSend = fn
}

// Init 初始化 - v2: 返回 tea.Cmd
func (t *TUI) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
	)
}

// Update 更新 - v2: 返回 (tea.Model, tea.Cmd)
func (t *TUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
		t.recalcLayout()

	case tea.KeyPressMsg:
		// v2: 使用 KeyPressMsg 和 key.String() 或 key.Key().Code
		key := msg.Key()
		switch key.String() {
		case "ctrl+c", "esc":
			return t, tea.Quit
		case "tab":
			t.focus = (t.focus + 1) % 3
			if t.focus == FocusInput {
				t.input.Focus()
			} else {
				t.input.Blur()
			}
		case "enter":
			if t.focus == FocusInput {
				t.sendMessage()
			}
		}

	case TUIMsg:
		if _, ok := t.messages[msg.ChatJID]; !ok {
			t.messages[msg.ChatJID] = []Message{}
		}
		t.messages[msg.ChatJID] = append(t.messages[msg.ChatJID], msg.Message)
		t.updateViewport(msg.ChatJID)

	case ThinkingMsg:
		t.thinking[msg.ChatJID] = msg.Thinking
		t.updateViewport(msg.ChatJID)
	}

	// 委托给子组件
	var cmds []tea.Cmd
	switch t.focus {
	case FocusSidebar:
		m, cmd := t.groupList.Update(msg)
		t.groupList = m
		cmds = append(cmds, cmd)
	case FocusInput:
		m, cmd := t.input.Update(msg)
		t.input = m
		cmds = append(cmds, cmd)
	}

	return t, tea.Batch(cmds...)
}

// View 渲染
func (t *TUI) View() string {
	if t.width == 0 {
		return "Loading..."
	}

	sidebarW := 25
	mainW := t.width - sidebarW - 4

	// 侧边栏
	t.groupList.SetWidth(sidebarW - 2)
	t.groupList.SetHeight(t.height - 4)
	sidebar := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Width(sidebarW).Height(t.height - 2).
		Render(t.groupList.View())

	// 主区域
	chatJID := t.currentChatJID()
	groupName := string(chatJID)
	for _, g := range t.groups {
		if g.JID == chatJID {
			groupName = g.Name
			break
		}
	}

	header := lipgloss.NewStyle().
		Bold(true).Foreground(lipgloss.Color("62")).
		Padding(0, 1).
		Render(fmt.Sprintf("nanoclaw  ›  %s", groupName))

	// 消息视图
	vpH := t.height - 10
	vp := t.getViewport(chatJID, mainW-2, vpH)
	main := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Width(mainW).Height(vpH + 2).
		Render(vp.View())

	// 输入框
	t.input.SetWidth(mainW - 2)
	input := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Width(mainW).Render(t.input.View())

	// 状态栏
	statusText := "Tab: switch  Enter: send  Ctrl+C: quit"
	if t.thinking[chatJID] {
		statusText = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Italic(true).Render("⟳ thinking...") + "  " + statusText
	}
	status := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1).Render(statusText)

	right := lipgloss.JoinVertical(lipgloss.Left, header, main, input, status)
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, right)
}

// Run 运行TUI
func (t *TUI) Run(ctx context.Context) error {
	p := tea.NewProgram(t, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// SendBotMessage 发送机器人消息到TUI
func (t *TUI) SendBotMessage(chatJID ChatJID, content string) {
	msg := Message{
		ID:           MessageID(fmt.Sprintf("bot-%d", time.Now().UnixNano())),
		ChatJID:      chatJID,
		Sender:       "Bot",
		SenderName:   t.cfg.App.Name,
		Content:      content,
		Timestamp:    time.Now(),
		IsBotMessage: true,
	}
	// 这里需要通过Program发送，简化处理
	_ = msg
}

func (t *TUI) sendMessage() {
	text := strings.TrimSpace(t.input.Value())
	if text == "" || t.onSend == nil {
		return
	}

	chatJID := t.currentChatJID()
	t.onSend(chatJID, text)
	t.input.Reset()
}

func (t *TUI) currentChatJID() ChatJID {
	if i := t.groupList.Index(); i >= 0 && i < len(t.groups) {
		return t.groups[i].JID
	}
	return ""
}

func (t *TUI) getViewport(chatJID ChatJID, w, h int) viewport.Model {
	vp, ok := t.viewports[chatJID]
	if !ok {
		// v2: viewport.New 使用 Options
		vp = viewport.New(viewport.WithWidth(w), viewport.WithHeight(h))
		vp.SetContent(t.renderMessages(chatJID))
		vp.GotoBottom()
	} else {
		// v2: 使用 SetWidth/SetHeight
		vp.SetWidth(w)
		vp.SetHeight(h)
	}
	t.viewports[chatJID] = vp
	return vp
}

func (t *TUI) updateViewport(chatJID ChatJID) {
	if vp, ok := t.viewports[chatJID]; ok {
		vp.SetContent(t.renderMessages(chatJID))
		vp.GotoBottom()
		t.viewports[chatJID] = vp
	}
}

func (t *TUI) renderMessages(chatJID ChatJID) string {
	msgs := t.messages[chatJID]
	if len(msgs) == 0 && !t.thinking[chatJID] {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("No messages yet. Type something below!")
	}

	var sb strings.Builder
	for _, m := range msgs {
		ts := m.Timestamp.Format("15:04")
		sender := lipgloss.NewStyle().Bold(true).Render(m.SenderName)
		prefix := fmt.Sprintf("[%s] %s: ", ts, sender)

		content := m.Content
		if m.IsBotMessage {
			content = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Render(prefix + m.Content)
		} else {
			content = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render(prefix + m.Content)
		}
		sb.WriteString(content)
		sb.WriteString("\n")
	}

	if t.thinking[chatJID] {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Italic(true).Render("⟳ thinking..."))
		sb.WriteString("\n")
	}
	return sb.String()
}

func (t *TUI) recalcLayout() {
	for jid, vp := range t.viewports {
		sidebarW := 25
		mainW := t.width - sidebarW - 4
		vpH := t.height - 10
		vp.SetWidth(mainW - 2)
		vp.SetHeight(vpH)
		t.viewports[jid] = vp
	}
}

// groupItem 列表项
type groupItem struct {
	group Group
}

func (g groupItem) Title() string       { return g.group.Name }
func (g groupItem) Description() string { return string(g.group.JID) }
func (g groupItem) FilterValue() string { return g.group.Name }
