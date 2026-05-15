<template>
  <div class="safe-md" :style="boxStyle" v-html="html" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { markdownToSafeHtml } from '../../utils/safeMarkdown'

const props = withDefaults(
  defineProps<{
    content?: string | null
    clampLines?: number
  }>(),
  { clampLines: 0 }
)

const html = computed(() => markdownToSafeHtml(props.content))

const boxStyle = computed(() => {
  if (!props.clampLines || props.clampLines <= 0) return {}
  return {
    display: '-webkit-box',
    WebkitBoxOrient: 'vertical',
    WebkitLineClamp: Number(props.clampLines),
    overflow: 'hidden',
    wordBreak: 'break-word'
  } as Record<string, string | number>
})
</script>

<style scoped>
.safe-md :deep(h1),
.safe-md :deep(h2),
.safe-md :deep(h3),
.safe-md :deep(h4) {
  margin: 0.5em 0 0.25em;
  font-size: 1em;
  font-weight: 600;
}
.safe-md :deep(h1:first-child),
.safe-md :deep(h2:first-child),
.safe-md :deep(h3:first-child) {
  margin-top: 0;
}
.safe-md :deep(p) {
  margin: 0.35em 0;
  line-height: 1.5;
}
.safe-md :deep(p:first-child) {
  margin-top: 0;
}
.safe-md :deep(p:last-child) {
  margin-bottom: 0;
}
.safe-md :deep(ul),
.safe-md :deep(ol) {
  margin: 0.35em 0;
  padding-left: 1.25em;
}
.safe-md :deep(pre) {
  margin: 0.5em 0;
  padding: 10px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  overflow: auto;
  font-size: 12px;
  line-height: 1.45;
}
.safe-md :deep(code) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, monospace;
  font-size: 0.9em;
}
.safe-md :deep(p code),
.safe-md :deep(li code) {
  padding: 1px 5px;
  border-radius: 4px;
  background: var(--el-fill-color-light);
}
.safe-md :deep(blockquote) {
  margin: 0.35em 0;
  padding-left: 10px;
  border-left: 3px solid var(--el-border-color);
  color: var(--el-text-color-secondary);
}
.safe-md :deep(table) {
  border-collapse: collapse;
  font-size: 13px;
  margin: 0.5em 0;
}
.safe-md :deep(th),
.safe-md :deep(td) {
  border: 1px solid var(--el-border-color-lighter);
  padding: 4px 8px;
}
.safe-md :deep(a) {
  color: var(--el-color-primary);
}
</style>
