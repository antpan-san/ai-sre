package models

import (
	"time"

	"github.com/google/uuid"
)

// Heartbeat stores client agent heartbeat data.
// This table is PARTITIONED by created_at (monthly) in PostgreSQL.
// GORM AutoMigrate cannot create this table; use migration_pg.sql instead.
type Heartbeat struct {
	ID             uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantID       uuid.UUID `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000001'" json:"tenant_id"`
	ClientID       string    `gorm:"size:100;not null" json:"client_id"`
	ClientVersion  string    `gorm:"size:50" json:"client_version"`
	ProcessID      int       `json:"process_id"`
	Status         string    `gorm:"size:20;not null;default:'unknown'" json:"status"`
	LocalIP        string    `gorm:"size:50" json:"local_ip"`
	BusinessModule string    `gorm:"size:100" json:"business_module"`
	TaskCount      int       `gorm:"not null;default:0" json:"task_count"`
	TaskLeft       int       `gorm:"not null;default:0" json:"task_left"`
	LastTaskTime   *time.Time `json:"last_task_time"`
	PrimaryHost    JSONB     `gorm:"type:jsonb;not null;default:'{}'" json:"primary_host"`
	SecondaryHosts JSONB     `gorm:"type:jsonb;not null;default:'[]'" json:"secondary_hosts"`
	CreatedAt      time.Time `gorm:"primaryKey" json:"created_at"` // part of composite PK for partitioning
}

func (Heartbeat) TableName() string {
	return "heartbeats"
}
