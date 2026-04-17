// Package collector - SSH-based remote host metrics collection.
// Used by master nodes to collect metrics from managed worker nodes.
package collector

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"ft-client/internal/config"
	"ft-client/internal/logger"
	"ft-client/internal/model"

	"golang.org/x/crypto/ssh"
)

// sshCollectTimeout is the max time for a single SSH collection per node.
const sshCollectTimeout = 5 * time.Second

// CollectRemote collects host metrics from managed worker nodes via SSH.
// Failed nodes are returned with Status="down". Collection runs in parallel.
func CollectRemote(nodes []config.ManagedNode) []model.HostInfo {
	if len(nodes) == 0 {
		return nil
	}

	results := make([]model.HostInfo, len(nodes))
	var wg sync.WaitGroup

	for i, node := range nodes {
		wg.Add(1)
		go func(idx int, n config.ManagedNode) {
			defer wg.Done()
			results[idx] = collectFromNode(n)
		}(i, node)
	}

	wg.Wait()
	return results
}

// collectFromNode connects to a single worker via SSH and collects its metrics.
func collectFromNode(node config.ManagedNode) model.HostInfo {
	downHost := model.HostInfo{
		IP:     node.IP,
		Status: "down",
	}

	client, err := dialSSH(node)
	if err != nil {
		logger.Warn("SSH connect failed", "ip", node.IP, "error", err)
		return downHost
	}
	defer client.Close()

	// Run a single compound command to collect all metrics at once
	cmd := `hostname 2>/dev/null; echo "---SEP---"; ` +
		`cat /etc/os-release 2>/dev/null; echo "---SEP---"; ` +
		`uname -r 2>/dev/null; echo "---SEP---"; ` +
		`nproc 2>/dev/null; echo "---SEP---"; ` +
		`cat /proc/meminfo 2>/dev/null | head -3; echo "---SEP---"; ` +
		`df / 2>/dev/null | tail -1; echo "---SEP---"; ` +
		`cat /proc/stat 2>/dev/null | head -1`

	ctx, cancel := context.WithTimeout(context.Background(), sshCollectTimeout)
	defer cancel()

	output, err := runSSHCommand(ctx, client, cmd)
	if err != nil {
		logger.Warn("SSH command failed", "ip", node.IP, "error", err)
		return downHost
	}

	return parseRemoteOutput(node.IP, output)
}

// dialSSH creates an SSH client connection to the node.
func dialSSH(node config.ManagedNode) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	// Try SSH key authentication
	if node.SSHKey != "" {
		key, err := os.ReadFile(node.SSHKey)
		if err == nil {
			signer, err := ssh.ParsePrivateKey(key)
			if err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
			}
		}
	}

	// Try SSH agent (from SSH_AUTH_SOCK)
	// Omitted for simplicity -- key-based auth should cover most production use

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no valid SSH auth methods for %s (key: %s)", node.IP, node.SSHKey)
	}

	sshConfig := &ssh.ClientConfig{
		User:            node.SSHUser,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // internal fleet
		Timeout:         sshCollectTimeout,
	}

	addr := net.JoinHostPort(node.IP, strconv.Itoa(node.SSHPort))
	return ssh.Dial("tcp", addr, sshConfig)
}

// runSSHCommand runs a command on the SSH client and returns stdout.
func runSSHCommand(ctx context.Context, client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	// Run with context timeout
	doneCh := make(chan struct{})
	var output []byte
	var runErr error

	go func() {
		output, runErr = session.CombinedOutput(cmd)
		close(doneCh)
	}()

	select {
	case <-doneCh:
		if runErr != nil {
			// Some commands may fail (e.g., no /proc on some systems) but output is still useful
			if len(output) > 0 {
				return string(output), nil
			}
			return "", runErr
		}
		return string(output), nil
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGTERM)
		return "", ctx.Err()
	}
}

// parseRemoteOutput parses the compound SSH command output into a HostInfo.
// Sections are separated by "---SEP---".
func parseRemoteOutput(ip, output string) model.HostInfo {
	info := model.HostInfo{
		IP:     ip,
		OSInfo: fmt.Sprintf("%s %s", runtime.GOOS, runtime.GOARCH),
		Status: "up",
	}

	sections := strings.Split(output, "---SEP---")
	// Expected: [hostname, os-release, kernel, nproc, meminfo, df, cpustat]

	if len(sections) >= 1 {
		info.Hostname = strings.TrimSpace(sections[0])
	}

	if len(sections) >= 2 {
		info.OSVersion = parseOSRelease(sections[1])
	}

	if len(sections) >= 3 {
		info.KernelVersion = strings.TrimSpace(sections[2])
	}

	if len(sections) >= 4 {
		cores, _ := strconv.Atoi(strings.TrimSpace(sections[3]))
		info.CPUCores = cores
	}

	if len(sections) >= 5 {
		info.MemoryTotal, info.MemoryUsed = parseRemoteMeminfo(sections[4])
		if info.MemoryTotal > 0 {
			info.MemoryUsage = float64(info.MemoryUsed) / float64(info.MemoryTotal) * 100
		}
	}

	if len(sections) >= 6 {
		info.DiskTotal, info.DiskUsed = parseRemoteDf(sections[5])
		if info.DiskTotal > 0 {
			info.DiskUsage = float64(info.DiskUsed) / float64(info.DiskTotal) * 100
		}
	}

	if len(sections) >= 7 {
		info.CPUUsage = parseRemoteCPUStat(sections[6])
	}

	return info
}

// parseOSRelease extracts PRETTY_NAME from /etc/os-release content.
func parseOSRelease(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			val := strings.TrimPrefix(line, "PRETTY_NAME=")
			return strings.Trim(val, "\"")
		}
	}
	return ""
}

// parseRemoteMeminfo parses partial /proc/meminfo output (first 3 lines).
func parseRemoteMeminfo(content string) (total, used int64) {
	var memTotal, memAvailable int64
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "MemTotal:") {
			memTotal = parseMeminfoKB(line)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			memAvailable = parseMeminfoKB(line)
		}
	}
	return memTotal * 1024, (memTotal - memAvailable) * 1024
}

// parseRemoteDf parses the `df /` output line.
// Example: "/dev/sda1    100000000 50000000 50000000 50% /"
func parseRemoteDf(content string) (total, used int64) {
	line := strings.TrimSpace(content)
	if line == "" {
		return 0, 0
	}
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return 0, 0
	}
	// fields[1]=1K-blocks total, fields[2]=used
	t, _ := strconv.ParseInt(fields[1], 10, 64)
	u, _ := strconv.ParseInt(fields[2], 10, 64)
	return t * 1024, u * 1024 // df outputs in 1K blocks
}

// parseRemoteCPUStat parses a single /proc/stat "cpu" line to give instantaneous idle%.
// Note: single snapshot can only give idle ratio, not delta-based usage.
// For remote nodes this is an approximation.
func parseRemoteCPUStat(content string) float64 {
	line := strings.TrimSpace(content)
	if !strings.HasPrefix(line, "cpu ") {
		return 0
	}
	fields := strings.Fields(line)
	if len(fields) < 5 {
		return 0
	}
	var sum uint64
	for _, f := range fields[1:] {
		v, _ := strconv.ParseUint(f, 10, 64)
		sum += v
	}
	idleVal, _ := strconv.ParseUint(fields[4], 10, 64)
	if sum == 0 {
		return 0
	}
	return (1.0 - float64(idleVal)/float64(sum)) * 100
}
