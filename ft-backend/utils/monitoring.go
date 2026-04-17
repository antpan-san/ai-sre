package utils

import (
	"time"

	"ft-backend/common/logger"
	"ft-backend/common/redis"
	"ft-backend/database"
	"ft-backend/models"

	"github.com/google/uuid"
)

// StartMachineStatusMonitor periodically checks machine online status.
// When Redis is available, it uses Redis TTL-based online keys (set by the consumer).
// When Redis is down, it falls back to DB-based heartbeat age checking.
// Machines that have gone offline are updated in the DB and broadcast via WebSocket.
func StartMachineStatusMonitor() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	logger.Info("Machine status monitor started")

	for range ticker.C {
		checkMachineStatus()
	}
}

// checkMachineStatus checks online/offline status and broadcasts changes.
func checkMachineStatus() {
	if redis.IsConnected() {
		checkMachineStatusRedis()
	} else {
		checkMachineStatusDB()
	}
}

// checkMachineStatusRedis uses Redis online keys to determine which machines are offline.
// It marks machines as offline in the DB if their Redis key has expired.
func checkMachineStatusRedis() {
	onlineIDs, err := redis.GetOnlineMachineIDs()
	if err != nil {
		logger.Error("Failed to get online machines from Redis: %v", err)
		return
	}

	onlineSet := make(map[string]bool, len(onlineIDs))
	for _, id := range onlineIDs {
		onlineSet[id] = true
	}

	// Find machines currently marked as "online" in the DB
	var onlineMachines []models.Machine
	if err := database.DB.Where("status = ?", "online").Find(&onlineMachines).Error; err != nil {
		logger.Error("Failed to query online machines: %v", err)
		return
	}

	var changedMachines []models.Machine
	for _, m := range onlineMachines {
		key := m.ClientID
		if key == "" {
			key = "worker:" + m.IP // SSH-collected workers use IP-based keys
		}

		if !onlineSet[key] {
			// Double-check with direct GET to avoid false negatives from SCAN snapshots.
			if redis.IsMachineOnline(key) {
				continue
			}
			// If heartbeat was updated very recently, skip this round.
			if m.LastHeartbeatAt != nil && time.Since(*m.LastHeartbeatAt) < 10*time.Second {
				continue
			}
			// Machine's Redis key expired -> mark offline
			database.DB.Model(&m).Update("status", "offline")
			m.Status = "offline"
			logger.Info("Machine marked offline (Redis TTL expired): client_id=%s ip=%s", m.ClientID, m.IP)
			changedMachines = append(changedMachines, m)
		}
	}

	// Broadcast if any changes occurred
	if len(changedMachines) > 0 {
		broadcastMachineStatusChanges(changedMachines)
	}
}

// checkMachineStatusDB is the fallback when Redis is unavailable.
// It marks machines offline if they haven't heartbeated within 20 seconds.
func checkMachineStatusDB() {
	threshold := time.Now().Add(-20 * time.Second)

	var toOffline []models.Machine
	if err := database.DB.
		Where("status = ? AND (last_heartbeat_at IS NULL OR last_heartbeat_at < ?)", "online", threshold).
		Find(&toOffline).Error; err != nil {
		logger.Error("Failed to query stale online machines: %v", err)
		return
	}
	if len(toOffline) == 0 {
		return
	}

	result := database.DB.Model(&models.Machine{}).
		Where("id IN ?", idsFromMachines(toOffline)).
		Update("status", "offline")

	if result.RowsAffected > 0 {
		logger.Info("Marked %d machines offline (no heartbeat for 20s)", result.RowsAffected)
		for i := range toOffline {
			toOffline[i].Status = "offline"
		}
		broadcastMachineStatusChanges(toOffline)
	}
}

func idsFromMachines(items []models.Machine) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

// broadcastMachineStatusChanges broadcasts only changed machines via WebSocket.
func broadcastMachineStatusChanges(changed []models.Machine) {
	if GlobalWebSocketManager == nil {
		return
	}
	if len(changed) == 0 {
		return
	}

	msg := WebSocketMessage{
		Type:    "machine_status_update",
		Message: "Machine status updated",
		Data:    changed,
	}

	GlobalWebSocketManager.Broadcast(msg)
	logger.Debug("Broadcasted machine status update to %d machines", len(changed))
}
