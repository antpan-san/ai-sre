package services

import (
	"testing"
	"time"
)

func TestSummarizeDiagnoseSamplesEmpty(t *testing.T) {
	t.Parallel()
	reg, err := NewSkillRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sum, err := SummarizeDiagnoseSamples(reg, time.Now().Add(-24*time.Hour), 24)
	if err != nil {
		t.Fatal(err)
	}
	if sum.TotalSamples != 0 {
		t.Fatalf("expected 0 samples, got %d", sum.TotalSamples)
	}
}

func TestListDiagnoseSamplesAfterAppend(t *testing.T) {
	t.Parallel()
	reg, err := NewSkillRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := DiagnoseSample{
		Time:         time.Now().UTC(),
		Topic:        "redis",
		CommandKind:  "check",
		SampleSource: "cli_check",
		RuleHit:      true,
		Target:       "127.0.0.1:6379",
	}
	if err := reg.AppendSample(s); err != nil {
		t.Fatal(err)
	}
	rows, err := ListDiagnoseSamples(reg, "redis", 10, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 sample, got %d", len(rows))
	}
	sum, err := SummarizeDiagnoseSamples(reg, time.Now().Add(-time.Hour), 1)
	if err != nil {
		t.Fatal(err)
	}
	if sum.TotalSamples != 1 || sum.RuleHitCount != 1 {
		t.Fatalf("summary=%+v", sum)
	}
}
