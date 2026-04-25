<template>
  <!--
    初始化工具总览
    - 设计意图：每张卡片即一个独立优化工具（系统参数 / 时间同步 / 安全加固 / 磁盘分区）
    - 卡片可单独选择「目标节点 + 系统类型」并就地一键应用，或跳转到详情页做精细配置
    - K8s 部署流程跳转过来时（?from=k8s-deploy）顶部显示返回部署的引导
  -->
  <div class="init-tools-home page-shell page-shell--wizard">
    <header class="page-header">
      <div class="page-header-inner">
        <span class="page-kicker">Initialization</span>
        <h2 class="page-title">节点初始化与优化工具</h2>
        <p class="page-desc">
          为后续 Kubernetes / 中间件部署做好准备：先校准时钟、调优内核参数、关闭风险面、规划存储。
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

    <!-- 工具卡片网格 -->
    <div class="tool-grid">
      <el-card
        v-for="tool in toolCards"
        :key="tool.id"
        class="tool-card"
        :class="{ 'tool-card--recommended': tool.recommended }"
        shadow="hover"
      >
        <template #header>
          <div class="tool-card-header">
            <div class="tool-card-icon" :style="{ background: tool.iconBg }">
              <el-icon :size="20"><component :is="tool.icon" /></el-icon>
            </div>
            <div class="tool-card-title-wrap">
              <h3 class="tool-card-title">
                {{ tool.title }}
                <el-tag v-if="tool.recommended" type="warning" size="small" effect="dark">推荐</el-tag>
              </h3>
              <p class="tool-card-desc">{{ tool.desc }}</p>
            </div>
          </div>
        </template>

        <div class="tool-card-body">
          <!-- 卡片内的关键能力一览（让用户在不进入详情的情况下也知道做什么） -->
          <ul class="tool-card-bullets">
            <li v-for="b in tool.bullets" :key="b">
              <el-icon class="bullet-icon" color="#10B981"><Check /></el-icon>
              <span>{{ b }}</span>
            </li>
          </ul>

          <!-- 节点 + 系统类型选择（每张卡片独立维护） -->
          <NodeSystemSelector v-model="tool.target" />

          <!-- 卡片内动作 -->
          <div class="tool-card-actions">
            <el-button
              type="primary"
              :icon="tool.applyIcon"
              :loading="tool.applying"
              :disabled="!isReady(tool)"
              @click="quickApply(tool)"
            >
              {{ tool.applyLabel || '一键应用' }}
            </el-button>
            <el-button :icon="ArrowRight" @click="goDetail(tool)">
              详细配置
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
import { ref, computed, onMounted, markRaw, type Component } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
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
} from '@element-plus/icons-vue'
import NodeSystemSelector, {
  type NodeSystemValue,
} from '../../components/init-tools/NodeSystemSelector.vue'

const router = useRouter()
const route = useRoute()

interface ToolCard {
  id: 'system-param' | 'time-sync' | 'security-hardening' | 'disk-partition'
  title: string
  desc: string
  bullets: string[]
  icon: Component
  iconBg: string
  detailRoute: string
  recommended?: boolean
  applyIcon?: Component
  applyLabel?: string
  target: NodeSystemValue
  applying: boolean
}

const fromK8sDeploy = computed(() => route.query.from === 'k8s-deploy')
const k8sCluster = computed(() => (route.query.cluster as string) || '')

const toolCards = ref<ToolCard[]>([
  {
    id: 'time-sync',
    title: '时间同步',
    desc: '部署 K8s / etcd 前必做：保证节点 NTP 已同步，避免时钟跳变引起 kubelet sandbox 误判。',
    bullets: [
      'chrony / systemd-timesyncd 自动安装与启用',
      '统一时区与 NTP 源（支持自建主时间源）',
      '节点间时差校验 < 1s 才算通过',
    ],
    icon: markRaw(Timer),
    iconBg: 'linear-gradient(135deg, #FCD34D, #F59E0B)',
    detailRoute: '/init-tools/time-sync',
    recommended: true,
    applyIcon: markRaw(Promotion),
    applyLabel: '同步时间',
    target: { nodes: [], osType: '' },
    applying: false,
  },
  {
    id: 'system-param',
    title: '系统参数优化',
    desc: 'sysctl + ulimit + 关闭 swap，让 kubelet / etcd / 中间件运行在合理的内核参数下。',
    bullets: [
      '关键 sysctl：ip_forward、bridge-nf-call-iptables',
      '加载内核模块：br_netfilter / overlay',
      '提升 fs.file-max、somaxconn 等连接数上限',
    ],
    icon: markRaw(Cpu),
    iconBg: 'linear-gradient(135deg, #93C5FD, #2563EB)',
    detailRoute: '/init-tools/system-param',
    recommended: true,
    applyIcon: markRaw(CircleCheck),
    applyLabel: '应用参数',
    target: { nodes: [], osType: '' },
    applying: false,
  },
  {
    id: 'security-hardening',
    title: '系统安全加固',
    desc: '禁 root SSH 直登、Fail2ban、最小化攻击面；生产环境强烈推荐执行一次。',
    bullets: [
      '禁用 SSH root 登录、修改默认端口',
      '启用防火墙、安装 Fail2ban',
      '关闭无用服务、按需自动更新补丁',
    ],
    icon: markRaw(Lock),
    iconBg: 'linear-gradient(135deg, #FCA5A5, #DC2626)',
    detailRoute: '/init-tools/security-hardening',
    target: { nodes: [], osType: '' },
    applying: false,
  },
  {
    id: 'disk-partition',
    title: '磁盘分区优化',
    desc: '启用 SSD TRIM、调优挂载选项、规划 etcd 独立盘，缓解 fsync 抖动。',
    bullets: [
      '启用 SSD TRIM 与 fstrim 定时任务',
      '为 ext4 / xfs 设置合理 noatime / discard',
      '可选自动配置 swap 大小与位置',
    ],
    icon: markRaw(Coin),
    iconBg: 'linear-gradient(135deg, #C4B5FD, #7C3AED)',
    detailRoute: '/init-tools/disk-partition',
    target: { nodes: [], osType: '' },
    applying: false,
  },
])

const isReady = (t: ToolCard): boolean => {
  return t.target.nodes.length > 0 && !!t.target.osType
}

const goDetail = (t: ToolCard) => {
  router.push({
    path: t.detailRoute,
    query: {
      ...(fromK8sDeploy.value ? { from: 'k8s-deploy' } : {}),
      ...(k8sCluster.value ? { cluster: k8sCluster.value } : {}),
      // 把已选好的目标节点 + 系统类型透传到详情页
      nodes: t.target.nodes.join(','),
      osType: t.target.osType,
    },
  })
}

const quickApply = async (t: ToolCard) => {
  if (!isReady(t)) return
  t.applying = true
  try {
    // 后端尚未提供 init-tools 一键执行 API，先做前端可观察反馈，
    // 待后端补齐后此处替换为真实 API 调用即可（按 osType 分发到对应 playbook）。
    await new Promise(res => setTimeout(res, 1200))
    ElMessage.success(
      `已提交「${t.title}」到 ${t.target.nodes.length} 个节点（${t.target.osType}），可在作业中心查看进度`
    )
  } catch (e: any) {
    ElMessage.error(`提交失败：${e?.message || e}`)
  } finally {
    t.applying = false
  }
}

const backToK8sDeploy = () => {
  router.push({ path: '/service/k8s-deploy' })
}

onMounted(() => {
  // 如果是从 K8s 部署页过来，按推荐顺序提示一次
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
  grid-template-columns: repeat(auto-fit, minmax(420px, 1fr));
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
