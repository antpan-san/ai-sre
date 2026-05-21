<template>
  <div class="main-layout">
    <aside class="sidebar" :class="{ 'sidebar-collapsed': isCollapse }">
      <div class="sidebar-header">
        <div class="sidebar-brand">
          <div class="sidebar-brand-mark" aria-hidden="true">{{ brandShort }}</div>
          <div v-show="!isCollapse" class="sidebar-brand-text">
            <span class="sidebar-brand-title">OpsFleetPilot</span>
            <span class="sidebar-brand-sub">{{ isAdminShell ? '控制台' : '工作台' }}</span>
          </div>
        </div>
        <el-button type="primary" link class="collapse-btn" @click="isCollapse = !isCollapse">
          <el-icon :size="18">
            <svg
              v-if="isCollapse"
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 1024 1024"
              width="20"
              height="20"
            >
              <path
                d="M877.824 505.728l-480.64 480.64c-12.544 12.544-32.96 12.544-45.504 0-12.544-12.544-12.544-32.96 0-45.504l458.112-458.112-458.112-458.112c-12.544-12.544-12.544-32.96 0-45.504 12.544-12.544 32.96-12.544 45.504 0l480.64 480.64c12.544 12.544 12.544 32.96 0 45.504z"
                fill="currentColor"
              />
            </svg>
            <svg
              v-else
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 1024 1024"
              width="20"
              height="20"
            >
              <path
                d="M867.84 512c0 12.544-10.048 22.528-22.528 22.528h-616.96l294.4 294.4c12.544 12.544 12.544 32.96 0 45.504-12.544 12.544-32.96 12.544-45.504 0l-360.96-360.96c-12.544-12.544-12.544-32.96 0-45.504l360.96-360.96c12.544-12.544 32.96-12.544 45.504 0 12.544 12.544 12.544 32.96 0 45.504l-294.4 294.4h616.96c12.544 0 22.528 10.048 22.528 22.528z"
                fill="currentColor"
              />
            </svg>
          </el-icon>
        </el-button>
      </div>
      <el-scrollbar class="sidebar-scroll">
        <el-menu
          :key="menuRemountKey"
          :default-active="activeMenu"
          :default-openeds="menuDefaultOpeneds"
          class="sidebar-menu"
          @select="handleMenuSelect"
          :collapse="isCollapse"
          :collapse-transition="true"
        >
          <template v-if="isAdminShell">
            <el-menu-item index="/admin/dashboard">
              <el-icon><PieChart /></el-icon>
              <template #title>概览</template>
            </el-menu-item>
            <el-sub-menu index="asm-aisre">
              <template #title>
                <el-icon><Collection /></el-icon>
                <span>ai-sre 中心</span>
              </template>
              <el-menu-item index="/admin/ai-sre/executions">
                <el-icon><List /></el-icon>
                <template #title>客户端执行</template>
              </el-menu-item>
              <el-menu-item v-if="isSuperAdmin" index="/admin/ai-sre/skill-refinement">
                <el-icon><MagicStick /></el-icon>
                <template #title>技能精炼</template>
              </el-menu-item>
              <el-menu-item index="/admin/execution-records">
                <el-icon><DocumentCopy /></el-icon>
                <template #title>通用执行审计</template>
              </el-menu-item>
              <el-menu-item v-if="featureVisible('feature.runtime_observe')" index="/admin/advanced/runtime-observe">
                <el-icon><Monitor /></el-icon>
                <template #title
                  >运行时诊断<span v-if="featureBillingEnabled('feature.runtime_observe')" class="menu-pack-tag">订阅</span></template
                >
              </el-menu-item>
            </el-sub-menu>
            <el-menu-item v-if="isSuperAdmin" index="/admin/auto-iterations">
              <el-icon><Refresh /></el-icon>
              <template #title>自动迭代</template>
            </el-menu-item>
            <el-sub-menu v-if="isSuperAdmin" index="asm-billing">
              <template #title>
                <el-icon><Setting /></el-icon>
                <span>订阅与计费</span>
              </template>
              <el-menu-item index="/admin/billing/features">
                <el-icon><Setting /></el-icon>
                <template #title>订阅包与计费</template>
              </el-menu-item>
              <el-menu-item index="/admin/billing/ai-sre-skills">
                <el-icon><Collection /></el-icon>
                <template #title>ai-sre 技能包</template>
              </el-menu-item>
            </el-sub-menu>
            <el-sub-menu
              v-if="featureVisible('feature.node_ops') || featureVisible('feature.k8s_delivery')"
              index="asm-workloads"
            >
              <template #title>
                <el-icon><Box /></el-icon>
                <span>工作负载</span>
              </template>
              <el-menu-item v-if="featureVisible('feature.node_ops')" index="/admin/service/deploy">
                <el-icon><Operation /></el-icon>
                <template #title>应用服务</template>
              </el-menu-item>
              <el-menu-item v-if="featureVisible('feature.k8s_delivery')" index="/admin/service/k8s-deploy">
                <el-icon><Connection /></el-icon>
                <template #title>Kubernetes<span v-if="featureBillingEnabled('feature.k8s_delivery')" class="menu-pack-tag">订阅</span></template>
              </el-menu-item>
              <el-menu-item v-if="featureVisible('feature.node_ops')" index="/admin/service/linux">
                <el-icon><Cpu /></el-icon>
                <template #title>Linux 主机</template>
              </el-menu-item>
            </el-sub-menu>
            <el-menu-item v-if="featureVisible('feature.k8s_delivery')" index="/admin/k8s-mirror">
              <el-icon><Download /></el-icon>
              <template #title>制品目录<span v-if="featureBillingEnabled('feature.k8s_delivery')" class="menu-pack-tag">订阅</span></template>
            </el-menu-item>
            <el-sub-menu v-if="featureVisible('feature.monitoring')" index="asm-observe">
              <template #title>
                <el-icon><Monitor /></el-icon>
                <span
                  >可观测性<span v-if="featureBillingEnabled('feature.monitoring')" class="menu-pack-tag">订阅</span></span
                >
              </template>
              <el-menu-item index="/admin/monitoring/prometheus">Prometheus</el-menu-item>
              <el-menu-item index="/admin/monitoring/node-exporter">Node</el-menu-item>
              <el-menu-item index="/admin/monitoring/jmx-exporter">JMX</el-menu-item>
              <el-menu-item index="/admin/monitoring/redis-exporter">Redis</el-menu-item>
              <el-menu-item index="/admin/monitoring/mongodb-exporter">MongoDB</el-menu-item>
              <el-menu-item index="/admin/monitoring/blackbox-exporter">Blackbox</el-menu-item>
            </el-sub-menu>
            <el-sub-menu index="asm-run">
              <template #title>
                <el-icon><Management /></el-icon>
                <span>任务</span>
              </template>
              <el-menu-item index="/admin/job/center">作业中心</el-menu-item>
            </el-sub-menu>
            <el-sub-menu index="asm-security">
              <template #title>
                <el-icon><Lock /></el-icon>
                <span>安全</span>
              </template>
              <el-menu-item index="/admin/security-audit/operation-logs">审计日志</el-menu-item>
              <el-menu-item v-if="isAdminUser" index="/admin/security-audit/permission-management"
                >权限</el-menu-item
              >
            </el-sub-menu>
            <el-sub-menu
              v-if="featureVisible('feature.backup_performance')"
              index="asm-advanced"
            >
              <template #title>
                <el-icon><DocumentCopy /></el-icon>
                <span>
                  数据
                  <span v-if="featureBillingEnabled('feature.backup_performance')" class="menu-pack-tag">订阅</span>
                </span>
              </template>
              <el-menu-item v-if="featureVisible('feature.backup_performance')" index="/admin/advanced/backup-restore"
                >备份</el-menu-item>
              <el-menu-item v-if="featureVisible('feature.backup_performance')" index="/admin/advanced/performance-analysis"
                >性能</el-menu-item>
            </el-sub-menu>
          </template>
          <template v-else>
            <el-menu-item index="/app/dashboard">
              <el-icon><PieChart /></el-icon>
              <template #title>概览</template>
            </el-menu-item>
            <el-menu-item index="/app/execution-records">
              <el-icon><DocumentCopy /></el-icon>
              <template #title>执行记录</template>
            </el-menu-item>
            <el-menu-item index="/app/deploy">
              <el-icon><Box /></el-icon>
              <template #title>部署中心</template>
            </el-menu-item>
            <el-menu-item index="/app/troubleshooting">
              <el-icon><Search /></el-icon>
              <template #title>问题排查</template>
            </el-menu-item>
            <el-menu-item index="/app/job/center">
              <el-icon><Management /></el-icon>
              <template #title>作业中心</template>
            </el-menu-item>
            <el-menu-item index="/app/settings">
              <el-icon><Setting /></el-icon>
              <template #title>设置</template>
            </el-menu-item>
          </template>
        </el-menu>
      </el-scrollbar>
    </aside>

    <div class="main-shell" :class="{ 'main-shell--collapsed': isCollapse }">
      <header class="layout-header" :class="{ 'layout-header--with-rings': isSuperAdmin }">
        <div class="layout-header-left">
          <div class="breadcrumb-wrap">
            <el-breadcrumb separator="/" class="custom-breadcrumb header-breadcrumb">
              <template v-for="item in breadcrumbItems" :key="item.key">
                <el-breadcrumb-item
                  v-if="item.to && !item.current"
                  :to="{ path: item.to }"
                  class="breadcrumb-item"
                >
                  <el-icon v-if="item.icon" class="breadcrumb-icon">
                    <component :is="item.icon" />
                  </el-icon>
                  <span>{{ item.title }}</span>
                </el-breadcrumb-item>
                <el-breadcrumb-item v-else class="breadcrumb-item breadcrumb-item--current">
                  <el-icon v-if="item.icon" class="breadcrumb-icon">
                    <component :is="item.icon" />
                  </el-icon>
                  <span>{{ item.title }}</span>
                </el-breadcrumb-item>
              </template>
            </el-breadcrumb>
          </div>
        </div>
        <HostResourceRings v-if="isSuperAdmin" class="layout-header-center" />
        <div class="layout-header-right">
          <router-link
            :to="{ path: errorCodesPath }"
            class="header-quick-codes"
            :aria-current="route.path.endsWith('/help/error-codes') ? 'page' : undefined"
          >
            <el-icon :size="18"><Reading /></el-icon>
            <span>错误码</span>
          </router-link>

          <el-popover
            placement="bottom-end"
            :width="268"
            trigger="hover"
            :show-after="180"
            popper-class="install-ai-sre-popover"
            @show="loadInstallAiSreAdvertisedVersion"
          >
            <template #reference>
              <button
                type="button"
                class="install-ai-sre-trigger"
                aria-haspopup="true"
                :aria-label="installAiSreTriggerAriaLabel"
              >
                <el-icon :size="16"><Download /></el-icon>
                <span class="install-ai-sre-trigger__text">ai-sre</span>
              </button>
            </template>
            <div class="install-ai-sre-panel">
              <p class="install-ai-sre-panel__meta">
                <template v-if="installAiSreAdvertisedVersion">v{{ installAiSreAdvertisedVersion }}</template>
                <span v-else-if="installAiSreVersionLoading">…</span>
                <span v-else>—</span>
                <span class="install-ai-sre-panel__dot">·</span>
                15 分钟有效
              </p>
              <el-input
                class="install-ai-sre-panel__input"
                type="textarea"
                :model-value="installAiSreCommand || INSTALL_AI_SRE_PLACEHOLDER"
                :autosize="{ minRows: 2, maxRows: 3 }"
                readonly
              />
              <div class="install-ai-sre-panel__actions">
                <span v-if="installAiSreExpiresAt" class="install-ai-sre-panel__expires">
                  {{ formatInstallExpiresAt(installAiSreExpiresAt) }} 前
                </span>
                <el-button
                  type="primary"
                  size="small"
                  :loading="installAiSreGenerating"
                  :disabled="!installAiSreCommandHasToken"
                  @click="copyInstallAiSreCommand"
                >
                  生成复制
                </el-button>
              </div>
            </div>
          </el-popover>

          <el-dropdown trigger="click" popper-class="layout-user-dropdown">
            <div class="user-trigger" role="button" tabindex="0">
              <el-avatar :size="34" class="user-avatar">{{ userInitial }}</el-avatar>
              <span class="user-trigger-name">{{ currentUser.username }}</span>
              <el-icon class="user-trigger-chevron"><ArrowDown /></el-icon>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item v-if="isAdminUser" @click="handleSwitchShell">
                  <el-icon><Monitor /></el-icon>
                  {{ isAdminShell ? '切换到工作台' : '切换到管理控制台' }}
                </el-dropdown-item>
                <el-dropdown-item v-if="isAdminUser" @click="handleUserManagement">
                  <el-icon><User /></el-icon>
                  用户
                </el-dropdown-item>
                <el-dropdown-item :divided="isAdminUser" @click="handleLogout">
                  <el-icon><SwitchButton /></el-icon>
                  退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </header>

      <main class="layout-main">
        <div class="layout-main-inner">
          <div class="layout-main-scroll" :class="{ 'layout-main-scroll--lock': lockMainScroll }">
            <router-view v-slot="{ Component }">
              <transition name="fade" mode="out-in">
                <component :is="Component" :key="route.fullPath" />
              </transition>
            </router-view>
          </div>
        </div>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
