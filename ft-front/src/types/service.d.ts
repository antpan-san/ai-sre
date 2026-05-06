// Service Management Type Definitions

// 环境变量类型
export interface EnvVar {
  key: string
  value: string
}

// 数据卷类型
export interface Volume {
  name: string
  mountPath: string
  hostPath: string
}

// 服务部署请求参数
export interface DeployServiceParams {
  name: string
  image: string
  replicas: number
  port: number
  description?: string
  type?: 'docker' | 'k8s' | 'linux'
  config?: Record<string, any>
}

// 服务部署响应
export interface DeployServiceResponse {
  code: number
  data: {
    deploymentId: string
    serviceId: string
  }
  msg: string
}

export interface CreateServiceDeploymentParams {
  service: string
  profile: string
  install_method: string
  version?: string
  params: Record<string, any>
}

export interface CreateServiceDeploymentResponse {
  deploymentId: string
  token: string
  curlCommand: string
  aiSreCommand: string
  status: string
}

// 服务列表请求参数
export interface GetServiceListParams {
  page?: number
  pageSize?: number
  name?: string
  status?: 'running' | 'stopped' | 'error'
  startTime?: string
  endTime?: string
}

// 服务信息类型
export interface ServiceInfo {
  id: string
  name: string
  image: string
  replicas: number
  desiredReplicas: number
  availableReplicas: number
  port: number
  status: 'running' | 'stopped' | 'error'
  createTime: string
  updateTime: string
  env?: Record<string, string>
  volume?: Volume[]
}

// 服务列表响应
export interface GetServiceListResponse {
  code: number
  data: {
    list: ServiceInfo[]
    total: number
  }
  msg: string
}

// 服务详情请求参数
export interface GetServiceDetailParams {
  serviceId: string
}

// 服务详情响应
export interface GetServiceDetailResponse {
  code: number
  data: ServiceInfo
  msg: string
}

// 服务操作请求参数
export interface ServiceOperationParams {
  serviceId: string
}

// 服务操作响应
export interface ServiceOperationResponse {
  code: number
  data: null
  msg: string
}

// 批量删除服务请求参数
export interface BatchDeleteServicesParams {
  serviceIds: string[]
}

// 批量删除服务响应
export interface BatchDeleteServicesResponse {
  code: number
  data: null
  msg: string
}

// Linux服务信息类型
export interface LinuxServiceInfo {
  id: string
  name: string
  status: 'active' | 'inactive' | 'failed'
  description: string
  machineId: string
  machineName: string
  pid?: number
  startCmd: string
  createTime: string
  updateTime: string
}

// Linux服务列表请求参数
export interface GetLinuxServiceListParams {
  page?: number
  pageSize?: number
  name?: string
  status?: 'active' | 'inactive' | 'failed'
  machineId?: string
}

// Linux服务列表响应
export interface GetLinuxServiceListResponse {
  code: number
  data: {
    list: LinuxServiceInfo[]
    total: number
  }
  msg: string
}

// Linux服务操作请求参数
export interface LinuxServiceOperationParams {
  serviceId: string
  operation: 'start' | 'stop' | 'restart' | 'enable' | 'disable'
}

// Linux服务操作响应
export interface LinuxServiceOperationResponse {
  code: number
  data: null
  msg: string
}