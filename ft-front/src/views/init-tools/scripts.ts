/**
 * 初始化工具卡片的脚本与 CLI 命令生成器。
 *
 * 设计原则：
 * - 每个生成器输出三种产物：
 *   1. fullScript：完整可执行的 bash 脚本（含 set -euo pipefail / 存在检测 / 幂等写入 / 验证）
 *   2. aiSreCommand：未来 ai-sre CLI 的等价调用（参数化）
 *   3. batchOneLiner：多节点批量执行示例（ssh 循环或 curl-pipe）
 * - 所有脚本内默认 ON_CONFLICT=skip：检测到已有同类配置时直接退出，输出当前状态
 * - 通过环境变量传入参数，便于 curl-pipe / 远程 SSH 执行
 * - 关键文件备份至 /var/backups/ai-sre/<timestamp>/，提示回滚命令
 */

import type { OsType, NodeSystemValue } from '../../components/init-tools/NodeSystemSelector.vue'

export type OnConflict = 'skip' | 'force'

export interface ScriptBundle {
  /** 弹窗顶部的副标题，例如 "在 3 个节点上配置时间同步" */
  subtitle: string
  /** 完整 bash 脚本 */
  fullScript: string
  /** ai-sre CLI 等价命令（roadmap 占位） */
  aiSreCommand: string
  /** 多节点批量执行示例 */
  batchOneLiner: string
  /** 在单节点上的 curl 一键示例（占位 URL，需后端 /scripts 接口配合） */
  curlOneLiner: string
}

const shellQuote = (s: string): string => {
  if (s === '') return "''"
  if (/^[A-Za-z0-9_./:-]+$/.test(s)) return s
  return `'${s.replace(/'/g, "'\\''")}'`
}

const sshTargets = (target: NodeSystemValue): string => {
  if (!target.nodes.length) return '<NODE_IPS>'
  return target.nodes.join(' ')
}

const formatBatchOneLiner = (target: NodeSystemValue, scriptName: string): string => {
  const nodes = sshTargets(target)
  return [
    `# 将下方完整脚本保存为 ${scriptName} 后，对所有目标节点逐一执行：`,
    `for ip in ${nodes}; do`,
    `  echo "==> $ip"`,
    `  ssh -o StrictHostKeyChecking=accept-new root@$ip "bash -s" < ${scriptName}`,
    `done`,
  ].join('\n')
}

const formatCurlOneLiner = (
  endpoint: string,
  envVars: Record<string, string | number | boolean>,
): string => {
  // 占位：需要后端提供 /ft-api/api/init-tools/scripts/<name>.sh
  // 当前直接给出执行模板，让用户清楚未来交付形态
  const envInline = Object.entries(envVars)
    .map(([k, v]) => `${k}=${shellQuote(String(v))}`)
    .join(' ')
  return `${envInline} bash -c "$(curl -fsSL ${endpoint})"`
}

// =====================================================================
// 1) Time Sync
// =====================================================================
export interface TimeSyncOptions {
  ntpServer: string
  fallbackNtpServer: string
  timezone: string
  syncIntervalMin: number
  preferredTool: 'chrony' | 'timesyncd'
  onConflict: OnConflict
}

