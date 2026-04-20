package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"ft-backend/common/logger"
	"ft-backend/common/redis"
	"ft-backend/database"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MachineListItem is the Machine model itself.
// Metrics are now persisted to DB columns directly, so no extra fields needed.
// If Redis has newer data, we overlay it on top.
type MachineListItem = models.Machine

// GetMachineList returns a paginated list of machines.
func GetMachineList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	name := c.Query("name")
	status := c.Query("status")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	offset := (page - 1) * pageSize

	db := database.DB.Model(&models.Machine{})

	if name != "" {
		db = db.Where("name ILIKE ?", "%"+name+"%")
	}
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if startDate != "" {
		db = db.Where("created_at >= ?", startDate)
	}
	if endDate != "" {
		db = db.Where("created_at <= ?", endDate)
	}

	var total int64
	db.Count(&total)

	var machines []models.Machine
	db.Limit(pageSize).Offset(offset).Order("created_at DESC").Find(&machines)
	items := mergeMachineMetrics(machines)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"list":  items,
			"total": total,
		},
		"msg": "success",
	})
}

// mergeMachineMetrics overlays Redis-cached metrics on top of DB data.
// Since metrics are now also persisted to DB, this provides a fresher snapshot
// when Redis is available, but falls back gracefully to DB columns.
func mergeMachineMetrics(machines []models.Machine) []MachineListItem {
	if !redis.IsConnected() {
		return machines
	}

	result := make([]MachineListItem, len(machines))
	copy(result, machines)

	for i := range result {
		m := &result[i]
		key := m.ClientID
		if key == "" && m.NodeRole == models.NodeRoleWorker {
			key = "worker:" + m.IP
		}
		if key == "" {
			continue
		}
		metrics, err := redis.GetMachineMetrics(key)
		if err != nil || metrics == nil {
			continue
		}
		// Overlay fresher Redis data
		if metrics.OSVersion != "" {
			m.OSVersion = metrics.OSVersion
		}
		if metrics.KernelVersion != "" {
			m.KernelVersion = metrics.KernelVersion
		}
		if metrics.CPUCores > 0 {
			m.CPUCores = metrics.CPUCores
		}
		m.CPUUsage = metrics.CPUUsage
		if metrics.MemoryTotal > 0 {
			m.MemoryTotal = metrics.MemoryTotal
		}
		if metrics.MemoryUsed > 0 {
			m.MemoryUsed = metrics.MemoryUsed
		}
		m.MemoryUsage = metrics.MemoryUsage
		if metrics.DiskTotal > 0 {
			m.DiskTotal = metrics.DiskTotal
		}
		if metrics.DiskUsed > 0 {
			m.DiskUsed = metrics.DiskUsed
		}
		m.DiskUsage = metrics.DiskUsage
	}
	return result
}

// GetMachineDetail returns a single machine by UUID.
func GetMachineDetail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的机器ID"})
		return
	}

	var machine models.Machine
	if err := database.DB.Where("id = ?", id).First(&machine).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "机器不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询机器失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": machine, "msg": "success"})
}

// AddMachine creates a new machine.
func AddMachine(c *gin.Context) {
	var machine models.Machine
	if err := c.ShouldBindJSON(&machine); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	if err := database.DB.Create(&machine).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "添加机器失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": machine, "msg": "success"})
}

