package models

import (
	"time"

	"github.com/google/uuid"
)

// RuntimeWatchSession stores one diagnose/observe run for a user (CLI auto or manual stream).
type RuntimeWatchSession struct {
	BaseModel
	UserID          uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	Namespace       string     `gorm:"size:253;not null" json:"namespace"`
	Pod             string     `gorm:"size:253;not null" json:"pod"`
	Container       string     `gorm:"size:253" json:"container,omitempty"`
	IntervalSec     int        `gorm:"not null;default:15" json:"interval_sec"`
	Status          string     `gorm:"size:20;not null;default:'active';index" json:"status"` // active | stopped
	SampleTokenHash string     `gorm:"size:64;not null" json:"-"`
	MachineNote     string     `gorm:"size:512" json:"machine_note,omitempty"`
	TargetDisplay   string     `gorm:"size:512" json:"target_display,omitempty"`
	ResourceKind    string     `gorm:"size:64" json:"resource_kind,omitempty"`
	ResourceName    string     `gorm:"size:253" json:"resource_name,omitempty"`
	WorkPod         string     `gorm:"size:253" json:"work_pod,omitempty"`
	DiagnosisLevel  string     `gorm:"size:32" json:"diagnosis_level,omitempty"`
	RootCause       string     `gorm:"type:text" json:"root_cause,omitempty"`
	Evidence        string     `gorm:"type:text" json:"evidence,omitempty"`
	DiagnosisSource string     `gorm:"size:16" json:"diagnosis_source,omitempty"`
	SampleCount     int        `gorm:"not null;default:0" json:"sample_count"`
	LastDiagnosedAt *time.Time `json:"last_diagnosed_at,omitempty"`
}

func (RuntimeWatchSession) TableName() string {
	return "runtime_watch_sessions"
}

// RuntimeWatchSample stores one uploaded watch payload JSON.
type RuntimeWatchSample struct {
	BaseModel
	SessionID  uuid.UUID `gorm:"type:uuid;not null;index" json:"session_id"`
	ObservedAt time.Time `gorm:"not null;index" json:"observed_at"`
	Payload    JSONB     `gorm:"type:jsonb;not null" json:"payload"`
}

func (RuntimeWatchSample) TableName() string {
	return "runtime_watch_samples"
}
