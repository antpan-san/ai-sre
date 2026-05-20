package services

import (
	"fmt"
	"strings"
	"time"

	"ft-backend/database"
	"ft-backend/models"

	"github.com/google/uuid"
)

// RecordAutoIterationExperienceSample appends a post-completion experience sample for skill accumulation.
func RecordAutoIterationExperienceSample(row models.AutoIteration, outcome, summary, enhanceStatus, enhanceReason string) error {
	topic := strings.ToLower(strings.TrimSpace(row.Topic))
	if topic == "" {
		return nil
	}
	meta := decodeRecordMetadata(row.Metadata)
	execID := strMeta(meta, "execution_id")
	rootDigest := strMeta(meta, "root_cause_digest")
	cmd := firstNonEmpty(strings.TrimSpace(row.Command), strMeta(meta, "repro_command"))
	body := strings.TrimSpace(firstNonEmpty(summary, row.Summary, row.Description))
	if body == "" {
		body = fmt.Sprintf("auto_iteration %s: %s", strings.TrimSpace(row.Source), strings.TrimSpace(row.Title))
	}
	recs := []string{
		fmt.Sprintf("自动迭代完成（%s）", limitAuditText(outcome, 80)),
	}
	if v := strings.TrimSpace(enhanceStatus); v != "" {
		recs = append(recs, "技能包增强: "+v)
	}
	if v := strings.TrimSpace(enhanceReason); v != "" {
		recs = append(recs, limitAuditText(v, 300))
	}
	if v := strings.TrimSpace(row.Source); v != "" {
		recs = append(recs, "来源: "+v)
	}
	review := SkillEnhancementReview{
		Time:              time.Now().UTC(),
		Topic:             topic,
		CommandKind:       "auto_iteration",
		RequestID:         strMeta(meta, "request_id"),
		NeedsEnhancement:  strings.EqualFold(enhanceStatus, "partial"),
		Priority:          "low",
		SavingsScore:      5,
		EnhancementStatus: "recorded",
		Recommendations:   recs,
		SuggestedActions:  []string{"verify_local_rule", "monitor_similar_samples"},
	}
	if strings.EqualFold(enhanceStatus, "yes") {
		review.SavingsScore = 25
		review.NeedsEnhancement = false
	}
	sample := DiagnoseSample{
		Time:                 time.Now().UTC(),
		Topic:                topic,
		Command:              limitAuditText(cmd, 500),
		CommandKind:          "auto_iteration",
		RequestID:            strMeta(meta, "request_id"),
		ExecutionID:          execID,
		AnswerHead:           headRunes(body, 800),
		AnswerLen:            len(body),
		RootCauseDigest:      firstNonEmpty(rootDigest, digestSampleText(body)),
		SampleSource:         "auto_iteration_completed",
		SampleStatus:         strings.TrimSpace(outcome),
		EnhancementReview:    &review,
		UserContext: map[string]string{
			"auto_iteration_id": row.ID.String(),
			"auto_iteration_source": strings.TrimSpace(row.Source),
			"skill_pack_enhanced": strings.TrimSpace(enhanceStatus),
		},
	}
	if err := AppendDiagnoseSample(DefaultSkillRegistry(), sample); err != nil {
		return err
	}
	patch := map[string]interface{}{
		"auto_iteration_experience_recorded": true,
		"auto_iteration_experience_at":       time.Now().UTC().Format(time.RFC3339),
	}
	if execID != "" {
		_ = PatchExecutionRecordMetadata(execID, map[string]interface{}{
			"auto_iteration_experience_recorded": true,
			"auto_iteration_id":                  row.ID.String(),
		})
	}
	_ = databasePatchAutoIterationMeta(row.ID, patch)
	return nil
}

func databasePatchAutoIterationMeta(id uuid.UUID, patch map[string]interface{}) error {
	if database.DB == nil || len(patch) == 0 {
		return nil
	}
	var row models.AutoIteration
	if err := database.DB.Where("id = ?", id).First(&row).Error; err != nil {
		return err
	}
	meta := mergeAutoIterationMeta(row.Metadata, patch)
	return database.DB.Model(&row).Update("metadata", meta).Error
}

func shouldRecordAutoIterationExperience(success bool, status string) bool {
	if !success {
		return false
	}
	switch status {
	case models.AutoIterationStatusCompleted,
		models.AutoIterationStatusApproved,
		models.AutoIterationStatusAwaitingApproval:
		return true
	default:
		return false
	}
}

func isSkillAccumulationSource(source string) bool {
	switch strings.TrimSpace(source) {
	case models.AutoIterationSourceSkillRefine,
		models.AutoIterationSourceRuleCandidate,
		models.AutoIterationSourceAICostReduce,
		models.AutoIterationSourceDiagnosisGap,
		models.AutoIterationSourceCLIFeedback,
		models.AutoIterationSourceCapabilityGap:
		return true
	default:
		return false
	}
}