// UpdateMachine updates an existing machine.
func UpdateMachine(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的机器ID"})
		return
	}

	var req models.Machine
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	var existing models.Machine
	if err := database.DB.Where("id = ?", id).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "机器不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询机器失败"})
		return
	}

	role := strings.TrimSpace(req.NodeRole)
	if role == "" {
		role = existing.NodeRole
	}
	if role != models.NodeRoleMaster && role != models.NodeRoleWorker && role != models.NodeRoleStandalone {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的节点角色"})
		return
	}

	updates := map[string]interface{}{
		"name":      req.Name,
		"ip":        req.IP,
		"cpu":       req.CPU,
		"memory":    req.Memory,
		"disk":      req.Disk,
		"status":    req.Status,
		"node_role": role,
	}

	if role == models.NodeRoleMaster {
		// Master must not point to any parent node.
		updates["master_machine_id"] = nil
		updates["cluster_id"] = req.ClusterID
		if req.ClusterID != nil {
			var anotherMaster models.Machine
			err := database.DB.
				Where("cluster_id = ? AND node_role = ? AND id <> ?", *req.ClusterID, models.NodeRoleMaster, id).
				First(&anotherMaster).Error
			if err == nil {
				c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "该集群已存在 Master，请先调整现有 Master"})
				return
			}
			if err != gorm.ErrRecordNotFound {
				c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "校验集群 Master 失败"})
				return
			}
		}
	} else if role == models.NodeRoleWorker {
		updates["cluster_id"] = req.ClusterID
		updates["master_machine_id"] = req.MasterMachineID
	} else {
		// Standalone should not keep cluster topology references.
		updates["cluster_id"] = nil
		updates["master_machine_id"] = nil
	}

	if err := database.DB.Model(&existing).Updates(updates).Error; err != nil {
		if strings.Contains(err.Error(), "idx_machines_tenant_cluster_master") {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "该集群已存在 Master，请先调整现有 Master"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "更新机器失败"})
		return
	}

	var updated models.Machine
	_ = database.DB.Where("id = ?", id).First(&updated)
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": updated, "msg": "success"})
}

// DeleteMachine soft-deletes a machine.
func DeleteMachine(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的机器ID"})
		return
	}

	var machine models.Machine
	if err := database.DB.Where("id = ?", id).First(&machine).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "机器不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询机器失败"})
		return
	}

	if err := database.DB.Delete(&machine).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "删除机器失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success"})
}

