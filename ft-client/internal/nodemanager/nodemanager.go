// Package nodemanager manages secondary (controlled) nodes for a master client.
// It maintains an in-memory registry of nodes, periodically probes their health,
// and provides thread-safe access to their current status.
package nodemanager

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"ft-client/internal/logger"
	"ft-client/internal/model"
)

// NodeConfig represents a single managed node's configuration.
type NodeConfig struct {
	IP          string `yaml:"ip"`
	Hostname    string `yaml:"hostname"`
	SSHPort     int    `yaml:"ssh_port"`
	SSHUser     string `yaml:"ssh_user"`
	AuthType    string `yaml:"auth_type"`    // "password" or "key" (default: "key")
	SSHPassword string `yaml:"ssh_password"` // SSH password (when AuthType=password)
	SSHKey      string `yaml:"ssh_key"`      // Path to SSH private key (when AuthType=key)
}

// NodeStatus represents the runtime status of a managed node.
type NodeStatus struct {
	Config    NodeConfig
	LastCheck time.Time
	Status    string // "up", "down", "degraded" (TCP ok but SSH failed), "unknown"

	// System information
	OSVersion     string // e.g. "Ubuntu 22.04", "CentOS 7.9"
	KernelVersion string // e.g. "5.15.0-91-generic"
	CPUCores      int    // Logical CPU count
	MemoryTotal   int64  // Total memory in GB (stored as bytes, displayed as GB)
	DiskTotal     int64  // Total disk in GB (stored as bytes, displayed as GB)

	// Runtime metrics
	CPUUsage    float64 // Percentage (0-100)
	MemoryUsage float64 // Percentage (0-100)
	MemoryUsed  int64   // Bytes
	DiskUsage   float64 // Percentage (0-100)
	DiskUsed    int64   // Bytes

	Latency int // Milliseconds
	Error   string
}

// Manager manages the lifecycle and status of secondary nodes.
type Manager struct {
	nodes       map[string]*NodeStatus // key: IP address
	mu          sync.RWMutex
	probeCtx    context.Context
	probeCancel context.CancelFunc
	probeWg     sync.WaitGroup

	// Configuration
	probeInterval time.Duration
	probeTimeout  time.Duration
}

// NewManager creates a new node manager.
func NewManager(probeInterval, probeTimeout time.Duration) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		nodes:         make(map[string]*NodeStatus),
		probeCtx:      ctx,
		probeCancel:   cancel,
		probeInterval: probeInterval,
		probeTimeout:  probeTimeout,
	}
}

// AddNode registers a new node to be managed.
// If the node already exists, it updates its configuration.
func (m *Manager) AddNode(config NodeConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if config.SSHPort == 0 {
		config.SSHPort = 22 // Default SSH port
	}

	m.nodes[config.IP] = &NodeStatus{
		Config:    config,
		LastCheck: time.Time{},
		Status:    "unknown",
		Latency:   0,
	}

	logger.Info("node registered",
		"ip", config.IP,
		"hostname", config.Hostname,
		"ssh_port", config.SSHPort,
	)
}

// RemoveNode unregisters a node from management.
// It returns true if the node was present and removed, false if it was already not managed.
func (m *Manager) RemoveNode(ip string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.nodes[ip]; !ok {
		return false
	}
	delete(m.nodes, ip)
	logger.Info("node unregistered", "ip", ip)
	return true
}

// GetNodes returns a snapshot of all managed nodes' current status.
// The returned slice is safe to use even after the manager is modified.
func (m *Manager) GetNodes() []model.HostInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]model.HostInfo, 0, len(m.nodes))
	for _, node := range m.nodes {
		result = append(result, model.HostInfo{
			IP:               node.Config.IP,
			Hostname:         node.Config.Hostname,
			OSInfo:           "linux amd64",
			OSVersion:        node.OSVersion,
			KernelVersion:    node.KernelVersion,
			CPUCores:         node.CPUCores,
			CPUUsage:         node.CPUUsage,
			MemoryTotal:      node.MemoryTotal,
			MemoryUsed:       node.MemoryUsed,
			MemoryUsage:      node.MemoryUsage,
			DiskTotal:        node.DiskTotal,
			DiskUsed:         node.DiskUsed,
			DiskUsage:        node.DiskUsage,
			NetworkDelay:     node.Latency,
			NetworkInterface: "eth0",
			Status:           node.Status,
			ProbeError:       node.Error,
		})
	}
	return result
}

