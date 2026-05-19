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
	"ft-backend/models"
)

// AutoIterationDingTalkKind classifies auto-iteration DingTalk notifications.
type AutoIterationDingTalkKind string

const (
	DingTalkKindWorkerCompleted      AutoIterationDingTalkKind = "worker_completed"
	DingTalkKindWorkerFailed         AutoIterationDingTalkKind = "worker_failed"
	DingTalkKindWorkerAwaitingReview AutoIterationDingTalkKind = "worker_awaiting_review"
	DingTalkKindTaskCreated          AutoIterationDingTalkKind = "task_created"
	DingTalkKindApproved             AutoIterationDingTalkKind = "approved"
	DingTalkKindCLIFeedbackQueued    AutoIterationDingTalkKind = "cli_feedback"
	DingTalkKindResend               AutoIterationDingTalkKind = "resend"
)

type autoIterationDingTalkFields struct {
	Title   string
	TaskID  string
	Source  string
	Topic   string
	Status  string
	Summary string
	Actor   string
	Extra   string
}

func autoIterationSourceLabelForDingTalk(source string) string {
	switch source {
	case models.AutoIterationSourceManual:
		return "页面手动"
	case models.AutoIterationSourceCLIFeedback:
		return "CLI 反馈"
	case models.AutoIterationSourceCapabilityGap:
		return "能力缺口"
	case models.AutoIterationSourceSkillRefine:
		return "技能精炼"
	case models.AutoIterationSourceRuleCandidate:
		return "规则候选"
	case models.AutoIterationSourceDiagnosisGap:
		return "诊断不足"
	case models.AutoIterationSourceAICostReduce:
		return "AI 成本优化"
	default:
		if source == "" {
			return "-"
		}
		return source
	}
}

func buildAutoIterationDingTalkMarkdown(kind AutoIterationDingTalkKind, f autoIterationDingTalkFields) (title, markdown string) {
	headline := "自动迭代"
	icon := "📋"
	switch kind {
	case DingTalkKindWorkerCompleted:
		icon, headline = "✅", "开发任务已完成"
	case DingTalkKindWorkerFailed:
		icon, headline = "❌", "开发任务失败"
	case DingTalkKindWorkerAwaitingReview:
		icon, headline = "⏳", "开发任务待审批"
	case DingTalkKindTaskCreated:
		icon, headline = "🆕", "新开发任务"
	case DingTalkKindApproved:
		icon, headline = "✔️", "任务已批准"
	case DingTalkKindCLIFeedbackQueued:
		icon, headline = "📥", "CLI 反馈已入队"
	case DingTalkKindResend:
		icon, headline = "🔔", "任务状态提醒"
	}
	title = fmt.Sprintf("%s %s", icon, headline)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("### %s %s\n\n", icon, headline))
	if t := strings.TrimSpace(f.Title); t != "" {
		b.WriteString(fmt.Sprintf("**标题**：%s  \n", escapeDingTalkMarkdownInline(t)))
	}
	if id := strings.TrimSpace(f.TaskID); id != "" {
		b.WriteString(fmt.Sprintf("**任务 ID**：`%s`  \n", escapeDingTalkMarkdownInline(id)))
	}
	if src := strings.TrimSpace(f.Source); src != "" {
		b.WriteString(fmt.Sprintf("**来源**：%s  \n", escapeDingTalkMarkdownInline(src)))
	}
	if topic := strings.TrimSpace(f.Topic); topic != "" {
		b.WriteString(fmt.Sprintf("**Topic**：%s  \n", escapeDingTalkMarkdownInline(topic)))
	}
	if st := strings.TrimSpace(f.Status); st != "" {
		b.WriteString(fmt.Sprintf("**状态**：`%s`  \n", escapeDingTalkMarkdownInline(st)))
	}
	if actor := strings.TrimSpace(f.Actor); actor != "" {
		if kind == DingTalkKindCLIFeedbackQueued {
			b.WriteString(fmt.Sprintf("**分类**：%s  \n", escapeDingTalkMarkdownInline(actor)))
		} else {
			b.WriteString(fmt.Sprintf("**操作人**：%s  \n", escapeDingTalkMarkdownInline(actor)))
		}
	}
	if extra := strings.TrimSpace(f.Extra); extra != "" {
		b.WriteString(fmt.Sprintf("**备注**：%s  \n", escapeDingTalkMarkdownInline(extra)))
	}
	if summary := strings.TrimSpace(f.Summary); summary != "" {
		b.WriteString("\n**摘要**  \n")
		b.WriteString("> ")
		b.WriteString(strings.ReplaceAll(escapeDingTalkMarkdownInline(limitAuditText(summary, 800)), "\n", "\n> "))
		b.WriteString("\n")
	}
	b.WriteString("\n---\n")
	b.WriteString("*OpsFleet 自动迭代*")
	return title, b.String()
}

func escapeDingTalkMarkdownInline(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "#", "\\#")
	s = strings.ReplaceAll(s, "*", "\\*")
	s = strings.ReplaceAll(s, "`", "'")
	s = strings.ReplaceAll(s, "[", "(")
	s = strings.ReplaceAll(s, "]", ")")
	return s
}

func ensureDingTalkKeyword(text, keyword string) string {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return text
	}
	if strings.Contains(text, keyword) {
		return text
	}
	return strings.TrimRight(text, "\n") + "\n\n`" + keyword + "`"
}

// SendDingTalkMarkdown posts a markdown message to a DingTalk custom robot webhook.
func SendDingTalkMarkdown(webhook, keyword, title, markdown string) error {
	webhook = strings.TrimSpace(webhook)
	if webhook == "" {
		return nil
	}
	title = strings.TrimSpace(title)
	markdown = strings.TrimSpace(markdown)
	if title == "" && markdown == "" {
		return nil
	}
	markdown = ensureDingTalkKeyword(markdown, keyword)
	title = ensureDingTalkKeyword(title, keyword)
	payload, err := json.Marshal(map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": limitAuditText(title, 120),
			"text":  limitAuditText(markdown, 18000),
		},
	})
	if err != nil {
		return err
	}
	return postDingTalkWebhook(webhook, payload, title)
}

// SendDingTalkText posts plain text (fallback / tests).
func SendDingTalkText(webhook, keyword, title, body string) error {
	webhook = strings.TrimSpace(webhook)
	if webhook == "" {
		return nil
	}
	content := strings.TrimSpace(title)
	if b := strings.TrimSpace(body); b != "" {
		if content != "" {
			content += "\n"
		}
		content += b
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
	return postDingTalkWebhook(webhook, payload, title)
}

func postDingTalkWebhook(webhook string, payload []byte, logTitle string) error {
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
	logger.Info("dingtalk notify ok: %s", logTitle)
	return nil
}

// SendAutoIterationDingTalkMarkdown sends a formatted auto-iteration notification.
func SendAutoIterationDingTalkMarkdown(kind AutoIterationDingTalkKind, f autoIterationDingTalkFields) error {
	cfg := config.ResolvedAutoIterationConfig()
	title, md := buildAutoIterationDingTalkMarkdown(kind, f)
	return SendDingTalkMarkdown(cfg.DingTalkWebhook, cfg.DingTalkKeyword, title, md)
}

// SendAutoIterationDingTalk posts a simple auto-iteration notification (markdown).
func SendAutoIterationDingTalk(title, body string) error {
	return SendAutoIterationDingTalkMarkdown(DingTalkKindResend, autoIterationDingTalkFields{
		Title:   title,
		Summary: body,
	})
}
