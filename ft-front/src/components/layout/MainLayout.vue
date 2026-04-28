<template>
  <div class="main-layout">
    <aside class="sidebar" :class="{ 'sidebar-collapsed': isCollapse }">
      <div class="sidebar-header">
        <div class="sidebar-brand">
          <div class="sidebar-brand-mark" aria-hidden="true">{{ brandShort }}</div>
          <div v-show="!isCollapse" class="sidebar-brand-text">
            <span class="sidebar-brand-title">OpsFleetPilot</span>
            <span class="sidebar-brand-sub">运维控制台</span>
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
          <el-menu-item index="/dashboard">
            <el-icon><PieChart /></el-icon>
            <template #title>仪表盘</template>
          </el-menu-item>
          <el-sub-menu index="service-delivery">
            <template #title>
              <el-icon><Box /></el-icon>
              <span>服务与交付</span>
            </template>
            <el-menu-item index="/service/deploy">
              <el-icon><Operation /></el-icon>
              <template #title>服务部署</template>
            </el-menu-item>
            <el-menu-item index="/service/k8s-deploy">
              <el-icon><Connection /></el-icon>
              <template #title>Kubernetes 部署</template>
            </el-menu-item>
            <el-menu-item index="/service/k8s-mirror">
              <el-icon><Download /></el-icon>
              <template #title>K8s 制品镜像</template>
            </el-menu-item>
            <el-menu-item index="/service/linux">
              <el-icon><Cpu /></el-icon>
              <template #title>Linux 服务管理</template>
            </el-menu-item>
          </el-sub-menu>
          <el-menu-item index="/proxy/config">
            <el-icon><Link /></el-icon>
            <template #title>代理配置</template>
          </el-menu-item>
          <el-sub-menu index="/monitoring">
            <template #title>
              <el-icon><Monitor /></el-icon>
              <span>监控告警</span>
            </template>
            <el-menu-item index="/monitoring/prometheus">Prometheus</el-menu-item>
            <el-menu-item index="/monitoring/node-exporter">Node Exporter</el-menu-item>
            <el-menu-item index="/monitoring/jmx-exporter">JMX Exporter</el-menu-item>
            <el-menu-item index="/monitoring/redis-exporter">Redis Exporter</el-menu-item>
            <el-menu-item index="/monitoring/mongodb-exporter">MongoDB Exporter</el-menu-item>
            <el-menu-item index="/monitoring/blackbox-exporter">Blackbox Exporter</el-menu-item>
          </el-sub-menu>
          <el-menu-item index="/job/center">
            <el-icon><Management /></el-icon>
            <template #title>作业中心</template>
          </el-menu-item>
          <el-sub-menu index="/security-audit">
            <template #title>
              <el-icon><Lock /></el-icon>
              <span>安全与审计</span>
            </template>
            <el-menu-item index="/security-audit/operation-logs">操作日志</el-menu-item>
            <el-menu-item index="/security-audit/permission-management">权限管理</el-menu-item>
          </el-sub-menu>
          <el-sub-menu index="/advanced">
            <template #title>
              <el-icon><DocumentCopy /></el-icon>
              <span>高级功能</span>
            </template>
            <el-menu-item index="/advanced/backup-restore">备份与恢复</el-menu-item>
            <el-menu-item index="/advanced/performance-analysis">性能分析</el-menu-item>
          </el-sub-menu>
          <el-menu-item index="/init-tools">
            <el-icon><Tools /></el-icon>
            <template #title>初始化工具</template>
          </el-menu-item>
          <el-menu-item index="/execution-records">
            <el-icon><DocumentCopy /></el-icon>
            <template #title>执行记录</template>
          </el-menu-item>
        </el-menu>
      </el-scrollbar>
    </aside>

    <div class="main-content" :class="{ 'sidebar-collapsed': isCollapse }">
      <header class="header">
        <div class="header-spacer" aria-hidden="true" />
        <div class="header-right">
          <el-dropdown trigger="click" popper-class="layout-user-dropdown">
            <div class="user-trigger" role="button" tabindex="0">
              <el-avatar :size="34" class="user-avatar">{{ userInitial }}</el-avatar>
              <span class="user-trigger-name">{{ currentUser.username }}</span>
              <el-icon class="user-trigger-chevron"><ArrowDown /></el-icon>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="handleUserManagement">
                  <el-icon><User /></el-icon>
                  用户管理
                </el-dropdown-item>
                <el-dropdown-item divided @click="handleLogout">
                  <el-icon><SwitchButton /></el-icon>
                  退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </header>

      <main class="content">
        <div class="breadcrumb-container">
          <el-breadcrumb separator="/" class="custom-breadcrumb">
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
        <div class="content-inner">
          <router-view />
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
  Download
} from '@element-plus/icons-vue'
import { wsService } from '../../utils/websocket'
import { useMachineStore } from '../../stores/machine'

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
  '/service/k8s-mirror': Download,
  '/service/linux': Cpu,
  '/proxy': Link,
  '/monitoring': Monitor,
  '/job': Management,
  '/security-audit': Lock,
  '/advanced': DocumentCopy,
  '/init-tools': Tools,
  '/execution-records': DocumentCopy
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
  '/execution-records': '/execution-records'
}

