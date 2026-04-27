package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// nodeCmd 提供 OpsFleet 控制台「初始化工具」的等价 CLI：
// 在控制机本地构建 Ansible inventory + playbook，调用 ansible-playbook
// 对一组节点执行时间同步、系统参数优化等操作；与前端
// ft-front/src/views/init-tools/scripts.ts 的 Ansible 脚本语义保持一致。
//
// 控制台「ai-sre CLI」Tab 复制下来的命令应可在节点上直接运行（控制机即任意一台
// 可 SSH 到目标节点的 Linux 主机；未指定 --clients/--nodes 时仅对 localhost 执行）。
func nodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node",
		Short: "节点初始化（time-sync / sys-param 等，对应控制台「初始化工具」）",
		Long: fmt.Sprintf(`通过 Ansible 在控制机上对一组节点统一执行初始化操作。

未填写 --clients/--nodes 时仅对 localhost 执行。
执行前会自动检测 ansible-playbook，缺失则按 apt/dnf/yum 安装（可用 --auto-install-ansible=false 关闭）。

示例:
  sudo %s node tune time-sync --ntp-mode public \
    --clients 192.168.1.10,192.168.1.11 \
    --ntp-server ntp.aliyun.com --tool chrony --on-conflict skip

  sudo %s node tune sys-param --nodes 192.168.1.10 \
    --disable-swap=true --raise-ulimit=true \
    --sysctl net.ipv4.ip_forward=1 --sysctl vm.swappiness=10`, progName, progName),
	}
	cmd.AddCommand(nodeTuneCmd())
	return cmd
}

func nodeTuneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tune",
		Short: "节点初始化的统一入口：time-sync | sys-param",
	}
	cmd.AddCommand(nodeTuneTimeSyncCmd(), nodeTuneSysParamCmd())
	return cmd
}

// ───────────────────────── time-sync ─────────────────────────

func nodeTuneTimeSyncCmd() *cobra.Command {
	var (
		ntpMode         string
		masterNode      string
		clients         string
		ntpServer       string
		fallbackNTP     string
		timezone        string
		tool            string
		onConflict      string
		syncIntervalMin int
		autoInstall     bool
		dryRun          bool
	)
	cmd := &cobra.Command{
		Use:   "time-sync",
		Short: "时间同步：chrony / systemd-timesyncd 校时与时区",
		RunE: func(cmd *cobra.Command, args []string) error {
			if ntpMode != "public" && ntpMode != "self-hosted" {
				return fmt.Errorf("--ntp-mode 必须是 public 或 self-hosted")
			}
			if tool != "chrony" && tool != "timesyncd" {
				return fmt.Errorf("--tool 必须是 chrony 或 timesyncd")
			}
			if onConflict != "skip" && onConflict != "force" {
				return fmt.Errorf("--on-conflict 必须是 skip 或 force")
			}
			masterIP := strings.TrimSpace(masterNode)
			isSelfHosted := ntpMode == "self-hosted"
			if isSelfHosted && masterIP == "" {
				return fmt.Errorf("--ntp-mode self-hosted 时必须提供 --master-node")
			}
			ntpTarget := strings.TrimSpace(ntpServer)
			if isSelfHosted {
				ntpTarget = masterIP
			}
			if ntpTarget == "" {
				return fmt.Errorf("--ntp-server 不能为空（或自建模式提供 --master-node）")
			}
			fallback := ""
			if !isSelfHosted {
				fallback = strings.TrimSpace(fallbackNTP)
			}

			clientIPs := splitCSV(clients)
			groups := []ansibleGroup{
				{Name: "clients", IPs: clientIPs, FallbackLocalhost: true},
			}
			if isSelfHosted {
				groups = append(groups, ansibleGroup{Name: "ntp_master", IPs: []string{masterIP}})
			}
			inventory := buildAnsibleInventory(groups)

			playbook := genTimeSyncPlaybook(timeSyncPlaybookOpts{
				IsSelfHosted:    isSelfHosted,
				Timezone:        strings.TrimSpace(timezone),
				Tool:            tool,
				OnConflict:      onConflict,
				NTPTarget:       ntpTarget,
				Fallback:        fallback,
				SyncIntervalMin: syncIntervalMin,
			})

			modeDesc := "公用 NTP: " + ntpTarget
			if isSelfHosted {
				modeDesc = "自建主节点 " + ntpTarget
			}
			nodeDesc := "本机（localhost）"
			if len(clientIPs) > 0 {
				nodeDesc = fmt.Sprintf("%d 个客户端节点", len(clientIPs))
				if isSelfHosted {
					nodeDesc += " + 1 个主节点"
				}
			}
			fmt.Fprintf(os.Stderr, "==> ai-sre 时间同步 | %s | 节点: %s\n", modeDesc, nodeDesc)

			if dryRun {
				printDryRun(inventory, playbook)
				return nil
			}
			return runAnsiblePlaybook(inventory, playbook, autoInstall)
		},
	}
	cmd.Flags().StringVar(&ntpMode, "ntp-mode", "public", "公用 NTP / 自建主节点：public | self-hosted")
	cmd.Flags().StringVar(&masterNode, "master-node", "", "self-hosted 模式下 NTP 主节点 IP")
	cmd.Flags().StringVar(&clients, "clients", "", "客户端节点 IP 列表，逗号分隔（留空：仅对控制机 localhost 执行）")
	cmd.Flags().StringVar(&ntpServer, "ntp-server", "ntp.aliyun.com", "公用 NTP 服务器地址")
	cmd.Flags().StringVar(&fallbackNTP, "fallback-ntp-server", "ntp1.aliyun.com", "备用 NTP 服务器地址（公用模式生效）")
	cmd.Flags().StringVar(&timezone, "timezone", "Asia/Shanghai", "时区")
	cmd.Flags().StringVar(&tool, "tool", "chrony", "NTP 工具：chrony | timesyncd")
	cmd.Flags().StringVar(&onConflict, "on-conflict", "skip", "已存在时的策略：skip | force")
	cmd.Flags().IntVar(&syncIntervalMin, "sync-interval-min", 15, "PollIntervalMinSec 的分钟值（仅 timesyncd 生效）")
	cmd.Flags().BoolVar(&autoInstall, "auto-install-ansible", true, "若未安装 ansible 则自动通过 apt/dnf/yum 安装")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "仅打印 inventory 与 playbook，不执行")
	return cmd
}

