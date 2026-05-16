package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	AutoIterationStatusDraft             = "draft"
	AutoIterationStatusPending           = "pending"
	AutoIterationStatusRunning           = "running"
	AutoIterationStatusPaused            = "paused"
	AutoIterationStatusAwaitingApproval  = "awaiting_approval"
	AutoIterationStatusApproved          = "approved"
	AutoIterationStatusRejected          = "rejected"
	AutoIterationStatusCancelled         = "cancelled"
	AutoIterationStatusCompleted         = "completed"
	AutoIterationStatusFailed            = "failed"

	AutoIterationRiskLow    = "low"
	AutoIterationRiskMedium = "medium"
	AutoIterationRiskHigh   = "high"

	AutoIterationSourceManual       = "manual"
	AutoIterationSourceCLIFeedback  = "cli_feedback"

	AutoIterationEventLog         = "log"
	AutoIterationEventStateChange = "state_change"
	AutoIterationEventWorker      = "worker"
	AutoIterationEventTest        = "test"
	AutoIterationEventNotification = "notification"

	CodeAgentStatusActive   = "active"
	CodeAgentStatusDisabled = "disabled"
)

type AutoIteration struct {
	BaseModel
	Title                      string     `gorm:"size:200;not null" json:"title"`
	Description                string     `gorm:"size:2000" json:"description,omitempty"`
	Status                     string     `gorm:"size:32;not null;default:'draft';index" json:"status"`
	Source                     string     `gorm:"size:32;not null;default:'manual';index" json:"source"`
	RiskLevel                  string     `gorm:"size:32;not null;default:'low';index" json:"risk_level"`
	RequiresSuperAdminApproval bool       `gorm:"not null;default:false" json:"requires_super_admin_approval"`
	Topic                      string     `gorm:"size:80;index" json:"topic,omitempty"`
	Command                    string     `gorm:"size:2000" json:"command,omitempty"`
	Summary                    string     `gorm:"size:2000" json:"summary,omitempty"`
	FeedbackID                 *uuid.UUID `gorm:"type:uuid;index" json:"feedback_id,omitempty"`
	CreatedByUserID            *uuid.UUID `gorm:"type:uuid;index" json:"created_by_user_id,omitempty"`
	CreatedBy                  string     `gorm:"size:80" json:"created_by"`
	ApprovedByUserID           *uuid.UUID `gorm:"type:uuid;index" json:"approved_by_user_id,omitempty"`
	ApprovedBy                 string     `gorm:"size:80" json:"approved_by,omitempty"`
	ApprovedAt                 *time.Time `json:"approved_at,omitempty"`
	AssignedAgentID            *uuid.UUID `gorm:"type:uuid;index" json:"assigned_agent_id,omitempty"`
	LastError                  string     `gorm:"size:2000" json:"last_error,omitempty"`
	Metadata                   JSONB      `gorm:"type:jsonb;not null;default:'{}'" json:"metadata"`
}

func (AutoIteration) TableName() string { return "auto_iterations" }

type AutoIterationEvent struct {
	BaseModel
	AutoIterationID uuid.UUID `gorm:"type:uuid;not null;index" json:"auto_iteration_id"`
	EventType       string    `gorm:"size:32;not null;index" json:"event_type"`
	ActorType       string    `gorm:"size:32;not null;default:'system'" json:"actor_type"`
	ActorName       string    `gorm:"size:80" json:"actor_name,omitempty"`
	Message         string    `gorm:"size:4000;not null" json:"message"`
	Payload         JSONB     `gorm:"type:jsonb;not null;default:'{}'" json:"payload"`
}

func (AutoIterationEvent) TableName() string { return "auto_iteration_events" }

type AutoIterationSettings struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	Enabled   bool      `gorm:"not null;default:false" json:"enabled"`
	MaxConcurrent int   `gorm:"not null;default:2" json:"max_concurrent"`
	HighRiskRequiresApproval bool `gorm:"not null;default:true" json:"high_risk_requires_approval"`
	Notes     string    `gorm:"size:2000" json:"notes,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy string    `gorm:"size:80" json:"updated_by,omitempty"`
}

func (AutoIterationSettings) TableName() string { return "auto_iteration_settings" }

type AutoIterationFeedback struct {
	BaseModel
	UserID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	CLIBindingID *uuid.UUID `gorm:"type:uuid;index" json:"cli_binding_id,omitempty"`
	Topic        string     `gorm:"size:80;index" json:"topic"`
	Classification string   `gorm:"size:64;index" json:"classification"`
	NeedIteration bool      `gorm:"not null;default:false" json:"need_iteration"`
	UserMessage  string     `gorm:"size:2000" json:"user_message"`
	RawPayload   JSONB      `gorm:"type:jsonb;not null;default:'{}'" json:"-"`
	AutoIterationID *uuid.UUID `gorm:"type:uuid;index" json:"auto_iteration_id,omitempty"`
}

func (AutoIterationFeedback) TableName() string { return "auto_iteration_feedbacks" }

type CodeAgentBinding struct {
	SoftDeleteModel
	Name            string     `gorm:"size:120;not null" json:"name"`
	TokenHash       string     `gorm:"size:64;not null;uniqueIndex" json:"-"`
	FingerprintHash string     `gorm:"size:64;not null;index" json:"-"`
	Status          string     `gorm:"size:32;not null;default:'active';index" json:"status"`
	LastHeartbeatAt *time.Time `json:"last_heartbeat_at,omitempty"`
}

func (CodeAgentBinding) TableName() string { return "code_agent_bindings" }
