package cli

import "testing"

func TestResolveOpsfleetAPIBasesForUpgradeIncludesProductionFallback(t *testing.T) {
	t.Setenv("OPSFLEET_SKIP_REMOTE", "")
	t.Setenv("OPSFLEET_API_URL", "")
	bases := resolveOpsfleetAPIBasesForUpgrade()
	if len(bases) < 2 {
		t.Fatalf("want at least lab+production bases, got %v", bases)
	}
	foundProd := false
	for _, b := range bases {
		if b == EmbeddedOpsfleetAPIBaseProduction {
			foundProd = true
		}
	}
	if !foundProd {
		t.Fatalf("missing production fallback in %v", bases)
	}
}
