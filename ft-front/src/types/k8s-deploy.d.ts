// K8s版本信息（与后端 K8sVersionDTO 对齐）
export interface K8sVersion {
  id: string;
  version: string;
  description: string;
  recommended: boolean;
  is_active: boolean;
}

// 机器信息（与后端 Machine 模型对齐，id 为 UUID 字符串）
export interface K8sMachineInfo {
  id: string;        // UUID string
  name: string;
  ip: string;
  cpu: number;
  memory: number;
  disk: number;
  status: 'online' | 'offline' | 'maintenance';
  node_role?: 'master' | 'worker' | 'standalone';
  client_id?: string;
  labels?: Record<string, string>;
  taints?: Taint[];
}

// 污点信息
export interface Taint {
  key: string;
  value: string;
  effect: 'NoSchedule' | 'PreferNoSchedule' | 'NoExecute';
}

// 键值对
export interface KeyValuePair {
  key: string;
  value: string;
}

// 基础集群信息
export interface ClusterBasicInfo {
  clusterName: string;
  version: string;
  deployMode: 'single' | 'cluster';
  /** 目标节点 CPU 架构（与 kubernetes/etcd linux 二进制一致；Fusion ARM 虚机选 arm64） */
  cpuArch: 'amd64' | 'arm64';
  imageSource: 'default' | 'aliyun' | 'tencent' | 'custom';
  /** 覆盖 ansible download_domain（内网制品机）；留空使用 inventory/group_vars/all.yml 默认 */
  downloadDomain?: string;
  /** 覆盖 download_protocol，如 http:// 或 https:// */
  downloadProtocol?: string;
  customRegistry?: string;
  registryUsername?: string;
  registryPassword?: string;
}

// 节点配置（executorNode 为执行部署的 Agent 节点，masterNodes/workerNodes 为 K8s 集群节点）
export interface NodeConfig {
  executorNode?: string;   // 在线部署：Agent 所在机器 UUID（离线包模式可不填）
  masterNodes: string[];
  workerNodes: string[];
  /** 离线安装包：节点 IP/主机名（与 masterNodes 二选一为主流程） */
  masterHosts?: string[];
  workerHosts?: string[];
  masterLabels?: Record<string, string>;
  workerLabels?: Record<string, string>;
  masterTaints?: Taint[];
  workerTaints?: Taint[];
}

// 核心组件配置
export interface CoreComponentsConfig {
  etcdVersion?: string;
  apiServerVersion?: string;
  controllerManagerVersion?: string;
  schedulerVersion?: string;
  pauseImage?: string;
  kubeProxyMode: 'iptables' | 'ipvs';
  enablePodSecurityPolicy: boolean;
  enableRBAC: boolean;
  enableAudit: boolean;
  auditPolicy?: string;
}

// 网络配置
export interface NetworkConfig {
  networkPlugin: 'calico' | 'flannel' | 'cilium' | 'weave';
  podCIDR: string;
  serviceCIDR: string;
  dnsServiceIP: string;
  clusterDomain: string;
  proxyMode: 'iptables' | 'ipvs';
  calicoConfig?: {
    vxlanMode: boolean;
    mtu: number;
    ipipMode?: boolean;
  };
  flannelConfig?: {
    backend: 'vxlan' | 'host-gw' | 'udp';
  };
}

// 存储配置
export interface StorageConfig {
  defaultStorageClass: boolean;
  storageProvisioner: 'local-path' | 'nfs-client' | 'csi';
  localPathConfig?: {
    path: string;
  };
  nfsConfig?: {
    server: string;
    path: string;
  };
  csiConfig?: {
    driver: string;
    controllerCount: number;
  };
}

// 高级配置
export interface AdvancedConfig {
  enableNodeLocalDNS: boolean;
  enableMetricsServer: boolean;
  enableDashboard: boolean;
  enablePrometheus: boolean;
  enableIngressNginx: boolean;
  enableHelm: boolean;
  /** 离线 install.sh / 在线部署脚本：是否在 Step 0 非交互执行预清理（停旧服务、删 etcd/K8s 数据等） */
  preDeployCleanup?: boolean;
  extraKubeletArgs?: KeyValuePair[];
  extraKubeProxyArgs?: KeyValuePair[];
  extraAPIServerArgs?: KeyValuePair[];
}

