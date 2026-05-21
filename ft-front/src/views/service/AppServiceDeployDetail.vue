<template>
  <div class="app-service-detail page-shell page-shell--crud-wide">
    <AppPageHeader :title="pageTitle" :description="pageDescription">
      <template #actions>
        <el-button size="small" @click="router.push('/app/workloads')">返回工作负载</el-button>
      </template>
    </AppPageHeader>

    <el-empty v-if="!service" description="未找到对应服务">
      <el-button type="primary" @click="router.push('/app/workloads')">返回工作负载</el-button>
    </el-empty>

    <template v-else>
      <el-card class="service-summary" shadow="never">
        <div class="service-summary__main">
          <div>
            <p class="service-summary__eyebrow">服务专页</p>
            <h3>{{ service.name }}</h3>
            <p class="service-summary__desc">{{ service.description }}</p>
          </div>
          <div class="service-summary__tags">
            <el-tag v-for="tag in serviceTags" :key="tag" size="small" effect="plain">{{ tag }}</el-tag>
          </div>
        </div>
      </el-card>

      <ServiceDeployConfigBody :service-key="service.key" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, watchEffect } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppPageHeader from '../../components/app/AppPageHeader.vue'
import ServiceDeployConfigBody from '../../components/deploy/ServiceDeployConfigBody.vue'
import { useServiceDeploy } from '../../composables/useServiceDeploy'

const route = useRoute()
const router = useRouter()
const { catalog } = useServiceDeploy()

const serviceKey = computed(() => String(route.params.serviceKey || '').trim())
const service = computed(() => catalog.find((item) => item.key === serviceKey.value) || null)
const pageTitle = computed(() => service.value?.name || '服务部署')
const pageDescription = computed(() => service.value?.description || '当前服务的独立部署页面。')
const serviceTags = computed(() => (Array.isArray(service.value?.tags) ? service.value.tags : []).filter(Boolean))

watchEffect(() => {
  document.title = pageTitle.value
})
</script>

<style scoped>
.app-service-detail {
  display: flex;
  flex-direction: column;
}

.service-summary {
  margin-bottom: 16px;
  border-radius: 16px;
  border: 1px solid var(--el-border-color-lighter);
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.98), rgba(248, 250, 252, 0.96));
}

.service-summary__main {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
}

.service-summary__eyebrow {
  margin: 0 0 6px;
  color: var(--el-color-primary);
  font-size: 11px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.service-summary h3 {
  margin: 0;
  font-size: 22px;
}

.service-summary__desc {
  margin: 6px 0 0;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}

.service-summary__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  justify-content: flex-end;
}

@media (max-width: 720px) {
  .service-summary__main {
    flex-direction: column;
  }

  .service-summary__tags {
    justify-content: flex-start;
  }
}
</style>
