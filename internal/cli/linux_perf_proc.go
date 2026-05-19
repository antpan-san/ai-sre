package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func linuxProcRoot() string {
	if v := strings.TrimSpace(os.Getenv("OPSFLEET_LINUX_PROC_ROOT")); v != "" {
		return v
	}
	return "/proc"
}

func procPath(parts ...string) string {
	all := append([]string{linuxProcRoot()}, parts...)
	return filepath.Join(all...)
}

func readProcFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func readProcLines(path string) ([]string, error) {
	b, err := readProcFile(path)
	if err != nil {
		return nil, err
	}
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, sc.Err()
}

type cpuStatSample struct {
	User, Nice, System, Idle, Iowait, Irq, Softirq, Steal uint64
}

func parseCPUStatLine(line string) (cpuStatSample, bool) {
	fields := strings.Fields(line)
	if len(fields) < 8 || fields[0] != "cpu" {
		return cpuStatSample{}, false
	}
	vals := make([]uint64, 0, 8)
	for _, f := range fields[1:8] {
		n, err := strconv.ParseUint(f, 10, 64)
		if err != nil {
			return cpuStatSample{}, false
		}
		vals = append(vals, n)
	}
	for len(vals) < 8 {
		vals = append(vals, 0)
	}
	return cpuStatSample{
		User: vals[0], Nice: vals[1], System: vals[2], Idle: vals[3],
		Iowait: vals[4], Irq: vals[5], Softirq: vals[6], Steal: vals[7],
	}, true
}

func readCPUStat() (cpuStatSample, error) {
	lines, err := readProcLines(procPath("stat"))
	if err != nil {
		return cpuStatSample{}, err
	}
	for _, ln := range lines {
		if s, ok := parseCPUStatLine(ln); ok {
			return s, nil
		}
	}
	return cpuStatSample{}, fmt.Errorf("cpu line not found in stat")
}

func cpuStatDelta(a, b cpuStatSample) (total, idle uint64) {
	sum := func(s cpuStatSample) (t, id uint64) {
		id = s.Idle + s.Iowait
		t = s.User + s.Nice + s.System + id + s.Irq + s.Softirq + s.Steal
		return
	}
	t1, i1 := sum(a)
	t2, i2 := sum(b)
	if t2 > t1 {
		return t2 - t1, i2 - i1
	}
	return 0, 0
}

func cpuUsagePct(a, b cpuStatSample) (user, system, iowait, steal, irq, softirq, idle float64) {
	total, idleDelta := cpuStatDelta(a, b)
	if total == 0 {
		return
	}
	pct := func(v uint64) float64 { return float64(v) / float64(total) * 100 }
	du := int64(b.User) - int64(a.User)
	dn := int64(b.Nice) - int64(a.Nice)
	ds := int64(b.System) - int64(a.System)
	di := int64(b.Idle) - int64(a.Idle)
	dio := int64(b.Iowait) - int64(a.Iowait)
	dst := int64(b.Steal) - int64(a.Steal)
	dir := int64(b.Irq) - int64(a.Irq)
	dso := int64(b.Softirq) - int64(a.Softirq)
	user = pct(uint64(max64(0, du+dn)))
	system = pct(uint64(max64(0, ds)))
	iowait = pct(uint64(max64(0, dio)))
	steal = pct(uint64(max64(0, dst)))
	irq = pct(uint64(max64(0, dir)))
	softirq = pct(uint64(max64(0, dso)))
	idle = float64(idleDelta) / float64(total) * 100
	_ = di
	return
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func countCPUCores() int {
	lines, err := readProcLines(procPath("stat"))
	if err != nil {
		return 1
	}
	n := 0
	for _, ln := range lines {
		f := strings.Fields(ln)
		if len(f) > 0 && strings.HasPrefix(f[0], "cpu") && f[0] != "cpu" {
			n++
		}
	}
	if n == 0 {
		if b, err := readProcFile(procPath("cpuinfo")); err == nil {
			for _, ln := range strings.Split(string(b), "\n") {
				if strings.HasPrefix(strings.TrimSpace(ln), "processor") {
					n++
				}
			}
		}
	}
	if n == 0 {
		return 1
	}
	return n
}

func parseLoadavg() (linuxPerfLoad, error) {
	b, err := readProcFile(procPath("loadavg"))
	if err != nil {
		return linuxPerfLoad{}, err
	}
	f := strings.Fields(string(b))
	if len(f) < 3 {
		return linuxPerfLoad{}, fmt.Errorf("invalid loadavg")
	}
	l1, _ := strconv.ParseFloat(f[0], 64)
	l5, _ := strconv.ParseFloat(f[1], 64)
	l15, _ := strconv.ParseFloat(f[2], 64)
	out := linuxPerfLoad{Load1: l1, Load5: l5, Load15: l15}
	if len(f) >= 4 && strings.Contains(f[3], "/") {
		parts := strings.SplitN(f[3], "/", 2)
		out.Running, _ = strconv.Atoi(parts[0])
		out.TotalTasks, _ = strconv.Atoi(parts[1])
	}
	return out, nil
}

func parseMeminfo() (map[string]int64, error) {
	lines, err := readProcLines(procPath("meminfo"))
	if err != nil {
		return nil, err
	}
	out := map[string]int64{}
	for _, ln := range lines {
		parts := strings.Fields(ln)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSuffix(parts[0], ":")
		v, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			continue
		}
		out[key] = v
	}
	return out, nil
}

