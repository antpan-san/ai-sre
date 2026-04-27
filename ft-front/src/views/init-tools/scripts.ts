/**
 * 初始化工具卡片的脚本生成器（Ansible 模式）
 *
 * 执行模型：
 *   1. 用户在 UI 填写节点 IP 和参数
 *   2. 前端生成一个 Shell 脚本（内嵌 Ansible Playbook）
 *   3. 用户在控制机上执行该脚本（或通过 curl 拉取执行）
 *   4. 脚本自动安装 ansible（如未安装），创建临时 inventory，运行 Playbook
 *   5. Ansible 对所有目标节点进行多机器操作，幂等安全
 */

export type OnConflict = 'skip' | 'force'

export interface ScriptBundle {
  /** 弹窗副标题 */
  subtitle: string
  /** 完整可执行 Shell 脚本（内嵌 Ansible Playbook） */
  fullScript: string
  /** ai-sre CLI 等价命令（部分卡片仍是 roadmap 占位） */
  aiSreCommand: string
  /**
   * ai-sre CLI 命令是否真正在 ai-sre 中实现并可直接执行：
   *   - true: 复制后 sudo bash 即可工作（如 time-sync / sys-param）；
   *   - false 或省略: roadmap 预览，命令本身在当前 ai-sre 版本中未实现。
   * 该值由前端 ScriptPreviewDialog 决定 CLI Tab 的执行/复制提示。
   */
  aiSreCommandExecutable?: boolean
  /** curl 一键执行示例 */
  curlOneLiner: string
}

// ─────────────────────────────────────────────────────────────────────────────
// 通用工具
// ─────────────────────────────────────────────────────────────────────────────

const shellQuote = (s: string): string => {
  if (s === '') return "''"
  if (/^[A-Za-z0-9_./:-]+$/.test(s)) return s
  return `'${s.replace(/'/g, "'\\''")}'`
}

/** 将节点 IP 字符串（换行/逗号分隔）解析为数组 */
export const parseNodeIps = (raw: string): string[] =>
  raw.split(/[\n,]+/).map(s => s.trim()).filter(Boolean)

/** 生成 Ansible inventory ini 文本 */
const buildInventory = (
  groups: { name: string; ips: string[]; fallbackLocalhost?: boolean }[],
): string => {
  return groups
    .map(({ name, ips, fallbackLocalhost }) => {
      const lines = ips.length > 0
        ? ips.map(ip => `${ip} ansible_user=root`)
        : (fallbackLocalhost ? ['localhost ansible_connection=local'] : [])
      if (lines.length === 0) return null
      return `[${name}]\n${lines.join('\n')}`
    })
    .filter(Boolean)
    .join('\n\n')
}

/** Ansible 安装引导 (apt/dnf/yum) */
const ANSIBLE_BOOTSTRAP = `# ── 1. 确保 ansible 已安装 ──────────────────────────────────────────────────
if ! command -v ansible-playbook &>/dev/null; then
  echo "==> 检测到未安装 ansible，正在安装..."
  if   command -v apt-get &>/dev/null; then
    DEBIAN_FRONTEND=noninteractive apt-get update -qq
    DEBIAN_FRONTEND=noninteractive apt-get install -y -q ansible
  elif command -v dnf &>/dev/null; then
    dnf install -y ansible-core 2>/dev/null || dnf install -y ansible
  elif command -v yum &>/dev/null; then
    yum install -y epel-release 2>/dev/null || true
    yum install -y ansible
  else
    echo "Error: 无法自动安装 ansible，请先手动安装后重试"; exit 1
  fi
  echo "==> ansible 安装完成: $(ansible --version | head -1)"
fi`

/** 脚本尾部：展示 inventory 并运行 Playbook */
const runPlaybook = (extraVarsLines: string[] = []): string => {
  const extraVarsStr = extraVarsLines.length > 0
    ? `  --extra-vars "${extraVarsLines.join(' ')}"`
    : ''
  return `# ── 5. 执行 ──────────────────────────────────────────────────────────────────
echo ""
echo "==> 目标 inventory:"
cat "$WORK/hosts.ini"
echo ""
echo "==> 开始执行 Ansible Playbook..."
ansible-playbook -i "$WORK/hosts.ini" "$WORK/playbook.yml" \\
  --timeout 60${extraVarsStr ? '\n' + extraVarsStr : ''}
echo ""
echo "==> 完成"`
}

