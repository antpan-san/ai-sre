package services

import (
	"strings"

	"ft-backend/database"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RecordOperationLog writes a security audit row for super_admin actions.
func RecordOperationLog(c *gin.Context, operation, resource, resourceID, status string, details map[string]interface{}) error {
	if details == nil {
		details = map[string]interface{}{}
	}
	username, _ := c.Get("username")
	name, _ := username.(string)
	if strings.TrimSpace(name) == "" {
		name = "system"
	}
	if uid, ok := c.Get("userID"); ok {
		if id, ok := uid.(uuid.UUID); ok && id != uuid.Nil {
			details["user_id"] = id.String()
		}
	}
	log := models.OperationLog{
		TenantID:   models.MustParseUUID(models.DefaultTenantID),
		Username:   name,
		Operation:  limitAuditText(operation, 100),
		Resource:   limitAuditText(resource, 100),
		ResourceID: limitAuditText(resourceID, 36),
		IP:         c.ClientIP(),
		UserAgent:  limitAuditText(c.GetHeader("User-Agent"), 255),
		Status:     limitAuditText(status, 20),
		Details:    models.NewJSONBFromMap(details),
	}
	return database.DB.Create(&log).Error
}

func limitAuditText(s string, n int) string {
	s = strings.TrimSpace(s)
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n]
}
