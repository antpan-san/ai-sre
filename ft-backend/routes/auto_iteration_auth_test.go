package routes_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"ft-backend/common/config"
	"ft-backend/common/logger"
	"ft-backend/common/response"
	"ft-backend/database"
	"ft-backend/handlers"
	"ft-backend/middleware"
	"ft-backend/models"
	"ft-backend/services"
	"ft-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testRouter(secret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api")
	api.Use(middleware.JWTAuth(secret))
	super := api.Group("")
	super.Use(middleware.RequireSuperAdmin())
	super.GET("/admin/auto-iterations", handlers.AdminListAutoIterations)
	super.PUT("/admin/auto-iterations/settings", handlers.AdminUpdateAutoIterationSettings)
	super.GET("/admin/auto-iterations/:id/events/stream", handlers.AdminStreamAutoIterationEvents)
	super.POST("/admin/auto-iterations/:id/approve", handlers.AdminApproveAutoIteration)

	pub := r.Group("/api")
	pub.POST("/cli/feedback/analyze", handlers.PostCLIFeedbackAnalyze)
	return r
}

func setupDB(t *testing.T) {
	t.Helper()
	logger.InitLogger("error", io.Discard)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	database.DB = db
	stmts := []string{
		`CREATE TABLE auto_iterations (id TEXT PRIMARY KEY, tenant_id TEXT, created_at DATETIME, updated_at DATETIME,
			title TEXT, description TEXT, status TEXT, source TEXT, risk_level TEXT,
			requires_super_admin_approval INTEGER, topic TEXT, command TEXT, summary TEXT,
			feedback_id TEXT, created_by_user_id TEXT, created_by TEXT, approved_by_user_id TEXT, approved_by TEXT,
			approved_at DATETIME, assigned_agent_id TEXT, last_error TEXT, metadata TEXT)`,
		`CREATE TABLE auto_iteration_events (id TEXT PRIMARY KEY, tenant_id TEXT, created_at DATETIME, updated_at DATETIME,
			auto_iteration_id TEXT, event_type TEXT, actor_type TEXT, actor_name TEXT, message TEXT, payload TEXT)`,
		`CREATE TABLE auto_iteration_settings (id INTEGER PRIMARY KEY, enabled INTEGER, max_concurrent INTEGER,
			high_risk_requires_approval INTEGER, notes TEXT, updated_at DATETIME, updated_by TEXT)`,
		`CREATE TABLE auto_iteration_feedbacks (id TEXT PRIMARY KEY, tenant_id TEXT, created_at DATETIME, updated_at DATETIME,
			user_id TEXT, cli_binding_id TEXT, topic TEXT, classification TEXT, need_iteration INTEGER,
			user_message TEXT, raw_payload TEXT, auto_iteration_id TEXT)`,
		`CREATE TABLE operation_logs (id TEXT PRIMARY KEY, tenant_id TEXT, username TEXT, operation TEXT,
			resource TEXT, resource_id TEXT, ip TEXT, user_agent TEXT, status TEXT, error_message TEXT,
			details TEXT, created_at DATETIME)`,
		`CREATE TABLE cli_bindings (id TEXT PRIMARY KEY, tenant_id TEXT, created_at DATETIME, updated_at DATETIME,
			user_id TEXT, username TEXT, token_hash TEXT, fingerprint_hash TEXT, expires_at DATETIME)`,
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			t.Fatal(err)
		}
	}
	_ = db.Exec(`INSERT INTO auto_iteration_settings (id, enabled, max_concurrent, high_risk_requires_approval, updated_at)
		VALUES (1, 0, 2, 1, datetime('now'))`).Error
}

func setupAuthTest(t *testing.T) (*gin.Engine, string, string, string) {
	t.Helper()
	setupDB(t)
	secret := "test-secret-auto-iteration"
	config.GlobalCfg = &config.Config{JWT: config.JWTConfig{SecretKey: secret, AccessTokenExp: 60}}
	router := testRouter(secret)
	superToken, _ := utils.GenerateAccessToken(uuid.NewString(), "super", "s@test", models.RoleSuperAdmin, secret, 60)
	adminToken, _ := utils.GenerateAccessToken(uuid.NewString(), "admin", "a@test", models.RoleAdmin, secret, 60)
	userToken, _ := utils.GenerateAccessToken(uuid.NewString(), "user", "u@test", models.RoleUser, secret, 60)
	return router, superToken, adminToken, userToken
}

func bearer(token string) string { return "Bearer " + token }

func TestAutoIterationAdminAuthMatrix(t *testing.T) {
	router, superTok, adminTok, userTok := setupAuthTest(t)
	cases := []struct {
		name string
		tok  string
		want int
	}{
		{"no_auth", "", http.StatusUnauthorized},
		{"user", userTok, http.StatusForbidden},
		{"admin", adminTok, http.StatusForbidden},
		{"super_admin", superTok, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/admin/auto-iterations", nil)
			if tc.tok != "" {
				req.Header.Set("Authorization", bearer(tc.tok))
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("status=%d body=%s want %d", w.Code, w.Body.String(), tc.want)
			}
		})
	}
}

func TestUserDeniedOnSSE(t *testing.T) {
	router, _, _, userTok := setupAuthTest(t)
	id := uuid.New()
	_ = database.DB.Exec(`INSERT INTO auto_iterations (id, title, status, source, risk_level, requires_super_admin_approval, metadata, created_at, updated_at)
		VALUES (?, 't', 'draft', 'manual', 'low', 0, '{}', datetime('now'), datetime('now'))`, id.String())
	req := httptest.NewRequest(http.MethodGet, "/api/admin/auto-iterations/"+id.String()+"/events/stream", nil)
	req.Header.Set("Authorization", bearer(userTok))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("sse status=%d want 403", w.Code)
	}
}

