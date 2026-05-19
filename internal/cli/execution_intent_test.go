package cli

import "testing"

func TestBuildExecutionIntentK8sCrashLoop(t *testing.T) {
	got := buildExecutionIntent("analyze", "k8s", map[string]string{"issue": "crashloop"})
	if got.NodePath != "ops.incident_diagnosis.kubernetes.workload.crashloop" {
		t.Fatalf("unexpected node path: %+v", got)
	}
	if got.SkillKey != "skill.k8s.workload.crashloop" || got.PackKey != "skillpack.k8s" {
		t.Fatalf("unexpected intent: %+v", got)
	}
}

func TestBuildExecutionIntentPostgreSQLGeneral(t *testing.T) {
	got := buildExecutionIntent("analyze", "postgresql", nil)
	if got.NodePath != "ops.incident_diagnosis.middleware.postgresql.general" {
		t.Fatalf("unexpected node path: %+v", got)
	}
	if got.SkillKey != "skill.postgresql.general" || got.PackKey != "skillpack.postgresql" || got.ProblemKey != "general" {
		t.Fatalf("unexpected intent: %+v", got)
	}
}

func TestBuildExecutionIntentGoRuntimeProcess(t *testing.T) {
	got := buildExecutionIntent("diagnose", "go-runtime", map[string]string{"pid": "1234"})
	if got.NodePath != "ops.incident_diagnosis.application.go_runtime.process" {
		t.Fatalf("unexpected node path: %+v", got)
	}
	if got.ExecutionMode != "local_ai_fallback" || got.PackKey != "pack.runtime_observe" {
		t.Fatalf("unexpected go runtime policy: %+v", got)
	}
}
