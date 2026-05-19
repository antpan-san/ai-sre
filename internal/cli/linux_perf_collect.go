package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	linuxPerfMinDuration = 3 * time.Second
	linuxPerfMaxDuration = 60 * time.Second
	linuxPerfDefaultDuration = 10 * time.Second
	linuxPerfMinTop = 5
	linuxPerfMaxTop = 30
	linuxPerfDefaultTop = 10
)

// ValidateLinuxPerfOptions checks duration/top/pid bounds.
func ValidateLinuxPerfOptions(opts LinuxPerfOptions) error {
	if opts.Duration < linuxPerfMinDuration || opts.Duration > linuxPerfMaxDuration {
		return fmt.Errorf("--duration 须在 %s 到 %s 之间", linuxPerfMinDuration, linuxPerfMaxDuration)
	}
	if opts.TopN < linuxPerfMinTop || opts.TopN > linuxPerfMaxTop {
		return fmt.Errorf("--top 须在 %d 到 %d 之间", linuxPerfMinTop, linuxPerfMaxTop)
	}
	if opts.PID < 0 {
		return fmt.Errorf("--pid 须为非负整数")
	}
	return nil
}

func normalizeLinuxPerfOptions(opts *LinuxPerfOptions) {
	if opts.Duration <= 0 {
		opts.Duration = linuxPerfDefaultDuration
	}
	if opts.TopN <= 0 {
		opts.TopN = linuxPerfDefaultTop
	}
}

// CollectLinuxPerf gathers read-only Linux host performance evidence.
func CollectLinuxPerf(opts LinuxPerfOptions) *LinuxPerfReport {
	normalizeLinuxPerfOptions(&opts)
	report := &LinuxPerfReport{
		Topic:                "linux",
		PSI:                  map[string]any{},
		EvidenceCompleteness: map[string]bool{},
	}
	if err := ValidateLinuxPerfOptions(opts); err != nil {
		report.Errors = append(report.Errors, err.Error())
		return report
	}
	if runtime.GOOS != "linux" && strings.TrimSpace(os.Getenv("OPSFLEET_LINUX_PROC_ROOT")) == "" {
		report.Errors = append(report.Errors, "Linux 性能诊断仅支持 Linux 主机（当前 GOOS="+runtime.GOOS+"）")
		return report
	}
	if _, err := os.Stat(linuxProcRoot()); err != nil {
		report.Errors = append(report.Errors, "无法访问 proc 根目录: "+linuxProcRoot())
		return report
	}

	started := time.Now()
	report.Sample = linuxPerfSample{
		DurationSeconds: opts.Duration.Seconds(),
		StartedAt:       started,
		TopN:            opts.TopN,
		TargetPID:       opts.PID,
	}

	report.Host.Hostname, _ = os.Hostname()
	report.Host.Kernel = readKernelVersion()
	report.Host.UptimeSeconds = readHostUptime()
	report.Host.CPUCores = countCPUCores()
	report.EvidenceCompleteness["host"] = true

	clkTck := 100.0
	if b, err := readProcFile(procPath("uptime")); err == nil && len(strings.Fields(string(b))) > 0 {
		_ = b
	}

	interval := opts.Duration / 2
	if interval < 500*time.Millisecond {
		interval = 500 * time.Millisecond
	}

	if load, err := parseLoadavg(); err == nil {
		report.Load = load
		report.EvidenceCompleteness["load"] = true
	} else {
		report.Errors = append(report.Errors, "loadavg: "+err.Error())
	}

	memKV, err := parseMeminfo()
	if err == nil {
		fillMemoryReport(report, memKV)
		report.EvidenceCompleteness["memory"] = true
	} else {
		report.Errors = append(report.Errors, "meminfo: "+err.Error())
	}

	report.Disks = collectDiskUsage()
	if len(report.Disks) > 0 {
		report.EvidenceCompleteness["disks"] = true
	}

	report.PSI = collectPSI()
	report.EvidenceCompleteness["psi"] = true

	cpu1, err := readCPUStat()
	if err != nil {
		report.Errors = append(report.Errors, "cpu stat: "+err.Error())
	}
	disk1, _ := parseDiskstats()
	pids, _ := listProcPIDs()
	snap1 := snapshotProcesses(pids)

	sleepSample(interval)

	cpu2, err2 := readCPUStat()
	disk2, _ := parseDiskstats()
	snap2 := snapshotProcesses(pids)

	if err == nil && err2 == nil {
		user, system, iowait, steal, irq, softirq, idle := cpuUsagePct(cpu1, cpu2)
		report.CPU = linuxPerfCPU{
			Cores: report.Host.CPUCores, UserPct: user, SystemPct: system, IowaitPct: iowait,
			StealPct: steal, IrqPct: irq, SoftirqPct: softirq, IdlePct: idle,
		}
		if report.Host.CPUCores > 0 {
			report.CPU.LoadPerCore1 = report.Load.Load1 / float64(report.Host.CPUCores)
		}
		report.EvidenceCompleteness["cpu"] = true
	}

	secs := interval.Seconds()
	for dev, b := range disk2 {
		a := disk1[dev]
		readBps, writeBps, ioPct := diskIODelta(a, b, secs)
		if readBps == 0 && writeBps == 0 && ioPct == 0 {
			continue
		}
		report.DiskIO = append(report.DiskIO, linuxPerfDiskIO{
			Device: dev, ReadBytesPerSec: readBps, WriteBytesPerSec: writeBps, IOTimePct: ioPct,
		})
	}
	sort.Slice(report.DiskIO, func(i, j int) bool { return report.DiskIO[i].Device < report.DiskIO[j].Device })
	if len(report.DiskIO) > 0 {
		report.EvidenceCompleteness["disk_io"] = true
	}

	procs := mergeProcessDeltas(snap1, snap2, secs, clkTck)
	report.ProcessTop = buildProcessTops(procs, opts.TopN)
	report.EvidenceCompleteness["process_top"] = true

	if opts.PID > 0 {
		enrichTargetPID(report, opts.PID)
	}

	report.LeakRisks = detectLeakRisks(procs, opts, interval)
	if len(report.LeakRisks) > 0 {
		report.EvidenceCompleteness["leak_risks"] = true
	}

	report.KernelSignals = collectKernelSignalsExec()
	if len(report.KernelSignals) > 0 {
		report.EvidenceCompleteness["kernel_signals"] = true
	}

	report.Findings = deriveLinuxFindings(report)
	report.Sample.EndedAt = time.Now()
	return report
}

