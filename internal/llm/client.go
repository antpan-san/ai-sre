package llm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/sashabaranov/go-openai"
)

const (
	defaultBaseURL = "https://api.deepseek.com/v1"
	defaultModel   = "deepseek-chat"
)

// Client wraps OpenAI-compatible chat API (DeepSeek).
type Client struct {
	api   *openai.Client
	model string
}

// NewFromEnv uses DEEPSEEK_API_KEY and optional DEEPSEEK_BASE_URL, DEEPSEEK_MODEL.
func NewFromEnv() (*Client, error) {
	key := strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY"))
	if key == "" {
		return nil, errors.New("DEEPSEEK_API_KEY is not set (export your DeepSeek API key)")
	}
	base := strings.TrimSpace(os.Getenv("DEEPSEEK_BASE_URL"))
	if base == "" {
		base = defaultBaseURL
	}
	cfg := openai.DefaultConfig(key)
	cfg.BaseURL = base
	model := strings.TrimSpace(os.Getenv("DEEPSEEK_MODEL"))
	if model == "" {
		model = defaultModel
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
