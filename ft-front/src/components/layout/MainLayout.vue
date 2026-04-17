<template>
  <div class="main-layout">
    <!-- 侧边栏 -->
    <aside class="sidebar" :class="{ 'sidebar-collapsed': isCollapse }">
      <div class="sidebar-header">
        <h2 class="logo" v-show="!isCollapse">FleetPilot</h2>
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
        :default-active="activeMenu"
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
        <el-menu-item index="/service/deploy">
          <el-icon><Setting /></el-icon>
          <template #title>服务管理</template>
        </el-menu-item>
        <el-menu-item index="/service/k8s-deploy">
          <el-icon><Setting /></el-icon>
          <template #title>Kubernetes部署</template>
        </el-menu-item>
        <el-menu-item index="/service/linux">
          <el-icon><Setting /></el-icon>
          <template #title>Linux服务管理</template>
        </el-menu-item>
        <el-menu-item index="/proxy/config">
          <el-icon><DataAnalysis /></el-icon>
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
        <el-sub-menu index="/init-tools">
          <template #title>
            <el-icon><Tools /></el-icon>
            <span>初始化工具</span>
          </template>
          <el-menu-item index="/init-tools/system-param">系统参数优化</el-menu-item>
          <el-menu-item index="/init-tools/time-sync">时间同步</el-menu-item>
          <el-menu-item index="/init-tools/security-hardening">系统安全加固</el-menu-item>
          <el-menu-item index="/init-tools/disk-partition">磁盘分区优化</el-menu-item>
        </el-sub-menu>
      </el-menu>
    </aside>

    <!-- 右侧内容区 -->
    <div class="main-content" :class="{ 'sidebar-collapsed': isCollapse }">
      <!-- 头部 -->
      <header class="header">
        <div class="header-left">
          <el-icon class="hamburger" @click="isCollapse = !isCollapse">
            <svg v-if="isCollapse" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024" width="24" height="24"><path d="M877.824 505.728l-480.64 480.64c-12.544 12.544-32.96 12.544-45.504 0-12.544-12.544-12.544-32.96 0-45.504l458.112-458.112-458.112-458.112c-12.544-12.544-12.544-32.96 0-45.504 12.544-12.544 32.96-12.544 45.504 0l480.64 480.64c12.544 12.544 12.544 32.96 0 45.504z" fill="currentColor"/></svg>
            <svg v-else xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024" width="24" height="24"><path d="M867.84 512c0 12.544-10.048 22.528-22.528 22.528h-616.96l294.4 294.4c12.544 12.544 12.544 32.96 0 45.504-12.544 12.544-32.96 12.544-45.504 0l-360.96-360.96c-12.544-12.544-12.544-32.96 0-45.504l360.96-360.96c12.544-12.544 32.96-12.544 45.504 0 12.544 12.544 12.544 32.96 0 45.504l-294.4 294.4h616.96c12.544 0 22.528 10.048 22.528 22.528z" fill="currentColor"/></svg>
          </el-icon>
        </div>

        <!-- 全局搜索 -->
        <div class="search-container">
          <el-input
            v-model="searchText"
            placeholder="全局搜索"
            clearable
            class="search-input"
            @keyup.enter="handleSearch"
          >
            <template #prefix>
              <el-icon color="#9ca3af">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024" width="18" height="18"><path d="M909.6 854.5L649.9 594.8C690.2 542.7 712 479 712 412c0-80.2-31.3-155.4-87.9-212.1-56.6-56.7-132-87.9-212.1-87.9s-155.5 31.3-212.1 87.9C143.2 256.5 112 331.8 112 412c0 80.3 31.3 155.6 87.9 212.2 56.6 56.6 132 87.8 212.1 87.8 66.9 0 130.6-21.8 182.7-62l259.7 259.6a8.2 8.2 0 0011.6 0l43.6-43.5a8.2 8.2 0 000-11.6zM504 768C364.9 768 256 659.1 256 520s108.9-248 248-248 248 108.9 248 248-108.9 248-248 248z" fill="currentColor"/></svg>
              </el-icon>
            </template>
          </el-input>
        </div>

        <div class="header-right">
          <!-- 客户端下载 -->
          <div class="client-download-container">
            <div class="client-download-progress" v-if="isDownloading || isDownloadCompleted">
              <!-- 上边框进度 -->
              <div class="border-progress border-top" :style="{ width: topBorderWidth + '%' }"></div>
              <!-- 右边框进度 -->
              <div class="border-progress border-right" :style="{ height: rightBorderHeight + '%' }"></div>
              <!-- 下边框进度 -->
              <div class="border-progress border-bottom" :style="{ width: bottomBorderWidth + '%' }"></div>
              <!-- 左边框进度 -->
              <div class="border-progress border-left" :style="{ height: leftBorderHeight + '%' }"></div>
            </div>
            <div class="client-download" @click="handleClientDownload" :class="{ 'downloading': isDownloading }" :disabled="isDownloading">
              <div class="button-content">
                <el-icon class="download-icon" :class="{ 'hidden': isDownloading }">
                  <Download />
                </el-icon>
                <div class="progress-text" v-if="isDownloading">{{ downloadProgress }}%</div>
              </div>
            </div>
          </div>

          <!-- 通知图标 -->
          <div class="notification">
            <el-badge :value="notificationCount" :max="99">
              <el-icon @click="showNotification">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024" width="24" height="24"><path d="M512 64c247.4 0 448 200.6 448 448 0 123.1-59.2 232.8-153.5 303.9 26.4 60.5 9.4 131.7-46.4 171.4-28.9 20.2-62.1 31.7-96.6 31.7H260.9c-34.5 0-67.7-11.5-96.6-31.7-55.8-39.7-72.8-110.9-46.4-171.4C125.2 744.8 66 635.1 66 512c0-247.4 200.6-448 446-448zm0 824c23.2 0 45.7-4.3 66.5-12.1 38.7-13.9 63.8-50.9 63.8-92.8H381.7c0 41.9 25.1 78.9 63.8 92.8 20.8 7.8 43.3 12.1 66.5 12.1zM512 128c-33.1 0-60 26.9-60 60v264c0 33.1 26.9 60 60 60s60-26.9 60-60V188c0-33.1-26.9-60-60-60z" fill="currentColor"/></svg>
              </el-icon>
            </el-badge>
          </div>

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
              <span>FleetPilot</span>
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
import { Grid, User, Setting, SwitchButton, ArrowDown, PieChart, DataAnalysis, Monitor, House, Management, Tools, Lock, DocumentCopy, Download } from '@element-plus/icons-vue'
import { ElMessageBox, ElMessage } from 'element-plus'
import { wsService } from '../../utils/websocket'
import { useMachineStore } from '../../stores/machine'

