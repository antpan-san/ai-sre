package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"

	"github.com/panshuai/ai-sre/internal/config"
)

// Client wraps OpenAI-compatible chat API (DeepSeek).
type Client struct {
	api   *openai.Client
	model string
}

// NewFromConfig builds a client from file-based credential config.
func NewFromConfig(c *config.LLM) (*Client, error) {
	if c == nil {
		return nil, errors.New("llm config is nil")
	}
	key := strings.TrimSpace(c.APIKey)
	if key == "" {
		return nil, errors.New("api key is empty")
	}
	cfg := openai.DefaultConfig(key)
	cfg.BaseURL = c.BaseURL
	if cfg.BaseURL == "" {
		cfg.BaseURL = config.DefaultBaseURL
	}
	model := strings.TrimSpace(c.Model)
	if model == "" {
		model = config.DefaultModel
	}
	return &Client{api: openai.NewClientWithConfig(cfg), model: model}, nil
}

// Chat sends a single user message with system priming for SRE tasks.
func (c *Client) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	req := openai.ChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		Temperature: 0.2,
	}
	resp, err := c.api.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("deepseek chat: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", errors.New("empty response from model")
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}
