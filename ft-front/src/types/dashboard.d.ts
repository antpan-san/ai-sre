// Dashboard Type Definitions

// 资源使用情况统计
export interface ResourceUsage {
  cpu: number; // CPU使用率百分比
  memory: number; // 内存使用率百分比
  disk: number; // 磁盘使用率百分比
  network: {
    in: number; // 网络入流量 (MB)
    out: number; // 网络出流量 (MB)
  };
}

// Kubernetes 资源概览
export interface KubernetesOverview {
  nodes: number; // 节点数量
  pods: number; // Pod总数
  runningPods: number; // 运行中的Pod数量
  services: number; // 服务数量
  deployments: number; // 部署数量
  replicasets: number; // ReplicaSet数量
}

// 服务状态统计
export interface ServiceStatusStats {
  running: number; // 运行中服务数量
  stopped: number; // 已停止服务数量
  error: number; // 错误服务数量
  total: number; // 总服务数量
}

// 最近部署的服务
export interface RecentDeployment {
  id: string;
  name: string;
  image: string;
  replicas: number;
  status: 'running' | 'stopped' | 'error';
  createTime: string;
  updateTime: string;
}

// 仪表盘数据
export interface DashboardData {
  resourceUsage: ResourceUsage;
  kubernetesOverview: KubernetesOverview;
  serviceStatusStats: ServiceStatusStats;
  recentDeployments: RecentDeployment[];
  activeAlerts?: number; // 活跃告警数量
}

// 获取仪表盘数据响应
export interface GetDashboardDataResponse {
  code: number;
  data: DashboardData;
  msg: string;
}
