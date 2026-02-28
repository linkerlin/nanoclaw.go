package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/linkerlin/nanoclaw.go/internal/agent"
	"github.com/linkerlin/nanoclaw.go/internal/db"
	"github.com/linkerlin/nanoclaw.go/internal/types"
)

// Scheduler polls for active tasks and runs them on their schedule.
type Scheduler struct {
	db   *db.DB
	cron *cron.Cron
}

// New creates a new Scheduler.
func New(database *db.DB) *Scheduler {
	return &Scheduler{
		db:   database,
		cron: cron.New(),
	}
}

// Start begins the scheduler loop.
func (s *Scheduler) Start(ctx context.Context) {
	s.cron.Start()
	go s.poll(ctx)
}

// Stop halts the scheduler.
func (s *Scheduler) Stop() {
	s.cron.Stop()
}

func (s *Scheduler) poll(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runDueTasks(ctx)
		}
	}
}

func (s *Scheduler) runDueTasks(ctx context.Context) {
	tasks, err := s.db.GetActiveTasks()
	if err != nil {
		log.Printf("scheduler: fetch tasks: %v", err)
		return
	}
	now := time.Now().UTC()
	for _, t := range tasks {
		if isDue(t, now) {
			go s.runTask(ctx, t, now)
		}
	}
}

func (s *Scheduler) runTask(ctx context.Context, t types.ScheduledTask, now time.Time) {
	resp, err := agent.RunAgent(ctx, t.GroupFolder, t.ID, t.Prompt)
	lastRun := now.Format(time.RFC3339)
	result := resp
	if err != nil {
		result = "error: " + err.Error()
		log.Printf("scheduler: task %s error: %v", t.ID, err)
	}

	var nextRun *string
	if t.ScheduleType != "once" {
		next := computeNext(t, now)
		if next != "" {
			nextRun = &next
		}
	} else {
		// Mark completed.
		if err := s.db.UpdateTaskRun(t.ID, lastRun, result, nil); err != nil {
			log.Printf("scheduler: update completed task %s: %v", t.ID, err)
		}
		return
	}
	if err := s.db.UpdateTaskRun(t.ID, lastRun, result, nextRun); err != nil {
		log.Printf("scheduler: update task %s: %v", t.ID, err)
	}
}

func isDue(t types.ScheduledTask, now time.Time) bool {
	if t.NextRun == nil {
		return false
	}
	next, err := time.Parse(time.RFC3339, *t.NextRun)
	if err != nil {
		return false
	}
	return !now.Before(next)
}

func computeNext(t types.ScheduledTask, from time.Time) string {
	switch t.ScheduleType {
	case "interval":
		d, err := time.ParseDuration(t.ScheduleValue)
		if err != nil {
			return ""
		}
		return from.Add(d).Format(time.RFC3339)
	case "cron":
		p, err := cron.ParseStandard(t.ScheduleValue)
		if err != nil {
			return ""
		}
		return p.Next(from).Format(time.RFC3339)
	}
	return ""
}
