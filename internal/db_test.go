package internal

import (
	"testing"
	"time"
)

func TestDB_SaveAndGetMessage(t *testing.T) {
	db := TestTempDB(t)

	msg := &Message{
		ID:           MessageID("test-msg-1"),
		ChatJID:      "test@nanoclaw",
		Sender:       "user1",
		SenderName:   "User One",
		Content:      "Hello, world!",
		Timestamp:    time.Now(),
		IsBotMessage: false,
	}

	if err := db.SaveMessage(msg); err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	msgs, err := db.GetMessages("test@nanoclaw", 10)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}

	if len(msgs) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(msgs))
	}

	got := msgs[0]
	if got.ID != msg.ID {
		t.Errorf("ID = %q, want %q", got.ID, msg.ID)
	}
	if got.Content != msg.Content {
		t.Errorf("Content = %q, want %q", got.Content, msg.Content)
	}
}

func TestDB_GetMessages_Ordering(t *testing.T) {
	db := TestTempDB(t)

	chatJID := ChatJID("test@nanoclaw")
	for i := 0; i < 5; i++ {
		msg := &Message{
			ID:        MessageID("msg-" + string(rune('0'+i)) + "-" + time.Now().Format("20060102150405")),
			ChatJID:   chatJID,
			Content:   "Message " + string(rune('0'+i)),
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
		}
		if err := db.SaveMessage(msg); err != nil {
			t.Fatalf("SaveMessage failed: %v", err)
		}
	}

	msgs, err := db.GetMessages(chatJID, 3)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}

	if len(msgs) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(msgs))
	}

	for i := 0; i < len(msgs)-1; i++ {
		if msgs[i].Timestamp.After(msgs[i+1].Timestamp) {
			t.Error("Messages are not in chronological order")
		}
	}
}

func TestDB_SaveAndGetGroup(t *testing.T) {
	db := TestTempDB(t)

	group := &Group{
		JID:             "main@nanoclaw",
		Name:            "Main",
		Folder:          "main",
		TriggerPattern:  `(?i)^@Andy\b`,
		RequiresTrigger: true,
		AddedAt:         time.Now(),
	}

	if err := db.SaveGroup(group); err != nil {
		t.Fatalf("SaveGroup failed: %v", err)
	}

	got, err := db.GetGroup("main@nanoclaw")
	if err != nil {
		t.Fatalf("GetGroup failed: %v", err)
	}

	if got.Name != group.Name {
		t.Errorf("Name = %q, want %q", got.Name, group.Name)
	}
}

func TestDB_SaveAndGetSession(t *testing.T) {
	db := TestTempDB(t)

	session := &Session{
		GroupFolder: "main",
		SessionID:   "sess-12345",
		UpdatedAt:   time.Now(),
	}

	if err := db.SaveSession(session); err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}

	got, err := db.GetSession("main")
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if got.SessionID != session.SessionID {
		t.Errorf("SessionID = %q, want %q", got.SessionID, session.SessionID)
	}
}

func TestDB_SaveAndGetTask(t *testing.T) {
	db := TestTempDB(t)

	nextRun := time.Now().Add(time.Hour)
	task := &Task{
		ID:            "task-1",
		GroupFolder:   "main",
		ChatJID:       "main@nanoclaw",
		Prompt:        "Send daily report",
		ScheduleType:  "cron",
		ScheduleValue: "0 9 * * *",
		NextRun:       &nextRun,
		Status:        "active",
		CreatedAt:     time.Now(),
	}

	if err := db.SaveTask(task); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	nextNextRun := time.Now().Add(24 * time.Hour)
	if err := db.UpdateTaskRun("task-1", "Success", &nextNextRun); err != nil {
		t.Fatalf("UpdateTaskRun failed: %v", err)
	}
}

func TestDB_GetDueTasks(t *testing.T) {
	db := TestTempDB(t)

	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	task1 := &Task{
		ID:            "due-task",
		GroupFolder:   "main",
		ChatJID:       "main@nanoclaw",
		Prompt:        "Past task",
		ScheduleType:  "once",
		NextRun:       &past,
		Status:        "active",
		CreatedAt:     now,
	}
	if err := db.SaveTask(task1); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	task2 := &Task{
		ID:            "future-task",
		GroupFolder:   "main",
		ChatJID:       "main@nanoclaw",
		Prompt:        "Future task",
		ScheduleType:  "once",
		NextRun:       &future,
		Status:        "active",
		CreatedAt:     now,
	}
	if err := db.SaveTask(task2); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	dueTasks, err := db.GetDueTasks(now)
	if err != nil {
		t.Fatalf("GetDueTasks failed: %v", err)
	}

	foundDue := false
	for _, task := range dueTasks {
		if task.ID == "due-task" {
			foundDue = true
		}
		if task.ID == "future-task" {
			t.Error("Future task should not be due")
		}
	}

	if !foundDue {
		t.Error("Expected to find due task")
	}
}
