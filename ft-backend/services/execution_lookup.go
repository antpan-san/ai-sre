package services

import (
	"strings"

	"ft-backend/database"
	"ft-backend/models"

	"github.com/google/uuid"
)

// FindClientExecutionIDByRequestID locates a client execution or ai_call row by AI request_id.
func FindClientExecutionIDByRequestID(requestID string) (string, string, error) {
	rid := strings.TrimSpace(requestID)
	if database.DB == nil || rid == "" {
		return "", "", nil
	}
	var child models.ExecutionRecord
	err := database.DB.Where("COALESCE(metadata->>'request_id','') = ?", rid).
		Order("created_at DESC").First(&child).Error
	if err != nil {
		return "", "", nil
	}
	if pid := child.ParentExecutionID; pid != nil && *pid != uuid.Nil {
		return pid.String(), child.ID.String(), nil
	}
	if strings.TrimSpace(child.CorrelationID) != "" {
		var parent models.ExecutionRecord
		if err := database.DB.Where("correlation_id = ?", child.CorrelationID).
			Where("parent_execution_id IS NULL").
			Where("COALESCE(metadata->>'record_kind','') IN ('client_execution','') OR source = ?", "cli").
			Order("created_at ASC").First(&parent).Error; err == nil {
			return parent.ID.String(), child.ID.String(), nil
		}
	}
	return child.ID.String(), child.ID.String(), nil
}
