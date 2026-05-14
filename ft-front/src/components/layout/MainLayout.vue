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
              <el-menu-item v-if="featureVisible('feature.k8s_delivery')" index="/admin/service/k8s/clusters">
                <el-icon><Grid /></el-icon>
                <template #title>集群</template>
              </el-menu-item>
              <el-menu-item v-if="featureVisible('feature.k8s_delivery')" index="/admin/service/k8s-mirror">
                <el-icon><Download /></el-icon>
                <template #title>制品目录</template>
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

    <div class="main-content" :class="{ 'sidebar-collapsed': isCollapse }">
      <header class="header">
        <div class="header-left">
          <div class="header-breadcrumb-wrap">
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
        <div class="header-right">
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
                在控制机执行，一键安装 <strong>ai-sre</strong> CLI（同源拉取引导脚本与二进制）。请妥善保管，勿泄露到公网。
              </p>
              <el-input
                class="install-ai-sre-panel__input"
                type="textarea"
                :model-value="installAiSreCommand"
                :autosize="{ minRows: 3, maxRows: 6 }"
                readonly
              />
              <div class="install-ai-sre-panel__actions">
                <el-button type="primary" size="small" @click="copyInstallAiSreCommand">复制命令</el-button>
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

      <main class="content">
        <div class="content-inner">
          <div class="content-route">
            <router-view />
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
  Grid,
  Reading,
  Setting
} from '@element-plus/icons-vue'
import { wsService } from '../../utils/websocket'
import { copyTextToClipboard } from '../../utils/clipboard'
import { getInstallAiSreShellCurlLine } from '../../utils/installAiSre'
import { useMachineStore } from '../../stores/machine'
import { getBillingCapabilities, type BillingCapabilityFeature } from '../../api/billing'

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
  '/service/k8s/clusters': Grid,
  '/service/k8s-mirror': Download,
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

const installAiSreCommand = computed(() => getInstallAiSreShellCurlLine())

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
  const cmd = installAiSreCommand.value
  if (!cmd?.trim()) return
  try {
    await copyTextToClipboard(cmd)
    ElMessage.success('已复制安装 ai-sre 命令')
  } catch {
    ElMessage.error('复制失败，请手动全选复制')
  }
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
.main-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
  background-color: var(--layout-page-bg);
}

.sidebar {
  width: var(--layout-sidebar-width);
  background-color: var(--layout-sidebar-bg);
  color: var(--layout-sidebar-text-strong);
  overflow: hidden;
  transition: width 0.28s cubic-bezier(0.4, 0, 0.2, 1);
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  z-index: 1000;
  border-right: 0;
  box-shadow: none;
  display: flex;
  flex-direction: column;
}

.sidebar.sidebar-collapsed {
  width: var(--layout-sidebar-collapsed-width);
}

