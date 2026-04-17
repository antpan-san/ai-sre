package iotservice

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"ft-backend/common/logger"
	"ft-backend/common/redis"
	"ft-backend/database"
	"ft-backend/models"
	"ft-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ClientHeartbeat is the top-level heartbeat payload from client agents.
// Must stay in sync with ft-client/internal/model.HeartbeatRequest.
type ClientHeartbeat struct {
	ClientID       string `json:"client_id" binding:"required"`
	Fingerprint    string `json:"fingerprint"` // Machine hardware fingerprint (SHA-256)
	HeartbeatTime  int64  `json:"heartbeat_time"`
	ClientVersion  string `json:"client_version"`
	PID            int    `json:"process_id"`
	Status         string `json:"status"`
	LocalIP        string `json:"local_ip"`
	OSInfo         string `json:"os_info"` // e.g. "linux amd64"
	BusinessModule string `json:"business_module"`

	// ---- Master/Worker Topology (sent by ft-client) ----
	Role        string `json:"role"`         // "master" or "worker"
	ClusterID   string `json:"cluster_id"`   // Cluster identifier (string, converted to UUID server-side)
	ClusterName string `json:"cluster_name"` // Cluster display name

	TaskCount      int        `json:"task_count"`
	TaskLeft       int        `json:"task_left"`
	LastTaskTime   int64      `json:"last_task_time"`
	PrimaryHost    HostInfo   `json:"primary_host"`
	SecondaryHosts []HostInfo `json:"secondary_hosts"`
}

// clusterIDNamespace is a fixed UUID v5 namespace for deterministic cluster_id generation.
// cluster string ID -> UUID: uuid.NewSHA1(clusterIDNamespace, []byte("default-cluster"))
var clusterIDNamespace = uuid.MustParse("a3bb189e-8bf9-3888-9912-ace4e6543002")

// onlineTTL controls how quickly a machine is considered offline when heartbeats stop.
// Heartbeat interval is 5s by default; 20s means ~4 missed heartbeats before offline.
const onlineTTL = 20 * time.Second

// HostInfo represents a host's status report.
// Must stay in sync with ft-client/internal/model.HostInfo.
type HostInfo struct {
	IP               string  `json:"ip"`
	Hostname         string  `json:"hostname"`
	OSInfo           string  `json:"os_info"`
	OSVersion        string  `json:"os_version"`
	KernelVersion    string  `json:"kernel_version"`
	CPUCores         int     `json:"cpu_cores"`
	CPUUsage         float64 `json:"cpu_usage"`
	MemoryTotal      int64   `json:"memory_total"`
	MemoryUsed       int64   `json:"memory_used"`
	MemoryUsage      float64 `json:"memory_usage"`
	DiskTotal        int64   `json:"disk_total"`
	DiskUsed         int64   `json:"disk_used"`
	DiskUsage        float64 `json:"disk_usage"`
	NetworkDelay     int     `json:"network_delay"`
	NetworkInterface string  `json:"network_interface"`
	Status           string  `json:"status"`
	ProbeError       string  `json:"probe_error,omitempty"`
}

// HeartbeatResponse is the response returned to the client agent.
type HeartbeatResponse struct {
	Message               string           `json:"message"`
	Commands              []models.Command `json:"commands,omitempty"`
	Upgrade               *UpgradeInfo     `json:"upgrade,omitempty"` // non-nil means client should upgrade
	ExcludeSecondaryIPs   []string         `json:"exclude_secondary_ips,omitempty"` // IPs of workers user deleted; client should stop reporting them
}

// UpgradeInfo tells the client a newer version is available.
type UpgradeInfo struct {
	Version     string `json:"version"`
	DownloadURL string `json:"download_url"`
	Checksum    string `json:"checksum"`
	Force       bool   `json:"force"` // if true, client must upgrade immediately
}

