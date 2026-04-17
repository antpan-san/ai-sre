package models

import (
	"time"

	"github.com/google/uuid"
)

// TaskType defines the type of a task.
type TaskType string

const (
	TaskTypeShell          TaskType = "shell"           // Shell 命令执行
	TaskTypeFileDistrib    TaskType = "file_distribute" // 文件分发
	TaskTypeK8sDeploy      TaskType = "k8s_deploy"      // Kubernetes 部署
	TaskTypeSysInit        TaskType = "sys_init"        // 系统初始化
	TaskTypeTimeSync       TaskType = "time_sync"       // 时间同步
	TaskTypeInstallMonitor TaskType = "install_monitor" // 安装监控组件
	TaskTypeSecurityHarden TaskType = "security_harden" // 安全加固
	TaskTypeDiskOptimize   TaskType = "disk_optimize"   // 磁盘分区优化
	TaskTypeRegisterNodes  TaskType = "register_nodes"  // 注册受控节点(同步到Client)
)

// TaskStatus defines the lifecycle states of a task.
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"    // 待处理
	TaskStatusDispatched TaskStatus = "dispatched" // 已下发
	TaskStatusRunning    TaskStatus = "running"    // 执行中
	TaskStatusSuccess    TaskStatus = "success"    // 成功
	TaskStatusFailed     TaskStatus = "failed"     // 失败
	TaskStatusCancelled  TaskStatus = "cancelled"  // 已取消
	TaskStatusTimeout    TaskStatus = "timeout"    // 超时
)

// Task represents a top-level task (may be split into sub-tasks for multiple machines).
type Task struct {
	BaseModel
	Name         string     `gorm:"size:200;not null" json:"name"`
	Type         string     `gorm:"size:50;not null" json:"type"`
	Status       string     `gorm:"size:20;not null;default:'pending'" json:"status"`
	Priority     int        `gorm:"not null;default:0" json:"priority"`                 // 优先级, 0=normal, 1=high, 2=urgent
	CreatedBy    string     `gorm:"size:50;not null" json:"created_by"`                 // 创建人用户名
	Description  string     `gorm:"size:500" json:"description"`                        // 任务描述
	Payload      JSONB      `gorm:"type:jsonb;not null;default:'{}'" json:"payload"`    // 任务参数 (脚本内容/配置等)
	TargetIDs    JSONB      `gorm:"type:jsonb;not null;default:'[]'" json:"target_ids"` // 目标机器ID列表
	TotalCount   int        `gorm:"not null;default:0" json:"total_count"`              // 子任务总数
	SuccessCount int        `gorm:"not null;default:0" json:"success_count"`            // 成功数
	FailedCount  int        `gorm:"not null;default:0" json:"failed_count"`             // 失败数
	TimeoutSec   int        `gorm:"not null;default:300" json:"timeout_sec"`            // 超时秒数
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
}

func (Task) TableName() string {
	return "tasks"
}

// SubTask represents a task assigned to a specific machine/client.
type SubTask struct {
	BaseModel
	TaskID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"task_id"`    // 父任务ID
	MachineID  uuid.UUID  `gorm:"type:uuid;not null;index" json:"machine_id"` // 目标机器ID
	ClientID   string     `gorm:"size:100;not null;index" json:"client_id"`   // Client Agent ID
	Command    string     `gorm:"size:50;not null" json:"command"`            // 命令类型: run_shell, install_k8s 等
	Status     string     `gorm:"size:20;not null;default:'pending'" json:"status"`
	Payload    JSONB      `gorm:"type:jsonb;not null;default:'{}'" json:"payload"` // 执行参数
	Output     string     `gorm:"type:text" json:"output,omitempty"`               // 执行输出
	ExitCode   *int       `json:"exit_code,omitempty"`                             // 退出码
	Error      string     `gorm:"size:500" json:"error,omitempty"`                 // 错误信息
	RetryCount int        `gorm:"not null;default:0" json:"retry_count"`           // 重试次数
	MaxRetry   int        `gorm:"not null;default:3" json:"max_retry"`             // 最大重试次数
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

func (SubTask) TableName() string {
	return "sub_tasks"
}

// TaskLog records task execution events.
type TaskLog struct {
	ID        uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantID  uuid.UUID  `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000001'" json:"tenant_id"`
	TaskID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"task_id"`
	SubTaskID *uuid.UUID `gorm:"type:uuid;index" json:"sub_task_id,omitempty"`
	MachineID *uuid.UUID `gorm:"type:uuid;index" json:"machine_id,omitempty"`
	ClientID  string     `gorm:"size:100" json:"client_id"`
	Level     string     `gorm:"size:20;not null;default:'info'" json:"level"` // info, warn, error
	Message   string     `gorm:"type:text;not null" json:"message"`
	Details   JSONB      `gorm:"type:jsonb;not null;default:'{}'" json:"details"`
	CreatedAt time.Time  `json:"created_at"`
}

func (TaskLog) TableName() string {
	return "task_logs"
}

// ---- Command Protocol ----

// Command is the instruction sent from Server to Client via heartbeat response.
type Command struct {
	TaskID    string      `json:"task_id"`
	SubTaskID string      `json:"sub_task_id"`
	Command   string      `json:"command"` // run_shell, install_k8s, sys_init, etc.
	Payload   interface{} `json:"payload"`
	Timeout   int         `json:"timeout"` // seconds
}

// CommandResult is the response from Client after executing a command.
type CommandResult struct {
	TaskID    string `json:"task_id"`
	SubTaskID string `json:"sub_task_id"`
	ClientID  string `json:"client_id"`
	Status    string `json:"status"` // success, failed
	Output    string `json:"output"`
	ExitCode  int    `json:"exit_code"`
	Error     string `json:"error,omitempty"`
}
