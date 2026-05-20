<template>
  <section class="workload-panel">
    <div class="workload-panel__head">
      <div>
        <h3>{{ title }}</h3>
        <p>{{ description }}</p>
      </div>
      <el-tag v-if="cap" size="small" :type="cap.status === '已订阅' || cap.status === '免费可用' ? 'success' : 'warning'">
        {{ cap?.status || '—' }}
      </el-tag>
    </div>
    <div v-if="commands?.length" class="workload-panel__cmds">
      <p class="cmd-label">常用 CLI</p>
      <pre v-for="(cmd, i) in commands" :key="i" class="cmd-line">{{ cmd }}</pre>
    </div>
    <div class="workload-panel__actions">
      <el-button v-if="cap?.can_open" type="primary" @click="$emit('open')">打开</el-button>
      <el-button v-else-if="cap?.can_subscribe" type="warning" @click="cap && $emit('subscribe', cap)">订阅</el-button>
      <el-button v-else disabled>联系管理员开通</el-button>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { ResolvedCapability } from '../../composables/useCapabilityCatalog'

defineProps<{
  title: string
  description: string
  cap?: ResolvedCapability
  commands?: string[]
}>()

defineEmits<{ open: []; subscribe: [ResolvedCapability] }>()
</script>

<style scoped>
.workload-panel {
  padding: 8px 4px 24px;
  max-width: 720px;
}
.workload-panel__head {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}
.workload-panel__head h3 {
  margin: 0 0 6px;
  font-size: 17px;
}
.workload-panel__head p {
  margin: 0;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.cmd-label {
  margin: 0 0 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.cmd-line {
  margin: 0 0 8px;
  padding: 10px 12px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
  font-size: 12px;
  overflow-x: auto;
}
.workload-panel__actions {
  margin-top: 16px;
}
</style>
