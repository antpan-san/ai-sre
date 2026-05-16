package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ft-backend/common/config"
	"ft-backend/common/logger"
)

// SendDingTalkText posts to a DingTalk custom robot webhook (server-side only).
// keyword is required when the robot has a security keyword (prepended if missing from body).
func SendDingTalkText(webhook, keyword, title, body string) error {
	webhook = strings.TrimSpace(webhook)
	if webhook == "" {
		return nil
	}
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" && body == "" {
		return nil
	}
	content := title
	if body != "" {
		if content != "" {
			content += "\n"
		}
		content += body
	}
	content = ensureDingTalkKeyword(content, keyword)
	payload, err := json.Marshal(map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": limitAuditText(content, 4000),
		},
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, webhook, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("dingtalk status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	_ = json.Unmarshal(raw, &out)
	if out.ErrCode != 0 {
		return fmt.Errorf("dingtalk errcode=%d errmsg=%s", out.ErrCode, out.ErrMsg)
	}
	logger.Info("dingtalk notify ok: %s", title)
	return nil
}

func ensureDingTalkKeyword(content, keyword string) string {
	kw := strings.TrimSpace(keyword)
	if kw == "" || strings.Contains(content, kw) {
		return content
	}
	return kw + "\n" + content
}

// SendAutoIterationDingTalk posts to the auto-iteration robot webhook.
func SendAutoIterationDingTalk(title, body string) error {
	cfg := config.ResolvedAutoIterationConfig()
	return SendDingTalkText(cfg.DingTalkWebhook, "", title, body)
}
