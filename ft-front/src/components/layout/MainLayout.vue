<template>
  <div class="main-layout">
    <!-- 侧边栏 -->
    <aside class="sidebar" :class="{ 'sidebar-collapsed': isCollapse }">
      <div class="sidebar-header">
        <h2 class="logo" v-show="!isCollapse">OpsFleetPilot</h2>
        <el-button
          type="text"
          class="collapse-btn"
          @click="isCollapse = !isCollapse"
        >
          <el-icon>
            <svg v-if="isCollapse" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024" width="20" height="20"><path d="M877.824 505.728l-480.64 480.64c-12.544 12.544-32.96 12.544-45.504 0-12.544-12.544-12.544-32.96 0-45.504l458.112-458.112-458.112-458.112c-12.544-12.544-12.544-32.96 0-45.504 12.544-12.544 32.96-12.544 45.504 0l480.64 480.64c12.544 12.544 12.544 32.96 0 45.504z" fill="currentColor"/></svg>
            <svg v-else xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024" width="20" height="20"><path d="M867.84 512c0 12.544-10.048 22.528-22.528 22.528h-616.96l294.4 294.4c12.544 12.544 12.544 32.96 0 45.504-12.544 12.544-32.96 12.544-45.504 0l-360.96-360.96c-12.544-12.544-12.544-32.96 0-45.504l360.96-360.96c12.544-12.544 32.96-12.544 45.504 0 12.544 12.544 12.544 32.96 0 45.504l-294.4 294.4h616.96c12.544 0 22.528 10.048 22.528 22.528z" fill="currentColor"/></svg>
          </el-icon>
        </el-button>
      </div>
      <el-menu
        :key="menuRemountKey"
        :default-active="activeMenu"
        :default-openeds="menuDefaultOpeneds"
        class="el-menu-vertical-demo"
        @select="handleMenuSelect"
        background-color="#001529"
        text-color="#fff"
        active-text-color="#ffffff"
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
      </el-menu>
    </aside>

    <!-- 右侧内容区 -->
    <div class="main-content" :class="{ 'sidebar-collapsed': isCollapse }">
      <!-- 头部 -->
      <header class="header">
        <div class="header-brand">
          <span class="header-brand-title">OpsFleetPilot</span>
          <span class="header-brand-sub">运维控制台</span>
        </div>

        <div class="header-right">
          <!-- 用户信息 -->
          <el-dropdown>
            <span class="user-info">
              <el-icon><User /></el-icon>
              {{ currentUser.username }}
              <el-icon class="el-icon--right"><ArrowDown /></el-icon>
            </span>
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

      <!-- 内容区 -->
      <main class="content">
        <!-- 面包屑导航 -->
        <div class="breadcrumb-container">
          <el-breadcrumb separator="/" class="custom-breadcrumb">
            <el-breadcrumb-item :to="{ path: '/' }" class="breadcrumb-item">
              <el-icon class="breadcrumb-icon"><House /></el-icon>
              <span>OpsFleetPilot</span>
            </el-breadcrumb-item>
            <el-breadcrumb-item
              v-for="(routeItem, index) in breadcrumbItems"
              :key="index"
              :to="{ path: routeItem.path }"
              class="breadcrumb-item"
              :class="{ 'last-item': index === breadcrumbItems.length - 1 }"
            >
              <el-icon v-if="getRouteIcon(routeItem.path)" class="breadcrumb-icon">
                <component :is="getRouteIcon(routeItem.path)" />
              </el-icon>
              <span>{{ routeItem.meta.title }}</span>
            </el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <router-view />
      </main>
    </div>


  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  User,
  SwitchButton,
  ArrowDown,
  PieChart,
  Monitor,
  House,
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
import { ElMessage } from 'element-plus'
import { wsService } from '../../utils/websocket'
import { useMachineStore } from '../../stores/machine'

// 路由路径到图标的映射
const routeIconMap: Record<string, any> = {
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
  '/init-tools': Tools
}