// HeartbeatCheck receives client heartbeats and returns pending tasks.
// Implements the reverse heartbeat model with Redis-based decoupling:
//   - Parse heartbeat payload
//   - Enqueue to Redis for async processing (DB persist + machine upsert + WS broadcast)
//   - Synchronously fetch pending commands (needed for response)
//   - Return response immediately
//
// If Redis is unavailable, falls back to synchronous processing.
func HeartbeatCheck(c *gin.Context) {
	var clientInfo ClientHeartbeat
	if err := c.ShouldBindJSON(&clientInfo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的心跳数据", "error": err.Error()})
		return
	}

	logger.Debug("Heartbeat from client=%s ip=%s version=%s role=%s",
		clientInfo.ClientID, clientInfo.LocalIP, clientInfo.ClientVersion, clientInfo.Role)

	// Refresh online keys as soon as heartbeat is received to avoid
	// false offline caused by async consumer lag.
	refreshOnlineStatusFromHeartbeat(clientInfo)

	// 1. Enqueue to Redis for async processing (decoupled)
	if redis.IsConnected() {
		data, err := json.Marshal(clientInfo)
		if err == nil {
			if err := redis.EnqueueHeartbeat(data); err != nil {
				logger.Error("Failed to enqueue heartbeat, falling back to sync: %v", err)
				ProcessHeartbeatSync(clientInfo)
			}
		} else {
			ProcessHeartbeatSync(clientInfo)
		}
	} else {
		// Fallback: process synchronously when Redis is unavailable
		ProcessHeartbeatSync(clientInfo)
	}

	// 2. Fetch pending commands (stays synchronous — needed for response)
	commands := fetchPendingCommands(clientInfo.ClientID, clientInfo.LocalIP)
	if len(commands) > 0 {
		logger.Info("Sending %d command(s) to client: client_id=%s local_ip=%s (match by client_id or local_ip)",
			len(commands), clientInfo.ClientID, clientInfo.LocalIP)
	}

	// 3. Check if agent needs upgrade
	upgrade := checkAgentUpgrade(clientInfo)

	// 4. Tell client to stop reporting workers that the user has deleted (so they stay removed)
	excludeIPs := getExcludeSecondaryIPs(clientInfo.ClientID)

	response := HeartbeatResponse{
		Message:             "pong",
		Commands:            commands,
		Upgrade:             upgrade,
		ExcludeSecondaryIPs: excludeIPs,
	}

	c.JSON(http.StatusOK, response)
}

// ProcessHeartbeatSync processes a heartbeat synchronously (fallback when Redis is down).
func ProcessHeartbeatSync(info ClientHeartbeat) {
	persistHeartbeat(info)
	updateMachineFromHeartbeat(info)
	processSecondaryHosts(info)
}

func refreshOnlineStatusFromHeartbeat(info ClientHeartbeat) {
	if !redis.IsConnected() {
		return
	}
	if info.ClientID != "" {
		if err := redis.SetMachineOnline(info.ClientID, onlineTTL); err != nil {
			logger.Error("Failed to refresh machine online key from heartbeat: %v", err)
		}
	}
	for _, host := range info.SecondaryHosts {
		if host.IP != "" && host.Status == "up" {
			_ = redis.SetMachineOnline("worker:"+host.IP, onlineTTL)
		}
	}
}

// persistHeartbeat saves the heartbeat to the partitioned table.
func persistHeartbeat(info ClientHeartbeat) {
	primaryHostJSON, _ := json.Marshal(info.PrimaryHost)
	secondaryHostsJSON, _ := json.Marshal(info.SecondaryHosts)

	var lastTaskTime *time.Time
	if info.LastTaskTime > 0 {
		t := time.UnixMilli(info.LastTaskTime)
		lastTaskTime = &t
	}

	heartbeat := models.Heartbeat{
		ClientID:       info.ClientID,
		ClientVersion:  info.ClientVersion,
		ProcessID:      info.PID,
		Status:         info.Status,
		LocalIP:        info.LocalIP,
		BusinessModule: info.BusinessModule,
		TaskCount:      info.TaskCount,
		TaskLeft:       info.TaskLeft,
		LastTaskTime:   lastTaskTime,
		PrimaryHost:    models.JSONB(primaryHostJSON),
		SecondaryHosts: models.JSONB(secondaryHostsJSON),
	}

	if err := database.DB.Create(&heartbeat).Error; err != nil {
		logger.Error("Failed to save heartbeat for client=%s: %v", info.ClientID, err)
	}
}

