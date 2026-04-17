// Package heartbeat implements the periodic heartbeat service.
// It sends machine status to the server every N seconds and processes returned commands.
package heartbeat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ft-client/internal/collector"
	"ft-client/internal/config"
	"ft-client/internal/logger"
	"ft-client/internal/model"
	"ft-client/internal/nodemanager"
	"ft-client/internal/transport"
)

// CommandHandler is the interface for processing commands received from the server.
// Implement this interface to add new command types (shell, ansible, k8s deploy, etc.).
type CommandHandler interface {
	// Execute runs a command and returns the result.
	Execute(ctx context.Context, cmd model.Command) *model.CommandResult
}

// Service manages the periodic heartbeat loop and command dispatching.
type Service struct {
	cfg         *config.Config
	client      transport.ServerAPI
	handler     CommandHandler
	fingerprint string               // Stable machine fingerprint for idempotent server-side upsert
	nodeMgr     *nodemanager.Manager // Manages secondary nodes (master role only)

	// Task tracking
	taskCount int
	taskLeft  int
	lastTask  time.Time
	mu        sync.Mutex

	// Consecutive failure counter
	failures atomic.Int32
}

// NewService creates a new heartbeat service.
// It pre-computes the machine fingerprint (stable across restarts).
// If node management is enabled, it initializes and starts the node manager.
func NewService(cfg *config.Config, client transport.ServerAPI, handler CommandHandler) *Service {
	fp := collector.GenerateFingerprint()
	logger.Info("machine fingerprint generated",
		"fingerprint", fp,
		"role", cfg.Client.Role,
		"cluster_id", cfg.Cluster.ID,
		"cluster_name", cfg.Cluster.Name,
	)

	s := &Service{
		cfg:         cfg,
		client:      client,
		handler:     handler,
		fingerprint: fp,
	}

	// Initialize node manager for master role with managed nodes
	if cfg.NodeManager.Enabled {
		probeInterval := time.Duration(cfg.NodeManager.ProbeInterval) * time.Second
		probeTimeout := time.Duration(cfg.NodeManager.ProbeTimeout) * time.Second
		s.nodeMgr = nodemanager.NewManager(probeInterval, probeTimeout)

		// Register all managed nodes from config
		for _, node := range cfg.ManagedNodes {
			s.nodeMgr.AddNode(nodemanager.NodeConfig{
				IP:          node.IP,
				Hostname:    node.Hostname,
				SSHPort:     node.SSHPort,
				SSHUser:     node.SSHUser,
				AuthType:    node.AuthType,
				SSHPassword: node.SSHPassword,
				SSHKey:      node.SSHKey,
			})
		}

		logger.Info("node manager initialized",
			"managed_node_count", len(cfg.ManagedNodes),
			"probe_interval", probeInterval,
			"probe_timeout", probeTimeout,
		)
	}

	return s
}

// Run starts the heartbeat loop. It blocks until the context is cancelled.
func (s *Service) Run(ctx context.Context) error {
	interval := time.Duration(s.cfg.Heartbeat.Interval) * time.Second

	logger.Info("heartbeat service started",
		"interval_sec", s.cfg.Heartbeat.Interval,
		"server", s.cfg.Server.URL,
		"client_id", s.cfg.Client.ID,
		"role", s.cfg.Client.Role,
		"cluster_id", s.cfg.Cluster.ID,
		"fingerprint", s.fingerprint,
		"node_manager_enabled", s.nodeMgr != nil,
	)

	// Start node manager if enabled (master role)
	if s.nodeMgr != nil {
		s.nodeMgr.Start()
		defer s.nodeMgr.Stop()
	}

	// Send the first heartbeat immediately
	s.tick(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.tick(ctx)
		case <-ctx.Done():
			logger.Info("heartbeat service stopped")
			return nil
		}
	}
}

