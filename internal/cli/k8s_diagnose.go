package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// k8s diagnose — runs on a control-plane / worker node (or control machine with kubeconfig)
// to auto-detect the most common K8s instability root causes we hit in this repo:
//
//	1. Clock skew / RTC jump back (VirtualBox, ARM64 nested). Corrupts kubelet PLEG ->
//	   calico-node/coredns "Killing" every 60-70s with SandboxChanged events.
//	2. etcd slow disk: "apply request took too long" -> apiserver context deadline exceeded.
//	3. Calico probes too strict for low-RAM/slow VM.
//	4. Swap enabled / br_netfilter missing / ip_forward=0 (preflight).
//	5. CoreDNS "plugin/loop" (kubelet resolv.conf -> systemd stub).
//
// The command is intentionally conservative (read-only, each shell-out has a tight timeout)
// so it is safe to run on a production-like node. It does NOT call the LLM by itself; users
// can pipe the JSON output into `ai-sre analyze k8s --issue instability` for an LLM summary.

type diagCheck struct {
	Name     string   `json:"name"`
	Category string   `json:"category"` // clock | etcd | kubelet | calico | coredns | preflight | general
	Status   string   `json:"status"`   // ok | warn | fail | skipped
	Summary  string   `json:"summary"`
	Detail   string   `json:"detail,omitempty"`
	Hints    []string `json:"hints,omitempty"`
}

type diagReport struct {
	Host       string      `json:"host"`
	StartedAt  time.Time   `json:"startedAt"`
	DurationMS int64       `json:"durationMs"`
	Mode       string      `json:"mode"` // preflight | post-install | auto
	Checks     []diagCheck `json:"checks"`
	Summary    struct {
		OK      int `json:"ok"`
		Warn    int `json:"warn"`
		Fail    int `json:"fail"`
		Skipped int `json:"skipped"`
	} `json:"summary"`
	TopSuspects []string `json:"topSuspects,omitempty"`
}

// shortCmd runs cmd with a bounded timeout and returns stdout (+stderr on non-zero for context).
func shortCmd(timeout time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	c := exec.CommandContext(ctx, name, args...)
	out, err := c.CombinedOutput()
	return string(out), err
}

