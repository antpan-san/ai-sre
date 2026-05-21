<template>
  <el-card class="service-deploy-card" shadow="hover">
    <template #header>
      <div class="service-deploy-card__head">
        <div>
          <h4 class="service-deploy-card__title">{{ item.name }}</h4>
          <p class="service-deploy-card__desc">{{ item.description }}</p>
        </div>
        <div class="service-deploy-card__tags">
          <el-tag v-for="t in item.tags" :key="t" size="small" type="info" effect="plain">{{ t }}</el-tag>
        </div>
      </div>
    </template>
    <el-collapse v-model="expanded" class="service-deploy-card__collapse">
      <el-collapse-item :name="item.key" :title="expanded.includes(item.key) ? '收起配置' : '展开配置'">
        <ServiceDeployConfigBody :service-key="item.key" compact />
      </el-collapse-item>
    </el-collapse>
  </el-card>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { CatalogItem } from '../../composables/useServiceDeploy'
import ServiceDeployConfigBody from './ServiceDeployConfigBody.vue'

const props = defineProps<{
  item: CatalogItem
  defaultExpanded?: boolean
}>()

const expanded = ref<string[]>(props.defaultExpanded ? [props.item.key] : [])

watch(
  () => props.defaultExpanded,
  (v) => {
    expanded.value = v ? [props.item.key] : []
  }
)
</script>

<style scoped>
.service-deploy-card {
  border-radius: 10px;
  height: 100%;
}
.service-deploy-card__head {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.service-deploy-card__title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
}
.service-deploy-card__desc {
  margin: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.service-deploy-card__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.service-deploy-card__collapse {
  border: none;
}
.service-deploy-card__collapse :deep(.el-collapse-item__header) {
  font-size: 13px;
  color: var(--el-color-primary);
  border-bottom: none;
  height: 36px;
}
.service-deploy-card__collapse :deep(.el-collapse-item__wrap) {
  border-bottom: none;
}
.service-deploy-card__collapse :deep(.el-collapse-item__content) {
  padding-bottom: 0;
}
</style>
