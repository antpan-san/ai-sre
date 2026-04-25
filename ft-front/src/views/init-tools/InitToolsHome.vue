<template>
  <!--
    初始化工具（单页）
    - 所有优化项（系统参数 / 时间同步 / 安全加固 / 磁盘分区）以卡片形式集中在本页
    - 每张卡片自包含：目标节点 + 系统类型 + 详细配置 + 一键应用
    - 不再有子菜单或子页面
    - K8s 部署页跳转过来时（?from=k8s-deploy）顶部显示返回部署的引导
  -->
  <div class="init-tools-home page-shell page-shell--wizard">
    <header class="page-header">
      <div class="page-header-inner">
        <span class="page-kicker">Initialization</span>
        <h2 class="page-title">节点初始化与优化工具</h2>
        <p class="page-desc">
          一站式完成节点系统优化：校准时钟 → 调优内核参数 → 关闭风险面 → 规划存储。所有项目集中在本页，不再跳子页。
        </p>
      </div>
    </header>

    <!-- 来自 K8s 部署页的回跳提示 -->
    <el-alert
      v-if="fromK8sDeploy"
      type="info"
      :closable="false"
      class="from-k8s-banner"
      show-icon
    >
      <template #title>
        <div class="from-k8s-banner-inner">
          <span>
            正在为 Kubernetes 集群
            <strong v-if="k8sCluster">「{{ k8sCluster }}」</strong>
            做部署前的环境优化。完成所需项后可以直接返回部署向导继续下一步。
          </span>
          <el-button type="primary" size="small" @click="backToK8sDeploy">
            返回 K8s 部署
            <el-icon class="el-icon--right"><ArrowRight /></el-icon>
          </el-button>
        </div>
      </template>
    </el-alert>

    <!-- ==================== 优化项卡片网格 ==================== -->
    <div class="tool-grid">
      <!-- 1. 时间同步 -->
      <el-card class="tool-card tool-card--recommended" shadow="hover">
        <template #header>
          <div class="tool-card-header">
            <div class="tool-card-icon" style="background: linear-gradient(135deg, #FCD34D, #F59E0B);">
              <el-icon :size="20"><Timer /></el-icon>
            </div>
            <div class="tool-card-title-wrap">
              <h3 class="tool-card-title">
                时间同步
                <el-tag type="warning" size="small" effect="dark">推荐</el-tag>
              </h3>
              <p class="tool-card-desc">
                部署 K8s / etcd 前必做：保证节点 NTP 已同步，避免时钟跳变引起 kubelet sandbox 误判。
              </p>
            </div>
          </div>
        </template>

        <div class="tool-card-body">
          <ul class="tool-card-bullets">
            <li><el-icon class="bullet-icon" color="#10B981"><Check /></el-icon><span>chrony / systemd-timesyncd 自动安装与启用</span></li>
            <li><el-icon class="bullet-icon" color="#10B981"><Check /></el-icon><span>统一时区与 NTP 源（支持自建主时间源）</span></li>
            <li><el-icon class="bullet-icon" color="#10B981"><Check /></el-icon><span>节点间时差校验 &lt; 1s 才算通过</span></li>
          </ul>

          <NodeSystemSelector v-model="timeSyncTarget" />

          <el-collapse v-model="timeSyncCollapse" class="tool-card-collapse">
            <el-collapse-item title="详细配置（NTP 主源 / 同步间隔 / 时区）" name="cfg">
              <el-form :model="timeSyncOptions" label-width="100px" size="small" class="tool-form">
                <el-form-item label="NTP 主源">
                  <el-input v-model="timeSyncOptions.ntpServer" placeholder="如：ntp.aliyun.com 或 192.168.56.10" clearable />
                </el-form-item>
                <el-form-item label="同步间隔">
                  <el-input-number v-model="timeSyncOptions.syncInterval" :min="1" :max="60" :step="1" :precision="0" style="width: 140px;" />
                  <span class="form-unit">分钟</span>
                </el-form-item>
                <el-form-item label="同步时区">
                  <el-select v-model="timeSyncOptions.timezone" style="width: 220px;">
                    <el-option label="Asia/Shanghai (CST)" value="Asia/Shanghai" />
                    <el-option label="UTC" value="UTC" />
                    <el-option label="Europe/London (GMT)" value="Europe/London" />
                    <el-option label="America/New_York (EST)" value="America/New_York" />
                  </el-select>
                </el-form-item>
                <el-form-item label="启用 NTP 服务">
                  <el-switch v-model="timeSyncOptions.enableNtp" />
                </el-form-item>
              </el-form>
            </el-collapse-item>
          </el-collapse>

          <div class="tool-card-actions">
            <el-button :disabled="!isReady(timeSyncTarget)" @click="resetTimeSync">
              <el-icon><RefreshRight /></el-icon>
              重置
            </el-button>
            <el-button type="primary" :icon="Promotion" :loading="timeSyncApplying" :disabled="!isReady(timeSyncTarget)" @click="applyTimeSync">
              同步时间
            </el-button>
          </div>
        </div>
      </el-card>

      <!-- 2. 系统参数优化 -->
      <el-card class="tool-card tool-card--recommended" shadow="hover">
        <template #header>
          <div class="tool-card-header">
            <div class="tool-card-icon" style="background: linear-gradient(135deg, #93C5FD, #2563EB);">
              <el-icon :size="20"><Cpu /></el-icon>
            </div>
            <div class="tool-card-title-wrap">
              <h3 class="tool-card-title">
                系统参数优化
                <el-tag type="warning" size="small" effect="dark">推荐</el-tag>
              </h3>
              <p class="tool-card-desc">
                sysctl + ulimit + 关闭 swap，让 kubelet / etcd / 中间件运行在合理的内核参数下。
              </p>
            </div>
          </div>
        </template>

        <div class="tool-card-body">
          <ul class="tool-card-bullets">
            <li><el-icon class="bullet-icon" color="#10B981"><Check /></el-icon><span>关键 sysctl：ip_forward、bridge-nf-call-iptables</span></li>
            <li><el-icon class="bullet-icon" color="#10B981"><Check /></el-icon><span>加载内核模块：br_netfilter / overlay</span></li>
            <li><el-icon class="bullet-icon" color="#10B981"><Check /></el-icon><span>提升 fs.file-max、somaxconn 等连接数上限</span></li>
          </ul>

          <NodeSystemSelector v-model="sysParamTarget" />

          <el-collapse v-model="sysParamCollapse" class="tool-card-collapse">
            <el-collapse-item title="详细配置（sysctl 参数表）" name="cfg">
              <el-table :data="sysParamRows" size="small" class="tool-table" border>
                <el-table-column prop="key" label="参数名" min-width="180">
                  <template #default="scope">
                    <div class="param-name">
                      <code>{{ scope.row.key }}</code>
                      <el-tag v-if="scope.row.required" type="danger" size="small">必填</el-tag>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column prop="value" label="值" width="160">
                  <template #default="scope">
                    <el-input v-model="scope.row.value" size="small" placeholder="参数值" />
                  </template>
                </el-table-column>
                <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
              </el-table>
            </el-collapse-item>
          </el-collapse>

          <div class="tool-card-actions">
            <el-button :disabled="!isReady(sysParamTarget)" @click="resetSysParam">
              <el-icon><RefreshRight /></el-icon>
              重置
            </el-button>
            <el-button type="primary" :icon="CircleCheck" :loading="sysParamApplying" :disabled="!isReady(sysParamTarget)" @click="applySysParam">
              应用参数
            </el-button>
          </div>
        </div>
      </el-card>

      <!-- 3. 系统安全加固 -->
      <el-card class="tool-card" shadow="hover">
        <template #header>
          <div class="tool-card-header">
            <div class="tool-card-icon" style="background: linear-gradient(135deg, #FCA5A5, #DC2626);">
              <el-icon :size="20"><Lock /></el-icon>
            </div>
            <div class="tool-card-title-wrap">
              <h3 class="tool-card-title">系统安全加固</h3>
              <p class="tool-card-desc">
                禁 root SSH 直登、Fail2ban、最小化攻击面；生产环境强烈推荐执行一次。
              </p>
            </div>
          </div>
        </template>

        <div class="tool-card-body">
          <ul class="tool-card-bullets">
            <li><el-icon class="bullet-icon" color="#10B981"><Check /></el-icon><span>禁用 SSH root 登录、修改默认端口</span></li>
            <li><el-icon class="bullet-icon" color="#10B981"><Check /></el-icon><span>启用防火墙、安装 Fail2ban</span></li>
            <li><el-icon class="bullet-icon" color="#10B981"><Check /></el-icon><span>关闭无用服务、按需自动更新补丁</span></li>
          </ul>

          <NodeSystemSelector v-model="securityTarget" />

          <el-collapse v-model="securityCollapse" class="tool-card-collapse">
            <el-collapse-item title="详细配置（安全策略复选）" name="cfg">
              <el-checkbox-group v-model="securityOptions" class="security-options">
                <div class="security-item">
                  <el-checkbox label="disable_ssh_root_login">禁用 SSH root 登录</el-checkbox>
                  <el-tooltip content="禁止使用 root 用户直接 SSH 登录系统" placement="top">
                    <el-icon class="help-icon"><QuestionFilled /></el-icon>
                  </el-tooltip>
                </div>
                <div class="security-item">
                  <el-checkbox label="change_ssh_port">修改 SSH 端口</el-checkbox>
                  <el-tooltip content="将 SSH 默认端口 22 修改为自定义端口，提高安全性" placement="top">
                    <el-icon class="help-icon"><QuestionFilled /></el-icon>
                  </el-tooltip>
                  <el-input-number
                    v-if="securityOptions.includes('change_ssh_port')"
                    v-model="securitySshPort"
                    :min="1024"
                    :max="65535"
                    :step="1"
                    :precision="0"
                    size="small"
                    style="width: 120px; margin-left: 10px;"
                    placeholder="端口号"
                  />
                </div>
                <div class="security-item">
                  <el-checkbox label="enable_firewall">启用防火墙</el-checkbox>
                  <el-tooltip content="启用系统防火墙，并配置基本规则" placement="top">
                    <el-icon class="help-icon"><QuestionFilled /></el-icon>
                  </el-tooltip>
                </div>
                <div class="security-item">
                  <el-checkbox label="disable_unnecessary_services">禁用不必要服务</el-checkbox>
                  <el-tooltip content="禁用系统中不需要的服务，减少安全风险" placement="top">
                    <el-icon class="help-icon"><QuestionFilled /></el-icon>
                  </el-tooltip>
                </div>
                <div class="security-item">
                  <el-checkbox label="update_system">系统更新</el-checkbox>
                  <el-tooltip content="更新系统到最新版本，修复安全漏洞" placement="top">
                    <el-icon class="help-icon"><QuestionFilled /></el-icon>
                  </el-tooltip>
                </div>
                <div class="security-item">
                  <el-checkbox label="setup_fail2ban">安装 Fail2ban</el-checkbox>
                  <el-tooltip content="安装并配置 Fail2ban，防止暴力破解" placement="top">
                    <el-icon class="help-icon"><QuestionFilled /></el-icon>
                  </el-tooltip>
                </div>
              </el-checkbox-group>
            </el-collapse-item>
          </el-collapse>

          <div class="tool-card-actions">
            <el-button :disabled="!isReady(securityTarget)" @click="resetSecurity">
              <el-icon><RefreshRight /></el-icon>
              重置
            </el-button>
            <el-button
              type="primary"
              :icon="CircleCheck"
              :loading="securityApplying"
              :disabled="!isReady(securityTarget) || securityOptions.length === 0"
              @click="applySecurity"
            >
              应用安全设置
            </el-button>
          </div>
        </div>
      </el-card>

      <!-- 4. 磁盘分区优化 -->
      <el-card class="tool-card" shadow="hover">
        <template #header>
          <div class="tool-card-header">
            <div class="tool-card-icon" style="background: linear-gradient(135deg, #C4B5FD, #7C3AED);">
              <el-icon :size="20"><Coin /></el-icon>
            </div>
            <div class="tool-card-title-wrap">
              <h3 class="tool-card-title">磁盘分区优化</h3>
              <p class="tool-card-desc">
                启用 SSD TRIM、调优挂载选项、规划 etcd 独立盘，缓解 fsync 抖动。
              </p>
            </div>
          </div>
        </template>

        <div class="tool-card-body">
          <ul class="tool-card-bullets">
            <li><el-icon class="bullet-icon" color="#10B981"><Check /></el-icon><span>启用 SSD TRIM 与 fstrim 定时任务</span></li>
            <li><el-icon class="bullet-icon" color="#10B981"><Check /></el-icon><span>为 ext4 / xfs 设置合理 noatime / discard</span></li>
            <li><el-icon class="bullet-icon" color="#10B981"><Check /></el-icon><span>可选自动配置 swap 大小与位置</span></li>
          </ul>

          <NodeSystemSelector v-model="diskTarget" />

          <el-alert
            type="warning"
            show-icon
            :closable="false"
            class="disk-alert"
            title="磁盘优化可能涉及挂载/重新分区操作，请确保已备份关键数据"
          />

          <el-collapse v-model="diskCollapse" class="tool-card-collapse">
            <el-collapse-item title="详细配置（优化项 / Swap）" name="cfg">
              <el-checkbox-group v-model="diskOptions">
                <div class="disk-option-item">
                  <el-checkbox label="enable_ssd_trim">启用 SSD TRIM 支持</el-checkbox>
                  <el-tooltip content="启用 SSD TRIM，提高 SSD 性能和寿命" placement="top">
                    <el-icon class="help-icon"><QuestionFilled /></el-icon>
                  </el-tooltip>
                </div>
                <div class="disk-option-item">
                  <el-checkbox label="tune_filesystem">优化文件系统参数</el-checkbox>
                  <el-tooltip content="优化 EXT4/XFS 等文件系统挂载参数（noatime/discard 等）" placement="top">
                    <el-icon class="help-icon"><QuestionFilled /></el-icon>
                  </el-tooltip>
                </div>
                <div class="disk-option-item">
                  <el-checkbox label="setup_swap">配置 Swap</el-checkbox>
                  <el-tooltip content="配置系统 Swap 分区大小（K8s 节点建议关闭，仅普通 Linux 主机使用）" placement="top">
                    <el-icon class="help-icon"><QuestionFilled /></el-icon>
                  </el-tooltip>
                  <el-select
                    v-if="diskOptions.includes('setup_swap')"
                    v-model="diskSwapSize"
                    size="small"
                    style="width: 150px; margin-left: 10px;"
                  >
                    <el-option label="1 GB" value="1G" />
                    <el-option label="2 GB" value="2G" />
                    <el-option label="4 GB" value="4G" />
                    <el-option label="8 GB" value="8G" />
                    <el-option label="16 GB" value="16G" />
                    <el-option label="自动（内存 2 倍）" value="auto" />
                  </el-select>
                </div>
              </el-checkbox-group>
            </el-collapse-item>
          </el-collapse>

          <div class="tool-card-actions">
            <el-button :disabled="!isReady(diskTarget)" @click="resetDisk">
              <el-icon><RefreshRight /></el-icon>
              重置
            </el-button>
            <el-button
              type="primary"
              :icon="Calendar"
              :loading="diskApplying"
              :disabled="!isReady(diskTarget) || diskOptions.length === 0"
              @click="applyDisk"
            >
              应用磁盘优化
            </el-button>
          </div>
        </div>
      </el-card>
    </div>

    <!-- 底部说明 -->
    <el-card class="footer-card" shadow="never">
      <template #header>
        <div class="footer-card-header">
          <el-icon><InfoFilled /></el-icon>
          <span>关于这些优化项</span>
        </div>
      </template>
      <ul class="footer-bullets">
        <li>
          <strong>系统参数优化</strong>：调整 vm.swappiness、somaxconn、tcp_max_tw_buckets、file-max 等内核参数，避免 etcd / kubelet 在高负载下抖动。
        </li>
        <li>
          <strong>时间同步</strong>：在每个目标节点上启用 chrony 或 systemd-timesyncd，节点间时差控制在 1s 内，避免 kubelet 误判 sandbox 过期、calico-node / coredns 反复 Killing。
        </li>
        <li>
          <strong>系统安全加固</strong>：禁 SSH root 直登、改默认端口、启防火墙、Fail2ban，给生产环境多一层防护（lab 环境可酌情跳过）。
        </li>
        <li>
          <strong>磁盘分区优化</strong>：启用 SSD TRIM、调优文件系统挂载选项，规划 etcd 独立盘，缓解 fsync p99 高带来的 apply request took too long。
        </li>
      </ul>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Cpu,
  Timer,
  Lock,
  Coin,
  ArrowRight,
  Check,
  InfoFilled,
  CircleCheck,
  Promotion,
  RefreshRight,
  QuestionFilled,
  Calendar,
} from '@element-plus/icons-vue'
import NodeSystemSelector, {
  type NodeSystemValue,
} from '../../components/init-tools/NodeSystemSelector.vue'

