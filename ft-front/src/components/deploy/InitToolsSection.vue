<template>
  <div class="init-tools-section">
    <div class="init-tools-subsection">
      <h4 class="init-tools-subsection__title">环境准备</h4>
      <div class="tool-grid tool-grid--deploy">

      <!-- ════════ 1. 时间同步 ════════ -->
      <el-card class="tool-card tool-card--recommended" shadow="hover">
        <template #header>
          <div class="tool-card-header">
            <div class="tool-card-icon" style="background: linear-gradient(135deg, #FCD34D, #F59E0B);">
              <el-icon :size="18"><Timer /></el-icon>
            </div>
            <div class="tool-card-title-wrap">
              <h3 class="tool-card-title">
                时间同步
                <el-tag type="warning" size="small" effect="dark">推荐</el-tag>
              </h3>
              <p class="tool-card-desc">NTP 校时 / 时区 / 节点间时差 &lt; 1s</p>
            </div>
          </div>
        </template>

        <div class="tool-card-body">
          <div class="tool-card-scroll">
            <el-form :model="timeSync.opts" label-width="80px" size="small" class="tool-form">

              <!-- NTP 模式 -->
              <el-form-item label="NTP 模式">
                <el-radio-group v-model="timeSync.opts.ntpMode" class="compact-radio">
                  <el-radio-button value="public">公用 NTP</el-radio-button>
                  <el-radio-button value="self-hosted">自建主节点</el-radio-button>
                </el-radio-group>
              </el-form-item>

              <!-- 自建主节点 IP -->
              <el-form-item v-if="timeSync.opts.ntpMode === 'self-hosted'" label="主节点 IP">
                <el-input
                  v-model="timeSync.opts.masterNodeIp"
                  placeholder="192.168.x.x（将安装 NTP 服务端）"
                  clearable
                />
              </el-form-item>

              <!-- 公用 NTP 服务器 -->
              <template v-if="timeSync.opts.ntpMode === 'public'">
                <el-form-item label="主源">
                  <el-input v-model="timeSync.opts.ntpServer" placeholder="ntp.aliyun.com" clearable />
                </el-form-item>
                <el-form-item label="备用源">
                  <el-input v-model="timeSync.opts.fallbackNtpServer" placeholder="ntp1.aliyun.com（可留空）" clearable />
                </el-form-item>
              </template>

              <!-- 客户端节点 -->
              <el-form-item label="客户端节点">
                <el-input
                  v-model="timeSync.opts.clientNodeIps"
                  type="textarea"
                  :rows="3"
                  placeholder="每行一个 IP（留空则仅在本机执行）&#10;192.168.1.10&#10;192.168.1.11"
                />
              </el-form-item>

              <!-- 工具 / 时区 / 间隔 / 冲突策略 -->
              <el-form-item label="NTP 工具">
                <el-radio-group v-model="timeSync.opts.preferredTool" class="compact-radio">
                  <el-radio-button value="chrony">chrony</el-radio-button>
                  <el-radio-button value="timesyncd">timesyncd</el-radio-button>
                </el-radio-group>
              </el-form-item>
              <el-form-item label="时区">
                <el-select v-model="timeSync.opts.timezone" style="width: 100%">
                  <el-option label="Asia/Shanghai (CST)" value="Asia/Shanghai" />
                  <el-option label="UTC" value="UTC" />
                  <el-option label="Europe/London" value="Europe/London" />
                  <el-option label="America/New_York" value="America/New_York" />
                </el-select>
              </el-form-item>
              <el-form-item label="已存在时">
                <el-radio-group v-model="timeSync.opts.onConflict" size="small" class="compact-radio">
                  <el-radio-button value="skip">跳过</el-radio-button>
                  <el-radio-button value="force">覆盖</el-radio-button>
                </el-radio-group>
              </el-form-item>
            </el-form>
          </div>

          <div class="tool-card-actions">
            <el-button @click="resetTimeSync"><el-icon><RefreshRight /></el-icon> 重置</el-button>
            <el-button type="primary" :icon="View" @click="openTimeSync">生成执行脚本</el-button>
          </div>
        </div>
      </el-card>

      <!-- ════════ 2. 系统参数优化 ════════ -->
      <el-card class="tool-card tool-card--recommended" shadow="hover">
        <template #header>
          <div class="tool-card-header">
            <div class="tool-card-icon" style="background: linear-gradient(135deg, #93C5FD, #2563EB);">
              <el-icon :size="18"><Cpu /></el-icon>
            </div>
            <div class="tool-card-title-wrap">
              <h3 class="tool-card-title">
                系统参数优化
                <el-tag type="warning" size="small" effect="dark">推荐</el-tag>
              </h3>
              <p class="tool-card-desc">sysctl + 内核模块 + ulimit + 关 swap</p>
            </div>
          </div>
        </template>

        <div class="tool-card-body">
          <div class="tool-card-scroll">
            <el-form :model="sysParam.opts" label-width="80px" size="small" class="tool-form">
              <!-- 目标节点 -->
              <el-form-item label="目标节点">
                <el-input
                  v-model="sysParam.opts.nodeIps"
                  type="textarea"
                  :rows="2"
                  placeholder="每行一个 IP（留空则仅在本机执行）"
                />
              </el-form-item>

              <!-- sysctl 参数 -->
              <el-collapse v-model="sysParam.collapse" class="tool-card-collapse">
                <el-collapse-item title="sysctl 参数（可编辑）" name="cfg">
                  <div class="param-list">
                    <div v-for="row in sysParam.rows" :key="row.key" class="param-row">
                      <div class="param-key">
                        <code class="param-code">{{ row.key }}</code>
                        <el-tag v-if="row.required" type="danger" size="small">K8s</el-tag>
                      </div>
                      <el-input v-model="row.value" size="small" class="param-value" />
                    </div>
                  </div>
                </el-collapse-item>
              </el-collapse>

              <el-form-item label="关 swap">
                <el-switch v-model="sysParam.opts.disableSwap" />
                <span class="form-hint">K8s 必关</span>
              </el-form-item>
              <el-form-item label="ulimit">
                <el-switch v-model="sysParam.opts.raiseUlimit" />
                <span class="form-hint">655350</span>
              </el-form-item>
              <el-form-item label="已存在时">
                <el-radio-group v-model="sysParam.opts.onConflict" size="small" class="compact-radio">
                  <el-radio-button value="skip">跳过</el-radio-button>
                  <el-radio-button value="force">覆盖</el-radio-button>
                </el-radio-group>
              </el-form-item>
            </el-form>
          </div>

          <div class="tool-card-actions">
            <el-button @click="resetSysParam"><el-icon><RefreshRight /></el-icon> 重置</el-button>
            <el-button type="primary" :icon="View" @click="openSysParam">生成执行脚本</el-button>
          </div>
        </div>
      </el-card>
      </div>
    </div>

    <div class="init-tools-subsection">
      <h4 class="init-tools-subsection__title">安全与存储</h4>
      <div class="tool-grid tool-grid--deploy">
      <!-- ════════ 3. 系统安全加固 ════════ -->
      <el-card class="tool-card" shadow="hover">
        <template #header>
          <div class="tool-card-header">
            <div class="tool-card-icon" style="background: linear-gradient(135deg, #FCA5A5, #DC2626);">
              <el-icon :size="18"><Lock /></el-icon>
            </div>
            <div class="tool-card-title-wrap">
              <h3 class="tool-card-title">系统安全加固</h3>
              <p class="tool-card-desc">SSH / 防火墙 / Fail2ban，自动备份回滚</p>
            </div>
          </div>
        </template>

        <div class="tool-card-body">
          <div class="tool-card-scroll">
            <el-form :model="security.opts" label-width="80px" size="small" class="tool-form">
              <el-form-item label="目标节点">
                <el-input
                  v-model="security.opts.nodeIps"
                  type="textarea"
                  :rows="2"
                  placeholder="每行一个 IP（留空则仅在本机执行）"
                />
              </el-form-item>
              <el-form-item label="禁 root SSH">
                <el-switch v-model="security.opts.disableSshRoot" />
              </el-form-item>
              <el-form-item label="SSH 端口">
                <el-switch v-model="security.opts.changeSshPort" />
                <el-input-number
                  v-if="security.opts.changeSshPort"
                  v-model="security.opts.sshPort"
                  :min="1024" :max="65535" :step="1" :precision="0"
                  controls-position="right"
                  style="width: 104px; margin-left: 8px;"
                />
              </el-form-item>
              <el-form-item label="防火墙">
                <el-switch v-model="security.opts.enableFirewall" />
              </el-form-item>
              <el-form-item label="禁服务">
                <el-switch v-model="security.opts.disableUnneeded" />
              </el-form-item>
              <el-form-item label="自动更新">
                <el-switch v-model="security.opts.enableAutoUpdate" />
              </el-form-item>
              <el-form-item label="Fail2ban">
                <el-switch v-model="security.opts.installFail2ban" />
              </el-form-item>
              <el-form-item label="已存在时">
                <el-radio-group v-model="security.opts.onConflict" size="small" class="compact-radio">
                  <el-radio-button value="skip">跳过</el-radio-button>
                  <el-radio-button value="force">覆盖</el-radio-button>
                </el-radio-group>
              </el-form-item>
            </el-form>
          </div>

          <div class="tool-card-actions">
            <el-button @click="resetSecurity"><el-icon><RefreshRight /></el-icon> 重置</el-button>
            <el-button
              type="primary"
              :icon="View"
              :disabled="!anySecurityOption"
              @click="openSecurity"
            >
              生成执行脚本
            </el-button>
          </div>
        </div>
      </el-card>

      <!-- ════════ 4. 磁盘分区优化 ════════ -->
      <el-card class="tool-card" shadow="hover">
        <template #header>
          <div class="tool-card-header">
            <div class="tool-card-icon" style="background: linear-gradient(135deg, #C4B5FD, #7C3AED);">
              <el-icon :size="18"><Coin /></el-icon>
            </div>
            <div class="tool-card-title-wrap">
              <h3 class="tool-card-title">磁盘分区优化</h3>
              <p class="tool-card-desc">SSD TRIM / 挂载优化 / Swap，自动备份 fstab</p>
            </div>
          </div>
        </template>

        <div class="tool-card-body">
          <div class="tool-card-scroll">
            <el-form :model="disk.opts" label-width="80px" size="small" class="tool-form">
              <el-form-item label="目标节点">
                <el-input
                  v-model="disk.opts.nodeIps"
                  type="textarea"
                  :rows="2"
                  placeholder="每行一个 IP（留空则仅在本机执行）"
                />
              </el-form-item>
              <el-form-item label="SSD TRIM">
                <el-switch v-model="disk.opts.enableSsdTrim" />
                <span class="form-hint">fstrim</span>
              </el-form-item>
              <el-form-item label="文件系统">
                <el-switch v-model="disk.opts.tuneFilesystem" />
                <span class="form-hint">noatime</span>
              </el-form-item>
              <el-form-item label="Swap">
                <el-switch v-model="disk.opts.setupSwap" />
                <el-select
                  v-if="disk.opts.setupSwap"
                  v-model="disk.opts.swapSize"
                  size="small"
                  style="width: 110px; margin-left: 8px;"
                >
                  <el-option label="auto" value="auto" />
                  <el-option label="1 GB" value="1G" />
                  <el-option label="2 GB" value="2G" />
                  <el-option label="4 GB" value="4G" />
                  <el-option label="8 GB" value="8G" />
                  <el-option label="16 GB" value="16G" />
                </el-select>
              </el-form-item>
              <el-form-item label="已存在时">
                <el-radio-group v-model="disk.opts.onConflict" size="small" class="compact-radio">
                  <el-radio-button value="skip">跳过</el-radio-button>
                  <el-radio-button value="force">覆盖</el-radio-button>
                </el-radio-group>
              </el-form-item>
            </el-form>
          </div>

          <div class="tool-card-actions">
            <el-button @click="resetDisk"><el-icon><RefreshRight /></el-icon> 重置</el-button>
            <el-button
              type="primary"
              :icon="View"
              :disabled="!anyDiskOption"
              @click="openDisk"
            >
              生成执行脚本
            </el-button>
          </div>
        </div>
      </el-card>
      </div>
    </div>

    <ScriptPreviewDialog
      v-model="dialogVisible"
      :title="dialogTitle"
      :bundle="dialogBundle"
      :default-filename="dialogFilename"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { Cpu, Timer, Lock, Coin, RefreshRight, View } from '@element-plus/icons-vue'