// tick performs a single heartbeat cycle: collect -> send -> process response.
func (s *Service) tick(ctx context.Context) {
	// 1. Collect host information
	hostInfo := collector.Collect()

	// 2. Collect secondary node status (if node manager is running)
	var secondaryHosts []model.HostInfo
	if s.nodeMgr != nil {
		secondaryHosts = s.nodeMgr.GetNodes()
		secondaryHosts = filterOutSelfFromSecondaryHosts(secondaryHosts, hostInfo)
		logger.Debug("collected secondary node status",
			"node_count", len(secondaryHosts),
			"online_count", s.nodeMgr.GetOnlineCount(),
		)
	}

	// 3. Build heartbeat request (includes role/cluster/fingerprint for topology)
	s.mu.Lock()
	req := &model.HeartbeatRequest{
		ClientID:       s.cfg.Client.ID,
		Fingerprint:    s.fingerprint,
		HeartbeatTime:  time.Now().UnixMilli(),
		ClientVersion:  s.cfg.Client.Version,
		ProcessID:      os.Getpid(),
		Status:         s.getClientStatus(),
		LocalIP:        hostInfo.IP,
		OSInfo:         hostInfo.OSInfo,
		BusinessModule: s.cfg.Client.BusinessModule,

		// Master/Worker topology fields
		Role:        s.cfg.Client.Role,
		ClusterID:   s.cfg.Cluster.ID,
		ClusterName: s.cfg.Cluster.Name,

		TaskCount:      s.taskCount,
		TaskLeft:       s.taskLeft,
		LastTaskTime:   s.lastTask.UnixMilli(),
		PrimaryHost:    hostInfo,
		SecondaryHosts: secondaryHosts,
	}
	s.mu.Unlock()

	// 4. Log detailed upload information before sending
	logger.Info("sending heartbeat to server",
		"client_id", req.ClientID,
		"role", req.Role,
		"cluster_id", req.ClusterID,
		"status", req.Status,
		"primary_host_ip", req.PrimaryHost.IP,
		"primary_host_hostname", req.PrimaryHost.Hostname,
		"primary_host_status", req.PrimaryHost.Status,
		"secondary_hosts_count", len(req.SecondaryHosts),
		"task_count", req.TaskCount,
		"task_left", req.TaskLeft,
	)

	// Log each secondary host's status with detailed system info
	if len(req.SecondaryHosts) > 0 {
		for i, host := range req.SecondaryHosts {
			// Convert bytes to GB for readability
			memoryTotalGB := float64(host.MemoryTotal) / (1024 * 1024 * 1024)
			memoryUsedGB := float64(host.MemoryUsed) / (1024 * 1024 * 1024)
			diskTotalGB := float64(host.DiskTotal) / (1024 * 1024 * 1024)
			diskUsedGB := float64(host.DiskUsed) / (1024 * 1024 * 1024)

			logger.Info("secondary host status",
				"index", i+1,
				"ip", host.IP,
				"hostname", host.Hostname,
				"status", host.Status,
				"os_version", host.OSVersion,
				"cpu_cores", host.CPUCores,
				"cpu_usage_percent", fmt.Sprintf("%.1f%%", host.CPUUsage),
				"memory_total_gb", fmt.Sprintf("%.2fG", memoryTotalGB),
				"memory_used_gb", fmt.Sprintf("%.2fG", memoryUsedGB),
				"memory_usage_percent", fmt.Sprintf("%.1f%%", host.MemoryUsage),
				"disk_total_gb", fmt.Sprintf("%.2fG", diskTotalGB),
				"disk_used_gb", fmt.Sprintf("%.2fG", diskUsedGB),
				"disk_usage_percent", fmt.Sprintf("%.1f%%", host.DiskUsage),
				"network_delay_ms", host.NetworkDelay,
			)
		}
	}

	// 5. Send heartbeat to server
	resp, err := s.client.SendHeartbeat(ctx, req)
	if err != nil {
		count := s.failures.Add(1)
		logger.Warn("heartbeat failed",
			"error", err,
			"consecutive_failures", count,
		)
		if int(count) >= s.cfg.Heartbeat.MaxFailures {
			logger.Error("heartbeat failures exceeded threshold",
				"max_failures", s.cfg.Heartbeat.MaxFailures,
			)
		}
		return
	}

	// Reset failure counter on success
	s.failures.Store(0)

	logger.Info("heartbeat sent successfully",
		"server_message", resp.Message,
		"commands_received", len(resp.Commands),
		"upgrade_available", resp.Upgrade != nil,
	)

	// 3.5. Stop reporting workers that the user deleted in the UI: remove from node manager
	// so we never manage or collect them again (as if the machine does not exist).
	if len(resp.ExcludeSecondaryIPs) > 0 && s.nodeMgr != nil {
		seen := make(map[string]struct{})
		for _, ip := range resp.ExcludeSecondaryIPs {
			ip = strings.TrimSpace(ip)
			if ip == "" {
				continue
			}
			if _, ok := seen[ip]; ok {
				continue
			}
			seen[ip] = struct{}{}
			if s.nodeMgr.RemoveNode(ip) {
				logger.Info("removed node from manager (user deleted in UI)", "ip", ip)
			}
		}
	}

	// 4. 被动接收的 server 指令：每条都打日志，便于排查未执行问题
	for i, cmd := range resp.Commands {
		logger.Info("[server command] received passive instruction from server",
			"index", i+1,
			"total", len(resp.Commands),
			"command_type", cmd.Command,
			"task_id", cmd.TaskID,
			"sub_task_id", cmd.SubTaskID,
			"timeout_sec", cmd.Timeout,
		)
		s.mu.Lock()
		s.taskCount++
		s.taskLeft++
		s.mu.Unlock()

		go s.handleCommand(ctx, cmd)
	}

	// 5. Handle upgrade notification
	if resp.Upgrade != nil {
		logger.Info("server notified upgrade available",
			"current_version", s.cfg.Client.Version,
			"new_version", resp.Upgrade.Version,
			"force", resp.Upgrade.Force,
		)
		// TODO(phase2): Implement self-upgrade mechanism
	}
}

