package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAllowedReadonlyDiagnosticCommand(t *testing.T) {
	tests := []struct {
		name string
		argv []string
		want bool
	}{
		{name: "get pods", argv: []string{"kubectl", "get", "pods", "-A", "-o", "wide"}, want: true},
		{name: "describe pod", argv: []string{"kubectl", "describe", "pod", "-n", "prod", "api-0"}, want: true},
		{name: "logs previous", argv: []string{"kubectl", "logs", "-n", "prod", "api-0", "--all-containers=true", "--previous", "--tail=400"}, want: true},
		{name: "reject apply", argv: []string{"kubectl", "apply", "-f", "x.yaml"}, want: false},
		{name: "reject exec", argv: []string{"kubectl", "exec", "-n", "prod", "api-0", "--", "id"}, want: false},
		{name: "reject shell", argv: []string{"sh", "-c", "kubectl get pods"}, want: false},
		{name: "reject metachar", argv: []string{"kubectl", "get", "pods", "-n", "prod;rm"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := allowedReadonlyDiagnosticCommand(tt.argv); got != tt.want {
				t.Fatalf("allowedReadonlyDiagnosticCommand(%v)=%v want %v", tt.argv, got, tt.want)
			}
		})
	}
}

func TestBuildK8sReadonlyDiagnosticPlan(t *testing.T) {
	steps := buildK8sReadonlyDiagnosticPlan(map[string]string{"namespace": "prod", "pod": "api-0"})
	if len(steps) < 8 {
		t.Fatalf("expected focused plan, got %d steps", len(steps))
	}
	if steps[0].ID != "kubectl_focus_describe" {
		t.Fatalf("expected focused describe first, got %s", steps[0].ID)
	}
	for _, st := range steps {
		if !allowedReadonlyDiagnosticCommand(st.Argv) {
			t.Fatalf("generated unsafe step: %#v", st)
		}
	}
}

func TestSanitizeSkillAssetName(t *testing.T) {
	if got := sanitizeSkillAssetName("K8s / Pod CrashLoop"); got != "k8s-pod-crashloop" {
		t.Fatalf("unexpected sanitized name: %q", got)
	}
	if got := sanitizeSkillAssetName(";;;;"); got != "unknown" {
		t.Fatalf("unexpected fallback: %q", got)
	}
}

func TestCreateCLIDiagnosticPlanRequiresBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/plan", CreateCLIDiagnosticPlan)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/plan", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
