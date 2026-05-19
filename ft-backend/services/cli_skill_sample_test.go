package services

import (
	"testing"

	"github.com/google/uuid"
)

func TestClassifyCLISkillSample(t *testing.T) {
	t.Parallel()
	if got := classifyCLISkillSample(CLISkillSampleInput{RuleHit: true}); got != "valuable_sample" {
		t.Fatalf("rule hit: got %q", got)
	}
	if got := classifyCLISkillSample(CLISkillSampleInput{UsedAI: true, EvidenceCompleteness: "partial"}); got != "diagnosis_insufficient" {
		t.Fatalf("partial evidence: got %q", got)
	}
	if got := classifyCLISkillSample(CLISkillSampleInput{UsedAI: true, EvidenceCompleteness: "complete"}); got != "rule_candidate" {
		t.Fatalf("ai without rule: got %q", got)
	}
}

func TestCLISkillSampleDedup(t *testing.T) {
	t.Parallel()
	key := "test-dedup-key"
	if cliSkillSampleRecentlySeen(key) {
		t.Fatal("expected fresh key")
	}
	markCLISkillSampleSeen(key)
	if !cliSkillSampleRecentlySeen(key) {
		t.Fatal("expected seen key")
	}
}

func TestSanitizeCLISampleContext(t *testing.T) {
	t.Parallel()
	out := sanitizeCLISampleContext(map[string]string{
		"password": "secret",
		"host":     "127.0.0.1",
		"dsn":      "postgres://user:pass@127.0.0.1/db",
	})
	if out["password"] != "<redacted>" {
		t.Fatalf("password=%q", out["password"])
	}
	if out["host"] != "127.0.0.1" {
		t.Fatalf("host=%q", out["host"])
	}
	if out["dsn"] == "postgres://user:pass@127.0.0.1/db" {
		t.Fatal("expected dsn redaction")
	}
}

func TestPatchExecutionRecordMetadataInvalidID(t *testing.T) {
	t.Parallel()
	if err := PatchExecutionRecordMetadata("not-a-uuid", map[string]interface{}{"x": 1}); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	_ = uuid.Nil
}
