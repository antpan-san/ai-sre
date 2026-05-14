package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	CLIInstallSessionStatusPending = "pending"
	CLIInstallSessionStatusUsed    = "used"
	CLIInstallSessionStatusExpired = "expired"
)

type CLIInstallSession struct {
	BaseModel
	UserID              uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	Username            string     `gorm:"size:80;not null" json:"username"`
	TokenHash           string     `gorm:"size:64;not null;uniqueIndex" json:"-"`
	Status              string     `gorm:"size:20;not null;default:'pending';index" json:"status"`
	ExpiresAt           time.Time  `gorm:"not null;index" json:"expires_at"`
	UsedAt              *time.Time `json:"used_at,omitempty"`
	UsedFingerprintHash string     `gorm:"size:64" json:"used_fingerprint_hash,omitempty"`
	CLIBindingID        *uuid.UUID `gorm:"type:uuid;index" json:"cli_binding_id,omitempty"`
}

func (CLIInstallSession) TableName() string {
	return "cli_install_sessions"
}

type CLIBinding struct {
	BaseModel
	UserID          uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	Username        string     `gorm:"size:80;not null;index" json:"username"`
	TokenHash       string     `gorm:"size:64;not null;uniqueIndex" json:"-"`
	FingerprintHash string     `gorm:"size:64;not null;index" json:"fingerprint_hash"`
	Hostname        string     `gorm:"size:200" json:"hostname"`
	OS              string     `gorm:"size:120" json:"os"`
	Arch            string     `gorm:"size:40" json:"arch"`
	InstallUser     string     `gorm:"size:80" json:"install_user"`
	Version         string     `gorm:"size:40" json:"version"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt       time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt       *time.Time `gorm:"index" json:"revoked_at,omitempty"`
}

func (CLIBinding) TableName() string {
	return "cli_bindings"
}
