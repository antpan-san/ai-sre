package handlers

import "testing"

func TestHeuristicRedisDiagnoseIssuesFragmentationOverreach(t *testing.T) {
	kv := map[string]string{
		"redis_diagnose_json": `{"memory":{"used_memory":"915176","mem_fragmentation_ratio":"16.02"},"clients":{"rejected_connections":"4"}}`,
	}
	answer := "根因是内存碎片率极高导致 rejected_connections"
	issues := heuristicRedisDiagnoseIssues(kv, answer)
	if len(issues) == 0 {
		t.Fatal("expected review issues for fragmentation overreach")
	}
}

func TestNormalizeDiagnosePlainText(t *testing.T) {
	in := "## 根因（一句话）\nfoo\n## 关键证据\n- bar"
	out := normalizeDiagnosePlainText(in)
	if !containsAll(out, "【根因】", "【关键证据】", "foo") {
		t.Fatalf("got %q", out)
	}
	if containsAny(out, "##", "**") {
		t.Fatalf("markdown leaked: %q", out)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if p != "" && !contains(s, p) {
			return false
		}
	}
	return true
}

func containsAny(s string, parts ...string) bool {
	for _, p := range parts {
		if contains(s, p) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && stringIndex(s, sub) >= 0)
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
