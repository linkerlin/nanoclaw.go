package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

		"github.com/linkerlin/nanoclaw.go/internal"
)

func main() {
	// 初始化日志
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// 加载配置
	cfg := internal.LoadConfig()

	// 确保目录存在
	os.MkdirAll(cfg.App.DataDir, 0755)
	os.MkdirAll(cfg.App.GroupsDir, 0755)

	// 打开数据库
	db, err := internal.OpenDB(cfg.DBPath())
	if err != nil {
		slog.Error("open db", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	// 初始化默认群组
	initDefaultGroup(db)

	// 初始化组件
	queue := internal.NewGroupQueue(cfg.App.MaxConcurrent)
	agent := internal.NewAgent(db)
	scheduler := internal.NewScheduler(db, agent)
	orch := internal.NewOrchestrator(db, queue, agent, cfg)

	// 创建TUI
	tui := internal.NewTUI(db, queue, agent, cfg)
	tui.SetOnSend(func(chatJID internal.ChatJID, content string) {
		orch.HandleMessage(chatJID, "You", content)
	})

	// 设置TUI到编排器
	orch.SetProgram(nil)  // 简化处理

	// 上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动调度器
	scheduler.Start(ctx)
	defer scheduler.Stop()

	// 信号处理
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		slog.Info("shutting down...")
		cancel()
		// 退出信号
	}()

	// 运行TUI
	slog.Info("nanoclaw started", "name", cfg.App.Name)
	if err := tui.Run(ctx); err != nil {
		slog.Error("tui error", "err", err)
		os.Exit(1)
	}
}

func initDefaultGroup(db *internal.DB) {
	group := &internal.Group{
		JID:             "main@nanoclaw",
		Name:            "Main",
		Folder:          "main",
		TriggerPattern:  `(?i)^@Andy\b`,
		RequiresTrigger: true,
	}
	// 忽略错误（可能已存在）
	db.SaveGroup(group)
}