// getExcludeSecondaryIPs returns IPs of workers that were soft-deleted by the user
// and belong to the master identified by clientID. The client should stop reporting
// these IPs in secondary_hosts so they stay removed. IPs that have been re-added
// (an active machine record exists for that IP) are not excluded.
func getExcludeSecondaryIPs(clientID string) []string {
	if clientID == "" {
		return nil
	}
	var master models.Machine
	if err := database.DB.Where("client_id = ?", clientID).First(&master).Error; err != nil {
		return nil
	}
	var deleted []models.Machine
	if err := database.DB.Unscoped().
		Where("master_machine_id = ? AND deleted_at IS NOT NULL", master.ID).
		Find(&deleted).Error; err != nil || len(deleted) == 0 {
		return nil
	}
	ips := make([]string, 0, len(deleted))
	for _, m := range deleted {
		if m.IP == "" {
			continue
		}
		// Do not exclude if the user has re-added this node (active record with same IP exists).
		var active models.Machine
		if err := database.DB.Where("ip = ?", m.IP).First(&active).Error; err == nil {
			continue
		}
		ips = append(ips, m.IP)
	}
	return ips
}

// fetchPendingCommands retrieves pending sub-tasks for a client.
// It first tries the Redis queue (fast path) then falls back to DB query.
// Regardless of source, corresponding DB records are marked as dispatched.
//
// Redis keys: tasks are enqueued to both client_id and IP. When client restarts
// with a new client_id, the DB may still have the old one; the task is in the
// IP queue. So we try both clientID and localIP queues.
func fetchPendingCommands(clientID string, localIP string) []models.Command {
	var commands []models.Command
	now := time.Now()

	// ---- Fast path: Redis queue ----
	// Try both clientID and localIP queues (tasks sent to either key when client_id
	// in DB differs from current client_id after restart).
	if redis.IsConnected() {
		redisKeys := []string{clientID}
		if localIP != "" && localIP != clientID {
			redisKeys = append(redisKeys, localIP)
		}
		seen := make(map[string]bool)
		for _, key := range redisKeys {
			redisCmds, err := redis.DequeueTask(key, 10)
			if err != nil || len(redisCmds) == 0 {
				continue
			}
			for _, raw := range redisCmds {
				var cmd models.Command
				if err := json.Unmarshal(raw, &cmd); err != nil {
					logger.Error("Failed to unmarshal Redis command: %v", err)
					continue
				}
				if seen[cmd.SubTaskID] {
					continue
				}
				seen[cmd.SubTaskID] = true
				commands = append(commands, cmd)

				// Mark the corresponding DB SubTask as dispatched
				if cmd.SubTaskID != "" {
					database.DB.Model(&models.SubTask{}).
						Where("id = ? AND status = ?", cmd.SubTaskID, string(models.TaskStatusPending)).
						Updates(map[string]interface{}{
							"status":     string(models.TaskStatusDispatched),
							"started_at": now,
						})
				}
				if cmd.TaskID != "" {
					database.DB.Model(&models.Task{}).
						Where("id = ? AND status = ?", cmd.TaskID, string(models.TaskStatusPending)).
						Updates(map[string]interface{}{
							"status":     string(models.TaskStatusDispatched),
							"started_at": now,
						})
				}

				logger.Info("Dispatched command=%s sub_task=%s to client=%s (via Redis)",
					cmd.Command, cmd.SubTaskID, clientID)
			}
		}
	}

	// ---- Slow path: DB query (picks up anything not yet in Redis) ----
	var subTasks []models.SubTask
	result := database.DB.Where(
		"(client_id = ? OR client_id = ?) AND status = ?",
		clientID, localIP, string(models.TaskStatusPending),
	).Order("created_at ASC").Limit(10).Find(&subTasks)

	if result.Error == nil && len(subTasks) > 0 {
		// Build a set of already-dispatched SubTask IDs from Redis to avoid duplicates
		dispatched := make(map[string]bool, len(commands))
		for _, cmd := range commands {
			dispatched[cmd.SubTaskID] = true
		}

		for _, st := range subTasks {
			if dispatched[st.ID.String()] {
				continue // Already dispatched via Redis
			}

			var task models.Task
			database.DB.Where("id = ?", st.TaskID).First(&task)

			cmd := models.Command{
				TaskID:    st.TaskID.String(),
				SubTaskID: st.ID.String(),
				Command:   st.Command,
				Payload:   json.RawMessage(st.Payload),
				Timeout:   task.TimeoutSec,
			}
			commands = append(commands, cmd)

			database.DB.Model(&st).Updates(map[string]interface{}{
				"status":     string(models.TaskStatusDispatched),
				"started_at": now,
			})

			if task.Status == string(models.TaskStatusPending) {
				database.DB.Model(&task).Updates(map[string]interface{}{
					"status":     string(models.TaskStatusDispatched),
					"started_at": now,
				})
			}

			logger.Info("Dispatched command=%s sub_task=%s to client=%s (via DB)",
				st.Command, st.ID.String(), clientID)
		}
	}

	return commands
}