const router = useRouter()
const route = useRoute()

const fromK8sDeploy = computed(() => route.query.from === 'k8s-deploy')
const k8sCluster = computed(() => (route.query.cluster as string) || '')

const isReady = (t: NodeSystemValue): boolean => t.nodes.length > 0 && !!t.osType

// ============== 1. 时间同步 ==============
const timeSyncTarget = ref<NodeSystemValue>({ nodes: [], osType: '' })
const timeSyncCollapse = ref<string[]>(['cfg'])
const timeSyncApplying = ref(false)
const timeSyncOptions = reactive({
  ntpServer: 'ntp.aliyun.com',
  syncInterval: 15,
  timezone: 'Asia/Shanghai',
  enableNtp: true,
})

const applyTimeSync = () => {
  if (!isReady(timeSyncTarget.value)) {
    ElMessage.warning('请先选择目标节点与系统类型')
    return
  }
  timeSyncApplying.value = true
  setTimeout(() => {
    ElMessage.success(
      `时间同步任务已下发到 ${timeSyncTarget.value.nodes.length} 个节点（${timeSyncTarget.value.osType}），主源：${timeSyncOptions.ntpServer || '系统默认'}`
    )
    timeSyncApplying.value = false
  }, 1200)
}

const resetTimeSync = () => {
  timeSyncOptions.ntpServer = 'ntp.aliyun.com'
  timeSyncOptions.syncInterval = 15
  timeSyncOptions.timezone = 'Asia/Shanghai'
  timeSyncOptions.enableNtp = true
  ElMessage.info('时间同步配置已重置')
}