type timeSyncPlaybookOpts struct {
	IsSelfHosted    bool
	Timezone        string
	Tool            string
	OnConflict      string
	NTPTarget       string
	Fallback        string
	SyncIntervalMin int
}

func genTimeSyncPlaybook(o timeSyncPlaybookOpts) string {
	tz := o.Timezone
	if tz == "" {
		tz = "Asia/Shanghai"
	}

	chronyConfBlock := fmt.Sprintf("server %s iburst prefer", o.NTPTarget)
	if o.Fallback != "" {
		chronyConfBlock += "\n          server " + o.Fallback + " iburst"
	}
	chronyConfBlock += "\n          makestep 1.0 3\n          rtcsync\n          driftfile /var/lib/chrony/chrony.drift\n          logdir /var/log/chrony"

	masterPlaybook := ""
	if o.IsSelfHosted {
		masterPlaybook = fmt.Sprintf(`
- name: 配置 NTP 主节点（chrony 服务端）
  hosts: ntp_master
  become: yes
  tasks:
    - name: 安装 chrony (Debian/Ubuntu)
      apt: { name: chrony, state: present, update_cache: yes }
      when: ansible_os_family == "Debian"
    - name: 安装 chrony (RedHat/CentOS)
      yum: { name: chrony, state: present }
      when: ansible_os_family == "RedHat"
    - name: 配置 chrony 作为服务端
      copy:
        dest: "{{ '/etc/chrony/chrony.conf' if ansible_os_family == 'Debian' else '/etc/chrony.conf' }}"
        content: |
          # Managed by ai-sre time-sync — NTP Master
          server ntp.aliyun.com iburst prefer
          server ntp1.aliyun.com iburst
          allow all
          local stratum 10
          makestep 1.0 3
          rtcsync
          driftfile /var/lib/chrony/chrony.drift
          logdir /var/log/chrony
    - name: 启动并开机自启 chrony
      service:
        name: "{{ 'chrony' if ansible_os_family == 'Debian' else 'chronyd' }}"
        state: restarted
        enabled: yes
    - name: 设置时区
      timezone: { name: %q }
`, tz)
	}

	pollSec := o.SyncIntervalMin * 60
	if pollSec <= 0 {
		pollSec = 15 * 60
	}
	clientPlaybook := fmt.Sprintf(`
- name: 配置 NTP 客户端节点
  hosts: clients
  become: yes
  tasks:
    - name: 安装 chrony (Debian/Ubuntu)
      apt: { name: chrony, state: present, update_cache: yes }
      when: ansible_os_family == "Debian" and %q == "chrony"
    - name: 安装 chrony (RedHat/CentOS)
      yum: { name: chrony, state: present }
      when: ansible_os_family == "RedHat" and %q == "chrony"
    - name: 检测现有时间同步服务
      shell: systemctl is-active chrony chronyd ntpd systemd-timesyncd 2>/dev/null | grep -c active || true
      register: existing_ntp
      changed_when: false
    - name: 跳过（ON_CONFLICT=skip 且已有服务）
      meta: end_play
      when: existing_ntp.stdout | int > 0 and %q == "skip"
    - name: 配置 chrony 客户端
      copy:
        dest: "{{ '/etc/chrony/chrony.conf' if ansible_os_family == 'Debian' else '/etc/chrony.conf' }}"
        content: |
          # Managed by ai-sre time-sync — NTP Client
          %s
      when: %q == "chrony"
    - name: 配置 systemd-timesyncd
      copy:
        dest: /etc/systemd/timesyncd.conf
        content: |
          [Time]
          NTP=%s
          %s
          PollIntervalMinSec=%d
      when: %q == "timesyncd"
    - name: 启动并开机自启 chrony
      service:
        name: "{{ 'chrony' if ansible_os_family == 'Debian' else 'chronyd' }}"
        state: restarted
        enabled: yes
      when: %q == "chrony"
    - name: 启用 systemd-timesyncd
      shell: timedatectl set-ntp true && systemctl restart systemd-timesyncd
      when: %q == "timesyncd"
    - name: 设置时区
      timezone: { name: %q }
    - name: 强制校时
      shell: chronyc makestep 2>/dev/null || true
      when: %q == "chrony"
`,
		o.Tool, o.Tool, o.OnConflict,
		chronyConfBlock,
		o.Tool,
		o.NTPTarget,
		fallbackLine(o.Fallback),
		pollSec,
		o.Tool,
		o.Tool, o.Tool,
		tz,
		o.Tool,
	)

	return "---" + masterPlaybook + clientPlaybook
}