// Start begins periodic health probing of all registered nodes.
// It runs until Stop() is called or the context is cancelled.
func (m *Manager) Start() {
	logger.Info("node manager starting",
		"probe_interval", m.probeInterval,
		"probe_timeout", m.probeTimeout,
		"node_count", len(m.nodes),
	)

	// Probe immediately on start
	m.probeAll()

	// Start periodic probing
	m.probeWg.Add(1)
	go m.probeLoop()
}

// Stop gracefully stops the node manager and waits for ongoing probes to complete.
func (m *Manager) Stop() {
	logger.Info("node manager stopping")
	m.probeCancel()
	m.probeWg.Wait()
	logger.Info("node manager stopped")
}

// probeLoop runs the periodic health check loop.
func (m *Manager) probeLoop() {
	defer m.probeWg.Done()

	ticker := time.NewTicker(m.probeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.probeAll()
		case <-m.probeCtx.Done():
			return
		}
	}
}

// probeAll probes the health of all registered nodes concurrently.
func (m *Manager) probeAll() {
	m.mu.RLock()
	nodes := make([]*NodeStatus, 0, len(m.nodes))
	for _, node := range m.nodes {
		nodes = append(nodes, node)
	}
	m.mu.RUnlock()

	if len(nodes) == 0 {
		return
	}

	logger.Debug("probing nodes", "count", len(nodes))

	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go func(ns *NodeStatus) {
			defer wg.Done()
			m.probeNode(ns)
		}(node)
	}
	wg.Wait()

	logger.Debug("node probe completed", "count", len(nodes))
}

// probeNode performs a health check on a single node via SSH.
// It updates the node's status in-place (caller must ensure no concurrent writes).
func (m *Manager) probeNode(node *NodeStatus) {
	ctx, cancel := context.WithTimeout(m.probeCtx, m.probeTimeout)
	defer cancel()

	startTime := time.Now()
	node.LastCheck = startTime

	hasSSHCreds := node.Config.SSHKey != "" || node.Config.SSHPassword != ""
	if hasSSHCreds {
		if err := m.probeNodeViaSSH(ctx, node); err != nil {
			// SSH failed — record the error and fall back to TCP to distinguish
			// "unreachable" from "reachable but SSH auth failed". If TCP succeeds,
			// report status as "degraded" so the frontend does not show the host
			// as fully "online" (metrics are 0 and SSH is unusable).
			sshErrMsg := fmt.Sprintf("ssh probe failed: %v", err)
			node.Error = sshErrMsg
			logger.Warn("ssh probe failed, metrics unavailable, falling back to tcp",
				"ip", node.Config.IP,
				"hostname", node.Config.Hostname,
				"auth_type", node.Config.AuthType,
				"ssh_user", node.Config.SSHUser,
				"error", err,
			)
			m.probeNodeViaTCP(ctx, node)
			// If TCP succeeded, probeNodeViaTCP set status=up and cleared Error.
			// Override to "degraded" and restore error so UI shows not-online and probe_error.
			if node.Status == "up" {
				node.Status = "degraded"
				node.Error = sshErrMsg
			}
		}
	} else {
		// No SSH credentials — only TCP reachability can be tested.
		// Metrics will be empty; register the node with credentials to collect them.
		logger.Warn("no ssh credentials configured, only checking tcp reachability",
			"ip", node.Config.IP,
			"hostname", node.Config.Hostname,
		)
		m.probeNodeViaTCP(ctx, node)
	}

	node.Latency = int(time.Since(startTime).Milliseconds())
}