const route = useRoute()
const router = useRouter()

const isCollapse = ref(false)

const menuTextColor = 'var(--layout-sidebar-text)'
const menuActiveColor = 'var(--el-color-primary)'

const menuDefaultOpeneds = computed(() => {
  const p = route.path
  const open: string[] = []
  if (p.startsWith('/service')) open.push('service-delivery')
  if (p.startsWith('/monitoring')) open.push('/monitoring')
  if (p.startsWith('/security-audit')) open.push('/security-audit')
  if (p.startsWith('/advanced')) open.push('/advanced')
  return open
})

const menuRemountKey = computed(() => menuDefaultOpeneds.value.join('|'))

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

const userInitial = computed(() => {
  const name = String(currentUser.value?.username ?? '?').trim()
  const ch = name.slice(0, 1)
  return ch || '?'
})

const brandShort = computed(() => 'OP')

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

  wsService.on('machine_heartbeat', handleMachineHeartbeatMessage)
  wsService.on('machine_status_update', handleMachineStatusMessage)
})

onUnmounted(() => {
  wsService.off('machine_heartbeat', handleMachineHeartbeatMessage)
  wsService.off('machine_status_update', handleMachineStatusMessage)
  wsService.disconnect()
})

const handleUserManagement = () => {
  router.push('/user/list')
}

const activeMenu = computed(() => {
  return route.path
})

const getRouteIcon = (path?: string): Component | undefined => {
  if (!path) return undefined
  if (routeIconMap[path]) {
    return routeIconMap[path]
  }

  const parentPath = path.substring(0, path.lastIndexOf('/'))
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
  const path = routeItem.path
  if (!path || path === '/' || path.includes(':')) return undefined
  return sectionDefaultPath[path] ?? path
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
  border-right: 1px solid var(--layout-sidebar-border);
  box-shadow: var(--layout-shadow-soft);
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
  border-bottom: 1px solid var(--layout-sidebar-border);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  background: linear-gradient(180deg, #ffffff 0%, var(--layout-sidebar-bg) 100%);
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
  letter-spacing: 0.02em;
  color: #fff;
  background: linear-gradient(135deg, var(--el-color-primary) 0%, var(--el-color-primary-light-3) 100%);
  box-shadow: 0 2px 8px rgba(30, 64, 175, 0.22);
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
  letter-spacing: -0.02em;
  color: var(--layout-sidebar-text-strong);
  line-height: 1.25;
  white-space: nowrap;
}

.sidebar-brand-sub {
  font-size: 11px;
  font-weight: 400;
  color: var(--layout-sidebar-text);
  letter-spacing: 0.02em;
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
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-bottom: 1px solid var(--layout-header-border);
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding: 0 20px 0 24px;
  position: sticky;
  top: 0;
  z-index: 900;
}

.header-spacer {
  flex: 1;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.user-trigger {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  padding: 4px 6px 4px 4px;
  border-radius: 999px;
  border: 1px solid transparent;
  transition:
    background-color 0.2s ease,
    border-color 0.2s ease,
    box-shadow 0.2s ease;
  outline: none;
}

.user-trigger:hover {
  background-color: rgba(255, 255, 255, 0.9);
  border-color: var(--layout-sidebar-border);
  box-shadow: var(--layout-shadow-soft);
}

.user-avatar {
  flex-shrink: 0;
  font-size: 14px;
  font-weight: 600;
  color: #fff;
  background: linear-gradient(145deg, var(--el-color-primary-light-3), var(--el-color-primary)) !important;
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

/* 内容 */
.content {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  padding: 0 var(--layout-content-padding) var(--layout-content-padding);
  min-height: 0;
}

.breadcrumb-container {
  flex-shrink: 0;
  padding: 14px 4px 12px;
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
  color: #cbd5e1;
  font-weight: 500;
}

.content-inner {
  flex: 1;
  min-height: 0;
  overflow: auto;
  overflow-x: hidden;
  background: var(--layout-content-surface);
  border-radius: 12px;
  border: 1px solid var(--layout-sidebar-border);
  box-shadow: var(--layout-shadow-soft);
  padding: var(--layout-content-padding);
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
</style>