import type { Component } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { RouteLocationMatched } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  User,
  SwitchButton,
  ArrowDown,
  PieChart,
  Monitor,
  Management,
  Tools,
  Lock,
  DocumentCopy,
  Box,
  Operation,
  Connection,
  Cpu,
  Download,
  Reading,
  Setting,
  List,
  Collection,
  Refresh,
  MagicStick,
  Search
} from '@element-plus/icons-vue'
import { wsService } from '../../utils/websocket'
import { copyTextToClipboard } from '../../utils/clipboard'
import { INSTALL_AI_SRE_PLACEHOLDER, getStoredAuthToken } from '../../utils/installAiSre'
import { useMachineStore } from '../../stores/machine'
import { getBillingCapabilities, type BillingCapabilityFeature } from '../../api/billing'
import { createCLIInstallSession, fetchAiSreCLIVersion } from '../../api/cli'
import HostResourceRings from './HostResourceRings.vue'
import { CAPABILITY_CATEGORY_LABELS, type CapabilityCategory } from '../../config/capabilityCatalog'

type BreadcrumbMetaItem = {
  title: string
  path?: string
}

type BreadcrumbItem = {
  key: string
  title: string
  to?: string
  icon?: Component
  current: boolean
}

const routeIconMap: Record<string, Component> = {
  '/dashboard': PieChart,
  '/service': Box,
  '/service/deploy': Operation,
  '/service/k8s-deploy': Connection,
  '/k8s-mirror': Download,
  '/service/linux': Cpu,
  '/monitoring': Monitor,
  '/job': Management,
  '/security-audit': Lock,
  '/advanced': DocumentCopy,
  '/init-tools': Tools,
  '/execution-records': DocumentCopy,
  '/help/error-codes': Reading,
  '/billing/features': Setting,
  '/billing/ai-sre-skills': Collection,
  '/auto-iterations': Refresh
}