type diskStatSample struct {
	ReadSectors, WriteSectors, IOTicks, WeightedIOTicks uint64
}

func parseDiskstats() (map[string]diskStatSample, error) {
	lines, err := readProcLines(procPath("diskstats"))
	if err != nil {
		return nil, err
	}
	out := map[string]diskStatSample{}
	for _, ln := range lines {
		f := strings.Fields(ln)
		if len(f) < 14 {
			continue
		}
		dev := f[2]
		if strings.HasPrefix(dev, "loop") || strings.HasPrefix(dev, "ram") {
			continue
		}
		readSec, _ := strconv.ParseUint(f[5], 10, 64)
		writeSec, _ := strconv.ParseUint(f[9], 10, 64)
		ioTicks, _ := strconv.ParseUint(f[12], 10, 64)
		weighted := uint64(0)
		if len(f) > 13 {
			weighted, _ = strconv.ParseUint(f[13], 10, 64)
		}
		out[dev] = diskStatSample{ReadSectors: readSec, WriteSectors: writeSec, IOTicks: ioTicks, WeightedIOTicks: weighted}
	}
	return out, nil
}

func diskIODelta(a, b diskStatSample, secs float64) (readBps, writeBps, ioTimePct float64) {
	if secs <= 0 {
		return
	}
	readBps = float64(b.ReadSectors-a.ReadSectors) * 512 / secs
	writeBps = float64(b.WriteSectors-a.WriteSectors) * 512 / secs
	if b.IOTicks >= a.IOTicks {
		ioTimePct = float64(b.IOTicks-a.IOTicks) / (secs * 10) // ms per second -> %
	}
	return
}

type procSnapshot struct {
	PID, PPID, Threads, OOMScore, FDCount int
	User, Comm, State, Cmdline, Cgroup    string
	UTime, STime, RSSPages, VMS           uint64
	ReadBytes, WriteBytes                 uint64
	StartTime                             uint64
}

func listProcPIDs() ([]int, error) {
	entries, err := os.ReadDir(linuxProcRoot())
	if err != nil {
		return nil, err
	}
	var pids []int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 {
			continue
		}
		pids = append(pids, pid)
	}
	return pids, nil
}

func readProcPIDStat(pid int) (procSnapshot, error) {
	var snap procSnapshot
	snap.PID = pid
	b, err := readProcFile(procPath(strconv.Itoa(pid), "stat"))
	if err != nil {
		return snap, err
	}
	content := string(b)
	// comm may contain spaces inside (...)
	closeIdx := strings.LastIndex(content, ")")
	if closeIdx < 0 {
		return snap, fmt.Errorf("invalid stat")
	}
	rest := strings.Fields(content[closeIdx+2:])
	if len(rest) < 22 {
		return snap, fmt.Errorf("short stat")
	}
	snap.Comm = strings.Trim(content[strings.Index(content, "(")+1:closeIdx], " ")
	snap.State = rest[0]
	snap.PPID, _ = strconv.Atoi(rest[1])
	snap.UTime, _ = strconv.ParseUint(rest[11], 10, 64)
	snap.STime, _ = strconv.ParseUint(rest[12], 10, 64)
	snap.VMS, _ = strconv.ParseUint(rest[20], 10, 64)
	snap.RSSPages, _ = strconv.ParseUint(rest[21], 10, 64)
	snap.StartTime, _ = strconv.ParseUint(rest[19], 10, 64)
	snap.Threads, _ = strconv.Atoi(rest[17])
	if st, err := readProcLines(procPath(strconv.Itoa(pid), "status")); err == nil {
		for _, ln := range st {
			if strings.HasPrefix(ln, "Uid:") {
				fields := strings.Fields(ln)
				if len(fields) >= 2 {
					snap.User = fields[1]
				}
			}
		}
	}
	if cmd, err := readProcFile(procPath(strconv.Itoa(pid), "cmdline")); err == nil {
		snap.Cmdline = strings.ReplaceAll(string(cmd), "\x00", " ")
		snap.Cmdline = strings.TrimSpace(snap.Cmdline)
	}
	if io, err := readProcLines(procPath(strconv.Itoa(pid), "io")); err == nil {
		for _, ln := range io {
			if strings.HasPrefix(ln, "read_bytes:") {
				snap.ReadBytes, _ = strconv.ParseUint(strings.TrimSpace(strings.TrimPrefix(ln, "read_bytes:")), 10, 64)
			}
			if strings.HasPrefix(ln, "write_bytes:") {
				snap.WriteBytes, _ = strconv.ParseUint(strings.TrimSpace(strings.TrimPrefix(ln, "write_bytes:")), 10, 64)
			}
		}
	}
	if fds, err := os.ReadDir(procPath(strconv.Itoa(pid), "fd")); err == nil {
		snap.FDCount = len(fds)
	}
	if b, err := readProcFile(procPath(strconv.Itoa(pid), "oom_score")); err == nil {
		snap.OOMScore, _ = strconv.Atoi(strings.TrimSpace(string(b)))
	}
	if cg, err := readProcFile(procPath(strconv.Itoa(pid), "cgroup")); err == nil {
		snap.Cgroup = strings.TrimSpace(string(cg))
		if len(snap.Cgroup) > 120 {
			snap.Cgroup = snap.Cgroup[:120] + "..."
		}
	}
	return snap, nil
}

