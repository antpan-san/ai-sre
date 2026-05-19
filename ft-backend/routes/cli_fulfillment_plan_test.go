package routes_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ft-backend/database"
	"ft-backend/handlers"
	"ft-backend/models"
	"ft-backend/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func cliFulfillmentRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/cli/fulfillment/plan", handlers.PostCLIFulfillmentPlan)
	return r
}

func insertCLIBinding(t *testing.T, token, fp, uid string) {
	t.Helper()
	_ = database.DB.Exec(`INSERT INTO users (id, username, role) VALUES (?, 'cliuser', 'user')`, uid)
	_ = database.DB.Exec(`INSERT INTO cli_bindings (id, user_id, username, token_hash, fingerprint_hash, expires_at, created_at, updated_at)
		VALUES (?, ?, 'cli', ?, ?, datetime('now','+1 day'), datetime('now'), datetime('now'))`,
		uuid.NewString(), uid, hashCLI(token), strings.ToLower(fp))
}

func TestFulfillmentPlanRequiresDigest(t *testing.T) {
	setupDB(t)
	router := cliFulfillmentRouter()
	token := "cli-fulfillment-token-0123456789abcdef0123456789ab"
	fp := "112233445566778899aabbccddeeff00112233445566778899aabbccddeeff00"
	insertCLIBinding(t, token, fp, uuid.NewString())

	body, _ := json.Marshal(map[string]interface{}{
		"topic":  "postgresql",
		"intent": map[string]interface{}{"topic": "postgresql"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/cli/fulfillment/plan", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearer(token))
	req.Header.Set("X-OpsFleet-CLI-Fingerprint", fp)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestFulfillmentPlanPublicFieldsOnly(t *testing.T) {
	setupDB(t)
	_ = database.DB.Exec(`UPDATE auto_iteration_settings SET enabled = 0, auto_dispatch_enabled = 0 WHERE id = 1`)
	router := cliFulfillmentRouter()
	token := "cli-fulfillment-token-0123456789abcdef0123456789ab"
	fp := "112233445566778899aabbccddeeff00112233445566778899aabbccddeeff00"
	insertCLIBinding(t, token, fp, uuid.NewString())

	body, _ := json.Marshal(map[string]interface{}{
		"command_catalog_digest": "abc123",
		"topic":                  "postgresql",
		"failure_kind":           "unsupported",
		"intent":                 map[string]interface{}{"topic": "postgresql"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/cli/fulfillment/plan", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearer(token))
	req.Header.Set("X-OpsFleet-CLI-Fingerprint", fp)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var env struct {
		Data map[string]interface{} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	for k := range env.Data {
		switch k {
		case "action", "message", "auto_iteration_created", "auto_iteration_id", "retry_allowed":
		default:
			t.Fatalf("unexpected field %q", k)
		}
	}
}

func TestFulfillmentHighRiskCreatesAwaitingApproval(t *testing.T) {
	setupDB(t)
	_ = database.DB.Exec(`UPDATE auto_iteration_settings SET enabled = 1, auto_dispatch_enabled = 1, high_risk_requires_approval = 1 WHERE id = 1`)
	uid := uuid.New()
	plan, err := services.HandleCLIFulfillmentPlan(uid, "cli", "digest", "ai-sre deploy", "billing", "product_gap", "needs db migration auth", nil, services.SkillExecutionIntent{Topic: "billing"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Action != services.FulfillmentActionAwaitingApproval {
		t.Fatalf("action=%s", plan.Action)
	}
	var row models.AutoIteration
	if err := database.DB.Where("id = ?", plan.AutoIterationID).First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != models.AutoIterationStatusAwaitingApproval {
		t.Fatalf("status=%s", row.Status)
	}
}
