import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { ElMessage } from 'element-plus'

const routes: Array<RouteRecordRaw> = [
  {
    path: '/',
    redirect: '/dashboard'
  },
  {
    path: '/dashboard',
    name: 'Dashboard',
    component: () => import('../components/layout/MainLayout.vue'),
    meta: { title: '仪表盘', requireAuth: true, roles: ['admin', 'user'] },
    children: [
      {
        path: '',
        name: 'DashboardIndex',
        component: () => import('../views/Dashboard.vue'),
        meta: { title: '仪表盘', requireAuth: true, roles: ['admin', 'user'] }
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
    path: '/machine',
    name: 'Machine',
    component: () => import('../components/layout/MainLayout.vue'),
    meta: { title: '机器管理', requireAuth: true, roles: ['admin', 'user'] },
    children: [
      {
        path: 'list',
        name: 'MachineList',
        component: () => import('../views/machine/MachineList.vue'),
        meta: { title: '机器管理', requireAuth: true, roles: ['admin', 'user'] }
      }
    ]
  },
  {
    path: '/user',
    name: 'User',
    component: () => import('../components/layout/MainLayout.vue'),
    meta: { title: '用户管理', requireAuth: true, roles: ['admin'] },
    children: [
      {
        path: 'list',
        name: 'UserList',
        component: () => import('../views/user/UserList.vue'),
        meta: { title: '用户列表', requireAuth: true, roles: ['admin'] }
      }
    ]
  },
  {
    path: '/service',
    name: 'Service',
    component: () => import('../components/layout/MainLayout.vue'),
    meta: { title: '服务管理', requireAuth: true, roles: ['admin'] },
    children: [
      {
        path: 'deploy',
        name: 'ServiceDeploy',
        component: () => import('../views/service/ServiceDeploy.vue'),
        meta: { title: '服务部署', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'k8s-deploy',
        name: 'K8sDeploy',
        component: () => import('../views/service/k8s-deploy/K8sDeployForm.vue'),
        meta: { title: 'Kubernetes部署', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'k8s-deploy/progress',
        name: 'K8sDeployProgress',
        component: () => import('../views/service/k8s-deploy/K8sDeployProgress.vue'),
        meta: { title: 'Kubernetes部署进度', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'k8s/clusters',
        name: 'K8sClusterList',
        component: () => import('../views/service/k8s-deploy/K8sClusterList.vue'),
        meta: { title: 'Kubernetes集群列表', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'linux',
        name: 'LinuxServiceManagement',
        component: () => import('../views/service/LinuxServiceManagement.vue'),
        meta: { title: 'Linux服务管理', requireAuth: true, roles: ['admin'] }
      }
    ]
  },
  {
    path: '/proxy',
    name: 'Proxy',
    component: () => import('../components/layout/MainLayout.vue'),
    meta: { title: '代理配置', requireAuth: true, roles: ['admin'] },
    children: [
      {
        path: 'config',
        name: 'ProxyConfig',
        component: () => import('../views/proxy/ProxyConfig.vue'),
        meta: { title: '代理配置管理', requireAuth: true, roles: ['admin'] }
      }
    ]
  },
  {
    path: '/monitoring',
    name: 'Monitoring',
    component: () => import('../components/layout/MainLayout.vue'),
    meta: { title: '监控告警', requireAuth: true, roles: ['admin'] },
    children: [
      {
        path: 'prometheus',
        name: 'Prometheus',
        component: () => import('../views/monitoring/PrometheusConfig.vue'),
        meta: { title: 'Prometheus', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'node-exporter',
        name: 'NodeExporter',
        component: () => import('../views/monitoring/NodeExporterConfig.vue'),
        meta: { title: 'Node Exporter', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'jmx-exporter',
        name: 'JmxExporter',
        component: () => import('../views/monitoring/JmxExporterConfig.vue'),
        meta: { title: 'JMX Exporter', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'redis-exporter',
        name: 'RedisExporter',
        component: () => import('../views/monitoring/RedisExporterConfig.vue'),
        meta: { title: 'Redis Exporter', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'mongodb-exporter',
        name: 'MongoDBExporter',
        component: () => import('../views/monitoring/MongoDBExporterConfig.vue'),
        meta: { title: 'MongoDB Exporter', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'blackbox-exporter',
        name: 'BlackboxExporter',
        component: () => import('../views/monitoring/BlackboxExporterConfig.vue'),
        meta: { title: 'Blackbox Exporter', requireAuth: true, roles: ['admin'] }
      }
    ]
  },
  {
    path: '/job',
    name: 'Job',
    component: () => import('../components/layout/MainLayout.vue'),
    meta: { title: '作业中心', requireAuth: true, roles: ['admin', 'user'] },
    children: [
      {
        path: 'center',
        name: 'JobCenter',
        component: () => import('../views/job/JobCenter.vue'),
        meta: { title: '作业中心', requireAuth: true, roles: ['admin', 'user'] }
      }
    ]
  },
  {
    path: '/init-tools',
    name: 'InitTools',
    component: () => import('../components/layout/MainLayout.vue'),
    meta: { title: '初始化工具', requireAuth: true, roles: ['admin'] },
    children: [
      {
        path: '',
        redirect: '/init-tools/system-param'
      },
      {
        path: 'system-param',
        name: 'SystemParamOptimize',
        component: () => import('../views/init-tools/SystemParamOptimize.vue'),
        meta: { title: '系统参数优化', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'time-sync',
        name: 'TimeSync',
        component: () => import('../views/init-tools/TimeSync.vue'),
        meta: { title: '时间同步', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'security-hardening',
        name: 'SecurityHardening',
        component: () => import('../views/init-tools/SecurityHardening.vue'),
        meta: { title: '系统安全加固', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'disk-partition',
        name: 'DiskPartitionOptimize',
        component: () => import('../views/init-tools/DiskPartitionOptimize.vue'),
        meta: { title: '磁盘分区优化', requireAuth: true, roles: ['admin'] }
      }
    ]
  },
  // 安全与审计模块
  {
    path: '/security-audit',
    name: 'SecurityAudit',
    component: () => import('../components/layout/MainLayout.vue'),
    meta: { title: '安全与审计', requireAuth: true, roles: ['admin'] },
    children: [
      {
        path: 'operation-logs',
        name: 'OperationLogs',
        component: () => import('../views/security-audit/OperationLogs.vue'),
        meta: { title: '操作日志', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'permission-management',
        name: 'PermissionManagement',
        component: () => import('../views/security-audit/PermissionManagement.vue'),
        meta: { title: '权限管理', requireAuth: true, roles: ['admin'] }
      }
    ]
  },
  // 高级功能模块
  {
    path: '/advanced',
    name: 'Advanced',
    component: () => import('../components/layout/MainLayout.vue'),
    meta: { title: '高级功能', requireAuth: true, roles: ['admin'] },
    children: [
      {
        path: 'backup-restore',
        name: 'BackupRestore',
        component: () => import('../views/advanced/BackupRestore.vue'),
        meta: { title: '备份与恢复', requireAuth: true, roles: ['admin'] }
      },
      {
        path: 'performance-analysis',
        name: 'PerformanceAnalysis',
        component: () => import('../views/advanced/PerformanceAnalysis.vue'),
        meta: { title: '性能分析', requireAuth: true, roles: ['admin'] }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes
})

// 路由守卫
router.beforeEach((to, _from, next) => {
  if (to.meta.requireAuth) {
    const token = localStorage.getItem('token')
    if (token) {
      // 从localStorage获取用户信息
      const userStr = localStorage.getItem('userInfo')
      if (userStr) {
        let user: { role?: string } | null = null
        try {
          user = JSON.parse(userStr)
        } catch {
          // localStorage 数据损坏，清理后跳转登录
          localStorage.removeItem('token')
          localStorage.removeItem('userInfo')
          next('/login')
          return
        }

        // 检查路由是否需要特定角色
        if (to.meta.roles && user?.role) {
          const roles = to.meta.roles as Array<string>
          // 检查用户角色是否在允许的角色列表中
          if (roles.includes(user.role)) {
            next()
          } else {
            // 没有权限，跳转到403页面（如果有）或者返回上一页
            ElMessage.error('没有权限访问该页面')
            next(false)
          }
        } else if (!to.meta.roles) {
          // 路由没有指定角色，直接放行
          next()
        } else {
          // 用户信息无效，跳转到登录页
          next('/login')
        }
      } else {
        // 没有用户信息，跳转到登录页
        next('/login')
      }
    } else {
      // 没有token，跳转到登录页
      next('/login')
    }
  } else {
    // 不需要认证的路由，直接放行
    next()
  }
})

export default router