// ─────────────────────────────────────────────────────────────────────────────
// 1) 时间同步
// ─────────────────────────────────────────────────────────────────────────────
export interface TimeSyncOptions {
  /** 模式：公用 NTP 服务器 或 自建主节点 */
  ntpMode: 'public' | 'self-hosted'
  /** 自建模式：NTP 主节点 IP（此节点将安装并运行 chrony 服务端） */
  masterNodeIp: string
  /** 目标客户端节点 IP，换行分隔；留空则仅本机执行 */
  clientNodeIps: string
  /** 公用 NTP 模式的主服务器地址 */
  ntpServer: string
  fallbackNtpServer: string
  timezone: string
  syncIntervalMin: number
  preferredTool: 'chrony' | 'timesyncd'
  onConflict: OnConflict
}

export function genTimeSyncScript(opts: TimeSyncOptions): ScriptBundle {
  const clientIps = parseNodeIps(opts.clientNodeIps)
  const masterIp = opts.masterNodeIp.trim()
  const isSelfHosted = opts.ntpMode === 'self-hosted'

  // NTP 服务器地址（公用 or 自建主节点 IP）
  const ntpTarget = isSelfHosted ? (masterIp || '<MASTER_IP>') : opts.ntpServer
  const fallback = isSelfHosted ? '' : opts.fallbackNtpServer

  // Inventory
  const inventoryGroups: { name: string; ips: string[]; fallbackLocalhost?: boolean }[] = [
    { name: 'clients', ips: clientIps, fallbackLocalhost: true },
  ]
  if (isSelfHosted && masterIp) {
    inventoryGroups.push({ name: 'ntp_master', ips: [masterIp] })
  }
  const inventory = buildInventory(inventoryGroups)

  // Chrony config template
  const chronyConf = (server: string, fb: string) =>
    `server ${server} iburst prefer\n` +
    (fb ? `server ${fb} iburst\n` : '') +
    `makestep 1.0 3\nrtcsync\ndriftfile /var/lib/chrony/chrony.drift\nlogdir /var/log/chrony`

  // Ansible Playbook YAML (written as string, substitutions already done)
  const masterPlaybook = isSelfHosted && masterIp ? `
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
      timezone: { name: "${opts.timezone}" }
` : ''

  const clientPlaybook = `
- name: 配置 NTP 客户端节点
  hosts: clients
  become: yes
  tasks:
    - name: 安装 chrony (Debian/Ubuntu)
      apt: { name: chrony, state: present, update_cache: yes }
      when: ansible_os_family == "Debian" and "${opts.preferredTool}" == "chrony"
    - name: 安装 chrony (RedHat/CentOS)
      yum: { name: chrony, state: present }
      when: ansible_os_family == "RedHat" and "${opts.preferredTool}" == "chrony"
    - name: 检测现有时间同步服务
      shell: systemctl is-active chrony chronyd ntpd systemd-timesyncd 2>/dev/null | grep -c active || true
      register: existing_ntp
      changed_when: false
    - name: 跳过（ON_CONFLICT=skip 且已有服务）
      meta: end_play
      when: existing_ntp.stdout | int > 0 and "${opts.onConflict}" == "skip"
    - name: 配置 chrony 客户端
      copy:
        dest: "{{ '/etc/chrony/chrony.conf' if ansible_os_family == 'Debian' else '/etc/chrony.conf' }}"
        content: |
          # Managed by ai-sre time-sync — NTP Client
          ${chronyConf(ntpTarget, fallback).split('\n').join('\n          ')}
      when: "${opts.preferredTool}" == "chrony"
    - name: 配置 systemd-timesyncd
      copy:
        dest: /etc/systemd/timesyncd.conf
        content: |
          [Time]
          NTP=${ntpTarget}
          ${fallback ? 'FallbackNTP=' + fallback : ''}
          PollIntervalMinSec=${opts.syncIntervalMin * 60}
      when: "${opts.preferredTool}" == "timesyncd"
    - name: 启动并开机自启 chrony
      service:
        name: "{{ 'chrony' if ansible_os_family == 'Debian' else 'chronyd' }}"
        state: restarted
        enabled: yes
      when: "${opts.preferredTool}" == "chrony"
    - name: 启用 systemd-timesyncd
      shell: timedatectl set-ntp true && systemctl restart systemd-timesyncd
      when: "${opts.preferredTool}" == "timesyncd"
    - name: 设置时区
      timezone: { name: "${opts.timezone}" }
    - name: 强制校时
      shell: chronyc makestep 2>/dev/null || true
      when: "${opts.preferredTool}" == "chrony"
`

  const playbook = `---${masterPlaybook}${clientPlaybook}`

  const nodeDesc = clientIps.length > 0
    ? `${clientIps.length} 个客户端节点${isSelfHosted && masterIp ? ' + 1 个主节点' : ''}`
    : '本机（localhost）'
  const modeDesc = isSelfHosted ? `自建主节点 ${ntpTarget}` : `公用 NTP: ${ntpTarget}`

  const fullScript = `#!/usr/bin/env bash
# ai-sre init-tools: time-sync (Ansible)
# 在控制机上执行：bash time-sync.sh
# Ansible 将同时配置所有目标节点，无需逐台 SSH
set -euo pipefail

echo "==> ai-sre 时间同步 | 模式: ${opts.ntpMode === 'self-hosted' ? '自建主节点' : '公用 NTP'} | 节点: ${nodeDesc}"

${ANSIBLE_BOOTSTRAP}

# ── 2. 临时工作目录 ────────────────────────────────────────────────────────
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

# ── 3. 生成 inventory ─────────────────────────────────────────────────────
cat > "$WORK/hosts.ini" << 'INVENTORY'
${inventory}
INVENTORY

# ── 4. 生成 Ansible Playbook ──────────────────────────────────────────────
cat > "$WORK/playbook.yml" << 'PLAYBOOK'
${playbook}
PLAYBOOK

${runPlaybook()}
`

  const aiSreCommand = `ai-sre node tune time-sync \\
  --ntp-mode ${opts.ntpMode} \\
${isSelfHosted ? `  --master-node ${shellQuote(masterIp)} \\\n` : ''}  --clients ${shellQuote(clientIps.join(',') || 'localhost')} \\
  --ntp-server ${shellQuote(ntpTarget)} \\
  --timezone ${shellQuote(opts.timezone)} \\
  --tool ${opts.preferredTool} \\
  --on-conflict ${opts.onConflict}`

  return {
    subtitle: `配置时间同步（${modeDesc}）→ ${nodeDesc}`,
    fullScript,
    aiSreCommand,
    aiSreCommandExecutable: true,
    curlOneLiner: `# 将脚本下载到控制机后执行：\nbash time-sync.sh\n\n# 或未来通过 ai-sre 控制台 curl 一键执行（需后端接口）:\n# bash -c "$(curl -fsSL http://<控制台IP>:9080/ft-api/api/init-tools/scripts/time-sync.sh)"`,
  }
}

