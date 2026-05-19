package handlers

import (
	"strings"
	"testing"
)

func TestSkillPackForTopicLinux(t *testing.T) {
	if got := skillPackForTopic("linux"); got != "pack.backup_performance" {
		t.Fatalf("skillPackForTopic(linux)=%q", got)
	}
}

func TestIsCollectedEvidenceKeyLinuxPrefix(t *testing.T) {
	if !isCollectedEvidenceKey("linux_perf_probe_json") {
		t.Fatal("linux_perf_probe_json should be evidence")
	}
}

func TestBuildLinuxPerformancePromptIncludesSections(t *testing.T) {
	p := buildLinuxPerformanceEvidencePromptWithSkill("linux", map[string]string{
		"linux_perf_probe_json": `{"topic":"linux","cpu":{"user_pct":10}}`,
	}, nil)
	for _, sec := range []string{"【根因判断】", "【关键证据】", "linux_perf_probe_json"} {
		if !strings.Contains(p, sec) {
			t.Fatalf("prompt missing %q", sec)
		}
	}
}

func TestHasDiagnoseProbeJSON(t *testing.T) {
	if !hasDiagnoseProbeJSON(map[string]string{"linux_perf_probe_json": "{}"}) {
		t.Fatal("expected probe json detected")
	}
}
