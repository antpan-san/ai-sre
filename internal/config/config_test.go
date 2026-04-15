package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadKeyFileTrimsQuotes(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "k")
	if err := os.WriteFile(p, []byte(`"sk-test-key-123"`), 0o600); err != nil {
		t.Fatal(err)
	}
	k, err := loadKeyFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if k != "sk-test-key-123" {
		t.Fatalf("got %q", k)
	}
}
