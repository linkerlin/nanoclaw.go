// Package agent implements the AI agent layer using the Google Gemini API.
// It mirrors the interface described in the ADK-GO design so the implementation
// can be swapped for google.golang.org/adk once it is available in the network.
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/linkerlin/nanoclaw.go/internal/config"
)

const geminiEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s"
const defaultModel = "gemini-2.5-flash"

// geminiRequest is the request body for the Gemini generateContent endpoint.
type geminiRequest struct {
	SystemInstruction *geminiContent  `json:"system_instruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// RunAgent loads the group's instruction, calls Gemini, and returns the text response.
// sessionID is accepted for API parity with an ADK-based implementation (session
// continuity is not currently maintained across calls in this implementation).
func RunAgent(ctx context.Context, groupFolder, _ /* sessionID */, prompt string) (string, error) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GOOGLE_API_KEY environment variable not set")
	}

	instruction := loadInstruction(groupFolder)

	reqBody := geminiRequest{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: instruction}},
		},
		Contents: []geminiContent{
			{Role: "user", Parts: []geminiPart{{Text: prompt}}},
		},
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	model := defaultModel
	if m := os.Getenv("GEMINI_MODEL"); m != "" {
		model = m
	}
	url := fmt.Sprintf(geminiEndpoint, model, apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini api call: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var result geminiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("gemini error: %s", result.Error.Message)
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from gemini")
	}
	return result.Candidates[0].Content.Parts[0].Text, nil
}

func loadInstruction(groupFolder string) string {
	path := filepath.Join(config.GroupsDir, groupFolder, "CLAUDE.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "You are a helpful assistant."
	}
	return string(data)
}
