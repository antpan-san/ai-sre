package handlers

import (
	"strings"

	"ft-backend/common/response"
	"ft-backend/middleware"
	"ft-backend/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CodeAgentHeartbeat(c *gin.Context) {
	bindingID := middleware.CodeAgentBindingID(c)
	if bindingID == uuid.Nil {
		response.Unauthorized(c, "未认证")
		return
	}
	if err := services.CodeAgentHeartbeat(bindingID); err != nil {
		response.ServerError(c, "心跳失败")
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func CodeAgentPullTask(c *gin.Context) {
	bindingID := middleware.CodeAgentBindingID(c)
	task, err := services.CodeAgentPullTask(bindingID)
	if err != nil {
		response.ServerError(c, "拉取任务失败")
		return
	}
	if task == nil {
		response.OK(c, gin.H{"task": nil})
		return
	}
	response.OK(c, gin.H{"task": services.CodeAgentTaskView(task)})
}

type codeAgentEventReq struct {
	Message string                 `json:"message"`
	Payload map[string]interface{} `json:"payload"`
}

func CodeAgentPostTaskEvents(c *gin.Context) {
	bindingID := middleware.CodeAgentBindingID(c)
	id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		response.BadRequest(c, "无效任务 ID")
		return
	}
	var req codeAgentEventReq
	_ = c.ShouldBindJSON(&req)
	if err := services.CodeAgentReportEvent(id, bindingID, req.Message, req.Payload); err != nil {
		response.ServerError(c, "上报事件失败")
		return
	}
	response.OK(c, gin.H{"ok": true})
}

type codeAgentResultReq struct {
	Success          bool   `json:"success"`
	Summary          string `json:"summary"`
	GitHubSync       string `json:"github_sync"`
	DeployStatus     string `json:"deploy_status"`
	RollbackRequired bool   `json:"rollback_required"`
}

func CodeAgentPostTaskResult(c *gin.Context) {
	bindingID := middleware.CodeAgentBindingID(c)
	id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		response.BadRequest(c, "无效任务 ID")
		return
	}
	var req codeAgentResultReq
	_ = c.ShouldBindJSON(&req)
	if err := services.CodeAgentReportResult(id, bindingID, services.CodeAgentTaskResult{
		Success:          req.Success,
		Summary:          req.Summary,
		GitHubSync:       req.GitHubSync,
		DeployStatus:     req.DeployStatus,
		RollbackRequired: req.RollbackRequired,
	}); err != nil {
		response.ServerError(c, "上报结果失败")
		return
	}
	response.OK(c, gin.H{"ok": true})
}
