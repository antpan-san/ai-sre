package handlers

import (
	"ft-backend/common/logger"
	"ft-backend/common/response"
	"ft-backend/services"
	"strings"

	"github.com/gin-gonic/gin"
)

type cliInstallRecoveryAnalyzeRequest struct {
	Topic     string                 `json:"topic"`
	Operation string                 `json:"operation"`
	Command   string                 `json:"command"`
	Context   map[string]interface{} `json:"context"`
	RequestID string                 `json:"request_id"`
}

type cliInstallRecoveryEventRequest struct {
	RequestID string `json:"request_id" binding:"required"`
	Step      string `json:"step" binding:"required"`
	Status    string `json:"status" binding:"required"`
	Message   string `json:"message"`
}

type cliInstallRecoveryFinishRequest struct {
	RequestID     string `json:"request_id" binding:"required"`
	Status        string `json:"status" binding:"required"`
	Message       string `json:"message"`
	RootCause     string `json:"root_cause"`
	NeedIteration bool   `json:"need_iteration"`
}

// PostCLIInstallRecoveryAnalyze returns a structured recovery plan for CLI allowlist execution.
func PostCLIInstallRecoveryAnalyze(c *gin.Context) {
	ident, ok := resolveCLIBearerIdentity(c)
	if !ok {
		return
	}
	var req cliInstallRecoveryAnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数")
		return
	}
	plan := services.AnalyzeInstallRecovery(req.Topic, req.Operation, req.Command, req.Context)
	if rid := strings.TrimSpace(req.RequestID); rid != "" {
		plan.RequestID = rid
	}
	response.OK(c, plan)
	_ = ident
}

// PostCLIInstallRecoveryEvent records recovery progress (best-effort audit).
func PostCLIInstallRecoveryEvent(c *gin.Context) {
	if _, ok := resolveCLIBearerIdentity(c); !ok {
		return
	}
	var req cliInstallRecoveryEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数")
		return
	}
	logger.Info("install-recovery event request_id=%s step=%s status=%s", req.RequestID, req.Step, req.Status)
	response.OK(c, gin.H{"ok": true})
}

// PostCLIInstallRecoveryFinish records recovery outcome and may trigger iteration.
func PostCLIInstallRecoveryFinish(c *gin.Context) {
	ident, ok := resolveCLIBearerIdentity(c)
	if !ok {
		return
	}
	var req cliInstallRecoveryFinishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数")
		return
	}
	services.FinishInstallRecovery(ident.UserID, req.RequestID, req.Status, req.RootCause, req.NeedIteration)
	response.OK(c, gin.H{"ok": true})
}
