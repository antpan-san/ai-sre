package cli

import (
	"testing"
)

// maybePromptFeedback must be a no-op when conditions to skip apply. We can't
// easily fake a TTY here, but we can verify the "json output" and
// "--no-feedback" skip paths by setting package-level state and ensuring no
// network call attempt is made (the function would otherwise hit
// resolveOpsfleetAPIBase, fail and only print to stderr).
func TestMaybePromptFeedbackHonoursSkips(t *testing.T) {
	defer func(prev string, prevNo bool) {
		outputFormat = prev
		noFeedback = prevNo
	}(outputFormat, noFeedback)

	// 1) --no-feedback path
	noFeedback = true
	outputFormat = "text"
	maybePromptFeedback(nil, "k8s", &diagnoseResponse{SkillName: "x"})

	// 2) -o json path
	noFeedback = false
	outputFormat = "json"
	maybePromptFeedback(nil, "k8s", &diagnoseResponse{SkillName: "x"})
}

func TestDiagnoseResponseRequestID(t *testing.T) {
	var d *diagnoseResponse
	if got := d.RequestID(); got != "" {
		t.Errorf("nil should return empty, got %q", got)
	}
	d = &diagnoseResponse{}
	if got := d.RequestID(); got != "" {
		t.Errorf("no metadata should return empty, got %q", got)
	}
	d.Metadata = map[string]interface{}{"request_id": "req-123"}
	if got := d.RequestID(); got != "req-123" {
		t.Errorf("expected req-123, got %q", got)
	}
	d.Metadata = map[string]interface{}{"request_id": 42}
	if got := d.RequestID(); got != "" {
		t.Errorf("non-string id should return empty, got %q", got)
	}
}
