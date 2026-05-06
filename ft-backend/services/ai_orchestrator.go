package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// ServerAIConfig reads DeepSeek-compatible settings from environment.
type ServerAIConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

func LoadServerAIConfig() ServerAIConfig {
	cfg := ServerAIConfig{
		APIKey:  strings.TrimSpace(os.Getenv("OPSFLEET_AI_API_KEY")),
		BaseURL: strings.TrimSpace(os.Getenv("OPSFLEET_AI_BASE_URL")),
		Model:   strings.TrimSpace(os.Getenv("OPSFLEET_AI_MODEL")),
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.deepseek.com/v1"
	}
	if cfg.Model == "" {
		cfg.Model = "deepseek-chat"
	}
	return cfg
}

// DiagnoseWithDeepSeek sends one diagnose prompt to DeepSeek-compatible chat/completions.
func DiagnoseWithDeepSeek(ctx context.Context, cfg ServerAIConfig, prompt string) (string, error) {
	if cfg.APIKey == "" {
		return "", fmt.Errorf("OPSFLEET_AI_API_KEY 未配置")
	}
	body := map[string]interface{}{
		"model": cfg.Model,
		"messages": []map[string]string{
			{"role": "system", "content": "你是AI SRE Copilot，回答需可执行且可验证。"},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.2,
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(cfg.BaseURL, "/")+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("deepseek status=%d", resp.StatusCode)
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("deepseek empty choices")
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}
