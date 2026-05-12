package handlers

import (
	"strings"
	"testing"

	"ft-backend/services"
)

func TestBuildServerDiagnosePromptInjectsMatchedSkill(t *testing.T) {
	reg, err := services.NewSkillRegistry("")
	if err != nil {
		t.Fatalf("NewSkillRegistry: %v", err)
	}
	matched := reg.Match("kafka", nil)
	if matched == nil {
		t.Fatalf("expected kafka builtin to be loaded")
	}
	prompt := buildServerDiagnosePromptWithSkill("kafka", map[string]string{"lag": "9000"}, matched)
	for _, want := range []string{
		"【适用技能包】",
		matched.Pack.Name,
		"分析步骤",
		matched.Pack.AnalysisSteps[0],
		"输出结构",
		"## ",
		"- lag=9000",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q\n--- prompt ---\n%s", want, prompt)
		}
	}
}

func TestBuildEvidenceRootCausePromptInjectsMatchedSkill(t *testing.T) {
	reg, err := services.NewSkillRegistry("")
	if err != nil {
		t.Fatalf("NewSkillRegistry: %v", err)
	}
	matched := reg.Match("k8s", nil)
	if matched == nil {
		t.Fatalf("expected k8s builtin to be loaded")
	}
	prompt := buildEvidenceRootCausePromptWithSkill("k8s", map[string]string{
		"diagnosis_style":          "evidence_root_cause",
		"kubectl_focus_describe":   "Name: kube-controller-manager-k8s-master-0\nEvents:\n  ProbeFailed",
		"kubectl_nodes":            "NAME STATUS\nk8s-master-0 NotReady",
		"namespace":                "kube-system",
	}, false, matched)
	for _, want := range []string{
		"集群采集输出（原文）",
		"kubectl_focus_describe",
		"kubectl_nodes",
		"【适用技能包】",
		matched.Pack.Name,
		matched.Pack.AnalysisSteps[0],
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q", want)
		}
	}
}

func TestEvidenceKeyListAndStripBulk(t *testing.T) {
	kv := map[string]string{
		"namespace":         "kube-system",
		"kubectl_nodes":     "huge output",
		"host_uptime":       "10d",
		"prior_answer_round1": "previous answer",
	}
	keys := evidenceKeyList(kv)
	want := []string{"host_uptime", "kubectl_nodes"}
	if len(keys) != len(want) || keys[0] != want[0] || keys[1] != want[1] {
		t.Fatalf("evidenceKeyList = %v, want %v", keys, want)
	}
	stripped := stripBulkEvidenceForSample(kv)
	if _, has := stripped["kubectl_nodes"]; has {
		t.Fatalf("strip should drop kubectl_*")
	}
	if _, has := stripped["host_uptime"]; has {
		t.Fatalf("strip should drop host_*")
	}
	if _, has := stripped["prior_answer_round1"]; has {
		t.Fatalf("strip should drop prior_answer_round1")
	}
	if stripped["namespace"] != "kube-system" {
		t.Fatalf("strip should keep user flags")
	}
}
