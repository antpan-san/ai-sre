package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestBuiltinSkillsLoadAndValidate asserts that every embedded YAML pack parses
// and satisfies the minimal schema validation.
func TestBuiltinSkillsLoadAndValidate(t *testing.T) {
	r, err := NewSkillRegistry("")
	if err != nil {
		t.Fatalf("NewSkillRegistry: %v", err)
	}
	for _, topic := range []string{"k8s", "kafka", "redis", "mysql", "nginx", "elasticsearch"} {
		rs := r.Match(topic, nil)
		if rs == nil {
			t.Fatalf("expected builtin skill for topic=%s", topic)
		}
		if rs.Source != SkillSourceBuiltin {
			t.Fatalf("topic=%s: expected source=builtin, got %s", topic, rs.Source)
		}
		if !ValidateSkillDraft(&rs.Pack) {
			t.Fatalf("topic=%s: builtin pack fails ValidateSkillDraft", topic)
		}
		if len(rs.Pack.AnalysisSteps) < 4 {
			t.Fatalf("topic=%s: analysis_steps should be >=4, got %d", topic, len(rs.Pack.AnalysisSteps))
		}
		if len(rs.Pack.OutputFormat) == 0 {
			t.Fatalf("topic=%s: output_format should be non-empty", topic)
		}
	}
}

func TestSkillRegistryGeneratedOverridesBuiltin(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "generated"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// hand-craft a generated YAML pack that should win against the builtin k8s pack
	gen := `name: k8s_root_cause_v_test
display_name: "Test refined k8s skill"
topics: [k8s]
match_keywords: [test]
input: [issue]
analysis_steps:
  - step1
  - step2
  - step3
  - step4
output_format:
  - root_cause
  - evidence
  - fix
extra_guidance: "test"
`
	if err := os.WriteFile(filepath.Join(dir, "generated", "k8s.yaml"), []byte(gen), 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
	r, err := NewSkillRegistry(dir)
	if err != nil {
		t.Fatalf("NewSkillRegistry: %v", err)
	}
	rs := r.Match("k8s", nil)
	if rs == nil {
		t.Fatalf("expected match for topic=k8s")
	}
	if rs.Source != SkillSourceGenerated {
		t.Fatalf("expected source=generated, got %s", rs.Source)
	}
	if rs.Pack.Name != "k8s_root_cause_v_test" {
		t.Fatalf("expected refined name to win, got %s", rs.Pack.Name)
	}
}

func TestSkillRegistryAppendSampleAndFeedback(t *testing.T) {
	dir := t.TempDir()
	r, err := NewSkillRegistry(dir)
	if err != nil {
		t.Fatalf("NewSkillRegistry: %v", err)
	}
	helpful := true
	if err := r.AppendSample(DiagnoseSample{
		Topic:      "k8s",
		SkillName:  "k8s_root_cause_v1",
		Style:      "evidence_root_cause",
		AnswerHead: "head",
		AnswerLen:  4,
		Time:       time.Now().UTC(),
	}); err != nil {
		t.Fatalf("AppendSample: %v", err)
	}
	if err := r.AppendFeedback(SkillFeedback{
		Topic:   "k8s",
		Helpful: &helpful,
		Note:    "ok",
	}); err != nil {
		t.Fatalf("AppendFeedback: %v", err)
	}
	samples, err := r.ReadRecentSamples("k8s", 10)
	if err != nil || len(samples) != 1 || samples[0].Topic != "k8s" {
		t.Fatalf("ReadRecentSamples = %v (err=%v)", samples, err)
	}
	fbs, err := r.ReadRecentFeedback("k8s", 10)
	if err != nil || len(fbs) != 1 || fbs[0].Helpful == nil || !*fbs[0].Helpful {
		t.Fatalf("ReadRecentFeedback = %v (err=%v)", fbs, err)
	}
	rawSample, err := os.ReadFile(filepath.Join(dir, "samples", "k8s.jsonl"))
	if err != nil || !strings.Contains(string(rawSample), `"topic":"k8s"`) {
		t.Fatalf("sample jsonl not written correctly: %s err=%v", string(rawSample), err)
	}
	rawFb, err := os.ReadFile(filepath.Join(dir, "feedback", "k8s.jsonl"))
	if err != nil || !strings.Contains(string(rawFb), `"helpful":true`) {
		t.Fatalf("feedback jsonl not written correctly: %s err=%v", string(rawFb), err)
	}
	// sanity: each line is valid JSON
	for _, line := range strings.Split(strings.TrimSpace(string(rawSample)), "\n") {
		var anyV any
		if err := json.Unmarshal([]byte(line), &anyV); err != nil {
			t.Fatalf("invalid sample jsonl line: %s", line)
		}
	}
}

func TestSaveGeneratedArchivesPrevious(t *testing.T) {
	dir := t.TempDir()
	r, err := NewSkillRegistry(dir)
	if err != nil {
		t.Fatalf("NewSkillRegistry: %v", err)
	}
	mk := func(name string) *SkillPack {
		return &SkillPack{
			Name:          name,
			DisplayName:   "x",
			Topics:        []string{"k8s"},
			MatchKeywords: []string{"k8s"},
			Input:         []string{"issue"},
			AnalysisSteps: []string{"a", "b", "c", "d"},
			OutputFormat:  []string{"root_cause", "evidence", "fix"},
		}
	}
	if _, err := r.SaveGenerated(mk("k8s_v_a")); err != nil {
		t.Fatalf("save first: %v", err)
	}
	if _, err := r.SaveGenerated(mk("k8s_v_b")); err != nil {
		t.Fatalf("save second: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(dir, "generated", "k8s.history"))
	if err != nil {
		t.Fatalf("read history dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected archived previous, got 0")
	}
	rs := r.Match("k8s", nil)
	if rs == nil || rs.Pack.Name != "k8s_v_b" {
		t.Fatalf("after save, expected k8s_v_b to be active, got %+v", rs)
	}
}

func TestExtractYAMLBlock(t *testing.T) {
	good := "前置文本\n```yaml\nname: x_v1\nfoo: bar\n```\n后置文本"
	if got := extractYAMLBlock(good); !strings.Contains(got, "name: x_v1") {
		t.Fatalf("extractYAMLBlock failed: %q", got)
	}
	bare := "name: x_v1\nfoo: bar"
	if got := extractYAMLBlock(bare); !strings.Contains(got, "name: x_v1") {
		t.Fatalf("extractYAMLBlock bare failed: %q", got)
	}
	if extractYAMLBlock("just some text") != "" {
		t.Fatalf("extractYAMLBlock should return empty for non-yaml")
	}
}

func TestNextRefinedName(t *testing.T) {
	if got := nextRefinedName("k8s_root_cause_v1"); got != "k8s_root_cause_v2" {
		t.Fatalf("nextRefinedName v1 -> v2 failed: %s", got)
	}
	if got := nextRefinedName("foo"); got != "foo_v2" {
		t.Fatalf("nextRefinedName foo failed: %s", got)
	}
	if got := nextRefinedName(""); got != "auto_refined_v1" {
		t.Fatalf("nextRefinedName empty failed: %s", got)
	}
}
