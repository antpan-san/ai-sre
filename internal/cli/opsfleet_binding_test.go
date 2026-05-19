package cli

import (
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
	if !strings.Contains(msg, "api_key") || !strings.Contains(msg, "opsfleet_token") {
		t.Fatalf("message=%q", msg)
	}
	_ = oldHome
}

func TestValidateOpsfleetCredentialsRequiresToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("OPSFLEET_API_URL", "http://example/ft-api")
	if err := validateOpsfleetCredentials(); err == nil {
		t.Fatal("expected error without token")
	}
}