// updateMachineFromHeartbeat auto-registers or updates a machine.
//
// Lookup priority:
//  1. host_fingerprint (hardware-level identity, survives IP/client_id changes)
//  2. client_id (agent-level identity)
//  3. IP (legacy fallback)
//
// On each heartbeat it also resolves topology: node_role, cluster_id, master_machine_id.
func updateMachineFromHeartbeat(info ClientHeartbeat) {
	ip := info.PrimaryHost.IP
	if ip == "" {
		ip = info.LocalIP
	}
	if ip == "" {
		return
	}

	now := time.Now()

	// ---- resolve topology fields ----
	nodeRole := resolveNodeRoleFromHeartbeat(info)
	clusterUUID := resolveClusterUUID(info.ClusterID)

	var machine models.Machine
	var found bool

	// 1) Lookup by fingerprint (most stable identity)
	if info.Fingerprint != "" {
		if err := database.DB.Where("host_fingerprint = ?", info.Fingerprint).First(&machine).Error; err == nil {
			found = true
		}
	}

	// 2) Lookup by client_id
	if !found && info.ClientID != "" {
		if err := database.DB.Where("client_id = ?", info.ClientID).First(&machine).Error; err == nil {
			found = true
		}
	}

	// 3) Fallback: lookup by IP
	if !found {
		if err := database.DB.Where("ip = ?", ip).First(&machine).Error; err == nil {
			found = true
		}
	}

	host := info.PrimaryHost

	if !found {
		// ---- Auto-register new machine ----
		if host.Hostname == "" {
			return
		}

		newMachine := models.Machine{
			Name:            host.Hostname,
			IP:              ip,
			ClientID:        info.ClientID,
			HostFingerprint: info.Fingerprint,
			Status:          "online",
			NodeRole:        nodeRole,
			ClusterID:       clusterUUID,
			LastHeartbeatAt: &now,
			// Persist real-time metrics to DB
			OSVersion:     host.OSVersion,
			KernelVersion: host.KernelVersion,
			CPUCores:      host.CPUCores,
			CPUUsage:      host.CPUUsage,
			MemoryTotal:   host.MemoryTotal,
			MemoryUsed:    host.MemoryUsed,
			MemoryUsage:   host.MemoryUsage,
			DiskTotal:     host.DiskTotal,
			DiskUsed:      host.DiskUsed,
			DiskUsage:     host.DiskUsage,
			Labels: models.NewJSONBFromMap(map[string]interface{}{
				"auto_registered": true,
				"os_info":         host.OSInfo,
			}),
			Metadata: models.NewJSONBFromMap(map[string]interface{}{
				"client_version":    info.ClientVersion,
				"network_interface": host.NetworkInterface,
				"cluster_name":      info.ClusterName,
			}),
		}

		if err := database.DB.Create(&newMachine).Error; err != nil {
			logger.Error("Failed to auto-register machine %s: %v", ip, err)
			return
		}
		logger.Info("Auto-registered machine: name=%s ip=%s client_id=%s role=%s cluster=%s fingerprint=%s",
			host.Hostname, ip, info.ClientID, nodeRole, info.ClusterID, shortFP(info.Fingerprint))

		// After creation, resolve master_machine_id for workers
		if nodeRole == models.NodeRoleWorker && clusterUUID != nil {
			resolveMasterLink(&newMachine, clusterUUID)
		}
		return
	}

	// ---- Update existing machine ----
	// Don't downgrade a manually-set master/worker role to standalone just because
	// the client config hasn't been updated yet.  The role can only be *promoted*
	// by the heartbeat (standalone → master when secondary_hosts arrive), never
	// silently demoted.  Explicit master/worker roles are preserved until the
	// client reports a non-standalone role itself.
	finalRole := nodeRole
	if nodeRole == models.NodeRoleStandalone &&
		(machine.NodeRole == models.NodeRoleMaster || machine.NodeRole == models.NodeRoleWorker) {
		finalRole = machine.NodeRole
	}

	updates := map[string]interface{}{
		"status":            "online",
		"last_heartbeat_at": now,
		"node_role":         finalRole,
		// Persist real-time metrics to DB on every heartbeat
		"os_version":     host.OSVersion,
		"kernel_version": host.KernelVersion,
		"cpu_cores":      host.CPUCores,
		"cpu_usage":      host.CPUUsage,
		"memory_total":   host.MemoryTotal,
		"memory_used":    host.MemoryUsed,
		"memory_usage":   host.MemoryUsage,
		"disk_total":     host.DiskTotal,
		"disk_used":      host.DiskUsed,
		"disk_usage":     host.DiskUsage,
	}

	// Always sync client_id from heartbeat — client may have restarted with a new ID.
	// This keeps DB in sync so future deploys use the correct client_id for Redis enqueue.
	if info.ClientID != "" {
		updates["client_id"] = info.ClientID
	}
	if machine.HostFingerprint == "" && info.Fingerprint != "" {
		updates["host_fingerprint"] = info.Fingerprint
	}

	// Keep IP in sync (client may change IP after DHCP renewal)
	if machine.IP != ip {
		updates["ip"] = ip
	}

	// Update name from hostname if auto-registered (may have been IP only)
	if host.Hostname != "" && machine.Name != host.Hostname {
		updates["name"] = host.Hostname
	}

	// Update cluster topology
	if clusterUUID != nil {
		updates["cluster_id"] = *clusterUUID
	}

	database.DB.Model(&machine).Updates(updates)

	// Resolve master link for workers (needs to happen after cluster_id is set)
	if nodeRole == models.NodeRoleWorker && clusterUUID != nil {
		resolveMasterLink(&machine, clusterUUID)
	}
	// If this machine is master, clear its own master_machine_id (self-referencing makes no sense)
	if nodeRole == models.NodeRoleMaster && machine.MasterMachineID != nil {
		database.DB.Model(&machine).Update("master_machine_id", nil)
	}
}

