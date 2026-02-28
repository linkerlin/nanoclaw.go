package internal

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGroupQueue_Enqueue(t *testing.T) {
	queue := NewGroupQueue(2) // 最大并发2
	
	var executed int32
	ctx := context.Background()
	
	// 添加任务
	for i := 0; i < 5; i++ {
		chatJID := ChatJID("test@nanoclaw")
		err := queue.Enqueue(ctx, chatJID, func() {
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt32(&executed, 1)
		})
		if err != nil {
			t.Fatalf("Enqueue failed: %v", err)
		}
	}
	
	// 等待所有任务完成
	time.Sleep(200 * time.Millisecond)
	
	if atomic.LoadInt32(&executed) != 5 {
		t.Errorf("Expected 5 tasks executed, got %d", executed)
	}
}

func TestGroupQueue_PerGroupFIFO(t *testing.T) {
	queue := NewGroupQueue(10)
	
	var order []string
	var mu sync.Mutex
	ctx := context.Background()
	
	chatJID := ChatJID("test@nanoclaw")
	
	// 为同一群组添加多个任务
	for i := 0; i < 3; i++ {
		id := string(rune('A' + i))
		err := queue.Enqueue(ctx, chatJID, func() {
			time.Sleep(10 * time.Millisecond)
			mu.Lock()
			order = append(order, id)
			mu.Unlock()
		})
		if err != nil {
			t.Fatalf("Enqueue failed: %v", err)
		}
	}
	
	// 等待所有任务完成
	time.Sleep(200 * time.Millisecond)
	
	// 验证FIFO顺序
	mu.Lock()
	defer mu.Unlock()
	
	if len(order) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(order))
	}
	
	expected := []string{"A", "B", "C"}
	for i, exp := range expected {
		if order[i] != exp {
			t.Errorf("Task order[%d] = %q, want %q", i, order[i], exp)
		}
	}
}

func TestGroupQueue_ConcurrentLimit(t *testing.T) {
	queue := NewGroupQueue(2) // 最大并发2
	
	var running int32
	var maxRunning int32
	ctx := context.Background()
	
	// 添加多个长时间运行的任务
	for i := 0; i < 10; i++ {
		chatJID := ChatJID("test@nanoclaw")
		err := queue.Enqueue(ctx, chatJID, func() {
			current := atomic.AddInt32(&running, 1)
			// 更新最大并发数
			for {
				max := atomic.LoadInt32(&maxRunning)
				if current <= max || atomic.CompareAndSwapInt32(&maxRunning, max, current) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			atomic.AddInt32(&running, -1)
		})
		if err != nil {
			t.Fatalf("Enqueue failed: %v", err)
		}
	}
	
	// 等待所有任务完成
	time.Sleep(500 * time.Millisecond)
	
	// 验证最大并发数不超过限制
	if atomic.LoadInt32(&maxRunning) > 2 {
		t.Errorf("Max concurrent = %d, want <= 2", maxRunning)
	}
}

func TestGroupQueueXSkip_ContextCancellation(t *testing.T) {
	queue := NewGroupQueue(1)
	
	ctx, cancel := context.WithCancel(context.Background())
	
	// 先添加一个长时间运行的任务
	chatJID := ChatJID("test@nanoclaw")
	queue.Enqueue(ctx, chatJID, func() {
		time.Sleep(500 * time.Millisecond)
	})
	
	// 立即取消上下文
	cancel()
	
	// 尝试添加新任务应该失败
	err := queue.Enqueue(ctx, chatJID, func() {})
	if err == nil {
		t.Error("Expected error after context cancellation")
	}
}

func TestGroupQueue_IsRunning(t *testing.T) {
	queue := NewGroupQueue(1)
	
	chatJID := ChatJID("test@nanoclaw")
	ctx := context.Background()
	
	// 初始状态
	if queue.IsRunning(chatJID) {
		t.Error("Expected not running initially")
	}
	
	// 添加长时间运行的任务
	done := make(chan bool)
	queue.Enqueue(ctx, chatJID, func() {
		time.Sleep(100 * time.Millisecond)
		close(done)
	})
	
	// 短暂等待任务启动
	time.Sleep(10 * time.Millisecond)
	
	// 验证运行状态
	if !queue.IsRunning(chatJID) {
		t.Error("Expected running after task started")
	}
	
	// 等待任务完成
	<-done
	time.Sleep(10 * time.Millisecond)
	
	if queue.IsRunning(chatJID) {
		t.Error("Expected not running after task completed")
	}
}

func TestGroupQueue_PendingCount(t *testing.T) {
	queue := NewGroupQueue(1) // 单并发，便于测试
	
	chatJID := ChatJID("test@nanoclaw")
	ctx := context.Background()
	
	// 添加一个长时间运行的任务
	queue.Enqueue(ctx, chatJID, func() {
		time.Sleep(200 * time.Millisecond)
	})
	
	// 快速添加更多任务
	for i := 0; i < 3; i++ {
		queue.Enqueue(ctx, chatJID, func() {
			time.Sleep(10 * time.Millisecond)
		})
	}
	
	// 短暂等待
	time.Sleep(10 * time.Millisecond)
	
	// 验证待处理数量
	pending := queue.PendingCount(chatJID)
	if pending != 3 {
		t.Errorf("PendingCount = %d, want 3", pending)
	}
	
	// 等待所有任务完成
	time.Sleep(500 * time.Millisecond)
	
	if queue.PendingCount(chatJID) != 0 {
		t.Error("Expected 0 pending after all tasks completed")
	}
}
