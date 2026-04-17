package models

import (
	"github.com/google/uuid"
)

type User struct {
	SoftDeleteModel
	Username string `gorm:"size:50;not null" json:"username"`
	Email    string `gorm:"size:100;not null" json:"email"`
	Phone    string `gorm:"size:20" json:"phone"`
	Password string `gorm:"size:100;not null" json:"-"`
	FullName string `gorm:"size:100" json:"full_name"`
	Avatar   string `gorm:"size:255" json:"avatar"`
	Role     string `gorm:"size:20;not null;default:'user'" json:"role"`

	// Associations
	Files     []File     `gorm:"foreignKey:UserID" json:"files,omitempty"`
	Transfers []Transfer `gorm:"foreignKey:UserID" json:"transfers,omitempty"`
}

// UserIDFromContext is a helper to extract uuid.UUID from a context value.
func UserIDFromContext(val interface{}) uuid.UUID {
	switch v := val.(type) {
	case uuid.UUID:
		return v
	case string:
		uid, _ := uuid.Parse(v)
		return uid
	default:
		return uuid.Nil
	}
}
