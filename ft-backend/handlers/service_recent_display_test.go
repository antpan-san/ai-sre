package handlers

import (
	"strings"
	"testing"

	"ft-backend/models"
)

func TestServiceProductNameFromConfig(t *testing.T) {
	s := &models.Service{
		Config: models.JSONB(`{"service_key":"nginx"}`),
		Type:   "linux",
	}
	if got := serviceProductNameForDashboard(s); got != "nginx" {
		t.Fatalf("want nginx got %q", got)
	}
}

func TestServiceProductNameDescriptionFallback(t *testing.T) {
	s := &models.Service{
		Config:      models.JSONB(`{}`),
		Description: "生产网关",
		Type:        "docker",
	}
	if got := serviceProductNameForDashboard(s); got != "生产网关" {
		t.Fatalf("want 生产网关 got %q", got)
	}
}

func TestServiceResourceSummary(t *testing.T) {
	s := &models.Service{
		Image:     "registry.io/app:1.0",
		MachineID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Port:      8080,
	}
	got := serviceResourceSummaryForDashboard(s)
	for _, sub := range []string{"registry.io/app:1.0", "主机", "8080"} {
		if !strings.Contains(got, sub) {
			t.Fatalf("expected %q in %q", sub, got)
		}
	}
}
