package services

import (
	"io"
	"testing"
	"time"

	"ft-backend/common/logger"
	"ft-backend/database"
	"ft-backend/models"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAutoIterationWorkerDB(t *testing.T) {
	t.Helper()
	logger.InitLogger("error", io.Discard)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	database.DB = db
	for _, s := range []string{
		`CREATE TABLE auto_iterations (
			id TEXT PRIMARY KEY, tenant_id TEXT, created_at DATETIME, updated_at DATETIME,
			title TEXT, description TEXT, status TEXT, source TEXT, risk_level TEXT,
			requires_super_admin_approval INTEGER, topic TEXT, command TEXT, summary TEXT,
			feedback_id TEXT, created_by_user_id TEXT, created_by TEXT, approved_by_user_id TEXT,
			approved_by TEXT, approved_at DATETIME, assigned_agent_id TEXT, last_error TEXT, metadata TEXT)`,
		`CREATE TABLE auto_iteration_events (id TEXT PRIMARY KEY, tenant_id TEXT, created_at DATETIME, updated_at DATETIME,
			auto_iteration_id TEXT, event_type TEXT, actor_type TEXT, actor_name TEXT, message TEXT, payload TEXT)`,
		`CREATE TABLE auto_iteration_settings (id INTEGER PRIMARY KEY, enabled INTEGER, max_concurrent INTEGER,
			high_risk_requires_approval INTEGER, auto_dispatch_enabled INTEGER DEFAULT 1,
			low_risk_auto_deploy_enabled INTEGER DEFAULT 0, github_sync_enabled INTEGER DEFAULT 1,
			dingtalk_notify_enabled INTEGER DEFAULT 1, notes TEXT, updated_at DATETIME, updated_by TEXT)`,
	} {
		if err := db.Exec(s).Error; err != nil {
			t.Fatal(err)
		}
	}
	_ = db.Exec(`INSERT INTO auto_iteration_settings (id, enabled, max_concurrent, high_risk_requires_approval, updated_at)
		VALUES (1, 1, 2, 1, datetime('now'))`).Error
}

func insertRunningIteration(t *testing.T, title string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now().UTC()
	if err := database.DB.Exec(`INSERT INTO auto_iterations (
		id, title, description, status, source, risk_level, requires_super_admin_approval,
		topic, command, metadata, created_at, updated_at
	) VALUES (?, ?, '', 'running', 'manual', 'low', 0, '', '', '{}', ?, ?)`,
		id.String(), title, now, now).Error; err != nil {
		t.Fatal(err)
	}
	return id
}

func insertPendingIteration(t *testing.T, title string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now().UTC()
	if err := database.DB.Exec(`INSERT INTO auto_iterations (
		id, title, description, status, source, risk_level, requires_super_admin_approval,
		topic, command, metadata, created_at, updated_at
	) VALUES (?, ?, '', 'pending', 'manual', 'low', 0, '', '', '{}', ?, ?)`,
		id.String(), title, now, now).Error; err != nil {
		t.Fatal(err)
	}
	return id
}

func TestCodeAgentPullTaskRespectsMaxConcurrent(t *testing.T) {
	setupAutoIterationWorkerDB(t)
	insertRunningIteration(t, "run-1")
	insertRunningIteration(t, "run-2")
	insertPendingIteration(t, "pending-1")

	task, err := CodeAgentPullTask(uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if task != nil {
		t.Fatalf("expected nil task at max concurrent, got %s", task.Title)
	}
}

func TestCodeAgentPullTaskAllowsWhenUnderCap(t *testing.T) {
	setupAutoIterationWorkerDB(t)
	insertRunningIteration(t, "run-1")
	insertPendingIteration(t, "pending-1")

	task, err := CodeAgentPullTask(uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if task == nil || task.Title != "pending-1" {
		t.Fatalf("expected pending-1, got %#v", task)
	}
}

func TestApproveRejectsPendingWithoutForce(t *testing.T) {
	setupAutoIterationWorkerDB(t)
	id := insertPendingIteration(t, "wait")
	_, err := ApproveAutoIteration(id, uuid.New(), "admin", "", false)
	if err != ErrAutoIterationInvalidState {
		t.Fatalf("expected invalid_state, got %v", err)
	}
}

func TestStartFromPendingIsIdempotent(t *testing.T) {
	setupAutoIterationWorkerDB(t)
	id := insertPendingIteration(t, "queued")
	row, err := StartAutoIteration(id, "admin")
	if err != nil || row.Status != models.AutoIterationStatusPending {
		t.Fatalf("start pending: err=%v status=%s", err, row.Status)
	}
}

func TestApproveAllowsPendingWithForce(t *testing.T) {
	setupAutoIterationWorkerDB(t)
	id := insertPendingIteration(t, "wait")
	row, err := ApproveAutoIteration(id, uuid.New(), "admin", "emergency", true)
	if err != nil {
		t.Fatalf("force approve err: %v", err)
	}
	if row == nil || row.Status != models.AutoIterationStatusApproved {
		st := ""
		if row != nil {
			st = row.Status
		}
		t.Fatalf("force approve status=%s", st)
	}
}

func insertRunningIterationForAgent(t *testing.T, title string, agentID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now().UTC()
	if err := database.DB.Exec(`INSERT INTO auto_iterations (
		id, title, description, status, source, risk_level, requires_super_admin_approval,
		topic, command, assigned_agent_id, metadata, created_at, updated_at
	) VALUES (?, ?, '', 'running', 'manual', 'low', 0, 'auto dev', '', ?, '{}', ?, ?)`,
		id.String(), title, agentID.String(), now, now).Error; err != nil {
		t.Fatal(err)
	}
	return id
}

func TestCodeAgentReportResultMarksCompleted(t *testing.T) {
	setupAutoIterationWorkerDB(t)
	agentID := uuid.New()
	iterID := insertRunningIterationForAgent(t, "done-task", agentID)
	if err := CodeAgentReportResult(iterID, agentID, CodeAgentTaskResult{Success: true, Summary: "release ok", GitHubSync: "ok", DeployStatus: "ok"}); err != nil {
		t.Fatalf("report result: %v", err)
	}
	var row models.AutoIteration
	if err := database.DB.Where("id = ?", iterID).First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != models.AutoIterationStatusCompleted {
		t.Fatalf("status=%s want completed", row.Status)
	}
	if row.Summary != "release ok" {
		t.Fatalf("summary=%q", row.Summary)
	}
}

func TestCodeAgentPullRespectsAutoDispatchOff(t *testing.T) {
	setupAutoIterationWorkerDB(t)
	_ = database.DB.Exec(`UPDATE auto_iteration_settings SET enabled = 1, auto_dispatch_enabled = 0 WHERE id = 1`)
	_ = database.DB.Exec(`INSERT INTO auto_iterations (id, title, status, source, risk_level, requires_super_admin_approval, metadata, created_at, updated_at)
		VALUES (?, 'x', 'pending', 'manual', 'low', 0, '{}', datetime('now'), datetime('now'))`, uuid.NewString())
	task, err := CodeAgentPullTask(uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if task != nil {
		t.Fatal("expected nil task when auto_dispatch disabled")
	}
}

func TestCodeAgentReportResultAwaitingApprovalWhenRequired(t *testing.T) {
	setupAutoIterationWorkerDB(t)
	agentID := uuid.New()
	id := uuid.New()
	now := time.Now().UTC()
	if err := database.DB.Exec(`INSERT INTO auto_iterations (
		id, title, description, status, source, risk_level, requires_super_admin_approval,
		topic, command, assigned_agent_id, metadata, created_at, updated_at
	) VALUES (?, 'risky', '', 'running', 'manual', 'high', 1, '', '', ?, '{}', ?, ?)`,
		id.String(), agentID.String(), now, now).Error; err != nil {
		t.Fatal(err)
	}
	if err := CodeAgentReportResult(id, agentID, CodeAgentTaskResult{Success: true, Summary: "needs review"}); err != nil {
		t.Fatalf("report result: %v", err)
	}
	var row models.AutoIteration
	if err := database.DB.Where("id = ?", id).First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != models.AutoIterationStatusAwaitingApproval {
		t.Fatalf("status=%s want awaiting_approval", row.Status)
	}
}

func TestUpdateAutoIterationSettingsAllToggles(t *testing.T) {
	setupAutoIterationWorkerDB(t)
	enabled := true
	github := true
	dingtalk := false
	got, err := UpdateAutoIterationSettings(&enabled, nil, nil, nil, nil, &github, &dingtalk, "", "tester")
	if err != nil {
		t.Fatalf("update settings: %v", err)
	}
	if !got.Enabled || !got.GitHubSyncEnabled || got.DingTalkNotifyEnabled {
		t.Fatalf("unexpected settings: %+v", got)
	}
}