// ============== 2. 系统参数优化 ==============
const sysParamTarget = ref<NodeSystemValue>({ nodes: [], osType: '' })
const sysParamCollapse = ref<string[]>(['cfg'])
const sysParamApplying = ref(false)

interface SysParamRow {
  key: string
  value: string
  description: string
  required: boolean
}

const defaultSysParamRows: SysParamRow[] = [
  { key: 'net.ipv4.ip_forward', value: '1', description: '开启 IP 转发；K8s 节点必置 1，否则跨节点 Pod 流量被丢弃', required: true },
  { key: 'net.bridge.bridge-nf-call-iptables', value: '1', description: '让 iptables 规则对桥接流量生效；K8s 必置 1', required: true },
  { key: 'net.bridge.bridge-nf-call-ip6tables', value: '1', description: 'IPv6 桥接 netfilter；K8s 建议置 1', required: true },
  { key: 'vm.swappiness', value: '10', description: '降低交换倾向，减少 IO 抖动', required: false },
  { key: 'net.core.somaxconn', value: '65535', description: 'TCP backlog 上限，提升连接吞吐', required: false },
  { key: 'net.ipv4.tcp_max_tw_buckets', value: '6000', description: 'TIME_WAIT 上限，避免端口耗尽', required: false },
  { key: 'fs.file-max', value: '655350', description: '系统全局最大文件句柄数', required: false },
]
const sysParamRows = ref<SysParamRow[]>(defaultSysParamRows.map(r => ({ ...r })))

