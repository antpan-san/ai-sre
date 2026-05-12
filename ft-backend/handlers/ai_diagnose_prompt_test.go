package handlers

import (
	"strings"
	"testing"
)

func TestBuildServerDiagnosePromptEvidenceRootCause(t *testing.T) {
	p := buildServerDiagnosePrompt("k8s", map[string]string{
		"diagnosis_style": "evidence_root_cause",
		"pod":             "pending",
		"kubectl_nodes":   "NAME     STATUS\nmaster   Ready",
	})
	if !strings.Contains(p, "集群采集输出") {
		head := p
		if len(head) > 200 {
			head = head[:200]
		}
		t.Fatalf("expected evidence section header, got: %s", head)
	}
	if !strings.Contains(p, "禁止") || !strings.Contains(p, "kubectl") {
		t.Fatalf("expected anti-tutorial instruction")
	}
	if !strings.Contains(p, "### kubectl_nodes") {
		t.Fatalf("expected labeled kubectl block")
	}
}

func TestBuildServerDiagnosePromptEvidenceRefine(t *testing.T) {
	p := buildServerDiagnosePrompt("k8s", map[string]string{
		"diagnosis_style":       "evidence_root_cause_refine",
		"kubectl_nodes":         "NAME\nm Ready",
		"prior_answer_round1":   "第一轮泛泛而谈",
	})
	if !strings.Contains(p, "第二轮") || !strings.Contains(p, "第一轮模型回答") {
		t.Fatalf("expected refine preamble")
	}
	if !strings.Contains(p, "第一轮泛泛而谈") {
		t.Fatalf("expected prior answer embedded")
	}
}

func TestBuildEvidenceRootCausePromptFocusOrderingAndRule(t *testing.T) {
	p := buildEvidenceRootCausePrompt("k8s", map[string]string{
		"diagnosis_style":        "evidence_root_cause",
		"kubectl_nodes":          "NODES",
		"kubectl_focus_describe": "DESCRIBE",
	}, false)
	iF := strings.Index(p, "### kubectl_focus_describe")
	iN := strings.Index(p, "### kubectl_nodes")
	if iF < 0 || iN < 0 || iF >= iN {
		t.Fatalf("kubectl_focus_* block must precede other evidence: focus=%d nodes=%d", iF, iN)
	}
	if !strings.Contains(p, "待深挖的 Pod") {
		t.Fatalf("expected focus-pod rule in prompt")
	}
}

func TestKvForSkillDraftStripsBulkEvidence(t *testing.T) {
	in := map[string]string{
		"pod":             "pending",
		"kubectl_nodes":   "huge",
		"diagnosis_style": "evidence_root_cause",
	}
	out := kvForSkillDraft(in)
	if out["kubectl_nodes"] != "" {
		t.Fatalf("kubectl_* should be stripped")
	}
	if out["diagnosis_style"] != "" {
		t.Fatalf("diagnosis_style should be stripped")
	}
	if out["pod"] != "pending" {
		t.Fatalf("expected pod flag preserved")
	}
}