// probeNodeViaTCP performs a simple TCP connection check.
func (m *Manager) probeNodeViaTCP(ctx context.Context, node *NodeStatus) {
	addr := fmt.Sprintf("%s:%d", node.Config.IP, node.Config.SSHPort)

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		node.Status = "down"
		node.Error = err.Error()
		logger.Warn("tcp probe failed",
			"ip", node.Config.IP,
			"hostname", node.Config.Hostname,
			"error", err,
		)
		return
	}
	defer conn.Close()

	node.Status = "up"
	node.Error = ""
	logger.Debug("tcp probe successful", "ip", node.Config.IP)
}

// probeNodeViaSSH performs SSH-based health check and metric collection.
func (m *Manager) probeNodeViaSSH(ctx context.Context, node *NodeStatus) error {
	client, err := m.createSSHClient(node)
	if err != nil {
		node.Status = "down"
		node.Error = fmt.Sprintf("ssh connection failed: %v", err)
		// Detailed connection error logged by the caller (probeNode) at Warn level.
		return err
	}
	defer client.Close()

	// SSH connection established — node is reachable.
	node.Status = "up"
	node.Error = ""

	// Collect system metrics over the established SSH channel.
	if err := m.collectSystemInfo(client, node); err != nil {
		// collectSystemInfo only returns an error for catastrophic failures;
		// per-command errors are handled internally and won't surface here.
		logger.Warn("system info collection failed",
			"ip", node.Config.IP,
			"hostname", node.Config.Hostname,
			"error", err,
		)
	}

	// Success — only log key results at Info; full detail at Debug.
	if node.OSVersion == "" || node.CPUCores == 0 {
		logger.Warn("ssh connected but metrics are empty, some commands may have failed",
			"ip", node.Config.IP,
			"hostname", node.Config.Hostname,
			"os_version", node.OSVersion,
			"cpu_cores", node.CPUCores,
		)
	} else {
		logger.Info("ssh probe ok",
			"ip", node.Config.IP,
			"os_version", node.OSVersion,
			"cpu_cores", node.CPUCores,
			"memory_gb", bytesToGB(node.MemoryTotal),
			"disk_gb", bytesToGB(node.DiskTotal),
		)
	}

	return nil
}