import ScriptPreviewDialog from '../init-tools/ScriptPreviewDialog.vue'
import {
  genTimeSyncScript,
  genSysParamScript,
  genSecurityScript,
  genDiskScript,
  type ScriptBundle,
  type SysParamRow,
} from '../../views/init-tools/scripts'

// ──── 弹窗 ────────────────────────────────────────────────────────────────────
const dialogVisible = ref(false)
const dialogTitle = ref('')
const dialogBundle = ref<ScriptBundle | null>(null)
const dialogFilename = ref('init.sh')

const showDialog = (title: string, bundle: ScriptBundle, filename: string) => {
  dialogTitle.value = title
  dialogBundle.value = bundle
  dialogFilename.value = filename
  dialogVisible.value = true
}

// ──── 1. 时间同步 ──────────────────────────────────────────────────────────────
const timeSyncDefaults = () => ({
  opts: {
    ntpMode: 'public' as 'public' | 'self-hosted',
    masterNodeIp: '',
    clientNodeIps: '',
    ntpServer: 'ntp.aliyun.com',
    fallbackNtpServer: 'ntp1.aliyun.com',
    timezone: 'Asia/Shanghai',
    syncIntervalMin: 15,
    preferredTool: 'chrony' as 'chrony' | 'timesyncd',
    onConflict: 'skip' as 'skip' | 'force',
  },
})
const timeSync = reactive(timeSyncDefaults())