.sidebar-header {
  flex-shrink: 0;
  height: var(--layout-header-height);
  padding: 0 12px 0 14px;
  border-bottom: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  background: var(--apple-canvas, #ffffff);
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
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0;
  color: #fff;
  background: var(--apple-primary, #0066cc);
  box-shadow: none;
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
  letter-spacing: 0;
  color: var(--layout-sidebar-text-strong);
  line-height: 1.25;
  white-space: nowrap;
}

.sidebar-brand-sub {
  font-size: 11px;
  font-weight: 400;
  color: var(--layout-sidebar-text);
  letter-spacing: 0;
  opacity: 0.92;
}

.collapse-btn {
  flex-shrink: 0;
  padding: 6px;
  margin: 0 -4px 0 0;
  height: auto;
  color: var(--layout-sidebar-text);
  border-radius: 8px;
}

.collapse-btn:hover {
  color: var(--el-color-primary);
  background-color: var(--layout-sidebar-hover-bg);
}

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

/* 顶栏 */
.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  margin-left: var(--layout-sidebar-width);
  transition: margin-left 0.28s cubic-bezier(0.4, 0, 0.2, 1);
  width: calc(100% - var(--layout-sidebar-width));
}

.main-content.sidebar-collapsed {
  margin-left: var(--layout-sidebar-collapsed-width);
  width: calc(100% - var(--layout-sidebar-collapsed-width));
}

.header {
  flex-shrink: 0;
  height: var(--layout-header-height);
  background: var(--layout-header-bg);
  backdrop-filter: none;
  -webkit-backdrop-filter: none;
  border-bottom: 0 !important;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 16px 0 20px;
  position: sticky;
  top: 0;
  z-index: 900;
}

.header-left {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
}

.header-breadcrumb-wrap {
  min-width: 0;
  max-width: 100%;
  overflow-x: auto;
  overflow-y: hidden;
  -webkit-overflow-scrolling: touch;
}

.header-breadcrumb-wrap::-webkit-scrollbar {
  height: 0;
}

.header-breadcrumb {
  flex-wrap: nowrap;
  white-space: nowrap;
  padding: 2px 0;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.header-quick-codes {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 999px;
  background: transparent;
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  color: var(--layout-sidebar-text-strong);
  text-decoration: none;
  white-space: nowrap;
  transition:
    background-color 0.2s ease,
    color 0.2s ease;
}

.header-quick-codes:hover {
  background-color: var(--apple-canvas-parchment, #f5f5f7);
  color: var(--apple-primary, #0066cc);
}

.header-quick-codes.router-link-active {
  background-color: var(--apple-canvas-parchment, #f5f5f7);
  color: var(--apple-primary, #0066cc);
  font-weight: 600;
}

.install-ai-sre-trigger {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  padding: 6px 12px;
  border: 0;
  border-radius: 999px;
  background: var(--apple-canvas-parchment, #f5f5f7);
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  color: var(--apple-primary, #0066cc);
  cursor: pointer;
  transition:
    background-color 0.2s ease,
    color 0.2s ease;
}

.install-ai-sre-trigger:hover {
  background-color: var(--apple-canvas-parchment, #f5f5f7);
  color: var(--apple-primary, #0066cc);
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
  padding: 4px 6px 4px 4px;
  border-radius: 999px;
  border: 0;
  transition:
    background-color 0.2s ease,
    color 0.2s ease;
  outline: none;
}

.user-trigger:hover {
  background-color: var(--apple-canvas-parchment, #f5f5f7);
}

.user-avatar {
  flex-shrink: 0;
  font-size: 14px;
  font-weight: 600;
  color: #fff;
  background: var(--apple-primary, #0066cc) !important;
}

.user-trigger-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--layout-sidebar-text-strong);
  max-width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-trigger-chevron {
  font-size: 12px;
  color: var(--layout-sidebar-text);
  margin-right: 4px;
}

/* 内容：外层仅保留一侧留白；页面内边距由各 page-shell 负责，避免双重 padding */
.content {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  padding: 0 var(--layout-content-gutter) var(--layout-content-gutter);
  min-height: 0;
}

.custom-breadcrumb {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 2px 0;
  font-size: 13px;
  color: var(--layout-sidebar-text);
}

.breadcrumb-item {
  display: inline-flex;
  align-items: center;
  padding: 2px 4px;
  border-radius: 6px;
  transition:
    color 0.15s ease,
    background-color 0.15s ease;
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
  color: #d0d0d0;
  font-weight: 500;
}

.content-inner {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  background: transparent;
  border-radius: 0;
  border: 0;
  box-shadow: none;
  padding: 0;
}

/* 单一纵向滚动条：业务页根节点用 height:100% / page-shell--fill 时可铺满剩余视口 */
.content-route {
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  -webkit-overflow-scrolling: touch;
}

/* Element Plus 菜单：浅色侧栏 */
.sidebar-menu :deep(.el-sub-menu__title),
.sidebar-menu :deep(.el-menu-item) {
  border-radius: 10px;
  margin: 2px 0;
  font-weight: 500;
}

.sidebar-menu :deep(.el-sub-menu__title:hover),
.sidebar-menu :deep(.el-menu-item:hover) {
  background-color: var(--layout-sidebar-hover-bg) !important;
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
  color: var(--apple-primary, #0066cc);
  font-size: 11px;
  font-weight: 600;
}
</style>