// BatchDeleteMachine soft-deletes multiple machines.
func BatchDeleteMachine(c *gin.Context) {
	var request struct {
		IDs []uuid.UUID `json:"ids"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	if err := database.DB.Where("id IN ?", request.IDs).Delete(&models.Machine{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "批量删除机器失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success"})
}

// RegisterWorkerNodes registers worker (slave) nodes for a master machine.
// It creates machine records in DB, then dispatches a sync_nodes command to
// the master client via the Task/SubTask system (with Redis queue acceleration).
//
// POST /api/machine/:id/register-workers
//
//	Body: { "workers": [ { "ip":"…","hostname":"…","ssh_port":22,"ssh_user":"root","ssh_key":"/…" }, … ] }
func RegisterWorkerNodes(c *gin.Context) {
	masterID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的机器ID"})
		return
	}

	var req struct {
		Workers []struct {
			IP          string `json:"ip" binding:"required"`
			Hostname    string `json:"hostname"`
			SSHPort     int    `json:"ssh_port"`
			SSHUser     string `json:"ssh_user"`
			AuthType    string `json:"auth_type"` // "password" or "key"
			SSHPassword string `json:"ssh_password"`
			SSHKey      string `json:"ssh_key"`
		} `json:"workers" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数", "error": err.Error()})
		return
	}

	// 1. Validate master machine
	var master models.Machine
	if err := database.DB.Where("id = ?", masterID).First(&master).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "主控机器不存在"})
			return
		}
		logger.Error("RegisterWorkerNodes: query master %s failed: %v", masterID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询机器失败", "error": err.Error()})
		return
	}
	if master.NodeRole != models.NodeRoleMaster {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "该机器不是 Master 角色，无法注册受控节点"})
		return
	}
	if master.ClientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Master 尚未上报 Client ID，请等待心跳上报后再试"})
		return
	}

	// 2. Validate credentials per worker
	for i, w := range req.Workers {
		authType := w.AuthType
		if authType == "" {
			authType = "key" // default for backward compat
		}
		if authType == "password" && w.SSHPassword == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"code": 400,
				"msg":  "受控节点 " + w.IP + " 选择了密码认证但未填写密码",
			})
			return
		}
		if authType == "key" && w.SSHKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"code": 400,
				"msg":  "受控节点 " + w.IP + " 选择了密钥认证但未填写密钥路径",
			})
			return
		}
		req.Workers[i].AuthType = authType
	}

	// 3. Upsert worker machine records in DB
	type workerPayload struct {
		IP          string `json:"ip"`
		Hostname    string `json:"hostname"`
		SSHPort     int    `json:"ssh_port"`
		SSHUser     string `json:"ssh_user"`
		AuthType    string `json:"auth_type"`
		SSHPassword string `json:"ssh_password,omitempty"`
		SSHKey      string `json:"ssh_key,omitempty"`
	}
	var syncPayloadWorkers []workerPayload
	var workersCreated int

	tx := database.DB.Begin()
	if tx.Error != nil {
		logger.Error("RegisterWorkerNodes: begin tx failed: %v", tx.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "开始事务失败", "error": tx.Error.Error()})
		return
	}

	for _, w := range req.Workers {
		// Trim all string fields to guard against accidental leading/trailing whitespace
		// entered in the UI — e.g. " 192.168.56.11" would miss the ip-based lookup and
		// then fail on the fingerprint/client_id unique constraints.
		ip := strings.TrimSpace(w.IP)
		port := w.SSHPort
		if port == 0 {
			port = 22
		}
		user := strings.TrimSpace(w.SSHUser)
		if user == "" {
			user = "root"
		}
		hostname := strings.TrimSpace(w.Hostname)
		if hostname == "" {
			hostname = ip
		}

		// Check if worker machine already exists (match by trimmed IP)
		var existing models.Machine
		err := tx.Where("ip = ?", ip).First(&existing).Error
		if err == nil {
			// Update existing machine to be a worker of this master
			if updateErr := tx.Model(&existing).Updates(map[string]interface{}{
				"node_role":         models.NodeRoleWorker,
				"cluster_id":        master.ClusterID,
				"master_machine_id": master.ID,
				"name":              hostname,
			}).Error; updateErr != nil {
				tx.Rollback()
				logger.Error("RegisterWorkerNodes: update worker %s failed: %v", ip, updateErr)
				c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "更新受控节点失败: " + ip, "error": updateErr.Error()})
				return
			}
		} else if err == gorm.ErrRecordNotFound {
			// Create new worker machine.
			// IMPORTANT: Omit both ClientID and HostFingerprint so these columns stay
			// NULL in PostgreSQL.  Both fields have partial unique indexes:
			//   (tenant_id, client_id)        WHERE client_id IS NOT NULL
			//   (tenant_id, host_fingerprint) WHERE host_fingerprint IS NOT NULL
			// Go's zero-value "" is NOT NULL in PostgreSQL, so inserting "" for multiple
			// workers would violate the constraint.  NULL is excluded by the index.
			newMachine := models.Machine{
				Name:            hostname,
				IP:              ip,
				Status:          "offline",
				NodeRole:        models.NodeRoleWorker,
				ClusterID:       master.ClusterID,
				MasterMachineID: &master.ID,
				Labels: models.NewJSONBFromMap(map[string]interface{}{
					"registered_by": "web_ui",
					"master_id":     master.ID.String(),
				}),
			}
			if err := tx.Omit("ClientID", "HostFingerprint").Create(&newMachine).Error; err != nil {
				tx.Rollback()
				logger.Error("RegisterWorkerNodes: create worker %s failed: %v", ip, err)
				c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建受控节点失败: " + ip, "error": err.Error()})
				return
			}
			workersCreated++
		} else {
			// Unexpected DB error while checking existing machine
			tx.Rollback()
			logger.Error("RegisterWorkerNodes: query worker %s failed: %v", ip, err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询受控节点失败: " + ip, "error": err.Error()})
			return
		}

		syncPayloadWorkers = append(syncPayloadWorkers, workerPayload{
			IP:          ip,
			Hostname:    hostname,
			SSHPort:     port,
			SSHUser:     user,
			AuthType:    w.AuthType,
			SSHPassword: strings.TrimSpace(w.SSHPassword),
			SSHKey:      strings.TrimSpace(w.SSHKey),
		})
	}

	// 4. Create a Task + SubTask for the master to sync its node manager
	username, _ := c.Get("username")
	usernameStr, _ := username.(string)

	payloadJSON, _ := json.Marshal(map[string]interface{}{
		"workers": syncPayloadWorkers,
	})

	targetIDsJSON, _ := json.Marshal([]string{masterID.String()})

	task := models.Task{
		Name:        "注册受控节点",
		Type:        string(models.TaskTypeRegisterNodes),
		Status:      string(models.TaskStatusPending),
		CreatedBy:   usernameStr,
		Description: "为 Master " + master.Name + " 注册 " + strconv.Itoa(len(req.Workers)) + " 个受控节点",
		Payload:     models.JSONB(payloadJSON),
		TargetIDs:   models.JSONB(targetIDsJSON),
		TotalCount:  1,
		TimeoutSec:  60,
	}
	if err := tx.Create(&task).Error; err != nil {
		tx.Rollback()
		logger.Error("RegisterWorkerNodes: create task for master %s failed: %v", masterID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建任务失败", "error": err.Error()})
		return
	}

	// Use the machine IP as the stable SubTask identifier so the DB slow-path
	// `(client_id = ? OR client_id = localIP)` query still matches even when
	// the client restarts and generates a new random client_id.
	// The Redis fast-path delivery (below) uses both the current clientID and
	// the IP so whichever key the client polls first will deliver the command.
	subTask := models.SubTask{
		TaskID:    task.ID,
		MachineID: masterID,
		ClientID:  master.IP,
		Command:   "sync_nodes",
		Status:    string(models.TaskStatusPending),
		Payload:   models.JSONB(payloadJSON),
		MaxRetry:  3,
	}
	if err := tx.Create(&subTask).Error; err != nil {
		tx.Rollback()
		logger.Error("RegisterWorkerNodes: create subtask for master %s failed: %v", masterID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建子任务失败", "error": err.Error()})
		return
	}

	// Log
	tx.Create(&models.TaskLog{
		TaskID:   task.ID,
		Level:    "info",
		Message:  "注册受控节点任务已创建，等待 Master 心跳拉取",
		ClientID: master.ClientID,
	})

	if err := tx.Commit().Error; err != nil {
		logger.Error("RegisterWorkerNodes: commit tx for master %s failed: %v", masterID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "提交事务失败", "error": err.Error()})
		return
	}

	// 5. Also push to Redis queue for faster delivery (if available).
	// Enqueue using BOTH the current client_id AND the machine IP so that:
	//   • If the client still has the same client_id → immediate delivery via clientID key.
	//   • If the client restarted with a new client_id → delivery via IP key (matched by
	//     fetchPendingCommands's `client_id = localIP` fallback condition).
	if redis.IsConnected() {
		cmd := models.Command{
			TaskID:    task.ID.String(),
			SubTaskID: subTask.ID.String(),
			Command:   "sync_nodes",
			Payload:   json.RawMessage(payloadJSON),
			Timeout:   60,
		}
		for _, key := range []string{master.ClientID, master.IP} {
			if key == "" {
				continue
			}
			if err := redis.EnqueueTask(key, cmd); err != nil {
				// Non-fatal: client will still pick it up via DB query on next heartbeat
				_ = err
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "注册请求已提交",
		"data": gin.H{
			"task_id":         task.ID,
			"workers_created": workersCreated,
		},
	})
}

// UpdateMachineStatus updates only the status of a machine.
func UpdateMachineStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的机器ID"})
		return
	}

	var request struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	var machine models.Machine
	if err := database.DB.Where("id = ?", id).First(&machine).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "机器不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询机器失败"})
		return
	}

	machine.Status = request.Status
	if err := database.DB.Save(&machine).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "更新机器状态失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": machine, "msg": "success"})
}