const openTimeSync = () => {
  showDialog('时间同步 — Ansible 执行脚本', genTimeSyncScript(timeSync.opts), 'time-sync.sh')
}
const resetTimeSync = () => {
  Object.assign(timeSync.opts, timeSyncDefaults().opts)
  ElMessage.info('时间同步配置已重置')
}

// ──── 2. 系统参数优化 ──────────────────────────────────────────────────────────
const defaultSysParamRows = (): SysParamRow[] => [
  { key: 'net.ipv4.ip_forward', value: '1', description: '开启 IP 转发；K8s 必置 1', required: true },
  { key: 'net.bridge.bridge-nf-call-iptables', value: '1', description: 'iptables 处理桥接流量', required: true },
  { key: 'net.bridge.bridge-nf-call-ip6tables', value: '1', description: 'IPv6 桥接 netfilter', required: true },
  { key: 'vm.swappiness', value: '10', description: '降低交换倾向', required: false },
  { key: 'net.core.somaxconn', value: '65535', description: 'TCP backlog', required: false },
  { key: 'net.ipv4.tcp_max_tw_buckets', value: '6000', description: 'TIME_WAIT 上限', required: false },
  { key: 'fs.file-max', value: '655350', description: '系统文件句柄上限', required: false },
]

const sysParamDefaults = () => ({
  rows: defaultSysParamRows(),
  collapse: ['cfg'] as string[],
  opts: {
    nodeIps: '',
    onConflict: 'skip' as 'skip' | 'force',
    disableSwap: true,
    raiseUlimit: true,
  },
})
const sysParam = reactive(sysParamDefaults())

