package go_runtime

import (
	"testing"
	"time"
)

func TestAnalyzeTrendMonotonicRSS(t *testing.T) {
	t0 := time.Unix(1_000_000, 0)
	reports := []*Report{
		{GeneratedAt: t0, Snapshot: ProcSnapshot{Status: ProcStatus{VmRSSBytes: 100 * MiB}}},
		{GeneratedAt: t0.Add(time.Second), Snapshot: ProcSnapshot{Status: ProcStatus{VmRSSBytes: 200 * MiB}}},
		{GeneratedAt: t0.Add(2 * time.Second), Snapshot: ProcSnapshot{Status: ProcStatus{VmRSSBytes: 400 * MiB}}},
	}
	findings := AnalyzeTrend(reports)
	if len(findings) < 1 {
		t.Fatalf("expected RSS trend finding, got %+v", findings)
	}
	if findings[0].Title == "" {
		t.Fatal("empty title")
	}
}

func TestAnalyzeTrendHighCPU(t *testing.T) {
	t0 := time.Unix(2_000_000, 0)
	reports := []*Report{
		{
			GeneratedAt: t0,
			Snapshot: ProcSnapshot{
				Stat: ProcStat{UtimeTicks: 0, StimeTicks: 0},
			},
			Cgroup: CgroupMetrics{},
		},
		{
			GeneratedAt: t0.Add(2 * time.Second),
			Snapshot: ProcSnapshot{
				Stat: ProcStat{UtimeTicks: 300, StimeTicks: 0},
			},
			Cgroup: CgroupMetrics{},
		},
	}
	findings := AnalyzeTrend(reports)
	found := false
	for _, f := range findings {
		if f.Title == "采样窗口内 CPU 时间占比偏高" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected CPU finding, got %+v", findings)
	}
}