// handleCommand executes a single command and reports the result.
// 所有由 server 被动下发的指令都会在此执行并打日志。
func (s *Service) handleCommand(ctx context.Context, cmd model.Command) {
	logger.Info("[server command] executing passive instruction from server",
		"command_type", cmd.Command,
		"task_id", cmd.TaskID,
		"sub_task_id", cmd.SubTaskID,
		"timeout_sec", cmd.Timeout,
	)

	var result *model.CommandResult

	switch cmd.Command {
	case model.CmdSyncNodes:
		result = s.handleSyncNodes(cmd)
	case model.CmdInstallK8s:
		result = s.handleInstallK8s(ctx, cmd)
	default:
		result = s.handler.Execute(ctx, cmd)
	}

	if err := s.client.ReportResult(ctx, result); err != nil {
		logger.Error("[server command] failed to report result to server",
			"sub_task_id", cmd.SubTaskID,
			"command_type", cmd.Command,
			"error", err,
		)
	} else {
		logger.Info("[server command] result reported to server",
			"sub_task_id", cmd.SubTaskID,
			"command_type", cmd.Command,
			"status", result.Status,
			"exit_code", result.ExitCode,
		)
	}

	// Update task tracking
	s.mu.Lock()
	s.taskLeft--
	s.lastTask = time.Now()
	s.mu.Unlock()
}