// ─────────────────────────────────────────────────────────────────────────────
// 2) 系统参数优化
// ─────────────────────────────────────────────────────────────────────────────
export interface SysParamRow {
  key: string
  value: string
  description: string
  required: boolean
}

export interface SysParamOptions {
  /** 目标节点 IP，换行分隔；留空则仅本机执行 */
  nodeIps: string
  rows: SysParamRow[]
  onConflict: OnConflict
  disableSwap: boolean
  raiseUlimit: boolean
}

export function genSysParamScript(opts: SysParamOptions): ScriptBundle {
  const nodeIps = parseNodeIps(opts.nodeIps)
  const inventory = buildInventory([{ name: 'targets', ips: nodeIps, fallbackLocalhost: true }])

  const sysctlLines = opts.rows.map(r => `${r.key} = ${r.value}`).join('\n          ')
  const nodeDesc = nodeIps.length > 0 ? `${nodeIps.length} 个节点` : '本机（localhost）'

  const playbook = `---
- name: 系统参数优化 (sysctl + ulimit + 内核模块 + swap)
  hosts: targets
  become: yes
  vars:
    sysctl_file: /etc/sysctl.d/99-ai-sre.conf
    limits_file: /etc/security/limits.d/99-ai-sre.conf
    modules_file: /etc/modules-load.d/ai-sre.conf
  tasks:
    - name: 跳过（on_conflict=skip 且已存在配置）
      stat: { path: "{{ sysctl_file }}" }
      register: sysctl_stat
    - name: 退出（已存在且 skip）
      meta: end_play
      when: sysctl_stat.stat.exists and "${opts.onConflict}" == "skip"
    - name: 加载内核模块 br_netfilter
      modprobe: { name: br_netfilter, state: present }
    - name: 加载内核模块 overlay
      modprobe: { name: overlay, state: present }
    - name: 持久化内核模块
      copy:
        dest: "{{ modules_file }}"
        content: "br_netfilter\\noverlay\\n"
    - name: 写入 sysctl 参数
      copy:
        dest: "{{ sysctl_file }}"
        content: |
          # Managed by ai-sre sys-param init tool
          ${sysctlLines}
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
      when: ${opts.raiseUlimit}
    - name: 关闭 swap（立即）
      command: swapoff -a
      ignore_errors: yes
      when: ${opts.disableSwap}
    - name: 注释 fstab 中的 swap 项
      replace:
        path: /etc/fstab
        regexp: '^([^#].*\\sswap\\s.*)'
        replace: '# \\1'
      when: ${opts.disableSwap}
`

  const fullScript = `#!/usr/bin/env bash
# ai-sre init-tools: sys-param (Ansible)
set -euo pipefail

echo "==> ai-sre 系统参数优化 | 节点: ${nodeDesc}"

${ANSIBLE_BOOTSTRAP}

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

cat > "$WORK/hosts.ini" << 'INVENTORY'
${inventory}
INVENTORY

cat > "$WORK/playbook.yml" << 'PLAYBOOK'
${playbook}
PLAYBOOK

${runPlaybook()}
`

  const sysctlFlags = opts.rows
    .map(r => `  --sysctl ${shellQuote(`${r.key}=${r.value}`)} \\`)
    .join('\n')
  return {
    subtitle: `写入 ${opts.rows.length} 项 sysctl + 内核模块${opts.disableSwap ? ' + 关 swap' : ''}${opts.raiseUlimit ? ' + ulimit' : ''} → ${nodeDesc}`,
    fullScript,
    aiSreCommand: `ai-sre node tune sys-param \\
  --nodes ${shellQuote(nodeIps.join(',') || 'localhost')} \\
  --on-conflict ${opts.onConflict} \\
  --disable-swap=${opts.disableSwap} \\
  --raise-ulimit=${opts.raiseUlimit} \\
  --extra-only \\
${sysctlFlags.replace(/ \\$/, '')}`,
    aiSreCommandExecutable: true,
    curlOneLiner: `bash sys-param.sh\n\n# 或未来 curl 一键:\n# bash -c "$(curl -fsSL http://<控制台IP>:9080/ft-api/api/init-tools/scripts/sys-param.sh)"`,
  }
}

