package cli

import "testing"

func TestK8sAnalyzePodFlagIsIssueKeyword(t *testing.T) {
	for _, s := range []string{"", "pending", "PENDING", "crashloop", "CrashLoopBackOff", "instability"} {
		if !k8sAnalyzePodFlagIsIssueKeyword(s) {
			t.Fatalf("expected issue keyword: %q", s)
		}
	}
	if k8sAnalyzePodFlagIsIssueKeyword("kube-controller-manager-k8s-master-0") {
		t.Fatal("real pod name should not be issue keyword")
	}
}