// handleSyncNodes processes the sync_nodes command:
//  1. Registers each worker node in the NodeManager (with SSH credentials).
//  2. Immediately probes every newly-added node via SSH to collect full system info.
//  3. Serialises the probe results into the CommandResult.Output (JSON) so the
//     server can upsert machine records right away — no heartbeat wait needed.
func (s *Service) handleSyncNodes(cmd model.Command) *model.CommandResult {
	result := &model.CommandResult{
		TaskID:    cmd.TaskID,
		SubTaskID: cmd.SubTaskID,
		ClientID:  s.cfg.Client.ID,
	}

	logger.Info("sync_nodes command received",
		"task_id", cmd.TaskID,
		"sub_task_id", cmd.SubTaskID,
	)

	// Parse payload
	var payload struct {
		Workers []syncNodeWorker `json:"workers"`
	}
	if err := json.Unmarshal(cmd.Payload, &payload); err != nil {
		logger.Error("sync_nodes: failed to parse payload",
			"task_id", cmd.TaskID,
			"error", err,
		)
		result.Status = model.StatusFailed
		result.ExitCode = 1
		result.Error = "invalid sync_nodes payload: " + err.Error()
		return result
	}

	logger.Info("sync_nodes: parsed worker list", "count", len(payload.Workers))

	if len(payload.Workers) == 0 {
		logger.Info("sync_nodes: no workers in payload, nothing to do")
		result.Status = model.StatusSuccess
		result.ExitCode = 0
		result.Output = `{"workers":[]}`
		return result
	}

	// -----------------------------------------------------------------------
	// Prepare NodeManager.
	// IMPORTANT: nodes must be added BEFORE Start() is called.
	// -----------------------------------------------------------------------
	justCreated := false
	if s.nodeMgr == nil {
		probeInterval := time.Duration(s.cfg.NodeManager.ProbeInterval) * time.Second
		probeTimeout := time.Duration(s.cfg.NodeManager.ProbeTimeout) * time.Second
		s.nodeMgr = nodemanager.NewManager(probeInterval, probeTimeout)
		justCreated = true
		logger.Info("sync_nodes: node manager created",
			"probe_interval_sec", s.cfg.NodeManager.ProbeInterval,
			"probe_timeout_sec", s.cfg.NodeManager.ProbeTimeout,
		)
	}

	// Register / update all worker nodes.
	var newIPs []string
	for _, w := range payload.Workers {
		if w.IP == "" {
			logger.Warn("sync_nodes: skipping worker with empty IP")
			continue
		}
		port := w.SSHPort
		if port == 0 {
			port = 22
		}
		user := w.SSHUser
		if user == "" {
			user = "root"
		}
		hostname := w.Hostname
		if hostname == "" {
			hostname = w.IP
		}
		authType := w.AuthType
		if authType == "" {
			if w.SSHPassword != "" {
				authType = "password"
			} else {
				authType = "key"
			}
		}

		// Validate credentials before registering so SSH errors are obvious.
		if authType == "password" && w.SSHPassword == "" {
			logger.Warn("sync_nodes: worker has password auth but empty password, ssh will fail",
				"ip", w.IP,
				"ssh_user", user,
			)
		}
		if authType == "key" && w.SSHKey == "" {
			logger.Warn("sync_nodes: worker has key auth but empty key path, ssh will fail",
				"ip", w.IP,
				"ssh_user", user,
			)
		}
		if authType == "key" && w.SSHKey != "" {
			logger.Debug("sync_nodes: registering worker with key auth",
				"ip", w.IP,
				"ssh_user", user,
				"key_path", w.SSHKey,
				"ssh_port", port,
			)
		} else {
			logger.Debug("sync_nodes: registering worker with password auth",
				"ip", w.IP,
				"ssh_user", user,
				"ssh_port", port,
			)
		}

		s.nodeMgr.AddOrUpdateNode(nodemanager.NodeConfig{
			IP:          w.IP,
			Hostname:    hostname,
			SSHPort:     port,
			SSHUser:     user,
			AuthType:    authType,
			SSHPassword: w.SSHPassword,
			SSHKey:      w.SSHKey,
		})
		newIPs = append(newIPs, w.IP)
	}

	logger.Info("sync_nodes: all workers registered, starting ssh probe",
		"probe_count", len(newIPs),
		"ips", newIPs,
	)

	// -----------------------------------------------------------------------
	// Immediately probe newly-registered nodes (synchronous — blocks until
	// all SSH connections complete or time out).
	// -----------------------------------------------------------------------
	if justCreated {
		s.nodeMgr.Start()
	} else {
		s.nodeMgr.ProbeImmediate(newIPs...)
	}

	logger.Info("sync_nodes: ssh probe finished, persisting config")

	// Persist SSH credentials to config so they survive a restart.
	s.persistManagedNodes(payload.Workers)

	// -----------------------------------------------------------------------
	// Collect probe results and embed them in the task result output.
	// -----------------------------------------------------------------------
	allNodes := s.nodeMgr.GetNodes()
	newIPSet := make(map[string]struct{}, len(newIPs))
	for _, ip := range newIPs {
		newIPSet[ip] = struct{}{}
	}
	var probeResults []model.HostInfo
	for _, n := range allNodes {
		if _, ok := newIPSet[n.IP]; ok {
			probeResults = append(probeResults, n)
		}
	}

	// Log per-node result summary so failures are immediately visible.
	for _, r := range probeResults {
		if r.OSVersion == "" || r.CPUCores == 0 {
			logger.Warn("sync_nodes: probe result incomplete (metrics empty)",
				"ip", r.IP,
				"hostname", r.Hostname,
				"status", r.Status,
				"os_version", r.OSVersion,
				"cpu_cores", r.CPUCores,
			)
		} else {
			logger.Info("sync_nodes: probe result ok",
				"ip", r.IP,
				"hostname", r.Hostname,
				"os_version", r.OSVersion,
				"cpu_cores", r.CPUCores,
				"memory_gb", fmt.Sprintf("%.2f", float64(r.MemoryTotal)/(1<<30)),
				"disk_gb", fmt.Sprintf("%.2f", float64(r.DiskTotal)/(1<<30)),
			)
		}
	}

	outputJSON, err := json.Marshal(map[string]interface{}{
		"workers": probeResults,
	})
	if err != nil {
		logger.Error("sync_nodes: failed to marshal probe results", "error", err)
		outputJSON = []byte(`{"workers":[]}`)
	}

	logger.Info("sync_nodes completed",
		"synced_count", len(newIPs),
		"probe_results", len(probeResults),
		"total_managed", s.nodeMgr.GetNodeCount(),
	)

	result.Status = model.StatusSuccess
	result.ExitCode = 0
	result.Output = string(outputJSON)
	return result
}