const applySysParam = () => {
  if (!isReady(sysParamTarget.value)) {
    ElMessage.warning('请先选择目标节点与系统类型')
    return
  }
  const missing = sysParamRows.value.filter(p => p.required && !p.value.trim())
  if (missing.length > 0) {
    ElMessage.error(`缺少必填参数：${missing.map(p => p.key).join(', ')}`)
    return
  }
  sysParamApplying.value = true
  setTimeout(() => {
    ElMessage.success(
      `系统参数已下发到 ${sysParamTarget.value.nodes.length} 个节点（${sysParamTarget.value.osType}）`
    )
    sysParamApplying.value = false
  }, 1200)
}

const resetSysParam = () => {
  sysParamRows.value = defaultSysParamRows.map(r => ({ ...r }))
  ElMessage.info('系统参数已重置为推荐默认值')
}

// ============== 3. 系统安全加固 ==============
const securityTarget = ref<NodeSystemValue>({ nodes: [], osType: '' })
const securityCollapse = ref<string[]>(['cfg'])
const securityApplying = ref(false)
const securityOptions = ref<string[]>([])
const securitySshPort = ref(2222)

const applySecurity = () => {
  if (!isReady(securityTarget.value)) {
    ElMessage.warning('请先选择目标节点与系统类型')
    return
  }
  if (securityOptions.value.length === 0) {
    ElMessage.warning('请至少勾选一项安全策略')
    return
  }
  ElMessageBox.confirm(
    `将向 ${securityTarget.value.nodes.length} 个节点（${securityTarget.value.osType}）应用 ${securityOptions.value.length} 项安全策略，可能影响 SSH/防火墙等系统功能，是否继续？`,
    '警告',
    { type: 'warning' }
  ).then(() => {
    securityApplying.value = true
    setTimeout(() => {
      ElMessage.success('系统安全加固任务已下发')
      securityApplying.value = false
    }, 1500)
  }).catch(() => {})
}

