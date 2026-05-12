package handlers

import (
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ft-backend/database"
	"ft-backend/models"
)

func defaultTenantUUID() uuid.UUID {
	return uuid.MustParse(models.DefaultTenantID)
}

func clampPct(v float64) float64 {
	if math.IsNaN(v) || v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return math.Round(v*10) / 10
}

// GetDashboardData 聚合租户内真实数据：资产、Kubernetes 清单、Linux 服务、任务与执行记录。
func GetDashboardData(c *gin.Context) {
	if _, exists := c.Get("userID"); !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权"})
		return
	}
	tid := defaultTenantUUID()
	now := time.Now()

	var totalMachines, onlineMachines, offlineMachines, masterMachines, workerMachines int64
	database.DB.Model(&models.Machine{}).Where("tenant_id = ?", tid).Count(&totalMachines)
	database.DB.Model(&models.Machine{}).Where("tenant_id = ? AND status = ?", tid, "online").Count(&onlineMachines)
	database.DB.Model(&models.Machine{}).Where("tenant_id = ? AND status = ?", tid, "offline").Count(&offlineMachines)
	database.DB.Model(&models.Machine{}).Where("tenant_id = ? AND node_role = ?", tid, models.NodeRoleMaster).Count(&masterMachines)
	database.DB.Model(&models.Machine{}).Where("tenant_id = ? AND node_role = ?", tid, models.NodeRoleWorker).Count(&workerMachines)

	var svcRunning, svcStopped, svcError, svcDeploying, svcTotal int64
	database.DB.Model(&models.Service{}).Where("tenant_id = ?", tid).Where("status = ?", "running").Count(&svcRunning)
	database.DB.Model(&models.Service{}).Where("tenant_id = ?", tid).Where("status = ?", "deploying").Count(&svcDeploying)
	database.DB.Model(&models.Service{}).Where("tenant_id = ?", tid).Where("status = ?", "stopped").Count(&svcStopped)
	database.DB.Model(&models.Service{}).Where("tenant_id = ?", tid).
		Where("status IN ?", []string{"error", "failed"}).Count(&svcError)
	database.DB.Model(&models.Service{}).Where("tenant_id = ?", tid).Count(&svcTotal)

	// 「运行态」圆圈：running + deploying
	runningPie := svcRunning + svcDeploying

	var totalK8s, k8sRunning, k8sPending, k8sFailed int64
	database.DB.Model(&models.K8sCluster{}).Where("tenant_id = ?", tid).Count(&totalK8s)
	database.DB.Model(&models.K8sCluster{}).Where("tenant_id = ? AND status = ?", tid, "running").Count(&k8sRunning)
	database.DB.Model(&models.K8sCluster{}).Where("tenant_id = ? AND status = ?", tid, "pending").Count(&k8sPending)
	database.DB.Model(&models.K8sCluster{}).Where("tenant_id = ? AND status = ?", tid, "failed").Count(&k8sFailed)

	var tasksActive int64
	database.DB.Model(&models.Task{}).Where("tenant_id = ?", tid).
		Where("status IN ?", []string{string(models.TaskStatusPending), string(models.TaskStatusDispatched), string(models.TaskStatusRunning)}).
		Count(&tasksActive)

	var executions24h int64
	t24 := now.Add(-24 * time.Hour)
	database.DB.Model(&models.ExecutionRecord{}).Where("tenant_id = ?", tid).
		Where("created_at >= ?", t24).Count(&executions24h)

	var totalUsers int64
	database.DB.Model(&models.User{}).Where("tenant_id = ?", tid).Count(&totalUsers)

	var totalOpLogs int64
	database.DB.Model(&models.OperationLog{}).Where("tenant_id = ?", tid).Count(&totalOpLogs)

	// 在线机器平均资源占用（离线机器不报心跳，不参与平均更准确）。
	type metricAgg struct {
		AvgCPU    float64 `gorm:"column:avg_cpu"`
		AvgMemory float64 `gorm:"column:avg_memory"`
		AvgDisk   float64 `gorm:"column:avg_disk"`
	}
	var onlineAgg metricAgg
	_ = database.DB.Model(&models.Machine{}).
		Where("tenant_id = ?", tid).
		Where("status = ?", "online").
		Select(`COALESCE(AVG(cpu_usage), 0) AS avg_cpu,
			COALESCE(AVG(memory_usage), 0) AS avg_memory,
			COALESCE(AVG(disk_usage), 0) AS avg_disk`).
		Scan(&onlineAgg).Error

	var recentSvcs []models.Service
	database.DB.Model(&models.Service{}).
		Where("tenant_id = ?", tid).
		Order("updated_at DESC").
		Limit(12).
		Find(&recentSvcs)

	recentDeployments := make([]gin.H, 0, len(recentSvcs))
	for _, s := range recentSvcs {
		st := s.Status
		recentDeployments = append(recentDeployments, gin.H{
			"id":         s.ID.String(),
			"name":       s.Name,
			"image":      s.Image,
			"replicas":   s.Replicas,
			"status":     st,
			"createTime": s.CreatedAt.Format(time.RFC3339),
			"updateTime": s.UpdatedAt.Format(time.RFC3339),
		})
	}

	var recentClusters []models.K8sCluster
	database.DB.Model(&models.K8sCluster{}).Where("tenant_id = ?", tid).
		Order("updated_at DESC").Limit(8).Find(&recentClusters)

	recentK8s := make([]gin.H, 0, len(recentClusters))
	for _, k := range recentClusters {
		recentK8s = append(recentK8s, gin.H{
			"id":          k.ID.String(),
			"clusterName": k.ClusterName,
			"status":      k.Status,
			"version":     k.Version,
			"masterNode":  k.MasterNode,
			"updatedAt":   k.UpdatedAt.Format(time.RFC3339),
		})
	}

	var recentInst []models.ServiceDeployment
	database.DB.Model(&models.ServiceDeployment{}).Where("tenant_id = ?", tid).
		Order("updated_at DESC").Limit(8).Find(&recentInst)

	recentInstalls := make([]gin.H, 0, len(recentInst))
	for _, d := range recentInst {
		recentInstalls = append(recentInstalls, gin.H{
			"id":           d.ID.String(),
			"service":      d.Service,
			"profile":      d.Profile,
			"status":       d.Status,
			"currentStep":  d.CurrentStep,
			"installMethod": d.InstallMethod,
			"updatedAt":    d.UpdatedAt.Format(time.RFC3339),
		})
	}

	var recentExec []models.ExecutionRecord
	database.DB.Model(&models.ExecutionRecord{}).Where("tenant_id = ?", tid).
		Order("created_at DESC").
		Limit(12).
		Find(&recentExec)

	recentExecutions := make([]gin.H, 0, len(recentExec))
	for _, e := range recentExec {
		fin := ""
		if e.FinishedAt != nil {
			fin = e.FinishedAt.Format(time.RFC3339)
		}
		recentExecutions = append(recentExecutions, gin.H{
			"id":             e.ID.String(),
			"name":           e.Name,
			"status":         e.Status,
			"category":       e.Category,
			"source":         e.Source,
			"targetHost":     e.TargetHost,
			"resourceName":   e.ResourceName,
			"finishedAt":     fin,
			"durationMs":     e.DurationMs,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "获取仪表盘数据成功",
		"data": gin.H{
			// 兼容原有前端卡片：语义见页面文案（Kubernetes 一栏为控制台侧清单，非 kube-apiserver）
			"resourceUsage": gin.H{
				"cpu":    clampPct(onlineAgg.AvgCPU),
				"memory": clampPct(onlineAgg.AvgMemory),
				"disk":   clampPct(onlineAgg.AvgDisk),
				"network": gin.H{
					"in":  0,
					"out": 0,
				},
			},
			"kubernetesOverview": gin.H{
				"nodes":       totalMachines,
				"pods":        totalK8s,
				"runningPods": onlineMachines,
				"services":    svcTotal,
				"deployments": k8sRunning,
				"replicasets": tasksActive,
			},
			"serviceStatusStats": gin.H{
				"running": runningPie,
				"stopped": svcStopped,
				"error":   svcError,
				"total":   svcTotal,
			},
			"recentDeployments": recentDeployments,

			"platformSummary": gin.H{
				"machines": gin.H{
					"total":   totalMachines,
					"online":  onlineMachines,
					"offline": offlineMachines,
					"masters": masterMachines,
					"workers": workerMachines,
				},
				"k8sClusters": gin.H{
					"total":   totalK8s,
					"running": k8sRunning,
					"pending": k8sPending,
					"failed":  k8sFailed,
				},
				"tasksActive":        tasksActive,
				"executionsLast24h":  executions24h,
				"usersTotal":         totalUsers,
				"operationLogsTotal": totalOpLogs,
			},
			"recentK8sClusters":     recentK8s,
			"recentServiceInstalls": recentInstalls,
			"recentExecutions":      recentExecutions,
		},
	})
}