const sectionDefaultPath: Record<string, string> = {
  '/service': '/service/deploy',
  '/monitoring': '/monitoring/prometheus',
  '/job': '/job/center',
  '/security-audit': '/security-audit/operation-logs',
  '/advanced': '/advanced/backup-restore',
  '/user': '/user/list',
  '/init-tools': '/init-tools',
  '/execution-records': '/execution-records',
  '/help': '/help/error-codes',
  '/billing': '/billing/features'
}

const route = useRoute()
const router = useRouter()

const currentUser = computed(() => {
  const userInfoStr = localStorage.getItem('userInfo')
  if (userInfoStr) {
    try {
      return JSON.parse(userInfoStr)
    } catch (e) {
      return { username: '管理员' }
    }
  }
  return { username: '管理员' }
})

const navBase = computed(() => (route.path.startsWith('/admin') ? '/admin' : '/app'))
/** 高频工具：顶栏直达，与同壳层侧栏路由一致 */
const errorCodesPath = computed(() => `${navBase.value}/help/error-codes`)
const isAdminShell = computed(() => route.path.startsWith('/admin'))
const isSuperAdmin = computed(() => String(currentUser.value?.role ?? '') === 'super_admin')
const lockMainScroll = computed(
  () =>
    route.path.includes('/admin/auto-iterations') ||
    route.path.endsWith('/dashboard') ||
    route.path.endsWith('/admin/dashboard') ||
    route.path.endsWith('/app/dashboard')
)
const isAdminUser = computed(() => ['admin', 'super_admin'].includes(String(currentUser.value?.role ?? '')))

