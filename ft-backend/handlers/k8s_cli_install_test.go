package handlers

import (
	"os"
	"testing"
)

func TestReadELFArchExecutable(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Skip(err)
	}
	a, err := readELFArch(exe)
	if err != nil {
		t.Skip("skip: test binary not Linux ELF", err)
	}
	if a != "amd64" && a != "arm64" {
		t.Fatalf("unexpected arch %q", a)
	}
}
