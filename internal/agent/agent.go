// Package agent implements the AI agent layer using Google ADK-GO.
// It uses google.golang.org/adk with the Gemini model backend.
package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"github.com/linkerlin/nanoclaw.go/internal/config"
)

const defaultModel = "gemini-2.0-flash"

// RunAgent loads the group's instruction file, creates an ADK LLM agent backed
// by Gemini, and returns the agent's text response to the given prompt.
// sessionID is used to maintain conversation continuity across calls within a
// group (each group keeps its own session).
func RunAgent(ctx context.Context, groupFolder, sessionID, prompt string) (string, error) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GOOGLE_API_KEY environment variable not set")
	}

	modelName := defaultModel
	if m := os.Getenv("GEMINI_MODEL"); m != "" {
		modelName = m
	}

	m, err := gemini.NewModel(ctx, modelName, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return "", fmt.Errorf("create gemini model: %w", err)
	}

	instruction := loadInstruction(groupFolder)

	a, err := llmagent.New(llmagent.Config{
		Name:        groupFolder,
		Model:       m,
		Instruction: instruction,
		Description: fmt.Sprintf("Assistant for group %s", groupFolder),
	})
	if err != nil {
		return "", fmt.Errorf("create llm agent: %w", err)
	}

	sessionSvc := session.InMemoryService()

	// Ensure the session exists.
	appName := "nanoclaw"
	userID := "user"
	_, err = sessionSvc.Create(ctx, &session.CreateRequest{
		AppName:   appName,
		UserID:    userID,
		SessionID: sessionID,
	})
	// Ignore "already exists" errors — the session may have been created before.
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return "", fmt.Errorf("create session: %w", err)
	}

	r, err := runner.New(runner.Config{
		AppName:        appName,
		Agent:          a,
		SessionService: sessionSvc,
	})
	if err != nil {
		return "", fmt.Errorf("create runner: %w", err)
	}

	userMsg := genai.NewContentFromText(prompt, genai.RoleUser)

	var sb strings.Builder
	for event, err := range r.Run(ctx, userID, sessionID, userMsg, agent.RunConfig{}) {
		if err != nil {
			return "", fmt.Errorf("agent run: %w", err)
		}
		if event.Content != nil {
			for _, part := range event.Content.Parts {
				if part.Text != "" {
					sb.WriteString(part.Text)
				}
			}
		}
	}

	return sb.String(), nil
}

func loadInstruction(groupFolder string) string {
	path := filepath.Join(config.GroupsDir, groupFolder, "CLAUDE.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "You are a helpful assistant."
	}
	return string(data)
}
