package handlers

import (
	"context"
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
	roleVal, _ := c.Get("role")
	role, _ := roleVal.(string)
	userVal, _ := c.Get("username")
	username, _ := userVal.(string)

	tid := defaultTenantUUID()
	now := time.Now()

	// 业务侧未统一上报 Machine 心跳时不在此聚合托管机；概览「服务端资源」仅 super_admin 见本机采样（见 resourceUsage / hostRuntime）。
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
	recentK8s := make([]gin.H, 0)
	recentInstalls := make([]gin.H, 0)
	if models.IsAdminRole(role) {
		database.DB.Model(&models.K8sCluster{}).Where("tenant_id = ?", tid).Count(&totalK8s)
		database.DB.Model(&models.K8sCluster{}).Where("tenant_id = ? AND status = ?", tid, "running").Count(&k8sRunning)
		database.DB.Model(&models.K8sCluster{}).Where("tenant_id = ? AND status = ?", tid, "pending").Count(&k8sPending)
		database.DB.Model(&models.K8sCluster{}).Where("tenant_id = ? AND status = ?", tid, "failed").Count(&k8sFailed)

		var recentClusters []models.K8sCluster
		database.DB.Model(&models.K8sCluster{}).Where("tenant_id = ?", tid).
			Order("updated_at DESC").Limit(8).Find(&recentClusters)
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
		for _, d := range recentInst {
			recentInstalls = append(recentInstalls, gin.H{
				"id":            d.ID.String(),
				"service":       d.Service,
				"profile":       d.Profile,
				"status":        d.Status,
				"currentStep":   d.CurrentStep,
				"installMethod": d.InstallMethod,
				"updatedAt":     d.UpdatedAt.Format(time.RFC3339),
			})
		}
	}

	var tasksActive int64
	database.DB.Model(&models.Task{}).Where("tenant_id = ?", tid).
		Where("status IN ?", []string{string(models.TaskStatusPending), string(models.TaskStatusDispatched), string(models.TaskStatusRunning)}).
		Count(&tasksActive)

	var executions24h int64
	t24 := now.Add(-24 * time.Hour)
	exec24 := database.DB.Model(&models.ExecutionRecord{}).Where("tenant_id = ?", tid).
		Where("created_at >= ?", t24)
	exec24 = applyExecutionConsoleMemberScope(exec24, role, username)
	exec24.Count(&executions24h)

	var totalUsers, totalOpLogs int64
	if models.IsSuperAdminRole(role) {
		database.DB.Model(&models.User{}).Where("tenant_id = ?", tid).Count(&totalUsers)
		database.DB.Model(&models.OperationLog{}).Where("tenant_id = ?", tid).Count(&totalOpLogs)
	}

	zeroMachines := gin.H{
		"total":   0,
		"online":  0,
		"offline": 0,
		"masters": 0,
		"workers": 0,
	}

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

	var recentExec []models.ExecutionRecord
	qExec := database.DB.Model(&models.ExecutionRecord{}).Where("tenant_id = ?", tid)
	qExec = applyExecutionConsoleMemberScope(qExec, role, username)
	qExec.Order("created_at DESC").Limit(12).Find(&recentExec)

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

	platformSummary := gin.H{
		"machines": zeroMachines,
		"k8sClusters": gin.H{
			"total":   totalK8s,
			"running": k8sRunning,
			"pending": k8sPending,
			"failed":  k8sFailed,
		},
		"tasksActive":       tasksActive,
		"executionsLast24h": executions24h,
	}
	if models.IsSuperAdminRole(role) {
		platformSummary["usersTotal"] = totalUsers
		platformSummary["operationLogsTotal"] = totalOpLogs
	}

	resourceUsage := gin.H{
		"cpu":    0.0,
		"memory": 0.0,
		"disk":   0.0,
		"network": gin.H{
			"in":  0,
			"out": 0,
		},
	}
	var hostRuntime gin.H
	if models.IsSuperAdminRole(role) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2500*time.Millisecond)
		hr := collectHostRuntime(ctx, 320*time.Millisecond)
		cancel()
		resourceUsage["cpu"] = clampPct(hr.CPU)
		resourceUsage["memory"] = clampPct(hr.Memory)
		resourceUsage["disk"] = clampPct(hr.Disk)
		hostRuntime = gin.H{
			"hostname":  hr.Hostname,
			"sampledAt": hr.SampledAt,
			"os":        hr.OS,
		}
		if hr.ErrCollect != "" {
			hostRuntime["error"] = hr.ErrCollect
		}
	}

	data := gin.H{
		"resourceUsage":      resourceUsage,
		"kubernetesOverview": gin.H{"nodes": 0, "pods": totalK8s, "runningPods": 0, "services": svcTotal, "deployments": k8sRunning, "replicasets": tasksActive},
		"serviceStatusStats": gin.H{"running": runningPie, "stopped": svcStopped, "error": svcError, "total": svcTotal},
		"recentDeployments":  recentDeployments,
		"platformSummary":    platformSummary,
		"recentK8sClusters":    recentK8s,
		"recentServiceInstalls": recentInstalls,
		"recentExecutions":   recentExecutions,
	}
	if hostRuntime != nil && len(hostRuntime) > 0 {
		data["hostRuntime"] = hostRuntime
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "获取仪表盘数据成功",
		"data": data,
	})
}
