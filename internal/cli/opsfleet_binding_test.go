package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatMissingOpsfleetTokenMentionsAPIKey(t *testing.T) {
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	cfg := filepath.Join(dir, ".config", "ai-sre")
	if err := os.MkdirAll(cfg, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "api_key"), []byte("sk-test"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := formatMissingOpsfleetTokenError("http://192.168.56.11:9080/ft-api")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "api_key") || !strings.Contains(msg, "install-ai-sre") {
		t.Fatalf("message=%q", msg)
	}
	_ = oldHome
}

func TestFormatOpsfleetAPIErrorIdempotent(t *testing.T) {
	raw := fmt.Errorf("cli sync status=401: CLI token 无效")
	once := formatOpsfleetAPIError(raw, "/api/cli/sync")
	twice := formatOpsfleetAPIError(once, "/api/cli/sync")
	if once.Error() != twice.Error() {
		t.Fatalf("formatted twice:\n1=%q\n2=%q", once, twice)
	}
	if strings.Count(once.Error(), "install-ai-sre") != 1 {
		t.Fatalf("want single hint: %q", once)
	}
}

func TestValidateOpsfleetCredentialsRequiresToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("OPSFLEET_API_URL", "http://example/ft-api")
	if err := validateOpsfleetCredentials(); err == nil {
		t.Fatal("expected error without token")
	}
}
