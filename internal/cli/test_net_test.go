package cli

import (
	"errors"
	"net"
	"os"
	"strings"
	"testing"
)

func requireLocalTCPListen(t *testing.T) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err == nil {
		_ = ln.Close()
		return
	}
	if errors.Is(err, os.ErrPermission) || strings.Contains(strings.ToLower(err.Error()), "operation not permitted") {
		t.Skipf("local TCP listen is not permitted in this sandbox: %v", err)
	}
	t.Fatalf("local TCP listen failed: %v", err)
}
