package cli

import "testing"

func TestClassifyCheckSample(t *testing.T) {
	t.Parallel()
	if got := classifyCheckSample(skillSampleReportInput{RuleHit: true}); got != "valuable_sample" {
		t.Fatalf("rule hit: got %q", got)
	}
	if got := classifyCheckSample(skillSampleReportInput{EvidenceCompleteness: "missing"}); got != "diagnosis_insufficient" {
		t.Fatalf("missing evidence: got %q", got)
	}
	if got := classifyCheckSample(skillSampleReportInput{UsedAI: true, EvidenceCompleteness: "complete"}); got != "rule_candidate" {
		t.Fatalf("ai: got %q", got)
	}
}

func TestDigestText(t *testing.T) {
	t.Parallel()
	a := digestText("hello")
	b := digestText("hello")
	if a == "" || a != b {
		t.Fatalf("digest unstable: %q %q", a, b)
	}
	if digestText("  HELLO  ") != a {
		t.Fatal("digest should normalize case/space")
	}
}

func TestScrubSampleContext(t *testing.T) {
	t.Parallel()
	out := scrubSampleContext(map[string]string{
		"password":            "x",
		"redis_diagnose_json": "{}",
		"host":                "127.0.0.1",
	})
	if out["password"] != "<redacted>" {
		t.Fatalf("password=%q", out["password"])
	}
	if out["redis_diagnose_json"] != "<evidence_omitted>" {
		t.Fatalf("json=%q", out["redis_diagnose_json"])
	}
}

func TestSkillSampleDedupLocal(t *testing.T) {
	t.Parallel()
	key := "local-dedup"
	if skillSampleRecentlyReported(key) {
		t.Fatal("fresh")
	}
	markSkillSampleReported(key)
	if !skillSampleRecentlyReported(key) {
		t.Fatal("seen")
	}
}
