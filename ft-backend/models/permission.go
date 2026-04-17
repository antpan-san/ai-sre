package models

type Permission struct {
	SoftDeleteModel
	Name        string `gorm:"size:100;not null" json:"name"`
	Code        string `gorm:"size:100;not null" json:"code"`
	Description string `gorm:"size:255" json:"description,omitempty"`
}
