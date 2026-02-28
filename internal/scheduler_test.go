package internal

import (
	"context"
	"testing"
	"time"
)

func TestSchedulerXSkipLLM_StartStop(t *testing.T) {
	if !HasLLMConfig() {
		t.Skip("No LLM config, skipping")
	}
	db := TestTempDB(t)
	agent := NewAgent(db)
	scheduler := NewScheduler(db, agent)

	ctx, cancel := context.WithCancel(context.Background())

	scheduler.Start(ctx)

	time.Sleep(100 * time.Millisecond)

	cancel()
	scheduler.Stop()
}

func TestSchedulerXSkipLLM_RunTask(t *testing.T) {
	SkipIfNoLLM(t)

	db := TestTempDB(t)
	agent := NewAgent(db)
	scheduler := NewScheduler(db, agent)

	chatJID := ChatJID("test@nanoclaw")
	now := time.Now()
	past := now.Add(-time.Second)

	task := &Task{
		ID:            "scheduled-task-1",
		GroupFolder:   "test",
		ChatJID:       chatJID,
		Prompt:        "Say 'Task executed' and nothing else.",
		ScheduleType:  "once",
		NextRun:       &past,
		Status:        "active",
		CreatedAt:     now,
	}

	if err := db.SaveTask(task); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	scheduler.Start(ctx)
	defer scheduler.Stop()

	time.Sleep(5 * time.Second)
}

func TestSchedulerXSkipLLM_MultipleTasks(t *testing.T) {
	if !HasLLMConfig() {
		t.Skip("No LLM config, skipping")
	}
	db := TestTempDB(t)
	agent := NewAgent(db)
	scheduler := NewScheduler(db, agent)

	now := time.Now()

	for i := 0; i < 3; i++ {
		past := now.Add(-time.Duration(i+1) * time.Second)
		task := &Task{
			ID:            "multi-task-" + string(rune('0'+i)),
			GroupFolder:   "test",
			ChatJID:       ChatJID("test" + string(rune('0'+i)) + "@nanoclaw"),
			Prompt:        "Test prompt",
			ScheduleType:  "once",
			NextRun:       &past,
			Status:        "active",
			CreatedAt:     now,
		}
		if err := db.SaveTask(task); err != nil {
			t.Fatalf("SaveTask failed: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	scheduler.Start(ctx)
	defer scheduler.Stop()

	time.Sleep(2 * time.Second)
}

func TestSchedulerXSkipLLM_PausedTask(t *testing.T) {
	if !HasLLMConfig() {
		t.Skip("No LLM config, skipping")
	}
	db := TestTempDB(t)
	agent := NewAgent(db)
	scheduler := NewScheduler(db, agent)

	now := time.Now()
	past := now.Add(-time.Second)

	task := &Task{
		ID:            "paused-task",
		GroupFolder:   "test",
		ChatJID:       "test@nanoclaw",
		Prompt:        "This should not run",
		ScheduleType:  "once",
		NextRun:       &past,
		Status:        "paused",
		CreatedAt:     now,
	}

	if err := db.SaveTask(task); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	scheduler.Start(ctx)
	defer scheduler.Stop()

	time.Sleep(2 * time.Second)
}

func TestSchedulerXSkipLLM_FutureTask(t *testing.T) {
	if !HasLLMConfig() {
		t.Skip("No LLM config, skipping")
	}
	db := TestTempDB(t)
	agent := NewAgent(db)
	scheduler := NewScheduler(db, agent)

	now := time.Now()
	future := now.Add(time.Hour)

	task := &Task{
		ID:            "future-task",
		GroupFolder:   "test",
		ChatJID:       "test@nanoclaw",
		Prompt:        "This should not run yet",
		ScheduleType:  "once",
		NextRun:       &future,
		Status:        "active",
		CreatedAt:     now,
	}

	if err := db.SaveTask(task); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	scheduler.Start(ctx)
	defer scheduler.Stop()

	time.Sleep(2 * time.Second)
}