// createSSHClient establishes an SSH connection to a node.
// It supports both password and public-key authentication, and automatically
// accepts host keys on first connection (equivalent to ssh -o StrictHostKeyChecking=no).
func (m *Manager) createSSHClient(node *NodeStatus) (*ssh.Client, error) {
	// Build authentication methods based on configuration
	var authMethods []ssh.AuthMethod

	authType := node.Config.AuthType
	if authType == "" {
		// Auto-detect: if password is set use it, otherwise try key
		if node.Config.SSHPassword != "" {
			authType = "password"
		} else {
			authType = "key"
		}
	}

	switch authType {
	case "password":
		if node.Config.SSHPassword == "" {
			return nil, fmt.Errorf("password auth selected but no password configured")
		}
		authMethods = append(authMethods, ssh.Password(node.Config.SSHPassword))
		// Also try keyboard-interactive for hosts that require it (e.g. PAM)
		authMethods = append(authMethods, ssh.KeyboardInteractive(
			func(user, instruction string, questions []string, echos []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range questions {
					answers[i] = node.Config.SSHPassword
				}
				return answers, nil
			},
		))
	case "key":
		if node.Config.SSHKey == "" {
			return nil, fmt.Errorf("key auth selected but no key path configured")
		}
		signer, err := m.loadSSHKey(node.Config.SSHKey)
		if err != nil {
			return nil, fmt.Errorf("load ssh key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	default:
		return nil, fmt.Errorf("unsupported auth type: %s", authType)
	}

	config := &ssh.ClientConfig{
		User: node.Config.SSHUser,
		Auth: authMethods,
		// Automatically accept host keys on first connection.
		// This is equivalent to ssh -o StrictHostKeyChecking=no,
		// which handles the initial "yes/no" prompt without user interaction.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         m.probeTimeout,
	}

	addr := fmt.Sprintf("%s:%d", node.Config.IP, node.Config.SSHPort)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("ssh dial (%s auth): %w", authType, err)
	}

	return client, nil
}

// loadSSHKey loads an SSH private key from file.
func (m *Manager) loadSSHKey(keyPath string) (ssh.Signer, error) {
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	return signer, nil
}

// AddOrUpdateNode registers a node or updates its configuration if it already exists.
// This is safe to call while the probe loop is running.
func (m *Manager) AddOrUpdateNode(config NodeConfig) {
	if config.SSHPort == 0 {
		config.SSHPort = 22
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.nodes[config.IP]; ok {
		existing.Config = config
		logger.Info("node config updated", "ip", config.IP, "hostname", config.Hostname)
	} else {
		m.nodes[config.IP] = &NodeStatus{
			Config:  config,
			Status:  "unknown",
			Latency: 0,
		}
		logger.Info("node registered", "ip", config.IP, "hostname", config.Hostname, "ssh_port", config.SSHPort)
	}
}

// ProbeImmediate performs an immediate synchronous health-check for the specified
// IP addresses.  It blocks until all probes are done (or their SSH timeouts fire).
// Use this to get fresh metrics right after adding new nodes, without waiting for
// the next periodic probe interval.
func (m *Manager) ProbeImmediate(ips ...string) {
	if len(ips) == 0 {
		return
	}

	m.mu.RLock()
	targets := make([]*NodeStatus, 0, len(ips))
	for _, ip := range ips {
		if node, ok := m.nodes[ip]; ok {
			targets = append(targets, node)
		}
	}
	m.mu.RUnlock()

	if len(targets) == 0 {
		return
	}

	var wg sync.WaitGroup
	for _, node := range targets {
		wg.Add(1)
		go func(ns *NodeStatus) {
			defer wg.Done()
			m.probeNode(ns)
		}(node)
	}
	wg.Wait()

	logger.Info("immediate probe completed", "count", len(targets))
}

// GetNodeCount returns the total number of managed nodes.
func (m *Manager) GetNodeCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.nodes)
}

// GetOnlineCount returns the number of nodes currently in "up" status.
func (m *Manager) GetOnlineCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, node := range m.nodes {
		if node.Status == "up" {
			count++
		}
	}
	return count
}

// =============================================================================
// System Information Collection via SSH
// =============================================================================

