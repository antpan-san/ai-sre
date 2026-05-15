<template>
  <div
    class="main-layout flex h-screen overflow-hidden bg-[var(--layout-page-bg)] font-sans text-[#1d1d1f] antialiased leading-[1.47] tracking-[-0.01em]"
  >
    <aside
      class="sidebar fixed bottom-0 left-0 top-0 z-[9980] flex flex-col overflow-hidden border-r border-black/[0.04] bg-white transition-[width] duration-[280ms] ease-[cubic-bezier(0.4,0,0.2,1)]"
      :class="isCollapse ? 'w-[var(--layout-sidebar-collapsed-width)]' : 'w-[var(--layout-sidebar-width)]'"
    >
      <div
        class="sidebar-header flex h-[var(--layout-header-height)] shrink-0 items-center justify-between gap-2 border-b border-black/[0.04] px-3.5 backdrop-blur-sm bg-white/90"
      >
        <div class="sidebar-brand flex min-w-0 flex-1 items-center gap-2.5">
          <div
            class="sidebar-brand-mark flex h-9 w-9 shrink-0 items-center justify-center rounded-[10px] bg-[#0066CC] text-xs font-semibold tracking-tight text-white shadow-none"
            aria-hidden="true"
          >
            {{ brandShort }}
          </div>
          <div v-show="!isCollapse" class="sidebar-brand-text flex min-w-0 flex-col gap-0.5">
            <span class="sidebar-brand-title truncate text-[15px] font-semibold leading-snug tracking-tight"
              >OpsFleetPilot</span
            >
            <span class="sidebar-brand-sub truncate text-[11px] font-normal leading-snug opacity-90">{{
              isAdminShell ? '控制台' : '工作台'
            }}</span>
          </div>
        </div>
        <el-button type="primary" link class="collapse-btn shrink-0 -mr-1 h-auto rounded-lg p-2" @click="isCollapse = !isCollapse">
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
          background-color="transparent"
          :text-color="menuTextColor"
          :active-text-color="menuActiveColor"
          :collapse="isCollapse"
          :collapse-transition="true"
        >
          <template v-if="isAdminShell">
            <el-menu-item index="/admin/dashboard">
              <el-icon><PieChart /></el-icon>
              <template #title>概览</template>
            </el-menu-item>
            <el-menu-item v-if="isSuperAdmin" index="/admin/billing/features">
              <el-icon><Setting /></el-icon>
              <template #title>订阅与计费</template>
            </el-menu-item>
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
            <el-menu-item v-if="featureVisible('feature.node_ops')" index="/admin/proxy/config">
              <el-icon><Link /></el-icon>
              <template #title>出口代理</template>
            </el-menu-item>
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
              <el-menu-item index="/admin/execution-records">执行记录</el-menu-item>
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
            <el-sub-menu v-if="featureVisible('feature.backup_performance')" index="asm-advanced">
              <template #title>
                <el-icon><DocumentCopy /></el-icon>
                <span
                  >数据<span v-if="featureBillingEnabled('feature.backup_performance')" class="menu-pack-tag">订阅</span></span
                >
              </template>
              <el-menu-item index="/admin/advanced/backup-restore">备份</el-menu-item>
              <el-menu-item index="/admin/advanced/performance-analysis">性能</el-menu-item>
            </el-sub-menu>
          </template>
          <template v-else>
            <el-menu-item index="/app/dashboard">
              <el-icon><PieChart /></el-icon>
              <template #title>概览</template>
            </el-menu-item>
            <el-menu-item index="/app/job/center">
              <el-icon><Management /></el-icon>
              <template #title>作业中心</template>
            </el-menu-item>
            <el-menu-item index="/app/execution-records">
              <el-icon><List /></el-icon>
              <template #title>执行记录</template>
            </el-menu-item>
            <el-menu-item v-if="featureVisible('feature.node_ops')" index="/app/init-tools">
              <el-icon><Tools /></el-icon>
              <template #title>节点初始化</template>
            </el-menu-item>
            <el-sub-menu v-if="featureVisible('feature.backup_performance')" index="app-advanced">
              <template #title>
                <el-icon><DocumentCopy /></el-icon>
                <span
                  >数据<span v-if="featureBillingEnabled('feature.backup_performance')" class="menu-pack-tag">订阅</span></span
                >
              </template>
              <el-menu-item index="/app/advanced/backup-restore">备份</el-menu-item>
              <el-menu-item index="/app/advanced/performance-analysis">性能</el-menu-item>
            </el-sub-menu>
          </template>
        </el-menu>
      </el-scrollbar>
    </aside>

    <div
      class="main-content flex flex-col overflow-hidden transition-[margin-left,width] duration-[280ms] ease-[cubic-bezier(0.4,0,0.2,1)]"
      :class="
        isCollapse
          ? 'ml-[var(--layout-sidebar-collapsed-width)] w-[calc(100%-var(--layout-sidebar-collapsed-width))]'
          : 'ml-[var(--layout-sidebar-width)] w-[calc(100%-var(--layout-sidebar-width))]'
      "
    >
      <header
        class="header sticky top-0 z-[10050] flex h-[var(--layout-header-height)] shrink-0 items-center justify-between gap-4 border-b border-black/[0.06] bg-white/70 px-5 shadow-apple-nav backdrop-blur-xl supports-[backdrop-filter]:bg-white/55"
      >
        <div class="header-left flex min-w-0 flex-1 items-center">
          <div class="header-breadcrumb-wrap min-w-0 max-w-full overflow-x-auto [-ms-overflow-style:none] [scrollbar-width:none] [&::-webkit-scrollbar]:h-0">
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
        <div class="header-right flex shrink-0 items-center gap-2.5">
          <router-link
            :to="{ path: errorCodesPath }"
            class="header-quick-codes rounded-full px-3 py-1.5 text-sm font-medium tracking-tight text-[#1d1d1f]"
            :aria-current="route.path.endsWith('/help/error-codes') ? 'page' : undefined"
          >
            <el-icon :size="18"><Reading /></el-icon>
            <span>错误码</span>
          </router-link>

          <el-popover
            placement="bottom-end"
            :width="440"
            trigger="hover"
            :show-after="180"
            popper-class="install-ai-sre-popover"
          >
            <template #reference>
              <button type="button" class="install-ai-sre-trigger" aria-haspopup="true" aria-label="安装 ai-sre，悬停显示命令">
                <el-icon :size="18"><Download /></el-icon>
                <span class="install-ai-sre-trigger__text">安装 ai-sre</span>
                <el-icon class="install-ai-sre-trigger__caret" :size="12"><ArrowDown /></el-icon>
              </button>
            </template>
            <div class="install-ai-sre-panel">
              <p class="install-ai-sre-panel__desc">
                生成一次性安装命令，15 分钟内在控制机执行。安装后写入专用 CLI token，并绑定当前账号与机器指纹。
              </p>
              <el-input
                class="install-ai-sre-panel__input"
                type="textarea"
                :model-value="installAiSreCommand || INSTALL_AI_SRE_PLACEHOLDER"
                :autosize="{ minRows: 3, maxRows: 6 }"
                readonly
              />
              <p v-if="installAiSreExpiresAt" class="install-ai-sre-panel__expires">
                有效期至 {{ formatInstallExpiresAt(installAiSreExpiresAt) }}
              </p>
              <div class="install-ai-sre-panel__actions">
                <el-button
                  type="primary"
                  size="small"
                  :loading="installAiSreGenerating"
                  :disabled="!installAiSreCommandHasToken"
                  @click="copyInstallAiSreCommand"
                >
                  生成并复制命令
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

      <main
        class="content flex flex-1 min-h-0 flex-col pb-[var(--layout-content-gutter)] pl-0 pr-[var(--layout-content-gutter)] pt-2"
      >
        <div class="content-inner flex min-h-0 flex-1 flex-col overflow-hidden bg-transparent p-0">
          <div
            class="content-route flex min-h-0 flex-1 flex-col overflow-y-auto overflow-x-hidden [-webkit-overflow-scrolling:touch]"
          >
            <router-view v-slot="{ Component }">
              <transition name="page-apple-fade-rise" mode="out-in">
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
  Link,
  Download,
  Reading,
  Setting,
  List
} from '@element-plus/icons-vue'
import { wsService } from '../../utils/websocket'
import { copyTextToClipboard } from '../../utils/clipboard'
import { INSTALL_AI_SRE_PLACEHOLDER, getStoredAuthToken } from '../../utils/installAiSre'
import { useMachineStore } from '../../stores/machine'
import { getBillingCapabilities, type BillingCapabilityFeature } from '../../api/billing'
import { createCLIInstallSession } from '../../api/cli'

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
  '/proxy': Link,
  '/monitoring': Monitor,
  '/job': Management,
  '/security-audit': Lock,
  '/advanced': DocumentCopy,
  '/init-tools': Tools,
  '/execution-records': DocumentCopy,
  '/help/error-codes': Reading,
  '/billing/features': Setting
}

