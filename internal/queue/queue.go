package queue

import (
	"context"
	"sync"
)

// Job is a unit of work that the queue will execute.
type Job func(ctx context.Context)

// GroupQueue manages per-group FIFO queues with a global concurrency limit.
type GroupQueue struct {
	mu          sync.Mutex
	queues      map[string][]Job
	active      int
	maxActive   int
}

// New creates a new GroupQueue limited to maxConcurrent simultaneous workers.
func New(maxConcurrent int) *GroupQueue {
	return &GroupQueue{
		queues:    make(map[string][]Job),
		maxActive: maxConcurrent,
	}
}

// Enqueue adds a job to the given group's queue and tries to dispatch.
// The provided ctx is forwarded to the job so cancellation propagates.
func (q *GroupQueue) Enqueue(ctx context.Context, group string, job Job) {
	q.mu.Lock()
	q.queues[group] = append(q.queues[group], job)
	q.mu.Unlock()
	q.tryDispatch(ctx)
}

func (q *GroupQueue) tryDispatch(ctx context.Context) {
	q.mu.Lock()
	if q.active >= q.maxActive {
		q.mu.Unlock()
		return
	}
	// Find a group with pending work.
	var chosen string
	for g, jobs := range q.queues {
		if len(jobs) > 0 {
			chosen = g
			break
		}
	}
	if chosen == "" {
		q.mu.Unlock()
		return
	}
	job := q.queues[chosen][0]
	q.queues[chosen] = q.queues[chosen][1:]
	if len(q.queues[chosen]) == 0 {
		delete(q.queues, chosen)
	}
	q.active++
	q.mu.Unlock()

	go func() {
		job(ctx)
		q.mu.Lock()
		q.active--
		q.mu.Unlock()
		q.tryDispatch(ctx)
	}()
}
