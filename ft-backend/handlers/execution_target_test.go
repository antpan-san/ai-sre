package handlers

import "testing"

func TestExecutionTargetFromAIContext(t *testing.T) {
	got := executionTargetFromAIContext(map[string]string{
		"addr": "192.168.56.11:6379",
	})
	if got != "192.168.56.11:6379" {
		t.Fatalf("got %q", got)
	}
	got = executionTargetFromAIContext(map[string]string{
		"host": "10.0.0.1",
		"port": "6379",
	})
	if got != "10.0.0.1:6379" {
		t.Fatalf("got %q", got)
	}
}
