package models

import (
	"time"

	"github.com/google/uuid"
)

// ServiceDeployment stores a server-side service installation spec and status.
type ServiceDeployment struct {
	SoftDeleteModel
	Service       string     `gorm:"size:40;not null;index" json:"service"`
	Profile       string     `gorm:"size:60;not null;default:'default'" json:"profile"`
	InstallMethod string     `gorm:"size:40;not null;default:'package'" json:"install_method"`
	Version       string     `gorm:"size:80" json:"version"`
	Params        JSONB      `gorm:"type:jsonb;not null;default:'{}'" json:"params"`
	TokenHash     string     `gorm:"size:128;not null;index" json:"-"`
	Status        string     `gorm:"size:30;not null;default:'pending';index" json:"status"`
	CurrentStep   string     `gorm:"size:80" json:"current_step"`
	LastError     string     `gorm:"type:text" json:"last_error"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
	CreatedBy     string     `gorm:"size:80" json:"created_by"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
}

func (ServiceDeployment) TableName() string {
	return "service_deployments"
}

// ServiceDeploymentEvent records ai-sre installation progress.
type ServiceDeploymentEvent struct {
	BaseModel
	DeploymentID uuid.UUID `gorm:"type:uuid;not null;index" json:"deployment_id"`
	Step         string    `gorm:"size:80;not null" json:"step"`
	Status       string    `gorm:"size:30;not null" json:"status"`
	Message      string    `gorm:"type:text" json:"message"`
}

func (ServiceDeploymentEvent) TableName() string {
	return "service_deployment_events"
}