// syncNodeWorker is the payload structure for a single worker in sync_nodes command.
type syncNodeWorker struct {
	IP          string `json:"ip"`
	Hostname    string `json:"hostname"`
	SSHPort     int    `json:"ssh_port"`
	SSHUser     string `json:"ssh_user"`
	AuthType    string `json:"auth_type"`
	SSHPassword string `json:"ssh_password"`
	SSHKey      string `json:"ssh_key"`
}

// persistManagedNodes appends new workers to the local config file for persistence.
func (s *Service) persistManagedNodes(workers []syncNodeWorker) {
	// Build a set of existing IPs to avoid duplicates
	existingIPs := make(map[string]bool)
	for _, node := range s.cfg.ManagedNodes {
		existingIPs[node.IP] = true
	}

	var newNodes []config.ManagedNode
	for _, w := range workers {
		if existingIPs[w.IP] {
			// Update existing entry
			for i, n := range s.cfg.ManagedNodes {
				if n.IP == w.IP {
					s.cfg.ManagedNodes[i].Hostname = w.Hostname
					if w.SSHPort > 0 {
						s.cfg.ManagedNodes[i].SSHPort = w.SSHPort
					}
					if w.SSHUser != "" {
						s.cfg.ManagedNodes[i].SSHUser = w.SSHUser
					}
					if w.AuthType != "" {
						s.cfg.ManagedNodes[i].AuthType = w.AuthType
					}
					if w.SSHPassword != "" {
						s.cfg.ManagedNodes[i].SSHPassword = w.SSHPassword
					}
					if w.SSHKey != "" {
						s.cfg.ManagedNodes[i].SSHKey = w.SSHKey
					}
					break
				}
			}
			continue
		}
		port := w.SSHPort
		if port == 0 {
			port = 22
		}
		user := w.SSHUser
		if user == "" {
			user = "root"
		}
		authType := w.AuthType
		if authType == "" {
			if w.SSHPassword != "" {
				authType = "password"
			} else {
				authType = "key"
			}
		}
		newNodes = append(newNodes, config.ManagedNode{
			IP:          w.IP,
			Hostname:    w.Hostname,
			SSHPort:     port,
			SSHUser:     user,
			AuthType:    authType,
			SSHPassword: w.SSHPassword,
			SSHKey:      w.SSHKey,
		})
	}
	s.cfg.ManagedNodes = append(s.cfg.ManagedNodes, newNodes...)

	// Enable node manager if not already
	s.cfg.NodeManager.Enabled = true

	// Write back to config file
	if err := config.Save(s.cfg); err != nil {
		logger.Warn("failed to persist managed_nodes to config file", "error", err)
	} else {
		logger.Info("managed_nodes persisted to config file", "total", len(s.cfg.ManagedNodes))
	}
}

