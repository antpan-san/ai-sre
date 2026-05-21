package cli

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckGoRuntimeAuthSendsCLICredentials(t *testing.T) {
	requireLocalTCPListen(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/cli/go-runtime/auth-check" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Fatalf("Authorization=%q", got)
		}
		if got := r.Header.Get("X-OpsFleet-CLI-Fingerprint"); got != "fp" {
			t.Fatalf("fingerprint=%q", got)
		}
		if got := r.Header.Get("X-OpsFleet-CLI-Version"); strings.TrimSpace(got) == "" {
			t.Fatalf("missing CLI version header")
		}
		fmt.Fprint(w, `{"code":200,"msg":"success","data":{}}`)
	}))
	defer srv.Close()

	if err := checkGoRuntimeAuth(context.Background(), srv.URL, "tok", "fp"); err != nil {
		t.Fatal(err)
	}
}

func TestCheckGoRuntimeAuthReturnsServerFailure(t *testing.T) {
	requireLocalTCPListen(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"code":401,"msg":"fingerprint mismatch"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	err := checkGoRuntimeAuth(context.Background(), srv.URL, "tok", "fp")
	if err == nil || !strings.Contains(err.Error(), "HTTP 401") {
		t.Fatalf("err=%v", err)
	}
}
