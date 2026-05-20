<template>
  <article class="cap-card">
    <header class="cap-card__head">
      <h3 class="cap-card__title">{{ item.name }}</h3>
      <el-tag size="small" :type="statusTagType">{{ item.status }}</el-tag>
    </header>
    <p class="cap-card__desc">{{ item.description }}</p>
    <p v-if="item.pack_key" class="cap-card__pack">功能包：{{ item.pack_display_name || item.pack_key }}</p>
    <p v-if="item.cli_topic" class="cap-card__cli">CLI：<code>ai-sre check {{ item.cli_topic }}</code></p>
    <footer class="cap-card__actions">
      <el-button v-if="item.can_open" type="primary" size="small" @click="$emit('open', item)">打开</el-button>
      <el-button v-else-if="item.can_subscribe" type="warning" size="small" @click="$emit('subscribe', item)">订阅</el-button>
      <el-button v-else size="small" disabled>联系管理员</el-button>
    </footer>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ResolvedCapability } from '../../composables/useCapabilityCatalog'

const props = defineProps<{ item: ResolvedCapability }>()
defineEmits<{ open: [ResolvedCapability]; subscribe: [ResolvedCapability] }>()

const statusTagType = computed(() => {
  switch (props.item.status) {
    case '已订阅':
    case '管理员已开通':
    case '免费可用':
      return 'success'
    case '未订阅':
      return 'warning'
    case '联系管理员开通':
      return 'info'
    default:
      return 'info'
  }
})
</script>

<style scoped>
.cap-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  padding: 16px;
  background: var(--layout-content-surface);
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-height: 168px;
}
.cap-card__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}
.cap-card__title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
}
.cap-card__desc {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
  flex: 1;
}
.cap-card__pack,
.cap-card__cli {
  margin: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.cap-card__cli code {
  font-size: 12px;
}
.cap-card__actions {
  margin-top: auto;
  padding-top: 4px;
}
</style>
