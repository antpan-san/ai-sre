package models

import "time"

// DiagnoseSampleRecord persists anonymized diagnose samples in PostgreSQL.
// Full sample body is stored in payload JSONB; indexed columns support admin queries.
type DiagnoseSampleRecord struct {
	BaseModel
	SampleTime             time.Time `gorm:"not null;index:idx_diagnose_samples_topic_time,priority:2" json:"sample_time"`
	Topic                  string    `gorm:"size:80;not null;index:idx_diagnose_samples_topic_time,priority:1;index" json:"topic"`
	SampleSource           string    `gorm:"size:32;index" json:"sample_source,omitempty"`
	CommandKind            string    `gorm:"size:32" json:"command_kind,omitempty"`
	SkillName              string    `gorm:"size:160" json:"skill_name,omitempty"`
	RequestID              string    `gorm:"size:64;index" json:"request_id,omitempty"`
	ExecutionID            string    `gorm:"size:64;index" json:"execution_id,omitempty"`
	UsedAI                 bool      `gorm:"not null;default:false" json:"used_ai"`
	RuleHit                bool      `gorm:"not null;default:false" json:"rule_hit"`
	EvidenceCompleteness   string    `gorm:"size:32" json:"evidence_completeness,omitempty"`
	RootCauseDigest        string    `gorm:"size:64;index" json:"root_cause_digest,omitempty"`
	RecommendationDigest   string    `gorm:"size:64" json:"recommendation_digest,omitempty"`
	Payload                JSONB     `gorm:"type:jsonb;not null;default:'{}'" json:"-"`
}

func (DiagnoseSampleRecord) TableName() string { return "diagnose_samples" }