const sectionDefaultPath: Record<string, string> = {
  '/service': '/service/deploy',
  '/proxy': '/proxy/config',
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
const isAdminUser = computed(() => ['admin', 'super_admin'].includes(String(currentUser.value?.role ?? '')))

const isCollapse = ref(false)
const capabilityByFeature = ref<Record<string, BillingCapabilityFeature>>({})

const menuTextColor = 'var(--layout-sidebar-text)'
const menuActiveColor = 'var(--el-color-primary)'

const menuDefaultOpeneds = computed(() => {
  const p = route.path
  const open: string[] = []
  if (p.startsWith('/app')) {
    if (p.includes('/app/advanced')) open.push('app-advanced')
    return open
  }
  if (p.includes('/admin/service')) open.push('asm-workloads')
  if (p.includes('/admin/monitoring')) open.push('asm-observe')
  if (p.includes('/admin/job') || p.includes('/admin/execution-records')) open.push('asm-run')
  if (p.includes('/admin/security-audit')) open.push('asm-security')
  if (p.includes('/admin/advanced')) open.push('asm-advanced')
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
const installAiSreCommandHasToken = computed(() => !!getStoredAuthToken())

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

const activeMenu = computed(() => {
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
  if (Array.isArray(explicitBreadcrumb) && explicitBreadcrumb.length > 0) {
    explicitBreadcrumb.forEach(item => {
      if (item.title?.trim()) {
        pushBreadcrumb(items, item.title.trim(), item.path)
      }
    })
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
/* Sidebar scroll + Element Plus（Tailwind :deep 难覆盖的细节） */
.sidebar-scroll {
  flex: 1;
  min-height: 0;
}

.sidebar-scroll :deep(.el-scrollbar__view) {
  padding-bottom: 16px;
}

.sidebar-menu {
  border-right: none !important;
  padding: 10px 8px 16px;
  --el-menu-base-level-padding: 14px;
  --el-menu-icon-width: 22px;
  --el-menu-item-height: 44px;
}

.sidebar-menu:not(.el-menu--collapse) {
  width: 100%;
}

.collapse-btn {
  flex-shrink: 0;
  color: var(--layout-sidebar-text);
}

.collapse-btn:hover {
  color: var(--el-color-primary);
  background-color: var(--layout-sidebar-hover-bg);
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
  font-weight: 400;
  line-height: 1.35;
  color: var(--layout-sidebar-text);
}

.breadcrumb-item {
  display: inline-flex;
  align-items: center;
  padding: 2px 4px;
  border-radius: 6px;
  transition:
    color 0.18s cubic-bezier(0.4, 0, 0.2, 1),
    background-color 0.18s cubic-bezier(0.4, 0, 0.2, 1);
}

.breadcrumb-item:hover {
  color: var(--el-color-primary);
}

.breadcrumb-item--current {
  color: var(--layout-sidebar-text-strong);
  font-weight: 600;
  cursor: default;
}

.breadcrumb-icon {
  margin-right: 4px;
  font-size: 14px;
  color: currentColor;
}

.custom-breadcrumb :deep(.el-breadcrumb__separator) {
  margin: 0 8px;
  color: #d2d2d7;
  font-weight: 500;
}

.header-quick-codes {
  transition:
    background-color 0.2s ease,
    color 0.2s ease;
}

.header-quick-codes:hover {
  background-color: var(--apple-canvas-parchment, #f5f5f7);
  color: #0071e3;
}

.header-quick-codes.router-link-active {
  background-color: var(--apple-canvas-parchment, #f5f5f7);
  color: #0071e3;
  font-weight: 600;
}

.install-ai-sre-trigger {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  padding: 6px 14px;
  border: 0;
  border-radius: 980px;
  background: rgba(0, 113, 227, 0.08);
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  line-height: 1.29;
  color: #0071e3;
  cursor: pointer;
  transition:
    background-color 0.2s ease,
    color 0.2s ease;
}

.install-ai-sre-trigger:hover {
  background-color: rgba(0, 113, 227, 0.14);
}

.install-ai-sre-trigger__text {
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.install-ai-sre-trigger__caret {
  color: var(--layout-sidebar-text);
  margin-left: -2px;
}

.user-trigger {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  padding: 4px 8px 4px 4px;
  border-radius: 999px;
  border: 0;
  transition:
    background-color 0.2s ease,
    color 0.2s ease;
  outline: none;
}

.user-trigger:hover {
  background-color: rgba(0, 0, 0, 0.045);
}

.user-avatar {
  flex-shrink: 0;
  font-size: 14px;
  font-weight: 600;
  color: #fff;
  background: #0071e3 !important;
}

.user-trigger-name {
  font-size: 14px;
  font-weight: 500;
  line-height: 1.29;
  color: var(--layout-sidebar-text-strong);
  max-width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-trigger-chevron {
  font-size: 12px;
  color: var(--layout-sidebar-text);
  margin-right: 2px;
}

.page-apple-fade-rise-enter-active,
.page-apple-fade-rise-leave-active {
  transition:
    opacity 0.42s cubic-bezier(0.4, 0, 0.2, 1),
    transform 0.42s cubic-bezier(0.4, 0, 0.2, 1);
}

.page-apple-fade-rise-enter-from {
  opacity: 0;
  transform: translateY(12px);
}

.page-apple-fade-rise-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

.sidebar-menu :deep(.el-sub-menu__title),
.sidebar-menu :deep(.el-menu-item) {
  border-radius: 10px;
  margin: 2px 0;
  font-weight: 500;
  letter-spacing: -0.015em;
  font-family: var(--apple-font-text), -apple-system, BlinkMacSystemFont, system-ui, sans-serif;
}

.sidebar-menu :deep(.el-sub-menu__title:hover),
.sidebar-menu :deep(.el-menu-item:hover) {
  background: linear-gradient(90deg, rgba(245, 245, 247, 0.98) 0%, rgba(255, 255, 255, 0.35) 65%, transparent 100%) !important;
}

.sidebar-menu :deep(.el-menu-item.is-active) {
  background-color: var(--layout-sidebar-active-bg) !important;
  color: var(--el-color-primary) !important;
  font-weight: 600;
}

.sidebar-menu :deep(.el-sub-menu.is-active > .el-sub-menu__title) {
  color: var(--layout-sidebar-text-strong);
}

.sidebar-menu :deep(.el-menu-item .el-icon),
.sidebar-menu :deep(.el-sub-menu__title .el-icon) {
  font-size: 18px;
}

.sidebar-menu :deep(.el-sub-menu .el-menu) {
  background-color: transparent !important;
}

.sidebar-menu :deep(.el-sub-menu .el-menu-item) {
  min-height: 40px;
  height: auto;
  line-height: 1.35;
  padding-top: 9px !important;
  padding-bottom: 9px !important;
  padding-right: 12px !important;
  margin: 1px 0;
  font-weight: 400;
  font-size: 13px;
}

.sidebar-menu :deep(.el-sub-menu .el-menu-item.is-active) {
  font-weight: 600;
}

.sidebar-menu.el-menu--collapse :deep(.el-sub-menu__title) {
  padding: 0 calc(var(--el-menu-base-level-padding) + 2px) !important;
}

.sidebar-menu.el-menu--collapse :deep(.el-tooltip__trigger) {
  justify-content: center;
}

.menu-pack-tag {
  display: inline-flex;
  align-items: center;
  height: 18px;
  margin-left: 6px;
  padding: 0 5px;
  border-radius: 4px;
  background: var(--apple-canvas-parchment, #f5f5f7);
  color: #0071e3;
  font-size: 11px;
  font-weight: 600;
}
</style>