export function genTimeSyncScript(target: NodeSystemValue, opts: TimeSyncOptions): ScriptBundle {
  const env = {
    NTP_SERVER: opts.ntpServer,
    FALLBACK_NTP: opts.fallbackNtpServer,
    TIMEZONE: opts.timezone,
    SYNC_INTERVAL: opts.syncIntervalMin,
    PREFERRED_TOOL: opts.preferredTool,
    ON_CONFLICT: opts.onConflict,
  }

  const fullScript = `#!/usr/bin/env bash
# ai-sre node tune time-sync
# Generated for: nodes=${sshTargets(target)} os=${target.osType || '<auto>'}
# Idempotent: 已存在 chrony / ntpd / systemd-timesyncd 等服务时按 ON_CONFLICT 决策
set -euo pipefail

NTP_SERVER="\${NTP_SERVER:-${opts.ntpServer}}"
FALLBACK_NTP="\${FALLBACK_NTP:-${opts.fallbackNtpServer}}"
TIMEZONE="\${TIMEZONE:-${opts.timezone}}"
SYNC_INTERVAL="\${SYNC_INTERVAL:-${opts.syncIntervalMin}}"   # 分钟
PREFERRED_TOOL="\${PREFERRED_TOOL:-${opts.preferredTool}}"   # chrony | timesyncd
ON_CONFLICT="\${ON_CONFLICT:-${opts.onConflict}}"            # skip | force

log()  { printf '[time-sync] %s\\n' "$*"; }
warn() { printf '[time-sync][warn] %s\\n' "$*" >&2; }

# 1) 探测已存在的时间同步服务
detect_existing() {
  local found=()
  for svc in chrony chronyd ntp ntpd ntpsec systemd-timesyncd; do
    if systemctl is-active --quiet "$svc" 2>/dev/null; then
      found+=("$svc")
    fi
  done
  printf '%s\\n' "\${found[@]+\${found[@]}}"
}

existing="$(detect_existing)"
if [ -n "$existing" ]; then
  log "检测到已存在时间同步工具:"
  echo "$existing" | sed 's/^/  - /'
  if [ "$ON_CONFLICT" = "skip" ]; then
    log "ON_CONFLICT=skip，已有其他时间同步配置，跳过本次安装与配置"
    log "当前 timedatectl 状态:"
    timedatectl status 2>/dev/null | sed 's/^/  /' || true
    exit 0
  fi
  log "ON_CONFLICT=force，将停用并覆盖现有配置"
  for svc in $existing; do systemctl disable --now "$svc" 2>/dev/null || true; done
fi

# 2) 安装时间同步工具
install_chrony() {
  if command -v apt-get >/dev/null 2>&1; then
    DEBIAN_FRONTEND=noninteractive apt-get update -y
    DEBIAN_FRONTEND=noninteractive apt-get install -y chrony
  elif command -v dnf >/dev/null 2>&1; then
    dnf install -y chrony
  elif command -v yum >/dev/null 2>&1; then
    yum install -y chrony
  elif command -v zypper >/dev/null 2>&1; then
    zypper -n install chrony
  else
    warn "未识别的包管理器（apt/dnf/yum/zypper），无法自动安装 chrony"
    exit 1
  fi
}

case "$PREFERRED_TOOL" in
  chrony)
    install_chrony
    CHRONY_CONF=/etc/chrony/chrony.conf
    [ -d /etc/chrony ] || CHRONY_CONF=/etc/chrony.conf
    cat > "$CHRONY_CONF" <<EOF
# Managed by ai-sre time-sync init tool
server $NTP_SERVER iburst prefer
$( [ -n "$FALLBACK_NTP" ] && echo "server $FALLBACK_NTP iburst" )
makestep 1.0 3
rtcsync
driftfile /var/lib/chrony/chrony.drift
logdir /var/log/chrony
EOF
    systemctl enable --now chrony 2>/dev/null || systemctl enable --now chronyd
    sleep 2
    chronyc -a 'burst 4/4' 2>/dev/null || true
    chronyc -a makestep 2>/dev/null || true
    ;;
  timesyncd)
    if [ ! -f /etc/systemd/timesyncd.conf ]; then
      warn "当前系统未提供 systemd-timesyncd，请改用 chrony"
      exit 1
    fi
    cat > /etc/systemd/timesyncd.conf <<EOF
[Time]
NTP=$NTP_SERVER
$( [ -n "$FALLBACK_NTP" ] && echo "FallbackNTP=$FALLBACK_NTP" )
RootDistanceMaxSec=5
PollIntervalMinSec=$(( SYNC_INTERVAL * 60 ))
PollIntervalMaxSec=$(( SYNC_INTERVAL * 60 * 4 ))
EOF
    timedatectl set-ntp true
    systemctl restart systemd-timesyncd
    ;;
  *) warn "未知 PREFERRED_TOOL=$PREFERRED_TOOL"; exit 1 ;;
esac

# 3) 时区
timedatectl set-timezone "$TIMEZONE"

# 4) 验证
log "时区: $(timedatectl show -p Timezone --value 2>/dev/null || readlink /etc/localtime)"
log "timedatectl:"
timedatectl status 2>/dev/null | sed 's/^/  /' || true
if command -v chronyc >/dev/null 2>&1; then
  log "chronyc tracking:"
  chronyc tracking 2>/dev/null | sed 's/^/  /' || true
fi
log "完成"
`

  const aiSreCommand = `ai-sre node tune time-sync \\
  --nodes ${target.nodes.join(',') || '<NODE_IPS>'} \\
  --os ${target.osType || '<auto>'} \\
  --ntp-server ${shellQuote(opts.ntpServer)} \\
  --fallback-ntp ${shellQuote(opts.fallbackNtpServer)} \\
  --timezone ${shellQuote(opts.timezone)} \\
  --sync-interval ${opts.syncIntervalMin} \\
  --tool ${opts.preferredTool} \\
  --on-conflict ${opts.onConflict}`

  return {
    subtitle: `在 ${target.nodes.length || '<目标>'} 个节点上配置时间同步（${opts.preferredTool} → ${opts.ntpServer}）`,
    fullScript,
    aiSreCommand,
    batchOneLiner: formatBatchOneLiner(target, 'time-sync.sh'),
    curlOneLiner: formatCurlOneLiner('https://<api-host>/ft-api/api/init-tools/scripts/time-sync.sh', env),
  }
}

