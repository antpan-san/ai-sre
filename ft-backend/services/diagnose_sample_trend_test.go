package services

import (
	"testing"
	"time"
)

func TestTrendDiagnoseSamplesBuckets(t *testing.T) {
	t.Parallel()
	reg, err := NewSkillRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Hour)
	for i := 0; i < 3; i++ {
		s := DiagnoseSample{
			Time:         now.Add(-time.Duration(i) * 24 * time.Hour),
			Topic:        "redis",
			SampleSource: "cli_check",
			CommandKind:  "check",
			RuleHit:      i%2 == 0,
			UsedAI:       i%2 == 1,
		}
		if err := reg.AppendSample(s); err != nil {
			t.Fatal(err)
		}
	}
	trend, err := TrendDiagnoseSamples(reg, now.Add(-7*24*time.Hour), 168, 24)
	if err != nil {
		t.Fatal(err)
	}
	if len(trend.Buckets) == 0 {
		t.Fatal("expected buckets")
	}
	total := 0
	for _, b := range trend.Buckets {
		total += b.Total
	}
	if total != 3 {
		t.Fatalf("expected 3 samples across buckets, got %d (%+v)", total, trend.Buckets)
	}
}
