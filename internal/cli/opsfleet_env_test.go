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

func TestResolveOpsfleetAPIBaseAutoBindsOnCrossEnvironment(t *testing.T) {
	t.Setenv("OPSFLEET_SKIP_REMOTE", "")
	t.Setenv("OPSFLEET_API_URL", EmbeddedOpsfleetAPIBaseProduction)
	autoBindingWarn = ""
	autoBindingWarnShown = false
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	cfg := filepath.Join(dir, ".config", "ai-sre")
	if err := os.MkdirAll(cfg, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "opsfleet_api_url"), []byte(EmbeddedOpsfleetAPIBase+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	base, err := resolveOpsfleetAPIBaseStrict()
	if err != nil {
		t.Fatalf("want auto-bind without error, got %v", err)
	}
	if base != EmbeddedOpsfleetAPIBase {
		t.Fatalf("want install lab base, got %q", base)
	}
	if !strings.Contains(autoBindingWarn, "已自动采用 install") {
		t.Fatalf("want auto-bind warn, got %q", autoBindingWarn)
	}
	if got := smartDefaultCheckTarget("redis"); got != "127.0.0.1:6379" {
		t.Fatalf("smart redis default prefers localhost, got %q", got)
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
