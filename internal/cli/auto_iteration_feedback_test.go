package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestCallCLIFeedbackAnalyzeUsesBindingAuth(t *testing.T) {
	const tok = "test-cli-token-012345678901234567890123456"
	const fp = "112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/cli/feedback/analyze" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+tok {
			t.Errorf("auth=%q", got)
		}
		if got := r.Header.Get("X-OpsFleet-CLI-Fingerprint"); got != fp {
			t.Errorf("fp=%q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 200,
			"data": map[string]interface{}{
				"feedback_id":    "fb-1",
				"classification": "bug",
				"need_iteration": true,
				"user_message":   "queued",
				"next_action":    "wait",
			},
		})
	}))
	defer srv.Close()

	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))
	t.Setenv("OPSFLEET_API_URL", srv.URL)
	t.Setenv("OPSFLEET_TOKEN", tok)
	t.Setenv("OPSFLEET_CLI_FINGERPRINT", fp)
	t.Setenv("OPSFLEET_SKIP_REMOTE", "")

	out, err := callCLIFeedbackAnalyze(context.Background(), "k8s", "analyze k8s", "crashloop", nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.FeedbackID != "fb-1" || !out.NeedIteration {
		t.Fatalf("unexpected %+v", out)
	}
}
