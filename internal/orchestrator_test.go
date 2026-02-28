package internal

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestOrchestratorX_HandleMessage(t *testing.T) {
	if !HasLLMConfig() {
		t.Skip("No LLM config, skipping")
	}
	db := TestTempDB(t)
	cfg := TestConfig(t)
	queue := NewGroupQueue(5)
	agent := NewAgent(db)
	orch := NewOrchestrator(db, queue, agent, cfg)

	chatJID := ChatJID("test@nanoclaw")

	orch.HandleMessage(chatJID, "User", "Hello")

	msgs, err := db.GetMessages(chatJID, 10)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}

	if len(msgs) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(msgs))
	}

	if msgs[0].Content != "Hello" {
		t.Errorf("Content = %q, want %q", msgs[0].Content, "Hello")
	}
}

func TestOrchestratorX_HandleMessage_WithTrigger(t *testing.T) {
	SkipIfNoLLM(t)

	db := TestTempDB(t)
	cfg := TestConfig(t)
	queue := NewGroupQueue(5)
	agent := NewAgent(db)
	orch := NewOrchestrator(db, queue, agent, cfg)

	var replyReceived int32
	var replyContent string
	var mu sync.Mutex

	orch.SetOnReply(func(chatJID ChatJID, content string) {
		atomic.AddInt32(&replyReceived, 1)
		mu.Lock()
		replyContent = content
		mu.Unlock()
	})

	chatJID := ChatJID("main@nanoclaw")

	group := &Group{
		JID:             chatJID,
		Name:            "Main",
		Folder:          "main",
		TriggerPattern:  cfg.App.TriggerPattern.String(),
		RequiresTrigger: true,
		AddedAt:         time.Now(),
	}
	if err := db.SaveGroup(group); err != nil {
		t.Fatalf("SaveGroup failed: %v", err)
	}

	orch.HandleMessage(chatJID, "User", "@Andy Say 'Pong' and nothing else.")

	time.Sleep(10 * time.Second)

	if atomic.LoadInt32(&replyReceived) == 0 {
		t.Error("Expected reply callback to be called")
	}

	mu.Lock()
	if replyContent == "" {
		t.Error("Expected non-empty reply content")
	}
	mu.Unlock()

	msgs, err := db.GetMessages(chatJID, 10)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}

	if len(msgs) < 2 {
		t.Errorf("Expected at least 2 messages, got %d", len(msgs))
	}
}

func TestOrchestratorX_HandleMessage_NoTrigger(t *testing.T) {
	if !HasLLMConfig() {
		t.Skip("No LLM config, skipping")
	}
	db := TestTempDB(t)
	cfg := TestConfig(t)
	queue := NewGroupQueue(5)
	agent := NewAgent(db)
	orch := NewOrchestrator(db, queue, agent, cfg)

	var replyReceived int32
	orch.SetOnReply(func(chatJID ChatJID, content string) {
		atomic.AddInt32(&replyReceived, 1)
	})

	chatJID := ChatJID("test@nanoclaw")

	orch.HandleMessage(chatJID, "User", "Just a regular message without trigger")

	time.Sleep(2 * time.Second)

	if atomic.LoadInt32(&replyReceived) != 0 {
		t.Error("Should not receive reply for non-trigger message")
	}

	msgs, err := db.GetMessages(chatJID, 10)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}

	if len(msgs) != 1 {
		t.Errorf("Expected 1 message, got %d", len(msgs))
	}
}

func TestOrchestratorXSkip_MultipleMessages(t *testing.T) {
	if !HasLLMConfig() {
		t.Skip("No LLM config, skipping")
	}
	db := TestTempDB(t)
	cfg := TestConfig(t)
	queue := NewGroupQueue(5)
	agent := NewAgent(db)
	orch := NewOrchestrator(db, queue, agent, cfg)

	chatJID := ChatJID("test@nanoclaw")

	for i := 0; i < 5; i++ {
		orch.HandleMessage(chatJID, "User", "Message "+string(rune('0'+i)))
	}

	msgs, err := db.GetMessages(chatJID, 10)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}

	if len(msgs) != 5 {
		t.Errorf("Expected 5 messages, got %d", len(msgs))
	}
}

func TestOrchestratorXSkip_ConcurrentMessages(t *testing.T) {
	if !HasLLMConfig() {
		t.Skip("No LLM config, skipping")
	}
	db := TestTempDB(t)
	cfg := TestConfig(t)
	queue := NewGroupQueue(10)
	agent := NewAgent(db)
	orch := NewOrchestrator(db, queue, agent, cfg)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			chatJID := ChatJID("test" + string(rune('0'+idx)) + "@nanoclaw")
			orch.HandleMessage(chatJID, "User", "Concurrent message")
		}(i)
	}

	wg.Wait()

	msgs, err := db.GetMessages("test0@nanoclaw", 10)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}

	if len(msgs) != 1 {
		t.Errorf("Expected 1 message, got %d", len(msgs))
	}
}