// resolveNodeRole normalizes the client-reported role to a valid model constant.
func resolveNodeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "master":
		return models.NodeRoleMaster
	case "worker":
		return models.NodeRoleWorker
	default:
		return models.NodeRoleStandalone
	}
}

// resolveNodeRoleFromHeartbeat derives a safer final node role from heartbeat semantics.
// If a node reports secondary hosts, it is the control node in practice and must be master.
func resolveNodeRoleFromHeartbeat(info ClientHeartbeat) string {
	role := resolveNodeRole(info.Role)
	if len(info.SecondaryHosts) > 0 && role != models.NodeRoleMaster {
		logger.Warn(
			"heartbeat role corrected to master due to secondary_hosts: client_id=%s raw_role=%s secondary_count=%d",
			info.ClientID, info.Role, len(info.SecondaryHosts),
		)
		return models.NodeRoleMaster
	}
	return role
}

// resolveClusterUUID converts a client-provided string cluster ID to a deterministic UUID v5.
// Returns nil if the cluster ID is empty.
func resolveClusterUUID(clusterID string) *uuid.UUID {
	clusterID = strings.TrimSpace(clusterID)
	if clusterID == "" {
		return nil
	}
	id := uuid.NewSHA1(clusterIDNamespace, []byte(clusterID))
	return &id
}

// resolveMasterLink looks up the master machine in the same cluster and sets master_machine_id on the worker.
func resolveMasterLink(worker *models.Machine, clusterUUID *uuid.UUID) {
	if clusterUUID == nil {
		return
	}

	var master models.Machine
	err := database.DB.
		Where("cluster_id = ? AND node_role = ? AND id != ?", *clusterUUID, models.NodeRoleMaster, worker.ID).
		First(&master).Error
	if err != nil {
		// Master not yet registered — will be linked on next heartbeat
		return
	}

	if worker.MasterMachineID == nil || *worker.MasterMachineID != master.ID {
		database.DB.Model(worker).Update("master_machine_id", master.ID)
		logger.Info("Linked worker %s -> master %s (cluster=%s)", worker.ID, master.ID, clusterUUID)
	}
}

// shortFP returns the first 12 chars of a fingerprint for logging.
func shortFP(fp string) string {
	if len(fp) > 12 {
		return fp[:12] + "..."
	}
	return fp
}