const resetSecurity = () => {
  securityOptions.value = []
  securitySshPort.value = 2222
  ElMessage.info('安全加固选项已重置')
}

// ============== 4. 磁盘分区优化 ==============
const diskTarget = ref<NodeSystemValue>({ nodes: [], osType: '' })
const diskCollapse = ref<string[]>(['cfg'])
const diskApplying = ref(false)
const diskOptions = ref<string[]>([])
const diskSwapSize = ref('auto')

const applyDisk = () => {
  if (!isReady(diskTarget.value)) {
    ElMessage.warning('请先选择目标节点与系统类型')
    return
  }
  if (diskOptions.value.length === 0) {
    ElMessage.warning('请至少勾选一项磁盘优化')
    return
  }
  ElMessageBox.confirm(
    `将向 ${diskTarget.value.nodes.length} 个节点（${diskTarget.value.osType}）执行 ${diskOptions.value.length} 项磁盘优化，可能涉及挂载/分区操作，请确保已备份重要数据，是否继续？`,
    '危险操作',
    { type: 'error', confirmButtonText: '确认优化', cancelButtonText: '取消' }
  ).then(() => {
    diskApplying.value = true
    setTimeout(() => {
      ElMessage.success('磁盘优化任务已下发')
      diskApplying.value = false
    }, 1500)
  }).catch(() => {})
}

