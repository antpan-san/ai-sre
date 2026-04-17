<template>
  <div class="label-group">
    <div class="label-list">
      <div
        v-for="(value, key) in labels"
        :key="key"
        class="label-item"
      >
        <div class="label-text">
          <span class="label-key">{{ key }}</span>
          <span class="label-separator">:</span>
          <span class="label-value">{{ value }}</span>
        </div>
        <el-button
          size="small"
          type="danger"
          circle
          @click="removeLabel(key)"
        >
          <el-icon><Delete /></el-icon>
        </el-button>
      </div>
    </div>

    <div class="label-input-group">
      <div class="label-input-wrapper">
        <div class="label-input-row">
          <div class="label-input-label">键</div>
          <el-input
            v-model="newLabelKey"
            placeholder="输入键"
            size="small"
            class="label-input"
            @keyup.enter="addLabel"
          />
        </div>
        <div class="label-input-row">
          <div class="label-input-label">值</div>
          <el-input
            v-model="newLabelValue"
            placeholder="输入值"
            size="small"
            class="label-input"
            @keyup.enter="addLabel"
          />
        </div>
      </div>
      <el-button
        size="small"
        type="primary"
        @click="addLabel"
        :disabled="!newLabelKey.trim() || !newLabelValue.trim()"
      >
        <el-icon><Plus /></el-icon> 添加
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { Delete, Plus } from '@element-plus/icons-vue'

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
  padding: 8px;
  background-color: transparent;
  border: none;
  border-radius: 0;
}

.label-list {
  margin-bottom: 12px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  max-height: 150px;
  overflow-y: auto;
}

.label-item {
  display: flex;
  align-items: center;
  padding: 6px 12px;
  background-color: #e5f3ff;
  border: 1px solid #91caff;
  border-radius: 16px;
  font-size: 12px;
  transition: all 0.2s ease;
}

.label-item:hover {
  background-color: #d4ebff;
  border-color: #73b3ff;
}

.label-text {
  display: flex;
  align-items: center;
  margin-right: 8px;
}

.label-key {
  font-weight: bold;
  color: #1890ff;
}

.label-separator {
  margin: 0 4px;
  color: #6b7280;
}

.label-value {
  color: #374151;
}

.label-input-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
  align-items: stretch;
  margin-top: 8px;
}

.label-input {
  width: 100%;
  border-radius: 4px;
}

.label-input-wrapper {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}

.label-input-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.label-input-label {
  font-weight: normal;
  color: #6b7280;
  min-width: 60px;
  font-size: 14px;
}
</style>