package database

import (
	"fmt"
	"time"

	"ft-backend/common/config"
	"ft-backend/common/logger"
	"ft-backend/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

var DB *gorm.DB

// Connect connects to the PostgreSQL database.
func Connect(cfg *config.DatabaseConfig) error {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.DBName,
		cfg.SSLMode,
		cfg.TimeZone,
	)

	logger.Info("Connecting to PostgreSQL: %s@%s:%s/%s", cfg.User, cfg.Host, cfg.Port, cfg.DBName)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Silent),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	})

	if err != nil {
		logger.Error("Database connection failed: %v", err)
		return fmt.Errorf("failed to connect database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := DB.DB()
	if err != nil {
		logger.Error("Failed to get database instance: %v", err)
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Ping
	if err := sqlDB.Ping(); err != nil {
		logger.Error("Database ping failed: %v", err)
		return fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connection established")
	return nil
}

// Migrate runs GORM AutoMigrate for all models.
// NOTE: The heartbeats table is partitioned and MUST be created via migration_pg.sql.
func Migrate() error {
	logger.Info("Starting database migration")

	err := DB.AutoMigrate(
		&models.Tenant{},
		&models.User{},
		&models.Role{},
		&models.File{},
		&models.Transfer{},
		&models.Share{},
		&models.Machine{},
		&models.OperationLog{},
		&models.Permission{},
		&models.RolePermission{},
		&models.PerformanceData{},
		&models.K8sVersion{},
		&models.K8sCluster{},
		&models.K8sBundleInvite{},
		&models.Task{},
		&models.SubTask{},
		&models.TaskLog{},
		&models.ExecutionRecord{},
		&models.ExecutionEvent{},
		&models.ExecutionDependency{},
		&models.MonitoringConfig{},
		&models.AlertRule{},
		&models.Service{},
		&models.ProxyConfig{},
		// NOTE: Heartbeat is NOT included here – it is a partitioned table
		// and must be created via the migration_pg.sql script.
	)

	if err != nil {
		logger.Error("Database migration failed: %v", err)
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	// Ensure default tenant exists
	ensureDefaultTenant()

	// Seed default roles
	initRoles()

	// Seed K8s versions
	initK8sVersions()

	logger.Info("Database migration completed")
	return nil
}

// ensureDefaultTenant makes sure the default tenant row exists.
func ensureDefaultTenant() {
	var count int64
	DB.Model(&models.Tenant{}).Where("code = ?", "default").Count(&count)
	if count == 0 {
		logger.Info("Creating default tenant")
		tenant := models.Tenant{
			Name:   "Default Tenant",
			Code:   "default",
			Status: "active",
		}
		// Use a well-known UUID for the default tenant
		tenant.ID = models.MustParseUUID(models.DefaultTenantID)
		if err := DB.Create(&tenant).Error; err != nil {
			logger.Error("Failed to create default tenant: %v", err)
		}
	}
}

// initRoles seeds default system roles.
func initRoles() {
	logger.Debug("Checking roles data")

	var count int64
	if err := DB.Model(&models.Role{}).Count(&count).Error; err != nil {
		logger.Error("Failed to count roles: %v", err)
		return
	}

	if count == 0 {
		logger.Info("Seeding default roles")
		roles := []models.Role{
			{Name: "管理员", Code: "admin", Description: "系统管理员，拥有所有权限", IsSystem: true},
			{Name: "普通用户", Code: "user", Description: "普通用户，拥有基础查看权限", IsSystem: true},
			{Name: "运维工程师", Code: "operator", Description: "运维人员，拥有机器管理和任务执行权限", IsSystem: true},
			{Name: "只读用户", Code: "viewer", Description: "只读用户，仅有查看权限", IsSystem: true},
		}
		if err := DB.CreateInBatches(roles, 5).Error; err != nil {
			logger.Error("Failed to seed roles: %v", err)
			return
		}
		logger.Info("Seeded %d roles", len(roles))
	}
}

// initK8sVersions seeds default K8s version data.
func initK8sVersions() {
	logger.Debug("Checking K8s version data")

	var count int64
	if err := DB.Model(&models.K8sVersion{}).Count(&count).Error; err != nil {
		logger.Error("Failed to count K8s versions: %v", err)
		return
	}

	if count == 0 {
		logger.Info("Seeding default K8s versions")

		// 与 deploy/k8s-mirror/k8s-mirror-versions.txt 保持一致（内网制品同步脚本按该列表拉 kubernetes-server）
		defaultVersions := []string{
			"v1.35.4", "v1.32.11", "v1.34.3",
			"v1.32.6", "v1.28.15", "v1.30.0",
		}

		var versions []models.K8sVersion
		for _, version := range defaultVersions {
			versions = append(versions, models.K8sVersion{
				Version:  version,
				IsActive: true,
			})
		}

		if err := DB.CreateInBatches(versions, 5).Error; err != nil {
			logger.Error("Failed to seed K8s versions: %v", err)
			return
		}

		logger.Info("Seeded %d K8s versions", len(versions))
	} else {
		logger.Debug("K8s version data already exists, skipping seed")
	}
}

// GetK8sVersions returns all active Kubernetes versions.
func GetK8sVersions() ([]models.K8sVersion, error) {
	logger.Debug("Fetching K8s versions")

	var versions []models.K8sVersion
	result := DB.Where("is_active = ?", true).Find(&versions)
	if result.Error != nil {
		logger.Error("Failed to fetch K8s versions: %v", result.Error)
		return nil, fmt.Errorf("failed to get k8s versions: %w", result.Error)
	}

	logger.Debug("Fetched %d K8s versions", len(versions))
	return versions, nil
}

// GetDBStatus returns database connection pool statistics.
func GetDBStatus() map[string]interface{} {
	status := make(map[string]interface{})

	sqlDB, err := DB.DB()
	if err != nil {
		status["status"] = "error"
		status["error"] = fmt.Sprintf("Failed to get database instance: %v", err)
		return status
	}

	stats := sqlDB.Stats()
	status["status"] = "connected"
	status["idle"] = stats.Idle
	status["in_use"] = stats.InUse
	status["max_open_connections"] = stats.MaxOpenConnections
	status["open_connections"] = stats.OpenConnections
	status["wait_count"] = stats.WaitCount
	status["wait_duration"] = stats.WaitDuration.String()

	return status
}

// Close closes the database connection.
func Close() error {
	logger.Info("Closing database connection")

	sqlDB, err := DB.DB()
	if err != nil {
		logger.Error("Failed to get database instance: %v", err)
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		logger.Error("Failed to close database connection: %v", err)
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	logger.Info("Database connection closed")
	return nil
}

// GetGormDB returns the GORM database instance.
func GetGormDB() *gorm.DB {
	return DB
}

// ExecRawSQL executes a raw SQL statement.
func ExecRawSQL(query string, args ...interface{}) error {
	logger.Warn("Executing raw SQL: %s", query)

	result := DB.Exec(query, args...)
	if result.Error != nil {
		logger.Error("Raw SQL execution failed: %v", result.Error)
		return fmt.Errorf("failed to execute SQL: %w", result.Error)
	}

	logger.Debug("Raw SQL executed, rows affected: %d", result.RowsAffected)
	return nil
}
