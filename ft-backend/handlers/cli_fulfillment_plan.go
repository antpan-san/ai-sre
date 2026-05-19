package handlers

import (
	"net/http"
	"strings"

	"ft-backend/services"

	"github.com/gin-gonic/gin"
)

type cliFulfillmentPlanRequest struct {
	Command              string                        `json:"command"`
	CommandCatalogDigest string                        `json:"command_catalog_digest"`
	Topic                string                        `json:"topic"`
	Context              map[string]string             `json:"context"`
	Intent               services.SkillExecutionIntent `json:"intent"`
	FailureKind          string                        `json:"failure_kind"`
	FailureMessage       string                        `json:"failure_message"`
}

// PostCLIFulfillmentPlan handles capability-layer CLI fulfillment decisions.
func PostCLIFulfillmentPlan(c *gin.Context) {
	ident, ok := resolveCLIBearerIdentity(c)
	if !ok {
		return
	}
	var req cliFulfillmentPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效参数: " + err.Error()})
		return
	}
	if strings.TrimSpace(req.CommandCatalogDigest) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "缺少 command_catalog_digest"})
		return
	}
	topic := strings.TrimSpace(req.Topic)
	if topic == "" {
		topic = strings.TrimSpace(req.Intent.Topic)
	}
	createdBy := strings.TrimSpace(ident.Username)
	if createdBy == "" {
		createdBy = ident.Subject
	}
	result, err := services.HandleCLIFulfillmentPlan(
		ident.UserID,
		createdBy,
		strings.TrimSpace(req.CommandCatalogDigest),
		strings.TrimSpace(req.Command),
		topic,
		strings.TrimSpace(req.FailureKind),
		strings.TrimSpace(req.FailureMessage),
		req.Context,
		req.Intent,
	)
	if err != nil {
		if strings.Contains(err.Error(), "command_catalog_digest") {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": result, "msg": "success"})
}
