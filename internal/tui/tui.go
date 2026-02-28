package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/linkerlin/nanoclaw.go/internal/types"
)

// ---- Styles ----------------------------------------------------------------

var (
	headerStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62")).Padding(0, 1)
	sidebarStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	mainStyle      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
	statusStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1)
	thinkingStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Italic(true)
	userMsgStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	agentMsgStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	senderStyle    = lipgloss.NewStyle().Bold(true)
)

// ---- Messages --------------------------------------------------------------

// SendMsg is the bubbletea message that carries a new user message.
type SendMsg struct{ Text string }

// BotResponseMsg carries the agent's reply back to the TUI.
type BotResponseMsg struct {
	ChatJID string
	Text    string
}

// MessagesUpdatedMsg signals that the message list for a chat was refreshed.
type MessagesUpdatedMsg struct {
	ChatJID  string
	Messages []types.NewMessage
}

// ThinkingMsg toggles the "thinking…" indicator.
type ThinkingMsg struct {
	ChatJID  string
	Thinking bool
}

// ---- Group list item -------------------------------------------------------

type groupItem struct {
	group types.RegisteredGroup
}

func (g groupItem) Title() string       { return g.group.Name }
func (g groupItem) Description() string { return g.group.JID }
func (g groupItem) FilterValue() string { return g.group.Name }

// ---- Focus panels ----------------------------------------------------------

type panel int

const (
	panelSidebar panel = iota
	panelMain
	panelInput
)

// ---- Model -----------------------------------------------------------------

// Model is the bubbletea model for the full TUI.
type Model struct {
	width, height int

	groups    []types.RegisteredGroup
	groupList list.Model

	// Per-group state.
	messages  map[string][]types.NewMessage
	thinking  map[string]bool
	viewports map[string]viewport.Model

	input       textarea.Model
	activePanel panel

	// Callback invoked when the user submits a message.
	OnSend func(chatJID, text string)
}

