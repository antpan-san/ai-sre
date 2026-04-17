package models

// K8sVersion represents an available Kubernetes version.
type K8sVersion struct {
	BaseModel
	Version  string `gorm:"size:20;not null" json:"version"`
	IsActive bool   `gorm:"not null;default:true" json:"is_active"`
}

func (K8sVersion) TableName() string {
	return "k8s_versions"
}
