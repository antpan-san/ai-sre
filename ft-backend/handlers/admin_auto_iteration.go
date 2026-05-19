package handlers

import (
	"errors"
	"strconv"
	"strings"

	"ft-backend/common/logger"
	"ft-backend/common/response"
	"ft-backend/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func auditAutoIteration(c *gin.Context, operation, id, status string, details map[string]interface{}) {
	if err := services.RecordOperationLog(c, operation, "auto_iteration", id, status, details); err != nil {
		logger.Error("audit auto_iteration %s: %v", operation, err)
	}
}

func AdminListAutoIterations(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	items, total, err := services.ListAutoIterations(services.AutoIterationListFilter{
		Status:   c.Query("status"),
		Topic:    c.Query("topic"),
		Source:   c.Query("source"),
		Keyword:  c.Query("keyword"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		logger.Error("AdminListAutoIterations: %v", err)
		response.ServerError(c, "查询自动迭代任务失败")
		return
	}
	response.OKPage(c, items, total)
}

func AdminGetAutoIteration(c *gin.Context) {
	id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		response.BadRequest(c, "无效任务 ID")
		return
	}
	row, err := services.GetAutoIteration(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "任务不存在")
			return
		}
		response.ServerError(c, "查询失败")
		return
	}
	events, _ := services.ListAutoIterationEvents(id, uuid.Nil, 50)
	response.OK(c, gin.H{"iteration": row, "events": events})
}

func AdminGetAutoIterationSamples(c *gin.Context) {
	id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		response.BadRequest(c, "无效任务 ID")
		return
	}
	ctx, err := services.GetAutoIterationSampleContext(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "任务不存在")
			return
		}
		logger.Error("AdminGetAutoIterationSamples: %v", err)
		response.ServerError(c, "查询触发样本失败")
		return
	}
	response.OK(c, ctx)
}

func AdminGetAutoIterationSettings(c *gin.Context) {
	settings, err := services.GetAutoIterationSettings()
	if err != nil {
		response.ServerError(c, "读取设置失败")
		return
	}
	response.OK(c, gin.H{"settings": settings})
}

type updateAutoIterationSettingsReq struct {
	Enabled                  *bool  `json:"enabled"`
	MaxConcurrent            *int   `json:"max_concurrent"`
	HighRiskRequiresApproval *bool  `json:"high_risk_requires_approval"`
	AutoDispatchEnabled      *bool  `json:"auto_dispatch_enabled"`
	LowRiskAutoDeployEnabled *bool  `json:"low_risk_auto_deploy_enabled"`
	GitHubSyncEnabled        *bool  `json:"github_sync_enabled"`
	DingTalkNotifyEnabled    *bool  `json:"dingtalk_notify_enabled"`
	Notes                    string `json:"notes"`
}

func AdminUpdateAutoIterationSettings(c *gin.Context) {
	var req updateAutoIterationSettingsReq
	_ = c.ShouldBindJSON(&req)
	name, _ := c.Get("username")
	username, _ := name.(string)
	settings, err := services.UpdateAutoIterationSettings(
		req.Enabled, req.MaxConcurrent, req.HighRiskRequiresApproval,
		req.AutoDispatchEnabled, req.LowRiskAutoDeployEnabled, req.GitHubSyncEnabled, req.DingTalkNotifyEnabled,
		req.Notes, username)
	if err != nil {
		response.ServerError(c, "更新设置失败")
		return
	}
	auditAutoIteration(c, "auto_iteration.settings.update", "", "success", map[string]interface{}{
		"enabled": settings.Enabled,
	})
	response.OK(c, gin.H{"settings": settings})
}

type manualAutoIterationReq struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Command     string `json:"command"`
	Topic       string `json:"topic"`
	AutoStart   bool   `json:"auto_start"`
}

func AdminCreateManualAutoIteration(c *gin.Context) {
	var req manualAutoIterationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数")
		return
	}
	uid, _ := c.Get("userID")
	userID, _ := uid.(uuid.UUID)
	name, _ := c.Get("username")
	username, _ := name.(string)
	row, err := services.CreateManualAutoIteration(req.Title, req.Description, req.Command, req.Topic, username, userID, req.AutoStart)
	if err != nil {
		if errors.Is(err, services.ErrAutoIterationDisabled) {
			response.BadRequest(c, "自动迭代未开启")
			return
		}
		response.ServerError(c, err.Error())
		return
	}
	auditAutoIteration(c, "auto_iteration.manual.create", row.ID.String(), "success", map[string]interface{}{
		"auto_start": req.AutoStart,
	})
	response.Created(c, gin.H{"iteration": row})
}

func iterationIDParam(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		response.BadRequest(c, "无效任务 ID")
		return uuid.Nil, false
	}
	return id, true
}

func actorName(c *gin.Context) string {
	name, _ := c.Get("username")
	s, _ := name.(string)
	return s
}