func fallbackLine(fb string) string {
	if strings.TrimSpace(fb) == "" {
		return ""
	}
	return "FallbackNTP=" + fb
}

// ───────────────────────── sys-param ─────────────────────────

func nodeTuneSysParamCmd() *cobra.Command {
	var (
		nodes       string
		onConflict  string
		disableSwap bool
		raiseUlimit bool
		sysctlKV    map[string]string
		extraOnly   bool
		autoInstall bool
		dryRun      bool
	)
	cmd := &cobra.Command{
		Use:   "sys-param",
		Short: "系统参数优化：sysctl + 内核模块（br_netfilter / overlay）+ ulimit + 关闭 swap",
		RunE: func(cmd *cobra.Command, args []string) error {
			if onConflict != "skip" && onConflict != "force" {
				return fmt.Errorf("--on-conflict 必须是 skip 或 force")
			}

			rows := map[string]string{}
			if !extraOnly {
				for k, v := range defaultSysctlRows() {
					rows[k] = v
				}
			}
			for k, v := range sysctlKV {
				k = strings.TrimSpace(k)
				v = strings.TrimSpace(v)
				if k == "" {
					continue
				}
				rows[k] = v
			}
			if len(rows) == 0 {
				return fmt.Errorf("至少需要一项 sysctl 参数（默认含 K8s 必填项；--extra-only 时必须提供 --sysctl）")
			}

			nodeIPs := splitCSV(nodes)
			groups := []ansibleGroup{
				{Name: "targets", IPs: nodeIPs, FallbackLocalhost: true},
			}
			inventory := buildAnsibleInventory(groups)

			playbook := genSysParamPlaybook(sysParamPlaybookOpts{
				Rows:        rows,
				OnConflict:  onConflict,
				DisableSwap: disableSwap,
				RaiseUlimit: raiseUlimit,
			})

			nodeDesc := "本机（localhost）"
			if len(nodeIPs) > 0 {
				nodeDesc = fmt.Sprintf("%d 个节点", len(nodeIPs))
			}
			fmt.Fprintf(os.Stderr, "==> ai-sre 系统参数优化 | %d 项 sysctl%s%s | 节点: %s\n",
				len(rows),
				ifThenElse(disableSwap, " + 关 swap", ""),
				ifThenElse(raiseUlimit, " + ulimit", ""),
				nodeDesc)

			if dryRun {
				printDryRun(inventory, playbook)
				return nil
			}
			return runAnsiblePlaybook(inventory, playbook, autoInstall)
		},
	}
	cmd.Flags().StringVar(&nodes, "nodes", "", "目标节点 IP 列表，逗号分隔（留空：仅对控制机 localhost 执行）")
	cmd.Flags().StringVar(&onConflict, "on-conflict", "skip", "已存在 /etc/sysctl.d/99-ai-sre.conf 时的策略：skip | force")
	cmd.Flags().BoolVar(&disableSwap, "disable-swap", true, "关闭 swap（K8s 必关）")
	cmd.Flags().BoolVar(&raiseUlimit, "raise-ulimit", true, "提升 ulimit 至 655350")
	cmd.Flags().StringToStringVar(&sysctlKV, "sysctl", map[string]string{}, "覆盖/追加 sysctl 项，可多次：--sysctl key=value")
	cmd.Flags().BoolVar(&extraOnly, "extra-only", false, "不附带默认 sysctl，仅使用 --sysctl 显式提供的项")
	cmd.Flags().BoolVar(&autoInstall, "auto-install-ansible", true, "若未安装 ansible 则自动通过 apt/dnf/yum 安装")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "仅打印 inventory 与 playbook，不执行")
	return cmd
}

