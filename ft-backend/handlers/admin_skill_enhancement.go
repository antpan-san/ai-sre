package handlers

import (
	"strconv"

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
