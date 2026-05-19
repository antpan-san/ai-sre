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

func setupAutoIterationListDB(t *testing.T) {
	t.Helper()
	logger.InitLogger("error", io.Discard)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	database.DB = db
	if err := db.Exec(`CREATE TABLE auto_iterations (
		id TEXT PRIMARY KEY, tenant_id TEXT, created_at DATETIME, updated_at DATETIME,
		title TEXT, description TEXT, status TEXT, source TEXT, risk_level TEXT,
		requires_super_admin_approval INTEGER, topic TEXT, command TEXT, summary TEXT,
		feedback_id TEXT, created_by_user_id TEXT, created_by TEXT, approved_by_user_id TEXT,
		approved_by TEXT, approved_at DATETIME, assigned_agent_id TEXT, last_error TEXT, metadata TEXT
	)`).Error; err != nil {
		t.Fatal(err)
	}
}

func insertAutoIteration(t *testing.T, title, status, source, topic, command string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now().UTC()
	if err := database.DB.Exec(`INSERT INTO auto_iterations (
		id, title, description, status, source, risk_level, requires_super_admin_approval,
		topic, command, metadata, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?, '{}', ?, ?)`,
		id.String(), title, title+" desc", status, source, models.AutoIterationRiskLow, topic, command, now, now,
	).Error; err != nil {
		t.Fatal(err)
	}
	return id
}

func TestListAutoIterationsFilters(t *testing.T) {
	setupAutoIterationListDB(t)
	insertAutoIteration(t, "Billing fix", models.AutoIterationStatusPending, models.AutoIterationSourceManual, "billing", "fix invoice")
	insertAutoIteration(t, "Monitor alert", models.AutoIterationStatusRunning, models.AutoIterationSourceCLIFeedback, "monitoring", "reduce noise")

	byStatus, total, err := ListAutoIterations(AutoIterationListFilter{Status: models.AutoIterationStatusRunning, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(byStatus) != 1 || byStatus[0].Title != "Monitor alert" {
		t.Fatalf("status filter: total=%d rows=%d", total, len(byStatus))
	}

	byTopic, total, err := ListAutoIterations(AutoIterationListFilter{Topic: "billing", Page: 1, PageSize: 20})
	if err != nil || total != 1 || byTopic[0].Topic != "billing" {
		t.Fatalf("topic filter: total=%d err=%v", total, err)
	}

	bySource, total, err := ListAutoIterations(AutoIterationListFilter{Source: models.AutoIterationSourceManual, Page: 1, PageSize: 20})
	if err != nil || total != 1 || bySource[0].Source != models.AutoIterationSourceManual {
		t.Fatalf("source filter: total=%d", total)
	}

	// Keyword uses ILIKE (PostgreSQL); skip on SQLite in-memory tests.
	t.Run("keyword_skipped_sqlite", func(t *testing.T) {
		t.Skip("keyword filter uses ILIKE; covered in production PostgreSQL")
	})
}
