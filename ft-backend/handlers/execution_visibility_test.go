package handlers

import (
	"testing"

	"ft-backend/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestExecutionCategoryUsesAICapability(t *testing.T) {
	cases := []struct {
		cat string
		ok  bool
	}{
		{"analyze", true},
		{"ask", true},
		{"runbook", true},
		{"skills", true},
		{"doctor", true},
		{"elasticsearch", true},
		{"go_runtime", true},
		{"k8s_install", false},
		{"install_ai_sre", false},
		{"version", false},
		{"upgrade", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := executionCategoryUsesAICapability(tc.cat); got != tc.ok {
			t.Fatalf("category %q: want %v got %v", tc.cat, tc.ok, got)
		}
	}
}

func TestExecutionCategoryVisibleToOwner(t *testing.T) {
	cases := []struct {
		cat string
		ok  bool
	}{
		{"analyze", true},
		{"ask_nginx", true},
		{"go_runtime", true},
		{"install_ai_sre", true},
		{"k8s_install", false},
	}
	for _, tc := range cases {
		if got := executionCategoryVisibleToOwner(tc.cat); got != tc.ok {
			t.Fatalf("category %q: want %v got %v", tc.cat, tc.ok, got)
		}
	}
}

func TestApplyExecutionConsoleMemberScopeSQL(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	var total int64
	tx := applyExecutionConsoleMemberScope(db.Model(&models.ExecutionRecord{}), "user", "testuser").Count(&total)
	if tx.Error != nil {
		t.Fatalf("member scope count dry-run: %v", tx.Error)
	}
	if tx.Statement.SQL.String() == "" {
		t.Fatal("expected generated SQL")
	}
}
