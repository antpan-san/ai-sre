package cli

import "testing"

func TestResolveOpsfleetAPIBasesForUpgradeNoProductionFallback(t *testing.T) {
	t.Setenv("OPSFLEET_SKIP_REMOTE", "")
	t.Setenv("OPSFLEET_API_URL", "")
	bases := resolveOpsfleetAPIBasesForUpgrade()
	if len(bases) != 1 {
		t.Fatalf("want exactly one base (lab default), got %v", bases)
	}
	if bases[0] == EmbeddedOpsfleetAPIBaseProduction {
		t.Fatal("must not auto-include production in upgrade chain")
	}
}