// =====================================================================
// 2) System Param (sysctl + ulimit + 模块加载)
// =====================================================================
export interface SysParamRow {
  key: string
  value: string
  description: string
  required: boolean
}

export interface SysParamOptions {
  rows: SysParamRow[]
  onConflict: OnConflict
  disableSwap: boolean
  raiseUlimit: boolean
}

export function genSysParamScript(target: NodeSystemValue, opts: SysParamOptions): ScriptBundle {
  const sysctlContent = opts.rows.map(r => `${r.key} = ${r.value}`).join('\n')
  const env = {
    ON_CONFLICT: opts.onConflict,
    DISABLE_SWAP: opts.disableSwap ? '1' : '0',
    RAISE_ULIMIT: opts.raiseUlimit ? '1' : '0',
  }

  const fullScript = `#!/usr/bin/env bash
# ai-sre node tune sys-param
# Generated for: nodes=${sshTargets(target)} os=${target.osType || '<auto>'}
set -euo pipefail

ON_CONFLICT="\${ON_CONFLICT:-${opts.onConflict}}"
DISABLE_SWAP="\${DISABLE_SWAP:-${opts.disableSwap ? 1 : 0}}"
RAISE_ULIMIT="\${RAISE_ULIMIT:-${opts.raiseUlimit ? 1 : 0}}"

SYSCTL_FILE=/etc/sysctl.d/99-ai-sre.conf
LIMITS_FILE=/etc/security/limits.d/99-ai-sre.conf
MODULES_FILE=/etc/modules-load.d/ai-sre.conf
BACKUP_DIR=/var/backups/ai-sre/$(date +%Y%m%d-%H%M%S)

log()  { printf '[sys-param] %s\\n' "$*"; }
warn() { printf '[sys-param][warn] %s\\n' "$*" >&2; }

# 1) 已存在配置检测
if [ -f "$SYSCTL_FILE" ] && [ "$ON_CONFLICT" = "skip" ]; then
  log "已存在 $SYSCTL_FILE，ON_CONFLICT=skip，跳过本次写入"
  log "当前内容:"
  sed 's/^/  /' "$SYSCTL_FILE"
  exit 0
fi

mkdir -p "$BACKUP_DIR"
[ -f "$SYSCTL_FILE" ] && cp "$SYSCTL_FILE" "$BACKUP_DIR/"
[ -f "$LIMITS_FILE" ] && cp "$LIMITS_FILE" "$BACKUP_DIR/"

# 2) 加载内核模块（K8s 必须）
log "加载内核模块: br_netfilter overlay"
modprobe br_netfilter || warn "br_netfilter 加载失败"
modprobe overlay      || warn "overlay 加载失败"
cat > "$MODULES_FILE" <<EOF
br_netfilter
overlay
EOF

# 3) 写入 sysctl
cat > "$SYSCTL_FILE" <<'EOF'
# Managed by ai-sre sys-param init tool
${sysctlContent}
EOF
sysctl --system >/dev/null

# 4) ulimit
if [ "$RAISE_ULIMIT" = "1" ]; then
  cat > "$LIMITS_FILE" <<'EOF'
# Managed by ai-sre sys-param init tool
*           soft   nofile      655350
*           hard   nofile      655350
*           soft   nproc       655350
*           hard   nproc       655350
root        soft   nofile      655350
root        hard   nofile      655350
EOF
  log "已写入 $LIMITS_FILE（重新登录后生效）"
fi

# 5) 关闭 swap（K8s 节点必须）
if [ "$DISABLE_SWAP" = "1" ]; then
  swapoff -a || true
  sed -i.bak '/\\sswap\\s/s/^/#/' /etc/fstab || true
  log "已关闭 swap 并注释 /etc/fstab 中的 swap 项"
fi

# 6) 验证
log "=== sysctl 当前值 ==="
${opts.rows.map(r => `sysctl -n ${r.key} 2>/dev/null | sed "s|^|  ${r.key} = |" || true`).join('\n')}

log "=== 已加载模块 ==="
lsmod | egrep '^(br_netfilter|overlay)' | sed 's/^/  /' || warn "模块未加载"

log "完成。回滚: 删除 $SYSCTL_FILE / $LIMITS_FILE / $MODULES_FILE 后 sysctl --system"
`

  const aiSreCommand = `ai-sre node tune sys-param \\
  --nodes ${target.nodes.join(',') || '<NODE_IPS>'} \\
  --os ${target.osType || '<auto>'} \\
  --on-conflict ${opts.onConflict} \\
  --disable-swap=${opts.disableSwap} \\
  --raise-ulimit=${opts.raiseUlimit} \\
  --sysctl ${shellQuote(opts.rows.map(r => `${r.key}=${r.value}`).join(','))}`

  return {
    subtitle: `在 ${target.nodes.length || '<目标>'} 个节点写入 ${opts.rows.length} 项 sysctl + 内核模块 + ulimit`,
    fullScript,
    aiSreCommand,
    batchOneLiner: formatBatchOneLiner(target, 'sys-param.sh'),
    curlOneLiner: formatCurlOneLiner('https://<api-host>/ft-api/api/init-tools/scripts/sys-param.sh', env),
  }
}

