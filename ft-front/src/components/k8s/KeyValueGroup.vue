<template>
  <div class="key-value-group">
    <div class="key-value-list">
      <div
        v-for="(item, index) in keyValues"
        :key="index"
        class="key-value-item"
      >
        <div class="key-value-text">
          <span class="key-value-key">{{ item.key }}</span>
          <span class="key-value-separator">=</span>
          <span class="key-value-value">{{ item.value }}</span>
        </div>
        <el-button
          size="small"
          type="danger"
          circle
          @click="removeKeyValue(index)"
        >
          <el-icon><Delete /></el-icon>
        </el-button>
      </div>
    </div>

    <div class="key-value-input-group">
      <el-input
        v-model="newKeyValue.key"
        placeholder="键"
        size="small"
        class="key-value-input"
        @keyup.enter="addKeyValue"
      />
      <el-input
        v-model="newKeyValue.value"
        placeholder="值"
        size="small"
        class="key-value-input"
        @keyup.enter="addKeyValue"
      />
      <el-button
        size="small"
        type="primary"
        @click="addKeyValue"
        :disabled="!isKeyValueValid"
      >
        <el-icon><Plus /></el-icon> 添加
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { Delete, Plus } from '@element-plus/icons-vue'
import type { KeyValuePair } from '../../types/k8s-deploy'

interface Props {
  modelValue: KeyValuePair[]
  disabled?: boolean
  maxItems?: number
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: () => ([]),
  disabled: false,
  maxItems: 15
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: KeyValuePair[]): void
}>()

const keyValues = ref<KeyValuePair[]>([...props.modelValue])
const newKeyValue = ref<KeyValuePair>({
  key: '',
  value: ''
})

watch(() => props.modelValue, (newValue) => {
  keyValues.value = [...newValue]
}, { deep: true })

const isFull = computed(() => keyValues.value.length >= props.maxItems)

const isKeyValueValid = computed(() => {
  return (
    newKeyValue.value.key.trim() &&
    !isFull.value
  )
})

const addKeyValue = () => {
  if (!isKeyValueValid.value) return

  keyValues.value.push({
    key: newKeyValue.value.key.trim(),
    value: newKeyValue.value.value.trim()
  })

  emitUpdate()

  newKeyValue.value = {
    key: '',
    value: ''
  }
}

const removeKeyValue = (index: number) => {
  if (props.disabled) return
  keyValues.value.splice(index, 1)
  emitUpdate()
}

const emitUpdate = () => {
  emit('update:modelValue', [...keyValues.value])
}
</script>

<style scoped>
.key-value-group {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 16px;
  background-color: #f9fafb;
}

.key-value-list {
  margin-bottom: 12px;
  max-height: 200px;
  overflow-y: auto;
}

.key-value-item {
  display: flex;
  align-items: center;
  padding: 6px 12px;
  background-color: #f0f9eb;
  border: 1px solid #b7eb8f;
  border-radius: 16px;
  font-size: 12px;
  margin-bottom: 8px;
}

.key-value-text {
  display: flex;
  align-items: center;
  margin-right: 8px;
}

.key-value-key {
  font-weight: bold;
  color: #52c41a;
}

.key-value-separator {
  margin: 0 4px;
  color: #6b7280;
}

.key-value-value {
  color: #374151;
}

.key-value-input-group {
  display: flex;
  gap: 8px;
  align-items: stretch;
}

.key-value-input {
  flex: 1;
  min-width: 100px;
}
</style>