const openSysParam = () => {
  const missing = sysParam.rows.filter(r => r.required && !r.value.toString().trim())
  if (missing.length > 0) {
    ElMessage.error(`缺少必填参数: ${missing.map(r => r.key).join(', ')}`)
    return
  }
  showDialog('系统参数优化 — Ansible 执行脚本', genSysParamScript({
    nodeIps: sysParam.opts.nodeIps,
    rows: sysParam.rows,
    onConflict: sysParam.opts.onConflict,
    disableSwap: sysParam.opts.disableSwap,
    raiseUlimit: sysParam.opts.raiseUlimit,
  }), 'sys-param.sh')
}
const resetSysParam = () => {
  sysParam.rows = defaultSysParamRows()
  Object.assign(sysParam.opts, sysParamDefaults().opts)
  ElMessage.info('系统参数已重置')
}

// ──── 3. 安全加固 ──────────────────────────────────────────────────────────────
const securityDefaults = () => ({
  opts: {
    nodeIps: '',
    disableSshRoot: true,
    changeSshPort: false,
    sshPort: 2222,
    enableFirewall: true,
    disableUnneeded: false,
    enableAutoUpdate: false,
    installFail2ban: true,
    onConflict: 'skip' as 'skip' | 'force',
  },
})
const security = reactive(securityDefaults())

const anySecurityOption = computed(() => {
  const o = security.opts
  return o.disableSshRoot || o.changeSshPort || o.enableFirewall || o.disableUnneeded || o.enableAutoUpdate || o.installFail2ban
})

const openSecurity = () => {
  showDialog('系统安全加固 — Ansible 执行脚本', genSecurityScript(security.opts), 'security.sh')
}
const resetSecurity = () => {
  Object.assign(security.opts, securityDefaults().opts)
  ElMessage.info('安全加固配置已重置')
}

// ──── 4. 磁盘优化 ──────────────────────────────────────────────────────────────
const diskDefaults = () => ({
  opts: {
    nodeIps: '',
    enableSsdTrim: true,
    tuneFilesystem: true,
    setupSwap: false,
    swapSize: 'auto',
    onConflict: 'skip' as 'skip' | 'force',
  },
})
const disk = reactive(diskDefaults())

