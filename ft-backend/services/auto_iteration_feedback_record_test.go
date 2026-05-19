package services

import (
	"testing"

	"ft-backend/models"

	"github.com/google/uuid"
)

func TestPopulateAutoIterationFeedbackFields(t *testing.T) {
	helpful := false
	fb := &models.AutoIterationFeedback{
		UserID: uuid.New(),
		Topic:  "redis",
	}
	populateAutoIterationFeedbackFields(fb, FeedbackSourceConsoleExec, "redis", "ai-sre check redis x", "root cause text", map[string]interface{}{
		"helpful":      helpful,
		"execution_id": "exec-1",
		"request_id":   "req-1",
		"used_ai":      true,
		"rule_hit":     false,
	})
	if fb.Source != FeedbackSourceConsoleExec {
		t.Fatalf("source=%q", fb.Source)
	}
	if fb.ExecutionID != "exec-1" || fb.RequestID != "req-1" {
		t.Fatalf("ids exec=%q req=%q", fb.ExecutionID, fb.RequestID)
	}
	if fb.Helpful == nil || *fb.Helpful != helpful {
		t.Fatalf("helpful=%v", fb.Helpful)
	}
	item := feedbackAdminItemFromModel(*fb)
	if item.Source != FeedbackSourceConsoleExec {
		t.Fatalf("item source=%q", item.Source)
	}
}
