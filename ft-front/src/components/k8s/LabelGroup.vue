<template>
  <div class="label-group">
    <!-- 已添加标签 -->
    <div class="label-list" v-if="Object.keys(labels).length">
      <div
        v-for="(value, key) in labels"
        :key="key"
        class="label-item"
      >
        <span class="label-key">{{ key }}</span>
        <span class="label-separator">:</span>
        <span class="label-value">{{ value }}</span>
        <el-button
          size="small"
          type="danger"
          link
          class="label-del-btn"
          @click="removeLabel(key)"
        >
          <el-icon><Close /></el-icon>
        </el-button>
      </div>
    </div>
    <p v-else class="label-empty">暂无标签</p>

    <!-- 紧凑单行添加 -->
    <div class="label-add-row">
      <el-input
        v-model="newLabelKey"
        placeholder="键"
        size="small"
        clearable
        @keyup.enter="addLabel"
      />
      <el-input
        v-model="newLabelValue"
        placeholder="值"
        size="small"
        clearable
        @keyup.enter="addLabel"
      />
      <el-button
        size="small"
        type="primary"
        :disabled="!newLabelKey.trim() || !newLabelValue.trim()"
        @click="addLabel"
      >
        <el-icon><Plus /></el-icon>
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { Close, Plus } from '@element-plus/icons-vue'

interface Props {
  modelValue: Record<string, string>
  disabled?: boolean
  maxLabels?: number
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: () => ({}),
  disabled: false,
  maxLabels: 20
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: Record<string, string>): void
}>()

const labels = ref<Record<string, string>>({ ...props.modelValue })
const newLabelKey = ref('')
const newLabelValue = ref('')

watch(() => props.modelValue, (newValue) => {
  labels.value = { ...newValue }
}, { deep: true })

const isFull = computed(() => Object.keys(labels.value).length >= props.maxLabels)

const addLabel = () => {
  if (!newLabelKey.value.trim() || !newLabelValue.value.trim()) return
  if (isFull.value) return
  const key = newLabelKey.value.trim()
  const value = newLabelValue.value.trim()
  labels.value[key] = value
  emitUpdate()
  newLabelKey.value = ''
  newLabelValue.value = ''
}

const removeLabel = (key: string) => {
  if (props.disabled) return
  delete labels.value[key]
  emitUpdate()
}

const emitUpdate = () => {
  emit('update:modelValue', { ...labels.value })
}
</script>

<style scoped>
.label-group {
  padding: 0;
}

.label-empty {
  margin: 0 0 6px;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

.label-list {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  max-height: 72px;
  overflow-y: auto;
  margin-bottom: 6px;
}

.label-item {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 2px 6px;
  background: #e8f4ff;
  border: 1px solid #91caff;
  border-radius: 10px;
  font-size: 11px;
  line-height: 1.4;
}

.label-key {
  font-weight: 600;
  color: #1677ff;
}

.label-separator {
  color: #9ca3af;
  margin: 0 2px;
}

.label-value {
  color: #374151;
}

.label-del-btn {
  padding: 0 0 0 2px !important;
  height: auto !important;
  font-size: 10px;
  color: #9ca3af !important;
}

.label-del-btn:hover {
  color: #f5222d !important;
}

/* 单行添加区 */
.label-add-row {
  display: flex;
  gap: 4px;
  align-items: center;
}

.label-add-row :deep(.el-input) {
  flex: 1;
  min-width: 0;
}
</style>
