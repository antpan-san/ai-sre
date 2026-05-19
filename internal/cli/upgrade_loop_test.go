package cli

import "testing"

func TestAppendAttemptUpgradeEnv(t *testing.T) {
	env := appendAttemptUpgradeEnv([]string{"HOME=/tmp", "OPSFLEET_AUTO_UPGRADE_ATTEMPT=0"})
	if len(env) != 2 {
		t.Fatalf("len=%d want 2", len(env))
	}
	if env[1] != "OPSFLEET_AUTO_UPGRADE_ATTEMPT=1" {
		t.Fatalf("got %q", env[1])
	}
}

func TestVersionIsOlderAfterUpgradeMismatch(t *testing.T) {
	// Simulates server claiming 0.5.25 while binary stays 0.5.24.
	if !versionIsOlder("0.5.24", "0.5.25") {
		t.Fatal("expected older")
	}
	if versionIsOlder("0.5.25", "0.5.25") {
		t.Fatal("equal should not be older")
	}
}
