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
	Recorded        bool                     `json:"recorded"`
	Topic           string                   `json:"topic,omitempty"`
	RequestID       string                   `json:"request_id,omitempty"`
	Evaluation      *SkillFeedbackEvalResult `json:"evaluation,omitempty"`
}

// SubmitExecutionSkillFeedback records helpful/unhelpful feedback for a client execution session.
func SubmitExecutionSkillFeedback(executionID uuid.UUID, role, username string, helpful bool, note string) (*ExecutionSkillFeedbackResult, error) {
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
	return out, nil
}

// ListAutoIterationFeedbacks returns recent CLI feedback rows for admin review.
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
		meta := decodeRecordMetadata(row.RawPayload)
		item := AutoIterationFeedbackAdminItem{
			ID:              row.ID,
			CreatedAt:       row.CreatedAt,
			Topic:           row.Topic,
			Classification:  row.Classification,
			NeedIteration:   row.NeedIteration,
			UserMessage:     row.UserMessage,
			AutoIterationID: row.AutoIterationID,
			RequestID:       strMeta(meta, "request_id"),
			ExecutionID:     strMeta(meta, "execution_id"),
			Command:         strMeta(meta, "command"),
			Summary:         strMeta(meta, "summary"),
		}
		if v, ok := meta["helpful"].(bool); ok {
			item.Helpful = &v
		}
		out = append(out, item)
	}
	return out, nil
}

// AutoIterationFeedbackAdminItem is a safe admin view of CLI feedback rows.
type AutoIterationFeedbackAdminItem struct {
	ID              uuid.UUID  `json:"id"`
	CreatedAt       time.Time  `json:"created_at"`
	Topic           string     `json:"topic"`
	Classification  string     `json:"classification"`
	NeedIteration   bool       `json:"need_iteration"`
	UserMessage     string     `json:"user_message"`
	AutoIterationID *uuid.UUID `json:"auto_iteration_id,omitempty"`
	RequestID       string     `json:"request_id,omitempty"`
	ExecutionID     string     `json:"execution_id,omitempty"`
	Command         string     `json:"command,omitempty"`
	Summary         string     `json:"summary,omitempty"`
	Helpful         *bool      `json:"helpful,omitempty"`
}