type sysParamPlaybookOpts struct {
	Rows        map[string]string
	OnConflict  string
	DisableSwap bool
	RaiseUlimit bool
}

// defaultSysctlRows 与 ft-front/src/views/init-tools/InitToolsHome.vue 中
// defaultSysParamRows() 保持一致；K8s 必填项在前。
func defaultSysctlRows() map[string]string {
	return map[string]string{
		"net.ipv4.ip_forward":                 "1",
		"net.bridge.bridge-nf-call-iptables":  "1",
		"net.bridge.bridge-nf-call-ip6tables": "1",
		"vm.swappiness":                       "10",
		"net.core.somaxconn":                  "65535",
		"net.ipv4.tcp_max_tw_buckets":         "6000",
		"fs.file-max":                         "655350",
	}
}

func genSysParamPlaybook(o sysParamPlaybookOpts) string {
	keys := make([]string, 0, len(o.Rows))
	for k := range o.Rows {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString("          ")
		sb.WriteString(k)
		sb.WriteString(" = ")
		sb.WriteString(o.Rows[k])
		sb.WriteString("\n")
	}
	sysctlBlock := strings.TrimRight(sb.String(), "\n")

	return fmt.Sprintf(`---
- name: 系统参数优化 (sysctl + ulimit + 内核模块 + swap)
  hosts: targets
  become: yes
  vars:
    sysctl_file: /etc/sysctl.d/99-ai-sre.conf
    limits_file: /etc/security/limits.d/99-ai-sre.conf
    modules_file: /etc/modules-load.d/ai-sre.conf
  tasks:
    - name: 检测 sysctl drop-in 是否存在
      stat: { path: "{{ sysctl_file }}" }
      register: sysctl_stat
    - name: 跳过（已存在且 on_conflict=skip）
      meta: end_play
      when: sysctl_stat.stat.exists and %q == "skip"
    - name: 加载内核模块 br_netfilter
      modprobe: { name: br_netfilter, state: present }
    - name: 加载内核模块 overlay
      modprobe: { name: overlay, state: present }
    - name: 持久化内核模块
      copy:
        dest: "{{ modules_file }}"
        content: "br_netfilter\noverlay\n"
    - name: 写入 sysctl 参数
      copy:
        dest: "{{ sysctl_file }}"
        content: |
          # Managed by ai-sre sys-param init tool
%s
    - name: 应用 sysctl
      command: sysctl --system
    - name: 写入 ulimit 限制
      copy:
        dest: "{{ limits_file }}"
        content: |
          # Managed by ai-sre sys-param init tool
          *    soft nofile 655350
          *    hard nofile 655350
          root soft nofile 655350
          root hard nofile 655350
      when: %t
    - name: 关闭 swap（立即）
      command: swapoff -a
      ignore_errors: yes
      when: %t
    - name: 注释 fstab 中的 swap 项
      replace:
        path: /etc/fstab
        regexp: '^([^#].*\sswap\s.*)'
        replace: '# \1'
      when: %t
`, o.OnConflict, sysctlBlock, o.RaiseUlimit, o.DisableSwap, o.DisableSwap)
}