const resetDisk = () => {
  diskOptions.value = []
  diskSwapSize.value = 'auto'
  ElMessage.info('磁盘优化选项已重置')
}

// ============== 公共：返回 K8s 部署 ==============
const backToK8sDeploy = () => {
  router.push({ path: '/service/k8s-deploy' })
}

onMounted(() => {
  if (fromK8sDeploy.value) {
    ElMessage.info('已为你按推荐顺序排列优化项：建议先「时间同步」→ 再「系统参数优化」')
  }
})
</script>

<style scoped>
.init-tools-home {
  min-height: 100%;
}

.page-header {
  margin-bottom: 16px;
}

.page-header-inner {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.page-kicker {
  font-size: 12px;
  color: #2563EB;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  font-weight: 600;
}

.page-title {
  margin: 0;
  font-size: 22px;
  color: #0f172a;
}

.page-desc {
  margin: 0;
  color: #475569;
  font-size: 13px;
}

.from-k8s-banner {
  margin-bottom: 16px;
}

.from-k8s-banner-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.tool-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(440px, 1fr));
  gap: 16px;
  margin-bottom: 20px;
}

.tool-card {
  border-radius: 10px;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.tool-card:hover {
  transform: translateY(-2px);
}

.tool-card--recommended {
  border-color: #F59E0B;
}

.tool-card-header {
  display: flex;
  gap: 12px;
  align-items: flex-start;
}

.tool-card-icon {
  flex: 0 0 40px;
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #ffffff;
  box-shadow: 0 6px 14px rgba(15, 23, 42, 0.12);
}

.tool-card-title-wrap {
  flex: 1;
  min-width: 0;
}

.tool-card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 4px;
  font-size: 16px;
  color: #0f172a;
}