// getClientStatus returns the current client status based on task load.
func (s *Service) getClientStatus() string {
	if s.taskLeft > 5 {
		return model.ClientStatusBusy
	}
	if s.failures.Load() > 0 {
		return model.ClientStatusDegraded
	}
	return model.ClientStatusNormal
}

// filterOutSelfFromSecondaryHosts removes "self" entries from secondary hosts.
// This prevents a master node from being mistakenly reported as a worker node.
func filterOutSelfFromSecondaryHosts(hosts []model.HostInfo, primary model.HostInfo) []model.HostInfo {
	if len(hosts) == 0 {
		return hosts
	}
	result := make([]model.HostInfo, 0, len(hosts))
	for _, h := range hosts {
		// If IP matches primary IP, this is self and must not be reported as secondary.
		if h.IP != "" && primary.IP != "" && h.IP == primary.IP {
			logger.Warn("ignoring self node in secondary_hosts",
				"ip", h.IP,
				"hostname", h.Hostname,
			)
			continue
		}
		// Additional safety: same hostname + missing/identical IP treated as self.
		if primary.Hostname != "" && h.Hostname != "" && h.Hostname == primary.Hostname {
			logger.Warn("ignoring self node in secondary_hosts by hostname",
				"hostname", h.Hostname,
			)
			continue
		}
		result = append(result, h)
	}
	return result
}

// =============================================================================
// StubHandler - Phase 1 placeholder command handler
// =============================================================================

// StubHandler is a placeholder CommandHandler for Phase 1.
// It logs received commands but does not actually execute them.
// Replace with real implementations (ShellHandler, AnsibleHandler, etc.) in Phase 2.
type StubHandler struct {
	ClientID string
}

// Execute logs the command and returns a "not implemented" result.
func (h *StubHandler) Execute(_ context.Context, cmd model.Command) *model.CommandResult {
	logger.Warn("[server command] stub handler: instruction not implemented (Phase 1)",
		"command_type", cmd.Command,
		"task_id", cmd.TaskID,
		"sub_task_id", cmd.SubTaskID,
	)

	return &model.CommandResult{
		TaskID:    cmd.TaskID,
		SubTaskID: cmd.SubTaskID,
		ClientID:  h.ClientID,
		Status:    model.StatusFailed,
		ExitCode:  -1,
		Error:     "command execution not implemented (Phase 1)",
	}
}
