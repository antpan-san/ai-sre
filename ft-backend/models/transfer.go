package models

import (
	"github.com/google/uuid"
)

type Transfer struct {
	BaseModel
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	FileID    uuid.UUID `gorm:"type:uuid;not null" json:"file_id"`
	Type      string    `gorm:"size:20;not null" json:"type"`
	Status    string    `gorm:"size:20;not null;default:'pending'" json:"status"`
	Progress  int       `gorm:"not null;default:0" json:"progress"`
	Speed     int64     `gorm:"not null;default:0" json:"speed"`
	IpAddress string    `gorm:"size:50" json:"ip_address"`
	UserAgent string    `gorm:"size:255" json:"user_agent"`

	// Associations
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	File File `gorm:"foreignKey:FileID" json:"file,omitempty"`
}