const isCollapse = ref(false)
const capabilityByFeature = ref<Record<string, BillingCapabilityFeature>>({})

const isRuntimeObservePath = (p: string) => p.includes('/advanced/runtime-observe')

const menuDefaultOpeneds = computed(() => {
  const p = route.path
  const open: string[] = []
  if (p.startsWith('/app')) {
    if (p.includes('/app/deploy') || p.includes('/app/workloads')) open.push('app-workloads')
    return open
  }
  if (p.includes('/admin/ai-sre') || p.includes('/admin/execution-records') || isRuntimeObservePath(p)) {
    open.push('asm-aisre')
  }
  if (p.includes('/admin/service')) open.push('asm-workloads')
  if (p.includes('/admin/monitoring')) open.push('asm-observe')
  if (p.includes('/admin/job')) open.push('asm-run')
  if (p.includes('/admin/security-audit')) open.push('asm-security')
  if (p.includes('/admin/advanced') && !isRuntimeObservePath(p)) open.push('asm-advanced')
  if (p.includes('/admin/billing')) open.push('asm-billing')
  return open
})

const menuRemountKey = computed(() => `${isAdminShell.value ? 'admin' : 'app'}|${menuDefaultOpeneds.value.join('|')}`)

const userInitial = computed(() => {
  const name = String(currentUser.value?.username ?? '?').trim()
  const ch = name.slice(0, 1)
  return ch || '?'
})