func fillMemoryReport(report *LinuxPerfReport, kv map[string]int64) {
	get := func(k string) int64 { return kv[k] }
	total := get("MemTotal")
	avail := get("MemAvailable")
	report.Memory = linuxPerfMemory{
		MemTotalKB: total, MemAvailableKB: avail,
		DirtyKB: get("Dirty"), WritebackKB: get("Writeback"),
		SlabKB: get("Slab"), SReclaimableKB: get("SReclaimable"), SUnreclaimKB: get("SUnreclaim"),
	}
	report.Swap = linuxPerfSwap{SwapTotalKB: get("SwapTotal"), SwapFreeKB: get("SwapFree")}
	if total > 0 {
		used := total - avail
		report.Memory.UsedPct = float64(used) / float64(total) * 100
	}
	report.Memory.OOMRisk = classifyOOMRisk(report.Memory.UsedPct, avail, total, report.Swap)
}

func classifyOOMRisk(usedPct float64, availKB, totalKB int64, swap linuxPerfSwap) string {
	if usedPct > 92 || (totalKB > 0 && availKB*100/totalKB < 5) {
		return "high"
	}
	if usedPct > 80 {
		return "medium"
	}
	if swap.SwapTotalKB > 0 && swap.SwapFreeKB*100/swap.SwapTotalKB < 20 {
		return "medium"
	}
	return "low"
}

type procDelta struct {
	linuxPerfProcess
	riskScore int
}

func snapshotProcesses(pids []int) map[int]procSnapshot {
	out := make(map[int]procSnapshot, len(pids))
	for _, pid := range pids {
		s, err := readProcPIDStat(pid)
		if err != nil {
			continue
		}
		out[pid] = s
	}
	return out
}

