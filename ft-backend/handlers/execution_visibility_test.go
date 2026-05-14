package handlers

import "testing"

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
		{"install_ai_sre", true},
		{"k8s_install", false},
	}
	for _, tc := range cases {
		if got := executionCategoryVisibleToOwner(tc.cat); got != tc.ok {
			t.Fatalf("category %q: want %v got %v", tc.cat, tc.ok, got)
		}
	}
}
