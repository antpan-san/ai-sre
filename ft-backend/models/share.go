package models

import (
	"time"

	"github.com/google/uuid"
)

type Share struct {
	BaseModel
	FileID      uuid.UUID `gorm:"type:uuid;not null" json:"file_id"`
	ShareKey    string    `gorm:"size:64;not null;uniqueIndex" json:"share_key"`
	ExpiresAt   time.Time `json:"expires_at"`
	AccessCount int       `gorm:"not null;default:0" json:"access_count"`

	// Associations
	File File `gorm:"foreignKey:FileID" json:"file,omitempty"`
}