const route = useRoute()
const router = useRouter()

// 侧边栏折叠状态
const isCollapse = ref(false)

// 子菜单随路由展开（与分组 index 一致）
const menuDefaultOpeneds = computed(() => {
  const p = route.path
  const open: string[] = []
  if (p.startsWith('/service')) open.push('service-delivery')
  if (p.startsWith('/monitoring')) open.push('/monitoring')
  if (p.startsWith('/security-audit')) open.push('/security-audit')
  if (p.startsWith('/advanced')) open.push('/advanced')
  return open
})

/** 仅当「应展开的子菜单集合」变化时 remount，以应用 default-openeds 且减少闪烁 */
const menuRemountKey = computed(() => menuDefaultOpeneds.value.join('|'))

// 获取当前用户信息
const currentUser = computed(() => {
  // 从localStorage获取用户信息（简化处理）
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

// ---- WebSocket Real-Time Connection ----
const machineStore = useMachineStore()
const handleMachineHeartbeatMessage = (msg: any) => {
  machineStore.handleMachineHeartbeat(msg.data)
}
const handleMachineStatusMessage = (msg: any) => {
  machineStore.handleMachineStatusUpdate(msg.data || [])
}

onMounted(() => {
  // Connect WebSocket using current user ID (or fallback)
  const userId = currentUser.value?.id || 'anonymous'
  wsService.connect(String(userId))

  // Register handler for machine heartbeat events
  wsService.on('machine_heartbeat', handleMachineHeartbeatMessage)
  wsService.on('machine_status_update', handleMachineStatusMessage)
})

onUnmounted(() => {
  wsService.off('machine_heartbeat', handleMachineHeartbeatMessage)
  wsService.off('machine_status_update', handleMachineStatusMessage)
  wsService.disconnect()
})

// 处理用户管理
const handleUserManagement = () => {
  router.push('/user/list')
}

// 计算当前激活的菜单
const activeMenu = computed(() => {
  return route.path
})

// 获取路由对应的图标组件
const getRouteIcon = (path: string): any => {
  // 精确匹配
  if (routeIconMap[path]) {
    return routeIconMap[path]
  }
  
  // 匹配父路径
  const parentPath = path.substring(0, path.lastIndexOf('/'))
  if (parentPath && routeIconMap[parentPath]) {
    return routeIconMap[parentPath]
  }
  
  // 默认不显示图标
  return null
}

// 计算面包屑项
const breadcrumbItems = computed(() => {
  // 定义面包屑项的类型
  interface BreadcrumbItem {
    path: string
    meta: { title: string; [key: string]: any }
  }
  
  // 获取当前路由的所有匹配项，过滤掉Login页面的路由
  const matchedRoutes = route.matched.filter(routeItem => 
    routeItem.name !== 'Login' && routeItem.meta && typeof routeItem.meta.title === 'string'
  )
  
  const breadcrumbs: BreadcrumbItem[] = []

  for (let i = 0; i < matchedRoutes.length; i++) {
    const routeItem = matchedRoutes[i]
    
    // 确保routeItem存在
    if (!routeItem) continue
    
    // 如果是根路径，跳过
    if (routeItem.path === '/') {
      continue
    }
    
    // 构建正确的路径
    let currentPath = ''
    if (routeItem.path.startsWith('/')) {
      currentPath = routeItem.path
    } else {
      // 这是一个子路由，使用完整路径
      currentPath = route.path
    }
    
    // 确保路由有标题
    if (routeItem.meta && typeof routeItem.meta.title === 'string') {
      // 如果是当前选中的路由，显示它
      if (i === matchedRoutes.length - 1) {
        // 检查当前标题是否与前一个标题重复
        const lastBreadcrumb = breadcrumbs[breadcrumbs.length - 1]
        if (breadcrumbs.length === 0 || routeItem.meta.title !== lastBreadcrumb?.meta.title) {
          breadcrumbs.push({
            path: currentPath,
            meta: { title: routeItem.meta.title }
          })
        }
      }
      // 如果是一级菜单且面包屑还没有任何项，显示它
      else if (i === 0 && breadcrumbs.length === 0) {
        breadcrumbs.push({
          path: currentPath,
          meta: { title: routeItem.meta.title }
        })
      }
    }
  }

  return breadcrumbs
})

// 处理菜单选择
const handleMenuSelect = (index: string) => {
  router.push(index)
}

// 处理退出登录
const handleLogout = () => {
  // 清除localStorage中的token和用户信息
  localStorage.removeItem('token')
  localStorage.removeItem('userInfo')
  router.push('/login')
}
</script>

<style scoped>
/* 主布局容器 */
.main-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
  background-color: #f9fafb;
}

/* 侧边栏 */
.sidebar {
  width: var(--layout-sidebar-width);
  background-color: #001529;
  color: #fff;
  overflow-y: auto;
  transition: width 0.3s ease-in-out;
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  z-index: 1000;
  box-shadow: 2px 0 8px rgba(0, 0, 0, 0.1);
}

.sidebar.sidebar-collapsed {
  width: var(--layout-sidebar-collapsed-width);
}

/* 侧边栏头部 */
.sidebar-header {
  height: var(--layout-header-height);
  padding: 0 20px;
  border-bottom: 1px solid #1f2d3d;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background-color: #001529;
  position: sticky;
  top: 0;
  z-index: 1001;
}

.logo {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  letter-spacing: -0.02em;
  color: #fff;
  white-space: nowrap;
  transition: opacity 0.3s ease;
}

.collapse-btn {
  color: #fff;
  padding: 8px;
  margin-left: 10px;
  transition: background-color 0.3s;
}

.collapse-btn:hover {
  background-color: rgba(255, 255, 255, 0.1);
}

/* 主内容区 */
.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  margin-left: var(--layout-sidebar-width);
  transition: margin-left 0.3s ease-in-out;
  width: calc(100% - var(--layout-sidebar-width));
}

