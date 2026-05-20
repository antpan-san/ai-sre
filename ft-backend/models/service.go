package models

// Service represents a deployed service managed by the platform.
type Service struct {
	SoftDeleteModel
	Name        string `gorm:"size:100;not null" json:"name"`
	Image       string `gorm:"size:255" json:"image"`
	Replicas    int    `gorm:"not null;default:1" json:"replicas"`
	Port        int    `json:"port"`
	Status      string `gorm:"size:20;not null;default:'stopped'" json:"status"` // running, stopped, error, deploying
	Type        string `gorm:"size:20;not null;default:'docker'" json:"type"`    // docker, k8s, linux
	MachineID   string `gorm:"size:36" json:"machine_id,omitempty"`
	Config      JSONB  `gorm:"type:jsonb;not null;default:'{}'" json:"config"`
	Description string `gorm:"size:500" json:"description"`
}

func (Service) TableName() string {
	return "services"
}
