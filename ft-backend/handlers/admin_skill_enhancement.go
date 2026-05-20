package handlers

import (
	"strconv"
	"strings"

	"ft-backend/common/logger"
	"ft-backend/common/response"
	"ft-backend/services"

	"github.com/gin-gonic/gin"
)

// AdminListSkillEnhancementReviews lists recent post-AI enhancement reviews.
func AdminListSkillEnhancementReviews(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	openOnly := c.DefaultQuery("open_only", "true") != "false"
	reviews, err := services.ListEnhancementReviews(services.DefaultSkillRegistry(), limit, openOnly)
	if err != nil {
		logger.Error("AdminListSkillEnhancementReviews: %v", err)
		response.ServerError(c, "查询技能包增强审查失败")
		return
	}
	response.OK(c, gin.H{"reviews": reviews, "limit": limit, "open_only": openOnly})
}

// AdminSkillEnhancementSummary returns aggregate pending enhancement metrics.
func AdminSkillEnhancementSummary(c *gin.Context) {
	recentLimit, _ := strconv.Atoi(c.DefaultQuery("recent", "20"))
	if recentLimit <= 0 || recentLimit > 100 {
		recentLimit = 20
	}
	sum, err := services.SummarizeEnhancementReviews(services.DefaultSkillRegistry(), recentLimit)
	if err != nil {
		logger.Error("AdminSkillEnhancementSummary: %v", err)
		response.ServerError(c, "查询增强审查汇总失败")
		return
	}
	response.OK(c, sum)
}

type adminEnhancementStatusRequest struct {
	RequestID string `json:"request_id"`
	ReviewKey string `json:"review_key"`
	Topic     string `json:"topic" binding:"required"`
	Status    string `json:"status" binding:"required"`
	Note      string `json:"note"`
}

// AdminUpdateSkillEnhancementStatus marks a review as refined or dismissed.
func AdminUpdateSkillEnhancementStatus(c *gin.Context) {
	var req adminEnhancementStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数")
		return
	}
	req.RequestID = strings.TrimSpace(req.RequestID)
	req.ReviewKey = strings.TrimSpace(req.ReviewKey)
	if req.RequestID == "" && req.ReviewKey == "" {
		response.BadRequest(c, "request_id 或 review_key 必填其一")
		return
	}
	if err := services.UpdateEnhancementReviewStatus(services.DefaultSkillRegistry(), req.RequestID, req.ReviewKey, req.Topic, req.Status, req.Note); err != nil {
		logger.Error("AdminUpdateSkillEnhancementStatus: %v", err)
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"updated": true, "status": req.Status})
}

// AdminLookupExecutionByRequestID resolves client execution id from AI request_id.
func AdminLookupExecutionByRequestID(c *gin.Context) {
	rid := strings.TrimSpace(c.Param("request_id"))
	if rid == "" {
		response.BadRequest(c, "request_id 不能为空")
		return
	}
	parentID, childID, _ := services.FindClientExecutionIDByRequestID(rid)
	response.OK(c, gin.H{
		"request_id":           rid,
		"execution_id":         parentID,
		"ai_child_execution_id": childID,
	})
}
