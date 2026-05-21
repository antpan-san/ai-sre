<template>
  <div class="deploy-center page-shell page-shell--crud-wide">
    <AppPageHeader
      title="工作负载"
      description="按能力域展示当前账号已开通的工作流入口。"
    >
      <template #actions>
        <el-button size="small" :loading="loading" @click="refresh">刷新</el-button>
        <el-button size="small" @click="router.push('/app/capabilities')">能力中心</el-button>
      </template>
    </AppPageHeader>

    <el-empty
      v-if="!loading && !visibleSections.length"
      class="workload-empty"
      description="暂无已开通工作流，请到能力中心订阅后使用。"
    >
      <el-button type="primary" @click="router.push('/app/capabilities')">前往能力中心</el-button>
    </el-empty>

    <div v-else class="workload-sections">
      <section v-for="section in visibleSections" :key="section.id" class="workload-section">
        <div class="workload-section__head">
          <h3>{{ section.title }}</h3>
          <p>{{ section.description }}</p>
        </div>

        <div v-if="section.groups?.length" class="workload-subgroups">
          <div v-for="group in section.groups" :key="group.id" class="workload-subgroup">
            <div class="workload-subgroup__head">
              <h4>{{ group.title }}</h4>
              <p>{{ group.description }}</p>
            </div>
            <div class="workload-grid">
              <WorkloadTile
                v-for="tile in group.tiles"
                :key="tile.key"
                :title="tile.title"
                :description="tile.description"
                :tags="tile.tags"
                :status="tile.status"
                :status-type="tile.statusType"
                :pack-label="tile.packLabel"
                :accent="tile.accent"
                :detail-path="tile.detailPath"
              />
            </div>
          </div>
        </div>

        <div v-else class="workload-grid">
          <WorkloadTile
            v-for="tile in section.tiles"
            :key="tile.key"
            :title="tile.title"
            :description="tile.description"
            :tags="tile.tags"
            :status="tile.status"
            :status-type="tile.statusType"
            :pack-label="tile.packLabel"
            :accent="tile.accent"
            :detail-path="tile.detailPath"
          />
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import AppPageHeader from '../../components/app/AppPageHeader.vue'
import WorkloadTile from '../../components/workload/WorkloadTile.vue'
import { useCapabilityCatalog, type ResolvedCapability } from '../../composables/useCapabilityCatalog'
import { useServiceDeploy } from '../../composables/useServiceDeploy'
import {
  WORKLOAD_CAPABILITY_META,
  WORKLOAD_DETAIL_CAPABILITY_ROUTE,
  WORKLOAD_DETAIL_SERVICE_LINK,
  WORKLOAD_SERVICE_GROUPS
} from '../../config/workloadPresentation'
import '../../assets/app-workbench.css'

interface TileModel {
  key: string
  title: string
  description: string
  tags?: string[]
  status: string
  statusType: 'success' | 'warning' | 'info'
  packLabel?: string
  accent: string
  detailPath: string
}

interface TileGroup {
  id: string
  title: string
  description: string
  tiles: TileModel[]
}

interface TileSection {
  id: string
  title: string
  description: string
  tiles?: TileModel[]
  groups?: TileGroup[]
}

interface ServiceSummary {
  key: string
  name: string
  description: string
  tags: string[]
}

const INIT_TOOL_GROUPS = [
  {
    id: 'init-preflight',
    title: '环境准备',
    description: '先把时间、内核参数和基础运行环境收敛到可交付状态。',
    items: ['init_time_sync', 'init_sys_param']
  },
  {
    id: 'init-security-storage',
    title: '安全与存储',
    description: '在批量交付前补齐安全基线和磁盘布局。',
    items: ['init_security_hardening', 'init_disk_optimize']
  }
]

const router = useRouter()
const { loading, load: loadCaps, filterCapabilities, isEntitledStatus } = useCapabilityCatalog()
const { catalog: serviceCatalog } = useServiceDeploy()

const deliveryCaps = computed(() => filterCapabilities({ category: 'delivery', status: 'all' }))

const capMap = computed<Record<string, ResolvedCapability>>(() =>
  Object.fromEntries(deliveryCaps.value.map((cap) => [cap.id, cap]))
)

const statusType = (status: string): 'success' | 'warning' | 'info' => {
  if (status === '已订阅' || status === '免费可用' || status === '管理员已开通') return 'success'
  if (status === '未订阅' || status === '联系管理员开通') return 'warning'
  return 'info'
}

const baseTileFromCapability = (cap: ResolvedCapability, metaId = cap.id): TileModel | null => {
  const meta = WORKLOAD_CAPABILITY_META[metaId]
  if (!meta || !isEntitledStatus(cap.status)) return null
  return {
    key: meta.id,
    title: meta.title,
    description: meta.description,
    tags: [],
    status: cap.status,
    statusType: statusType(cap.status),
    accent: meta.accent,
    detailPath: WORKLOAD_DETAIL_CAPABILITY_ROUTE(meta.id)
  }
}