// 路由路径到图标的映射
const routeIconMap: Record<string, any> = {
  '/dashboard': PieChart,
  '/machine': Grid,
  '/service': Setting,
  '/service/deploy': Setting,
  '/service/k8s-deploy': Setting,
  '/proxy': DataAnalysis,
  '/monitoring': Monitor,
  '/job': Management,
  '/security-audit': Lock,
  '/advanced': DocumentCopy,
  '/init-tools': Tools,
  '/init-tools/system-param': Tools,
  '/init-tools/time-sync': Tools,
  '/init-tools/security-hardening': Tools,
  '/init-tools/disk-partition': Tools
}

const route = useRoute()
const router = useRouter()

// 侧边栏折叠状态
const isCollapse = ref(false)

// 搜索文本
const searchText = ref('')

// 通知数量
const notificationCount = ref(3)

// 下载相关状态
const isDownloading = ref(false)
const downloadProgress = ref(0)
const isDownloadCompleted = ref(false)

// 计算矩形边框的进度值
const topBorderWidth = computed(() => {
  // 进度 0-25%: 上边框宽度从0%到100%
  if (downloadProgress.value <= 25) {
    return (downloadProgress.value / 25) * 100
  }
  // 进度 >25%: 上边框保持100%
  return 100
})

const rightBorderHeight = computed(() => {
  // 进度 0-25%: 右边框高度为0%
  if (downloadProgress.value <= 25) {
    return 0
  }
  // 进度 25-50%: 右边框高度从0%到100%
  if (downloadProgress.value <= 50) {
    return ((downloadProgress.value - 25) / 25) * 100
  }
  // 进度 >50%: 右边框保持100%
  return 100
})

