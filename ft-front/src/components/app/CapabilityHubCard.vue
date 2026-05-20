<template>
  <article :class="['hub-card', { 'hub-card--highlight': highlighted }]">
    <header class="hub-card__head">
      <div class="hub-card__title-row">
        <el-icon v-if="iconComp" class="hub-card__icon"><component :is="iconComp" /></el-icon>
        <h3>{{ item.name }}</h3>
      </div>
      <SubscriptionBadge :status="item.status" />
    </header>
    <p class="hub-card__desc">{{ item.description }}</p>
    <p v-if="item.pack_key" class="hub-card__pack">功能包：{{ item.pack_display_name || item.pack_key }}</p>

    <div v-if="hasCli" class="hub-card__cli-toggle">
      <el-button size="small" link @click="cliExpanded = !cliExpanded">
        {{ cliExpanded ? '收起 CLI' : '查看 CLI' }}
      </el-button>
    </div>
    <el-collapse-transition>
      <div v-show="cliExpanded && hasCli" class="hub-card__cli">
        <CliCommandBlock
          v-for="cmd in item.commands || []"
          :key="cmd.template"
          :label="cmd.label"
          :command="cmd.template"
        />
        <CliCommandBlock v-if="item.cli_topic && !item.commands?.length" :command="`ai-sre check ${item.cli_topic}`" />
      </div>
    </el-collapse-transition>

    <footer class="hub-card__actions">
      <el-tooltip v-if="!item.can_open" content="请先订阅或联系管理员开通" placement="top">
        <span>
          <el-button type="primary" size="small" disabled>打开控制台</el-button>
        </span>
      </el-tooltip>
      <el-button v-else type="primary" size="small" @click="$emit('open', item)">打开控制台</el-button>

      <el-button
        v-if="item.can_subscribe"
        type="warning"
        size="small"
        @click="$emit('subscribe', item)"
      >
        订阅
      </el-button>
      <el-button
        v-else-if="needsAdmin"
        type="info"
        size="small"
        @click="$emit('contact-admin', item)"
      >
        联系管理员
      </el-button>
      <span v-else-if="isEntitled" class="hub-card__entitled-hint">已开通</span>
    </footer>

    <div v-if="showPackSubscribe" class="hub-card__pack-action">
      <span class="hub-card__pack-label">订阅整包：{{ packInfo?.display_name }}</span>
      <el-button type="warning" size="small" plain @click="$emit('subscribe-pack', item.pack_key!)">订阅整包</el-button>
    </div>

    <el-link
      v-if="item.execution_source && item.category === 'delivery'"
      type="primary"
      :underline="false"
      class="hub-card__exec-link"
      @click.prevent="$emit('executions', item)"
    >
      查看相关执行记录
    </el-link>
  </article>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import CliCommandBlock from './CliCommandBlock.vue'
import SubscriptionBadge from './SubscriptionBadge.vue'
import type { ResolvedCapability } from '../../composables/useCapabilityCatalog'
import { resolveCatalogIcon } from '../../utils/catalogIcons'

const ENTITLED = new Set(['已订阅', '免费可用', '管理员已开通'])

const props = defineProps<{
  item: ResolvedCapability
  highlighted?: boolean
  packInfo?: { entitled: boolean; can_subscribe: boolean; display_name: string }
}>()

defineEmits<{
  open: [ResolvedCapability]
  subscribe: [ResolvedCapability]
  'subscribe-pack': [string]
  'contact-admin': [ResolvedCapability]
  executions: [ResolvedCapability]
}>()

const cliExpanded = ref(false)
const iconComp = computed(() => resolveCatalogIcon(props.item.icon))

const hasCli = computed(() => Boolean(props.item.commands?.length || props.item.cli_topic))
const isEntitled = computed(() => ENTITLED.has(props.item.status))
const needsAdmin = computed(
  () => !isEntitled.value && !props.item.can_subscribe && props.item.status !== '暂不可用'
)

const showPackSubscribe = computed(
  () =>
    Boolean(props.item.pack_key) &&
    props.packInfo &&
    !props.packInfo.entitled &&
    props.packInfo.can_subscribe &&
    !props.item.can_subscribe
)

watch(
  () => props.highlighted,
  (v) => {
    if (v && hasCli.value) cliExpanded.value = true
  },
  { immediate: true }
)
</script>

<style scoped>
.hub-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  padding: 16px;
  background: var(--layout-content-surface);
  display: flex;
  flex-direction: column;
  gap: 8px;
  transition: border-color 0.2s, box-shadow 0.2s;
}
.hub-card--highlight {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.15);
}
.hub-card__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}
.hub-card__title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.hub-card__title-row h3 {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
}
.hub-card__icon {
  font-size: 20px;
  color: var(--el-color-primary);
}
.hub-card__desc {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
.hub-card__pack {
  margin: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.hub-card__actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
}
.hub-card__entitled-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.hub-card__pack-action {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
  font-size: 12px;
}
.hub-card__pack-label {
  color: var(--el-text-color-regular);
}
.hub-card__exec-link {
  font-size: 13px;
  margin-top: 2px;
}
</style>