const capabilityTile = (capId: string): TileModel | null => {
  const cap = capMap.value[capId]
  return cap ? baseTileFromCapability(cap, capId) : null
}

const initToolTile = (metaId: string): TileModel | null => {
  const base = capMap.value.init_tools
  return base ? baseTileFromCapability(base, metaId) : null
}

const k8sTiles = computed(() => [capabilityTile('k8s_delivery')].filter(Boolean) as TileModel[])

const mirrorTiles = computed(() => [capabilityTile('k8s_mirror')].filter(Boolean) as TileModel[])

const linuxTiles = computed(() => [capabilityTile('linux_hosts')].filter(Boolean) as TileModel[])

const serviceGroups = computed<TileGroup[]>(() => {
  const base = capMap.value.service_deploy
  if (!base || !isEntitledStatus(base.status)) return []

  const byKey = Object.fromEntries(serviceCatalog.map((item) => [item.key, item])) as Record<string, ServiceSummary | undefined>
  const hasService = (service: ServiceSummary | undefined): service is ServiceSummary => Boolean(service)

  return WORKLOAD_SERVICE_GROUPS.map((group) => {
    const tiles = group.services
      .map((serviceKey) => byKey[serviceKey])
      .filter(hasService)
      .map((service) => ({
        key: service.key,
        title: service.name,
        description: service.description,
        tags: [],
        status: base.status,
        statusType: statusType(base.status),
        accent: group.accent,
        detailPath: WORKLOAD_DETAIL_SERVICE_LINK(service.key)
      }))

    return {
      id: group.id,
      title: group.title,
      description: group.desc,
      tiles
    }
  }).filter((group) => group.tiles.length > 0)
})

const initToolGroups = computed<TileGroup[]>(() => {
  const base = capMap.value.init_tools
  if (!base || !isEntitledStatus(base.status)) return []

  return INIT_TOOL_GROUPS.map((group) => ({
    id: group.id,
    title: group.title,
    description: group.description,
    tiles: group.items
      .map((itemId) => initToolTile(itemId))
      .filter(Boolean) as TileModel[]
  })).filter((group) => group.tiles.length > 0)
})

const visibleSections = computed<TileSection[]>(() => {
  const sections: TileSection[] = []

  if (k8sTiles.value.length) {
    sections.push({
      id: 'k8s-delivery',
      title: 'Kubernetes 交付',
      description: '集群安装、恢复与卸载入口。',
      tiles: k8sTiles.value
    })
  }

  if (serviceGroups.value.length) {
    sections.push({
      id: 'service-deploy',
      title: '应用服务部署',
      description: '按中间件领域直达各服务专页。',
      groups: serviceGroups.value
    })
  }

  if (linuxTiles.value.length) {
    sections.push({
      id: 'linux-hosts',
      title: 'Linux 主机',
      description: '主机查询与 systemd 运维入口。',
      tiles: linuxTiles.value
    })
  }

  if (initToolGroups.value.length) {
    sections.push({
      id: 'init-tools',
      title: '节点初始化',
      description: '环境准备、安全基线与磁盘优化入口。',
      groups: initToolGroups.value
    })
  }

  if (mirrorTiles.value.length) {
    sections.push({
      id: 'k8s-mirror',
      title: 'K8s 制品目录',
      description: '离线制品与 manifest 入口。',
      tiles: mirrorTiles.value
    })
  }

  return sections
})

const refresh = async () => {
  await loadCaps(true)
}

onMounted(async () => {
  await loadCaps()
})
</script>

<style scoped>
.deploy-center {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.workload-empty {
  padding: 36px 0;
  border: 1px dashed var(--el-border-color);
  border-radius: 18px;
  background: var(--el-fill-color-extra-light);
}

.workload-sections {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.workload-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.workload-section__head h3,
.workload-subgroup__head h4 {
  margin: 0;
}

.workload-section__head h3 {
  font-size: 17px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}

.workload-section__head p,
.workload-subgroup__head p {
  margin: 6px 0 0;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}

.workload-section__head p {
  font-size: 12px;
  max-width: 640px;
}

.workload-subgroups {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.workload-subgroup {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 14px;
  border-radius: 18px;
  border: 1px solid var(--el-border-color-lighter);
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.96), rgba(249, 250, 251, 0.92));
}

.workload-subgroup__head h4 {
  font-size: 13px;
  font-weight: 650;
}

.workload-subgroup__head p {
  font-size: 11px;
}

.workload-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  align-items: stretch;
}

@media (max-width: 1180px) {
  .workload-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .workload-grid {
    grid-template-columns: 1fr;
  }

  .workload-subgroup {
    padding: 14px;
  }
}
</style>