func AdminStartAutoIteration(c *gin.Context) {
	id, ok := iterationIDParam(c)
	if !ok {
		return
	}
	row, err := services.StartAutoIteration(id, actorName(c))
	if mapIterationActionErr(c, err) {
		return
	}
	auditAutoIteration(c, "auto_iteration.start", id.String(), "success", nil)
	response.OK(c, gin.H{"iteration": row})
}

func AdminPauseAutoIteration(c *gin.Context) {
	id, ok := iterationIDParam(c)
	if !ok {
		return
	}
	row, err := services.PauseAutoIteration(id, actorName(c))
	if mapIterationActionErr(c, err) {
		return
	}
	auditAutoIteration(c, "auto_iteration.pause", id.String(), "success", nil)
	response.OK(c, gin.H{"iteration": row})
}

func AdminResumeAutoIteration(c *gin.Context) {
	id, ok := iterationIDParam(c)
	if !ok {
		return
	}
	row, err := services.ResumeAutoIteration(id, actorName(c))
	if mapIterationActionErr(c, err) {
		return
	}
	auditAutoIteration(c, "auto_iteration.resume", id.String(), "success", nil)
	response.OK(c, gin.H{"iteration": row})
}

func AdminCancelAutoIteration(c *gin.Context) {
	id, ok := iterationIDParam(c)
	if !ok {
		return
	}
	row, err := services.CancelAutoIteration(id, actorName(c))
	if mapIterationActionErr(c, err) {
		return
	}
	auditAutoIteration(c, "auto_iteration.cancel", id.String(), "success", nil)
	response.OK(c, gin.H{"iteration": row})
}

type approveRejectReq struct {
	Notes  string `json:"notes"`
	Reason string `json:"reason"`
	Force  bool   `json:"force"`
}

func AdminApproveAutoIteration(c *gin.Context) {
	id, ok := iterationIDParam(c)
	if !ok {
		return
	}
	var req approveRejectReq
	_ = c.ShouldBindJSON(&req)
	uid, _ := c.Get("userID")
	userID, _ := uid.(uuid.UUID)
	row, err := services.ApproveAutoIteration(id, userID, actorName(c), req.Notes, req.Force)
	if mapIterationActionErr(c, err) {
		return
	}
	auditAutoIteration(c, "auto_iteration.approve", id.String(), "success", map[string]interface{}{"notes": req.Notes})
	response.OK(c, gin.H{"iteration": row})
}

func AdminRejectAutoIteration(c *gin.Context) {
	id, ok := iterationIDParam(c)
	if !ok {
		return
	}
	var req approveRejectReq
	_ = c.ShouldBindJSON(&req)
	row, err := services.RejectAutoIteration(id, actorName(c), req.Reason)
	if mapIterationActionErr(c, err) {
		return
	}
	auditAutoIteration(c, "auto_iteration.reject", id.String(), "success", nil)
	response.OK(c, gin.H{"iteration": row})
}

func AdminRollbackAutoIteration(c *gin.Context) {
	id, ok := iterationIDParam(c)
	if !ok {
		return
	}
	var req approveRejectReq
	_ = c.ShouldBindJSON(&req)
	row, err := services.RollbackAutoIteration(id, actorName(c), req.Reason)
	if mapIterationActionErr(c, err) {
		return
	}
	auditAutoIteration(c, "auto_iteration.rollback", id.String(), "success", nil)
	response.OK(c, gin.H{"iteration": row})
}

func AdminRunAutoIterationTests(c *gin.Context) {
	id, ok := iterationIDParam(c)
	if !ok {
		return
	}
	row, err := services.RunAutoIterationTests(id, actorName(c))
	if mapIterationActionErr(c, err) {
		return
	}
	auditAutoIteration(c, "auto_iteration.run_tests", id.String(), "success", nil)
	response.OK(c, gin.H{"iteration": row})
}

func AdminSyncAutoIterationGitHub(c *gin.Context) {
	id, ok := iterationIDParam(c)
	if !ok {
		return
	}
	row, err := services.SyncAutoIterationGitHub(id, actorName(c))
	if mapIterationActionErr(c, err) {
		return
	}
	auditAutoIteration(c, "auto_iteration.sync_github", id.String(), "success", nil)
	response.OK(c, gin.H{"iteration": row})
}

func AdminResendAutoIterationNotification(c *gin.Context) {
	id, ok := iterationIDParam(c)
	if !ok {
		return
	}
	row, err := services.ResendAutoIterationNotification(id, actorName(c))
	if mapIterationActionErr(c, err) {
		return
	}
	auditAutoIteration(c, "auto_iteration.resend_notification", id.String(), "success", nil)
	response.OK(c, gin.H{"iteration": row})
}

func mapIterationActionErr(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.NotFound(c, "任务不存在")
		return true
	}
	if errors.Is(err, services.ErrAutoIterationInvalidState) {
		response.BadRequest(c, "当前状态不允许该操作")
		return true
	}
	response.ServerError(c, err.Error())
	return true
}
