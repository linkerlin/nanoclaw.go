package internal

import (
	"context"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler 定时任务调度器
type Scheduler struct {
	db     *DB
	agent  *Agent
	cron   *cron.Cron
	ticker *time.Ticker
	stop   chan struct{}
}

// NewScheduler 创建调度器
func NewScheduler(db *DB, agent *Agent) *Scheduler {
	return &Scheduler{
		db:    db,
		agent: agent,
		cron:  cron.New(),
		stop:  make(chan struct{}),
	}
}

// Start 启动调度器
func (s *Scheduler) Start(ctx context.Context) {
	s.cron.Start()
	s.ticker = time.NewTicker(60 * time.Second)

	go s.loop(ctx)
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.cron.Stop()
	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.stop)
}

func (s *Scheduler) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stop:
			return
		case <-s.ticker.C:
			s.checkTasks(ctx)
		}
	}
}

func (s *Scheduler) checkTasks(ctx context.Context) {
	tasks, err := s.db.GetDueTasks(time.Now())
	if err != nil {
		slog.Error("get due tasks", "err", err)
		return
	}

	for _, task := range tasks {
		go s.runTask(ctx, task)
	}
}

func (s *Scheduler) runTask(ctx context.Context, task Task) {
	slog.Info("running task", "id", task.ID, "group", task.GroupFolder)

	// 获取历史消息作为上下文
	messages, err := s.db.GetMessages(task.ChatJID, 10)
	if err != nil {
		slog.Error("get messages", "err", err)
		s.db.UpdateTaskRun(task.ID, "error: "+err.Error(), nil)
		return
	}

	// 添加任务提示
	messages = append(messages, Message{
		ID:        MessageID("task-" + task.ID),
		ChatJID:   task.ChatJID,
		Sender:    "System",
		Content:   task.Prompt,
		Timestamp: time.Now(),
	})

	// 调用Agent
	resp, err := s.agent.Run(ctx, task.GroupFolder, messages)
	if err != nil {
		slog.Error("agent run", "err", err)
		s.db.UpdateTaskRun(task.ID, "error: "+err.Error(), nil)
		return
	}

	// 保存结果
	var nextRun *time.Time
	if task.ScheduleType == "interval" {
		d, _ := time.ParseDuration(task.ScheduleValue)
		if d > 0 {
			nr := time.Now().Add(d)
			nextRun = &nr
		}
	} else if task.ScheduleType == "cron" {
		// 解析cron表达式计算下次执行时间
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		if sched, err := parser.Parse(task.ScheduleValue); err == nil {
			nr := sched.Next(time.Now())
			nextRun = &nr
		}
	}

	s.db.UpdateTaskRun(task.ID, resp, nextRun)
	slog.Info("task completed", "id", task.ID)
}
