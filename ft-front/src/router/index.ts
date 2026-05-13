import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { ElMessage } from 'element-plus'

function redirectByRole(adminPath: string, appPath: string) {
  return () => {
    const s = localStorage.getItem('userInfo')
    try {
      const u = JSON.parse(s || '{}') as { role?: string }
      return u.role === 'admin' ? adminPath : appPath
    } catch {
      return '/login'
    }
  }
}

const routes: Array<RouteRecordRaw> = [
  { path: '/', redirect: redirectByRole('/admin/dashboard', '/app/dashboard') },
  { path: '/dashboard', redirect: redirectByRole('/admin/dashboard', '/app/dashboard') },
  { path: '/user', redirect: '/admin/user/list' },
  { path: '/user/list', redirect: '/admin/user/list' },
  { path: '/service/deploy', redirect: '/admin/service/deploy' },
  { path: '/service/k8s-deploy/progress', redirect: '/admin/service/k8s-deploy/progress' },
  { path: '/service/k8s-deploy', redirect: '/admin/service/k8s-deploy' },
  { path: '/service/k8s-mirror', redirect: '/admin/service/k8s-mirror' },
  { path: '/service/k8s/clusters', redirect: '/admin/service/k8s/clusters' },
  { path: '/service/linux', redirect: '/admin/service/linux' },
  { path: '/service', redirect: '/admin/service/deploy' },
  { path: '/proxy/config', redirect: '/admin/proxy/config' },
  { path: '/proxy', redirect: '/admin/proxy/config' },
  { path: '/monitoring/prometheus', redirect: '/admin/monitoring/prometheus' },
  { path: '/monitoring/node-exporter', redirect: '/admin/monitoring/node-exporter' },
  { path: '/monitoring/jmx-exporter', redirect: '/admin/monitoring/jmx-exporter' },
  { path: '/monitoring/redis-exporter', redirect: '/admin/monitoring/redis-exporter' },
  { path: '/monitoring/mongodb-exporter', redirect: '/admin/monitoring/mongodb-exporter' },
  { path: '/monitoring/blackbox-exporter', redirect: '/admin/monitoring/blackbox-exporter' },
  { path: '/monitoring', redirect: '/admin/monitoring/prometheus' },
  { path: '/job', redirect: '/app/job/center' },
  { path: '/job/center', redirect: '/app/job/center' },
  { path: '/init-tools', redirect: '/app/init-tools' },
  { path: '/execution-records', redirect: '/admin/execution-records' },
  { path: '/help', redirect: '/app/help/error-codes' },
  { path: '/help/error-codes', redirect: '/app/help/error-codes' },
  { path: '/security-audit/operation-logs', redirect: '/admin/security-audit/operation-logs' },
  { path: '/security-audit/permission-management', redirect: '/admin/security-audit/permission-management' },
  { path: '/security-audit', redirect: '/admin/security-audit/operation-logs' },
  { path: '/advanced/backup-restore', redirect: '/admin/advanced/backup-restore' },
  { path: '/advanced/performance-analysis', redirect: '/admin/advanced/performance-analysis' },
  { path: '/advanced', redirect: '/admin/advanced/backup-restore' },

  {
    path: '/admin',
    name: 'AdminShell',
    component: () => import('../components/layout/MainLayout.vue'),
    meta: { title: '管理端', requireAuth: true, roles: ['admin'] },
    redirect: '/admin/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'AdminDashboard',
        component: () => import('../views/Dashboard.vue'),
        meta: { title: '仪表盘', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'user/list',
        name: 'AdminUserList',
        component: () => import('../views/user/UserList.vue'),
        meta: { title: '用户列表', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'billing/features',
        name: 'AdminBillingFeatures',
        component: () => import('../views/admin/FeatureBilling.vue'),
        meta: { title: '功能与计费', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'service/deploy',
        name: 'AdminServiceDeploy',
        component: () => import('../views/service/ServiceDeploy.vue'),
        meta: { title: '服务部署', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'service/k8s-deploy',
        name: 'AdminK8sDeploy',
        component: () => import('../views/service/k8s-deploy/K8sDeployForm.vue'),
        meta: {
          title: 'Kubernetes 部署',
          requireAuth: true,
          roles: ['admin'],
          breadcrumb: [
            { title: '服务与交付', path: '/admin/service/deploy' },
            { title: 'Kubernetes 部署', path: '/admin/service/k8s-deploy' }
          ]
        }
      },
      {
        path: 'service/k8s-deploy/progress',
        name: 'AdminK8sDeployProgress',
        component: () => import('../views/service/k8s-deploy/K8sDeployProgress.vue'),
        meta: {
          title: 'Kubernetes 部署进度',
          requireAuth: true,
          roles: ['admin'],
          breadcrumb: [
            { title: '服务与交付', path: '/admin/service/deploy' },
            { title: 'Kubernetes 部署', path: '/admin/service/k8s-deploy' },
            { title: '部署进度' }
          ]
        }
      },
      {
        path: 'service/k8s-mirror',
        name: 'AdminK8sMirrorCatalog',
        component: () => import('../views/service/k8s-mirror/K8sMirrorCatalog.vue'),
        meta: { title: 'K8s 制品镜像', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'service/k8s/clusters',
        name: 'AdminK8sClusterList',
        component: () => import('../views/service/k8s-deploy/K8sClusterList.vue'),
        meta: { title: 'Kubernetes 集群列表', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'service/linux',
        name: 'AdminLinuxServiceManagement',
        component: () => import('../views/service/LinuxServiceManagement.vue'),
        meta: { title: 'Linux服务管理', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'proxy/config',
        name: 'AdminProxyConfig',
        component: () => import('../views/proxy/ProxyConfig.vue'),
        meta: { title: '代理配置管理', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'monitoring/prometheus',
        name: 'AdminPrometheus',
        component: () => import('../views/monitoring/PrometheusConfig.vue'),
        meta: { title: 'Prometheus', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'monitoring/node-exporter',
        name: 'AdminNodeExporter',
        component: () => import('../views/monitoring/NodeExporterConfig.vue'),
        meta: { title: 'Node Exporter', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'monitoring/jmx-exporter',
        name: 'AdminJmxExporter',
        component: () => import('../views/monitoring/JmxExporterConfig.vue'),
        meta: { title: 'JMX Exporter', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'monitoring/redis-exporter',
        name: 'AdminRedisExporter',
        component: () => import('../views/monitoring/RedisExporterConfig.vue'),
        meta: { title: 'Redis Exporter', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'monitoring/mongodb-exporter',
        name: 'AdminMongoDBExporter',
        component: () => import('../views/monitoring/MongoDBExporterConfig.vue'),
        meta: { title: 'MongoDB Exporter', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'monitoring/blackbox-exporter',
        name: 'AdminBlackboxExporter',
        component: () => import('../views/monitoring/BlackboxExporterConfig.vue'),
        meta: { title: 'Blackbox Exporter', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'job/center',
        name: 'AdminJobCenter',
        component: () => import('../views/job/JobCenter.vue'),
        meta: { title: '作业中心', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'init-tools',
        name: 'AdminInitTools',
        component: () => import('../views/init-tools/InitToolsHome.vue'),
        meta: { title: '初始化工具', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'execution-records',
        name: 'AdminExecutionRecords',
        component: () => import('../views/execution-records/ExecutionRecords.vue'),
        meta: { title: '执行记录', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'help/error-codes',
        name: 'AdminErrorCodesLookup',
        component: () => import('../views/help/ErrorCodesLookup.vue'),
        meta: {
          title: '部署错误码查询',
          requireAuth: true,
          roles: ['admin'],
          breadcrumb: [{ title: '帮助中心' }, { title: '部署错误码查询' }]
        }
      },
      {
        path: 'security-audit/operation-logs',
        name: 'AdminOperationLogs',
        component: () => import('../views/security-audit/OperationLogs.vue'),
        meta: { title: '操作日志', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'security-audit/permission-management',
        name: 'AdminPermissionManagement',
        component: () => import('../views/security-audit/PermissionManagement.vue'),
        meta: { title: '权限管理', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'advanced/backup-restore',
        name: 'AdminBackupRestore',
        component: () => import('../views/advanced/BackupRestore.vue'),
        meta: { title: '备份与恢复', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'advanced/performance-analysis',
        name: 'AdminPerformanceAnalysis',
        component: () => import('../views/advanced/PerformanceAnalysis.vue'),
        meta: { title: '性能分析', requireAuth: true, roles: ['admin'] }
      }
    ]
  },

  {
    path: '/app',
    name: 'AppShell',
    component: () => import('../components/layout/MainLayout.vue'),
    meta: { title: '工作台', requireAuth: true, roles: ['admin', 'user'] },
    redirect: '/app/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'AppDashboard',
        component: () => import('../views/Dashboard.vue'),
        meta: { title: '仪表盘', requireAuth: true, roles: ['admin', 'user'] }
      },
      {
        path: 'job/center',
        name: 'AppJobCenter',
        component: () => import('../views/job/JobCenter.vue'),
        meta: { title: '作业中心', requireAuth: true, roles: ['admin', 'user'] }
      },
      {
        path: 'init-tools',
        name: 'AppInitTools',
        component: () => import('../views/init-tools/InitToolsHome.vue'),
        meta: { title: '初始化工具', requireAuth: true, roles: ['admin', 'user'] }
      },
      {
        path: 'help/error-codes',
        name: 'AppErrorCodesLookup',
        component: () => import('../views/help/ErrorCodesLookup.vue'),
        meta: {
          title: '部署错误码查询',
          requireAuth: true,
          roles: ['admin', 'user'],
          breadcrumb: [{ title: '帮助中心' }, { title: '部署错误码查询' }]
        }
      }
    ]
  },

  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/login/Login.vue'),
    meta: { title: '登录', requireAuth: false }
  },
  {
    path: '/register',
    name: 'Register',
    component: () => import('../views/login/Register.vue'),
    meta: { title: '注册', requireAuth: false }
  },
  {
    path: '/init-tools/:legacy(system-param|time-sync|security-hardening|disk-partition)',
    redirect: '/app/init-tools'
  }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes
})

router.beforeEach((to, _from, next) => {
  if (to.meta.requireAuth) {
    const token = localStorage.getItem('token')
    if (token) {
      const userStr = localStorage.getItem('userInfo')
      if (userStr) {
        let user: { role?: string } | null = null
        try {
          user = JSON.parse(userStr)
        } catch {
          localStorage.removeItem('token')
          localStorage.removeItem('userInfo')
          next('/login')
          return
        }

        if (to.path.startsWith('/admin') && user?.role !== 'admin') {
          ElMessage.error('没有权限访问管理端')
          next('/app/dashboard')
          return
        }

        if (to.meta.roles && user?.role) {
          const roles = to.meta.roles as Array<string>
          if (roles.includes(user.role)) {
            next()
          } else {
            ElMessage.error('没有权限访问该页面')
            next(false)
          }
        } else if (!to.meta.roles) {
          next()
        } else {
          next('/login')
        }
      } else {
        next('/login')
      }
    } else {
      next('/login')
    }
  } else {
    next()
  }
})

export default router
