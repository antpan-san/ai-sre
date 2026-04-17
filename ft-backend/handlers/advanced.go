package handlers

import (
	"net/http"
	"time"

	"ft-backend/database"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Backup creates a backup (mock implementation).
func Backup(c *gin.Context) {
	var backupRequest struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&backupRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	backup := struct {
		ID          uuid.UUID `json:"id"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		Size        int64     `json:"size"`
		Status      string    `json:"status"`
		CreateTime  time.Time `json:"createTime"`
		UpdateTime  time.Time `json:"updateTime"`
		BackupTime  time.Time `json:"backupTime"`
	}{
		ID:          uuid.New(),
		Name:        backupRequest.Name,
		Description: backupRequest.Description,
		Size:        1024 * 1024 * 100,
		Status:      "completed",
		CreateTime:  time.Now(),
		UpdateTime:  time.Now(),
		BackupTime:  time.Now(),
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": backup, "msg": "success"})
}

// Restore restores from a backup (mock implementation).
func Restore(c *gin.Context) {
	if _, err := uuid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的备份ID"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success"})
}

// DeleteBackup deletes a backup (mock implementation).
func DeleteBackup(c *gin.Context) {
	if _, err := uuid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的备份ID"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "删除成功"})
}

// GetPerformanceData returns performance metrics.
func GetPerformanceData(c *gin.Context) {
	machineIDStr := c.Query("machineId")
	startTime := c.Query("startTime")
	endTime := c.Query("endTime")

	db := database.DB.Model(&models.PerformanceData{})

	if machineIDStr != "" {
		machineID, err := uuid.Parse(machineIDStr)
		if err == nil {
			db = db.Where("machine_id = ?", machineID)
		}
	}

	if startTime != "" {
		db = db.Where("timestamp >= ?", startTime)
	}
	if endTime != "" {
		db = db.Where("timestamp <= ?", endTime)
	}

	var performanceData []models.PerformanceData
	db.Order("timestamp ASC").Find(&performanceData)

	var machines []models.Machine
	database.DB.Find(&machines)

	machineInfo := make([]map[string]interface{}, 0, len(machines))
	for _, machine := range machines {
		machineInfo = append(machineInfo, map[string]interface{}{
			"id":   machine.ID,
			"name": machine.Name,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"data":     performanceData,
			"metrics":  []string{"cpu", "memory", "disk", "network"},
			"machines": machineInfo,
		},
		"msg": "success",
	})
}

// GeneratePerformanceReport generates a performance report.
func GeneratePerformanceReport(c *gin.Context) {
	var reportRequest struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		StartTime   string `json:"startTime" binding:"required"`
		EndTime     string `json:"endTime" binding:"required"`
	}
	if err := c.ShouldBindJSON(&reportRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	startTime, err := time.Parse(time.RFC3339, reportRequest.StartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的开始时间格式"})
		return
	}

	endTime, err := time.Parse(time.RFC3339, reportRequest.EndTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的结束时间格式"})
		return
	}

	var performanceData []models.PerformanceData
	database.DB.Where("timestamp BETWEEN ? AND ?", startTime, endTime).Find(&performanceData)

	var cpuTotal, memoryTotal, diskTotal, networkInTotal, networkOutTotal float64
	var cpuMax, memoryMax, diskMax float64
	var cpuMin, memoryMin, diskMin float64 = 100, 100, 100

	for _, data := range performanceData {
		cpuTotal += data.CPUUsage
		memoryTotal += data.MemoryUsage
		diskTotal += data.DiskUsage
		networkInTotal += data.NetworkIn
		networkOutTotal += data.NetworkOut

		if data.CPUUsage > cpuMax {
			cpuMax = data.CPUUsage
		}
		if data.CPUUsage < cpuMin {
			cpuMin = data.CPUUsage
		}
		if data.MemoryUsage > memoryMax {
			memoryMax = data.MemoryUsage
		}
		if data.MemoryUsage < memoryMin {
			memoryMin = data.MemoryUsage
		}
		if data.DiskUsage > diskMax {
			diskMax = data.DiskUsage
		}
		if data.DiskUsage < diskMin {
			diskMin = data.DiskUsage
		}
	}

	count := len(performanceData)
	cpuAvg, memoryAvg, diskAvg := cpuTotal, memoryTotal, diskTotal
	networkInAvg, networkOutAvg := networkInTotal, networkOutTotal

	if count > 0 {
		cpuAvg /= float64(count)
		memoryAvg /= float64(count)
		diskAvg /= float64(count)
		networkInAvg /= float64(count)
		networkOutAvg /= float64(count)
	}

	type metricSummary struct {
		Average float64 `json:"average"`
		Max     float64 `json:"max"`
		Min     float64 `json:"min"`
	}

	report := struct {
		ID          uuid.UUID `json:"id"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		StartTime   string    `json:"startTime"`
		EndTime     string    `json:"endTime"`
		ReportData  struct {
			CPU     metricSummary `json:"cpu"`
			Memory  metricSummary `json:"memory"`
			Disk    metricSummary `json:"disk"`
			Network struct {
				In  struct{ Average float64 `json:"average"` } `json:"in"`
				Out struct{ Average float64 `json:"average"` } `json:"out"`
			} `json:"network"`
		} `json:"reportData"`
		CreateTime time.Time `json:"createTime"`
	}{
		ID:          uuid.New(),
		Name:        reportRequest.Name,
		Description: reportRequest.Description,
		StartTime:   reportRequest.StartTime,
		EndTime:     reportRequest.EndTime,
		CreateTime:  time.Now(),
	}

	report.ReportData.CPU = metricSummary{cpuAvg, cpuMax, cpuMin}
	report.ReportData.Memory = metricSummary{memoryAvg, memoryMax, memoryMin}
	report.ReportData.Disk = metricSummary{diskAvg, diskMax, diskMin}
	report.ReportData.Network.In.Average = networkInAvg
	report.ReportData.Network.Out.Average = networkOutAvg

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": report, "msg": "success"})
}

// GetBackupList returns a list of backups (mock implementation).
func GetBackupList(c *gin.Context) {
	backups := []struct {
		ID          uuid.UUID `json:"id"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		Size        int64     `json:"size"`
		Status      string    `json:"status"`
		CreateTime  time.Time `json:"createTime"`
		UpdateTime  time.Time `json:"updateTime"`
		BackupTime  time.Time `json:"backupTime"`
	}{
		{
			ID:          uuid.New(),
			Name:        "系统备份_20251222",
			Description: "系统数据备份",
			Size:        1024 * 1024 * 100,
			Status:      "completed",
			CreateTime:  time.Now().Add(-24 * time.Hour),
			UpdateTime:  time.Now().Add(-24 * time.Hour),
			BackupTime:  time.Now().Add(-24 * time.Hour),
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{"list": backups, "total": len(backups)},
		"msg":  "success",
	})
}

// GetBackupDetail returns a single backup by UUID (mock implementation).
func GetBackupDetail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的备份ID"})
		return
	}

	backup := struct {
		ID          uuid.UUID `json:"id"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		Size        int64     `json:"size"`
		Status      string    `json:"status"`
		CreateTime  time.Time `json:"createTime"`
		UpdateTime  time.Time `json:"updateTime"`
		BackupTime  time.Time `json:"backupTime"`
	}{
		ID:          id,
		Name:        "系统备份_20251222",
		Description: "系统数据备份",
		Size:        1024 * 1024 * 100,
		Status:      "completed",
		CreateTime:  time.Now().Add(-24 * time.Hour),
		UpdateTime:  time.Now().Add(-24 * time.Hour),
		BackupTime:  time.Now().Add(-24 * time.Hour),
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": backup, "msg": "success"})
}