.main-content.sidebar-collapsed {
  margin-left: var(--layout-sidebar-collapsed-width);
  width: calc(100% - var(--layout-sidebar-collapsed-width));
}

/* 顶部导航栏 */
.header {
  height: var(--layout-header-height);
  background-color: #fff;
  border-bottom: 1px solid #e5e7eb;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  gap: 16px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
  position: sticky;
  top: 0;
  z-index: 900;
}

.header-brand {
  display: flex;
  align-items: baseline;
  gap: 10px;
  min-width: 0;
  flex-shrink: 1;
}

.header-brand-title {
  font-size: 16px;
  font-weight: 600;
  color: #111827;
  letter-spacing: -0.02em;
}

.header-brand-sub {
  font-size: 13px;
  color: #6b7280;
  font-weight: 400;
}

/* 头部右侧 */
.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

/* 用户信息 */
.user-info {
  display: flex;
  align-items: center;
  cursor: pointer;
  padding: 8px 12px;
  border-radius: 8px;
  transition: background-color 0.3s;
}

.user-info:hover {
  background-color: #f3f4f6;
}

.user-info .el-icon {
  margin-right: 8px;
  font-size: 18px;
}

/* 内容区域 */
.content-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* 面包屑 */
.breadcrumb-container {
  margin-bottom: 16px;
  padding: 4px 0;
  border-bottom: 1px solid #e5e7eb;
  height: auto;
  line-height: 1.5;
}

.custom-breadcrumb {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  font-size: 14px;
  color: #666;
}

.breadcrumb-item {
  display: flex;
  align-items: center;
  cursor: pointer;
  padding: 2px 0;
  border-radius: 4px;
  transition: all 0.3s ease;
  animation: breadcrumbSlideIn 0.3s ease forwards;
  opacity: 0;
  transform: translateX(-10px);
}

.breadcrumb-item:nth-child(1) {
  animation-delay: 0.05s;
}