const brandShort = computed(() => 'OP')

const installAiSreCommand = ref('')
const installAiSreExpiresAt = ref('')
const installAiSreGenerating = ref(false)
const installAiSreAdvertisedVersion = ref('')
const installAiSreVersionLoading = ref(false)
const installAiSreCommandHasToken = computed(() => !!getStoredAuthToken())
const installAiSreTriggerAriaLabel = computed(() => {
  const ver = installAiSreAdvertisedVersion.value
  return ver ? `安装 ai-sre v${ver}，悬停显示命令` : '安装 ai-sre，悬停显示命令'
})

const loadInstallAiSreAdvertisedVersion = async () => {
  if (installAiSreVersionLoading.value) return
  installAiSreVersionLoading.value = true
  try {
    const info = await fetchAiSreCLIVersion()
    installAiSreAdvertisedVersion.value = info?.version?.trim() || ''
  } finally {
    installAiSreVersionLoading.value = false
  }
}

const featureVisible = (featureKey: string) => {
  const row = capabilityByFeature.value[featureKey]
  return row ? row.visible_enabled && row.can_view !== false : true
}

const featureBillingEnabled = (featureKey: string) => {
  return capabilityByFeature.value[featureKey]?.billing_enabled === true
}

const loadBillingCapabilities = async () => {
  try {
    const data = await getBillingCapabilities()
    const next: Record<string, BillingCapabilityFeature> = {}
    ;(data.features || []).forEach((item) => {
      next[item.feature_key] = item
    })
    capabilityByFeature.value = next
  } catch {
    capabilityByFeature.value = {}
  }
}

const copyInstallAiSreCommand = async () => {
  if (!installAiSreCommandHasToken.value) {
    ElMessage.warning('请先登录后再生成安装命令')
    return
  }
  installAiSreGenerating.value = true
  try {
    const data = await createCLIInstallSession()
    installAiSreCommand.value = data.command
    installAiSreExpiresAt.value = data.expires_at
    await copyTextToClipboard(data.command)
    ElMessage.success('已生成并复制安装命令')
  } catch {
    ElMessage.error('生成或复制失败，请稍后重试')
  } finally {
    installAiSreGenerating.value = false
  }
}

