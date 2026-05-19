import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { ElMessage } from 'element-plus'

const adminRoles = ['admin', 'super_admin']
const superAdminRoles = ['super_admin']
const appRoles = ['admin', 'super_admin', 'user']

const routes: Array<RouteRecordRaw> = [
  { path: '/', redirect: '/admin/dashboard' },
  { path: '/dashboard', redirect: '/admin/dashboard' },
  { path: '/user', redirect: '/admin/user/list' },
  { path: '/user/list', redirect: '/admin/user/list' },
  { path: '/service/deploy', redirect: '/admin/service/deploy' },
  { path: '/service/k8s-deploy/progress', redirect: '/admin/service/k8s-deploy/progress' },
  { path: '/service/k8s-deploy', redirect: '/admin/service/k8s-deploy' },
  { path: '/service/k8s-mirror', redirect: '/admin/k8s-mirror' },
  { path: '/admin/service/k8s-mirror', redirect: '/admin/k8s-mirror' },
  { path: '/service/k8s/clusters', redirect: '/admin/execution-records?tab=k8s' },
  { path: '/admin/service/k8s/clusters', redirect: '/admin/execution-records?tab=k8s' },
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
  { path: '/job', redirect: '/admin/job/center' },
  { path: '/job/center', redirect: '/admin/job/center' },
  { path: '/init-tools', redirect: '/admin/init-tools' },
  { path: '/execution-records', redirect: '/admin/execution-records' },
  { path: '/help', redirect: '/admin/help/error-codes' },
  { path: '/help/error-codes', redirect: '/admin/help/error-codes' },
  { path: '/security-audit/operation-logs', redirect: '/admin/security-audit/operation-logs' },
  { path: '/security-audit/permission-management', redirect: '/admin/security-audit/permission-management' },
  { path: '/security-audit', redirect: '/admin/security-audit/operation-logs' },
  { path: '/advanced/backup-restore', redirect: '/admin/advanced/backup-restore' },
  { path: '/advanced/performance-analysis', redirect: '/admin/advanced/performance-analysis' },
  { path: '/advanced/runtime-observe', redirect: '/admin/advanced/runtime-observe' },
  { path: '/advanced', redirect: '/admin/advanced/backup-restore' },

  {
    path: '/admin',
    name: 'AdminShell',
    component: () => import('../components/layout/MainLayout.vue'),
    meta: { title: '控制台', requireAuth: true, roles: appRoles },
    redirect: '/admin/dashboard',
    children: [
      { path: 'service/k8s-mirror', redirect: '/admin/k8s-mirror' },
      { path: 'service/k8s/clusters', redirect: { path: '/admin/execution-records', query: { tab: 'k8s' } } },
      {
        path: 'dashboard',
        name: 'AdminDashboard',
        component: () => import('../views/Dashboard.vue'),
        meta: { title: '概览', requireAuth: true, roles: appRoles }
      },
      {
        path: 'user/list',
        name: 'AdminUserList',
        component: () => import('../views/user/UserList.vue'),
        meta: { title: '用户', requireAuth: true, roles: adminRoles }
      },
      { path: 'billing', redirect: '/admin/billing/features' },
      {
        path: 'billing/features',
        name: 'AdminBillingFeatures',
        component: () => import('../views/admin/FeatureBilling.vue'),
        meta: {
          title: '套餐与计费',
          requireAuth: true,
          roles: superAdminRoles,
          breadcrumb: [{ title: '订阅与计费' }, { title: '套餐与计费' }]
        }
      },
      {
        path: 'billing/ai-sre-skills',
        name: 'AdminAisreSkills',
        component: () => import('../views/admin/AisreSkillsCatalog.vue'),
        meta: {
          title: 'ai-sre 技能包',
          requireAuth: true,
          roles: superAdminRoles,
          breadcrumb: [
            { title: '订阅与计费', path: '/admin/billing/features' },
            { title: 'ai-sre 技能包' }
          ]
        }
      },
      {
        path: 'auto-iterations',
        name: 'AdminAutoIterations',
        component: () => import('../views/admin/AutoIterations.vue'),
        meta: {
          title: '自动迭代',
          requireAuth: true,
          roles: superAdminRoles,
          breadcrumb: [{ title: '自动迭代' }]
        }
      },
      {
        path: 'auto-iterations/:id',
        redirect: (to) => ({
          path: '/admin/auto-iterations',
          query: { id: String(to.params.id || '') }
        })
      },
      {
        path: 'ai-sre/executions',
        name: 'AdminAISreExecutions',
        component: () => import('../views/ai-sre/ClientExecutions.vue'),
        meta: {
          title: '客户端执行',
          requireAuth: true,
          roles: appRoles,
          breadcrumb: [{ title: 'ai-sre 中心' }, { title: '客户端执行' }]
        }
      },
      {
        path: 'ai-sre/executions/:id',
        name: 'AdminAISreExecutionDetail',
        component: () => import('../views/ai-sre/ClientExecutionDetail.vue'),
        meta: {
          title: '执行复盘',
          requireAuth: true,
          roles: appRoles,
          breadcrumb: [
            { title: 'ai-sre 中心', path: '/admin/ai-sre/executions' },
            { title: '执行复盘' }
          ]
        }
      },
      {
        path: 'ai-sre/skills',
        redirect: '/admin/billing/ai-sre-skills'
      },
      {
        path: 'service/deploy',
        name: 'AdminServiceDeploy',
        component: () => import('../views/service/ServiceDeploy.vue'),
        meta: { title: '应用服务', requireAuth: true, roles: appRoles }
      },
      {
        path: 'service/k8s-deploy',
        name: 'AdminK8sDeploy',
        component: () => import('../views/service/k8s-deploy/K8sDeployForm.vue'),
        meta: {
          title: 'Kubernetes',
          requireAuth: true,
          roles: appRoles,
          breadcrumb: [
            { title: '工作负载', path: '/admin/service/deploy' },
            { title: 'Kubernetes', path: '/admin/service/k8s-deploy' }
          ]
        }
      },
      {
        path: 'service/k8s-deploy/progress',
        name: 'AdminK8sDeployProgress',
        component: () => import('../views/service/k8s-deploy/K8sDeployProgress.vue'),
        meta: {
          title: '进度',
          requireAuth: true,
          roles: appRoles,
          breadcrumb: [
            { title: '工作负载', path: '/admin/service/deploy' },
            { title: 'Kubernetes', path: '/admin/service/k8s-deploy' },
            { title: '进度' }
          ]
        }
      },
      {
        path: 'k8s-mirror',
        name: 'AdminK8sMirrorCatalog',
        component: () => import('../views/service/k8s-mirror/K8sMirrorCatalog.vue'),
        meta: { title: '制品目录', requireAuth: true, roles: appRoles }
      },
      {
        path: 'service/linux',
        name: 'AdminLinuxServiceManagement',
        component: () => import('../views/service/LinuxServiceManagement.vue'),
        meta: { title: 'Linux 主机', requireAuth: true, roles: appRoles }
      },
      {
        path: 'proxy/config',
        name: 'AdminProxyConfig',
        component: () => import('../views/proxy/ProxyConfig.vue'),
        meta: { title: '出口代理', requireAuth: true, roles: appRoles }
      },
      {
        path: 'monitoring/prometheus',
        name: 'AdminPrometheus',
        component: () => import('../views/monitoring/PrometheusConfig.vue'),
        meta: { title: 'Prometheus', requireAuth: true, roles: appRoles }
      },
      {
        path: 'monitoring/node-exporter',
        name: 'AdminNodeExporter',
        component: () => import('../views/monitoring/NodeExporterConfig.vue'),
        meta: { title: 'Node Exporter', requireAuth: true, roles: appRoles }
      },
      {
        path: 'monitoring/jmx-exporter',
        name: 'AdminJmxExporter',
        component: () => import('../views/monitoring/JmxExporterConfig.vue'),
        meta: { title: 'JMX Exporter', requireAuth: true, roles: appRoles }
      },
      {
        path: 'monitoring/redis-exporter',
        name: 'AdminRedisExporter',
        component: () => import('../views/monitoring/RedisExporterConfig.vue'),
        meta: { title: 'Redis Exporter', requireAuth: true, roles: appRoles }
      },
      {
        path: 'monitoring/mongodb-exporter',
        name: 'AdminMongoDBExporter',
        component: () => import('../views/monitoring/MongoDBExporterConfig.vue'),
        meta: { title: 'MongoDB Exporter', requireAuth: true, roles: appRoles }
      },
      {
        path: 'monitoring/blackbox-exporter',
        name: 'AdminBlackboxExporter',
        component: () => import('../views/monitoring/BlackboxExporterConfig.vue'),
        meta: { title: 'Blackbox Exporter', requireAuth: true, roles: appRoles }
      },
      {
        path: 'job/center',
        name: 'AdminJobCenter',
        component: () => import('../views/job/JobCenter.vue'),
        meta: { title: '作业中心', requireAuth: true, roles: appRoles }
      },
      {
        path: 'init-tools',
        name: 'AdminInitTools',
        component: () => import('../views/init-tools/InitToolsHome.vue'),
        meta: { title: '初始化', requireAuth: true, roles: appRoles }
      },
      {
        path: 'execution-records',
        name: 'AdminExecutionRecords',
        component: () => import('../views/execution-records/ExecutionRecords.vue'),
        meta: { title: '执行记录', requireAuth: true, roles: appRoles }
      },
      {
        path: 'help/error-codes',
        name: 'AdminErrorCodesLookup',
        component: () => import('../views/help/ErrorCodesLookup.vue'),
        meta: {
          title: '错误码',
          requireAuth: true,
          roles: appRoles,
          breadcrumb: [{ title: '工具' }, { title: '错误码' }]
        }
      },
      {
        path: 'security-audit/operation-logs',
        name: 'AdminOperationLogs',
        component: () => import('../views/security-audit/OperationLogs.vue'),
        meta: { title: '审计日志', requireAuth: true, roles: appRoles }
      },
      {
        path: 'security-audit/permission-management',
        name: 'AdminPermissionManagement',
        component: () => import('../views/security-audit/PermissionManagement.vue'),
        meta: { title: '权限', requireAuth: true, roles: adminRoles }
      },
      {
        path: 'advanced/backup-restore',
        name: 'AdminBackupRestore',
        component: () => import('../views/advanced/BackupRestore.vue'),
        meta: { title: '备份', requireAuth: true, roles: appRoles }
      },
      {
        path: 'advanced/performance-analysis',
        name: 'AdminPerformanceAnalysis',
        component: () => import('../views/advanced/PerformanceAnalysis.vue'),
        meta: { title: '性能', requireAuth: true, roles: appRoles }
      },
      {
        path: 'advanced/runtime-observe',
        name: 'AdminRuntimeObserve',
        component: () => import('../views/advanced/RuntimeObserve.vue'),
        meta: {
          title: '运行时诊断',
          requireAuth: true,
          roles: appRoles,
          breadcrumb: [{ title: '数据' }, { title: '运行时诊断' }]
        }
      }
    ]
  },

  {
    path: '/app',
    name: 'AppShell',
    component: () => import('../components/layout/MainLayout.vue'),
    meta: { title: '工作台', requireAuth: true, roles: appRoles },
    redirect: '/app/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'AppDashboard',
        component: () => import('../views/Dashboard.vue'),
        meta: { title: '概览', requireAuth: true, roles: appRoles }
      },
      {
        path: 'job/center',
        name: 'AppJobCenter',
        component: () => import('../views/job/JobCenter.vue'),
        meta: { title: '作业中心', requireAuth: true, roles: appRoles }
      },
      {
        path: 'ai-sre/executions',
        name: 'AppAISreExecutions',
        component: () => import('../views/ai-sre/ClientExecutions.vue'),
        meta: { title: '客户端执行', requireAuth: true, roles: appRoles }
      },
      {
        path: 'ai-sre/executions/:id',
        name: 'AppAISreExecutionDetail',
        component: () => import('../views/ai-sre/ClientExecutionDetail.vue'),
        meta: { title: '执行复盘', requireAuth: true, roles: appRoles }
      },
      {
        path: 'execution-records',
        name: 'AppExecutionRecords',
        component: () => import('../views/execution-records/ExecutionRecords.vue'),
        meta: { title: '执行记录', requireAuth: true, roles: appRoles }
      },
      {
        path: 'init-tools',
        name: 'AppInitTools',
        component: () => import('../views/init-tools/InitToolsHome.vue'),
        meta: { title: '初始化', requireAuth: true, roles: appRoles }
      },
      {
        path: 'advanced/backup-restore',
        name: 'AppBackupRestore',
        component: () => import('../views/advanced/BackupRestore.vue'),
        meta: { title: '备份', requireAuth: true, roles: appRoles }
      },
      {
        path: 'advanced/performance-analysis',
        name: 'AppPerformanceAnalysis',
        component: () => import('../views/advanced/PerformanceAnalysis.vue'),
        meta: { title: '性能', requireAuth: true, roles: appRoles }
      },
      {
        path: 'advanced/runtime-observe',
        name: 'AppRuntimeObserve',
        component: () => import('../views/advanced/RuntimeObserve.vue'),
        meta: {
          title: '运行时诊断',
          requireAuth: true,
          roles: appRoles,
          breadcrumb: [{ title: '数据' }, { title: '运行时诊断' }]
        }
      },
      {
        path: 'help/error-codes',
        name: 'AppErrorCodesLookup',
        component: () => import('../views/help/ErrorCodesLookup.vue'),
        meta: {
          title: '错误码',
          requireAuth: true,
          roles: appRoles,
          breadcrumb: [{ title: '工具' }, { title: '错误码' }]
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
    redirect: '/admin/init-tools'
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
