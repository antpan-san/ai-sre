<template>
  <div class="cli-block">
    <span v-if="label" class="cli-block__label">{{ label }}</span>
    <div class="cli-block__row">
      <code>{{ command }}</code>
      <el-button size="small" link type="primary" @click="copy">复制</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { copyTextToClipboard } from '../../utils/clipboard'

const props = defineProps<{
  command: string
  label?: string
}>()

const copy = async () => {
  await copyTextToClipboard(props.command)
  ElMessage.success('已复制命令')
}
</script>

<style scoped>
.cli-block {
  margin-bottom: 8px;
}
.cli-block__label {
  display: block;
  font-size: 11px;
  color: var(--el-text-color-secondary);
  margin-bottom: 4px;
}
.cli-block__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
}
.cli-block__row code {
  font-size: 12px;
  word-break: break-all;
}
</style>
