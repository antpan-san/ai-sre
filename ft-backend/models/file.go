package models

import (
	"github.com/google/uuid"
)

type File struct {
	SoftDeleteModel
	UserID        uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	Filename      string     `gorm:"size:255;not null" json:"filename"`
	OriginalName  string     `gorm:"size:255;not null" json:"original_name"`
	Size          int64      `json:"size"`
	Path          string     `gorm:"size:500;not null" json:"path"`
	MimeType      string     `gorm:"size:100" json:"mime_type"`
	Extension     string     `gorm:"size:20" json:"extension"`
	Hash          string     `gorm:"size:64" json:"hash"`
	Status        string     `gorm:"size:20;not null;default:'available'" json:"status"`
	Visibility    string     `gorm:"size:20;not null;default:'private'" json:"visibility"`
	DownloadCount int        `gorm:"not null;default:0" json:"download_count"`

	// Associations
	User      User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Transfers []Transfer `gorm:"foreignKey:FileID" json:"transfers,omitempty"`
	Shares    []Share    `gorm:"foreignKey:FileID" json:"shares,omitempty"`
}