const formatInstallExpiresAt = (value: string) => {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

const machineStore = useMachineStore()
const handleMachineHeartbeatMessage = (msg: any) => {
  machineStore.handleMachineHeartbeat(msg.data)
}
const handleMachineStatusMessage = (msg: any) => {
  machineStore.handleMachineStatusUpdate(msg.data || [])
}

onMounted(() => {
  const userId = currentUser.value?.id || 'anonymous'
  wsService.connect(String(userId))
  void loadInstallAiSreAdvertisedVersion()
  void loadBillingCapabilities()

  wsService.on('machine_heartbeat', handleMachineHeartbeatMessage)
  wsService.on('machine_status_update', handleMachineStatusMessage)
})

onUnmounted(() => {
  wsService.off('machine_heartbeat', handleMachineHeartbeatMessage)
  wsService.off('machine_status_update', handleMachineStatusMessage)
  wsService.disconnect()
})

const handleUserManagement = () => {
  router.push('/admin/user/list')
}

const handleSwitchShell = () => {
  router.push(isAdminShell.value ? '/app/dashboard' : '/admin/dashboard')
}

const activeMenu = computed(() => {
  const hub = route.meta.hubMenu as string | undefined
  if (hub) return hub
  return route.path
})

function toLogicalRoutePath(full: string): string {
  if (!full) return '/'
  if (full.startsWith('/admin/')) return full.slice(6)
  if (full === '/admin') return '/'
  if (full.startsWith('/app/')) return full.slice(4)
  if (full === '/app') return '/'
  return full.startsWith('/') ? full : `/${full}`
}

const getRouteIcon = (path?: string): Component | undefined => {
  const raw = path ?? route.path
  const logical = toLogicalRoutePath(raw)
  if (routeIconMap[logical]) {
    return routeIconMap[logical]
  }

  const parentPath = logical.substring(0, logical.lastIndexOf('/'))
  if (parentPath && routeIconMap[parentPath]) {
    return routeIconMap[parentPath]
  }

  return undefined
}

function titleOfRoute(routeItem: RouteLocationMatched): string | undefined {
  const title = routeItem.meta?.breadcrumbTitle ?? routeItem.meta?.title
  return typeof title === 'string' && title.trim() ? title.trim() : undefined
}

function pathOfRoute(routeItem: RouteLocationMatched): string | undefined {
  let p = routeItem.path
  if (!p || p === '/' || p.includes(':')) return undefined
  const base = navBase.value
  const fullUnderShell =
    p.startsWith('/admin') || p.startsWith('/app') ? p : `${base}/${p.replace(/^\//, '')}`
  const logical = toLogicalRoutePath(fullUnderShell)
  if (logical === '/' || logical === '') return undefined
  const def = sectionDefaultPath[logical]
  if (def) {
    return def.startsWith('/admin') || def.startsWith('/app') ? def : base + def
  }
  return fullUnderShell
}

function pushBreadcrumb(items: BreadcrumbItem[], title: string, to?: string): void {
  const last = items[items.length - 1]
  if (last?.title === title) return
  items.push({
    key: `${to ?? title}-${items.length}`,
    title,
    to,
    icon: getRouteIcon(to),
    current: false
  })
}

const breadcrumbItems = computed<BreadcrumbItem[]>(() => {
  const items: BreadcrumbItem[] = []

  const explicitBreadcrumb = route.meta.breadcrumb as BreadcrumbMetaItem[] | undefined
  const hubMenu = route.meta.hubMenu as string | undefined
  const hubSection = route.meta.hubSection as CapabilityCategory | undefined

  if (Array.isArray(explicitBreadcrumb) && explicitBreadcrumb.length > 0) {
    explicitBreadcrumb.forEach(item => {
      if (item.title?.trim()) {
        pushBreadcrumb(items, item.title.trim(), item.path)
      }
    })
  } else if (hubMenu && hubSection) {
    pushBreadcrumb(items, '部署中心', hubMenu)
    const sectionTitle = CAPABILITY_CATEGORY_LABELS[hubSection]
    if (sectionTitle) {
      pushBreadcrumb(items, sectionTitle, hubMenu)
    }
    const pageTitle = route.meta.title
    if (typeof pageTitle === 'string' && pageTitle.trim()) {
      pushBreadcrumb(items, pageTitle.trim(), undefined)
    }
  } else {
    route.matched
      .filter(routeItem => routeItem.name !== 'Login')
      .forEach(routeItem => {
        const title = titleOfRoute(routeItem)
        if (!title) return
        pushBreadcrumb(items, title, pathOfRoute(routeItem))
      })
  }

  const last = items[items.length - 1]
  if (last) {
    last.current = true
    last.to = undefined
  }
  return items
})

const handleMenuSelect = (index: string) => {
  router.push(index)
}

const handleLogout = () => {
  localStorage.removeItem('token')
  localStorage.removeItem('userInfo')
  router.push('/login')
}
</script>

<style scoped>
.main-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
  background: var(--el-bg-color-page);
}

.sidebar {
  position: fixed;
  inset: 0 auto 0 0;
  z-index: 1001;
  width: var(--layout-sidebar-width);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--el-bg-color);
  border-right: 1px solid var(--el-border-color-light);
  transition: width 0.25s ease;
}

