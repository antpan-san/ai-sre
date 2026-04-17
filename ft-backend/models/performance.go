package models

import (
	"time"

	"github.com/google/uuid"
)

type PerformanceData struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantID    uuid.UUID `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000001'" json:"tenant_id"`
	MachineID   uuid.UUID `gorm:"type:uuid;not null" json:"machine_id"`
	MachineName string    `gorm:"size:100;not null" json:"machine_name"`
	CPUUsage    float64   `gorm:"not null;default:0" json:"cpu_usage"`
	MemoryUsage float64   `gorm:"not null;default:0" json:"memory_usage"`
	DiskUsage   float64   `gorm:"not null;default:0" json:"disk_usage"`
	NetworkIn   float64   `gorm:"not null;default:0" json:"network_in"`
	NetworkOut  float64   `gorm:"not null;default:0" json:"network_out"`
	Metrics     JSONB     `gorm:"type:jsonb;not null;default:'{}'" json:"metrics"`
	Timestamp   time.Time `json:"timestamp"`
	CreatedAt   time.Time `json:"created_at"`
}
