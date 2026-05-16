package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeSkillPackWithRegistry(t *testing.T) {
	dir := t.TempDir()
	builtinDir := filepath.Join(dir, "builtin")
	if err := os.MkdirAll(builtinDir, 0o755); err != nil {
		t.Fatal(err)
	}
	baseYAML := []byte(`name: k8s_root_cause_v1
display_name: K8s base
topics: [k8s]
match_keywords: [k8s]
input: [namespace]
analysis_steps:
  - step one
  - step two
output_format: [root_cause]
extra_guidance: base guidance
`)
	if err := os.WriteFile(filepath.Join(builtinDir, "k8s.yaml"), baseYAML, 0o644); err != nil {
		t.Fatal(err)
	}
	reg, err := NewSkillRegistry(dir)
	if err != nil {
		t.Fatalf("NewSkillRegistry: %v", err)
	}
	incoming := &SkillPack{
		Name:          "k8s_diagnostic_readonly",
		DisplayName:   "diag",
		Topics:        []string{"k8s"},
		AnalysisSteps: []string{"diag step"},
		OutputFormat:  []string{"root_cause", "solution", "verification_commands"},
		ExtraGuidance: "diag extra",
		MatchKeywords: []string{"readonly"},
		Input:         []string{"pod"},
	}
	merged, did := MergeSkillPackWithRegistry(reg, incoming)
	if !did {
		t.Fatalf("expected merge")
	}
	if merged.Name == "" {
		t.Fatalf("expected base pack name")
	}
	if !containsStr(merged.AnalysisSteps, "diag step") {
		t.Fatalf("diag steps not merged: %v", merged.AnalysisSteps)
	}
	if len(merged.AnalysisSteps) < 3 {
		t.Fatalf("expected base+diag steps, got %v", merged.AnalysisSteps)
	}
	if !strings.Contains(merged.ExtraGuidance, "diag extra") {
		t.Fatalf("guidance not merged: %q", merged.ExtraGuidance)
	}
}

func containsStr(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
