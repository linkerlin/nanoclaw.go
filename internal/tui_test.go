package internal

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
)

func TestTUIXSkipLLM_New(t *testing.T) {
	if !HasLLMConfig() {
		t.Skip("No LLM config, skipping")
	}
	db := TestTempDB(t)
	cfg := TestConfig(t)
	queue := NewGroupQueue(5)
	agent := NewAgent(db)

	tui := NewTUI(db, queue, agent, cfg)

	if tui == nil {
		t.Fatal("NewTUI returned nil")
	}

	if tui.cfg != cfg {
		t.Error("TUI config not set correctly")
	}

	if len(tui.groups) == 0 {
		t.Error("Expected at least one group")
	}
}

func TestTUIXSkipLLM_Init(t *testing.T) {
	if !HasLLMConfig() {
		t.Skip("No LLM config, skipping")
	}
	db := TestTempDB(t)
	cfg := TestConfig(t)
	queue := NewGroupQueue(5)
	agent := NewAgent(db)

	tui := NewTUI(db, queue, agent, cfg)

	cmd := tui.Init()
	if cmd == nil {
		t.Error("Init should return a command")
	}
}

func TestTUIXSkipLLM_Update_WindowSize(t *testing.T) {
	if !HasLLMConfig() {
		t.Skip("No LLM config, skipping")
	}
	db := TestTempDB(t)
	cfg := TestConfig(t)
	queue := NewGroupQueue(5)
	agent := NewAgent(db)

	tui := NewTUI(db, queue, agent, cfg)

	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	newTUI, _ := tui.Update(msg)

	updated := newTUI.(*TUI)
	if updated.width != 100 {
		t.Errorf("width = %d, want 100", updated.width)
	}
	if updated.height != 30 {
		t.Errorf("height = %d, want 30", updated.height)
	}
}

func TestTUIXSkipLLM_Update_TUIMsg(t *testing.T) {
	if !HasLLMConfig() {
		t.Skip("No LLM config, skipping")
	}
	db := TestTempDB(t)
	cfg := TestConfig(t)
	queue := NewGroupQueue(5)
	agent := NewAgent(db)

	tui := NewTUI(db, queue, agent, cfg)

	tuiMsg := TUIMsg{
		ChatJID: "main@nanoclaw",
		Message: Message{
			ID:        MessageID("test-1"),
			ChatJID:   "main@nanoclaw",
			Content:   "Hello",
			Timestamp: time.Now(),
		},
	}

	newTUI, _ := tui.Update(tuiMsg)
	updated := newTUI.(*TUI)

	msgs := updated.messages["main@nanoclaw"]
	if len(msgs) != 1 {
		t.Errorf("Expected 1 message, got %d", len(msgs))
	}

	if msgs[0].Content != "Hello" {
		t.Errorf("Content = %q, want %q", msgs[0].Content, "Hello")
	}
}

func TestTUIXSkipLLM_Update_ThinkingMsg(t *testing.T) {
	if !HasLLMConfig() {
		t.Skip("No LLM config, skipping")
	}
	db := TestTempDB(t)
	cfg := TestConfig(t)
	queue := NewGroupQueue(5)
	agent := NewAgent(db)

	tui := NewTUI(db, queue, agent, cfg)

	msg := ThinkingMsg{
		ChatJID:  "main@nanoclaw",
		Thinking: true,
	}

	newTUI, _ := tui.Update(msg)
	updated := newTUI.(*TUI)

	if !updated.thinking["main@nanoclaw"] {
		t.Error("Expected thinking to be true")
	}

	msg.Thinking = false
	newTUI, _ = updated.Update(msg)
	updated = newTUI.(*TUI)

	if updated.thinking["main@nanoclaw"] {
		t.Error("Expected thinking to be false")
	}
}

func TestTUIXSkipLLM_View(t *testing.T) {
	if !HasLLMConfig() {
		t.Skip("No LLM config, skipping")
	}
	db := TestTempDB(t)
	cfg := TestConfig(t)
	queue := NewGroupQueue(5)
	agent := NewAgent(db)

	tui := NewTUI(db, queue, agent, cfg)

	tui.width = 100
	tui.height = 30

	view := tui.View()

	if view == "" {
		t.Error("View should not be empty")
	}

	if view == "Loading..." {
		t.Error("View should not be loading state")
	}
}

func TestTUIXSkipLLM_View_Empty(t *testing.T) {
	if !HasLLMConfig() {
		t.Skip("No LLM config, skipping")
	}
	db := TestTempDB(t)
	cfg := TestConfig(t)
	queue := NewGroupQueue(5)
	agent := NewAgent(db)

	tui := NewTUI(db, queue, agent, cfg)

	view := tui.View()

	if view != "Loading..." {
		t.Errorf("View = %q, want 'Loading...'", view)
	}
}

func TestTUIXSkipLLM_SetOnSend(t *testing.T) {
	if !HasLLMConfig() {
		t.Skip("No LLM config, skipping")
	}
	db := TestTempDB(t)
	cfg := TestConfig(t)
	queue := NewGroupQueue(5)
	agent := NewAgent(db)

	tui := NewTUI(db, queue, agent, cfg)

	tui.SetOnSend(func(chatJID ChatJID, content string) {
		// callback set
	})

	if tui.onSend == nil {
		t.Error("onSend callback not set")
	}
}

func TestTUIXSkipLLM_currentChatJID(t *testing.T) {
	if !HasLLMConfig() {
		t.Skip("No LLM config, skipping")
	}
	db := TestTempDB(t)
	cfg := TestConfig(t)
	queue := NewGroupQueue(5)
	agent := NewAgent(db)

	tui := NewTUI(db, queue, agent, cfg)

	jid := tui.currentChatJID()
	if jid != "main@nanoclaw" {
		t.Errorf("currentChatJID = %q, want 'main@nanoclaw'", jid)
	}
}

func TestTUIXSkipLLM_renderMessages(t *testing.T) {
	if !HasLLMConfig() {
		t.Skip("No LLM config, skipping")
	}
	db := TestTempDB(t)
	cfg := TestConfig(t)
	queue := NewGroupQueue(5)
	agent := NewAgent(db)

	tui := NewTUI(db, queue, agent, cfg)

	chatJID := ChatJID("test@nanoclaw")

	rendered := tui.renderMessages(chatJID)
	if rendered == "" {
		t.Error("renderMessages should return non-empty string for empty messages")
	}

	tui.messages[chatJID] = []Message{
		{
			ID:         MessageID("msg-1"),
			ChatJID:    chatJID,
			SenderName: "User",
			Content:    "Hello",
			Timestamp:  time.Now(),
		},
		{
			ID:           MessageID("msg-2"),
			ChatJID:      chatJID,
			SenderName:   "Bot",
			Content:      "Hi there!",
			Timestamp:    time.Now(),
			IsBotMessage: true,
		},
	}

	rendered = tui.renderMessages(chatJID)
	if rendered == "" {
		t.Error("renderMessages should return non-empty string")
	}
}
