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

// ProxyConfig represents a proxy/reverse-proxy configuration.
type ProxyConfig struct {
	SoftDeleteModel
	Name        string `gorm:"size:100;not null" json:"name"`
	Type        string `gorm:"size:20;not null;default:'nginx'" json:"type"` // nginx, haproxy, envoy
	ListenPort  int    `gorm:"not null" json:"listen_port"`
	TargetHost  string `gorm:"size:255;not null" json:"target_host"`
	TargetPort  int    `gorm:"not null" json:"target_port"`
	Status      string `gorm:"size:20;not null;default:'inactive'" json:"status"` // active, inactive
	SSLEnabled  bool   `gorm:"not null;default:false" json:"ssl_enabled"`
	Config      JSONB  `gorm:"type:jsonb;not null;default:'{}'" json:"config"`
	MachineID   string `gorm:"size:36" json:"machine_id,omitempty"`
	Description string `gorm:"size:500" json:"description"`
}

func (ProxyConfig) TableName() string {
	return "proxy_configs"
}