const bottomBorderWidth = computed(() => {
  // 进度 0-50%: 下边框宽度为0%
  if (downloadProgress.value <= 50) {
    return 0
  }
  // 进度 50-75%: 下边框宽度从0%到100%
  if (downloadProgress.value <= 75) {
    return ((downloadProgress.value - 50) / 25) * 100
  }
  // 进度 >75%: 下边框保持100%
  return 100
})

const leftBorderHeight = computed(() => {
  // 进度 0-75%: 左边框高度为0%
  if (downloadProgress.value <= 75) {
    return 0
  }
  // 进度 75-100%: 左边框高度从0%到100%
  return ((downloadProgress.value - 75) / 25) * 100
})

// 处理搜索
const handleSearch = () => {
  if (searchText.value.trim()) {
    // 这里可以实现全局搜索逻辑
    console.log('搜索:', searchText.value)
    // 例如：跳转到搜索结果页面或触发搜索API
  }
}

// 显示通知
const showNotification = () => {
  // 这里可以实现通知面板的显示逻辑
  console.log('显示通知列表')
  // 例如：打开通知抽屉或弹窗
}

// 复制下载命令
// @ts-ignore - 函数在模板中使用但TypeScript未检测到
const copyCommand = (command: string) => {
  navigator.clipboard.writeText(command).then(() => {
    ElMessage.success('命令已复制到剪贴板！')
  }).catch(err => {
    console.error('复制失败:', err)
    ElMessage.error('复制失败，请手动复制')
  })
}

// 客户端下载
const handleClientDownload = () => {
  // 如果正在下载中，不允许再次点击
  if (isDownloading.value) {
    return
  }

  // 定义可复制的命令
  const downloadCommand = 'curl https://example.com/xxx.xxx'

  // 自定义对话框内容
  const message = `
    是否下载客户端？
    <div class="download-command-container">
      <span class="download-command" @click="copyCommand(downloadCommand)">
        ${downloadCommand}
      </span>
      <span class="copy-hint">点击复制</span>
    </div>
  `

  ElMessageBox.confirm(message, '下载确认', {
    confirmButtonText: '下载',
    cancelButtonText: '取消',
    type: 'info',
    dangerouslyUseHTMLString: true,
  }).then(() => {
    // 开始模拟下载
    isDownloading.value = true
    downloadProgress.value = 0
    
    ElMessage.success('开始下载客户端...')
    
    // 模拟下载进度
        const downloadInterval = setInterval(() => {
          downloadProgress.value += 2
          
          // 下载完成
          if (downloadProgress.value >= 100) {
            clearInterval(downloadInterval)
            downloadProgress.value = 100
            
            // 下载完成后延迟重置状态
            setTimeout(() => {
              isDownloading.value = false
              isDownloadCompleted.value = true
              ElMessage.success('客户端下载完成！')
            }, 1000)
          }
        }, 100)
    
    // 模拟实际下载（示例）
    // window.location.href = '/api/download/client'
  }).catch(() => {
    // 用户取消下载
    console.log('用户取消下载')
  })
}

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
  font-size: 18px;
  font-weight: 600;
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
  padding: 0 16px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
  position: sticky;
  top: 0;
  z-index: 900;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.hamburger {
  cursor: pointer;
  padding: 8px;
  border-radius: 4px;
  transition: background-color 0.3s;
}

.hamburger:hover {
  background-color: #f3f4f6;
}

/* 搜索容器 */
.search-container {
  flex: 1;
  max-width: 400px;
  margin: 0 24px;
}

