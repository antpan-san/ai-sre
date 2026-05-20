<template>
  <article :class="['zone-card', { 'zone-card--highlight': highlighted, 'zone-card--secondary': item.workload_tier === 'secondary' }]">
    <header class="zone-card__head">
      <div class="zone-card__title-row">
        <el-icon v-if="iconComp" class="zone-card__icon"><component :is="iconComp" /></el-icon>
        <h3>{{ item.name }}</h3>
      </div>
      <SubscriptionBadge :status="item.status" />
    </header>
    <p class="zone-card__desc">{{ item.description }}</p>
    <div class="zone-card__actions">
      <el-tooltip v-if="!item.can_open" content="请先订阅或联系管理员开通" placement="top">
        <span>
          <el-button type="primary" size="small" disabled>打开控制台</el-button>
        </span>
      </el-tooltip>
      <el-button v-else type="primary" size="small" @click="$emit('open', item)">打开控制台</el-button>
      <el-button v-if="item.can_subscribe" type="warning" size="small" link @click="$emit('subscribe', item)">订阅</el-button>
      <el-button v-else-if="!item.can_open" size="small" link disabled>联系管理员</el-button>
      <el-button size="small" link @click="expanded = !expanded">{{ expanded ? '收起' : '更多' }}</el-button>
    </div>
    <el-collapse-transition>
      <div v-show="expanded" class="zone-card__detail">
        <template v-if="item.commands?.length">
          <p class="zone-card__cmd-label">常用 CLI</p>
          <CliCommandBlock
            v-for="cmd in item.commands"
            :key="cmd.template"
            :label="cmd.label"
            :command="cmd.template"
          />
        </template>
        <p v-else-if="item.cli_topic" class="zone-card__cmd-label">CLI 排查</p>
        <CliCommandBlock v-if="item.cli_topic" :command="`ai-sre check ${item.cli_topic}`" />
        <el-link
          v-if="execLink"
          type="primary"
          :underline="false"
          class="zone-card__exec-link"
          @click.prevent="$emit('executions', item)"
        >
          查看相关执行记录
        </el-link>
      </div>
    </el-collapse-transition>
  </article>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import CliCommandBlock from './CliCommandBlock.vue'
import SubscriptionBadge from './SubscriptionBadge.vue'
import type { ResolvedCapability } from '../../composables/useCapabilityCatalog'
import { resolveCatalogIcon } from '../../utils/catalogIcons'

const props = defineProps<{
  item: ResolvedCapability
  highlighted?: boolean
}>()

defineEmits<{ open: [ResolvedCapability]; subscribe: [ResolvedCapability]; executions: [ResolvedCapability] }>()

const expanded = ref(false)
const iconComp = computed(() => resolveCatalogIcon(props.item.icon))

watch(
  () => props.highlighted,
  (v) => {
    if (v) expanded.value = true
  },
  { immediate: true }
)

const execLink = computed(() => Boolean(props.item.execution_source))
</script>

<style scoped>
.zone-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  padding: 16px;
  background: var(--layout-content-surface);
  transition: border-color 0.2s, box-shadow 0.2s;
}
.zone-card--highlight {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.15);
}
.zone-card--secondary {
  min-height: auto;
}
.zone-card__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
}
.zone-card__title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.zone-card__title-row h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}
.zone-card__icon {
  font-size: 20px;
  color: var(--el-color-primary);
}
.zone-card__desc {
  margin: 0 0 12px;
  font-size: 13px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
.zone-card__actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}
.zone-card__detail {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.zone-card__cmd-label {
  margin: 0 0 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.zone-card__exec-link {
  margin-top: 8px;
  font-size: 13px;
}
</style>
