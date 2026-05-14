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
