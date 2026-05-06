package handlers

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"ft-backend/common/logger"
	"ft-backend/common/response"
	"ft-backend/services"

	"github.com/gin-gonic/gin"
)

type aiDiagnoseRequest struct {
	Topic     string            `json:"topic" binding:"required"`
	Context   map[string]string `json:"context"`
	Command   string            `json:"command"`
	RequestID string            `json:"request_id"`
}

type aiSkillPack struct {
	Name           string   `json:"name" yaml:"name"`
	DisplayName    string   `json:"display_name" yaml:"display_name"`
	Topics         []string `json:"topics" yaml:"topics"`
	MatchKeywords  []string `json:"match_keywords" yaml:"match_keywords"`
	Input          []string `json:"input" yaml:"input"`
	AnalysisSteps  []string `json:"analysis_steps" yaml:"analysis_steps"`
	OutputFormat   []string `json:"output_format" yaml:"output_format"`
	ExtraGuidance  string   `json:"extra_guidance,omitempty" yaml:"extra_guidance,omitempty"`
	PromptTemplate string   `json:"prompt_template,omitempty" yaml:"prompt_template,omitempty"`
}

type aiDiagnoseResponse struct {
	Source       string                 `json:"source"`
	Answer       string                 `json:"answer"`
	SkillName    string                 `json:"skill_name,omitempty"`
	SkillDisplay string                 `json:"skill_display,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	SkillDraft   *aiSkillPack           `json:"skill_draft,omitempty"`
}

type aiEvolveRequest struct {
	Topic         string            `json:"topic" binding:"required"`
	Context       map[string]string `json:"context"`
	Answer        string            `json:"answer"`
	Feedback      string            `json:"feedback"`
	ExistingSkill string            `json:"existing_skill"`
}

// AIDiagnose runs server-side AI diagnosis and optionally returns a skill draft.
func AIDiagnose(c *gin.Context) {
	var req aiDiagnoseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数: "+err.Error())
		return
	}
	topic := strings.TrimSpace(strings.ToLower(req.Topic))
	if topic == "" {
		response.BadRequest(c, "topic 不能为空")
		return
	}
	answer, err := runServerDeepSeek(c.Request.Context(), buildServerDiagnosePrompt(topic, req.Context))
	if err != nil {
		logger.Error("AIDiagnose deepseek failed: %v", err)
		response.ServerError(c, "服务端 AI 诊断失败: "+err.Error())
		return
	}
	draft := buildSkillDraftFromContext(topic, req.Context)
	if !isSkillDraftValid(draft) {
		draft = nil
	}
	response.OK(c, aiDiagnoseResponse{
		Source:       "server-ai",
		Answer:       answer,
		SkillName:    skillNameForTopic(topic),
		SkillDisplay: "Auto evolved " + strings.ToUpper(topic) + " skill",
		Metadata: map[string]interface{}{
			"request_id": requestIDOrNow(req.RequestID),
			"topic":      topic,
			"fallback":   "server_deepseek",
		},
		SkillDraft: draft,
	})
}

// AISkillsEvolve generates skill draft from diagnosis sample.
func AISkillsEvolve(c *gin.Context) {
	var req aiEvolveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数: "+err.Error())
		return
	}
	topic := strings.TrimSpace(strings.ToLower(req.Topic))
	if topic == "" {
		response.BadRequest(c, "topic 不能为空")
		return
	}
	draft := buildSkillDraftFromContext(topic, req.Context)
	if !isSkillDraftValid(draft) {
		response.ServerError(c, "技能草案生成失败：结果不符合最小 schema")
		return
	}
	response.OK(c, gin.H{
		"draft": draft,
		"metadata": gin.H{
			"topic":          topic,
			"existing_skill": req.ExistingSkill,
			"feedback":       strings.TrimSpace(req.Feedback),
		},
	})
}

func buildServerDiagnosePrompt(topic string, kv map[string]string) string {
	var b strings.Builder
	b.WriteString("你是资深SRE，请输出可执行的中文诊断。\n")
	b.WriteString("要求：1) 先结论 2) 最可能原因排序 3) 最快验证命令 4) 临时缓解与根治建议。\n")
	b.WriteString("topic=" + topic + "\n")
	if len(kv) > 0 {
		keys := make([]string, 0, len(kv))
		for k := range kv {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			b.WriteString(fmt.Sprintf("- %s=%s\n", k, kv[k]))
		}
	}
	return b.String()
}

func runServerDeepSeek(ctx context.Context, userPrompt string) (string, error) {
	cfg := services.LoadServerAIConfig()
	return services.DiagnoseWithDeepSeek(ctx, cfg, userPrompt)
}

func buildSkillDraftFromContext(topic string, kv map[string]string) *aiSkillPack {
	draft := services.BuildSkillDraft(topic, kv)
	if draft == nil {
		return nil
	}
	return &aiSkillPack{
		Name:           draft.Name,
		DisplayName:    draft.DisplayName,
		Topics:         draft.Topics,
		MatchKeywords:  draft.MatchKeywords,
		Input:          draft.Input,
		AnalysisSteps:  draft.AnalysisSteps,
		OutputFormat:   draft.OutputFormat,
		ExtraGuidance:  draft.ExtraGuidance,
		PromptTemplate: draft.PromptTemplate,
	}
}

func isSkillDraftValid(p *aiSkillPack) bool {
	return services.ValidateSkillDraft(&services.SkillPack{
		Name:           p.Name,
		DisplayName:    p.DisplayName,
		Topics:         p.Topics,
		MatchKeywords:  p.MatchKeywords,
		Input:          p.Input,
		AnalysisSteps:  p.AnalysisSteps,
		OutputFormat:   p.OutputFormat,
		ExtraGuidance:  p.ExtraGuidance,
		PromptTemplate: p.PromptTemplate,
	})
}

func skillNameForTopic(topic string) string {
	return services.SkillNameForTopic(topic)
}

func requestIDOrNow(id string) string {
	id = strings.TrimSpace(id)
	if id != "" {
		return id
	}
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}
