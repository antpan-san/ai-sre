package handlers

import (
	"encoding/json"
	"strings"
	"time"

	"ft-backend/models"
)

func applyWatchDiagnosisToSession(sess *models.RuntimeWatchSession, watch map[string]interface{}) {
	if sess == nil || watch == nil {
		return
	}
	target := mapFromAny(watch["target"])
	summary := mapFromAny(watch["summary"])
	diag := mapFromAny(watch["diagnosis"])

	level := cleanString(summary["level"])
	root := cleanString(diag["root_cause"])
	evidence := cleanString(diag["evidence"])
	source := cleanString(diag["diagnosis_source"])
	if source == "" {
		source = cleanString(diag["source"])
	}
	if root == "" {
		root = cleanString(summary["title"])
	}
	if evidence == "" {
		evidence = cleanString(summary["evidence"])
	}

	sess.ResourceKind = cleanString(target["resource_kind"])
	sess.ResourceName = cleanString(target["resource_name"])
	sess.WorkPod = cleanString(target["pod"])
	sess.TargetDisplay = cleanString(target["target"])
	if sess.TargetDisplay == "" && sess.ResourceKind != "" && sess.ResourceName != "" {
		sess.TargetDisplay = sess.ResourceKind + "/" + sess.Namespace + "/" + sess.ResourceName
	}
	if sess.TargetDisplay == "" {
		sess.TargetDisplay = strings.Trim(strings.Join([]string{sess.Namespace, sess.Pod}, "/"), "/")
	}
	sess.DiagnosisLevel = level
	sess.RootCause = root
	sess.Evidence = evidence
	sess.DiagnosisSource = source
	sess.SampleCount = intFromAny(watch["sample_count"])
	now := time.Now().UTC()
	sess.LastDiagnosedAt = &now
}

func extractDiagnosisFromWatchJSON(raw []byte) map[string]interface{} {
	var watch map[string]interface{}
	if json.Unmarshal(raw, &watch) != nil {
		return nil
	}
	return watch
}

func sessionDiagnosisRow(r models.RuntimeWatchSession) map[string]interface{} {
	row := map[string]interface{}{
		"id":               r.ID,
		"namespace":        r.Namespace,
		"pod":              r.Pod,
		"container":        r.Container,
		"interval_sec":     r.IntervalSec,
		"status":           r.Status,
		"created_at":       r.CreatedAt,
		"machine_note":     r.MachineNote,
		"target_display":   r.TargetDisplay,
		"resource_kind":    r.ResourceKind,
		"resource_name":    r.ResourceName,
		"work_pod":         r.WorkPod,
		"diagnosis_level":  r.DiagnosisLevel,
		"root_cause":       r.RootCause,
		"evidence":         r.Evidence,
		"diagnosis_source": r.DiagnosisSource,
		"sample_count":     r.SampleCount,
		"has_diagnosis":    strings.TrimSpace(r.RootCause) != "",
	}
	if r.LastDiagnosedAt != nil {
		row["last_diagnosed_at"] = r.LastDiagnosedAt
	}
	return row
}
