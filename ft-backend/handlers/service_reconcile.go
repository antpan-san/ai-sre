package handlers

import (
	"time"

	"ft-backend/models"

	"gorm.io/gorm"
)

// staleDeployingServiceMaxAge is how long a row may stay in status "deploying"
// without any row update before we treat it as abandoned and mark stopped.
// The legacy POST /api/service/deploy path never tied task lifecycle to this table,
// so rows could remain "deploying" indefinitely and pollute 概览 / 台账.
const staleDeployingServiceMaxAge = 24 * time.Hour

// reconcileStaleDeployingServices marks old stuck "deploying" services as stopped
// so dashboard and service list reflect reality. Idempotent; safe to call on every read.
func reconcileStaleDeployingServices(db *gorm.DB) error {
	cutoff := time.Now().Add(-staleDeployingServiceMaxAge)
	now := time.Now()
	return db.Model(&models.Service{}).
		Where("status = ? AND updated_at < ?", "deploying", cutoff).
		Updates(map[string]interface{}{
			"status":     "stopped",
			"updated_at": now,
		}).Error
}
