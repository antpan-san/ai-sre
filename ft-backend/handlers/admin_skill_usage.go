package handlers

import (
	"strconv"
	"time"

	"ft-backend/common/logger"
	"ft-backend/common/response"
	"ft-backend/services"

	"github.com/gin-gonic/gin"
)

// AdminSkillUsageSummary aggregates diagnostic plans, assets, and AI execution records.
func AdminSkillUsageSummary(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days <= 0 || days > 365 {
		days = 30
	}
	since := time.Now().UTC().AddDate(0, 0, -days)
	out, err := services.GetSkillUsageStats(since)
	if err != nil {
		logger.Error("AdminSkillUsageSummary: %v", err)
		response.ServerError(c, "查询运营统计失败")
		return
	}
	response.OK(c, gin.H{
		"since": since,
		"days":  days,
		"stats": out,
	})
}

// AdminSkillUsageCSV exports a simple CSV snapshot for spreadsheets.
func AdminSkillUsageCSV(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days <= 0 || days > 365 {
		days = 30
	}
	since := time.Now().UTC().AddDate(0, 0, -days)
	csv, err := services.SkillUsageCSV(since)
	if err != nil {
		logger.Error("AdminSkillUsageCSV: %v", err)
		response.ServerError(c, "导出失败")
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=skill-usage.csv")
	c.String(200, csv)
}
