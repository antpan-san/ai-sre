// Dashboard Type Definitions

// 资源使用情况统计
export interface ResourceUsage {
  cpu: number
  memory: number
  disk: number
  network: {
    in: number
    out: number
  }
}

// 兼容字段：服务端仍可返回；概览页已不再使用（历史上 UI 多块误标）。
export interface KubernetesOverview {
  nodes: number
  pods: number
  runningPods: number
  services: number
  deployments: number
  replicasets: number
}

// 服务状态统计（Linux / Docker 等业务服务）
export interface ServiceStatusStats {
  running: number
  stopped: number
  error: number
  total: number
}

// 最近部署的服务（台账）
export interface RecentDeployment {
  id: string
  name: string
  image: string
  replicas: number
  status: 'running' | 'stopped' | 'error' | 'deploying' | string
  createTime: string
  updateTime: string
}

export interface PlatformSummary {
  machines: {
    total: number
    online: number
    offline: number
    masters: number
    workers: number
  }
  k8sClusters: {
    total: number
    running: number
    pending: number
    failed: number
  }
  tasksActive: number
  executionsLast24h: number
  /** 仅 super_admin 的仪表盘响应包含 */
  usersTotal?: number
  /** 仅 super_admin 的仪表盘响应包含 */
  operationLogsTotal?: number
}

export interface RecentK8sClusterRow {
  id: string
  clusterName: string
  status: string
  version?: string
  masterNode?: string
  updatedAt: string
}

export interface RecentServiceInstallRow {
  id: string
  service: string
  profile?: string
  status: string
  currentStep?: string
  installMethod?: string
  updatedAt: string
}

export interface RecentExecutionRow {
  id: string
  name: string
  status: string
  category?: string
  source?: string
  targetHost?: string
  resourceName?: string
  finishedAt?: string
  durationMs?: number
}

// 仪表盘数据
export interface DashboardData {
  resourceUsage: ResourceUsage
  kubernetesOverview?: KubernetesOverview
  serviceStatusStats: ServiceStatusStats
  recentDeployments: RecentDeployment[]
  /** 租户内汇总快照 */
  platformSummary?: PlatformSummary
  recentK8sClusters?: RecentK8sClusterRow[]
  recentServiceInstalls?: RecentServiceInstallRow[]
  recentExecutions?: RecentExecutionRow[]
  activeAlerts?: number
}

/** 网关包装；axios 拦截器通常只返回内层 data（见 dashboard store）。 */
export interface GetDashboardDataResponse {
  code: number
  data: DashboardData
  msg: string
}
