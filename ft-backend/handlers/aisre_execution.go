package handlers

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"ft-backend/common/logger"
	"ft-backend/common/response"
	"ft-backend/models"
	"ft-backend/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListAISreExecutions lists ai-sre client execution sessions for the console hub.
func ListAISreExecutions(c *gin.Context) {
	p := response.GetPagination(c)
	roleVal, _ := c.Get("role")
	role, _ := roleVal.(string)
	userVal, _ := c.Get("username")
	username, _ := userVal.(string)
	tid := uuid.MustParse(models.DefaultTenantID)

	q := services.ClientExecutionListQuery{
		TenantID:      tid,
		Role:          role,
		Username:      username,
		View:          strings.TrimSpace(c.Query("view")),
		Status:        strings.TrimSpace(c.Query("status")),
		Topic:         strings.TrimSpace(c.Query("topic")),
		Target:        strings.TrimSpace(c.Query("target")),
		SkillPack:     strings.TrimSpace(c.Query("skillPack")),
		PackKey:       strings.TrimSpace(c.Query("packKey")),
		UsedAI:        strings.TrimSpace(c.Query("usedAI")),
		Severity:      strings.TrimSpace(c.Query("severity")),
		ClientVersion: strings.TrimSpace(c.Query("clientVersion")),
		Machine:       strings.TrimSpace(c.Query("machine")),
		HasAutoIter:   c.Query("hasAutoIteration") == "true" || c.Query("hasAutoIteration") == "1",
		StartDate:     strings.TrimSpace(c.Query("startDate")),
		EndDate:       strings.TrimSpace(c.Query("endDate")),
		Page:          p.Page,
		PageSize:      p.PageSize,
	}
	items, total, err := services.ListClientExecutions(q)
	if err != nil {
		logger.Error("ListAISreExecutions: %v", err)
		response.ServerError(c, "查询客户端执行失败")
		return
	}
	response.OKPage(c, items, total)
}

// GetAISreExecutionStats returns 24h hub counters.
func GetAISreExecutionStats(c *gin.Context) {
	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "24"))
	if hours <= 0 || hours > 168 {
		hours = 24
	}
	roleVal, _ := c.Get("role")
	role, _ := roleVal.(string)
	userVal, _ := c.Get("username")
	username, _ := userVal.(string)
	tid := uuid.MustParse(models.DefaultTenantID)
	since := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
	st, err := services.GetClientExecutionStats(tid, role, username, since)
	if err != nil {
		logger.Error("GetAISreExecutionStats: %v", err)
		response.ServerError(c, "查询统计失败")
		return
	}
	response.OK(c, gin.H{"since": since, "hours": hours, "stats": st})
}

// GetAISreExecutionDetail returns one client execution session with children and timeline.
func GetAISreExecutionDetail(c *gin.Context) {
	id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		response.BadRequest(c, "无效的执行 ID")
		return
	}
	roleVal, _ := c.Get("role")
	role, _ := roleVal.(string)
	userVal, _ := c.Get("username")
	username, _ := userVal.(string)
	detail, err := services.GetClientExecutionDetail(id, role, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "执行记录不存在或无权查看")
			return
		}
		logger.Error("GetAISreExecutionDetail: %v", err)
		response.ServerError(c, "查询执行详情失败")
		return
	}
	response.OK(c, detail)
}

type executionFeedbackRequest struct {
	Helpful bool   `json:"helpful"`
	Note    string `json:"note"`
}

// PostAISreExecutionFeedback records user helpful/unhelpful feedback for an execution session.
func PostAISreExecutionFeedback(c *gin.Context) {
	id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		response.BadRequest(c, "无效的执行 ID")
		return
	}
	var req executionFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数")
		return
	}
	roleVal, _ := c.Get("role")
	role, _ := roleVal.(string)
	userVal, _ := c.Get("username")
	username, _ := userVal.(string)
	out, err := services.SubmitExecutionSkillFeedback(id, role, username, req.Helpful, req.Note)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "执行记录不存在或无权操作")
			return
		}
		logger.Error("PostAISreExecutionFeedback: %v", err)
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, out)
}
