package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	ExecutionEngagementView                = "view"
	ExecutionEngagementCopyRootCause       = "copy_root_cause"
	ExecutionEngagementCopyRecommendations = "copy_recommendations"
	ExecutionEngagementCopyEvidence        = "copy_evidence"
	ExecutionEngagementCopyFull            = "copy_full"
)

// RecordExecutionEngagementSample appends a lightweight console engagement sample.
func RecordExecutionEngagementSample(executionID uuid.UUID, role, username, action string) error {
	action = strings.ToLower(strings.TrimSpace(action))
	if action == "" {
		return fmt.Errorf("action required")
	}
	detail, err := GetClientExecutionDetail(executionID, role, username)
	if err != nil {
		return err
	}
	meta := decodeRecordMetadata(detail.Record.Metadata)
	topic := strings.ToLower(strings.TrimSpace(firstNonEmpty(strMeta(meta, "topic"), detail.Record.Category)))
	if topic == "" {
		return fmt.Errorf("execution has no topic")
	}
	target := firstNonEmpty(strMeta(meta, "target"), strMeta(meta, "diagnosis_target"), detail.Record.TargetHost, detail.Record.ResourceName)
	summary := firstNonEmpty(strMeta(meta, "root_cause"), strMeta(meta, "summary"), detail.Record.StdoutSummary)
	sample := DiagnoseSample{
		Time:                 time.Now().UTC(),
		Topic:                topic,
		Target:               limitAuditText(target, 200),
		Command:              limitAuditText(detail.Record.Command, 500),
		CommandKind:          "engagement",
		RequestID:            strMeta(meta, "request_id"),
		ExecutionID:          executionID.String(),
		UsedAI:               boolMeta(meta, "used_ai"),
		RuleHit:              boolMeta(meta, "rule_hit"),
		EvidenceCompleteness: strMeta(meta, "evidence_completeness"),
		RootCauseDigest:      digestSampleText(summary),
		AnswerHead:           headRunes(fmt.Sprintf("[%s] %s", action, summary), 600),
		SampleSource:         "console_" + action,
		SampleStatus:         "engagement",
	}
	if err := AppendDiagnoseSample(DefaultSkillRegistry(), sample); err != nil {
		return err
	}
	patch := map[string]interface{}{
		"engagement_" + action + "_at": time.Now().UTC().Format(time.RFC3339),
	}
	if action == ExecutionEngagementView {
		patch["execution_view_recorded"] = true
	}
	return PatchExecutionRecordMetadata(executionID.String(), patch)
}

func digestSampleText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:16])
}
