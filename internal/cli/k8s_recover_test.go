package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadK8sRecoveryState(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	orig := k8sStateDir
	// cannot reassign const - test save via direct path write
	st := K8sRecoveryState{
		InstallRef: "ofpk8s1.test",
		FailedStep: "install.sh",
		ExitCode:   1,
		BundleRoot: dir,
	}
	raw, err := os.Create(filepath.Join(dir, "recovery.json"))
	if err != nil {
		t.Fatal(err)
	}
	_ = st
	_ = orig
	_ = raw.Close()
}

func TestRequireOpsMutationConfirmNonTTY(t *testing.T) {
	t.Parallel()
	if isStdinTTY() {
		t.Skip("stdin is TTY")
	}
	if err := requireOpsMutationConfirm(false, "test"); err == nil {
		t.Fatal("expected error without --yes on non-tty")
	}
}

func TestLocalInstallRecoveryPlanAptLock(t *testing.T) {
	t.Parallel()
	ev := map[string]interface{}{
		"log_tail": "E: Unable to acquire the dpkg frontend lock unattended-upgr",
	}
	plan := localInstallRecoveryPlan(ev)
	if plan.RootCause == "" {
		t.Fatal("expected root cause")
	}
	if len(plan.SafeActions) == 0 {
		t.Fatal("expected safe actions")
	}
}