// processSecondaryHosts processes worker nodes reported by the master via SSH collection.
// For each secondary host, it upserts the machine in the DB.
func processSecondaryHosts(info ClientHeartbeat) {
	if len(info.SecondaryHosts) == 0 {
		return
	}

	clusterUUID := resolveClusterUUID(info.ClusterID)
	primaryIP := strings.TrimSpace(info.PrimaryHost.IP)
	localIP := strings.TrimSpace(info.LocalIP)
	primaryHostname := strings.TrimSpace(info.PrimaryHost.Hostname)

	for _, host := range info.SecondaryHosts {
		if host.IP == "" {
			continue
		}
		secondaryIP := strings.TrimSpace(host.IP)
		secondaryHostname := strings.TrimSpace(host.Hostname)

		// Guard 1: Never process master/self node as worker.
		// This can happen if client managed_nodes mistakenly includes itself.
		if (primaryIP != "" && secondaryIP == primaryIP) ||
			(localIP != "" && secondaryIP == localIP) ||
			(primaryHostname != "" && secondaryHostname != "" && secondaryHostname == primaryHostname) {
			logger.Warn(
				"skip self node in secondary_hosts to avoid role overwrite: client_id=%s secondary_ip=%s primary_ip=%s local_ip=%s secondary_hostname=%s primary_hostname=%s",
				info.ClientID, secondaryIP, primaryIP, localIP, secondaryHostname, primaryHostname,
			)
			continue
		}

		var machine models.Machine
		err := database.DB.Where("ip = ?", secondaryIP).First(&machine).Error

		now := time.Now()

		if err != nil {
			// Do not re-register if user has deleted this worker (soft-deleted same IP in same tenant).
			var deletedMachine models.Machine
			tenantID := models.MustParseUUID(models.DefaultTenantID)
			if errDeleted := database.DB.Unscoped().Where("ip = ? AND tenant_id = ? AND deleted_at IS NOT NULL", secondaryIP, tenantID).First(&deletedMachine).Error; errDeleted == nil {
				logger.Info("Skip re-registering deleted worker (user removed): ip=%s", secondaryIP)
				continue
			}

			// Auto-register worker machine from SSH collection
			hostname := host.Hostname
			if hostname == "" {
				hostname = host.IP
			}
			status := "online"
			if host.Status == "down" {
				status = "offline"
			}

			// Use unique client_id and host_fingerprint to satisfy unique constraints:
		// - idx_machines_tenant_client_id: (tenant_id, client_id) when client_id IS NOT NULL
		// - idx_machines_tenant_fingerprint: (tenant_id, host_fingerprint) when host_fingerprint IS NOT NULL
		// Empty strings would collide when multiple workers are auto-registered in the same tenant.
		workerIdentity := "worker:" + secondaryIP

		newMachine := models.Machine{
				Name:            hostname,
				IP:              secondaryIP,
				ClientID:        workerIdentity,
				HostFingerprint: workerIdentity,
				Status:          status,
				NodeRole:        models.NodeRoleWorker,
				ClusterID:       clusterUUID,
				LastHeartbeatAt: &now,
				// Persist worker metrics to DB
				OSVersion:     host.OSVersion,
				KernelVersion: host.KernelVersion,
				CPUCores:      host.CPUCores,
				CPUUsage:      host.CPUUsage,
				MemoryTotal:   host.MemoryTotal,
				MemoryUsed:    host.MemoryUsed,
				MemoryUsage:   host.MemoryUsage,
				DiskTotal:     host.DiskTotal,
				DiskUsed:      host.DiskUsed,
				DiskUsage:     host.DiskUsage,
				Labels: models.NewJSONBFromMap(map[string]interface{}{
					"auto_registered": true,
					"registered_by":   "master_ssh",
				}),
				Metadata: models.NewJSONBFromMap(map[string]interface{}{
					"cluster_name": info.ClusterName,
					"probe_status": host.Status,
					"probe_error":  strings.TrimSpace(host.ProbeError),
				}),
			}

			if err := database.DB.Create(&newMachine).Error; err != nil {
				logger.Error("Failed to register secondary host %s: %v", secondaryIP, err)
			} else {
				logger.Info("Registered secondary host: %s (%s) from master=%s", hostname, secondaryIP, info.ClientID)
				if clusterUUID != nil {
					resolveMasterLink(&newMachine, clusterUUID)
				}
			}
			continue
		}

		// Guard 2: existing machine is the heartbeat source itself -> never demote to worker.
		if (info.ClientID != "" && machine.ClientID == info.ClientID) ||
			(info.Fingerprint != "" && machine.HostFingerprint == info.Fingerprint) {
			logger.Warn(
				"skip secondary host update for primary machine identity match: machine_id=%s machine_ip=%s client_id=%s",
				machine.ID, machine.IP, info.ClientID,
			)
			continue
		}

		// Update existing worker machine with fresh metrics
		status := "online"
		if host.Status == "down" {
			status = "offline"
		}

		updates := map[string]interface{}{
			"status":            status,
			"last_heartbeat_at": now,
			"node_role":         models.NodeRoleWorker,
			// Persist worker metrics to DB on each heartbeat
			"os_version":     host.OSVersion,
			"kernel_version": host.KernelVersion,
			"cpu_cores":      host.CPUCores,
			"cpu_usage":      host.CPUUsage,
			"memory_total":   host.MemoryTotal,
			"memory_used":    host.MemoryUsed,
			"memory_usage":   host.MemoryUsage,
			"disk_total":     host.DiskTotal,
			"disk_used":      host.DiskUsed,
			"disk_usage":     host.DiskUsage,
			"metadata":       mergeWorkerProbeMetadata(machine.Metadata, info.ClusterName, host.Status, host.ProbeError),
		}

		// Update hostname if available
		if host.Hostname != "" && machine.Name != host.Hostname {
			updates["name"] = host.Hostname
		}

		if clusterUUID != nil {
			updates["cluster_id"] = *clusterUUID
		}

		database.DB.Model(&machine).Updates(updates)

		if clusterUUID != nil {
			resolveMasterLink(&machine, clusterUUID)
		}
	}
}

