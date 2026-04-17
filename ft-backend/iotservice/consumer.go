package iotservice

import (
	"context"
	"encoding/json"
	"time"

	"ft-backend/common/logger"
	"ft-backend/common/redis"
	"ft-backend/utils"
)

// StartHeartbeatConsumer starts a background goroutine that processes heartbeats
// from the Redis queue. This decouples heartbeat reception from DB writes and
// WebSocket broadcasting, ensuring the HeartbeatCheck handler responds quickly.
//
// Call this from main.go after Redis is connected.
func StartHeartbeatConsumer(ctx context.Context) {
	logger.Info("Heartbeat consumer started (Redis queue: %s)", redis.QueueHeartbeat)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Heartbeat consumer stopped")
			return
		default:
		}

		// Block for up to 5 seconds waiting for a heartbeat
		data, err := redis.DequeueHeartbeat(5 * time.Second)
		if err != nil {
			logger.Error("Heartbeat consumer dequeue error: %v", err)
			time.Sleep(1 * time.Second) // back off on error
			continue
		}

		if data == nil {
			// Timeout — no heartbeat in queue, loop back
			continue
		}

		// Parse the heartbeat
		var info ClientHeartbeat
		if err := json.Unmarshal(data, &info); err != nil {
			logger.Error("Heartbeat consumer unmarshal error: %v", err)
			continue
		}

		// 1. Persist heartbeat record to DB
		persistHeartbeat(info)

		// 2. Upsert machine from heartbeat
		updateMachineFromHeartbeat(info)

		// 3. Refresh online status in Redis.
		if info.ClientID != "" {
			if err := redis.SetMachineOnline(info.ClientID, onlineTTL); err != nil {
				logger.Error("Failed to set machine online status: %v", err)
			}
			_ = redis.SetMachineMetrics(info.ClientID, redis.MachineMetrics{
				OSVersion:     info.PrimaryHost.OSVersion,
				KernelVersion: info.PrimaryHost.KernelVersion,
				CPUCores:      info.PrimaryHost.CPUCores,
				CPUUsage:      info.PrimaryHost.CPUUsage,
				MemoryTotal:   info.PrimaryHost.MemoryTotal,
				MemoryUsed:    info.PrimaryHost.MemoryUsed,
				MemoryUsage:   info.PrimaryHost.MemoryUsage,
				DiskTotal:     info.PrimaryHost.DiskTotal,
				DiskUsed:      info.PrimaryHost.DiskUsed,
				DiskUsage:     info.PrimaryHost.DiskUsage,
			}, onlineTTL)
		}

		// 4. Process secondary hosts (workers reported by master)
		processSecondaryHosts(info)

		// 5. Set online status for secondary hosts
		for _, host := range info.SecondaryHosts {
			if host.IP != "" && host.Status == "up" {
				// Use IP as identifier for SSH-collected workers (they may not have a client_id)
				workerKey := "worker:" + host.IP
				_ = redis.SetMachineOnline(workerKey, onlineTTL)
				_ = redis.SetMachineMetrics(workerKey, redis.MachineMetrics{
					OSVersion:   host.OSVersion,
					CPUCores:    host.CPUCores,
					CPUUsage:    host.CPUUsage,
					MemoryUsage: host.MemoryUsage,
					DiskUsage:   host.DiskUsage,
				}, onlineTTL)
			}
		}

		// 6. Broadcast via WebSocket for real-time frontend updates
		broadcastMachineStatus(info)
	}
}

// MachineStatusEvent is the WebSocket event sent to frontend clients.
type MachineStatusEvent struct {
	Type      string        `json:"type"`
	Timestamp int64         `json:"timestamp"`
	Data      MachineUpdate `json:"data"`
}

// MachineUpdate contains the machine status data for WebSocket broadcast.
type MachineUpdate struct {
	ClientID      string         `json:"client_id"`
	IP            string         `json:"ip"`
	Hostname      string         `json:"hostname"`
	Status        string         `json:"status"`
	NodeRole      string         `json:"node_role"`
	ClusterID     string         `json:"cluster_id"`
	OSVersion     string         `json:"os_version"`
	KernelVersion string         `json:"kernel_version"`
	CPUCores      int            `json:"cpu_cores"`
	CPUUsage      float64        `json:"cpu_usage"`
	MemoryTotal   int64          `json:"memory_total"`
	MemoryUsed    int64          `json:"memory_used"`
	MemoryUsage   float64        `json:"memory_usage"`
	DiskTotal     int64          `json:"disk_total"`
	DiskUsed      int64          `json:"disk_used"`
	DiskUsage     float64        `json:"disk_usage"`
	Workers       []WorkerStatus `json:"workers,omitempty"`
}

// WorkerStatus represents a worker node's status in the WebSocket event.
type WorkerStatus struct {
	IP            string  `json:"ip"`
	Hostname      string  `json:"hostname"`
	Status        string  `json:"status"`
	ProbeError    string  `json:"probe_error,omitempty"`
	OSVersion     string  `json:"os_version"`
	KernelVersion string  `json:"kernel_version"`
	CPUCores      int     `json:"cpu_cores"`
	CPUUsage      float64 `json:"cpu_usage"`
	MemoryTotal   int64   `json:"memory_total"`
	MemoryUsed    int64   `json:"memory_used"`
	MemoryUsage   float64 `json:"memory_usage"`
	DiskTotal     int64   `json:"disk_total"`
	DiskUsed      int64   `json:"disk_used"`
	DiskUsage     float64 `json:"disk_usage"`
}

// broadcastMachineStatus broadcasts the heartbeat data via WebSocket to all connected frontends.
func broadcastMachineStatus(info ClientHeartbeat) {
	if utils.GlobalWebSocketManager == nil {
		return
	}

	host := info.PrimaryHost

	update := MachineUpdate{
		ClientID:      info.ClientID,
		IP:            host.IP,
		Hostname:      host.Hostname,
		Status:        "online",
		NodeRole:      info.Role,
		ClusterID:     info.ClusterID,
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
	}

	// Include worker status for master nodes
	for _, w := range info.SecondaryHosts {
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

	msg := utils.WebSocketMessage{
		Type:    "machine_heartbeat",
		Message: "Machine heartbeat update",
		Data:    update,
	}

	utils.GlobalWebSocketManager.Broadcast(msg)
}
