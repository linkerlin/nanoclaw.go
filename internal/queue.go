package internal

import (
	"context"
	"sync"

	"golang.org/x/sync/semaphore"
)

// GroupQueue 使用Google官方semaphore实现加权并发控制
type GroupQueue struct {
	sem      *semaphore.Weighted
	mu       sync.Mutex
	queues   map[ChatJID][]func()
	running  map[ChatJID]bool
}

// NewGroupQueue 创建队列
func NewGroupQueue(maxConcurrent int64) *GroupQueue {
	return &GroupQueue{
		sem:     semaphore.NewWeighted(maxConcurrent),
		queues:  make(map[ChatJID][]func()),
		running: make(map[ChatJID]bool),
	}
}

// Enqueue 将任务加入队列
func (q *GroupQueue) Enqueue(ctx context.Context, chatJID ChatJID, job func()) error {
	q.mu.Lock()
	q.queues[chatJID] = append(q.queues[chatJID], job)
	q.mu.Unlock()

	return q.tryDispatch(ctx)
}

// tryDispatch 尝试分发任务
func (q *GroupQueue) tryDispatch(ctx context.Context) error {
	q.mu.Lock()

	// 找一个有待处理任务且未在运行的群组
	var target ChatJID
	for jid, jobs := range q.queues {
		if len(jobs) > 0 && !q.running[jid] {
			target = jid
			break
		}
	}

	if target == "" {
		q.mu.Unlock()
		return nil
	}

	// 取出任务
	job := q.queues[target][0]
	q.queues[target] = q.queues[target][1:]
	if len(q.queues[target]) == 0 {
		delete(q.queues, target)
	}
	q.running[target] = true
	q.mu.Unlock()

	// 获取semaphore许可
	if err := q.sem.Acquire(ctx, 1); err != nil {
		q.mu.Lock()
		q.running[target] = false
		q.mu.Unlock()
		return err
	}

	// 异步执行
	go func() {
		defer func() {
			q.sem.Release(1)
			q.mu.Lock()
			q.running[target] = false
			q.mu.Unlock()
			// 尝试分发下一个
			q.tryDispatch(ctx)
		}()
		job()
	}()

	return nil
}

// PendingCount 返回待处理任务数
func (q *GroupQueue) PendingCount(chatJID ChatJID) int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.queues[chatJID])
}

// IsRunning 检查群组是否正在处理
func (q *GroupQueue) IsRunning(chatJID ChatJID) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.running[chatJID]
}