func mergeProcessDeltas(a, b map[int]procSnapshot, secs, clkTck float64) []procDelta {
	var out []procDelta
	for pid, sb := range b {
		sa, ok := a[pid]
		if !ok {
			sa = sb
		}
		cpuTicks := float64(int64(sb.UTime+sb.STime) - int64(sa.UTime+sa.STime))
		cpuPct := 0.0
		if secs > 0 && clkTck > 0 {
			cpuPct = cpuTicks / clkTck / secs * 100
		}
		readBps := float64(int64(sb.ReadBytes)-int64(sa.ReadBytes)) / secs
		writeBps := float64(int64(sb.WriteBytes)-int64(sa.WriteBytes)) / secs
		if readBps < 0 {
			readBps = 0
		}
		if writeBps < 0 {
			writeBps = 0
		}
		p := procToLinuxProcess(sb, cpuPct, readBps, writeBps, clkTck)
		score := 0
		if cpuPct > 30 {
			score += 3
		} else if cpuPct > 10 {
			score += 1
		}
		if p.OOMScore > 300 {
			score += 2
		}
		if p.FDCount > 500 {
			score += 2
		}
		if p.Threads > 200 {
			score += 1
		}
		p.RiskScore = score
		out = append(out, procDelta{linuxPerfProcess: p, riskScore: score})
	}
	return out
}

func buildProcessTops(procs []procDelta, topN int) linuxPerfProcessTop {
	top := func(cmp func(a, b procDelta) bool) []linuxPerfProcess {
		cp := append([]procDelta(nil), procs...)
		sort.Slice(cp, func(i, j int) bool { return cmp(cp[i], cp[j]) })
		if len(cp) > topN {
			cp = cp[:topN]
		}
		out := make([]linuxPerfProcess, len(cp))
		for i, p := range cp {
			out[i] = p.linuxPerfProcess
		}
		return out
	}
	return linuxPerfProcessTop{
		CPU: top(func(a, b procDelta) bool { return a.CPUPercent > b.CPUPercent }),
		Memory: top(func(a, b procDelta) bool { return a.RSSBytes > b.RSSBytes }),
		IO: top(func(a, b procDelta) bool { return a.ReadBps+a.WriteBps > b.ReadBps+b.WriteBps }),
		FD: top(func(a, b procDelta) bool { return a.FDCount > b.FDCount }),
		Threads: top(func(a, b procDelta) bool { return a.Threads > b.Threads }),
		Risk: top(func(a, b procDelta) bool { return a.riskScore > b.riskScore }),
	}
}

func detectLeakRisks(procs []procDelta, opts LinuxPerfOptions, interval time.Duration) []linuxPerfLeakRisk {
	// Top memory candidates
	cp := append([]procDelta(nil), procs...)
	sort.Slice(cp, func(i, j int) bool { return cp[i].RSSBytes > cp[j].RSSBytes })
	if len(cp) > 5 {
		cp = cp[:5]
	}
	if len(cp) == 0 {
		return nil
	}
	trendInterval := interval / 2
	if trendInterval < 300*time.Millisecond {
		trendInterval = 300 * time.Millisecond
	}
	history := map[int][]leakTrendSample{}
	for round := 0; round < 3; round++ {
		for _, p := range cp {
			pid := p.PID
			rss := readPIDRSSKB(pid)
			anon := readPIDAnonKB(pid)
			s, _ := readProcPIDStat(pid)
			history[pid] = append(history[pid], leakTrendSample{rss: rss, anon: anon, fds: s.FDCount, threads: s.Threads})
		}
		if round < 2 {
			sleepSample(trendInterval)
		}
	}
	var risks []linuxPerfLeakRisk
	for _, p := range cp {
		samples := history[p.PID]
		if len(samples) < 2 {
			continue
		}
		var signals []string
		if leakTrendRising(samples, func(s leakTrendSample) int64 { return s.rss }) {
			signals = append(signals, "RSS 连续上升")
		}
		if leakTrendRising(samples, func(s leakTrendSample) int64 { return s.anon }) {
			signals = append(signals, "Anonymous 内存占比上升")
		}
		if leakTrendRising(samples, func(s leakTrendSample) int64 { return int64(s.fds) }) {
			signals = append(signals, "fd_count 持续上升")
		}
		if leakTrendRising(samples, func(s leakTrendSample) int64 { return int64(s.threads) }) {
			signals = append(signals, "线程数持续上升")
		}
		if p.OOMScore > 500 && p.RSSBytes > 0 {
			signals = append(signals, "oom_score 偏高且内存占用显著")
		}
		if len(signals) == 0 {
			continue
		}
		sev := "medium"
		if len(signals) >= 3 {
			sev = "high"
		}
		risks = append(risks, linuxPerfLeakRisk{PID: p.PID, Comm: p.Comm, Signals: signals, Severity: sev})
	}
	return risks
}

