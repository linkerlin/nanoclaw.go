package orchestrator

import (
	"context"
	"fmt"
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/linkerlin/nanoclaw.go/internal/agent"
	"github.com/linkerlin/nanoclaw.go/internal/config"
	"github.com/linkerlin/nanoclaw.go/internal/db"
	"github.com/linkerlin/nanoclaw.go/internal/queue"
	"github.com/linkerlin/nanoclaw.go/internal/router"
	"github.com/linkerlin/nanoclaw.go/internal/tui"
	"github.com/linkerlin/nanoclaw.go/internal/types"
)

// Orchestrator ties together the database, agent queue, and TUI.
type Orchestrator struct {
	db      *db.DB
	queue   *queue.GroupQueue
	program *tea.Program
	lastSeen map[string]string // chatJID -> last processed timestamp
}

// New creates a new Orchestrator.
func New(database *db.DB, program *tea.Program) *Orchestrator {
	return &Orchestrator{
		db:       database,
		queue:    queue.New(config.MaxConcurrentAgents),
		program:  program,
		lastSeen: make(map[string]string),
	}
}

// HandleUserMessage stores a message from the TUI and enqueues agent processing.
func (o *Orchestrator) HandleUserMessage(chatJID, senderName, text string) {
	msg := types.NewMessage{
		ID:         uuid.New().String(),
		ChatJID:    chatJID,
		Sender:     senderName,
		SenderName: senderName,
		Content:    text,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		IsFromMe:   true,
		IsBotMessage: false,
	}
	if err := o.db.SaveMessage(msg); err != nil {
		log.Printf("orchestrator: save message: %v", err)
		return
	}
	o.refreshTUI(chatJID)

	// Check trigger.
	if config.TriggerPattern.MatchString(text) {
		o.enqueueAgent(context.Background(), chatJID)
	}
}

func (o *Orchestrator) enqueueAgent(ctx context.Context, chatJID string) {
	o.program.Send(tui.ThinkingMsg{ChatJID: chatJID, Thinking: true})
	o.queue.Enqueue(ctx, chatJID, func(ctx context.Context) {
		defer func() {
			o.program.Send(tui.ThinkingMsg{ChatJID: chatJID, Thinking: false})
		}()
		o.runAgent(ctx, chatJID)
	})
}

func (o *Orchestrator) runAgent(ctx context.Context, chatJID string) {
	group, err := o.groupForChat(chatJID)
	if err != nil {
		log.Printf("orchestrator: find group for %s: %v", chatJID, err)
		return
	}

	sessionID, err := o.db.GetOrCreateSession(group.Folder)
	if err != nil {
		log.Printf("orchestrator: session for %s: %v", group.Folder, err)
		return
	}

	msgs, err := o.db.GetRecentMessages(chatJID, 20)
	if err != nil {
		log.Printf("orchestrator: get messages: %v", err)
		return
	}

	prompt := router.FormatMessages(msgs)
	resp, err := agent.RunAgent(ctx, group.Folder, sessionID, prompt)
	if err != nil {
		log.Printf("orchestrator: agent error for %s: %v", chatJID, err)
		resp = fmt.Sprintf("(agent error: %v)", err)
	}
	resp = router.FormatOutbound(resp)

	botMsg := types.NewMessage{
		ID:           uuid.New().String(),
		ChatJID:      chatJID,
		Sender:       config.AssistantName,
		SenderName:   config.AssistantName,
		Content:      resp,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		IsFromMe:     false,
		IsBotMessage: true,
	}
	if err := o.db.SaveMessage(botMsg); err != nil {
		log.Printf("orchestrator: save bot message: %v", err)
	}
	o.refreshTUI(chatJID)
}

// Poll periodically checks for any new unprocessed messages (e.g. from scheduled tasks).
func (o *Orchestrator) Poll(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(config.PollInterval) * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.checkGroups(ctx)
		}
	}
}

func (o *Orchestrator) checkGroups(ctx context.Context) {
	groups, err := o.db.GetRegisteredGroups()
	if err != nil {
		return
	}
	for _, g := range groups {
		since := o.lastSeen[g.JID]
		msgs, err := o.db.GetNewMessages(g.JID, since)
		if err != nil || len(msgs) == 0 {
			continue
		}
		// Update last seen.
		o.lastSeen[g.JID] = msgs[len(msgs)-1].Timestamp

		for _, m := range msgs {
			if config.TriggerPattern.MatchString(m.Content) {
				o.enqueueAgent(ctx, g.JID)
				break
			}
		}
	}
}

func (o *Orchestrator) refreshTUI(chatJID string) {
	msgs, err := o.db.GetRecentMessages(chatJID, 50)
	if err != nil {
		return
	}
	o.program.Send(tui.MessagesUpdatedMsg{ChatJID: chatJID, Messages: msgs})
}

func (o *Orchestrator) groupForChat(chatJID string) (types.RegisteredGroup, error) {
	groups, err := o.db.GetRegisteredGroups()
	if err != nil {
		return types.RegisteredGroup{}, err
	}
	for _, g := range groups {
		if g.JID == chatJID {
			return g, nil
		}
	}
	return types.RegisteredGroup{}, fmt.Errorf("no group registered for jid %s", chatJID)
}
