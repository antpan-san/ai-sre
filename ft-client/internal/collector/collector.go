// Package collector gathers host machine resource information.
// Uses /proc and syscall for real system metrics on Linux.
// Falls back to Go runtime on non-Linux platforms (macOS dev).
package collector

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"net"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"ft-client/internal/model"
)

var (
	fingerprintOnce   sync.Once
	cachedFingerprint string
)

// Collect gathers the current machine's resource information using real system metrics.
func Collect() model.HostInfo {
	hostname, _ := os.Hostname()
	ip := getLocalIP()

	memTotal, memUsed := getMemoryInfo()
	var memUsage float64
	if memTotal > 0 {
		memUsage = float64(memUsed) / float64(memTotal) * 100
	}

	diskTotal, diskUsed := getDiskInfo()
	var diskUsage float64
	if diskTotal > 0 {
		diskUsage = float64(diskUsed) / float64(diskTotal) * 100
	}

	return model.HostInfo{
		IP:               ip,
		Hostname:         hostname,
		OSInfo:           fmt.Sprintf("%s %s", runtime.GOOS, runtime.GOARCH),
		OSVersion:        getOSVersion(),
		KernelVersion:    getKernelVersion(),
		CPUCores:         runtime.NumCPU(),
		CPUUsage:         getCPUUsage(),
		MemoryTotal:      memTotal,
		MemoryUsed:       memUsed,
		MemoryUsage:      memUsage,
		DiskTotal:        diskTotal,
		DiskUsed:         diskUsed,
		DiskUsage:        diskUsage,
		NetworkDelay:     0,
		NetworkInterface: getNetworkInterface(),
		Status:           "up",
	}
}

// =============================================================================
// OS / Kernel
// =============================================================================

// getOSVersion reads PRETTY_NAME from /etc/os-release (Linux).
func getOSVersion() string {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return runtime.GOOS
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			val := strings.TrimPrefix(line, "PRETTY_NAME=")
			return strings.Trim(val, "\"")
		}
	}
	return runtime.GOOS
}

// getKernelVersion reads /proc/version (Linux) or falls back to runtime info.
func getKernelVersion() string {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return ""
	}
	// Format: "Linux version 5.15.0-91-generic (buildd@...) ..."
	parts := strings.Fields(string(data))
	if len(parts) >= 3 {
		return parts[2]
	}
	return strings.TrimSpace(string(data))
}

// =============================================================================
// CPU Usage (from /proc/stat)
// =============================================================================

// getCPUUsage reads /proc/stat twice with a 200ms interval to compute real CPU usage %.
func getCPUUsage() float64 {
	idle1, total1 := readCPUStat()
	if total1 == 0 {
		return 0
	}

	time.Sleep(200 * time.Millisecond)

	idle2, total2 := readCPUStat()
	if total2 == 0 {
		return 0
	}

	idleDelta := float64(idle2 - idle1)
	totalDelta := float64(total2 - total1)
	if totalDelta <= 0 {
		return 0
	}

	usage := (1.0 - idleDelta/totalDelta) * 100
	if usage < 0 {
		return 0
	}
	return usage
}

// readCPUStat reads the aggregate "cpu" line from /proc/stat and returns idle, total.
func readCPUStat() (idle, total uint64) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 5 {
				return 0, 0
			}
			// fields: cpu user nice system idle iowait irq softirq steal ...
			var sum uint64
			for _, f := range fields[1:] {
				v, _ := strconv.ParseUint(f, 10, 64)
				sum += v
			}
			idleVal, _ := strconv.ParseUint(fields[4], 10, 64)
			return idleVal, sum
		}
	}
	return 0, 0
}

// =============================================================================
// Memory (from /proc/meminfo)
// =============================================================================

// getMemoryInfo parses /proc/meminfo and returns (total, used) in bytes.
// used = MemTotal - MemAvailable (consistent with `free` command).
func getMemoryInfo() (total, used int64) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		// Fallback for non-Linux (dev)
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return int64(m.Sys), int64(m.HeapInuse)
	}
	defer f.Close()

	var memTotal, memAvailable int64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			memTotal = parseMeminfoKB(line)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			memAvailable = parseMeminfoKB(line)
		}
		if memTotal > 0 && memAvailable > 0 {
			break
		}
	}

	return memTotal * 1024, (memTotal - memAvailable) * 1024
}

// parseMeminfoKB extracts the kB value from a /proc/meminfo line like "MemTotal:  16000000 kB".
func parseMeminfoKB(line string) int64 {
	fields := strings.Fields(line)
	if len(fields) >= 2 {
		v, _ := strconv.ParseInt(fields[1], 10, 64)
		return v
	}
	return 0
}

// =============================================================================
// Disk (syscall.Statfs)
// =============================================================================

// getDiskInfo returns (total, used) bytes for the root filesystem.
func getDiskInfo() (total, used int64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return 0, 0
	}
	blockSize := int64(stat.Bsize)
	total = int64(stat.Blocks) * blockSize
	free := int64(stat.Bavail) * blockSize // available to non-root
	used = total - free
	return total, used
}

// =============================================================================
// Network
// =============================================================================

// getLocalIP returns the primary non-loopback IPv4 address.
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}

// getNetworkInterface returns the name of the primary network interface.
func getNetworkInterface() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "unknown"
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil && !ipNet.IP.IsLoopback() {
				return iface.Name
			}
		}
	}
	return "eth0"
}

// =============================================================================
// Machine Fingerprint
// =============================================================================

// GenerateFingerprint creates a stable, unique machine fingerprint.
// The fingerprint is based on hostname + MAC addresses + OS/arch, ensuring:
//   - Same machine always generates the same fingerprint (idempotent)
//   - Different machines generate different fingerprints (unique)
//   - The fingerprint survives agent restarts and redeployments
//
// The result is cached after first computation (machine hardware doesn't change at runtime).
func GenerateFingerprint() string {
	fingerprintOnce.Do(func() {
		cachedFingerprint = computeFingerprint()
	})
	return cachedFingerprint
}

// computeFingerprint builds the fingerprint hash from hardware characteristics.
func computeFingerprint() string {
	hostname, _ := os.Hostname()
	macs := getMACAddresses()
	osInfo := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)

	// Build a deterministic string from machine attributes
	// Format: hostname|mac1,mac2,...|os-arch
	raw := fmt.Sprintf("%s|%s|%s", hostname, strings.Join(macs, ","), osInfo)

	// SHA-256 hash to produce a fixed-length, collision-resistant fingerprint
	hash := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", hash)
}

// getMACAddresses returns sorted, non-loopback MAC addresses of all network interfaces.
// Using all MACs (sorted) ensures the fingerprint is deterministic regardless of interface order.
func getMACAddresses() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return []string{"unknown"}
	}

	var macs []string
	for _, iface := range ifaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		mac := iface.HardwareAddr.String()
		if mac != "" {
			macs = append(macs, mac)
		}
	}

	if len(macs) == 0 {
		return []string{"no-mac"}
	}

	// Sort for deterministic output
	sort.Strings(macs)
	return macs
}
