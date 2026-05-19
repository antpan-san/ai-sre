package handlers

import (
	"context"
	"strings"
	"time"

	"ft-backend/common/logger"
	"ft-backend/common/response"
	"ft-backend/services"

	"github.com/gin-gonic/gin"
)

type adminSkillsRefineRequest struct {
	Topic       string `json:"topic" binding:"required"`
	MaxSamples  int    `json:"max_samples"`
	MaxFeedback int    `json:"max_feedback"`
	UserHint    string `json:"user_hint"`
	DryRun      bool   `json:"dry_run"`
	TimeoutSec  int    `json:"timeout_sec"`
}

// AdminRefineSkill triggers LLM skill pack refinement for super_admin.
func AdminRefineSkill(c *gin.Context) {
	var req adminSkillsRefineRequest
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
		response.ServerError(c, "OPSFLEET_AI_SKILL_DATA_DIR 未配置")
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
		DryRun:          req.DryRun,
	})
	if err != nil {
		logger.Error("AdminRefineSkill topic=%s: %v", topic, err)
		response.ServerError(c, "技能精炼失败: "+err.Error())
		return
	}
	response.OK(c, result)
}
