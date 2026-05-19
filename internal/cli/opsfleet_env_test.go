package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveOpsfleetAPIBasesSingleEnvironment(t *testing.T) {
	t.Setenv("OPSFLEET_SKIP_REMOTE", "")
	t.Setenv("OPSFLEET_API_URL", "")
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	cfg := filepath.Join(dir, ".config", "ai-sre")
	if err := os.MkdirAll(cfg, 0o700); err != nil {
		t.Fatal(err)
	}
	bases := resolveOpsfleetAPIBasesForUpgrade()
	if len(bases) != 1 {
		t.Fatalf("want single base, got %v", bases)
	}
	if bases[0] != EmbeddedOpsfleetAPIBase {
		t.Fatalf("default lab base=%q", bases[0])
	}
	if classifyOpsfleetBase(bases[0]) != opsfleetEnvLab {
		t.Fatalf("want lab env")
	}
}

func TestResolveOpsfleetAPIBaseRejectsCrossEnvironment(t *testing.T) {
	t.Setenv("OPSFLEET_SKIP_REMOTE", "")
	t.Setenv("OPSFLEET_API_URL", EmbeddedOpsfleetAPIBaseProduction)
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	cfg := filepath.Join(dir, ".config", "ai-sre")
	if err := os.MkdirAll(cfg, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "opsfleet_api_url"), []byte(EmbeddedOpsfleetAPIBase+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := resolveOpsfleetAPIBaseStrict()
	if err == nil || !strings.Contains(err.Error(), "禁止混用") {
		t.Fatalf("want cross-env error, got %v", err)
	}
	base, warn := resolveOpsfleetAPIBaseForUpgrade()
	if base != EmbeddedOpsfleetAPIBase {
		t.Fatalf("upgrade probe should prefer install file, got %q", base)
	}
	if warn == "" || !strings.Contains(warn, "禁止混用") {
		t.Fatalf("want upgrade warn, got %q", warn)
	}
	bases := resolveOpsfleetAPIBasesForUpgrade()
	if len(bases) != 1 || bases[0] != EmbeddedOpsfleetAPIBase {
		t.Fatalf("upgrade bases=%v", bases)
	}
}

func TestOpsfleetAPIBasesEquivalent(t *testing.T) {
	a := "http://192.168.56.11:9080/ft-api"
	b := "http://192.168.56.11/ft-api"
	if !opsfleetAPIBasesEquivalent(a, b) {
		t.Fatal("same lab host should match")
	}
	if opsfleetAPIBasesEquivalent(a, EmbeddedOpsfleetAPIBaseProduction) {
		t.Fatal("lab vs production must not match")
	}
}
