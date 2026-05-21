<template>
  <el-card
    class="workload-tile"
    shadow="hover"
    :style="tileStyle"
    :body-style="bodyStyle"
    @click="go"
  >
    <div class="workload-tile__top">
      <div class="workload-tile__title-copy">
        <h4 class="workload-tile__title">{{ title }}</h4>
        <p class="workload-tile__desc">{{ description }}</p>
      </div>
      <el-tag size="small" :type="statusType">{{ status }}</el-tag>
    </div>

    <div v-if="packLabel" class="workload-tile__meta">
      <span class="workload-tile__pack">{{ packLabel }}</span>
    </div>

    <div v-if="visibleTags.length" class="workload-tile__tags">
      <el-tag v-for="tag in visibleTags" :key="tag" size="small" effect="plain" type="info">{{ tag }}</el-tag>
      <el-tag v-if="overflowTag" size="small" effect="plain" type="info">{{ overflowTag }}</el-tag>
    </div>

    <div class="workload-tile__footer">
      <el-button type="primary" @click.stop="go">进入</el-button>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'

const props = withDefaults(
  defineProps<{
    title: string
    description: string
    tags?: string[]
    status: string
    statusType?: 'success' | 'warning' | 'info'
    packLabel?: string
    accent?: string
    detailPath: string
  }>(),
  {
    tags: () => [],
    statusType: 'info',
    packLabel: ''
  }
)

const router = useRouter()

const tileStyle = computed(() => ({
  '--workload-accent': props.accent || 'var(--el-color-primary)'
}))

const bodyStyle = {
  display: 'flex',
  flexDirection: 'column',
  height: '100%'
}

const visibleTags = computed(() => props.tags.slice(0, 3))
const overflowTag = computed(() => {
  const rest = props.tags.length - visibleTags.value.length
  return rest > 0 ? `+${rest}` : ''
})

const go = () => {
  void router.push(props.detailPath)
}
</script>

<style scoped>
.workload-tile {
  height: 100%;
  min-height: 176px;
  cursor: pointer;
  border-radius: 18px;
  border: 1px solid var(--el-border-color-lighter);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.96), rgba(248, 250, 252, 0.98)),
    radial-gradient(circle at top right, color-mix(in srgb, var(--workload-accent) 16%, transparent), transparent 36%);
  position: relative;
  overflow: hidden;
}
.workload-tile::before {
  content: '';
  position: absolute;
  inset: 0 auto auto 0;
  width: 100%;
  height: 4px;
  background: var(--workload-accent);
}
.workload-tile__top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}
.workload-tile__title-copy {
  min-width: 0;
}
.workload-tile__title {
  margin: 0;
  font-size: 14px;
  font-weight: 650;
}
.workload-tile__desc {
  margin: 4px 0 0;
  font-size: 12px;
  line-height: 1.45;
  color: var(--el-text-color-secondary);
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 1;
  overflow: hidden;
}
.workload-tile__meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
  color: var(--el-text-color-secondary);
  font-size: 11px;
}
.workload-tile__pack {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.workload-tile__tags {
  display: flex;
  flex-wrap: nowrap;
  gap: 6px;
  margin-bottom: 10px;
  overflow: hidden;
}
.workload-tile__tags :deep(.el-tag) {
  flex-shrink: 0;
  max-width: 100%;
}
.workload-tile__footer {
  margin-top: auto;
  display: flex;
  justify-content: flex-end;
}
.workload-tile__footer :deep(.el-button) {
  min-width: 72px;
}
</style>