// ─────────────────────────────────────────────────────────────────────────────
// 3) 系统安全加固
// ─────────────────────────────────────────────────────────────────────────────
export interface SecurityOptions {
  /** 目标节点 IP，换行分隔 */
  nodeIps: string
  disableSshRoot: boolean
  changeSshPort: boolean
  sshPort: number
  enableFirewall: boolean
  disableUnneeded: boolean
  enableAutoUpdate: boolean
  installFail2ban: boolean
  onConflict: OnConflict
}

export function genSecurityScript(opts: SecurityOptions): ScriptBundle {
  const nodeIps = parseNodeIps(opts.nodeIps)
  const inventory = buildInventory([{ name: 'targets', ips: nodeIps, fallbackLocalhost: true }])
  const nodeDesc = nodeIps.length > 0 ? `${nodeIps.length} 个节点` : '本机（localhost）'

  const sshLines = [
    '# Managed by ai-sre security init tool',
    opts.disableSshRoot ? 'PermitRootLogin no' : null,
    opts.changeSshPort ? `Port ${opts.sshPort}` : null,
    'PasswordAuthentication yes',
    'ClientAliveInterval 60',
    'ClientAliveCountMax 3',
  ].filter(Boolean).join('\n          ')

  const playbook = `---
- name: 系统安全加固
  hosts: targets
  become: yes
  tasks:
    - name: 检查 SSH 配置是否已存在
      stat: { path: /etc/ssh/sshd_config.d/99-ai-sre.conf }
      register: ssh_dropin
    - name: 写入 SSH 加固配置 (drop-in)
      copy:
        dest: /etc/ssh/sshd_config.d/99-ai-sre.conf
        content: |
          ${sshLines}
      when: not ssh_dropin.stat.exists or "${opts.onConflict}" == "force"
    - name: 验证并重载 sshd
      shell: sshd -t && (systemctl reload sshd || systemctl reload ssh)
      ignore_errors: yes
      when: not ssh_dropin.stat.exists or "${opts.onConflict}" == "force"
    - name: 启用 ufw (Debian/Ubuntu)
      shell: |
        ufw --force enable
        ufw allow ${opts.changeSshPort ? opts.sshPort : 22}/tcp
      when: ansible_os_family == "Debian" and ${opts.enableFirewall}
      ignore_errors: yes
    - name: 启用 firewalld (RedHat)
      shell: |
        systemctl enable --now firewalld
        firewall-cmd --permanent --add-port=${opts.changeSshPort ? opts.sshPort : 22}/tcp
        firewall-cmd --reload
      when: ansible_os_family == "RedHat" and ${opts.enableFirewall}
      ignore_errors: yes
    - name: 禁用无用服务
      service: { name: "{{ item }}", state: stopped, enabled: no }
      loop: [cups, avahi-daemon, bluetooth, ModemManager]
      ignore_errors: yes
      when: ${opts.disableUnneeded}
    - name: 安装 fail2ban (Debian)
      apt: { name: fail2ban, state: present, update_cache: yes }
      when: ansible_os_family == "Debian" and ${opts.installFail2ban}
    - name: 安装 fail2ban (RedHat)
      yum: { name: fail2ban, state: present }
      when: ansible_os_family == "RedHat" and ${opts.installFail2ban}
    - name: 启用 fail2ban
      service: { name: fail2ban, state: started, enabled: yes }
      when: ${opts.installFail2ban}
      ignore_errors: yes
`

  const fullScript = `#!/usr/bin/env bash
# ai-sre init-tools: security (Ansible)
set -euo pipefail

echo "==> ai-sre 系统安全加固 | 节点: ${nodeDesc}"

${ANSIBLE_BOOTSTRAP}

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

cat > "$WORK/hosts.ini" << 'INVENTORY'
${inventory}
INVENTORY

cat > "$WORK/playbook.yml" << 'PLAYBOOK'
${playbook}
PLAYBOOK

${runPlaybook()}
`

  return {
    subtitle: `SSH/防火墙${opts.installFail2ban ? '/Fail2ban' : ''} 加固 → ${nodeDesc}`,
    fullScript,
    aiSreCommand: `ai-sre node tune security \\
  --nodes ${shellQuote(nodeIps.join(',') || 'localhost')} \\
  --on-conflict ${opts.onConflict} \\
  --disable-root-ssh=${opts.disableSshRoot} \\
  --ssh-port ${opts.changeSshPort ? opts.sshPort : 22} \\
  --firewall=${opts.enableFirewall} \\
  --fail2ban=${opts.installFail2ban}`,
    curlOneLiner: `bash security.sh\n\n# 或未来 curl 一键:\n# bash -c "$(curl -fsSL http://<控制台IP>:9080/ft-api/api/init-tools/scripts/security.sh)"`,
  }
}

