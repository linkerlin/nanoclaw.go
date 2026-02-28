package internal

import (
	"context"
	"fmt"
	"io"

	"github.com/sashabaranov/go-openai"
)

// Agent LLM代理
type Agent struct {
	client *openai.Client
	model  string
	db     *DB
}

// NewAgent 从环境变量创建Agent
func NewAgent(db *DB) *Agent {
	cfg := LoadConfig()

	if cfg.LLM.APIKey == "" {
		panic("OPENAI_API_KEY is required")
	}

	config := openai.DefaultConfig(cfg.LLM.APIKey)
	config.BaseURL = cfg.LLM.BaseURL

	return &Agent{
		client: openai.NewClientWithConfig(config),
		model:  cfg.LLM.Model,
		db:     db,
	}
}

// Run 执行单次对话
func (a *Agent) Run(ctx context.Context, groupFolder string, messages []Message) (string, error) {
	// 转换消息格式
	var msgs []openai.ChatCompletionMessage
	for _, m := range messages {
		role := openai.ChatMessageRoleUser
		if m.IsBotMessage {
			role = openai.ChatMessageRoleAssistant
		}
		msgs = append(msgs, openai.ChatCompletionMessage{
			Role:    role,
			Content: m.Content,
		})
	}

	// 调用API
	resp, err := a.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    a.model,
		Messages: msgs,
	})
	if err != nil {
		return "", fmt.Errorf("llm error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	return resp.Choices[0].Message.Content, nil
}

// RunStream 流式执行
func (a *Agent) RunStream(ctx context.Context, groupFolder string, messages []Message) (<-chan StreamEvent, error) {
	// 转换消息格式
	var msgs []openai.ChatCompletionMessage
	for _, m := range messages {
		role := openai.ChatMessageRoleUser
		if m.IsBotMessage {
			role = openai.ChatMessageRoleAssistant
		}
		msgs = append(msgs, openai.ChatCompletionMessage{
			Role:    role,
			Content: m.Content,
		})
	}

	// 创建流
	stream, err := a.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:    a.model,
		Messages: msgs,
		Stream:   true,
	})
	if err != nil {
		return nil, fmt.Errorf("llm stream error: %w", err)
	}

	ch := make(chan StreamEvent)
	go func() {
		defer close(ch)
		defer stream.Close()

		for {
			select {
			case <-ctx.Done():
				ch <- StreamEvent{Err: ctx.Err(), Done: true}
				return
			default:
				response, err := stream.Recv()
				if err != nil {
					if err == io.EOF {
						ch <- StreamEvent{Done: true}
						return
					}
					ch <- StreamEvent{Err: err, Done: true}
					return
				}
				if len(response.Choices) > 0 {
					ch <- StreamEvent{
						Content: response.Choices[0].Delta.Content,
					}
				}
			}
		}
	}()

	return ch, nil
}
