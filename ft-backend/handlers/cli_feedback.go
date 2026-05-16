package handlers

import (
	"ft-backend/common/logger"
	"ft-backend/common/response"
	"ft-backend/services"

	"github.com/gin-gonic/gin"
)

type cliFeedbackAnalyzeRequest struct {
	Topic     string                 `json:"topic"`
	Command   string                 `json:"command"`
	Summary   string                 `json:"summary"`
	Context   map[string]interface{} `json:"context"`
	RequestID string                 `json:"request_id"`
}

// PostCLIFeedbackAnalyze accepts CLI feedback; response exposes only public fields.
func PostCLIFeedbackAnalyze(c *gin.Context) {
	ident, ok := resolveCLIBearerIdentity(c)
	if !ok {
		return
	}
	var req cliFeedbackAnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数")
		return
	}
	out, err := services.AnalyzeCLIFeedback(ident.UserID, ident.CLIBindingID, req.Topic, req.Command, req.Summary, req.Context)
	if err != nil {
		logger.Error("PostCLIFeedbackAnalyze: %v", err)
		response.ServerError(c, "反馈处理失败")
		return
	}
	response.OK(c, out)
}
