import request from '../utils/request'
import type {
  GetK8sVersionsParams,
  GetK8sVersionsResponse,
  GetMachinesParams,
  GetMachinesResponse,
  CheckClusterNameParams,
  CheckClusterNameResponse,
  SubmitDeployConfigParams,
  SubmitDeployConfigResponse,
  GetDeployProgressParams,
  GetDeployProgressResponse,
  GetDeployLogsParams,
  GetDeployLogsResponse,
  GetDeployRecordsResponse,
  DeployConfig,
} from '../types/k8s-deploy'

/** 获取支持的 K8s 版本列表 */
export const getK8sVersions = (params?: GetK8sVersionsParams): Promise<GetK8sVersionsResponse> => {
  return request.get('/api/k8s/deploy/versions', { params })
}

/** 获取可用机器列表 */
export const getMachines = (params?: GetMachinesParams): Promise<GetMachinesResponse> => {
  return request.get('/api/k8s/deploy/machines', { params })
}

/** 校验集群名称唯一性 */
export const checkClusterName = (params: CheckClusterNameParams): Promise<CheckClusterNameResponse> => {
  return request.get('/api/k8s/deploy/check-name', { params })
}

/**
 * 提交 K8s 集群部署配置。
 * 将前端嵌套的 DeployConfig 转换为后端期望的扁平 K8sDeployRequest 格式，
 * 包含 7 个步骤的全量配置，供后端生成完整 Ansible 脚本。
 */
export const submitDeployConfig = (config: DeployConfig): Promise<SubmitDeployConfigResponse> => {
  const { clusterBasicInfo: basic, nodeConfig: nodes, coreComponentsConfig: core,
          networkConfig: net, storageConfig: storage, advancedConfig: adv } = config

  const body: SubmitDeployConfigParams = {
    // Step 1
    clusterName:        basic.clusterName,
    version:            basic.version,
    deployMode:         basic.deployMode,
    imageSource:        basic.imageSource,
    customRegistry:     basic.customRegistry,
    registryUsername:   basic.registryUsername,
    registryPassword:   basic.registryPassword,

    // Step 2
    executorNode:  nodes.executorNode,
    masterNodes:   nodes.masterNodes,
    workerNodes:   nodes.workerNodes,
    masterLabels:  nodes.masterLabels,
    workerLabels:  nodes.workerLabels,
    masterTaints:  nodes.masterTaints,
    workerTaints:  nodes.workerTaints,

    // Step 3
    kubeProxyMode:            core.kubeProxyMode,
    enableRBAC:               core.enableRBAC,
    enablePodSecurityPolicy:  core.enablePodSecurityPolicy,
    enableAudit:              core.enableAudit,
    auditPolicy:              core.auditPolicy,
    pauseImage:               core.pauseImage,

    // Step 4
    networkPlugin: net.networkPlugin,
    podCidr:       net.podCIDR,
    serviceCidr:   net.serviceCIDR,
    dnsServiceIP:  net.dnsServiceIP,
    clusterDomain: net.clusterDomain,
    calicoConfig:  net.calicoConfig  as Record<string, unknown> | undefined,
    flannelConfig: net.flannelConfig as Record<string, unknown> | undefined,

    // Step 5
    defaultStorageClass:  storage.defaultStorageClass,
    storageProvisioner:   storage.storageProvisioner,
    storageConfig: {
      localPath: storage.localPathConfig,
      nfs:       storage.nfsConfig,
      csi:       storage.csiConfig,
    },

    // Step 6
    enableNodeLocalDNS:  adv.enableNodeLocalDNS,
    enableMetricsServer: adv.enableMetricsServer,
    enableDashboard:     adv.enableDashboard,
    enablePrometheus:    adv.enablePrometheus,
    enableIngressNginx:  adv.enableIngressNginx,
    enableHelm:          adv.enableHelm,
    extraKubeletArgs:    adv.extraKubeletArgs,
    extraKubeProxyArgs:  adv.extraKubeProxyArgs,
    extraAPIServerArgs:  adv.extraAPIServerArgs,
  }
  return request.post('/api/k8s/deploy/submit', body)
}

/** 获取 K8s 集群部署进度 */
export const getDeployProgress = (params: GetDeployProgressParams): Promise<GetDeployProgressResponse> => {
  return request.get('/api/k8s/deploy/progress', { params })
}

/** 获取 K8s 集群部署日志 */
export const getDeployLogs = (params: GetDeployLogsParams): Promise<GetDeployLogsResponse> => {
  return request.get('/api/k8s/deploy/logs', { params })
}

/** 获取 K8s 部署记录列表（用于第一步展示部署记录与正在部署） */
export const getDeployRecords = (): Promise<GetDeployRecordsResponse> => {
  return request.get('/api/k8s/deploy/records') as Promise<GetDeployRecordsResponse>
}

/** 终止 K8s 部署并下发清理任务，使 client 端恢复到部署前状态 */
export const terminateDeploy = (deployId: string): Promise<void> => {
  return request.post('/api/k8s/deploy/terminate', { deployId })
}

/** 获取已部署的 K8s 集群列表（与后端 /api/k8s/clusters 路由对齐） */
export const getClusterList = (): Promise<any> => {
  return request.get('/api/k8s/clusters')
}