// ───────────────────────── 共用辅助 ─────────────────────────

type ansibleGroup struct {
	Name              string
	IPs               []string
	FallbackLocalhost bool
}

func buildAnsibleInventory(groups []ansibleGroup) string {
	var sb strings.Builder
	first := true
	for _, g := range groups {
		var lines []string
		switch {
		case len(g.IPs) > 0:
			for _, ip := range g.IPs {
				lines = append(lines, ip+" ansible_user=root")
			}
		case g.FallbackLocalhost:
			lines = append(lines, "localhost ansible_connection=local")
		default:
			continue
		}
		if !first {
			sb.WriteString("\n")
		}
		first = false
		sb.WriteString("[")
		sb.WriteString(g.Name)
		sb.WriteString("]\n")
		sb.WriteString(strings.Join(lines, "\n"))
		sb.WriteString("\n")
	}
	return sb.String()
}

func printDryRun(inventory, playbook string) {
	fmt.Println("==> inventory")
	fmt.Println(inventory)
	fmt.Println("==> playbook")
	fmt.Println(playbook)
}

// runAnsiblePlaybook 把 inventory / playbook 写入临时目录，调用本机
// ansible-playbook 执行；ansible-playbook 缺失时按 apt/dnf/yum 自动安装。
func runAnsiblePlaybook(inventory, playbook string, autoInstall bool) error {
	if _, err := exec.LookPath("ansible-playbook"); err != nil {
		if !autoInstall {
			return fmt.Errorf("未安装 ansible-playbook，且 --auto-install-ansible=false；请先 apt-get install -y ansible 或 dnf install -y ansible-core")
		}
		fmt.Fprintln(os.Stderr, "==> 检测到未安装 ansible，正在自动安装…")
		if err := installAnsible(); err != nil {
			return fmt.Errorf("自动安装 ansible 失败: %w（请手动安装后重试）", err)
		}
		if _, err := exec.LookPath("ansible-playbook"); err != nil {
			return fmt.Errorf("安装后仍未在 PATH 找到 ansible-playbook：%w", err)
		}
	}

	work, err := os.MkdirTemp("", "ai-sre-node-tune-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)

	hostsPath := filepath.Join(work, "hosts.ini")
	if err := os.WriteFile(hostsPath, []byte(inventory), 0o600); err != nil {
		return err
	}
	playPath := filepath.Join(work, "playbook.yml")
	if err := os.WriteFile(playPath, []byte(playbook), 0o600); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "==> 目标 inventory:")
	fmt.Fprintln(os.Stderr, strings.TrimRight(inventory, "\n"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "==> 开始执行 Ansible Playbook...")
	c := exec.Command("ansible-playbook", "-i", hostsPath, playPath, "--timeout", "60")
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	if err := c.Run(); err != nil {
		return fmt.Errorf("ansible-playbook 执行失败: %w", err)
	}
	fmt.Fprintln(os.Stderr, "==> 完成")
	return nil
}

// installAnsible 复用前端 ANSIBLE_BOOTSTRAP 的策略：apt-get / dnf / yum。
func installAnsible() error {
	cases := [][]string{
		{"apt-get", "update", "-qq"},
		{"apt-get", "install", "-y", "-q", "ansible"},
	}
	if _, err := exec.LookPath("apt-get"); err == nil {
		for _, args := range cases {
			c := exec.Command(args[0], args[1:]...)
			c.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			if err := c.Run(); err != nil {
				return fmt.Errorf("%s: %w", strings.Join(args, " "), err)
			}
		}
		return nil
	}
	if _, err := exec.LookPath("dnf"); err == nil {
		c := exec.Command("dnf", "install", "-y", "ansible-core")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err == nil {
			return nil
		}
		c = exec.Command("dnf", "install", "-y", "ansible")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	}
	if _, err := exec.LookPath("yum"); err == nil {
		_ = exec.Command("yum", "install", "-y", "epel-release").Run()
		c := exec.Command("yum", "install", "-y", "ansible")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	}
	return fmt.Errorf("未识别到 apt-get / dnf / yum，无法自动安装 ansible")
}

func ifThenElse(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