func readHostUptime() float64 {
	b, err := readProcFile(procPath("uptime"))
	if err != nil {
		return 0
	}
	f := strings.Fields(string(b))
	if len(f) == 0 {
		return 0
	}
	v, _ := strconv.ParseFloat(f[0], 64)
	return v
}

func readKernelVersion() string {
	b, err := readProcFile(procPath("version"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func readPSI(kind string) map[string]string {
	path := procPath("pressure", kind)
	b, err := readProcFile(path)
	if err != nil {
		return nil
	}
	out := map[string]string{}
	for _, ln := range strings.Split(string(b), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		if i := strings.IndexByte(ln, ' '); i > 0 {
			out[ln[:i]] = strings.TrimSpace(ln[i:])
		} else {
			out["raw"] = ln
		}
	}
	return out
}

func collectPSI() map[string]any {
	out := map[string]any{}
	for _, k := range []string{"cpu", "memory", "io"} {
		if m := readPSI(k); len(m) > 0 {
			out[k] = m
		} else {
			out[k] = "unsupported"
		}
	}
	return out
}

func isPseudoFSType(fs string) bool {
	switch fs {
	case "proc", "sysfs", "tmpfs", "devtmpfs", "devpts", "cgroup", "cgroup2", "pstore", "securityfs", "debugfs", "tracefs", "bpf", "hugetlbfs", "mqueue", "configfs", "fusectl", "binfmt_misc":
		return true
	default:
		return false
	}
}

func collectDiskUsage() []linuxPerfDisk {
	lines, err := readProcLines(procPath("mounts"))
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var disks []linuxPerfDisk
	for _, ln := range lines {
		f := strings.Fields(ln)
		if len(f) < 3 {
			continue
		}
		mount := f[1]
		fs := f[2]
		if seen[mount] {
			continue
		}
		seen[mount] = true
		pseudo := isPseudoFSType(fs)
		if pseudo && fs != "overlay" {
			// still include overlay with pseudo flag per spec
			if fs != "overlay" {
				continue
			}
		}
		var st syscall.Statfs_t
		if err := syscall.Statfs(mount, &st); err != nil {
			continue
		}
		total := int64(st.Blocks) * int64(st.Bsize)
		free := int64(st.Bfree) * int64(st.Bsize)
		used := total - free
		usedPct := 0.0
		if total > 0 {
			usedPct = float64(used) / float64(total) * 100
		}
		inodePct := 0.0
		if st.Files > 0 {
			inodePct = float64(st.Files-st.Ffree) / float64(st.Files) * 100
		}
		disks = append(disks, linuxPerfDisk{
			Mount: mount, FSType: fs, TotalBytes: total, UsedBytes: used,
			UsedPct: usedPct, InodeUsedPct: inodePct, PseudoFS: pseudo,
		})
	}
	return disks
}

func procToLinuxProcess(s procSnapshot, cpuPct float64, readBps, writeBps float64, clkTck float64) linuxPerfProcess {
	uptime := readHostUptime()
	startSec := float64(s.StartTime) / clkTck
	runSec := uptime - startSec
	if runSec < 0 {
		runSec = 0
	}
	cmdline := s.Cmdline
	if cmdline == "" {
		cmdline = s.Comm
	}
	if len(cmdline) > 200 {
		cmdline = cmdline[:200] + "..."
	}
	return linuxPerfProcess{
		PID: s.PID, PPID: s.PPID, User: s.User, Comm: s.Comm, Cmdline: cmdline,
		State: s.State, Threads: s.Threads, FDCount: s.FDCount,
		CPUPercent: cpuPct, RSSBytes: int64(s.RSSPages) * 4096, VMSBytes: int64(s.VMS) * 1024,
		ReadBps: readBps, WriteBps: writeBps, OOMScore: s.OOMScore, UptimeSec: runSec, Cgroup: s.Cgroup,
	}
}

func sleepSample(d time.Duration) {
	if d < 500*time.Millisecond {
		d = 500 * time.Millisecond
	}
	time.Sleep(d)
}
