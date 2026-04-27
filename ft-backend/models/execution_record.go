package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	ExecutionStatusPending   = "pending"
	ExecutionStatusRunning   = "running"
	ExecutionStatusSuccess   = "success"
	ExecutionStatusFailed    = "failed"
	ExecutionStatusCancelled = "cancelled"

	RollbackCapabilityAuto   = "auto"
	RollbackCapabilityManual = "manual"
	RollbackCapabilityNone   = "none"

	RollbackStatusNotStarted = "not_started"
	RollbackStatusPending    = "pending"
	RollbackStatusSuccess    = "success"
	RollbackStatusFailed     = "failed"
	RollbackStatusBlocked    = "blocked"
)

// ExecutionRecord is the durable, user-facing history for CLI commands,
// copied scripts, and server-side tasks.
type ExecutionRecord struct {
	BaseModel
	CorrelationID      string     `gorm:"size:100;index" json:"correlation_id"`
	Source             string     `gorm:"size:40;not null;index" json:"source"` // cli, script, job, k8s, init-tools, rollback
	Category           string     `gorm:"size:60;index" json:"category"`
	Name               string     `gorm:"size:200;not null" json:"name"`
	Command            string     `gorm:"type:text" json:"command"`
	CommandDigest      string     `gorm:"size:64;index" json:"command_digest"`
	Status             string     `gorm:"size:20;not null;default:'pending';index" json:"status"`
	ExitCode           *int       `json:"exit_code,omitempty"`
	CreatedBy          string     `gorm:"size:80;index" json:"created_by"`
	TriggerUser        string     `gorm:"size:80" json:"trigger_user"`
	TargetHost         string     `gorm:"size:200;index" json:"target_host"`
	TargetIPs          JSONB      `gorm:"type:jsonb;not null;default:'[]'" json:"target_ips"`
	ResourceType       string     `gorm:"size:80;index" json:"resource_type"`
	ResourceID         string     `gorm:"size:120;index" json:"resource_id"`
	ResourceName       string     `gorm:"size:200" json:"resource_name"`
	TaskID             *uuid.UUID `gorm:"type:uuid;index" json:"task_id,omitempty"`
	ParentExecutionID  *uuid.UUID `gorm:"type:uuid;index" json:"parent_execution_id,omitempty"`
	StartedAt          *time.Time `json:"started_at,omitempty"`
	FinishedAt         *time.Time `json:"finished_at,omitempty"`
	DurationMs         int64      `gorm:"not null;default:0" json:"duration_ms"`
	StdoutSummary      string     `gorm:"type:text" json:"stdout_summary"`
	StderrSummary      string     `gorm:"type:text" json:"stderr_summary"`
	Effects            JSONB      `gorm:"type:jsonb;not null;default:'{}'" json:"effects"`
	Metadata           JSONB      `gorm:"type:jsonb;not null;default:'{}'" json:"metadata"`
	RollbackCapability string     `gorm:"size:20;not null;default:'none';index" json:"rollback_capability"`
	RollbackStatus     string     `gorm:"size:20;not null;default:'not_started';index" json:"rollback_status"`
	RollbackPlan       JSONB      `gorm:"type:jsonb;not null;default:'{}'" json:"rollback_plan"`
	RollbackAdvice     string     `gorm:"type:text" json:"rollback_advice"`
	ReportTokenHash    string     `gorm:"size:64;index" json:"-"`
}

func (ExecutionRecord) TableName() string {
	return "execution_records"
}

// ExecutionEvent stores append-only progress, output snippets, and effect
// updates for an execution record.
type ExecutionEvent struct {
	BaseModel
	ExecutionID uuid.UUID `gorm:"type:uuid;not null;index" json:"execution_id"`
	Level       string    `gorm:"size:20;not null;default:'info'" json:"level"`
	Phase       string    `gorm:"size:60;index" json:"phase"`
	Message     string    `gorm:"type:text;not null" json:"message"`
	Output      string    `gorm:"type:text" json:"output"`
	Details     JSONB     `gorm:"type:jsonb;not null;default:'{}'" json:"details"`
}

func (ExecutionEvent) TableName() string {
	return "execution_events"
}

// ExecutionDependency records explicit or inferred dependency edges that make
// rollback risky.
type ExecutionDependency struct {
	BaseModel
	ExecutionID          uuid.UUID `gorm:"type:uuid;not null;index" json:"execution_id"`
	DependsOnExecutionID uuid.UUID `gorm:"type:uuid;not null;index" json:"depends_on_execution_id"`
	Relation             string    `gorm:"size:60;not null" json:"relation"`
	ImpactLevel          string    `gorm:"size:20;not null;default:'warning'" json:"impact_level"`
	Message              string    `gorm:"type:text" json:"message"`
	Details              JSONB     `gorm:"type:jsonb;not null;default:'{}'" json:"details"`
}

func (ExecutionDependency) TableName() string {
	return "execution_dependencies"
}
