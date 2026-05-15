package models

import (
	"time"

	"github.com/google/uuid"
)

// RuntimeWatchSession is a user-scoped continuous observe session for pod/process metrics.
type RuntimeWatchSession struct {
	BaseModel
	UserID          uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Namespace       string    `gorm:"size:253;not null" json:"namespace"`
	Pod             string    `gorm:"size:253;not null" json:"pod"`
	Container       string    `gorm:"size:253" json:"container,omitempty"`
	IntervalSec     int       `gorm:"not null;default:15" json:"interval_sec"`
	Status          string    `gorm:"size:20;not null;default:'active';index" json:"status"` // active | stopped
	SampleTokenHash string    `gorm:"size:64;not null" json:"-"`
	MachineNote     string    `gorm:"size:512" json:"machine_note,omitempty"`
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
