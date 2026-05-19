package services

import (
	"fmt"
	"strings"
	"time"

	"ft-backend/database"
	"ft-backend/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ExecutionSkillFeedbackResult is the public response for console feedback on an execution.
type ExecutionSkillFeedbackResult struct {
	Recorded   bool                     `json:"recorded"`
	Topic      string                   `json:"topic,omitempty"`
	RequestID  string                   `json:"request_id,omitempty"`
	FeedbackID string                   `json:"feedback_id,omitempty"`
	Evaluation *SkillFeedbackEvalResult `json:"evaluation,omitempty"`
}

// SubmitExecutionSkillFeedback records helpful/unhelpful feedback for a client execution session.
func SubmitExecutionSkillFeedback(executionID uuid.UUID, userID uuid.UUID, role, username string, helpful bool, note string) (*ExecutionSkillFeedbackResult, error) {
	detail, err := GetClientExecutionDetail(executionID, role, username)
	if err != nil {
		return nil, err
	}
	meta := decodeRecordMetadata(detail.Record.Metadata)
	topic := firstNonEmpty(strMeta(meta, "topic"), detail.Record.Category)
	topic = strings.ToLower(strings.TrimSpace(topic))
	if topic == "" {
		return nil, fmt.Errorf("execution has no topic")
	}
	requestID := strMeta(meta, "request_id")
	skillName := firstNonEmpty(strMeta(meta, "skill_pack"), strMeta(meta, "skill_name"))
	if requestID == "" && len(detail.Children) > 0 {
		for _, ch := range detail.Children {
			cm := decodeRecordMetadata(ch.Metadata)
			if rid := strMeta(cm, "request_id"); rid != "" {
				requestID = rid
				if skillName == "" {
					skillName = firstNonEmpty(strMeta(cm, "pack_key"), strMeta(cm, "skill_pack"))
				}
				break
			}
		}
	}
	reg := DefaultSkillRegistry()
	fb := SkillFeedback{
		Topic:     topic,
		SkillName: skillName,
		RequestID: requestID,
		Helpful:   &helpful,
		Note:      limitAuditText(note, 500),
	}
	if err := reg.AppendFeedback(fb); err != nil {
		return nil, err
	}
	eval, _ := ProcessSkillFeedback(reg, fb)
	out := &ExecutionSkillFeedbackResult{
		Recorded:  true,
		Topic:     topic,
		RequestID: requestID,
		Evaluation: eval,
	}
	patch := map[string]interface{}{
		"user_feedback_helpful": helpful,
	}
	if strings.TrimSpace(note) != "" {
		patch["user_feedback_note"] = limitAuditText(note, 200)
	}
	if eval != nil && eval.ReviewTriggered {
		patch["enhancement_review_triggered"] = true
	}
	_ = PatchExecutionRecordMetadata(executionID.String(), patch)

	if userID != uuid.Nil && database.DB != nil {
		summary := firstNonEmpty(strMeta(meta, "root_cause"), strMeta(meta, "summary"), detail.Record.StdoutSummary)
		payload := map[string]interface{}{
			"helpful":                helpful,
			"request_id":             requestID,
			"execution_id":           executionID.String(),
			"skill_name":             skillName,
			"source":                 FeedbackSourceConsoleExec,
			"used_ai":                boolMeta(meta, "used_ai"),
			"rule_hit":               boolMeta(meta, "rule_hit"),
			"evidence_completeness":  strMeta(meta, "evidence_completeness"),
			"root_cause_digest":      strMeta(meta, "root_cause_digest"),
			"recommendation_digest":  strMeta(meta, "recommendation_digest"),
		}
		classification := "console_helpful"
		if !helpful {
			classification = "console_unhelpful"
		}
		row := models.AutoIterationFeedback{
			UserID:         userID,
			Topic:          topic,
			Classification: classification,
			NeedIteration:  false,
			UserMessage:    "感谢反馈，我们已记录。",
		}
		populateAutoIterationFeedbackFields(&row, FeedbackSourceConsoleExec, topic, detail.Record.Command, summary, payload)
		if err := database.DB.Create(&row).Error; err == nil {
			out.FeedbackID = row.ID.String()
		}
	}
	return out, nil
}

// ListAutoIterationFeedbacks returns recent feedback rows for admin review.
func ListAutoIterationFeedbacks(limit int) ([]AutoIterationFeedbackAdminItem, error) {
	if database.DB == nil {
		return nil, nil
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var rows []models.AutoIterationFeedback
	err := database.DB.Order("created_at DESC").Limit(limit).Find(&rows).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	out := make([]AutoIterationFeedbackAdminItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, feedbackAdminItemFromModel(row))
	}
	return out, nil
}

// AutoIterationFeedbackAdminItem is a safe admin view of feedback rows.
type AutoIterationFeedbackAdminItem struct {
	ID                   uuid.UUID  `json:"id"`
	CreatedAt            time.Time  `json:"created_at"`
	Topic                string     `json:"topic"`
	Classification       string     `json:"classification"`
	NeedIteration        bool       `json:"need_iteration"`
	UserMessage          string     `json:"user_message"`
	AutoIterationID      *uuid.UUID `json:"auto_iteration_id,omitempty"`
	Source               string     `json:"source,omitempty"`
	RequestID            string     `json:"request_id,omitempty"`
	ExecutionID          string     `json:"execution_id,omitempty"`
	Command              string     `json:"command,omitempty"`
	Summary              string     `json:"summary,omitempty"`
	SkillName            string     `json:"skill_name,omitempty"`
	Helpful              *bool      `json:"helpful,omitempty"`
	RuleHit              *bool      `json:"rule_hit,omitempty"`
	UsedAI               *bool      `json:"used_ai,omitempty"`
	EvidenceCompleteness string     `json:"evidence_completeness,omitempty"`
}