.sidebar.sidebar-collapsed {
  width: var(--layout-sidebar-collapsed-width);
}

.sidebar-header {
  flex-shrink: 0;
  height: var(--layout-header-height);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 0 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
}

.sidebar-brand {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  flex: 1;
}

.sidebar-brand-mark {
  flex-shrink: 0;
  width: 36px;
  height: 36px;
  border-radius: var(--el-border-radius-base);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 700;
  color: #fff;
  background: var(--el-color-primary);
}

.sidebar-brand-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.sidebar-brand-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.sidebar-brand-sub {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}

.sidebar-scroll {
  flex: 1;
  min-height: 0;
}

.sidebar-scroll :deep(.el-scrollbar__view) {
  padding-bottom: 12px;
}

.sidebar-menu {
  border-right: none !important;
}

.sidebar-menu:not(.el-menu--collapse) {
  width: 100%;
}

.collapse-btn {
  flex-shrink: 0;
}

.main-shell {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  margin-left: var(--layout-sidebar-width);
  width: calc(100% - var(--layout-sidebar-width));
  min-width: 0;
  transition:
    margin-left 0.25s ease,
    width 0.25s ease;
}

.main-shell--collapsed {
  margin-left: var(--layout-sidebar-collapsed-width);
  width: calc(100% - var(--layout-sidebar-collapsed-width));
}

.layout-header {
  flex-shrink: 0;
  height: var(--layout-header-height);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 16px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-light);
}

.layout-header--with-rings {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
  align-items: center;
}

.layout-header--with-rings .layout-header-left {
  justify-self: start;
}

.layout-header--with-rings .layout-header-center {
  justify-self: center;
}

.layout-header--with-rings .layout-header-right {
  justify-self: end;
}

.layout-header-left {
  flex: 1;
  min-width: 0;
}

.layout-header-center {
  flex-shrink: 0;
}

.breadcrumb-wrap {
  min-width: 0;
  overflow-x: auto;
}

.layout-header-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.header-breadcrumb {
  flex-wrap: nowrap;
  white-space: nowrap;
  padding: 2px 0;
}

.custom-breadcrumb {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 2px 0;
  font-size: 13px;
  color: var(--el-text-color-regular);
}

.breadcrumb-item {
  display: inline-flex;
  align-items: center;
}

.breadcrumb-item--current {
  color: var(--el-text-color-primary);
  font-weight: 500;
}

.breadcrumb-icon {
  margin-right: 4px;
  font-size: 14px;
}

.custom-breadcrumb :deep(.el-breadcrumb__separator) {
  margin: 0 8px;
}

.header-quick-codes {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  font-size: 14px;
  color: var(--el-text-color-primary);
  text-decoration: none;
  border-radius: var(--el-border-radius-base);
}

.header-quick-codes:hover {
  background: var(--el-fill-color-light);
  color: var(--el-color-primary);
}

.header-quick-codes.router-link-active {
  color: var(--el-color-primary);
  font-weight: 500;
}

.install-ai-sre-trigger {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin: 0;
  padding: 4px 8px;
  border: 1px solid var(--el-border-color);
  border-radius: var(--el-border-radius-base);
  background: var(--el-fill-color-blank);
  font: inherit;
  font-size: 12px;
  font-weight: 500;
  color: var(--el-color-primary);
  cursor: pointer;
}

.install-ai-sre-trigger:hover {
  border-color: var(--el-color-primary-light-5);
}

.install-ai-sre-trigger__text {
  max-width: 4.5rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-trigger {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: var(--el-border-radius-base);
}

.user-trigger:hover {
  background: var(--el-fill-color-light);
}

.user-avatar {
  flex-shrink: 0;
  font-size: 14px;
  font-weight: 600;
}

.user-trigger-name {
  font-size: 14px;
  color: var(--el-text-color-primary);
  max-width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.layout-main {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 8px var(--layout-content-gutter) var(--layout-content-gutter);
}

.layout-main-inner {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.layout-main-scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
}

.layout-main-scroll--lock {
  overflow: hidden;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.menu-pack-tag {
  display: inline-flex;
  align-items: center;
  height: 18px;
  margin-left: 6px;
  padding: 0 5px;
  border-radius: var(--el-border-radius-small);
  background: var(--el-fill-color-light);
  color: var(--el-color-primary);
  font-size: 11px;
  font-weight: 500;
}
</style>