// ─────────────────────────────────────────────────────────────────────────────
// 4) 磁盘分区优化
// ─────────────────────────────────────────────────────────────────────────────
export interface DiskOptions {
  /** 目标节点 IP，换行分隔 */
  nodeIps: string
  enableSsdTrim: boolean
  tuneFilesystem: boolean
  setupSwap: boolean
  swapSize: string
  onConflict: OnConflict
}

export function genDiskScript(opts: DiskOptions): ScriptBundle {
  const nodeIps = parseNodeIps(opts.nodeIps)
  const inventory = buildInventory([{ name: 'targets', ips: nodeIps, fallbackLocalhost: true }])
  const nodeDesc = nodeIps.length > 0 ? `${nodeIps.length} 个节点` : '本机（localhost）'

  const playbook = `---
- name: 磁盘分区优化
  hosts: targets
  become: yes
  tasks:
    - name: 备份 fstab
      copy:
        src: /etc/fstab
        dest: "/var/backups/fstab.ai-sre.{{ ansible_date_time.iso8601_basic }}"
        remote_src: yes
      ignore_errors: yes
    - name: 启用 SSD TRIM (fstrim.timer)
      service: { name: fstrim.timer, state: started, enabled: yes }
      when: ${opts.enableSsdTrim}
      ignore_errors: yes
    - name: 为根分区添加 noatime（ext4/xfs）
      replace:
        path: /etc/fstab
        regexp: '(\\s/\\s+(ext4|xfs)\\s+)(\\S+)'
        replace: '\\1\\3,noatime'
      when: ${opts.tuneFilesystem}
      ignore_errors: yes
    - name: 检查是否已有 swap
      command: swapon --show=NAME --noheadings
      register: swap_status
      changed_when: false
      ignore_errors: yes
    - name: 跳过 swap（已存在且 skip）
      meta: end_play
      when: ${opts.setupSwap} and swap_status.stdout | length > 0 and "${opts.onConflict}" == "skip"
    - name: 创建 swapfile
      shell: |
        swapoff -a || true
        rm -f /swapfile
        fallocate -l ${opts.swapSize === 'auto' ? '$(awk \'/MemTotal/{printf "%dM", $2/512}\' /proc/meminfo)' : opts.swapSize} /swapfile
        chmod 600 /swapfile
        mkswap /swapfile
        swapon /swapfile
        grep -q '^/swapfile' /etc/fstab || echo '/swapfile none swap sw 0 0' >> /etc/fstab
      when: ${opts.setupSwap}
`

  const fullScript = `#!/usr/bin/env bash
# ai-sre init-tools: disk (Ansible)
set -euo pipefail

echo "==> ai-sre 磁盘优化 | 节点: ${nodeDesc}"

${ANSIBLE_BOOTSTRAP}

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

cat > "$WORK/hosts.ini" << 'INVENTORY'
${inventory}
INVENTORY

cat > "$WORK/playbook.yml" << 'PLAYBOOK'
${playbook}
PLAYBOOK

${runPlaybook()}
`

  return {
    subtitle: `磁盘优化（${[opts.enableSsdTrim && 'TRIM', opts.tuneFilesystem && 'noatime', opts.setupSwap && 'swap'].filter(Boolean).join('/')}）→ ${nodeDesc}`,
    fullScript,
    aiSreCommand: `ai-sre node tune disk \\
  --nodes ${shellQuote(nodeIps.join(',') || 'localhost')} \\
  --on-conflict ${opts.onConflict} \\
  --ssd-trim=${opts.enableSsdTrim} \\
  --tune-fs=${opts.tuneFilesystem} \\
  --setup-swap=${opts.setupSwap} \\
  --swap-size ${opts.swapSize}`,
    curlOneLiner: `bash disk.sh\n\n# 或未来 curl 一键:\n# bash -c "$(curl -fsSL http://<控制台IP>:9080/ft-api/api/init-tools/scripts/disk.sh)"`,
  }
}
