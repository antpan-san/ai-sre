package services

import "testing"

func TestAnalyzeInstallRecoverySSHFail(t *testing.T) {
	t.Parallel()
	plan := AnalyzeInstallRecovery("k8s", "install_recovery", "ops k8s recover", map[string]interface{}{
		"ssh_preflight": map[string]interface{}{"status": "fail"},
	})
	if plan.RootCause == "" {
		t.Fatal("expected root cause")
	}
	if !plan.NeedIteration {
		t.Fatal("expected need_iteration for ssh fail")
	}
}