func mergeWorkerProbeMetadata(existing models.JSONB, clusterName, probeStatus, probeError string) models.JSONB {
	meta := map[string]interface{}{}
	if len(existing) > 0 {
		_ = json.Unmarshal(existing, &meta)
	}
	if clusterName != "" {
		meta["cluster_name"] = clusterName
	}
	meta["probe_status"] = probeStatus
	trimmedErr := strings.TrimSpace(probeError)
	if trimmedErr != "" {
		meta["probe_error"] = trimmedErr
	} else {
		delete(meta, "probe_error")
	}
	return models.NewJSONBFromMap(meta)
}

// ApplySyncNodesResult immediately upserts worker machine records and broadcasts a
// WebSocket event after the ft-client reports sync_nodes success.
//
// Unlike processSecondaryHosts (driven by heartbeats), this function has direct
// access to the master machine's DB record, so it can set cluster_id and
// master_machine_id precisely — no re-hashing of the cluster ID needed.
//
// workersJSON is the JSON-serialised []HostInfo payload embedded in
// CommandResult.Output by the ft-client.
func ApplySyncNodesResult(masterMachineID uuid.UUID, masterClientID string, workersJSON string) {
	// Load the master machine to get its true cluster_id and IP.
	var master models.Machine
	if err := database.DB.Where("id = ?", masterMachineID).First(&master).Error; err != nil {
		logger.Warn("ApplySyncNodesResult: master machine %s not found: %v", masterMachineID, err)
		return
	}

	var workers []HostInfo
	if err := json.Unmarshal([]byte(workersJSON), &workers); err != nil {
		logger.Warn("ApplySyncNodesResult: failed to parse workers JSON: %v", err)
		return
	}
	if len(workers) == 0 {
		return
	}

	now := time.Now()

	for _, w := range workers {
		if w.IP == "" {
			continue
		}
		// Never overwrite the master's own record.
		if w.IP == master.IP {
			continue
		}

		status := "online"
		if w.Status == "down" {
			status = "offline"
		}
		hostname := w.Hostname
		if hostname == "" {
			hostname = w.IP
		}

		var existing models.Machine
		err := database.DB.Where("ip = ?", w.IP).First(&existing).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			logger.Error("ApplySyncNodesResult: query worker %s failed: %v", w.IP, err)
			continue
		}
		if err == gorm.ErrRecordNotFound {
			// New worker — create it, inheriting the master's cluster topology.
			// Omit ClientID so the column is NULL in the DB: the partial unique index
			// (tenant_id, client_id) WHERE client_id IS NOT NULL must not fire for
			// workers that haven't run ft-client yet.
			newWorker := models.Machine{
				Name:            hostname,
				IP:              w.IP,
				Status:          status,
				NodeRole:        models.NodeRoleWorker,
				ClusterID:       master.ClusterID,
				MasterMachineID: &master.ID,
				LastHeartbeatAt: &now,
				OSVersion:       w.OSVersion,
				KernelVersion:   w.KernelVersion,
				CPUCores:        w.CPUCores,
				CPUUsage:        w.CPUUsage,
				MemoryTotal:     w.MemoryTotal,
				MemoryUsed:      w.MemoryUsed,
				MemoryUsage:     w.MemoryUsage,
				DiskTotal:       w.DiskTotal,
				DiskUsed:        w.DiskUsed,
				DiskUsage:       w.DiskUsage,
				Labels: models.NewJSONBFromMap(map[string]interface{}{
					"auto_registered": true,
					"registered_by":   "sync_nodes",
				}),
				Metadata: mergeWorkerProbeMetadata(nil, "", w.Status, w.ProbeError),
			}
			if err := database.DB.Omit("ClientID", "HostFingerprint").Create(&newWorker).Error; err != nil {
				logger.Error("ApplySyncNodesResult: create worker %s failed: %v", w.IP, err)
			} else {
				logger.Info("ApplySyncNodesResult: registered worker %s (%s)", hostname, w.IP)
			}
			continue
		}

		// Existing record — guard against overwriting the master itself.
		if existing.ID == master.ID {
			continue
		}

		updates := map[string]interface{}{
			"status":            status,
			"last_heartbeat_at": now,
			"node_role":         models.NodeRoleWorker,
			"master_machine_id": master.ID,
			"os_version":        w.OSVersion,
			"kernel_version":    w.KernelVersion,
			"cpu_cores":         w.CPUCores,
			"cpu_usage":         w.CPUUsage,
			"memory_total":      w.MemoryTotal,
			"memory_used":       w.MemoryUsed,
			"memory_usage":      w.MemoryUsage,
			"disk_total":        w.DiskTotal,
			"disk_used":         w.DiskUsed,
			"disk_usage":        w.DiskUsage,
			"metadata":          mergeWorkerProbeMetadata(existing.Metadata, "", w.Status, w.ProbeError),
		}
		if hostname != "" && existing.Name != hostname {
			updates["name"] = hostname
		}
		if master.ClusterID != nil {
			updates["cluster_id"] = *master.ClusterID
		}
		database.DB.Model(&existing).Updates(updates)
		logger.Info("ApplySyncNodesResult: updated worker %s (%s) status=%s", hostname, w.IP, status)
	}

	// Broadcast via WebSocket so the frontend updates immediately.
	if utils.GlobalWebSocketManager == nil {
		return
	}
	update := MachineUpdate{
		ClientID: masterClientID,
		IP:       master.IP,
		Hostname: master.Name,
		Status:   "online",
		NodeRole: models.NodeRoleMaster,
	}
	for _, w := range workers {
		update.Workers = append(update.Workers, WorkerStatus{
			IP:            w.IP,
			Hostname:      w.Hostname,
			Status:        w.Status,
			ProbeError:    w.ProbeError,
			OSVersion:     w.OSVersion,
			KernelVersion: w.KernelVersion,
			CPUCores:      w.CPUCores,
			CPUUsage:      w.CPUUsage,
			MemoryTotal:   w.MemoryTotal,
			MemoryUsed:    w.MemoryUsed,
			MemoryUsage:   w.MemoryUsage,
			DiskTotal:     w.DiskTotal,
			DiskUsed:      w.DiskUsed,
			DiskUsage:     w.DiskUsage,
		})
	}
	utils.GlobalWebSocketManager.Broadcast(utils.WebSocketMessage{
		Type:    "machine_heartbeat",
		Message: "Worker nodes updated via sync_nodes",
		Data:    update,
	})
}

// checkAgentUpgrade compares the client's version against the latest known version.
func checkAgentUpgrade(info ClientHeartbeat) *UpgradeInfo {
	if !redis.IsConnected() || info.ClientVersion == "" {
		return nil
	}

	// Parse OS/arch from the client's os_info field ("linux amd64")
	osArch := info.OSInfo
	if osArch == "" {
		osArch = info.PrimaryHost.OSInfo
	}
	parts := strings.SplitN(osArch, " ", 2)
	if len(parts) < 2 {
		return nil
	}
	clientOS, clientArch := parts[0], parts[1]

	latestVersion, err := redis.GetAgentVersion(clientOS, clientArch)
	if err != nil || latestVersion == nil {
		return nil
	}

	if latestVersion.Version != info.ClientVersion {
		logger.Info("Agent %s needs upgrade: %s -> %s", info.ClientID, info.ClientVersion, latestVersion.Version)
		return &UpgradeInfo{
			Version:     latestVersion.Version,
			DownloadURL: latestVersion.DownloadURL,
			Checksum:    latestVersion.Checksum,
			Force:       false,
		}
	}

	return nil
}
