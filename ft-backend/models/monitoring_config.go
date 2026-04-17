package models

// MonitoringConfig stores monitoring tool configurations (Prometheus, exporters, etc.)
type MonitoringConfig struct {
	SoftDeleteModel
	Name        string `gorm:"size:100;not null" json:"name"`
	Type        string `gorm:"size:50;not null" json:"type"`        // prometheus, node_exporter, jmx_exporter, etc.
	Status      string `gorm:"size:20;not null;default:'inactive'" json:"status"` // active, inactive
	Config      JSONB  `gorm:"type:jsonb;not null;default:'{}'" json:"config"`
	Description string `gorm:"size:500" json:"description"`
	MachineID   string `gorm:"size:36" json:"machine_id,omitempty"` // 关联的机器ID（可选）
}

func (MonitoringConfig) TableName() string {
	return "monitoring_configs"
}

// AlertRule stores alerting rules for monitoring.
type AlertRule struct {
	SoftDeleteModel
	Name        string `gorm:"size:100;not null" json:"name"`
	Type        string `gorm:"size:50;not null" json:"type"` // cpu, memory, disk, custom
	Condition   string `gorm:"size:200;not null" json:"condition"`
	Threshold   string `gorm:"size:100;not null" json:"threshold"`
	Severity    string `gorm:"size:20;not null;default:'warning'" json:"severity"` // info, warning, critical
	Status      string `gorm:"size:20;not null;default:'active'" json:"status"`    // active, inactive
	Config      JSONB  `gorm:"type:jsonb;not null;default:'{}'" json:"config"`
	Description string `gorm:"size:500" json:"description"`
}

func (AlertRule) TableName() string {
	return "alert_rules"
}
