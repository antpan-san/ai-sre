package handlers

import (
	"strconv"
	"time"

	"ft-backend/common/logger"
	"ft-backend/common/response"
	"ft-backend/services"

	"github.com/gin-gonic/gin"
)

// AdminListDiagnoseSamples lists recent DiagnoseSample rows from the skill data dir.
func AdminListDiagnoseSamples(c *gin.Context) {
	topic := c.Query("topic")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "168"))
	if hours <= 0 || hours > 24*90 {
		hours = 168
	}
	since := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
	samples, err := services.ListDiagnoseSamples(services.DefaultSkillRegistry(), topic, limit, since)
	if err != nil {
		logger.Error("AdminListDiagnoseSamples: %v", err)
		response.ServerError(c, "查询诊断样本失败")
		return
	}
	response.OK(c, gin.H{
		"samples": samples,
		"limit":   limit,
		"hours":   hours,
		"topic":   topic,
	})
}

// AdminDiagnoseSampleSummary returns aggregate sample pool metrics.
func AdminDiagnoseSampleSummary(c *gin.Context) {
	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "24"))
	if hours <= 0 || hours > 24*90 {
		hours = 24
	}
	since := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
	sum, err := services.SummarizeDiagnoseSamples(services.DefaultSkillRegistry(), since, hours)
	if err != nil {
		logger.Error("AdminDiagnoseSampleSummary: %v", err)
		response.ServerError(c, "查询样本汇总失败")
		return
	}
	response.OK(c, sum)
}

// AdminBackfillDiagnoseSamples imports JSONL skill samples into PostgreSQL.
func AdminBackfillDiagnoseSamples(c *gin.Context) {
	out, err := services.BackfillDiagnoseSamplesFromJSONL(services.DefaultSkillRegistry())
	if err != nil {
		logger.Error("AdminBackfillDiagnoseSamples: %v", err)
		response.ServerError(c, "回填样本失败")
		return
	}
	response.OK(c, out)
}
