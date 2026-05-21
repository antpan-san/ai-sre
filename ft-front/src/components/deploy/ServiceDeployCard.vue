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
    <el-button class="service-deploy-card__toggle" type="primary" link size="small" @click="expanded = !expanded">
      {{ expanded ? '收起配置' : '展开配置' }}
    </el-button>
    <ServiceDeployConfigBody v-if="expanded" :service-key="item.key" compact />
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

const expanded = ref(Boolean(props.defaultExpanded))

watch(
  () => props.defaultExpanded,
  (v) => {
    expanded.value = Boolean(v)
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
.service-deploy-card__toggle {
  margin-bottom: 8px;
  padding-left: 0;
}
</style>
