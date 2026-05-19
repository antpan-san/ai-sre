// Dashboard Type Definitions

// 资源使用情况统计
export interface ResourceUsage {
  cpu: number
  /** 1 分钟 load average 相对 CPU 核数，0–100 */
  load?: number
  memory: number
  disk: number
  /** 磁盘 IO 忙百分比，0–100 */
  diskIo?: number
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
  deploying: number
  stopped: number
  error: number
  total: number
  /** 运行 + 部署中，与「健康台数」口径一致 */
  operational?: number
}

// 最近部署的服务（台账）
export interface RecentDeployment {
  id: string
  name: string
  /** 平台部署/配置的产品或功能名（来自 config 约定键、description 或类型缺省文案） */
  productName: string
  /** 资源摘要：镜像、关联主机、端口等 */
  resource: string
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
  /** 近 24h 内状态为 failed 的执行条数（与 executionsLast24h 同一角色/用户范围） */
  executionsFailedLast24h?: number
  /** 近 24h status=success，同一可见范围 */
  executionsSuccessLast24h?: number
  /** 近 24h status=cancelled */
  executionsCancelledLast24h?: number
  /** 近 24h 按 source 字段计数（任意状态） */
  executionsBySourceLast24h?: {
    cli: number
    k8s: number
    job: number
  }
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

/** 仅 super_admin：运行 opsfleet-backend 的本机采样元数据 */
export interface HostRuntimeMeta {
  hostname: string
  sampledAt: string
  os?: string
  /** 原始 1 分钟 load average */
  load1?: number
  error?: string
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
  /** 仅 super_admin */
  hostRuntime?: HostRuntimeMeta
  activeAlerts?: number
}

/** 导航栏资源圆环专用（super_admin） */
export interface DashboardHostResources {
  resourceUsage: ResourceUsage
  hostRuntime?: HostRuntimeMeta
}

/** 网关包装；axios 拦截器通常只返回内层 data（见 dashboard store）。 */
export interface GetDashboardDataResponse {
  code: number
  data: DashboardData
  msg: string
}
