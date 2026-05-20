<template>
  <article class="pack-card">
    <header class="pack-card__head">
      <h3 class="pack-card__title">{{ pack.display_name }}</h3>
      <SubscriptionBadge :status="pack.status" />
    </header>
    <p class="pack-card__meta">{{ pack.capabilities.length }} 项能力</p>
    <ul class="pack-card__list">
      <li v-for="cap in pack.capabilities.slice(0, 4)" :key="cap.id">{{ cap.name }}</li>
      <li v-if="pack.capabilities.length > 4" class="pack-card__more">+{{ pack.capabilities.length - 4 }} 更多</li>
    </ul>
    <footer class="pack-card__actions">
      <el-button v-if="pack.entitled" type="primary" size="small" @click="$emit('manage', pack)">管理能力</el-button>
      <el-button v-else-if="pack.can_subscribe" type="warning" size="small" @click="$emit('subscribe', pack)">订阅整包</el-button>
      <el-button v-else size="small" disabled>联系管理员</el-button>
    </footer>
  </article>
</template>

<script setup lang="ts">
import SubscriptionBadge from './SubscriptionBadge.vue'
import type { PackWithCapabilities } from '../../composables/useCapabilityCatalog'

defineProps<{ pack: PackWithCapabilities }>()
defineEmits<{ manage: [PackWithCapabilities]; subscribe: [PackWithCapabilities] }>()
</script>

<style scoped>
.pack-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  padding: 14px;
  background: var(--layout-content-surface);
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-height: 160px;
}
.pack-card__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}
.pack-card__title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
}
.pack-card__meta {
  margin: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.pack-card__list {
  margin: 0;
  padding-left: 16px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-regular);
  flex: 1;
}
.pack-card__more {
  list-style: none;
  margin-left: -16px;
  color: var(--el-text-color-secondary);
}
.pack-card__actions {
  margin-top: auto;
}
</style>
