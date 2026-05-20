<template>
  <article class="cap-card">
    <header class="cap-card__head">
      <div class="cap-card__title-row">
        <el-icon v-if="iconComp" class="cap-card__icon"><component :is="iconComp" /></el-icon>
        <div>
          <h3 class="cap-card__title">{{ item.name }}</h3>
          <el-tag v-if="categoryShort" size="small" type="info" effect="plain" class="cap-card__cat">{{ categoryShort }}</el-tag>
        </div>
      </div>
      <SubscriptionBadge :status="item.status" />
    </header>
    <p class="cap-card__desc">{{ item.description }}</p>
    <p v-if="item.pack_key" class="cap-card__pack">功能包：{{ item.pack_display_name || item.pack_key }}</p>
    <CliCommandBlock v-if="item.cli_topic" :command="`ai-sre check ${item.cli_topic}`" />
    <footer class="cap-card__actions">
      <el-button v-if="item.can_open" type="primary" size="small" @click="$emit('open', item)">打开</el-button>
      <el-button v-else-if="item.can_subscribe" type="warning" size="small" @click="$emit('subscribe', item)">订阅</el-button>
      <el-popover v-else placement="top" :width="280" trigger="click">
        <template #reference>
          <el-button size="small">了解详情</el-button>
        </template>
        <p class="cap-popover__desc">{{ item.description }}</p>
        <p v-if="item.pack_key" class="cap-popover__pack">功能包：{{ item.pack_display_name || item.pack_key }}</p>
        <p class="cap-popover__hint">请联系管理员开通此能力。</p>
      </el-popover>
    </footer>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import CliCommandBlock from './CliCommandBlock.vue'
import SubscriptionBadge from './SubscriptionBadge.vue'
import { CAPABILITY_CATEGORY_SHORT } from '../../config/capabilityCatalog'
import type { ResolvedCapability } from '../../composables/useCapabilityCatalog'
import { resolveCatalogIcon } from '../../utils/catalogIcons'

const props = defineProps<{ item: ResolvedCapability }>()
defineEmits<{ open: [ResolvedCapability]; subscribe: [ResolvedCapability] }>()

const iconComp = computed(() => resolveCatalogIcon(props.item.icon))
const categoryShort = computed(() => CAPABILITY_CATEGORY_SHORT[props.item.category])
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
  min-height: 180px;
}
.cap-card__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}
.cap-card__title-row {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}
.cap-card__icon {
  font-size: 22px;
  color: var(--el-color-primary);
  margin-top: 2px;
}
.cap-card__title {
  margin: 0 0 4px;
  font-size: 15px;
  font-weight: 600;
}
.cap-card__cat {
  vertical-align: middle;
}
.cap-card__desc {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
  flex: 1;
}
.cap-card__pack {
  margin: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.cap-card__actions {
  margin-top: auto;
  padding-top: 4px;
}
.cap-popover__desc {
  margin: 0 0 8px;
  font-size: 13px;
  line-height: 1.5;
}
.cap-popover__pack {
  margin: 0 0 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.cap-popover__hint {
  margin: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
