package services

import (
	"testing"

	"ft-backend/models"
)

func TestBindingCoversPathSubtree(t *testing.T) {
	b := models.SkillProductNodeBinding{
		NodePath:   "ops.incident_diagnosis.kubernetes",
		GrantScope: models.ProductGrantScopeSubtree,
	}
	if !bindingCoversPath(b, "ops.incident_diagnosis.kubernetes.workload.pod_pending") {
		t.Fatalf("expected subtree cover")
	}
	if bindingCoversPath(b, "ops.delivery_implementation.kubernetes") {
		t.Fatalf("expected no cover for sibling branch")
	}
}

func TestBindingCoversPathNode(t *testing.T) {
	b := models.SkillProductNodeBinding{
		NodePath:   "ops.incident_diagnosis.middleware.kafka.lag",
		GrantScope: models.ProductGrantScopeNode,
	}
	if !bindingCoversPath(b, "ops.incident_diagnosis.middleware.kafka.lag") {
		t.Fatalf("expected exact node")
	}
	if bindingCoversPath(b, "ops.incident_diagnosis.middleware.kafka") {
		t.Fatalf("expected no parent cover")
	}
}

func TestFallbackCommercialMatch(t *testing.T) {
	m, ok := fallbackCommercialMatch("ops.incident_diagnosis.kubernetes.workload.pod_pending")
	if !ok || m.ProductKey != models.SkillPackK8s {
		t.Fatalf("unexpected match: %+v ok=%v", m, ok)
	}
}
