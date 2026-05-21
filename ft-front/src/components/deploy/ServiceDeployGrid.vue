<template>
  <div v-if="grouped" class="service-deploy-groups">
    <section v-for="group in groupedCatalog" :key="group.key" class="service-deploy-group">
      <div class="service-deploy-group__head">
        <h4>{{ group.title }}</h4>
        <p>{{ group.desc }}</p>
      </div>
      <div class="service-deploy-grid">
        <ServiceDeployCard
          v-for="item in group.items"
          :key="item.key"
          :item="item"
          :default-expanded="highlightService === item.key"
        />
      </div>
    </section>
  </div>

  <div v-else class="service-deploy-grid">
    <ServiceDeployCard
      v-for="item in catalog"
      :key="item.key"
      :item="item"
      :default-expanded="highlightService === item.key"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useServiceDeploy, type CatalogItem } from '../../composables/useServiceDeploy'
import ServiceDeployCard from './ServiceDeployCard.vue'

withDefaults(defineProps<{
  grouped?: boolean
}>(), {
  grouped: false
})

const route = useRoute()
const { catalog } = useServiceDeploy()

const highlightService = computed(() => String(route.query.service || '').trim())

const SERVICE_GROUPS = [
  { key: 'gateway', title: '网关与负载', desc: 'Web 网关、反向代理与四/七层负载均衡。', services: ['nginx', 'haproxy'] },
  { key: 'cache-mq', title: '缓存与消息', desc: '缓存、键值存储与消息队列基础组件。', services: ['redis', 'kafka'] },
  { key: 'database', title: '数据库', desc: '关系型数据库单机部署与基础参数生成。', services: ['mysql', 'postgresql'] },
  { key: 'search', title: '搜索与分析', desc: '搜索引擎与分析组件部署配置。', services: ['elasticsearch'] },
]

const byKey = computed<Record<string, CatalogItem>>(() =>
  Object.fromEntries(catalog.map((item) => [item.key, item]))
)

const hasItem = (item: CatalogItem | undefined): item is CatalogItem => Boolean(item)

const groupedCatalog = computed(() =>
  SERVICE_GROUPS.map((group) => ({
    ...group,
    items: group.services.map((key) => byKey.value[key]).filter(hasItem),
  })).filter((group) => group.items.length)
)
</script>

<style scoped>
.service-deploy-groups {
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.service-deploy-group {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.service-deploy-group__head h4 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.service-deploy-group__head p {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.service-deploy-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
@media (max-width: 960px) {
  .service-deploy-grid {
    grid-template-columns: 1fr;
  }
}
</style>
