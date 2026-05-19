package services

import (
	"testing"
	"time"

	"ft-backend/models"
)

func TestLegacyKindForRecord(t *testing.T) {
	rec := models.ExecutionRecord{Source: "ai", Metadata: models.NewJSONBFromMap(map[string]interface{}{"record_kind": "ai_call"})}
	if got := legacyKindForRecord(rec); got != "legacy_ai_diagnose" {
		t.Fatalf("expected legacy_ai_diagnose, got %q", got)
	}
	rec2 := models.ExecutionRecord{Source: "cli", Metadata: models.NewJSONBFromMap(map[string]interface{}{})}
	if got := legacyKindForRecord(rec2); got != "legacy_cli" {
		t.Fatalf("expected legacy_cli, got %q", got)
	}
	rec3 := models.ExecutionRecord{Metadata: models.NewJSONBFromMap(map[string]interface{}{"record_kind": "client_execution"})}
	if got := legacyKindForRecord(rec3); got != "" {
		t.Fatalf("expected empty legacy, got %q", got)
	}
}

func TestBuildClientExecutionListItem(t *testing.T) {
	rec := &models.ExecutionRecord{
		Command: "ai-sre check redis",
		Status:  models.ExecutionStatusSuccess,
		Metadata: models.NewJSONBFromMap(map[string]interface{}{
			"record_kind":           "client_execution",
			"topic":                 "redis",
			"evidence_completeness": "complete",
			"version":               "0.5.51",
		}),
	}
	rec.CreatedAt = time.Now()
	item, err := buildClientExecutionListItem(rec)
	if err != nil {
		t.Fatal(err)
	}
	if item.Topic != "redis" {
		t.Fatalf("topic=%q", item.Topic)
	}
	if item.EvidenceCompleteness != "complete" {
		t.Fatalf("evidence=%q", item.EvidenceCompleteness)
	}
}