func TestCLIFeedbackAnalyzePublicFieldsOnly(t *testing.T) {
	_, _, _, _ = setupAuthTest(t)
	_ = database.DB.Exec(`UPDATE auto_iteration_settings SET enabled = 1 WHERE id = 1`)
	router := gin.New()
	router.POST("/api/cli/feedback/analyze", handlers.PostCLIFeedbackAnalyze)

	cliTok := "cli-feedback-token-0123456789abcdef0123456789ab"
	fp := "112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	uid := uuid.New().String()
	_ = database.DB.Exec(`INSERT INTO cli_bindings (id, user_id, username, token_hash, fingerprint_hash, expires_at, created_at, updated_at)
		VALUES (?, ?, 'cli', ?, ?, datetime('now','+1 day'), datetime('now'), datetime('now'))`,
		uuid.NewString(), uid, hashCLI(cliTok), hashCLI(fp))

	body, _ := json.Marshal(map[string]interface{}{"topic": "k8s", "summary": "bug: crashloop"})
	req := httptest.NewRequest(http.MethodPost, "/api/cli/feedback/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearer(cliTok))
	req.Header.Set("X-OpsFleet-CLI-Fingerprint", fp)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var envelope response.R
	_ = json.Unmarshal(w.Body.Bytes(), &envelope)
	data, _ := envelope.Data.(map[string]interface{})
	allowed := map[string]bool{"feedback_id": true, "classification": true, "need_iteration": true, "user_message": true, "next_action": true}
	for k := range data {
		if !allowed[k] {
			t.Fatalf("unexpected key %q", k)
		}
	}
}

func hashCLI(s string) string { return services.HashSecretForAgent(s) }

func TestCLIAndAgentDeniedOnAdminList(t *testing.T) {
	router, _, _, _ := setupAuthTest(t)
	cliTok := "cli-feedback-token-0123456789abcdef0123456789ab"
	fp := "112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	_ = database.DB.Exec(`INSERT INTO cli_bindings (id, user_id, username, token_hash, fingerprint_hash, expires_at, created_at, updated_at)
		VALUES (?, ?, 'cli', ?, ?, datetime('now','+1 day'), datetime('now'), datetime('now'))`,
		uuid.NewString(), uuid.NewString(), hashCLI(cliTok), hashCLI(fp))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/auto-iterations", nil)
	req.Header.Set("Authorization", bearer(cliTok))
	req.Header.Set("X-OpsFleet-CLI-Fingerprint", fp)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized && w.Code != http.StatusForbidden {
		t.Fatalf("cli admin list status=%d want 401/403", w.Code)
	}

	agentTok := "agent-test-token-0123456789abcdef0123456789ab"
	agentFP := "aabbccddeeff00112233445566778899aabbccddeeff001122334455667788"
	req2 := httptest.NewRequest(http.MethodGet, "/api/admin/auto-iterations", nil)
	req2.Header.Set("Authorization", bearer(agentTok))
	req2.Header.Set("X-OpsFleet-Agent-Fingerprint", agentFP)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusUnauthorized && w2.Code != http.StatusForbidden {
		t.Fatalf("agent admin list status=%d want 401/403", w2.Code)
	}
}

func TestSuperAdminApprovesHighRiskFixture(t *testing.T) {
	router, superTok, _, _ := setupAuthTest(t)
	id := uuid.New()
	_ = database.DB.Exec(`INSERT INTO auto_iterations (id, title, status, source, risk_level, requires_super_admin_approval, metadata, created_at, updated_at)
		VALUES (?, 'high', 'awaiting_approval', 'manual', 'high', 1, '{}', datetime('now'), datetime('now'))`, id.String())
	req := httptest.NewRequest(http.MethodPost, "/api/admin/auto-iterations/"+id.String()+"/approve", bytes.NewReader([]byte(`{"notes":"ok"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearer(superTok))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("approve status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestAgentDeniedOnAdminApprove(t *testing.T) {
	router, _, _, _ := setupAuthTest(t)
	id := uuid.New()
	_ = database.DB.Exec(`INSERT INTO auto_iterations (id, title, status, source, risk_level, requires_super_admin_approval, metadata, created_at, updated_at)
		VALUES (?, 't', 'awaiting_approval', 'manual', 'high', 1, '{}', datetime('now'), datetime('now'))`, id.String())
	agentTok := "agent-test-token-0123456789abcdef0123456789ab"
	agentFP := "aabbccddeeff00112233445566778899aabbccddeeff001122334455667788"
	req := httptest.NewRequest(http.MethodPost, "/api/admin/auto-iterations/"+id.String()+"/approve", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Authorization", bearer(agentTok))
	req.Header.Set("X-OpsFleet-Agent-Fingerprint", agentFP)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized && w.Code != http.StatusForbidden && w.Code != http.StatusNotFound {
		t.Fatalf("agent approve status=%d want 401/403/404", w.Code)
	}
}

func TestSuperAdminSettingsWritesAudit(t *testing.T) {
	router, superTok, _, _ := setupAuthTest(t)
	req := httptest.NewRequest(http.MethodPut, "/api/admin/auto-iterations/settings", bytes.NewReader([]byte(`{"enabled":true}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearer(superTok))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d %s", w.Code, w.Body.String())
	}
	var count int64
	database.DB.Table("operation_logs").Where("operation = ?", "auto_iteration.settings.update").Count(&count)
	if count < 1 {
		t.Fatal("expected audit log")
	}
}
