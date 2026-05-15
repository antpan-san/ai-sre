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
		public.GET("/cli/install-ai-sre.sh", middleware.RateLimit("cli-install-script", 120, time.Minute), handlers.ServeAiSreInstallScriptForSession)
		public.POST("/cli/install-bind", middleware.RateLimit("cli-install-bind", 60, time.Minute), handlers.BindCLIInstallSession)
		public.GET("/k8s/deploy/cli/ai-sre/version", handlers.GetAiSreCLIVersion)
		public.GET("/k8s/deploy/cli/ai-sre", handlers.DownloadAiSreCLI)
		public.GET("/service-deploy/deployments/:id/bootstrap.sh", handlers.ServeServiceDeploymentBootstrap)
		public.GET("/service-deploy/deployments/:id/spec", handlers.GetServiceDeploymentSpec)
		public.POST("/service-deploy/deployments/:id/events", handlers.PostServiceDeploymentEvent)
		public.POST("/service-deploy/deployments/:id/finish", handlers.FinishServiceDeployment)
		public.POST("/execution-records/report/start", handlers.StartExecutionRecord)
		public.POST("/execution-records/report/event", handlers.PostExecutionEvent)
		public.POST("/execution-records/report/finish", handlers.FinishExecutionRecord)
		public.POST("/runtime-watch/sample", middleware.RateLimit("runtime-watch-sample", 3000, time.Minute), handlers.PostRuntimeWatchSample)
		public.GET("/cli/go-runtime/auth-check", middleware.RateLimit("cli-go-runtime-auth-check", 300, time.Minute), handlers.CheckCLIGoRuntimeAuth)
		public.POST("/cli/go-runtime/reports", middleware.RateLimit("cli-go-runtime-report", 300, time.Minute), handlers.PostCLIGoRuntimeReport)
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
		console := protected.Group("")
		console.Use(middleware.RequireConsoleMember())

		adminOnly := protected.Group("")
		adminOnly.Use(middleware.RequireAdmin())

		superAdmin := protected.Group("")
		superAdmin.Use(middleware.RequireSuperAdmin())

		// 已登录：个性化安装脚本（写入 opsfleet_token，关联订阅与 AI 配额）
		protected.POST("/me/cli/install-session", handlers.CreateCLIInstallSession)
		protected.GET("/me/cli/install-ai-sre.sh", handlers.ServeAiSreInstallScriptForUser)

		protected.GET("/billing/me", handlers.GetBillingMe)
		protected.GET("/billing/capabilities", handlers.GetBillingCapabilities)
		protected.GET("/billing/packages", handlers.ListBillingPackages)
		protected.POST("/billing/checkout-session", handlers.CreateStripeCheckoutSession)

		// ---- Dashboard ----
		protected.GET("/dashboard/data", handlers.GetDashboardData)

		// ---- User Management（仅管理员） ----
		protected.GET("/auth/info", handlers.GetUserProfile)
		protected.PUT("/users/profile", handlers.UpdateUserProfile)
		adminOnly.GET("/user", handlers.GetUserList)
		adminOnly.GET("/user/:id", handlers.GetUserDetail)
		adminOnly.POST("/user", handlers.AddUser)
		adminOnly.PUT("/user/:id", handlers.UpdateUser)
		adminOnly.DELETE("/user/:id", handlers.DeleteUser)
		adminOnly.DELETE("/user/batch", handlers.BatchDeleteUser)
		adminOnly.PATCH("/user/:id/role", handlers.UpdateUserRole)
		superAdmin.GET("/admin/billing/features", handlers.AdminListFeatureBilling)
		superAdmin.PUT("/admin/billing/features", handlers.AdminPutFeatureBilling)
		superAdmin.POST("/admin/users/:id/entitlement", handlers.AdminGrantEntitlement)

		// ---- Machine Management（普通登录用户可查，变更仅管理员） ----
		console.GET("/machine", handlers.GetMachineList)
		console.GET("/machine/:id", handlers.GetMachineDetail)
		adminOnly.POST("/machine", handlers.AddMachine)
		adminOnly.PUT("/machine/:id", handlers.UpdateMachine)
		adminOnly.DELETE("/machine/:id", handlers.DeleteMachine)
		adminOnly.DELETE("/machine/batch", handlers.BatchDeleteMachine)
		adminOnly.PATCH("/machine/:id/status", handlers.UpdateMachineStatus)
		adminOnly.POST("/machine/:id/register-workers", handlers.RegisterWorkerNodes)

		// ---- Task System (core) ----
		adminOnly.POST("/task", middleware.RequireCapability(models.FeatureKeyNodeOps, middleware.CapabilityActionExecute), handlers.CreateTask)
		protected.GET("/task", handlers.GetTaskList)
		protected.GET("/task/:id", handlers.GetTaskDetail)
		adminOnly.POST("/task/:id/cancel", handlers.CancelTask)
		protected.GET("/task/:id/logs", handlers.GetTaskLogs)

		// ---- Job Center ----
		protected.GET("/job/machines", handlers.GetJobMachines)
		adminOnly.POST("/job/execute", middleware.RequireCapability(models.FeatureKeyNodeOps, middleware.CapabilityActionExecute), handlers.ExecuteJob)
		protected.GET("/job/result/:jobId", handlers.GetJobResult)

		// ---- Execution Records ----
		protected.GET("/execution-records", handlers.GetExecutionRecords)
		protected.POST("/execution-records/prepare", handlers.PrepareExecutionRecord)
		protected.GET("/execution-records/:id", handlers.GetExecutionRecordDetail)
		protected.GET("/execution-records/:id/events", handlers.GetExecutionRecordEvents)
		protected.GET("/execution-records/:id/dependencies", handlers.GetExecutionRecordDependencies)
		adminOnly.POST("/execution-records/:id/rollback-preview", handlers.PreviewExecutionRollback)
		adminOnly.POST("/execution-records/:id/rollback", handlers.RollbackExecutionRecord)

		// ---- Service Management（计费开启时需 pack.node_ops） ----
		svcAdmin := console.Group("")
		svcAdmin.GET("/service/list", middleware.RequireCapability(models.FeatureKeyNodeOps, middleware.CapabilityActionView), handlers.GetServiceList)
		svcAdmin.GET("/service/detail", middleware.RequireCapability(models.FeatureKeyNodeOps, middleware.CapabilityActionView), handlers.GetServiceDetail)
		svcAdmin.GET("/service/linux/list", middleware.RequireCapability(models.FeatureKeyNodeOps, middleware.CapabilityActionView), handlers.GetLinuxServiceList)
		svcAdmin.POST("/service/deploy", middleware.RequireCapability(models.FeatureKeyNodeOps, middleware.CapabilityActionExecute), handlers.DeployService)
		svcAdmin.POST("/service/start", middleware.RequireCapability(models.FeatureKeyNodeOps, middleware.CapabilityActionExecute), handlers.ServiceAction("start"))
		svcAdmin.POST("/service/stop", middleware.RequireCapability(models.FeatureKeyNodeOps, middleware.CapabilityActionExecute), handlers.ServiceAction("stop"))
		svcAdmin.POST("/service/restart", middleware.RequireCapability(models.FeatureKeyNodeOps, middleware.CapabilityActionExecute), handlers.ServiceAction("restart"))
		svcAdmin.DELETE("/service/delete", middleware.RequireCapability(models.FeatureKeyNodeOps, middleware.CapabilityActionExecute), handlers.DeleteService)
		svcAdmin.POST("/service/batch-delete", middleware.RequireCapability(models.FeatureKeyNodeOps, middleware.CapabilityActionExecute), handlers.BatchDeleteService)
		svcAdmin.POST("/service/linux/operate", middleware.RequireCapability(models.FeatureKeyNodeOps, middleware.CapabilityActionExecute), handlers.OperateLinuxService)
		svcAdmin.POST("/service-deploy/deployments", middleware.RequireCapability(models.FeatureKeyNodeOps, middleware.CapabilityActionExecute), handlers.CreateServiceDeployment)
		svcAdmin.PUT("/service-deploy/deployments/:id", middleware.RequireCapability(models.FeatureKeyNodeOps, middleware.CapabilityActionExecute), handlers.UpdateServiceDeployment)

		// ---- K8s Deployment（计费开启时需 pack.k8s_delivery） ----
		k8sPaid := protected.Group("")
		k8sPaid.GET("/k8s/deploy/versions", middleware.RequireCapability(models.FeatureKeyK8sDelivery, middleware.CapabilityActionView), handlers.GetK8sVersions)
		k8sPaid.GET("/k8s/deploy/component-catalog", middleware.RequireCapability(models.FeatureKeyK8sDelivery, middleware.CapabilityActionView), handlers.GetK8sComponentCatalog)
		k8sPaid.GET("/k8s/deploy/progress", middleware.RequireCapability(models.FeatureKeyK8sDelivery, middleware.CapabilityActionView), handlers.GetK8sDeployProgress)
		k8sPaid.GET("/k8s/deploy/logs", middleware.RequireCapability(models.FeatureKeyK8sDelivery, middleware.CapabilityActionView), handlers.GetK8sDeployLogs)
		k8sPaid.GET("/k8s/mirror/catalog", middleware.RequireCapability(models.FeatureKeyK8sDelivery, middleware.CapabilityActionView), handlers.GetK8sMirrorCatalog)
		k8sAdminPaid := console.Group("")
		k8sAdminPaid.GET("/k8s/deploy/machines", middleware.RequireCapability(models.FeatureKeyK8sDelivery, middleware.CapabilityActionView), handlers.GetK8sDeployMachines)
		k8sAdminPaid.GET("/k8s/deploy/check-name", middleware.RequireCapability(models.FeatureKeyK8sDelivery, middleware.CapabilityActionView), handlers.CheckClusterName)
		k8sAdminPaid.GET("/k8s/clusters", middleware.RequireCapability(models.FeatureKeyK8sDelivery, middleware.CapabilityActionView), handlers.GetK8sClusters)
		k8sAdminPaid.GET("/k8s/deploy/relay/preflight", middleware.RequireCapability(models.FeatureKeyK8sDelivery, middleware.CapabilityActionView), handlers.GetK8sRelayPreflight)
		k8sAdminPaid.POST("/k8s/deploy/submit", middleware.RequireCapability(models.FeatureKeyK8sDelivery, middleware.CapabilityActionExecute), handlers.SubmitK8sDeployWithAnsible) // Ansible-integrated
		k8sAdminPaid.POST("/k8s/deploy/terminate", middleware.RequireCapability(models.FeatureKeyK8sDelivery, middleware.CapabilityActionExecute), handlers.TerminateK8sDeploy)      // 终止部署并下发清理任务
		k8sAdminPaid.POST("/k8s/deploy/bundle", middleware.RequireCapability(models.FeatureKeyK8sDelivery, middleware.CapabilityActionDownload), handlers.GenerateK8sOfflineBundle)
		k8sAdminPaid.POST("/k8s/deploy/bundle-invite", middleware.RequireCapability(models.FeatureKeyK8sDelivery, middleware.CapabilityActionDownload), handlers.CreateK8sBundleInvite)
		k8sAdminPaid.POST("/k8s/deploy/relay/warm", middleware.RequireCapability(models.FeatureKeyK8sDelivery, middleware.CapabilityActionDownload), handlers.PostK8sRelayWarm)

		// ---- Proxy / Monitoring / Init Tools（计费开启时需 pack.node_ops / pack.monitoring） ----
		infraAdmin := console.Group("")
		infraAdmin.GET("/proxy/config/list", middleware.RequireCapability(models.FeatureKeyNodeOps, middleware.CapabilityActionView), handlers.GetProxyConfigList)
		infraAdmin.GET("/proxy/config/detail", middleware.RequireCapability(models.FeatureKeyNodeOps, middleware.CapabilityActionView), handlers.GetProxyConfigDetail)
		infraAdmin.POST("/proxy/config/save", middleware.RequireCapability(models.FeatureKeyNodeOps, middleware.CapabilityActionExecute), handlers.SaveProxyConfig)
		infraAdmin.DELETE("/proxy/config/delete", middleware.RequireCapability(models.FeatureKeyNodeOps, middleware.CapabilityActionExecute), handlers.DeleteProxyConfig)
		infraAdmin.POST("/proxy/config/apply", middleware.RequireCapability(models.FeatureKeyNodeOps, middleware.CapabilityActionExecute), handlers.ApplyProxyConfig)
		infraAdmin.GET("/init-tools/system-params", middleware.RequireCapability(models.FeatureKeyNodeOps, middleware.CapabilityActionView), handlers.GetSystemParams)
		infraAdmin.POST("/init-tools/system-params", middleware.RequireCapability(models.FeatureKeyNodeOps, middleware.CapabilityActionExecute), handlers.ApplySystemParams)
		infraAdmin.POST("/init-tools/time-sync", middleware.RequireCapability(models.FeatureKeyNodeOps, middleware.CapabilityActionExecute), handlers.ApplyTimeSync)
		infraAdmin.POST("/init-tools/security-harden", middleware.RequireCapability(models.FeatureKeyNodeOps, middleware.CapabilityActionExecute), handlers.ApplySecurityHarden)
		infraAdmin.POST("/init-tools/disk-optimize", middleware.RequireCapability(models.FeatureKeyNodeOps, middleware.CapabilityActionExecute), handlers.ApplyDiskOptimize)
		infraAdmin.GET("/monitoring/configs", middleware.RequireCapability(models.FeatureKeyMonitoring, middleware.CapabilityActionView), handlers.GetMonitoringConfigList)
		infraAdmin.GET("/monitoring/configs/:id", middleware.RequireCapability(models.FeatureKeyMonitoring, middleware.CapabilityActionView), handlers.GetMonitoringConfig)
		infraAdmin.POST("/monitoring/configs", middleware.RequireCapability(models.FeatureKeyMonitoring, middleware.CapabilityActionExecute), handlers.CreateMonitoringConfig)
		infraAdmin.PUT("/monitoring/configs/:id", middleware.RequireCapability(models.FeatureKeyMonitoring, middleware.CapabilityActionExecute), handlers.UpdateMonitoringConfig)
		infraAdmin.DELETE("/monitoring/configs/:id", middleware.RequireCapability(models.FeatureKeyMonitoring, middleware.CapabilityActionExecute), handlers.DeleteMonitoringConfig)
		infraAdmin.GET("/monitoring/alert-rules", middleware.RequireCapability(models.FeatureKeyMonitoring, middleware.CapabilityActionView), handlers.GetAlertRules)
		infraAdmin.POST("/monitoring/alert-rules", middleware.RequireCapability(models.FeatureKeyMonitoring, middleware.CapabilityActionExecute), handlers.CreateAlertRule)
		infraAdmin.PUT("/monitoring/alert-rules/:id", middleware.RequireCapability(models.FeatureKeyMonitoring, middleware.CapabilityActionExecute), handlers.UpdateAlertRule)
		infraAdmin.DELETE("/monitoring/alert-rules/:id", middleware.RequireCapability(models.FeatureKeyMonitoring, middleware.CapabilityActionExecute), handlers.DeleteAlertRule)

		// ---- Security & Audit（读：租户成员；写：管理员） ----
		console.GET("/security-audit/operation-logs", handlers.GetOperationLogs)
		console.GET("/security-audit/operation-logs/:id", handlers.GetOperationLogDetail)
		console.GET("/security-audit/permissions", handlers.GetPermissions)
		console.GET("/security-audit/permissions/:id", handlers.GetPermissionDetail)
		adminOnly.POST("/security-audit/permissions", handlers.AddPermission)
		adminOnly.PUT("/security-audit/permissions/:id", handlers.UpdatePermission)
		adminOnly.DELETE("/security-audit/permissions/:id", handlers.DeletePermission)
		adminOnly.DELETE("/security-audit/permissions/batch", handlers.BatchDeletePermissions)
		console.GET("/security-audit/roles/:role/permissions", handlers.GetRolePermissions)
		adminOnly.POST("/security-audit/roles/:role/permissions", handlers.AssignRolePermissions)

		adv := protected.Group("")
		advAdmin := protected.Group("")
		advAdmin.Use(middleware.RequireAdmin())
		// ---- Advanced / Backup (path aligned with frontend: /backups) ----
		adv.GET("/advanced/backups", middleware.RequireCapability(models.FeatureKeyBackupPerformance, middleware.CapabilityActionReport), handlers.GetBackupList)
		adv.GET("/advanced/backups/:id", middleware.RequireCapability(models.FeatureKeyBackupPerformance, middleware.CapabilityActionReport), handlers.GetBackupDetail)
		advAdmin.POST("/advanced/backups", middleware.RequireCapability(models.FeatureKeyBackupPerformance, middleware.CapabilityActionExecute), handlers.Backup)
		advAdmin.POST("/advanced/backups/:id/restore", middleware.RequireCapability(models.FeatureKeyBackupPerformance, middleware.CapabilityActionExecute), handlers.Restore)
		advAdmin.DELETE("/advanced/backups/:id", middleware.RequireCapability(models.FeatureKeyBackupPerformance, middleware.CapabilityActionExecute), handlers.DeleteBackup)
		// Legacy path (keep for compatibility)
		adv.GET("/advanced/backup", middleware.RequireCapability(models.FeatureKeyBackupPerformance, middleware.CapabilityActionReport), handlers.GetBackupList)
		advAdmin.POST("/advanced/backup", middleware.RequireCapability(models.FeatureKeyBackupPerformance, middleware.CapabilityActionExecute), handlers.Backup)

		// ---- Advanced / Performance ----
		adv.GET("/advanced/performance", middleware.RequireCapability(models.FeatureKeyBackupPerformance, middleware.CapabilityActionReport), handlers.GetPerformanceData)
		adv.POST("/advanced/performance/report/generate", middleware.RequireCapability(models.FeatureKeyBackupPerformance, middleware.CapabilityActionReport), handlers.GeneratePerformanceReport)
		// Legacy path
		adv.POST("/advanced/performance/report", middleware.RequireCapability(models.FeatureKeyBackupPerformance, middleware.CapabilityActionReport), handlers.GeneratePerformanceReport)

		runtimeWatch := protected.Group("")
		runtimeWatch.GET("/runtime-watch/sessions", middleware.RequireCapability(models.FeatureKeyRuntimeObserve, middleware.CapabilityActionReport), handlers.ListRuntimeWatchSessions)
		runtimeWatch.POST("/runtime-watch/sessions", middleware.RequireCapability(models.FeatureKeyRuntimeObserve, middleware.CapabilityActionExecute), handlers.CreateRuntimeWatchSession)
		runtimeWatch.GET("/runtime-watch/sessions/:id/samples", middleware.RequireCapability(models.FeatureKeyRuntimeObserve, middleware.CapabilityActionReport), handlers.GetRuntimeWatchSamples)
		runtimeWatch.POST("/runtime-watch/sessions/:id/stop", middleware.RequireCapability(models.FeatureKeyRuntimeObserve, middleware.CapabilityActionExecute), handlers.StopRuntimeWatchSession)
		runtimeWatch.DELETE("/runtime-watch/sessions/:id", middleware.RequireCapability(models.FeatureKeyRuntimeObserve, middleware.CapabilityActionExecute), handlers.DeleteRuntimeWatchSession)

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
