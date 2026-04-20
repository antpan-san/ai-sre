import request from '../utils/request'

export interface K8sMirrorFileRow {
  relativePath: string
  sizeBytes: number
  sha512: string
  downloadUrl: string
}

export interface K8sMirrorCatalog {
  generatedAt?: string
  mirrorRoot?: string
  publicBaseUrl?: string
  files?: K8sMirrorFileRow[]
  fetchError?: string
  manifestUrl?: string
}

/** 拉取制品站 manifest（后端代理，避免浏览器直连 CORS） */
export const getK8sMirrorCatalog = (): Promise<K8sMirrorCatalog> => {
  return request.get('/api/k8s/mirror/catalog', { timeout: 120000 })
}
