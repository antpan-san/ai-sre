package models

import (
	"github.com/google/uuid"
)

type RolePermission struct {
	RoleID       string    `gorm:"primaryKey;size:20" json:"role_id"`
	PermissionID uuid.UUID `gorm:"type:uuid;primaryKey" json:"permission_id"`
	TenantID     uuid.UUID `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000001'" json:"tenant_id"`
}