.search-input {
  width: 100%;
  border-radius: 8px;
  border: 1px solid #e5e7eb;
  transition: border-color 0.3s;
}

.search-input:focus-within {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 3px rgba(30, 64, 175, 0.1);
}

/* 头部右侧 */
.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

/* 客户端下载 */
.client-download-container {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 52px;
  width: 70px;
  overflow: visible;
  margin: 0;
}

.client-download-progress {
  position: absolute;
  top: -1px;
  left: -1px;
  right: -1px;
  bottom: -1px;
  border-radius: 8px;
  z-index: 3;
  pointer-events: none;
  overflow: visible;
}

.border-progress {
  position: absolute;
  background-color: #67c23a;
  transition: all 0.3s ease-in-out;
  z-index: 3;
}

/* 上边框 */
.border-top {
  top: 0;
  left: 0;
  height: 2px;
  border-top-left-radius: 8px;
  border-top-right-radius: 8px;
  z-index: 4;
}

/* 右边框 */
.border-right {
  top: 0;
  right: 0;
  width: 2px;
  border-top-right-radius: 8px;
  border-bottom-right-radius: 8px;
  z-index: 4;
}

/* 下边框 */
.border-bottom {
  bottom: 0;
  right: 0;
  height: 2px;
  border-bottom-left-radius: 8px;
  border-bottom-right-radius: 8px;
  z-index: 4;
}

/* 左边框 */
.border-left {
  bottom: 0;
  left: 0;
  width: 2px;
  border-top-left-radius: 8px;
  border-bottom-left-radius: 8px;
  z-index: 4;
}

.client-download {
  position: relative;
  cursor: pointer;
  padding: 10px;
  border-radius: 8px;
  transition: all 0.3s;
  background-color: #fff;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  user-select: none;
  border: none;
  outline: none;
  height: 52px;
  width: 70px;
}

.client-download:hover:not(.downloading) {
  background-color: #f3f4f6;
  transform: scale(1.05);
}

.client-download.downloading {
  cursor: not-allowed;
  background-color: #f5f7fa;
}

.button-content {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  transition: all 0.3s ease;
  position: relative;
  z-index: 1;
}

.download-icon {
  font-size: 24px;
  color: #67c23a;
  transition: all 0.5s ease-in-out;
  opacity: 1;
  transform: scale(1);
}

.download-icon.hidden {
  opacity: 0;
  transform: scale(0.8);
}

.progress-text {
  font-size: 16px;
  font-weight: 600;
  color: #67c23a;
  transition: all 0.5s ease-in-out;
  opacity: 0;
  transform: scale(0.8);
  position: relative;
  z-index: 1;
  width: auto;
  text-align: center;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  margin: 0;
}

.client-download.downloading .progress-text {
  opacity: 1;
  transform: scale(1);
  animation: pulse 1s infinite;
}

@keyframes pulse {
  0% {
    transform: scale(1);
  }
  50% {
    transform: scale(1.05);
  }
  100% {
    transform: scale(1);
  }
}

/* 下载命令样式 */
.download-command-container {
  margin-top: 12px;
  display: flex;
  align-items: center;
  gap: 8px;
  background-color: #f5f7fa;
  padding: 10px;
  border-radius: 4px;
  border: 1px solid #ebeef5;
}

.download-command {
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  color: #67c23a;
  font-size: 14px;
  cursor: pointer;
  padding: 2px 4px;
  border-radius: 2px;
  transition: all 0.3s;
}

.download-command:hover {
  background-color: rgba(103, 194, 58, 0.1);
  color: #85ce61;
  text-decoration: underline;
}

.copy-hint {
  font-size: 12px;
  color: #909399;
}

.client-download:hover:not(.downloading) .download-icon {
  color: #409eff;
}

/* 通知 */
.notification {
  position: relative;
  cursor: pointer;
  padding: 8px;
  border-radius: 4px;
  transition: background-color 0.3s;
}

.notification:hover {
  background-color: #f3f4f6;
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
