package cli

import "testing"

func TestMaybePromptFeedbackIsNoOp(t *testing.T) {
	maybePromptFeedback(nil, "domain", &diagnoseResponse{SkillName: "x"})
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
