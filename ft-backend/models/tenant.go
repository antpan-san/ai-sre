package models

// Tenant represents a tenant in the multi-tenant system.
// All business entities reference a tenant via tenant_id.
type Tenant struct {
	BaseModel
	Name     string `gorm:"size:100;not null" json:"name"`
	Code     string `gorm:"size:50;not null;uniqueIndex" json:"code"`
	Status   string `gorm:"size:20;not null;default:'active'" json:"status"`
	Metadata JSONB  `gorm:"type:jsonb;not null;default:'{}'" json:"metadata"`
}

func (Tenant) TableName() string {
	return "tenants"
}
