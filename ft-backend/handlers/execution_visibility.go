package handlers

import (
	"strings"

	"ft-backend/common/response"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// executionCategoryUsesAICapability reports whether the execution category
// corresponds to an ai-sre subcommand that invokes LLM / skill analysis (not
// pure housekeeping like version or k8s download).
func executionCategoryUsesAICapability(category string) bool {
	c := strings.ToLower(strings.TrimSpace(category))
	if c == "" {
		return false
	}
	bases := []string{"analyze", "ask", "runbook", "skills", "doctor", "elasticsearch"}
	for _, b := range bases {
		if c == b || strings.HasPrefix(c, b+"_") {
			return true
		}
	}
	return false
}

func executionStatusFailedLike(status string) bool {
	s := strings.ToLower(strings.TrimSpace(status))
	return s == models.ExecutionStatusFailed || s == models.ExecutionStatusCancelled
}

// applyExecutionConsoleMemberScope limits execution_records rows for JWT
// console members with role user: own rows (created_by / trigger_user) that
// either failed/cancelled or used an AI-facing category. Admins and
// super_admins see the full tenant list (caller still filters tenant_id).
func applyExecutionConsoleMemberScope(db *gorm.DB, role, username string) *gorm.DB {
	u := strings.TrimSpace(username)
	if models.IsAdminRole(role) || u == "" {
		return db
	}
	term := []string{models.ExecutionStatusFailed, models.ExecutionStatusCancelled}
	aiCats := []string{"analyze", "ask", "runbook", "skills", "doctor", "elasticsearch"}
	return db.Where("(created_by = ? OR trigger_user = ?)", u, u).
		Where(
			"(LOWER(status) IN ? OR LOWER(category) IN ? OR LOWER(category) LIKE ? OR LOWER(category) LIKE ?)",
			term,
			aiCats,
			"analyze%",
			"elasticsearch%",
		)
}

// assertExecutionRecordVisibleToConsoleMember returns false and sends 403 if
// the current JWT user may not read this record (used by detail/events APIs).
func assertExecutionRecordVisibleToConsoleMember(c *gin.Context, rec models.ExecutionRecord) bool {
	roleVal, _ := c.Get("role")
	role, _ := roleVal.(string)
	userVal, _ := c.Get("username")
	username, _ := userVal.(string)
	if models.IsAdminRole(role) {
		return true
	}
	u := strings.TrimSpace(username)
	if u == "" {
		response.Forbidden(c, "无权查看该执行记录")
		return false
	}
	if rec.CreatedBy != u && rec.TriggerUser != u {
		response.Forbidden(c, "无权查看该执行记录")
		return false
	}
	if executionStatusFailedLike(rec.Status) || executionCategoryUsesAICapability(rec.Category) {
		return true
	}
	response.Forbidden(c, "无权查看该执行记录")
	return false
}