// =====================================================================
// 3) Security Hardening
// =====================================================================
export interface SecurityOptions {
  disableSshRoot: boolean
  changeSshPort: boolean
  sshPort: number
  enableFirewall: boolean
  disableUnneeded: boolean
  enableAutoUpdate: boolean
  installFail2ban: boolean
  onConflict: OnConflict
}

export function genSecurityScript(target: NodeSystemValue, opts: SecurityOptions): ScriptBundle {
  const env = {
    ON_CONFLICT: opts.onConflict,
    DISABLE_ROOT_SSH: opts.disableSshRoot ? '1' : '0',
    CHANGE_SSH_PORT: opts.changeSshPort ? '1' : '0',
    SSH_PORT: opts.sshPort,
    ENABLE_FW: opts.enableFirewall ? '1' : '0',
    DISABLE_SERVICES: opts.disableUnneeded ? '1' : '0',
    AUTO_UPDATE: opts.enableAutoUpdate ? '1' : '0',
    INSTALL_FAIL2BAN: opts.installFail2ban ? '1' : '0',
  }

  const fullScript = `#!/usr/bin/env bash
# ai-sre node tune security
# Generated for: nodes=${sshTargets(target)} os=${target.osType || '<auto>'}
# 谨慎执行：会修改 SSH/防火墙等关键设置；运行前自动备份至 /var/backups/ai-sre/<ts>/
set -euo pipefail

ON_CONFLICT="\${ON_CONFLICT:-${opts.onConflict}}"
DISABLE_ROOT_SSH="\${DISABLE_ROOT_SSH:-${opts.disableSshRoot ? 1 : 0}}"
CHANGE_SSH_PORT="\${CHANGE_SSH_PORT:-${opts.changeSshPort ? 1 : 0}}"
SSH_PORT="\${SSH_PORT:-${opts.sshPort}}"
ENABLE_FW="\${ENABLE_FW:-${opts.enableFirewall ? 1 : 0}}"
DISABLE_SERVICES="\${DISABLE_SERVICES:-${opts.disableUnneeded ? 1 : 0}}"
AUTO_UPDATE="\${AUTO_UPDATE:-${opts.enableAutoUpdate ? 1 : 0}}"
INSTALL_FAIL2BAN="\${INSTALL_FAIL2BAN:-${opts.installFail2ban ? 1 : 0}}"

DROPIN=/etc/ssh/sshd_config.d/99-ai-sre.conf
BACKUP_DIR=/var/backups/ai-sre/$(date +%Y%m%d-%H%M%S)

log()  { printf '[security] %s\\n' "$*"; }
warn() { printf '[security][warn] %s\\n' "$*" >&2; }

mkdir -p "$BACKUP_DIR"
[ -f /etc/ssh/sshd_config ] && cp /etc/ssh/sshd_config "$BACKUP_DIR/"

# 1) 已加固检测
if [ -f "$DROPIN" ] && [ "$ON_CONFLICT" = "skip" ]; then
  log "已存在 $DROPIN，ON_CONFLICT=skip，跳过 SSH 加固"
else
  : > "$DROPIN"
  echo "# Managed by ai-sre security init tool" >> "$DROPIN"
  if [ "$DISABLE_ROOT_SSH" = "1" ]; then
    echo "PermitRootLogin no" >> "$DROPIN"
  fi
  if [ "$CHANGE_SSH_PORT" = "1" ]; then
    echo "Port $SSH_PORT" >> "$DROPIN"
  fi
  echo "PasswordAuthentication yes" >> "$DROPIN"
  echo "ClientAliveInterval 60" >> "$DROPIN"
  echo "ClientAliveCountMax 3" >> "$DROPIN"
  if sshd -t 2>/dev/null; then
    systemctl reload sshd 2>/dev/null || systemctl reload ssh 2>/dev/null || true
    log "SSH 加固已应用（drop-in: $DROPIN）"
  else
    warn "sshd -t 校验失败，自动回滚"
    rm -f "$DROPIN"
  fi
fi

# 2) 防火墙
if [ "$ENABLE_FW" = "1" ]; then
  if command -v ufw >/dev/null 2>&1; then
    ufw --force enable
    [ "$CHANGE_SSH_PORT" = "1" ] && ufw allow "$SSH_PORT"/tcp || ufw allow 22/tcp
    log "ufw 已启用"
  elif command -v firewall-cmd >/dev/null 2>&1; then
    systemctl enable --now firewalld || true
    if [ "$CHANGE_SSH_PORT" = "1" ]; then
      firewall-cmd --permanent --add-port="$SSH_PORT"/tcp || true
    else
      firewall-cmd --permanent --add-service=ssh || true
    fi
    firewall-cmd --reload || true
    log "firewalld 已启用"
  else
    warn "未发现 ufw / firewalld，跳过防火墙配置"
  fi
fi

# 3) 关闭无用服务
if [ "$DISABLE_SERVICES" = "1" ]; then
  for svc in cups avahi-daemon bluetooth ModemManager; do
    if systemctl is-active --quiet "$svc" 2>/dev/null; then
      systemctl disable --now "$svc" || true
      log "已禁用: $svc"
    fi
  done
fi

# 4) 自动安全更新
if [ "$AUTO_UPDATE" = "1" ]; then
  if command -v apt-get >/dev/null 2>&1; then
    DEBIAN_FRONTEND=noninteractive apt-get install -y unattended-upgrades
    dpkg-reconfigure -fnoninteractive unattended-upgrades || true
  elif command -v dnf >/dev/null 2>&1; then
    dnf install -y dnf-automatic
    systemctl enable --now dnf-automatic.timer
  fi
fi

# 5) Fail2ban
if [ "$INSTALL_FAIL2BAN" = "1" ]; then
  if command -v apt-get >/dev/null 2>&1; then
    DEBIAN_FRONTEND=noninteractive apt-get install -y fail2ban
  elif command -v dnf >/dev/null 2>&1; then
    dnf install -y epel-release || true
    dnf install -y fail2ban
  fi
  systemctl enable --now fail2ban || true
fi

log "完成。回滚 SSH: rm $DROPIN && systemctl reload sshd"
`

  const aiSreCommand = `ai-sre node tune security \\
  --nodes ${target.nodes.join(',') || '<NODE_IPS>'} \\
  --os ${target.osType || '<auto>'} \\
  --on-conflict ${opts.onConflict} \\
  --disable-root-ssh=${opts.disableSshRoot} \\
  --ssh-port ${opts.changeSshPort ? opts.sshPort : 22} \\
  --firewall=${opts.enableFirewall} \\
  --disable-unneeded=${opts.disableUnneeded} \\
  --auto-update=${opts.enableAutoUpdate} \\
  --fail2ban=${opts.installFail2ban}`

  return {
    subtitle: `在 ${target.nodes.length || '<目标>'} 个节点应用 SSH/防火墙/Fail2ban 加固`,
    fullScript,
    aiSreCommand,
    batchOneLiner: formatBatchOneLiner(target, 'security.sh'),
    curlOneLiner: formatCurlOneLiner('https://<api-host>/ft-api/api/init-tools/scripts/security.sh', env),
  }
}

