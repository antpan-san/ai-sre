// API响应通用类型
export interface ApiResponse<T = any> {
  code: number
  data: T
  msg: string
}

// 节点角色
export type NodeRole = 'master' | 'worker' | 'standalone'

// 机器相关类型
export interface Machine {
  id: string
  name: string
  ip: string
  cpu: number
  memory: number
  disk: number
  status: 'online' | 'offline' | 'maintenance'
  created_at: string
  updated_at: string
  // 集群拓扑字段
  node_role: NodeRole
  cluster_id?: string | null
  master_machine_id?: string | null
  // 身份标识
  client_id?: string
  host_fingerprint?: string
  // 扩展信息
  labels?: Record<string, string>
  metadata?: Record<string, any>
  owner_user_id?: string | null
  last_heartbeat_at?: string | null
  // 实时指标 (WebSocket 推送)
  os_version?: string
  kernel_version?: string
  cpu_cores?: number
  cpu_usage?: number
  memory_total?: number
  memory_used?: number
  memory_usage?: number
  disk_total?: number
  disk_used?: number
  disk_usage?: number
}

// WebSocket 推送的机器心跳数据
export interface MachineHeartbeatData {
  client_id: string
  ip: string
  hostname: string
  status: string
  node_role: string
  cluster_id: string
  os_version: string
  kernel_version: string
  cpu_cores: number
  cpu_usage: number
  memory_total: number
  memory_used: number
  memory_usage: number
  disk_total: number
  disk_used: number
  disk_usage: number
  workers?: WorkerStatusData[]
}

export interface WorkerStatusData {
  ip: string
  hostname: string
  status: string
  probe_error?: string
  os_version: string
  kernel_version: string
  cpu_cores: number
  cpu_usage: number
  memory_total: number
  memory_used: number
  memory_usage: number
  disk_total: number
  disk_used: number
  disk_usage: number
}

export interface MachineStatusUpdateData {
  id?: string
  client_id?: string
  ip: string
  status: string
  last_heartbeat_at?: string | null
}

// SSH 认证方式
export type SSHAuthType = 'password' | 'key'

// 注册受控节点表单
export interface RegisterWorkerNode {
  ip: string
  hostname: string
  ssh_port: number
  ssh_user: string
  auth_type: SSHAuthType
  ssh_password: string
  ssh_key: string
}

export interface RegisterWorkersResponse {
  task_id: string
  workers_created: number
}

// 树形展示节点（master 含 children）
export interface MachineTreeNode extends Machine {
  children?: MachineTreeNode[]
}

export interface MachineListParams {
  page?: number
  pageSize?: number
  name?: string
  status?: string
  startDate?: string
  endDate?: string
  cluster_id?: string
  node_role?: NodeRole
}

export interface MachineListResponse {
  list: Machine[]
  total: number
}

export interface MachineForm {
  id?: string
  name: string
  ip: string
  cpu: number
  memory: number
  disk: number
  status: 'online' | 'offline' | 'maintenance'
  node_role: NodeRole
  cluster_id?: string | null
  master_machine_id?: string | null
}

// 用户相关类型
export interface User {
  id: number
  username: string
  phone: string
  role: 'admin' | 'user'
  createTime: string
  updateTime: string
}

export interface UserListParams {
  page?: number
  pageSize?: number
  username?: string
  role?: string
}

export interface UserListResponse {
  list: User[]
  total: number
}

export interface UserForm {
  id?: number
  username: string
  password?: string
  phone: string
  role: 'admin' | 'user'
}

// 登录相关类型
export interface LoginForm {
  username: string
  password: string
  remember: boolean
}

export interface LoginResponse {
  token: string
  user: User
}

// 服务部署相关类型
export interface DeployForm {
  name: string
  image: string
  replicas: number
  port: number
  env?: Record<string, string>
  volume?: Array<{
    name: string
    mountPath: string
    hostPath?: string
  }>
}

// 安全与审计模块 - 操作日志相关类型
export interface OperationLog {
  id: number
  username: string
  operation: string
  resource: string
  resourceId: number
  ip: string
  userAgent: string
  status: 'success' | 'fail'
  errorMessage?: string
  createTime: string
}

export interface OperationLogParams {
  page?: number
  pageSize?: number
  username?: string
  operation?: string
  resource?: string
  status?: string
  startDate?: string
  endDate?: string
}

export interface OperationLogListResponse {
  list: OperationLog[]
  total: number
}

// 安全与审计模块 - 权限管理相关类型
export interface Permission {
  id: number
  name: string
  code: string
  description?: string
  createTime: string
  updateTime: string
}

export interface PermissionListParams {
  page?: number
  pageSize?: number
  name?: string
  code?: string
}

export interface PermissionListResponse {
  list: Permission[]
  total: number
}

// 高级功能模块 - 备份与恢复相关类型
export interface Backup {
  id: number
  name: string
  description?: string
  size: number
  status: 'completed' | 'running' | 'failed'
  createTime: string
  updateTime: string
  backupTime: string
}

export interface BackupParams {
  page?: number
  pageSize?: number
  name?: string
  status?: string
  startDate?: string
  endDate?: string
}

export interface BackupListResponse {
  list: Backup[]
  total: number
}

// 高级功能模块 - 性能分析相关类型
export interface PerformanceData {
  id: number
  machineId: number
  machineName: string
  cpuUsage: number
  memoryUsage: number
  diskUsage: number
  networkIn: number
  networkOut: number
  timestamp: string
}

export interface PerformanceParams {
  machineId?: number
  startTime?: string
  endTime?: string
  interval?: string
  metrics?: Array<'cpu' | 'memory' | 'disk' | 'network'>
}

export interface PerformanceDataResponse {
  data: PerformanceData[]
  metrics: string[]
  machines: Array<{ id: number; name: string }>
}

export interface PerformanceReport {
  id: number
  name: string
  description?: string
  startTime: string
  endTime: string
  reportData: {
    cpu: { average: number; max: number; min: number }
    memory: { average: number; max: number; min: number }
    disk: { average: number; max: number; min: number }
    network: { in: { average: number }; out: { average: number } }
  }
  createTime: string
}
