package services

import (
	"testing"

	"ft-backend/models"
)

func TestNormalizeSkillExecutionIntentK8sPending(t *testing.T) {
	got := NormalizeSkillExecutionIntent("kubernetes", map[string]string{"pod": "pending"}, SkillExecutionIntent{})
	if got.NodePath != "ops.incident_diagnosis.kubernetes.workload.pod_pending" {
		t.Fatalf("unexpected node path: %+v", got)
	}
	if got.SkillKey != "skill.k8s.workload.pod_pending" || got.PackKey != "skillpack.k8s" {
		t.Fatalf("unexpected intent coordinates: %+v", got)
	}
}

func TestNormalizeSkillExecutionIntentGoRuntimeK8sWorkload(t *testing.T) {
	got := NormalizeSkillExecutionIntent("go-runtime", map[string]string{"pod": "prod/api-0"}, SkillExecutionIntent{})
	if got.NodePath != "ops.incident_diagnosis.application.go_runtime.k8s_workload" {
		t.Fatalf("unexpected node path: %+v", got)
	}
	if got.PackKey != "pack.runtime_observe" || got.ExecutionMode != ExecutionModeServerPlanReadonly {
		t.Fatalf("unexpected go runtime policy: %+v", got)
	}
}

func TestNormalizeSkillExecutionIntentHonorsKnownCandidate(t *testing.T) {
	got := NormalizeSkillExecutionIntent("k8s", nil, SkillExecutionIntent{
		CandidateNodePath: "ops.delivery_implementation.kubernetes.preflight",
	})
	if got.ProblemKey != "preflight" || got.CapabilityKey != "cap.delivery.k8s" {
		t.Fatalf("candidate path not honored: %+v", got)
	}
}

func TestApplySkillTreeAssetStatsRollsUpToParents(t *testing.T) {
	nodes := SkillTreeNodes()
	applySkillTreeAssetStats(nodes, []models.SkillAsset{
		{
			Status:       models.SkillAssetStatusReview,
			Topic:        "k8s",
			SkillKey:     "skill.k8s.workload.crashloop",
			ProblemKey:   "crashloop",
			CategoryPath: "ops.incident_diagnosis.kubernetes.workload.crashloop",
		},
		{
			Status:        models.SkillAssetStatusApproved,
			Topic:         "redis",
			ProblemKey:    "latency",
			CapabilityKey: "cap.diagnosis.redis",
		},
	})
	stats := map[string]SkillTreeAssetStats{}
	for _, n := range nodes {
		if n.AssetStats != nil {
			stats[n.Path] = *n.AssetStats
		}
	}
	if got := stats["ops.incident_diagnosis.kubernetes.workload"].Review; got != 1 {
		t.Fatalf("expected k8s workload review rollup=1, got %d", got)
	}
	if got := stats["ops.incident_diagnosis"].Total; got != 2 {
		t.Fatalf("expected incident diagnosis total rollup=2, got %d", got)
	}
	if got := stats["ops.incident_diagnosis.middleware.redis"].Approved; got != 1 {
		t.Fatalf("expected redis approved rollup=1, got %d", got)
	}
}
