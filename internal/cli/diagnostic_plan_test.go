package cli

import "testing"

func TestAllowedCLIDiagnosticPlanCommand(t *testing.T) {
	tests := []struct {
		name string
		argv []string
		want bool
	}{
		{name: "get pods all namespaces", argv: []string{"kubectl", "get", "pods", "-A", "-o", "wide"}, want: true},
		{name: "describe pod", argv: []string{"kubectl", "describe", "pod", "-n", "prod", "api-0"}, want: true},
		{name: "logs previous", argv: []string{"kubectl", "logs", "-n", "prod", "api-0", "--all-containers=true", "--previous", "--tail=400"}, want: true},
		{name: "reject apply", argv: []string{"kubectl", "apply", "-f", "x.yaml"}, want: false},
		{name: "reject exec", argv: []string{"kubectl", "exec", "-n", "prod", "api-0", "--", "id"}, want: false},
		{name: "reject shell metachar", argv: []string{"kubectl", "get", "pods", ";", "rm"}, want: false},
		{name: "reject unsafe namespace", argv: []string{"kubectl", "get", "pods", "-n", "prod;rm", "-o", "wide"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := allowedCLIDiagnosticPlanCommand(tt.argv); got != tt.want {
				t.Fatalf("allowedCLIDiagnosticPlanCommand(%v)=%v want %v", tt.argv, got, tt.want)
			}
		})
	}
}

func TestShouldRequestServerDiagnosticPlan(t *testing.T) {
	if !shouldRequestServerDiagnosticPlan("k8s", map[string]string{}) {
		t.Fatal("expected k8s without kubectl evidence to request plan")
	}
	if shouldRequestServerDiagnosticPlan("k8s", map[string]string{"kubectl_nodes": "ok"}) {
		t.Fatal("expected local kubectl evidence to skip plan")
	}
	if shouldRequestServerDiagnosticPlan("kafka", map[string]string{}) {
		t.Fatal("expected non-k8s topic to skip plan")
	}
}
