package models

import (
	"time"

	"github.com/google/uuid"
)

// NodeRole enumerates valid machine roles in a cluster topology.
const (
	NodeRoleMaster     = "master"
	NodeRoleWorker     = "worker"
	NodeRoleStandalone = "standalone"
)

// Machine represents a managed server in the fleet.
//
// Uniqueness constraints (enforced at DB level via partial unique indexes):
//   - (tenant_id, host_fingerprint) WHERE deleted_at IS NULL
//   - (tenant_id, client_id)        WHERE deleted_at IS NULL
//   - (tenant_id, cluster_id)       WHERE node_role = 'master' AND deleted_at IS NULL
//
// Topology: master_machine_id is a self-referential FK. Workers point to their master.
type Machine struct {
	SoftDeleteModel

	// --- basic info (existing) ---
	Name     string `gorm:"size:100;not null" json:"name"`
	IP       string `gorm:"size:50;not null" json:"ip"`
	CPU      int    `gorm:"not null;default:0" json:"cpu"`
	Memory   int    `gorm:"not null;default:0" json:"memory"`
	Disk     int    `gorm:"not null;default:0" json:"disk"`
	Status   string `gorm:"size:20;not null;default:'offline'" json:"status"`
	Labels   JSONB  `gorm:"type:jsonb;not null;default:'{}'" json:"labels"`
	Metadata JSONB  `gorm:"type:jsonb;not null;default:'{}'" json:"metadata"`

	// --- uniqueness & identity ---
	ClientID        string `gorm:"size:100;index:idx_machines_client_id" json:"client_id"`
	HostFingerprint string `gorm:"size:255" json:"host_fingerprint"`

	// --- cluster topology ---
	NodeRole        string     `gorm:"size:20;not null;default:'standalone'" json:"node_role"`
	ClusterID       *uuid.UUID `gorm:"type:uuid;index:idx_machines_cluster_id" json:"cluster_id,omitempty"`
	MasterMachineID *uuid.UUID `gorm:"type:uuid;index:idx_machines_master_machine_id" json:"master_machine_id,omitempty"`

	// --- ownership ---
	OwnerUserID *uuid.UUID `gorm:"type:uuid;index:idx_machines_owner_user_id" json:"owner_user_id,omitempty"`

	// --- heartbeat tracking ---
	LastHeartbeatAt *time.Time `gorm:"index:idx_machines_last_heartbeat_at" json:"last_heartbeat_at,omitempty"`

	// --- real-time metrics (updated on each heartbeat) ---
	OSVersion     string  `gorm:"size:100" json:"os_version,omitempty"`
	KernelVersion string  `gorm:"size:100" json:"kernel_version,omitempty"`
	CPUCores      int     `gorm:"not null;default:0" json:"cpu_cores"`
	CPUUsage      float64 `gorm:"not null;default:0" json:"cpu_usage"`
	MemoryTotal   int64   `gorm:"not null;default:0" json:"memory_total"`
	MemoryUsed    int64   `gorm:"not null;default:0" json:"memory_used"`
	MemoryUsage   float64 `gorm:"not null;default:0" json:"memory_usage"`
	DiskTotal     int64   `gorm:"not null;default:0" json:"disk_total"`
	DiskUsed      int64   `gorm:"not null;default:0" json:"disk_used"`
	DiskUsage     float64 `gorm:"not null;default:0" json:"disk_usage"`
}

// IsOnline returns true when the machine has heartbeated within the given window.
func (m *Machine) IsOnline(window time.Duration) bool {
	if m.LastHeartbeatAt == nil {
		return false
	}
	return time.Since(*m.LastHeartbeatAt) <= window
}

// IsMaster returns true if this machine is the master in its cluster.
func (m *Machine) IsMaster() bool {
	return m.NodeRole == NodeRoleMaster
}

// IsWorker returns true if this machine is a worker node.
func (m *Machine) IsWorker() bool {
	return m.NodeRole == NodeRoleWorker
}