// New initialises the TUI model.
func New(groups []types.RegisteredGroup, onSend func(chatJID, text string)) Model {
	// Group list.
	items := make([]list.Item, len(groups))
	for i, g := range groups {
		items[i] = groupItem{g}
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Groups"
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	// Textarea.
	ta := textarea.New()
	ta.Placeholder = "Type a message… (Enter to send, Ctrl+C to quit)"
	ta.Focus()
	ta.CharLimit = 2000
	ta.SetWidth(60)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false

	return Model{
		groups:    groups,
		groupList: l,
		messages:  make(map[string][]types.NewMessage),
		thinking:  make(map[string]bool),
		viewports: make(map[string]viewport.Model),
		input:     ta,
		OnSend:    onSend,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalcLayout()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.activePanel = (m.activePanel + 1) % 3
			if m.activePanel == panelInput {
				m.input.Focus()
				cmds = append(cmds, textarea.Blink)
			} else {
				m.input.Blur()
			}
		case "enter":
			if m.activePanel == panelInput {
				text := strings.TrimSpace(m.input.Value())
				if text != "" && m.OnSend != nil {
					chatJID := m.selectedChatJID()
					if chatJID != "" {
						m.OnSend(chatJID, text)
					}
				}
				m.input.Reset()
				return m, nil
			}
		}

	case MessagesUpdatedMsg:
		m.messages[msg.ChatJID] = msg.Messages
		if vp, ok := m.viewports[msg.ChatJID]; ok {
			vp.SetContent(m.renderMessages(msg.ChatJID))
			vp.GotoBottom()
			m.viewports[msg.ChatJID] = vp
		}

	case ThinkingMsg:
		m.thinking[msg.ChatJID] = msg.Thinking
		if vp, ok := m.viewports[msg.ChatJID]; ok {
			vp.SetContent(m.renderMessages(msg.ChatJID))
			vp.GotoBottom()
			m.viewports[msg.ChatJID] = vp
		}
	}

	// Delegate to active panel.
	switch m.activePanel {
	case panelSidebar:
		var listCmd tea.Cmd
		m.groupList, listCmd = m.groupList.Update(msg)
		cmds = append(cmds, listCmd)
	case panelMain:
		chatJID := m.selectedChatJID()
		if vp, ok := m.viewports[chatJID]; ok {
			var vpCmd tea.Cmd
			vp, vpCmd = vp.Update(msg)
			m.viewports[chatJID] = vp
			cmds = append(cmds, vpCmd)
		}
	case panelInput:
		var taCmd tea.Cmd
		m.input, taCmd = m.input.Update(msg)
		cmds = append(cmds, taCmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 {
		return "Initializing…"
	}

	sidebarW := 24
	mainW := m.width - sidebarW - 4 // borders

	// ---- Sidebar ----
	m.groupList.SetWidth(sidebarW - 2)
	m.groupList.SetHeight(m.height - 4)
	sidebar := sidebarStyle.Width(sidebarW).Height(m.height - 2).Render(m.groupList.View())

	// ---- Header ----
	chatJID := m.selectedChatJID()
	groupName := chatJID
	for _, g := range m.groups {
		if g.JID == chatJID {
			groupName = g.Name
			break
		}
	}
	header := headerStyle.Render("nanoclaw.go  ›  " + groupName)

	// ---- Viewport ----
	vpH := m.height - 8 // header + borders + input + status
	vp := m.getViewport(chatJID, mainW-2, vpH)
	vpView := mainStyle.Width(mainW).Height(vpH + 2).Render(vp.View())

	// ---- Input ----
	m.input.SetWidth(mainW - 2)
	inputView := mainStyle.Width(mainW).Render(m.input.View())

	// ---- Status bar ----
	statusText := "Tab: switch panel  Enter: send  Ctrl+C: quit"
	if m.thinking[chatJID] {
		statusText = thinkingStyle.Render("⟳ thinking…") + "  " + statusText
	}
	status := statusStyle.Render(statusText)

	// ---- Right column ----
	right := lipgloss.JoinVertical(lipgloss.Left, header, vpView, inputView, status)

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, right)
}

// ---- Helpers ---------------------------------------------------------------

func (m *Model) selectedChatJID() string {
	if i := m.groupList.Index(); i >= 0 && i < len(m.groups) {
		return m.groups[i].JID
	}
	return ""
}

func (m *Model) getViewport(chatJID string, w, h int) viewport.Model {
	vp, ok := m.viewports[chatJID]
	if !ok {
		vp = viewport.New(w, h)
		vp.SetContent(m.renderMessages(chatJID))
		vp.GotoBottom()
	} else {
		vp.Width = w
		vp.Height = h
	}
	m.viewports[chatJID] = vp
	return vp
}

func (m *Model) recalcLayout() {
	for jid, vp := range m.viewports {
		sidebarW := 24
		mainW := m.width - sidebarW - 4
		vpH := m.height - 8
		vp.Width = mainW - 2
		vp.Height = vpH
		m.viewports[jid] = vp
	}
}

func (m *Model) renderMessages(chatJID string) string {
	msgs := m.messages[chatJID]
	if len(msgs) == 0 && !m.thinking[chatJID] {
		return statusStyle.Render("No messages yet. Type something below!")
	}
	var sb strings.Builder
	for _, msg := range msgs {
		ts := ""
		if t, err := time.Parse(time.RFC3339, msg.Timestamp); err == nil {
			ts = t.Format("15:04")
		}
		sender := senderStyle.Render(msg.SenderName)
		prefix := fmt.Sprintf("[%s] %s: ", ts, sender)
		if msg.IsBotMessage {
			sb.WriteString(agentMsgStyle.Render(prefix + msg.Content))
		} else {
			sb.WriteString(userMsgStyle.Render(prefix + msg.Content))
		}
		sb.WriteString("\n")
	}
	if m.thinking[chatJID] {
		sb.WriteString(thinkingStyle.Render("⟳ thinking…") + "\n")
	}
	return sb.String()
}