// =====================================================================
// 4) Disk Partition
// =====================================================================
export interface DiskOptions {
  enableSsdTrim: boolean
  tuneFilesystem: boolean
  setupSwap: boolean
  swapSize: string
  onConflict: OnConflict
}

export function genDiskScript(target: NodeSystemValue, opts: DiskOptions): ScriptBundle {
  const env = {
    ON_CONFLICT: opts.onConflict,
    ENABLE_TRIM: opts.enableSsdTrim ? '1' : '0',
    TUNE_FS: opts.tuneFilesystem ? '1' : '0',
    SETUP_SWAP: opts.setupSwap ? '1' : '0',
    SWAP_SIZE: opts.swapSize,
  }

  const fullScript = `#!/usr/bin/env bash
# ai-sre node tune disk
# Generated for: nodes=${sshTargets(target)} os=${target.osType || '<auto>'}
# 危险操作：会修改 fstab/swap，请确保已备份关键数据
set -euo pipefail

ON_CONFLICT="\${ON_CONFLICT:-${opts.onConflict}}"
ENABLE_TRIM="\${ENABLE_TRIM:-${opts.enableSsdTrim ? 1 : 0}}"
TUNE_FS="\${TUNE_FS:-${opts.tuneFilesystem ? 1 : 0}}"
SETUP_SWAP="\${SETUP_SWAP:-${opts.setupSwap ? 1 : 0}}"
SWAP_SIZE="\${SWAP_SIZE:-${opts.swapSize}}"   # 1G/2G/4G/8G/16G/auto

BACKUP_DIR=/var/backups/ai-sre/$(date +%Y%m%d-%H%M%S)
mkdir -p "$BACKUP_DIR"
[ -f /etc/fstab ] && cp /etc/fstab "$BACKUP_DIR/"

log()  { printf '[disk] %s\\n' "$*"; }
warn() { printf '[disk][warn] %s\\n' "$*" >&2; }

# 1) SSD TRIM
if [ "$ENABLE_TRIM" = "1" ]; then
  if systemctl is-enabled --quiet fstrim.timer 2>/dev/null && [ "$ON_CONFLICT" = "skip" ]; then
    log "fstrim.timer 已启用，跳过"
  else
    if systemctl list-unit-files | grep -q '^fstrim\\.timer'; then
      systemctl enable --now fstrim.timer
      log "fstrim.timer 已启用"
    else
      warn "未发现 fstrim.timer 单元，请确认 util-linux 已安装"
    fi
  fi
fi

# 2) 文件系统调优
if [ "$TUNE_FS" = "1" ]; then
  log "为根分区添加 noatime 挂载选项（如未启用）"
  if grep -E '\\s/\\s+(ext4|xfs)\\s' /etc/fstab | grep -qv noatime; then
    sed -i.bak -E 's|(\\s/\\s+(ext4|xfs)\\s+)([^[:space:]]+)|\\1\\3,noatime|' /etc/fstab
    log "已为 / 添加 noatime（重启或 mount -o remount,noatime / 后生效）"
  else
    log "/ 已含 noatime 或非 ext4/xfs，跳过"
  fi
fi

# 3) Swap 配置
if [ "$SETUP_SWAP" = "1" ]; then
  if swapon --show=NAME --noheadings | grep -q . && [ "$ON_CONFLICT" = "skip" ]; then
    log "已存在 swap，ON_CONFLICT=skip，跳过"
    swapon --show
  else
    case "$SWAP_SIZE" in
      auto) SZ=$(awk '/MemTotal/{printf "%dM", $2/512}' /proc/meminfo) ;;  # 内存 2 倍（KB→2*KB→以 M 为单位）
      *G)   SZ="$SWAP_SIZE" ;;
      *)    SZ="$SWAP_SIZE" ;;
    esac
    log "创建 /swapfile（$SZ）"
    swapoff -a || true
    rm -f /swapfile
    fallocate -l "$SZ" /swapfile || dd if=/dev/zero of=/swapfile bs=1M count=$(numfmt --from=iec "$SZ" | awk '{print int($1/1024/1024)}')
    chmod 600 /swapfile
    mkswap /swapfile
    swapon /swapfile
    grep -q '^/swapfile' /etc/fstab || echo '/swapfile none swap sw 0 0' >> /etc/fstab
    log "swap 已生效"
  fi
fi

# 4) 验证
log "=== 当前块设备 ==="
lsblk -o NAME,SIZE,TYPE,MOUNTPOINT,ROTA,DISC-GRAN | sed 's/^/  /'
log "=== 文件系统挂载 ==="
findmnt -t ext4,xfs -o TARGET,OPTIONS | sed 's/^/  /' || true
log "=== Swap ==="
swapon --show 2>/dev/null | sed 's/^/  /' || echo "  (无)"

log "完成。回滚 fstab: cp $BACKUP_DIR/fstab /etc/fstab"
`

  const aiSreCommand = `ai-sre node tune disk \\
  --nodes ${target.nodes.join(',') || '<NODE_IPS>'} \\
  --os ${target.osType || '<auto>'} \\
  --on-conflict ${opts.onConflict} \\
  --ssd-trim=${opts.enableSsdTrim} \\
  --tune-fs=${opts.tuneFilesystem} \\
  --setup-swap=${opts.setupSwap} \\
  --swap-size ${opts.swapSize}`

  return {
    subtitle: `在 ${target.nodes.length || '<目标>'} 个节点应用磁盘优化`,
    fullScript,
    aiSreCommand,
    batchOneLiner: formatBatchOneLiner(target, 'disk.sh'),
    curlOneLiner: formatCurlOneLiner('https://<api-host>/ft-api/api/init-tools/scripts/disk.sh', env),
  }
}

// 让 OsType 类型推断在外部可用
export type { OsType }
