// Package model defines the communication protocol between Client and Server.
// All types here must match the Server-side definitions in ft-backend/iotservice and ft-backend/models.
package model

import "encoding/json"

// =============================================================================
// Heartbeat Protocol
// =============================================================================

// HeartbeatRequest is the payload sent to the server on each heartbeat.
// Endpoint: POST /api/v1/heartbeats
type HeartbeatRequest struct {
	ClientID       string     `json:"client_id"`
	Fingerprint    string     `json:"fingerprint"`       // Machine unique fingerprint for idempotent upsert
	HeartbeatTime  int64      `json:"heartbeat_time"`    // Unix milliseconds
	ClientVersion  string     `json:"client_version"`
	ProcessID      int        `json:"process_id"`
	Status         string     `json:"status"`            // "normal", "busy", "degraded"
	LocalIP        string     `json:"local_ip"`
	OSInfo         string     `json:"os_info"`           // e.g. "linux amd64"
	BusinessModule string     `json:"business_module"`

	// ---- Master/Worker Topology ----
	Role        string `json:"role"`         // Node role: "master" or "worker"
	ClusterID   string `json:"cluster_id"`   // Cluster identifier this node belongs to
	ClusterName string `json:"cluster_name"` // Cluster display name

	TaskCount      int        `json:"task_count"`        // Total tasks executed
	TaskLeft       int        `json:"task_left"`         // Tasks still in progress
	LastTaskTime   int64      `json:"last_task_time"`    // Unix milliseconds of last task completion
	PrimaryHost    HostInfo   `json:"primary_host"`      // This machine's status
	SecondaryHosts []HostInfo `json:"secondary_hosts"`   // Managed nodes' status
}

// HeartbeatResponse is the server's reply to a heartbeat.
type HeartbeatResponse struct {
	Message               string       `json:"message"`                        // "pong"
	Commands              []Command    `json:"commands,omitempty"`            // Pending tasks to execute
	Upgrade               *UpgradeInfo `json:"upgrade,omitempty"`              // Non-nil if upgrade available
	ExcludeSecondaryIPs   []string     `json:"exclude_secondary_ips,omitempty"` // IPs of workers user deleted; client should stop reporting them
}

// =============================================================================
// Command Protocol
// =============================================================================

// Command is an instruction sent from Server to Client via heartbeat response.
type Command struct {
	TaskID    string          `json:"task_id"`
	SubTaskID string          `json:"sub_task_id"`
	Command   string          `json:"command"`   // Command type (see constants below)
	Payload   json.RawMessage `json:"payload"`   // Command-specific parameters
	Timeout   int             `json:"timeout"`   // Execution timeout in seconds
}

// CommandResult is the execution result reported back to the server.
// Endpoint: POST /api/v1/task/report
type CommandResult struct {
	TaskID    string `json:"task_id"`
	SubTaskID string `json:"sub_task_id"`
	ClientID  string `json:"client_id"`
	Status    string `json:"status"`           // "success" or "failed"
	Output    string `json:"output"`           // Execution stdout/stderr
	ExitCode  int    `json:"exit_code"`        // Process exit code
	Error     string `json:"error,omitempty"`  // Error message if failed
}

// =============================================================================
// Host Information
// =============================================================================

// HostInfo represents a machine's resource and status information.
type HostInfo struct {
	IP               string  `json:"ip"`
	Hostname         string  `json:"hostname"`
	OSInfo           string  `json:"os_info"`            // e.g. "linux amd64"
	OSVersion        string  `json:"os_version"`         // e.g. "Ubuntu 22.04.3 LTS"
	KernelVersion    string  `json:"kernel_version"`     // e.g. "5.15.0-91-generic"
	CPUCores         int     `json:"cpu_cores"`          // Logical CPU count
	CPUUsage         float64 `json:"cpu_usage"`          // CPU usage 0-100%
	MemoryTotal      int64   `json:"memory_total"`       // Total memory in bytes
	MemoryUsed       int64   `json:"memory_used"`        // Used memory in bytes
	MemoryUsage      float64 `json:"memory_usage"`       // Memory usage 0-100%
	DiskTotal        int64   `json:"disk_total"`         // Total disk in bytes
	DiskUsed         int64   `json:"disk_used"`          // Used disk in bytes
	DiskUsage        float64 `json:"disk_usage"`         // Disk usage 0-100%
	NetworkDelay     int     `json:"network_delay"`      // Network delay in ms
	NetworkInterface string  `json:"network_interface"`  // Primary NIC name
	Status           string  `json:"status"`             // "up", "down", "degraded" (TCP ok, SSH failed), "unknown"
	ProbeError       string  `json:"probe_error,omitempty"` // Last probe failure reason (for worker diagnostics)
}

// =============================================================================
// Upgrade Protocol
// =============================================================================

// UpgradeInfo describes an available agent upgrade.
type UpgradeInfo struct {
	Version     string `json:"version"`      // New version string
	DownloadURL string `json:"download_url"` // Binary download URL
	Checksum    string `json:"checksum"`     // SHA256 checksum
	Force       bool   `json:"force"`        // If true, must upgrade immediately
}

// =============================================================================
// Command Type Constants
// =============================================================================

const (
	CmdRunShell       = "run_shell"        // Execute shell script
	CmdSysInit        = "sys_init"         // System initialization
	CmdTimeSync       = "time_sync"        // Time synchronization
	CmdSecurityHarden = "security_harden"  // Security hardening
	CmdDiskOptimize   = "disk_optimize"    // Disk optimization
	CmdInstallK8s     = "install_k8s"      // Kubernetes installation
	CmdInstallMonitor = "install_monitor"  // Monitoring agent installation
	CmdSyncNodes      = "sync_nodes"       // Sync managed node list
	CmdRunPlaybook    = "run_playbook"     // Execute Ansible playbook
)

// LogEntry is a single log line that a client sends to the server while a
// long-running task (e.g. install_k8s) is executing.
// Endpoint: POST /api/v1/task/log
type LogEntry struct {
	TaskID    string `json:"task_id"`
	SubTaskID string `json:"sub_task_id"`
	ClientID  string `json:"client_id"`
	Level     string `json:"level"`   // "info" | "warn" | "error"
	Message   string `json:"message"`
}

// =============================================================================
// Result Status Constants
// =============================================================================

const (
	StatusSuccess = "success"
	StatusFailed  = "failed"
)

// =============================================================================
// Client Status Constants
// =============================================================================

const (
	ClientStatusNormal   = "normal"
	ClientStatusBusy     = "busy"
	ClientStatusDegraded = "degraded"
)

// =============================================================================
// Node Role Constants (Master/Worker Topology)
// =============================================================================

const (
	RoleMaster = "master" // Cluster master node (only 1 per cluster)
	RoleWorker = "worker" // Cluster worker node (multiple per cluster)
)
