//go:build integration
// +build integration

package internal

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAgent_Run_Integration(t *testing.T) {
	SkipIfNoLLM(t)

	db := TestTempDB(t)
	agent := NewAgent(db)

	ctx := context.Background()

	tests := []struct {
		name     string
		messages []Message
		contains []string
	}{
		{
			name: "simple greeting",
			messages: []Message{
				{
					ID:        MessageID("msg-1"),
					ChatJID:   "test@nanoclaw",
					Sender:    "user",
					Content:   "Say 'Hello, World!' and nothing else.",
					Timestamp: time.Now(),
				},
			},
			contains: []string{"Hello", "World"},
		},
		{
			name: "math question",
			messages: []Message{
				{
					ID:        MessageID("msg-2"),
					ChatJID:   "test@nanoclaw",
					Sender:    "user",
					Content:   "What is 2+2? Answer with just the number.",
					Timestamp: time.Now(),
				},
			},
			contains: []string{"4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			resp, err := agent.Run(ctx, "test", tt.messages)
			if err != nil {
				t.Fatalf("Agent.Run failed: %v", err)
			}

			if resp == "" {
				t.Error("Expected non-empty response")
			}

			for _, want := range tt.contains {
				if !strings.Contains(resp, want) {
					t.Errorf("Response %q does not contain %q", resp, want)
				}
			}

			t.Logf("Response: %s", resp)
		})
	}
}

func TestAgent_RunStream_Integration(t *testing.T) {
	SkipIfNoLLM(t)

	db := TestTempDB(t)
	agent := NewAgent(db)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages := []Message{
		{
			ID:        MessageID("stream-1"),
			ChatJID:   "test@nanoclaw",
			Sender:    "user",
			Content:   "Count from 1 to 3.",
			Timestamp: time.Now(),
		},
	}

	stream, err := agent.RunStream(ctx, "test", messages)
	if err != nil {
		t.Fatalf("Agent.RunStream failed: %v", err)
	}

	var fullResponse strings.Builder
	eventCount := 0

	for event := range stream {
		if event.Err != nil {
			if event.Err != io.EOF {
				t.Errorf("Stream error: %v", event.Err)
			}
			break
		}

		fullResponse.WriteString(event.Content)
		eventCount++
	}

	if eventCount == 0 {
		t.Error("Expected at least one stream event")
	}

	response := fullResponse.String()
	if response == "" {
		t.Error("Expected non-empty stream response")
	}

	t.Logf("Stream response (%d events): %s", eventCount, response)
}

func TestAgent_Run_DifferentProviders(t *testing.T) {
	origKey := os.Getenv("OPENAI_API_KEY")
	origBaseURL := os.Getenv("OPENAI_BASE_URL")
	origModel := os.Getenv("OPENAI_MODEL")

	defer func() {
		os.Setenv("OPENAI_API_KEY", origKey)
		os.Setenv("OPENAI_BASE_URL", origBaseURL)
		os.Setenv("OPENAI_MODEL", origModel)
	}()

	providers := []struct {
		name     string
		envKey   string
		envURL   string
		envModel string
	}{
		{
			name:     "OpenAI",
			envKey:   "OPENAI_API_KEY",
			envURL:   "OPENAI_BASE_URL",
			envModel: "OPENAI_MODEL",
		},
		{
			name:     "Groq",
			envKey:   "GROQ_API_KEY",
			envURL:   "GROQ_BASE_URL",
			envModel: "GROQ_MODEL",
		},
	}

	for _, provider := range providers {
		t.Run(provider.name, func(t *testing.T) {
			if os.Getenv(provider.envKey) == "" {
				t.Skipf("%s not set, skipping", provider.envKey)
			}

			os.Setenv("OPENAI_API_KEY", os.Getenv(provider.envKey))
			if url := os.Getenv(provider.envURL); url != "" {
				os.Setenv("OPENAI_BASE_URL", url)
			}
			if model := os.Getenv(provider.envModel); model != "" {
				os.Setenv("OPENAI_MODEL", model)
			}

			db := TestTempDB(t)
			agent := NewAgent(db)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			messages := []Message{
				{
					ID:        MessageID("provider-test"),
					ChatJID:   "test@nanoclaw",
					Sender:    "user",
					Content:   "Say 'Pong'.",
					Timestamp: time.Now(),
				},
			}

			resp, err := agent.Run(ctx, "test", messages)
			if err != nil {
				t.Fatalf("Agent.Run failed: %v", err)
			}

			if !strings.Contains(strings.ToLower(resp), "pong") {
				t.Errorf("Response %q does not contain 'Pong'", resp)
			}

			t.Logf("%s response: %s", provider.name, resp)
		})
	}
}

func TestAgent_Run_Timeout(t *testing.T) {
	SkipIfNoLLM(t)

	db := TestTempDB(t)
	agent := NewAgent(db)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	messages := []Message{
		{
			ID:        MessageID("timeout-1"),
			ChatJID:   "test@nanoclaw",
			Sender:    "user",
			Content:   "Write a long story.",
			Timestamp: time.Now(),
		},
	}

	_, err := agent.Run(ctx, "test", messages)
	if err == nil {
		t.Error("Expected timeout error")
	}
}
