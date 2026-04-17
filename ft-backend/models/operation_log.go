package models

import (
	"time"

	"github.com/google/uuid"
)

type OperationLog struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantID     uuid.UUID `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000001'" json:"tenant_id"`
	Username     string    `gorm:"size:50;not null" json:"username"`
	Operation    string    `gorm:"size:100;not null" json:"operation"`
	Resource     string    `gorm:"size:100;not null" json:"resource"`
	ResourceID   string    `gorm:"size:36" json:"resource_id"`
	IP           string    `gorm:"size:50;not null" json:"ip"`
	UserAgent    string    `gorm:"size:255" json:"user_agent"`
	Status       string    `gorm:"size:20;not null" json:"status"`
	ErrorMessage string    `gorm:"size:500" json:"error_message,omitempty"`
	Details      JSONB     `gorm:"type:jsonb;not null;default:'{}'" json:"details"`
	CreatedAt    time.Time `json:"created_at"`
}
