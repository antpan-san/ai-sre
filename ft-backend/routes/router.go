package routes

import (
	"ft-backend/common/config"
	"ft-backend/handlers"
	"ft-backend/iotservice"
	"ft-backend/middleware"
	"ft-backend/models"
	"time"

	"github.com/gin-gonic/gin"
)

// SetupRouter configures all API routes.
func SetupRouter(cfg *config.Config) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()
	// 本机 Nginx / Vite 反代时 RemoteAddr 为 127.0.0.1，须信任上游才能正确解析 X-Forwarded-For（与限流键一致）。
	if err := r.SetTrustedProxies([]string{"127.0.0.1", "::1", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}); err != nil {
		gin.DefaultWriter.Write([]byte("gin SetTrustedProxies: " + err.Error() + "\n"))
	}
	r.Use(middleware.StripOptionalFtAPIPrefix())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg.Security.CORSAllowedOrigins))
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
		public.GET("/auth/public-options", handlers.PublicAuthOptions)
		public.GET("/auth/login-captcha", middleware.RateLimit("login-captcha", 40, time.Minute), handlers.GetLoginCaptcha)
		public.POST("/auth/register", middleware.RateLimit("register", 12, time.Minute), handlers.Register)
		public.POST("/auth/login", middleware.RateLimit("login", 30, time.Minute), handlers.Login)
		public.POST("/auth/logout", handlers.Logout)
		public.POST("/billing/stripe/webhook", handlers.StripeWebhook)

		// File download is public only for explicitly public files or valid share keys.
		public.GET("/files/download/:file_id", handlers.DownloadFile)

		// K8s 离线包：凭 invite id + token 下载（与控制台「一键安装引用」配套，无需 JWT）
		public.GET("/k8s/deploy/bundle-invite/:id/zip", handlers.DownloadK8sBundleInviteZip)
		public.GET("/k8s/deploy/bootstrap.sh", handlers.ServeK8sInstallBootstrap)
		public.GET("/k8s/deploy/install-ai-sre.sh", handlers.ServeAiSreInstallScript)
		public.GET("/k8s/deploy/cli/ai-sre/version", handlers.GetAiSreCLIVersion)
		public.GET("/k8s/deploy/cli/ai-sre", handlers.DownloadAiSreCLI)
		public.GET("/service-deploy/deployments/:id/bootstrap.sh", handlers.ServeServiceDeploymentBootstrap)
		public.GET("/service-deploy/deployments/:id/spec", handlers.GetServiceDeploymentSpec)
		public.POST("/service-deploy/deployments/:id/events", handlers.PostServiceDeploymentEvent)
		public.POST("/service-deploy/deployments/:id/finish", handlers.FinishServiceDeployment)
		public.POST("/execution-records/report/start", handlers.StartExecutionRecord)
		public.POST("/execution-records/report/event", handlers.PostExecutionEvent)
		public.POST("/execution-records/report/finish", handlers.FinishExecutionRecord)
		// AI diagnosis/evolution public endpoints for ai-sre runtime fallback
		aiPublic := public.Group("/ai")
		aiPublic.Use(middleware.RateLimit("public-ai", 30, time.Minute))
		aiPublic.POST("/diagnose", handlers.AIDiagnose)
		aiPublic.POST("/ask", handlers.AIAsk)
		aiPublic.POST("/runbook", handlers.AIRunbook)
		aiPublic.POST("/skills/evolve", handlers.AISkillsEvolve)
		// Self-iterating skill registry
		aiPublic.GET("/skills", handlers.AISkillsList)
		aiPublic.POST("/skills/refine", handlers.AISkillsRefine)
		aiPublic.POST("/skills/feedback", handlers.AISkillsFeedback)
		// 错误码 → 根因 卡片（控制台「部署错误码诊断」+ ai-sre analyze code 共用，纯只读）
		aiPublic.GET("/error-codes", handlers.ErrorCodesList)
		aiPublic.POST("/error-codes/analyze", handlers.ErrorCodeAnalyze)

		// Client Agent endpoints (authenticated by client_id, not JWT)
		agentPublic := public.Group("/v1")
		agentPublic.Use(middleware.RateLimit("agent-public", 300, time.Minute))
		agentPublic.POST("/heartbeats", iotservice.HeartbeatCheck)
		agentPublic.POST("/task/report", handlers.ReportTaskResult)
		agentPublic.POST("/task/log", handlers.PostTaskLog)
		agentPublic.GET("/tasks/running", handlers.GetRunningTasks)

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
		admin := protected.Group("")
		admin.Use(middleware.RequireAdmin())

		protected.GET("/billing/me", handlers.GetBillingMe)
		protected.POST("/billing/checkout-session", handlers.CreateStripeCheckoutSession)

		// ---- Dashboard ----
		protected.GET("/dashboard/data", handlers.GetDashboardData)

		// ---- User Management ----
		protected.GET("/auth/info", handlers.GetUserProfile)
		protected.PUT("/users/profile", handlers.UpdateUserProfile)
		admin.GET("/user", handlers.GetUserList)
		admin.GET("/user/:id", handlers.GetUserDetail)
		admin.POST("/user", handlers.AddUser)
		admin.PUT("/user/:id", handlers.UpdateUser)
		admin.DELETE("/user/:id", handlers.DeleteUser)
		admin.DELETE("/user/batch", handlers.BatchDeleteUser)
		admin.PATCH("/user/:id/role", handlers.UpdateUserRole)
		admin.GET("/admin/billing/features", handlers.AdminListFeatureBilling)
		admin.PUT("/admin/billing/features", handlers.AdminPutFeatureBilling)
		admin.POST("/admin/users/:id/entitlement", handlers.AdminGrantEntitlement)

		// ---- Machine Management ----
		admin.GET("/machine", handlers.GetMachineList)
		admin.GET("/machine/:id", handlers.GetMachineDetail)
		admin.POST("/machine", handlers.AddMachine)
		admin.PUT("/machine/:id", handlers.UpdateMachine)
		admin.DELETE("/machine/:id", handlers.DeleteMachine)
		admin.DELETE("/machine/batch", handlers.BatchDeleteMachine)
		admin.PATCH("/machine/:id/status", handlers.UpdateMachineStatus)
		admin.POST("/machine/:id/register-workers", handlers.RegisterWorkerNodes)

		// ---- Task System (core) ----
		admin.POST("/task", handlers.CreateTask)
		protected.GET("/task", handlers.GetTaskList)
		protected.GET("/task/:id", handlers.GetTaskDetail)
		admin.POST("/task/:id/cancel", handlers.CancelTask)
		protected.GET("/task/:id/logs", handlers.GetTaskLogs)

		// ---- Job Center ----
		protected.GET("/job/machines", handlers.GetJobMachines)
		admin.POST("/job/execute", handlers.ExecuteJob)
		protected.GET("/job/result/:jobId", handlers.GetJobResult)

		// ---- Execution Records ----
		protected.GET("/execution-records", handlers.GetExecutionRecords)
		protected.POST("/execution-records/prepare", handlers.PrepareExecutionRecord)
		protected.GET("/execution-records/:id", handlers.GetExecutionRecordDetail)
		protected.GET("/execution-records/:id/events", handlers.GetExecutionRecordEvents)
		protected.GET("/execution-records/:id/dependencies", handlers.GetExecutionRecordDependencies)
		admin.POST("/execution-records/:id/rollback-preview", handlers.PreviewExecutionRollback)
		admin.POST("/execution-records/:id/rollback", handlers.RollbackExecutionRecord)

		// ---- Service Management ----
		admin.POST("/service/deploy", handlers.DeployService)
		admin.GET("/service/list", handlers.GetServiceList)
		admin.GET("/service/detail", handlers.GetServiceDetail)
		admin.POST("/service/start", handlers.ServiceAction("start"))
		admin.POST("/service/stop", handlers.ServiceAction("stop"))
		admin.POST("/service/restart", handlers.ServiceAction("restart"))
		admin.DELETE("/service/delete", handlers.DeleteService)
		admin.POST("/service/batch-delete", handlers.BatchDeleteService)
		admin.GET("/service/linux/list", handlers.GetLinuxServiceList)
		admin.POST("/service/linux/operate", handlers.OperateLinuxService)
		admin.POST("/service-deploy/deployments", handlers.CreateServiceDeployment)
		admin.PUT("/service-deploy/deployments/:id", handlers.UpdateServiceDeployment)

		// ---- K8s Deployment ----
		protected.GET("/k8s/deploy/versions", handlers.GetK8sVersions)
		protected.GET("/k8s/deploy/component-catalog", handlers.GetK8sComponentCatalog)
		admin.GET("/k8s/deploy/machines", handlers.GetK8sDeployMachines)
		admin.GET("/k8s/deploy/check-name", handlers.CheckClusterName)
		admin.POST("/k8s/deploy/submit", handlers.SubmitK8sDeployWithAnsible) // Ansible-integrated
		admin.POST("/k8s/deploy/terminate", handlers.TerminateK8sDeploy)      // 终止部署并下发清理任务
		protected.GET("/k8s/deploy/progress", handlers.GetK8sDeployProgress)
		protected.GET("/k8s/deploy/logs", handlers.GetK8sDeployLogs)
		admin.GET("/k8s/clusters", handlers.GetK8sClusters)
		admin.POST("/k8s/deploy/bundle", handlers.GenerateK8sOfflineBundle)
		admin.POST("/k8s/deploy/bundle-invite", handlers.CreateK8sBundleInvite)
		protected.GET("/k8s/mirror/catalog", handlers.GetK8sMirrorCatalog)
		admin.GET("/k8s/deploy/relay/preflight", handlers.GetK8sRelayPreflight)
		admin.POST("/k8s/deploy/relay/warm", handlers.PostK8sRelayWarm)

		// ---- Proxy Configuration ----
		admin.GET("/proxy/config/list", handlers.GetProxyConfigList)
		admin.GET("/proxy/config/detail", handlers.GetProxyConfigDetail)
		admin.POST("/proxy/config/save", handlers.SaveProxyConfig)
		admin.DELETE("/proxy/config/delete", handlers.DeleteProxyConfig)
		admin.POST("/proxy/config/apply", handlers.ApplyProxyConfig)

		// ---- Init Tools ----
		admin.GET("/init-tools/system-params", handlers.GetSystemParams)
		admin.POST("/init-tools/system-params", handlers.ApplySystemParams)
		admin.POST("/init-tools/time-sync", handlers.ApplyTimeSync)
		admin.POST("/init-tools/security-harden", handlers.ApplySecurityHarden)
		admin.POST("/init-tools/disk-optimize", handlers.ApplyDiskOptimize)

		// ---- Monitoring ----
		admin.GET("/monitoring/configs", handlers.GetMonitoringConfigList)
		admin.GET("/monitoring/configs/:id", handlers.GetMonitoringConfig)
		admin.POST("/monitoring/configs", handlers.CreateMonitoringConfig)
		admin.PUT("/monitoring/configs/:id", handlers.UpdateMonitoringConfig)
		admin.DELETE("/monitoring/configs/:id", handlers.DeleteMonitoringConfig)
		admin.GET("/monitoring/alert-rules", handlers.GetAlertRules)
		admin.POST("/monitoring/alert-rules", handlers.CreateAlertRule)
		admin.PUT("/monitoring/alert-rules/:id", handlers.UpdateAlertRule)
		admin.DELETE("/monitoring/alert-rules/:id", handlers.DeleteAlertRule)

		// ---- Security & Audit ----
		admin.GET("/security-audit/operation-logs", handlers.GetOperationLogs)
		admin.GET("/security-audit/operation-logs/:id", handlers.GetOperationLogDetail)
		admin.GET("/security-audit/permissions", handlers.GetPermissions)
		admin.GET("/security-audit/permissions/:id", handlers.GetPermissionDetail)
		admin.POST("/security-audit/permissions", handlers.AddPermission)
		admin.PUT("/security-audit/permissions/:id", handlers.UpdatePermission)
		admin.DELETE("/security-audit/permissions/:id", handlers.DeletePermission)
		admin.DELETE("/security-audit/permissions/batch", handlers.BatchDeletePermissions)
		admin.GET("/security-audit/roles/:role/permissions", handlers.GetRolePermissions)
		admin.POST("/security-audit/roles/:role/permissions", handlers.AssignRolePermissions)

		adv := protected.Group("")
		adv.Use(middleware.RequireEntitlementOrAdmin(models.FeatureKeyAdvanced))
		// ---- Advanced / Backup (path aligned with frontend: /backups) ----
		adv.GET("/advanced/backups", handlers.GetBackupList)
		adv.GET("/advanced/backups/:id", handlers.GetBackupDetail)
		adv.POST("/advanced/backups", handlers.Backup)
		adv.POST("/advanced/backups/:id/restore", handlers.Restore)
		adv.DELETE("/advanced/backups/:id", handlers.DeleteBackup)
		// Legacy path (keep for compatibility)
		adv.GET("/advanced/backup", handlers.GetBackupList)
		adv.POST("/advanced/backup", handlers.Backup)

		// ---- Advanced / Performance ----
		adv.GET("/advanced/performance", handlers.GetPerformanceData)
		adv.POST("/advanced/performance/report/generate", handlers.GeneratePerformanceReport)
		// Legacy path
		adv.POST("/advanced/performance/report", handlers.GeneratePerformanceReport)

		// ---- File Management ----
		protected.POST("/files/upload", handlers.UploadFile)
		protected.GET("/files/list", handlers.ListFiles)
		protected.GET("/files/:file_id", handlers.GetFileInfo)
		protected.GET("/files/:file_id/download", handlers.DownloadOwnedFile)
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

	return r
}
