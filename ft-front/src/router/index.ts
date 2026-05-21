import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { ElMessage } from 'element-plus'

const adminRoles = ['admin', 'super_admin']
const superAdminRoles = ['super_admin']
const appRoles = ['admin', 'super_admin', 'user']

type AppHubSection = 'delivery' | 'troubleshoot' | 'observe' | 'monitoring' | 'data'

function roleHomePath() {
  try {
    const token = localStorage.getItem('token')
    if (!token) return '/app/dashboard'
    const raw = localStorage.getItem('userInfo') || '{}'
    const role = String((JSON.parse(raw) as { role?: string }).role || '')
    return role === 'admin' || role === 'super_admin' ? '/admin/dashboard' : '/app/dashboard'
  } catch {
    return '/app/dashboard'
  }
}

function appHub(section: AppHubSection, hubCapabilityId?: string) {
  const isDelivery = section === 'delivery'
  return {
    hubMenu: isDelivery ? '/app/workloads' : '/app/capabilities',
    hubTitle: isDelivery ? '工作负载' : '能力中心',
    hubSection: section,
    ...(hubCapabilityId ? { hubCapabilityId } : {})
  }
}

const routes: Array<RouteRecordRaw> = [
  { path: '/', redirect: () => roleHomePath() },
  { path: '/dashboard', redirect: () => roleHomePath() },
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
          title: '订阅包与计费',
          requireAuth: true,
          roles: superAdminRoles,
          breadcrumb: [{ title: '订阅与计费' }, { title: '订阅包与计费' }]
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
        path: 'ai-sre/skill-refinement',
        name: 'AdminAISreSkillRefinement',
        component: () => import('../views/ai-sre/SkillRefinement.vue'),
        meta: {
          title: '技能精炼',
          requireAuth: true,
          roles: superAdminRoles,
          breadcrumb: [{ title: 'ai-sre 中心' }, { title: '技能精炼' }]
        }
      },
      {
        path: 'ai-sre/skills',
        redirect: '/admin/ai-sre/skill-refinement'
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
        meta: {
          title: '通用执行审计',
          requireAuth: true,
          roles: appRoles,
          breadcrumb: [{ title: 'ai-sre 中心', path: '/admin/ai-sre/executions' }, { title: '通用执行审计' }]
        }
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
          breadcrumb: [{ title: 'ai-sre 中心', path: '/admin/ai-sre/executions' }, { title: '运行时诊断' }]
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
        component: () => import('../views/app/AppDashboard.vue'),
        meta: { title: '概览', requireAuth: true, roles: appRoles }
      },
      {
        path: 'execution-records',
        name: 'AppExecutionRecords',
        component: () => import('../views/execution-records/ExecutionRecords.vue'),
        meta: { title: '执行记录', requireAuth: true, roles: appRoles }
      },
      {
        path: 'executions/:id',
        name: 'AppExecutionDetail',
        component: () => import('../views/ai-sre/ClientExecutionDetail.vue'),
        meta: { title: '执行详情', requireAuth: true, roles: appRoles }
      },
      {
        path: 'deploy',
        redirect: (to) => ({
          path: '/app/workloads',
          query: to.query,
          hash: to.hash || undefined
        })
      },
      {
        path: 'workloads',
        name: 'AppWorkloads',
        component: () => import('../views/app/DeployCenter.vue'),
        meta: { title: '工作负载', requireAuth: true, roles: appRoles }
      },
      {
        path: 'workloads/service/:serviceKey',
        redirect: (to) => ({ path: `/app/service/deploy/${encodeURIComponent(String(to.params.serviceKey || ''))}` })
      },
      {
        path: 'workloads/:capId',
        name: 'AppWorkloadCapabilityDetail',
        component: () => import('../views/app/WorkloadCapabilityDetail.vue'),
        meta: {
          title: '工作负载详情',
          requireAuth: true,
          roles: appRoles,
          ...appHub('delivery')
        }
      },
      {
        path: 'capabilities',
        name: 'AppCapabilityCenter',
        component: () => import('../views/app/CapabilityCenter.vue'),
        meta: { title: '能力中心', requireAuth: true, roles: appRoles }
      },
      {
        path: 'troubleshooting',
        name: 'AppTroubleshooting',
        component: () => import('../views/app/Troubleshooting.vue'),
        meta: { title: '问题排查', requireAuth: true, roles: appRoles }
      },
      {
        path: 'job/center',
        name: 'AppJobCenter',
        component: () => import('../views/job/JobCenter.vue'),
        meta: { title: '作业中心', requireAuth: true, roles: appRoles }
      },
      {
        path: 'settings',
        name: 'AppSettings',
        component: () => import('../views/app/AppSettings.vue'),
        meta: { title: '设置', requireAuth: true, roles: appRoles }
      },
      {
        path: 'service/deploy/:serviceKey',
        name: 'AppServiceDeployDetail',
        component: () => import('../views/service/AppServiceDeployDetail.vue'),
        meta: { title: '服务部署', requireAuth: true, roles: appRoles, ...appHub('delivery', 'service_deploy') }
      },
      {
        path: 'service/deploy',
        name: 'AppServiceDeploy',
        component: () => import('../views/service/ServiceDeploy.vue'),
        meta: { title: '应用服务', requireAuth: true, roles: appRoles, ...appHub('delivery', 'service_deploy') }
      },
      {
        path: 'service/k8s-deploy',
        name: 'AppK8sDeploy',
        component: () => import('../views/service/k8s-deploy/K8sDeployForm.vue'),
        meta: { title: 'Kubernetes', requireAuth: true, roles: appRoles, ...appHub('delivery', 'k8s_delivery') }
      },
      {
        path: 'service/k8s-deploy/progress',
        name: 'AppK8sDeployProgress',
        component: () => import('../views/service/k8s-deploy/K8sDeployProgress.vue'),
        meta: { title: 'K8s 进度', requireAuth: true, roles: appRoles, ...appHub('delivery', 'k8s_delivery') }
      },
      {
        path: 'service/linux',
        name: 'AppLinuxServiceManagement',
        component: () => import('../views/service/LinuxServiceManagement.vue'),
        meta: { title: 'Linux 主机', requireAuth: true, roles: appRoles, ...appHub('delivery', 'linux_hosts') }
      },
      {
        path: 'k8s-mirror',
        name: 'AppK8sMirrorCatalog',
        component: () => import('../views/service/k8s-mirror/K8sMirrorCatalog.vue'),
        meta: { title: '制品目录', requireAuth: true, roles: appRoles, ...appHub('delivery', 'k8s_mirror') }
      },
      {
        path: 'init-tools',
        name: 'AppInitTools',
        component: () => import('../views/init-tools/InitToolsHome.vue'),
        meta: { title: '初始化', requireAuth: true, roles: appRoles, ...appHub('delivery', 'init_tools') }
      },
      {
        path: 'monitoring/prometheus',
        name: 'AppPrometheus',
        component: () => import('../views/monitoring/PrometheusConfig.vue'),
        meta: { title: 'Prometheus', requireAuth: true, roles: appRoles, ...appHub('monitoring', 'prometheus') }
      },
      {
        path: 'monitoring/node-exporter',
        name: 'AppNodeExporter',
        component: () => import('../views/monitoring/NodeExporterConfig.vue'),
        meta: { title: 'Node Exporter', requireAuth: true, roles: appRoles, ...appHub('monitoring', 'exporters') }
      },
      {
        path: 'monitoring/jmx-exporter',
        name: 'AppJmxExporter',
        component: () => import('../views/monitoring/JmxExporterConfig.vue'),
        meta: { title: 'JMX Exporter', requireAuth: true, roles: appRoles, ...appHub('monitoring', 'exporters') }
      },
      {
        path: 'monitoring/redis-exporter',
        name: 'AppRedisExporter',
        component: () => import('../views/monitoring/RedisExporterConfig.vue'),
        meta: { title: 'Redis Exporter', requireAuth: true, roles: appRoles, ...appHub('monitoring', 'exporters') }
      },
      {
        path: 'monitoring/mongodb-exporter',
        name: 'AppMongoDBExporter',
        component: () => import('../views/monitoring/MongoDBExporterConfig.vue'),
        meta: { title: 'MongoDB Exporter', requireAuth: true, roles: appRoles, ...appHub('monitoring', 'exporters') }
      },
      {
        path: 'monitoring/blackbox-exporter',
        name: 'AppBlackboxExporter',
        component: () => import('../views/monitoring/BlackboxExporterConfig.vue'),
        meta: { title: 'Blackbox Exporter', requireAuth: true, roles: appRoles, ...appHub('monitoring', 'exporters') }
      },
      {
        path: 'advanced/backup-restore',
        name: 'AppBackupRestore',
        component: () => import('../views/advanced/BackupRestore.vue'),
        meta: { title: '备份', requireAuth: true, roles: appRoles, ...appHub('data', 'backup') }
      },
      {
        path: 'advanced/performance-analysis',
        name: 'AppPerformanceAnalysis',
        component: () => import('../views/advanced/PerformanceAnalysis.vue'),
        meta: { title: '性能', requireAuth: true, roles: appRoles, ...appHub('data', 'performance') }
      },
      {
        path: 'advanced/runtime-observe',
        name: 'AppRuntimeObserve',
        component: () => import('../views/advanced/RuntimeObserve.vue'),
        meta: { title: '运行时诊断', requireAuth: true, roles: appRoles, ...appHub('observe', 'runtime_observe') }
      },
      {
        path: 'help/error-codes',
        name: 'AppErrorCodesLookup',
        component: () => import('../views/help/ErrorCodesLookup.vue'),
        meta: { title: '错误码', requireAuth: true, roles: appRoles }
      },
      { path: 'ai-sre/executions', redirect: '/app/execution-records' },
      { path: 'ai-sre/executions/:id', redirect: (to) => ({ path: `/app/executions/${to.params.id}` }) }
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

router.onError((error, to) => {
  const msg = String(error?.message || error || '')
  if (!/Failed to fetch dynamically imported module|Importing a module script failed|error loading dynamically imported module/i.test(msg)) {
    return
  }

  const target = to.fullPath || to.path || window.location.pathname
  const reloadKey = `route-chunk-reload:${target}`
  if (sessionStorage.getItem(reloadKey) === '1') {
    sessionStorage.removeItem(reloadKey)
    ElMessage.error('页面资源已更新，请手动刷新后重试')
    return
  }

  sessionStorage.setItem(reloadKey, '1')
  window.location.assign(target)
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

        const role = user?.role || ''
        if (role === 'user' && to.path.startsWith('/admin')) {
          next('/app/dashboard')
          return
        }

        if (to.meta.roles && role) {
          const roles = to.meta.roles as Array<string>
          if (roles.includes(role)) {
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
