package cli

import (
	"strings"
	"testing"
)

func TestBuildExecutionIntentPostgreSQLGeneral(t *testing.T) {
	got := buildExecutionIntent("analyze", "postgresql", map[string]string{})
	if got.NodePath != "ops.incident_diagnosis.middleware.postgresql.general" {
		t.Fatalf("unexpected node path: %+v", got)
	}
	if got.SkillKey != "skill.postgresql.general" || got.PackKey != "skillpack.postgresql" {
		t.Fatalf("unexpected intent: %+v", got)
	}
}

func TestBuildExecutionIntentPostgresAlias(t *testing.T) {
	got := buildExecutionIntent("analyze", "postgres", nil)
	if got.Topic != "postgresql" {
		t.Fatalf("expected topic postgresql, got %q", got.Topic)
	}
}

func TestMaskPostgreSQLDSN(t *testing.T) {
	in := "postgres://admin:secret@127.0.0.1:5432/app?sslmode=disable"
	got := maskPostgreSQLDSN(in)
	if strings.Contains(got, "secret") {
		t.Fatalf("password not masked: %q", got)
	}
}