type leakTrendSample struct {
	rss, anon int64
	fds, threads int
}

func leakTrendRising(samples []leakTrendSample, fn func(leakTrendSample) int64) bool {
	if len(samples) < 2 {
		return false
	}
	inc := 0
	for i := 1; i < len(samples); i++ {
		if fn(samples[i]) > fn(samples[i-1]) {
			inc++
		}
	}
	return inc >= len(samples)-1
}

func readPIDRSSKB(pid int) int64 {
	b, err := readProcFile(procPath(strconv.Itoa(pid), "status"))
	if err != nil {
		return 0
	}
	for _, ln := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(ln, "VmRSS:") {
			f := strings.Fields(ln)
			if len(f) >= 2 {
				v, _ := strconv.ParseInt(f[1], 10, 64)
				return v
			}
		}
	}
	return 0
}

func readPIDAnonKB(pid int) int64 {
	b, err := readProcFile(procPath(strconv.Itoa(pid), "smaps_rollup"))
	if err != nil {
		return readPIDRSSKB(pid)
	}
	for _, ln := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(ln, "Anonymous:") {
			f := strings.Fields(ln)
			if len(f) >= 2 {
				v, _ := strconv.ParseInt(f[1], 10, 64)
				return v
			}
		}
	}
	return 0
}

func enrichTargetPID(report *LinuxPerfReport, pid int) {
	s, err := readProcPIDStat(pid)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("pid %d: %s", pid, err))
		return
	}
	p := procToLinuxProcess(s, 0, 0, 0, 100)
	report.Findings = append(report.Findings, fmt.Sprintf("目标进程 PID=%d comm=%s rss=%d fd=%d threads=%d oom_score=%d",
		pid, p.Comm, p.RSSBytes, p.FDCount, p.Threads, p.OOMScore))
}

func collectKernelSignalsExec() []string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	keywords := []string{"Out of memory", "oom-kill", "blocked for more than", "I/O error", "EXT4-fs error", "XFS ("}
	var lines []string
	cmd := exec.CommandContext(ctx, "dmesg", "-T", "--level=err,warn")
	out, err := cmd.CombinedOutput()
	if err != nil {
		cmd2 := exec.CommandContext(ctx, "journalctl", "-k", "-n", "80", "--no-pager")
		out, err = cmd2.CombinedOutput()
		if err != nil {
			if strings.Contains(strings.ToLower(string(out)), "permission") {
				return []string{"permission denied: kernel log"}
			}
			return nil
		}
	}
	for _, ln := range strings.Split(string(out), "\n") {
		low := strings.ToLower(ln)
		for _, kw := range keywords {
			if strings.Contains(ln, kw) || strings.Contains(low, strings.ToLower(kw)) {
				lines = append(lines, strings.TrimSpace(ln))
				break
			}
		}
	}
	if len(lines) > 15 {
		lines = lines[len(lines)-15:]
	}
	return lines
}

func deriveLinuxFindings(r *LinuxPerfReport) []string {
	var f []string
	if r.CPU.IowaitPct > 20 {
		f = append(f, fmt.Sprintf("CPU iowait 偏高 (%.1f%%)", r.CPU.IowaitPct))
	}
	if r.CPU.UserPct+r.CPU.SystemPct > 70 {
		f = append(f, fmt.Sprintf("CPU 使用率偏高 (user+sys=%.1f%%)", r.CPU.UserPct+r.CPU.SystemPct))
	}
	if r.Host.CPUCores > 0 && r.Load.Load1/float64(r.Host.CPUCores) > 1.5 {
		f = append(f, fmt.Sprintf("负载相对核数偏高 (load1/core=%.2f)", r.Load.Load1/float64(r.Host.CPUCores)))
	}
	if r.Memory.OOMRisk == "high" {
		f = append(f, "内存压力高，存在 OOM 风险")
	}
	for _, d := range r.Disks {
		if !d.PseudoFS && d.UsedPct > 90 {
			f = append(f, fmt.Sprintf("磁盘 %s 使用率 %.1f%%", d.Mount, d.UsedPct))
		}
	}
	return f
}

