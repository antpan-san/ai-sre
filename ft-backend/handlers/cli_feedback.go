package handlers

import (
	"ft-backend/common/logger"
	"ft-backend/common/response"
	"ft-backend/services"
	"strings"

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
	out, err := services.AnalyzeCLIFeedback(ident.UserID, ident.CLIBindingID, req.Topic, req.Command, req.Summary, mergeCLIFeedbackContext(req))
	if err != nil {
		logger.Error("PostCLIFeedbackAnalyze: %v", err)
		response.ServerError(c, "反馈处理失败")
		return
	}
	response.OK(c, out)
}

func mergeCLIFeedbackContext(req cliFeedbackAnalyzeRequest) map[string]interface{} {
	ctx := req.Context
	if ctx == nil {
		ctx = map[string]interface{}{}
	}
	if rid := strings.TrimSpace(req.RequestID); rid != "" {
		ctx["request_id"] = rid
	}
	return ctx
}
