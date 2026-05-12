package handlers

import (
	"fmt"
	"strings"

	"ft-backend/common/response"
	"ft-backend/services"

	"github.com/gin-gonic/gin"
)

// ErrorCodesList returns all structured deploy/runtime root-cause cards.
// Public so that both the OpsFleet console and the `ai-sre` CLI can pull the list,
// avoiding hard-coded duplicates.
func ErrorCodesList(c *gin.Context) {
	reg := services.DefaultSkillRegistry()
	codes := reg.ListErrorCodes()
	response.OK(c, gin.H{
		"codes": codes,
		"count": len(codes),
	})
}

type errorCodeAnalyzeRequest struct {
	Code   string `json:"code"   binding:"required"`
	Detail string `json:"detail"`
}

// ErrorCodeAnalyze resolves an error code to its root-cause card. Lookup is local-first
// (zero LLM call); falls back to AI synthesis only when no curated card exists.
// 设计为客户端 `ai-sre analyze code <CODE>` 与控制台「错误码诊断」共用端点：
//   - 命中已有 SkillErrorCode → 直接返回（< 50ms，无 LLM 调用）
//   - 未命中 → 临时构造一个 "unknown" 占位卡，并标注 LLM-synthesis fallback path（暂未启用）
func ErrorCodeAnalyze(c *gin.Context) {
	var req errorCodeAnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数: "+err.Error())
		return
	}
	code := strings.TrimSpace(strings.ToUpper(req.Code))
	if code == "" {
		response.BadRequest(c, "code 不能为空")
		return
	}
	reg := services.DefaultSkillRegistry()
	ec, owner := reg.LookupErrorCode(code)
	if ec != nil {
		body := gin.H{
			"code":               ec.Code,
			"summary":            ec.Summary,
			"root_cause":         ec.RootCause,
			"typical_evidence":   ec.TypicalEvidence,
			"recovery_one_liner": ec.RecoveryOneLiner,
			"platform_followup":  ec.PlatformFollowup,
			"related_codes":      ec.RelatedCodes,
			"source":             "skill_catalog",
		}
		if owner != nil {
			body["skill_name"] = owner.Pack.Name
			body["skill_source"] = string(owner.Source)
		}
		if strings.TrimSpace(req.Detail) != "" {
			body["detail_echo"] = req.Detail
		}
		response.OK(c, body)
		return
	}
	response.OK(c, gin.H{
		"code":       code,
		"source":     "fallback_unknown",
		"root_cause": fmt.Sprintf("未在错误码目录中找到 %s。请粘贴 detail 至 ai-sre analyze code 让服务端 LLM 推断；或向 deploy/SKILL 团队提交新条目。", code),
	})
}