func hasBin(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

// addCheck appends a diagCheck; used with a small helper for brevity.
func (r *diagReport) add(c diagCheck) {
	r.Checks = append(r.Checks, c)
}

// runClockChecks: the #1 root cause in our lab. We check NTPSynchronized, time since last
// sync, and scan journalctl --list-boots for hours-level backward jumps within a boot.
func (r *diagReport) runClockChecks() {
	// timedatectl
	if !hasBin("timedatectl") {
		r.add(diagCheck{Name: "clock.timedatectl", Category: "clock", Status: "skipped", Summary: "timedatectl 不存在（非 systemd 系统？）"})
	} else {
		out, _ := shortCmd(5*time.Second, "timedatectl", "show", "-p", "NTPSynchronized", "-p", "LocalRTC", "-p", "TimeUSec", "--value")
		lines := strings.Split(strings.TrimSpace(out), "\n")
		synced := len(lines) > 0 && strings.EqualFold(strings.TrimSpace(lines[0]), "yes")
		c := diagCheck{Name: "clock.ntp_synchronized", Category: "clock", Summary: "NTPSynchronized=" + strings.TrimSpace(firstLine(out))}
		if synced {
			c.Status = "ok"
		} else {
			c.Status = "fail"
			c.Hints = []string{
				"systemctl enable --now chrony || systemctl enable --now systemd-timesyncd",
				"chronyc sources -v  # 查看 NTP 源是否可达",
				"关键：kubelet / containerd / etcd 必须在 chrony 同步完成后再启动（可用 chrony-wait.service + After= 串联）",
			}
		}
		c.Detail = out
		r.add(c)
	}

	// journalctl --list-boots: detect backward jumps within the current boot
	if hasBin("journalctl") {
		out, _ := shortCmd(5*time.Second, "journalctl", "--list-boots", "--no-pager")
		c := diagCheck{Name: "clock.boot_history", Category: "clock", Detail: out}
		if strings.TrimSpace(out) == "" {
			c.Status = "skipped"
			c.Summary = "未获取到 journal boot 列表"
		} else {
			// Heuristic: if a single boot line contains two dates differing by > 1h, it's a jump.
			// journalctl output: "IDX BOOTID YYYY-MM-DD HH:MM:SS — YYYY-MM-DD HH:MM:SS"
			jump := detectBootTimeJump(out)
			if jump {
				c.Status = "fail"
				c.Summary = "检测到 boot 内时钟跳变（本仓库最常见的 calico-node/coredns 60-70s Killing 根因）"
				c.Hints = []string{
					"确保 chrony/systemd-timesyncd 在 kubelet/containerd 之前完成同步",
					"VirtualBox 建议: VBoxManage setextradata <VM> VBoxInternal/Devices/VMMDev/0/Config/GetHostTimeDisabled 0",
					"ARM 嵌套机: 宿主 chrony 稳定后再开虚拟机；或在 /etc/systemd/system/kubelet.service.d/ 里加 After=chrony-wait.service",
				}
			} else {
				c.Status = "ok"
				c.Summary = "近期 boot 内未发现明显的小时级时间跳变"
			}
		}
		r.add(c)
	}
}

// detectBootTimeJump inspects `journalctl --list-boots` lines. Each line typically looks like:
//
//	-0 <boot-id> Tue 2026-04-22 03:12:01 UTC—Wed 2026-04-22 10:45:33 UTC
//
// We treat a boot line as "jumped" if the second datetime is earlier than the first
// AND the reverse span is more than 1 hour. That is the signature we actually hit on VBox ARM.
func detectBootTimeJump(list string) bool {
	layouts := []string{
		"2006-01-02 15:04:05 MST",
		"2006-01-02 15:04:05",
		"Mon 2006-01-02 15:04:05 MST",
	}
	parse := func(s string) (time.Time, bool) {
		s = strings.TrimSpace(s)
		for _, l := range layouts {
			if t, err := time.Parse(l, s); err == nil {
				return t, true
			}
		}
		return time.Time{}, false
	}
	for _, ln := range strings.Split(list, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		// split on em-dash (—) or regular dash with spaces around
		parts := splitAroundDash(ln)
		if len(parts) != 2 {
			continue
		}
		// left field: take last 25 chars chunk that parses
		leftT, okL := bestTail(parts[0], parse)
		rightT, okR := bestTail(parts[1], parse)
		if !okL || !okR {
			continue
		}
		if leftT.Sub(rightT) > time.Hour {
			return true
		}
	}
	return false
}

func splitAroundDash(s string) []string {
	// try em-dash first (journalctl default), fall back to " - "
	if strings.Contains(s, "—") {
		return strings.SplitN(s, "—", 2)
	}
	if i := strings.Index(s, " - "); i >= 0 {
		return []string{s[:i], s[i+3:]}
	}
	return nil
}

// bestTail tries to parse progressively longer suffix substrings (space-separated) of s.
func bestTail(s string, parse func(string) (time.Time, bool)) (time.Time, bool) {
	fields := strings.Fields(s)
	for start := 0; start < len(fields); start++ {
		cand := strings.Join(fields[start:], " ")
		if t, ok := parse(cand); ok {
			return t, true
		}
	}
	return time.Time{}, false
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// runPreflightChecks: swap, br_netfilter, sysctl, hostname, arch, memory.
func (r *diagReport) runPreflightChecks() {
	// swap
	if out, _ := shortCmd(3*time.Second, "swapon", "--show"); true {
		c := diagCheck{Name: "preflight.swap", Category: "preflight", Detail: out}
		if strings.TrimSpace(out) == "" {
			c.Status = "ok"
			c.Summary = "swap 已关闭"
		} else {
			c.Status = "fail"
			c.Summary = "swap 处于开启状态（kubelet 默认拒绝启动）"
			c.Hints = []string{"swapoff -a && sed -i '/\\sswap\\s/s/^/#/' /etc/fstab"}
		}
		r.add(c)
	}

	// kernel modules (Linux-only; macOS/BSD have no lsmod)
	c := diagCheck{Name: "preflight.kernel_modules", Category: "preflight"}
	if !hasBin("lsmod") {
		c.Status = "skipped"
		c.Summary = "lsmod 不存在（非 Linux 节点？）"
	} else {
		out, _ := shortCmd(3*time.Second, "sh", "-c", "lsmod | grep -E '^(br_netfilter|overlay) ' || true")
		c.Detail = out
		has := func(mod string) bool {
			return strings.Contains(out, mod+" ") || strings.HasPrefix(strings.TrimSpace(out), mod+" ") || strings.Contains(out, "\n"+mod+" ")
		}
		missing := []string{}
		if !has("br_netfilter") {
			missing = append(missing, "br_netfilter")
		}
		if !has("overlay") {
			missing = append(missing, "overlay")
		}
		if len(missing) == 0 {
			c.Status = "ok"
			c.Summary = "br_netfilter / overlay 均已加载"
		} else {
			c.Status = "fail"
			c.Summary = "缺失内核模块: " + strings.Join(missing, ", ")
			c.Hints = []string{
				"modprobe br_netfilter overlay",
				"echo -e 'br_netfilter\\noverlay' > /etc/modules-load.d/k8s.conf",
			}
		}
	}
	r.add(c)

	// sysctl — read two keys; bridge-nf-call-iptables may not exist until br_netfilter is loaded.
	readSysctl := func(key string) string {
		out, _ := shortCmd(2*time.Second, "sh", "-c", "sysctl -n "+key+" 2>/dev/null")
		return strings.TrimSpace(out)
	}
	ipFwd := readSysctl("net.ipv4.ip_forward")
	brNF := readSysctl("net.bridge.bridge-nf-call-iptables")
	sc := diagCheck{Name: "preflight.sysctl", Category: "preflight", Detail: fmt.Sprintf("net.ipv4.ip_forward=%q, net.bridge.bridge-nf-call-iptables=%q", ipFwd, brNF)}
	if ipFwd == "1" && brNF == "1" {
		sc.Status = "ok"
		sc.Summary = "ip_forward=1, bridge-nf-call-iptables=1"
	} else if ipFwd == "" && brNF == "" {
		sc.Status = "skipped"
		sc.Summary = "当前内核不支持这些 sysctl（非 Linux？）"
	} else {
		sc.Status = "fail"
		sc.Summary = fmt.Sprintf("关键 sysctl 未置 1: ip_forward=%q bridge-nf-call-iptables=%q", ipFwd, brNF)
		sc.Hints = []string{
			"cat >/etc/sysctl.d/k8s.conf <<EOF\nnet.ipv4.ip_forward=1\nnet.bridge.bridge-nf-call-iptables=1\nnet.bridge.bridge-nf-call-ip6tables=1\nEOF",
			"sysctl --system",
		}
	}
	r.add(sc)

	// memory
	memOut, _ := shortCmd(3*time.Second, "sh", "-c", "awk '/MemAvailable/ {printf \"%d\\n\", $2/1024}' /proc/meminfo")
	mc := diagCheck{Name: "preflight.memory_available_mib", Category: "preflight", Detail: strings.TrimSpace(memOut)}
	if mib, err := atoiSafe(strings.TrimSpace(memOut)); err == nil {
		mc.Summary = fmt.Sprintf("MemAvailable=%dMiB", mib)
		switch {
		case mib < 2048:
			mc.Status = "fail"
			mc.Hints = []string{"master 节点建议 ≥ 8GiB；worker 建议 ≥ 4GiB；内存不足会导致 etcd fsync 抖动、kubelet OOM、sandbox 反复重建"}
		case mib < 4096:
			mc.Status = "warn"
			mc.Hints = []string{"master 节点建议 ≥ 8GiB；虚拟机尤其要留余量给 etcd"}
		default:
			mc.Status = "ok"
		}
	} else {
		mc.Status = "skipped"
		mc.Summary = "/proc/meminfo 解析失败"
	}
	r.add(mc)
}

// runKubeletEtcdContainerdChecks: post-install signals.
func (r *diagReport) runKubeletEtcdContainerdChecks() {
	for _, unit := range []string{"kubelet", "containerd", "etcd"} {
		st, _ := shortCmd(3*time.Second, "systemctl", "is-active", unit)
		active := strings.TrimSpace(st) == "active"
		c := diagCheck{Name: "service." + unit, Category: unit, Summary: "systemctl is-active " + unit + "=" + strings.TrimSpace(st)}
		if active {
			c.Status = "ok"
		} else {
			// etcd only exists on master; not a fail if it's simply not installed here.
			if unit == "etcd" && !hasBin("etcd") && !strings.Contains(st, "failed") {
				c.Status = "skipped"
				c.Summary = "本机无 etcd（worker 节点正常跳过）"
			} else {
				c.Status = "fail"
				c.Hints = []string{
					"journalctl -u " + unit + " -n 80 --no-pager",
					"systemctl status " + unit,
				}
			}
		}
		r.add(c)

		// recent restart count
		if active && hasBin("systemctl") {
			nr, _ := shortCmd(3*time.Second, "systemctl", "show", "-p", "NRestarts", "--value", unit)
			nrTrim := strings.TrimSpace(nr)
			rc := diagCheck{Name: "service." + unit + ".restarts", Category: unit, Summary: unit + " NRestarts=" + nrTrim, Detail: nr}
			if n, err := atoiSafe(nrTrim); err == nil && n >= 3 {
				rc.Status = "fail"
				rc.Hints = []string{
					"journalctl -u " + unit + " --since '15 min ago' --no-pager",
					"若伴随时钟跳变告警 → 先修时间，再 systemctl restart " + unit,
				}
			} else {
				rc.Status = "ok"
			}
			r.add(rc)
		}
	}

	// etcd slow apply
	if hasBin("journalctl") {
		out, _ := shortCmd(4*time.Second, "sh", "-c", "journalctl -u etcd --since '30 min ago' --no-pager 2>/dev/null | grep -iE 'apply request took too long|took.*expected|timed out|fsync' | tail -40")
		c := diagCheck{Name: "etcd.slow_apply", Category: "etcd", Detail: out}
		if strings.TrimSpace(out) == "" {
			c.Status = "ok"
			c.Summary = "近 30 分钟 etcd 无慢 apply / fsync 告警"
		} else {
			c.Status = "fail"
			c.Summary = "etcd 出现慢 apply / fsync 告警（盘 IO 不够）"
			c.Hints = []string{
				"本仓库已在 etcd.service.j2 内置 heartbeat-interval=500/election-timeout=5000/snapshot-count=10000/quota-backend-bytes=8GiB",
				"虚拟机建议把 /var/lib/etcd 放到独立 SSD 或 tmpfs-raid；或在 group_vars 里 etcd_unsafe_no_fsync=true（仅 lab，生产禁用）",
			}
		}
		r.add(c)
	}

	// kubelet SandboxChanged / PLEG signals
	if hasBin("journalctl") {
		out, _ := shortCmd(4*time.Second, "sh", "-c", "journalctl -u kubelet --since '15 min ago' --no-pager 2>/dev/null | grep -iE 'SandboxChanged|PLEG|context deadline' | tail -40")
		c := diagCheck{Name: "kubelet.sandbox_churn", Category: "kubelet", Detail: out}
		if strings.TrimSpace(out) == "" {
			c.Status = "ok"
			c.Summary = "近 15 分钟 kubelet 无 SandboxChanged / PLEG 告警"
		} else {
			c.Status = "fail"
			c.Summary = "kubelet 在频繁 SandboxChanged 或 PLEG 超时"
			c.Hints = []string{
				"几乎总是由时钟跳变、containerd 重启或 OOM 引发 —— 请先看本报告的 clock.* 与 service.*.restarts 项",
				"排除时钟后若仍抖动，把 calico-node 探针阈值放宽（本仓库 patch_calico_manifest.py 已默认处理）",
			}
		}
		r.add(c)
	}
}

// runClusterChecks: only runs if kubectl + kubeconfig are available.
func (r *diagReport) runClusterChecks() {
	if !hasBin("kubectl") {
		r.add(diagCheck{Name: "cluster.kubectl", Category: "general", Status: "skipped", Summary: "kubectl 不在 PATH，跳过集群侧检查"})
		return
	}
	// quick reachability probe
	out, err := shortCmd(5*time.Second, "kubectl", "version", "--short=true", "--request-timeout=3s")
	if err != nil {
		r.add(diagCheck{Name: "cluster.reachable", Category: "general", Status: "skipped", Summary: "kubectl 无法连到 apiserver（可能是 worker 或无 kubeconfig）", Detail: out})
		return
	}
	r.add(diagCheck{Name: "cluster.reachable", Category: "general", Status: "ok", Summary: "apiserver 可连", Detail: out})

	// nodes
	if nOut, err := shortCmd(6*time.Second, "kubectl", "get", "nodes", "--no-headers"); err == nil {
		notReady := 0
		for _, ln := range strings.Split(strings.TrimSpace(nOut), "\n") {
			if ln == "" {
				continue
			}
			fields := strings.Fields(ln)
			if len(fields) >= 2 && !strings.EqualFold(fields[1], "Ready") {
				notReady++
			}
		}
		c := diagCheck{Name: "cluster.nodes", Category: "general", Detail: nOut}
		if notReady == 0 {
			c.Status = "ok"
			c.Summary = "所有 node 状态为 Ready"
		} else {
			c.Status = "fail"
			c.Summary = fmt.Sprintf("%d 个 node 未 Ready", notReady)
			c.Hints = []string{"kubectl describe node <未Ready节点> | sed -n '/Conditions:/,/Addresses/p'"}
		}
		r.add(c)
	}

	// kube-system crashloops
	if pOut, err := shortCmd(6*time.Second, "kubectl", "-n", "kube-system", "get", "pods", "--no-headers"); err == nil {
		crash := []string{}
		restarting := []string{}
		for _, ln := range strings.Split(strings.TrimSpace(pOut), "\n") {
			if ln == "" {
				continue
			}
			fields := strings.Fields(ln)
			if len(fields) < 4 {
				continue
			}
			name, ready, status, rest := fields[0], fields[1], fields[2], fields[3]
			_ = ready
			if strings.Contains(strings.ToLower(status), "crashloop") || strings.EqualFold(status, "Error") {
				crash = append(crash, name+"("+status+")")
			}
			if n, err := atoiSafe(rest); err == nil && n >= 3 {
				restarting = append(restarting, fmt.Sprintf("%s(restarts=%d)", name, n))
			}
		}
		c := diagCheck{Name: "cluster.kube_system_pods", Category: "general", Detail: pOut}
		if len(crash) == 0 && len(restarting) == 0 {
			c.Status = "ok"
			c.Summary = "kube-system 无 CrashLoop/高重启"
		} else {
			c.Status = "fail"
			parts := []string{}
			if len(crash) > 0 {
				parts = append(parts, "CrashLoop: "+strings.Join(crash, ", "))
			}
			if len(restarting) > 0 {
				parts = append(parts, "高重启: "+strings.Join(restarting, ", "))
			}
			c.Summary = strings.Join(parts, "; ")
			c.Hints = []string{
				"kubectl -n kube-system describe pod <pod> | tail -40",
				"kubectl -n kube-system logs --previous <pod>",
				"calico-node/coredns 反复重启时先看 kubelet 的 SandboxChanged —— 通常是时钟 or 内存 or etcd",
			}
		}
		r.add(c)
	}

	// recent warning events
	if eOut, err := shortCmd(6*time.Second, "sh", "-c", "kubectl get events -A --field-selector type=Warning --sort-by=.lastTimestamp --no-headers 2>/dev/null | tail -20"); err == nil {
		c := diagCheck{Name: "cluster.recent_warnings", Category: "general", Detail: eOut}
		if strings.TrimSpace(eOut) == "" {
			c.Status = "ok"
			c.Summary = "近期无 Warning 事件"
		} else {
			c.Status = "warn"
			c.Summary = "存在 Warning 事件（详见 Detail）"
		}
		r.add(c)
	}
}

func atoiSafe(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty")
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("non-digit: %q", s)
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}

func k8sDiagnoseCmd() *cobra.Command {
	var (
		preflightOnly bool
		postOnly      bool
		jsonOut       bool
	)
	cmd := &cobra.Command{
		Use:   "diagnose",
		Short: "在本机自动识别 K8s 集群常见抖动根因（时钟跳变 / etcd 慢盘 / SandboxChanged / calico-node 抖动 / 预检缺项）",
		Long: `本命令只做只读采集（systemctl / journalctl / kubectl / /proc），
针对本仓库踩过的真实根因逐项判定：

  1) 时钟：timedatectl NTPSynchronized + journalctl --list-boots 是否有 boot 内小时级回拨
  2) etcd：journalctl -u etcd 是否出现 "apply request took too long"
  3) kubelet：journalctl -u kubelet 是否反复 SandboxChanged / PLEG
  4) 预检：swap / br_netfilter / ip_forward / 内存
  5) 集群侧（kubectl 可用时）：node 状态、kube-system CrashLoop、近 Warning 事件

输出可以是可读文本（默认）或 JSON（-o json / --json）；可用 --preflight 只跑预检，
--post-install 只跑安装后检查。
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			report := diagReport{StartedAt: time.Now()}
			host, _ := os.Hostname()
			report.Host = host
			switch {
			case preflightOnly && postOnly:
				return errors.New("--preflight 与 --post-install 不可同时指定")
			case preflightOnly:
				report.Mode = "preflight"
			case postOnly:
				report.Mode = "post-install"
			default:
				report.Mode = "auto"
			}

			if report.Mode != "post-install" {
				report.runPreflightChecks()
				report.runClockChecks()
			}
			if report.Mode != "preflight" {
				report.runClockChecks() // clock always matters in post-install mode
				report.runKubeletEtcdContainerdChecks()
				report.runClusterChecks()
			}
			// dedup clock checks if both modes ran them
			report.Checks = dedupChecksByName(report.Checks)

			for _, c := range report.Checks {
				switch c.Status {
				case "ok":
					report.Summary.OK++
				case "warn":
					report.Summary.Warn++
				case "fail":
					report.Summary.Fail++
				case "skipped":
					report.Summary.Skipped++
				}
			}
			report.TopSuspects = deriveTopSuspects(report.Checks)
			report.DurationMS = time.Since(report.StartedAt).Milliseconds()

			if jsonOut || strings.EqualFold(outputFormat, "json") {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(report)
			}
			printDiagText(&report)
			return nil
		},
	}
	cmd.Flags().BoolVar(&preflightOnly, "preflight", false, "只执行部署前预检（swap / 内核模块 / sysctl / 时钟 / 内存）")
	cmd.Flags().BoolVar(&postOnly, "post-install", false, "只执行安装后诊断（etcd / kubelet / containerd / kube-system）")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "以 JSON 输出（可直接喂给 ai-sre analyze k8s --issue instability）")
	return cmd
}

func dedupChecksByName(in []diagCheck) []diagCheck {
	seen := map[string]struct{}{}
	out := in[:0]
	for _, c := range in {
		if _, ok := seen[c.Name]; ok {
			continue
		}
		seen[c.Name] = struct{}{}
		out = append(out, c)
	}
	return out
}

func deriveTopSuspects(cs []diagCheck) []string {
	// priority order: clock -> etcd -> kubelet -> service restarts -> preflight -> cluster
	order := map[string]int{
		"clock.boot_history":       1,
		"clock.ntp_synchronized":   2,
		"etcd.slow_apply":          3,
		"kubelet.sandbox_churn":    4,
		"service.etcd.restarts":    5,
		"service.kubelet.restarts": 6,
		"preflight.memory_available_mib": 7,
		"preflight.swap":                 8,
		"preflight.sysctl":               9,
		"preflight.kernel_modules":       10,
		"cluster.nodes":                  11,
		"cluster.kube_system_pods":       12,
	}
	type cand struct {
		name  string
		rank  int
		brief string
	}
	var list []cand
	for _, c := range cs {
		if c.Status != "fail" {
			continue
		}
		rk, ok := order[c.Name]
		if !ok {
			rk = 99
		}
		list = append(list, cand{c.Name, rk, c.Summary})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].rank < list[j].rank })
	out := []string{}
	for _, c := range list {
		out = append(out, c.name+": "+c.brief)
	}
	if len(out) > 5 {
		out = out[:5]
	}
	return out
}

func printDiagText(r *diagReport) {
	fmt.Printf("== %s k8s diagnose on %s (mode=%s) ==\n", progName, r.Host, r.Mode)
	fmt.Printf("duration=%dms  ok=%d warn=%d fail=%d skipped=%d\n\n", r.DurationMS, r.Summary.OK, r.Summary.Warn, r.Summary.Fail, r.Summary.Skipped)
	for _, c := range r.Checks {
		icon := map[string]string{"ok": "[OK]", "warn": "[WARN]", "fail": "[FAIL]", "skipped": "[SKIP]"}[c.Status]
		if icon == "" {
			icon = "[?]"
		}
		fmt.Printf("%s %-38s  %s\n", icon, c.Name, c.Summary)
		for _, h := range c.Hints {
			fmt.Printf("        ↳ %s\n", h)
		}
	}
	if len(r.TopSuspects) > 0 {
		fmt.Println("\n最有可能的根因（按优先级）:")
		for i, s := range r.TopSuspects {
			fmt.Printf("  %d. %s\n", i+1, s)
		}
		fmt.Printf("\n下一步: sudo %s analyze k8s --issue instability -d sample=\"$(sudo %s k8s diagnose --json)\"\n", progName, progName)
	}
}