const anyDiskOption = computed(() => {
  const o = disk.opts
  return o.enableSsdTrim || o.tuneFilesystem || o.setupSwap
})

const openDisk = () => {
  showDialog('磁盘分区优化 — Ansible 执行脚本', genDiskScript(disk.opts), 'disk.sh')
}
const resetDisk = () => {
  Object.assign(disk.opts, diskDefaults().opts)
  ElMessage.info('磁盘优化配置已重置')
}
</script>

<style scoped>
.init-tools-section {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.init-tools-subsection__title {
  margin: 0 0 10px;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.tool-grid--deploy {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}
@media (max-width: 960px) {
  .tool-grid--deploy {
    grid-template-columns: 1fr;
  }
}

.tool-card {
  min-width: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  border-radius: 10px;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
  overflow: hidden;
}

.tool-card:hover {
  transform: translateY(-2px);
}

.tool-card--recommended {
  border-color: #F59E0B;
}

.tool-card :deep(.el-card__header) {
  flex: 0 0 auto;
  padding: 12px 14px;
}

.tool-card :deep(.el-card__body) {
  flex: 1 1 auto;
  min-height: 0;
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.tool-card-header {
  display: flex;
  gap: 10px;
  align-items: center;
}

.tool-card-icon {
  flex: 0 0 32px;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #ffffff;
  box-shadow: 0 4px 10px rgba(15, 23, 42, 0.12);
}

.tool-card-title-wrap {
  flex: 1;
  min-width: 0;
}

.tool-card-title {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0 0 2px;
  font-size: 14.5px;
  color: #0f172a;
  min-width: 0;
}

.tool-card-desc {
  margin: 0;
  font-size: 12px;
  color: #64748b;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tool-card-body {
  flex: 1 1 auto;
  min-height: 0;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow: hidden;
}

.tool-card-scroll {
  flex: 1 1 auto;
  min-height: 0;
  min-width: 0;
  overflow-y: auto;
  overflow-x: hidden;
  padding-right: 4px;
  scrollbar-gutter: stable;
}

.tool-card-scroll::-webkit-scrollbar {
  width: 6px;
  height: 0;
}

.tool-card-scroll::-webkit-scrollbar-thumb {
  background: #cbd5e1;
  border-radius: 3px;
}

.tool-card-scroll::-webkit-scrollbar-thumb:hover {
  background: #94a3b8;
}

.tool-card-scroll::-webkit-scrollbar-track {
  background: transparent;
}

.tool-form :deep(.el-form-item) {
  margin-bottom: 8px;
  min-width: 0;
}

.tool-form :deep(.el-form-item__content) {
  min-width: 0;
  flex-wrap: wrap;
  row-gap: 6px;
}

.tool-form :deep(.el-form-item__label) {
  font-size: 12.5px;
  color: #475569;
}

.tool-form :deep(.el-textarea__inner) {
  font-size: 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  resize: vertical;
}

.tool-form .form-hint {
  margin-left: 6px;
  color: #94a3b8;
  font-size: 12px;
}

.tool-card-collapse {
  min-width: 0;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  background: #ffffff;
  padding: 0 10px;
  margin-bottom: 8px;
}

.tool-card-collapse :deep(.el-collapse-item__header) {
  font-size: 12.5px;
  color: #1f2937;
  border-bottom: none;
  height: 34px;
}

.tool-card-collapse :deep(.el-collapse-item__wrap) {
  border-bottom: none;
}

.tool-card-collapse :deep(.el-collapse-item__content) {
  padding-bottom: 10px;
}

.param-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.param-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 88px;
  gap: 8px;
  align-items: center;
  min-width: 0;
}

.param-key {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
}

.param-code {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  font-size: 12px;
  color: #0f172a;
}

.param-value {
  width: 88px;
  min-width: 0;
}

.tool-card-actions {
  flex: 0 0 auto;
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  border-top: 1px dashed #e2e8f0;
  padding-top: 10px;
  margin-top: 4px;
}

.compact-radio {
  max-width: 100%;
  overflow: hidden;
}

.compact-radio :deep(.el-radio-button__inner) {
  padding: 7px 9px;
}

.tool-form :deep(.el-input),
.tool-form :deep(.el-select),
.tool-form :deep(.el-textarea),
.tool-form :deep(.el-input-number) {
  max-width: 100%;
}

.tool-card-actions :deep(.el-button) {
  white-space: nowrap;
}
</style>
