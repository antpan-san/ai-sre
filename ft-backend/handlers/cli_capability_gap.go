package handlers

import (
	"net/http"
	"strings"

	"ft-backend/services"

	"github.com/gin-gonic/gin"
)

type cliCapabilityGapRequest struct {
	Topic   string                        `json:"topic"`
	Context map[string]string             `json:"context"`
	Intent  services.SkillExecutionIntent `json:"intent"`
}

// PostCLICapabilityGap handles CLI requests when a capability is missing from sync.
func PostCLICapabilityGap(c *gin.Context) {
	ident, ok := resolveCLIBearerIdentity(c)
	if !ok {
		return
	}
	var req cliCapabilityGapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效参数: " + err.Error()})
		return
	}
	topic := strings.TrimSpace(req.Topic)
	if topic == "" {
		topic = strings.TrimSpace(req.Intent.Topic)
	}
	if topic == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "topic 不能为空"})
		return
	}
	createdBy := strings.TrimSpace(ident.Username)
	if createdBy == "" {
		createdBy = ident.Subject
	}
	result, err := services.HandleCLICapabilityGap(ident.UserID, createdBy, topic, req.Context, req.Intent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": result, "msg": "success"})
}