func formatLinuxProbeText(r *LinuxPerfReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Linux 性能快采 (%s, %.0fs)\n", r.Host.Hostname, r.Sample.DurationSeconds)
	fmt.Fprintf(&b, "主机: %d CPU, uptime %.0fh, kernel: %s\n", r.Host.CPUCores, r.Host.UptimeSeconds/3600, truncStr(r.Host.Kernel, 60))
	fmt.Fprintf(&b, "负载: %.2f / %.2f / %.2f", r.Load.Load1, r.Load.Load5, r.Load.Load15)
	if r.Host.CPUCores > 0 {
		fmt.Fprintf(&b, " (load1/core=%.2f)", r.Load.Load1/float64(r.Host.CPUCores))
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "CPU: user=%.1f%% sys=%.1f%% iowait=%.1f%% idle=%.1f%%\n",
		r.CPU.UserPct, r.CPU.SystemPct, r.CPU.IowaitPct, r.CPU.IdlePct)
	fmt.Fprintf(&b, "内存: used=%.1f%% avail=%d MB oom_risk=%s\n",
		r.Memory.UsedPct, r.Memory.MemAvailableKB/1024, r.Memory.OOMRisk)
	if len(r.ProcessTop.CPU) > 0 {
		b.WriteString("\nCPU Top:\n")
		for i, p := range r.ProcessTop.CPU {
			fmt.Fprintf(&b, "  %d. pid=%d %.1f%% %s\n", i+1, p.PID, p.CPUPercent, truncStr(p.Cmdline, 80))
		}
	}
	if len(r.ProcessTop.Memory) > 0 {
		b.WriteString("\n内存 Top:\n")
		for i, p := range r.ProcessTop.Memory {
			fmt.Fprintf(&b, "  %d. pid=%d rss=%.0fMB %s\n", i+1, p.PID, float64(p.RSSBytes)/1024/1024, truncStr(p.Cmdline, 80))
		}
	}
	if len(r.LeakRisks) > 0 {
		b.WriteString("\n泄露风险预警:\n")
		for _, lr := range r.LeakRisks {
			fmt.Fprintf(&b, "  pid=%d %s [%s]: %s\n", lr.PID, lr.Comm, lr.Severity, strings.Join(lr.Signals, "; "))
		}
	}
	if len(r.Findings) > 0 {
		b.WriteString("\n发现:\n")
		for _, ln := range r.Findings {
			fmt.Fprintf(&b, "  - %s\n", ln)
		}
	}
	if len(r.Errors) > 0 {
		b.WriteString("\n采集错误:\n")
		for _, e := range r.Errors {
			fmt.Fprintf(&b, "  - %s\n", e)
		}
	}
	return b.String()
}

func truncStr(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// collectLinuxProbeJSON returns JSON bytes for check evidence injection.
func collectLinuxProbeJSON(ctx context.Context, flags map[string]string) (string, *LinuxPerfReport, error) {
	opts := linuxPerfOptionsFromFlags(flags)
	if err := ValidateLinuxPerfOptions(opts); err != nil {
		return "", nil, err
	}
	report := CollectLinuxPerf(opts)
	b, err := json.Marshal(report)
	if err != nil {
		return "", report, err
	}
	return string(b), report, nil
}

func linuxPerfOptionsFromFlags(flags map[string]string) LinuxPerfOptions {
	opts := LinuxPerfOptions{Duration: linuxPerfDefaultDuration, TopN: linuxPerfDefaultTop}
	if flags == nil {
		return opts
	}
	if v := strings.TrimSpace(flags["duration"]); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			opts.Duration = d
		}
	}
	if v := strings.TrimSpace(flags["top"]); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			opts.TopN = n
		}
	}
	if v := strings.TrimSpace(flags["pid"]); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			opts.PID = n
		}
	}
	return opts
}