.breadcrumb-item:nth-child(2) {
  animation-delay: 0.1s;
}

.breadcrumb-item:nth-child(3) {
  animation-delay: 0.15s;
}

.breadcrumb-item:nth-child(4) {
  animation-delay: 0.2s;
}

.breadcrumb-item:nth-child(5) {
  animation-delay: 0.25s;
}

.breadcrumb-item:nth-child(6) {
  animation-delay: 0.3s;
}

.breadcrumb-item:hover {
  background-color: rgba(30, 64, 175, 0.05);
  color: #1E40AF;
  transform: translateY(-1px);
}

.breadcrumb-item.last-item {
  font-weight: 500;
  color: #1E40AF;
}

.breadcrumb-icon {
  margin-right: 3px;
  margin-left: 2px;
  font-size: 14px;
  color: #909399;
  transition: all 0.3s ease;
}

.breadcrumb-item:hover .breadcrumb-icon {
  color: #1E40AF;
  transform: scale(1.1);
}

/* 优化分隔符样式 */
.el-breadcrumb__separator {
  margin: 0;
  padding: 0 4px;
  color: #909399;
  font-size: 12px;
  transition: color 0.3s ease;
}

.breadcrumb-item:hover + .el-breadcrumb__separator {
  color: #1E40AF;
}

/* 动画关键帧 */
@keyframes breadcrumbSlideIn {
  from {
    opacity: 0;
    transform: translateX(-10px) scale(0.95);
  }
  to {
    opacity: 1;
    transform: translateX(0) scale(1);
  }
}

@keyframes breadcrumbLastItemIn {
  from {
    opacity: 0;
    transform: translateX(-10px) scale(0.9);
  }
  to {
    opacity: 1;
    transform: translateX(0) scale(1);
  }
}

/* 激活状态动画 */
.breadcrumb-item.last-item {
  animation: breadcrumbLastItemIn 0.4s cubic-bezier(0.4, 0, 0.2, 1) forwards;
}

/* 主内容 */
.content {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: var(--layout-content-padding);
  background-color: #ffffff;
  border-radius: 4px;
  margin: 8px 0;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
  width: 100%;
  display: flex;
  flex-direction: column;
}

/* 优化面包屑样式 */
.el-breadcrumb {
  font-size: 14px;
  color: #6b7280;
}

.el-breadcrumb__item:last-child {
  color: #1E40AF;
  font-weight: 500;
}

/* 优化子菜单样式 */
.el-menu-vertical-demo .el-sub-menu .el-menu {
  background-color: #002a4f !important;
  padding: 4px 0;
}

.el-menu-vertical-demo .el-sub-menu .el-menu-item {
  font-size: 13px;
  height: 36px;
  line-height: 36px;
  padding: 0 20px 0 50px !important;
  margin: 2px 8px;
  border-radius: 4px;
}

.el-menu-vertical-demo .el-sub-menu .el-menu-item:hover {
  background-color: rgba(30, 64, 175, 0.2) !important;
}

.el-menu-vertical-demo .el-sub-menu .el-menu-item.is-active {
  background-color: rgba(30, 64, 175, 0.3) !important;
  color: #409eff;
}

/* 优化子菜单标题样式 */
.el-menu-vertical-demo .el-sub-menu__title {
  height: 48px;
  line-height: 48px;
  font-weight: 500;
}
/* 子菜单样式调整 */
:deep(.el-sub-menu__title) {
  background-color: #001529 !important;
}

:deep(.el-menu--popup) {
  background-color: #0a1930 !important;
}

:deep(.el-menu-item) {
  background-color: #0a1930 !important;
}

:deep(.el-menu-item:hover) {
  background-color: #1a365d !important;
}

:deep(.el-menu-item.is-active) {
  background-color: #1E40AF !important;
  color: #ffffff !important;
  font-weight: 500;
}

:deep(.el-sub-menu .el-menu-item) {
  background-color: #0a1930 !important;
}
</style>
