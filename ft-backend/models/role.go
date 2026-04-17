package models

// Role represents a system role.
type Role struct {
	BaseModel
	Name        string `gorm:"size:50;not null" json:"name"`
	Code        string `gorm:"size:50;not null;uniqueIndex" json:"code"`
	Description string `gorm:"size:255" json:"description"`
	IsSystem    bool   `gorm:"not null;default:false" json:"is_system"` // 系统内置角色不可删除
}

func (Role) TableName() string {
	return "roles"
}