// 完整部署配置（前端表单使用）
export interface DeployConfig {
  clusterBasicInfo: ClusterBasicInfo;
  nodeConfig: NodeConfig;
  coreComponentsConfig: CoreComponentsConfig;
  networkConfig: NetworkConfig;
  storageConfig: StorageConfig;
  advancedConfig: AdvancedConfig;
}

// 提交到后端的请求体（与后端 K8sDeployRequest 完整对齐）
export interface K8sDeploySubmitRequest {
  // Step 1 — Basic
  clusterName: string;
  version: string;
  deployMode: string;
  /** amd64 | arm64 */
  archVersion?: string;
  imageSource: string;
  downloadDomain?: string;
  downloadProtocol?: string;
  customRegistry?: string;
  registryUsername?: string;
  registryPassword?: string;

  // Step 2 — Nodes
  executorNode?: string;   // 执行部署的 Agent 节点 UUID（可选）
  masterNodes: string[];   // UUIDs
  workerNodes: string[];
  masterHosts?: string[];
  workerHosts?: string[];
  masterLabels?: Record<string, string>;
  workerLabels?: Record<string, string>;
  masterTaints?: Taint[];
  workerTaints?: Taint[];

  // Step 3 — Core components
  kubeProxyMode?: string;
  enableRBAC?: boolean;
  enablePodSecurityPolicy?: boolean;
  enableAudit?: boolean;
  auditPolicy?: string;
  pauseImage?: string;

  // Step 4 — Network
  networkPlugin: string;
  podCidr: string;
  serviceCidr: string;
  dnsServiceIP?: string;
  clusterDomain?: string;
  calicoConfig?: Record<string, unknown>;
  flannelConfig?: Record<string, unknown>;

  // Step 5 — Storage
  storageProvisioner?: string;
  defaultStorageClass?: boolean;
  storageConfig?: Record<string, unknown>;

  // Step 6 — Advanced
  enableNodeLocalDNS?: boolean;
  enableMetricsServer?: boolean;
  enableDashboard?: boolean;
  enablePrometheus?: boolean;
  enableIngressNginx?: boolean;
  enableHelm?: boolean;
  preDeployCleanup?: boolean;
  extraKubeletArgs?: KeyValuePair[];
  extraKubeProxyArgs?: KeyValuePair[];
  extraAPIServerArgs?: KeyValuePair[];
}

// 部署进度（与后端 DeployProgressDTO 对齐）
export interface DeployProgress {
  deployId?: string;
  progress: number;
  status: 'pending' | 'running' | 'success' | 'failed' | 'cancelled';
  currentStep: string;
  stepProgress: number;
  startTime?: string;
  endTime?: string;
  error?: string;
  totalCount?: number;
  successCount?: number;
  failedCount?: number;
}

// 部署记录（列表项，与后端 GetK8sDeployRecords 对齐）
export interface DeployRecord {
  deployId: string;
  clusterName: string;
  status: string;
  progress: number;
  currentStep: string;
  stepProgress: number;
  startTime?: string;
  endTime?: string;
  error?: string;
  createdAt: string;
}

// 部署日志（与后端 DeployLogDTO 对齐）
export interface DeployLog {
  timestamp: string;
  level: 'info' | 'warning' | 'error';
  message: string;
  step?: string;
}

// ---- API 请求参数 ----
export interface GetK8sVersionsParams {}
export interface GetMachinesParams {
  status?: string;
  keyword?: string;
  minCpu?: number;
  minMemory?: number;
}
export interface CheckClusterNameParams {
  clusterName: string;
}
export type SubmitDeployConfigParams = K8sDeploySubmitRequest;
export interface GetDeployProgressParams {
  deployId: string;
}
export interface GetDeployLogsParams {
  deployId: string;
  offset?: number;
  limit?: number;
}

// ---- API 响应类型 ----
export type GetK8sVersionsResponse = K8sVersion[];
export type GetMachinesResponse = K8sMachineInfo[];
export type GetDeployRecordsResponse = DeployRecord[];
export interface CheckClusterNameResponse {
  isAvailable: boolean;
}
export interface SubmitDeployConfigResponse {
  deployId: string;
}
export type GetDeployProgressResponse = DeployProgress;
export interface GetDeployLogsResponse {
  logs: DeployLog[];
  total: number;
  hasMore: boolean;
}