// collectSystemInfo executes SSH commands to collect system information.
// Individual command failures are logged at Debug level (it is normal for some
// commands to be unavailable on certain distros); only a catastrophic error
// (e.g. all commands fail) is surfaced to the caller.
func (m *Manager) collectSystemInfo(client *ssh.Client, node *NodeStatus) error {
	ip := node.Config.IP

	// Helper: run a command and log failures at Debug level.
	run := func(cmd string) (string, bool) {
		out, err := m.executeSSHCommand(client, cmd)
		if err != nil {
			logger.Debug("ssh command failed",
				"ip", ip,
				"cmd", cmd,
				"error", err,
			)
			return "", false
		}
		return strings.TrimSpace(out), true
	}

	// 1. OS version
	if out, ok := run("lsb_release -ds 2>/dev/null || grep PRETTY_NAME /etc/os-release 2>/dev/null | cut -d'\"' -f2 || uname -s"); ok {
		node.OSVersion = simplifyOSVersion(out)
	}

	// 2. Kernel version
	if out, ok := run("uname -r"); ok {
		node.KernelVersion = out
	}

	// 3. CPU cores
	if out, ok := run("nproc"); ok {
		if cores, err := strconv.Atoi(out); err == nil {
			node.CPUCores = cores
		} else {
			logger.Debug("parse cpu cores failed", "ip", ip, "raw", out, "error", err)
		}
	}

	// 4. Memory total (KB → bytes)
	if out, ok := run("grep MemTotal /proc/meminfo | awk '{print $2}'"); ok {
		if kb, err := strconv.ParseInt(out, 10, 64); err == nil {
			node.MemoryTotal = kb * 1024
		} else {
			logger.Debug("parse memory total failed", "ip", ip, "raw", out, "error", err)
		}
	}

	// 5. Memory used
	if out, ok := run("free -b | awk '/^Mem:/{print $3}'"); ok {
		if used, err := strconv.ParseInt(out, 10, 64); err == nil {
			node.MemoryUsed = used
			if node.MemoryTotal > 0 {
				node.MemoryUsage = float64(used) / float64(node.MemoryTotal) * 100
			}
		} else {
			logger.Debug("parse memory used failed", "ip", ip, "raw", out, "error", err)
		}
	}

	// 6. Disk (root partition)
	if out, ok := run("df -B1 / | awk 'NR==2{print $2,$3,$5}'"); ok {
		parts := strings.Fields(out)
		if len(parts) >= 3 {
			if total, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
				node.DiskTotal = total
			}
			if used, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
				node.DiskUsed = used
			}
			if usageStr := strings.TrimSuffix(parts[2], "%"); usageStr != "" {
				if usage, err := strconv.ParseFloat(usageStr, 64); err == nil {
					node.DiskUsage = usage
				}
			}
		} else {
			logger.Debug("parse disk info failed", "ip", ip, "raw", out)
		}
	}

	// 7. CPU usage (1-second sample via mpstat or top fallback)
	if out, ok := run("mpstat 1 1 2>/dev/null | awk '/Average.*all/{printf \"%.1f\", 100-$NF}' || top -bn2 -d 0.5 2>/dev/null | grep 'Cpu(s)' | tail -1 | awk '{print $2}' | cut -d'%' -f1"); ok && out != "" {
		if usage, err := strconv.ParseFloat(out, 64); err == nil {
			node.CPUUsage = usage
		} else {
			logger.Debug("parse cpu usage failed", "ip", ip, "raw", out, "error", err)
		}
	}

	return nil
}

// executeSSHCommand executes a single command via SSH and returns its output.
func (m *Manager) executeSSHCommand(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return "", fmt.Errorf("execute command: %w", err)
	}

	return string(output), nil
}

// simplifyOSVersion simplifies OS version strings for cleaner display.
// Examples:
//   - "Ubuntu 22.04.3 LTS" -> "Ubuntu 22.04"
//   - "CentOS Linux 7 (Core)" -> "CentOS 7"
//   - "Red Hat Enterprise Linux Server 7.9" -> "RHEL 7.9"
func simplifyOSVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return "Unknown"
	}

	// Replace common long names
	replacements := map[string]string{
		"Red Hat Enterprise Linux": "RHEL",
		"CentOS Linux":             "CentOS",
	}
	for old, new := range replacements {
		version = strings.Replace(version, old, new, 1)
	}

	// Extract version number pattern (e.g., "22.04", "7.9", "8")
	parts := strings.Fields(version)
	simplified := ""
	foundVersion := false

	for i, part := range parts {
		// Keep the distribution name
		if i == 0 {
			simplified = part
			continue
		}

		// Extract version number (digits and dots)
		if !foundVersion && (strings.Contains(part, ".") || isDigit(part)) {
			// Clean up version (remove trailing chars)
			cleaned := strings.TrimRight(part, ",()[]")
			if len(cleaned) > 0 && (strings.Contains(cleaned, ".") || isDigit(cleaned)) {
				simplified += " " + cleaned
				foundVersion = true
				break
			}
		}
	}

	if simplified == "" {
		return version
	}
	return simplified
}

// bytesToGB converts bytes to gigabytes (rounded to 2 decimal places).
func bytesToGB(bytes int64) float64 {
	if bytes == 0 {
		return 0
	}
	gb := float64(bytes) / (1024 * 1024 * 1024)
	return float64(int(gb*100)) / 100 // Round to 2 decimals
}

// isDigit checks if a string contains only digits.
func isDigit(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