.tool-card-desc {
  margin: 0;
  font-size: 12.5px;
  color: #475569;
  line-height: 1.6;
}

.tool-card-body {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.tool-card-bullets {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.tool-card-bullets li {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #1f2937;
}

.bullet-icon {
  flex: 0 0 16px;
}

.tool-card-collapse {
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #ffffff;
  padding: 0 12px;
}

.tool-card-collapse :deep(.el-collapse-item__header) {
  font-size: 13px;
  color: #1f2937;
  border-bottom: none;
  height: 38px;
}

.tool-card-collapse :deep(.el-collapse-item__wrap) {
  border-bottom: none;
}

.tool-card-collapse :deep(.el-collapse-item__content) {
  padding-bottom: 12px;
}

.tool-form .form-unit {
  margin-left: 8px;
  color: #64748b;
}

.tool-table .param-name {
  display: flex;
  align-items: center;
  gap: 6px;
}

.tool-table .param-name code {
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  font-size: 12px;
  color: #0f172a;
}

.security-options {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.security-item,
.disk-option-item {
  display: flex;
  align-items: center;
  gap: 6px;
}

.help-icon {
  color: #94a3b8;
  cursor: help;
}

.disk-alert {
  margin-bottom: 4px;
}

.tool-card-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  border-top: 1px dashed #e2e8f0;
  padding-top: 12px;
}

.footer-card {
  border-radius: 10px;
  border-style: dashed;
  background: #fafbfc;
}

.footer-card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #2563EB;
  font-weight: 600;
}

.footer-bullets {
  list-style: disc;
  padding-left: 20px;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
  color: #334155;
  font-size: 13px;
  line-height: 1.7;
}
</style>
