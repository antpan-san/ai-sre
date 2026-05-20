<template>
  <div class="service-deploy-grid">
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
import { useServiceDeploy } from '../../composables/useServiceDeploy'
import ServiceDeployCard from './ServiceDeployCard.vue'

const route = useRoute()
const { catalog } = useServiceDeploy()

const highlightService = computed(() => String(route.query.service || '').trim())
</script>

<style scoped>
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
