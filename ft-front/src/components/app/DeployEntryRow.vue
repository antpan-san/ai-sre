<template>
  <article :class="['deploy-row', { 'deploy-row--highlight': highlighted }]">
    <div class="deploy-row__main">
      <div class="deploy-row__copy">
        <div class="deploy-row__title-line">
          <h3 class="deploy-row__title">{{ item.name }}</h3>
        </div>
        <p class="deploy-row__desc">{{ item.description }}</p>
      </div>
      <div class="deploy-row__action">
        <template v-if="mode === 'entitled'">
          <el-tooltip v-if="!item.can_open" content="请先订阅或联系管理员开通" placement="top">
            <span>
              <el-button type="primary" size="small" disabled>进入</el-button>
            </span>
          </el-tooltip>
          <el-button v-else type="primary" size="small" @click="$emit('open', item)">进入</el-button>
        </template>
        <template v-else>
          <el-button v-if="item.can_subscribe" type="warning" size="small" @click="$emit('subscribe', item)">
            订阅
          </el-button>
          <el-button v-else type="info" size="small" @click="$emit('contact-admin', item)">联系管理员</el-button>
        </template>
      </div>
    </div>
    <div v-if="cliLines.length" class="deploy-row__cli">
      <CliCommandBlock v-for="line in cliLines" :key="line" :command="line" />
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import CliCommandBlock from './CliCommandBlock.vue'
import type { ResolvedCapability } from '../../composables/useCapabilityCatalog'

const props = defineProps<{
  item: ResolvedCapability
  mode: 'entitled' | 'subscribe'
  highlighted?: boolean
}>()

defineEmits<{
  open: [ResolvedCapability]
  subscribe: [ResolvedCapability]
  'contact-admin': [ResolvedCapability]
}>()

const cliLines = computed(() => {
  const cmds = props.item.commands?.map((c) => c.template) || []
  if (cmds.length) return cmds
  if (props.item.cli_topic) return [`ai-sre check ${props.item.cli_topic}`]
  return []
})
</script>

<style scoped>
.deploy-row {
  padding: 14px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.deploy-row--highlight {
  padding-left: 10px;
  border-left: 3px solid var(--el-color-primary);
}
.deploy-row__main {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}
.deploy-row__copy {
  flex: 1;
  min-width: 0;
}
.deploy-row__title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
}
.deploy-row__desc {
  margin: 4px 0 0;
  font-size: 13px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
.deploy-row__action {
  flex-shrink: 0;
}
.deploy-row__cli {
  margin-top: 10px;
}
.deploy-row__cli :deep(.cli-block) {
  margin-bottom: 6px;
}
.deploy-row__cli :deep(.cli-block:last-child) {
  margin-bottom: 0;
}
@media (max-width: 640px) {
  .deploy-row__main {
    flex-direction: column;
    align-items: stretch;
  }
  .deploy-row__action {
    align-self: flex-start;
  }
}
</style>
