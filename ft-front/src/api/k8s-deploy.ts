import axios from 'axios'
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

/** 将嵌套 DeployConfig 转为后端扁平 K8sDeployRequest（在线部署 / 离线包共用） */
export function buildK8sDeployFlatBody(config: DeployConfig): SubmitDeployConfigParams {
  const { clusterBasicInfo: basic, nodeConfig: nodes, coreComponentsConfig: core,
    networkConfig: net, storageConfig: storage, advancedConfig: adv } = config

  return {
    clusterName: basic.clusterName,
    version: basic.version,
    deployMode: basic.deployMode,
    archVersion: basic.cpuArch,
    // 与后端 normalizeImageSource 一致，避免首尾空格导致未命中阿里云公网下载覆盖
    imageSource: typeof basic.imageSource === 'string' ? basic.imageSource.trim() : basic.imageSource,
    ...(basic.downloadDomain?.trim()
      ? { downloadDomain: basic.downloadDomain.trim() }
      : {}),
    ...(basic.downloadProtocol?.trim()
      ? { downloadProtocol: basic.downloadProtocol.trim() }
      : {}),
    customRegistry: basic.customRegistry,
    registryUsername: basic.registryUsername,
    registryPassword: basic.registryPassword,

    executorNode: nodes.executorNode,
    masterNodes: nodes.masterNodes,
    workerNodes: nodes.workerNodes,
    masterHosts: nodes.masterHosts ?? [],
    workerHosts: nodes.workerHosts ?? [],
    masterLabels: nodes.masterLabels,
    workerLabels: nodes.workerLabels,
    masterTaints: nodes.masterTaints,
    workerTaints: nodes.workerTaints,

    kubeProxyMode: core.kubeProxyMode,
    enableRBAC: core.enableRBAC,
    enablePodSecurityPolicy: core.enablePodSecurityPolicy,
    enableAudit: core.enableAudit,
    auditPolicy: core.auditPolicy,
    pauseImage: core.pauseImage,

    networkPlugin: net.networkPlugin,
    podCidr: net.podCIDR,
    serviceCidr: net.serviceCIDR,
    dnsServiceIP: net.dnsServiceIP,
    clusterDomain: net.clusterDomain,
    calicoConfig: net.calicoConfig as Record<string, unknown> | undefined,
    flannelConfig: net.flannelConfig as Record<string, unknown> | undefined,

    defaultStorageClass: storage.defaultStorageClass,
    storageProvisioner: storage.storageProvisioner,
    storageConfig: {
      localPath: storage.localPathConfig,
      nfs: storage.nfsConfig,
      csi: storage.csiConfig,
    },

    enableNodeLocalDNS: adv.enableNodeLocalDNS,
    enableMetricsServer: adv.enableMetricsServer,
    enableDashboard: adv.enableDashboard,
    enablePrometheus: adv.enablePrometheus,
    enableIngressNginx: adv.enableIngressNginx,
    enableHelm: adv.enableHelm,
    preDeployCleanup: !!adv.preDeployCleanup,
    extraKubeletArgs: adv.extraKubeletArgs,
    extraKubeProxyArgs: adv.extraKubeProxyArgs,
    extraAPIServerArgs: adv.extraAPIServerArgs,
  }
}

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
 * 提交 K8s 集群部署配置（在线：经 Agent 执行 Ansible）。
 */
export const submitDeployConfig = (config: DeployConfig): Promise<SubmitDeployConfigResponse> => {
  return request.post('/api/k8s/deploy/submit', buildK8sDeployFlatBody(config))
}

/**
 * 生成离线 zip 安装包（Ubuntu 24.04 上一键 sudo bash install.sh），浏览器直接下载。
 * 依赖 masterHosts / workerHosts，不经 Agent。
 */
/** 登记离线配置并返回一键安装引用（目标机执行 sudo ai-sre k8s install '<installRef>'） */
export function createK8sBundleInvite(
  config: DeployConfig,
  publicApiBase: string
): Promise<{
  id: string
  expiresAt: string
  installRef: string
  installCommand: string
  /** 无 ai-sre 时：curl 拉引导脚本 + bash + python3 完成拉包与 install.sh */
  bootstrapCommand: string
  /** 部署失败或需重置：按页面 inventory 对全部节点执行 pre_cleanup */
  cleanupCommand: string
}> {
  const body = {
    ...buildK8sDeployFlatBody(config),
    publicApiBase: publicApiBase.replace(/\/$/, '')
  }
  return request.post('/api/k8s/deploy/bundle-invite', body)
}

export async function downloadOfflineBundle(config: DeployConfig): Promise<void> {
  const body = buildK8sDeployFlatBody(config)
  const token = localStorage.getItem('token')
  const base = import.meta.env.VITE_BASE_API || '/ft-api'
  const res = await axios.post(`${base}/api/k8s/deploy/bundle`, body, {
    responseType: 'blob',
    timeout: 300000,
    headers: {
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
  })
  const blob = res.data as Blob
  if (blob.type && blob.type.includes('application/json')) {
    const text = await blob.text()
    try {
      const j = JSON.parse(text) as { msg?: string; message?: string }
      throw new Error(j.msg || j.message || '生成失败')
    } catch (e) {
      if (e instanceof Error && e.message !== 'Unexpected end of JSON input') throw e
      throw new Error(text || '生成失败')
    }
  }
  const cd = res.headers['content-disposition'] as string | undefined
  let name = 'opsfleet-k8s-bundle.zip'
  if (cd) {
    const m = cd.match(/filename\*=UTF-8''([^;]+)|filename="([^"]+)"|filename=([^;\s]+)/)
    const raw = m ? (m[1] || m[2] || m[3]) : ''
    if (raw) {
      try {
        name = decodeURIComponent(raw)
      } catch {
        name = raw
      }
    }
  }
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = name
  a.click()
  URL.revokeObjectURL(url)
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
