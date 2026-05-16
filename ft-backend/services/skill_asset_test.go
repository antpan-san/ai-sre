package services

import (
	"testing"

	"ft-backend/models"
)

func TestSkillPackFromDiagnosticAsset(t *testing.T) {
	asset := &models.SkillAsset{
		Topic:       "k8s",
		DisplayName: "只读诊断计划: k8s",
		Name:        "diagnostic.k8s.readonly-plan",
	}
	content := map[string]interface{}{
		"topic":               "k8s",
		"observation_summary": "Pod CrashLoopBackOff in prod",
		"steps": []interface{}{
			map[string]interface{}{"id": "a", "title": "读取节点状态"},
			map[string]interface{}{"id": "b", "title": "读取 Pod 日志"},
		},
		"observations": map[string]interface{}{
			"kubectl_focus_logs_current": "error: back-off",
		},
	}
	ver := &models.SkillAssetVersion{Content: models.NewJSONBFromMap(content)}
	pack, err := SkillPackFromDiagnosticAsset(asset, ver)
	if err != nil {
		t.Fatalf("SkillPackFromDiagnosticAsset: %v", err)
	}
	if !ValidateSkillDraft(pack) {
		t.Fatalf("invalid pack: %+v", pack)
	}
	if pack.Name != "k8s_diagnostic_readonly" {
		t.Fatalf("unexpected name %q", pack.Name)
	}
	if len(pack.AnalysisSteps) < 2 {
		t.Fatalf("expected analysis steps")
	}
	if pack.ExtraGuidance == "" {
		t.Fatalf("expected extra guidance")
	}
}

func TestAnalysisStepsFromDiagnosticContentFallback(t *testing.T) {
	steps := analysisStepsFromDiagnosticContent(map[string]interface{}{})
	if len(steps) < 2 {
		t.Fatalf("expected fallback steps")
	}
}
