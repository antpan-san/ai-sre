package handlers

import (
	"context"
	"errors"
	"strings"
	"time"

	"ft-backend/common/logger"
	"ft-backend/common/response"
	"ft-backend/services"

	"github.com/gin-gonic/gin"
)

// AISkillsList returns all registered skills (builtin + generated).
func AISkillsList(c *gin.Context) {
	reg := services.DefaultSkillRegistry()
	response.OK(c, gin.H{
		"skills":   reg.List(),
		"data_dir": reg.DataDir(),
	})
}

// AISkillsGet returns a single registered skill (full pack) by pack name.
func AISkillsGet(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		response.BadRequest(c, "无效技能标识")
		return
	}
	reg := services.DefaultSkillRegistry()
	rs := reg.LookupByName(name)
	if rs == nil {
		response.NotFound(c, "技能不存在或已下线")
		return
	}
	response.OK(c, gin.H{"skill": rs})
}

type aiSkillsRefineRequest struct {
	Topic        string `json:"topic" binding:"required"`
	MaxSamples   int    `json:"max_samples"`
	MaxFeedback  int    `json:"max_feedback"`
	UserHint     string `json:"user_hint"`
	DryRun       bool   `json:"dry_run"`
	TimeoutSec   int    `json:"timeout_sec"`
}

// AISkillsRefine asks the configured LLM to produce a refined skill pack for a topic.
func AISkillsRefine(c *gin.Context) {
	var req aiSkillsRefineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数: "+err.Error())
		return
	}
	topic := strings.TrimSpace(strings.ToLower(req.Topic))
	if topic == "" {
		response.BadRequest(c, "topic 不能为空")
		return
	}
	reg := services.DefaultSkillRegistry()
	if reg.DataDir() == "" && !req.DryRun {
		response.ServerError(c, "OPSFLEET_AI_SKILL_DATA_DIR 未配置：无可写持久目录")
		return
	}
	timeout := time.Duration(req.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout+5*time.Second)
	defer cancel()
	result, err := services.RefineSkill(ctx, reg, services.RefineSkillInput{
		Topic:           topic,
		MaxSamples:      req.MaxSamples,
		MaxFeedback:     req.MaxFeedback,
		UserHint:        req.UserHint,
		ForceLLMTimeout: timeout,
	})
	if err != nil {
		logger.Error("AISkillsRefine topic=%s failed: %v", topic, err)
		response.ServerError(c, "技能精炼失败: "+err.Error())
		return
	}
	response.OK(c, result)
}

type aiSkillsFeedbackRequest struct {
	Topic     string `json:"topic" binding:"required"`
	SkillName string `json:"skill_name"`
	RequestID string `json:"request_id"`
	Helpful   *bool  `json:"helpful"`
	Note      string `json:"note"`
}

// AISkillsFeedback receives a feedback record from CLI clients.
func AISkillsFeedback(c *gin.Context) {
	var req aiSkillsFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数: "+err.Error())
		return
	}
	topic := strings.TrimSpace(strings.ToLower(req.Topic))
	if topic == "" {
		response.BadRequest(c, "topic 不能为空")
		return
	}
	reg := services.DefaultSkillRegistry()
	if err := reg.AppendFeedback(services.SkillFeedback{
		Topic:     topic,
		SkillName: strings.TrimSpace(req.SkillName),
		RequestID: strings.TrimSpace(req.RequestID),
		Helpful:   req.Helpful,
		Note:      strings.TrimSpace(req.Note),
	}); err != nil {
		if errors.Is(err, errDataDirUnset) {
			response.OK(c, gin.H{"recorded": false, "reason": "server has no skill data dir; feedback dropped"})
			return
		}
		response.ServerError(c, "记录反馈失败: "+err.Error())
		return
	}
	response.OK(c, gin.H{"recorded": true})
}

// errDataDirUnset is kept for symmetry; AppendFeedback currently returns nil when not configured.
var errDataDirUnset = errors.New("data dir unset")
