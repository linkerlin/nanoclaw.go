package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/linkerlin/nanoclaw.go/internal/config"
	"github.com/linkerlin/nanoclaw.go/internal/db"
	"github.com/linkerlin/nanoclaw.go/internal/orchestrator"
	"github.com/linkerlin/nanoclaw.go/internal/scheduler"
	"github.com/linkerlin/nanoclaw.go/internal/tui"
	"github.com/linkerlin/nanoclaw.go/internal/types"
)

func main() {
	// Ensure data directory exists.
	if err := os.MkdirAll(config.DataDir, 0o755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}
	if err := os.MkdirAll(config.GroupsDir, 0o755); err != nil {
		log.Fatalf("create groups dir: %v", err)
	}

	// Open database.
	database, err := db.Open(config.DBPath())
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer database.Close()

	// Seed a default "main" group if none exists.
	if err := ensureMainGroup(database); err != nil {
		log.Fatalf("seed main group: %v", err)
	}

	// Load groups for TUI.
	groups, err := database.GetRegisteredGroups()
	if err != nil {
		log.Fatalf("load groups: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Wire up TUI with a send callback (placeholder until program is created).
	var orch *orchestrator.Orchestrator
	sendFn := func(chatJID, text string) {
		if orch != nil {
			orch.HandleUserMessage(chatJID, "You", text)
		}
	}

	tuiModel := tui.New(groups, sendFn)
	program := tea.NewProgram(tuiModel, tea.WithAltScreen())

	// Now that we have the program, create the orchestrator.
	orch = orchestrator.New(database, program)

	// Start background services.
	sched := scheduler.New(database)
	sched.Start(ctx)
	defer sched.Stop()

	go orch.Poll(ctx)

	// Handle OS signals for graceful shutdown.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
		program.Quit()
	}()

	// Run TUI (blocks until quit).
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}

// ensureMainGroup registers a default "main" group if the database is empty.
func ensureMainGroup(database *db.DB) error {
	groups, err := database.GetRegisteredGroups()
	if err != nil {
		return err
	}
	if len(groups) > 0 {
		return nil
	}

	mainDir := filepath.Join(config.GroupsDir, config.MainGroupFolder)
	if err := os.MkdirAll(mainDir, 0o755); err != nil {
		return err
	}
	// Write a default instruction file if absent.
	claudeMD := filepath.Join(mainDir, "CLAUDE.md")
	if _, err := os.Stat(claudeMD); os.IsNotExist(err) {
		content := fmt.Sprintf("You are %s, a helpful AI assistant.\n", config.AssistantName)
		if err := os.WriteFile(claudeMD, []byte(content), 0o644); err != nil {
			return err
		}
	}

	g := types.RegisteredGroup{
		JID:             "main@nanoclaw",
		Name:            "Main",
		Folder:          config.MainGroupFolder,
		TriggerPattern:  `(?i)^@` + config.AssistantName + `\b`,
		AddedAt:         time.Now().UTC().Format(time.RFC3339),
		RequiresTrigger: true,
	}
	return database.RegisterGroup(g)
}
