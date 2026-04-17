package routes

import (
	"ft-backend/common/config"
	"ft-backend/handlers"
	"ft-backend/iotservice"
	"ft-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRouter configures all API routes.
func SetupRouter(cfg *config.Config) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()
	r.Use(middleware.CORS())
	r.Use(func(c *gin.Context) {
		c.Set("config", cfg)
		c.Next()
	})

	// ================================================
	// Health check (public)
	// ================================================
	r.GET("/health", handlers.HealthCheck)

	// ================================================
	// Public API routes (no JWT required)
	// ================================================
	public := r.Group("/api")
	{
		// Auth
		public.POST("/auth/login", handlers.Login)
		public.POST("/auth/logout", handlers.Logout)

		// File download (public)
		public.GET("/files/download/:file_id", handlers.DownloadFile)

		// Client Agent endpoints (authenticated by client_id, not JWT)
		public.POST("/v1/heartbeats", iotservice.HeartbeatCheck)
		public.POST("/v1/task/report", handlers.ReportTaskResult)
		public.POST("/v1/task/log", handlers.PostTaskLog)
		public.GET("/v1/tasks/running", handlers.GetRunningTasks)

		// Debug endpoints (only in debug mode)
		if cfg.Log.Level == "debug" {
			public.GET("/debug/token", handlers.DebugGetToken)
		}
	}

	// ================================================
	// Protected API routes (JWT required)
	// ================================================
	protected := r.Group("/api")
	protected.Use(middleware.JWTAuth(cfg.JWT.SecretKey))
	{
		// ---- Dashboard ----
		protected.GET("/dashboard/data", handlers.GetDashboardData)

		// ---- User Management ----
		protected.GET("/auth/info", handlers.GetUserProfile)
		protected.PUT("/users/profile", handlers.UpdateUserProfile)
		protected.GET("/user", handlers.GetUserList)
		protected.GET("/user/:id", handlers.GetUserDetail)
		protected.POST("/user", handlers.AddUser)
		protected.PUT("/user/:id", handlers.UpdateUser)
		protected.DELETE("/user/:id", handlers.DeleteUser)
		protected.DELETE("/user/batch", handlers.BatchDeleteUser)
		protected.PATCH("/user/:id/role", handlers.UpdateUserRole)

		// ---- Machine Management ----
		protected.GET("/machine", handlers.GetMachineList)
		protected.GET("/machine/:id", handlers.GetMachineDetail)
		protected.POST("/machine", handlers.AddMachine)
		protected.PUT("/machine/:id", handlers.UpdateMachine)
		protected.DELETE("/machine/:id", handlers.DeleteMachine)
		protected.DELETE("/machine/batch", handlers.BatchDeleteMachine)
		protected.PATCH("/machine/:id/status", handlers.UpdateMachineStatus)
		protected.POST("/machine/:id/register-workers", handlers.RegisterWorkerNodes)

		// ---- Task System (core) ----
		protected.POST("/task", handlers.CreateTask)
		protected.GET("/task", handlers.GetTaskList)
		protected.GET("/task/:id", handlers.GetTaskDetail)
		protected.POST("/task/:id/cancel", handlers.CancelTask)
		protected.GET("/task/:id/logs", handlers.GetTaskLogs)

		// ---- Job Center ----
		protected.GET("/job/machines", handlers.GetJobMachines)
		protected.POST("/job/execute", handlers.ExecuteJob)
		protected.GET("/job/result/:jobId", handlers.GetJobResult)

		// ---- Service Management ----
		protected.POST("/service/deploy", handlers.DeployService)
		protected.GET("/service/list", handlers.GetServiceList)
		protected.GET("/service/detail", handlers.GetServiceDetail)
		protected.POST("/service/start", handlers.ServiceAction("start"))
		protected.POST("/service/stop", handlers.ServiceAction("stop"))
		protected.POST("/service/restart", handlers.ServiceAction("restart"))
		protected.DELETE("/service/delete", handlers.DeleteService)
		protected.POST("/service/batch-delete", handlers.BatchDeleteService)
		protected.GET("/service/linux/list", handlers.GetLinuxServiceList)
		protected.POST("/service/linux/operate", handlers.OperateLinuxService)

		// ---- K8s Deployment ----
		protected.GET("/k8s/deploy/versions", handlers.GetK8sVersions)
		protected.GET("/k8s/deploy/machines", handlers.GetK8sDeployMachines)
		protected.GET("/k8s/deploy/check-name", handlers.CheckClusterName)
		protected.POST("/k8s/deploy/submit", handlers.SubmitK8sDeployWithAnsible)   // Ansible-integrated
		protected.POST("/k8s/deploy/terminate", handlers.TerminateK8sDeploy)      // 终止部署并下发清理任务
		protected.GET("/k8s/deploy/progress", handlers.GetK8sDeployProgress)
		protected.GET("/k8s/deploy/logs", handlers.GetK8sDeployLogs)
		protected.GET("/k8s/deploy/records", handlers.GetK8sDeployRecords)
		protected.GET("/k8s/clusters", handlers.GetK8sClusters)
		protected.POST("/k8s/deploy/bundle", handlers.GenerateK8sOfflineBundle)

		// ---- Proxy Configuration ----
		protected.GET("/proxy/config/list", handlers.GetProxyConfigList)
		protected.GET("/proxy/config/detail", handlers.GetProxyConfigDetail)
		protected.POST("/proxy/config/save", handlers.SaveProxyConfig)
		protected.DELETE("/proxy/config/delete", handlers.DeleteProxyConfig)
		protected.POST("/proxy/config/apply", handlers.ApplyProxyConfig)

		// ---- Init Tools ----
		protected.GET("/init-tools/system-params", handlers.GetSystemParams)
		protected.POST("/init-tools/system-params", handlers.ApplySystemParams)
		protected.POST("/init-tools/time-sync", handlers.ApplyTimeSync)
		protected.POST("/init-tools/security-harden", handlers.ApplySecurityHarden)
		protected.POST("/init-tools/disk-optimize", handlers.ApplyDiskOptimize)

		// ---- Monitoring ----
		protected.GET("/monitoring/configs", handlers.GetMonitoringConfigList)
		protected.GET("/monitoring/configs/:id", handlers.GetMonitoringConfig)
		protected.POST("/monitoring/configs", handlers.CreateMonitoringConfig)
		protected.PUT("/monitoring/configs/:id", handlers.UpdateMonitoringConfig)
		protected.DELETE("/monitoring/configs/:id", handlers.DeleteMonitoringConfig)
		protected.GET("/monitoring/alert-rules", handlers.GetAlertRules)
		protected.POST("/monitoring/alert-rules", handlers.CreateAlertRule)
		protected.PUT("/monitoring/alert-rules/:id", handlers.UpdateAlertRule)
		protected.DELETE("/monitoring/alert-rules/:id", handlers.DeleteAlertRule)

		// ---- Security & Audit ----
		protected.GET("/security-audit/operation-logs", handlers.GetOperationLogs)
		protected.GET("/security-audit/operation-logs/:id", handlers.GetOperationLogDetail)
		protected.GET("/security-audit/permissions", handlers.GetPermissions)
		protected.GET("/security-audit/permissions/:id", handlers.GetPermissionDetail)
		protected.POST("/security-audit/permissions", handlers.AddPermission)
		protected.PUT("/security-audit/permissions/:id", handlers.UpdatePermission)
		protected.DELETE("/security-audit/permissions/:id", handlers.DeletePermission)
		protected.DELETE("/security-audit/permissions/batch", handlers.BatchDeletePermissions)
		protected.GET("/security-audit/roles/:role/permissions", handlers.GetRolePermissions)
		protected.POST("/security-audit/roles/:role/permissions", handlers.AssignRolePermissions)

		// ---- Advanced / Backup (path aligned with frontend: /backups) ----
		protected.GET("/advanced/backups", handlers.GetBackupList)
		protected.GET("/advanced/backups/:id", handlers.GetBackupDetail)
		protected.POST("/advanced/backups", handlers.Backup)
		protected.POST("/advanced/backups/:id/restore", handlers.Restore)
		protected.DELETE("/advanced/backups/:id", handlers.DeleteBackup)
		// Legacy path (keep for compatibility)
		protected.GET("/advanced/backup", handlers.GetBackupList)
		protected.POST("/advanced/backup", handlers.Backup)

		// ---- Advanced / Performance ----
		protected.GET("/advanced/performance", handlers.GetPerformanceData)
		protected.POST("/advanced/performance/report/generate", handlers.GeneratePerformanceReport)
		// Legacy path
		protected.POST("/advanced/performance/report", handlers.GeneratePerformanceReport)

		// ---- File Management ----
		protected.POST("/files/upload", handlers.UploadFile)
		protected.GET("/files/list", handlers.ListFiles)
		protected.GET("/files/:file_id", handlers.GetFileInfo)
		protected.DELETE("/files/:file_id", handlers.DeleteFile)
		protected.POST("/files/share/:file_id", handlers.ShareFile)
		protected.GET("/files/shared", handlers.GetSharedFiles)

		// ---- Transfer History ----
		protected.GET("/transfers", handlers.GetTransferHistory)

		// ---- Debug (only in debug mode) ----
		if cfg.Log.Level == "debug" {
			protected.GET("/debug/test-auth", handlers.DebugTestAuth)
		}
	}

	// ================================================
	// WebSocket
	// ================================================
	r.GET("/ws/:user_id", handlers.WebSocketHandler)

	// ================================================
	// Static files
	// ================================================
	r.Static("/uploads", cfg.File.UploadDir)

	return r
}
