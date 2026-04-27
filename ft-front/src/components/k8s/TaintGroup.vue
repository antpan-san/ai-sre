<template>
  <div class="taint-group">
    <!-- 已添加污点 -->
    <div class="taint-list" v-if="taints.length">
      <div
        v-for="(taint, index) in taints"
        :key="index"
        class="taint-item"
      >
        <span class="taint-key">{{ taint.key }}</span>
        <span class="taint-separator">:</span>
        <span class="taint-value">{{ taint.value }}</span>
        <span class="taint-effect">{{ taint.effect }}</span>
        <el-button
          size="small"
          type="danger"
          link
          class="taint-del-btn"
          @click="removeTaint(index)"
        >
          <el-icon><Close /></el-icon>
        </el-button>
      </div>
    </div>
    <p v-else class="taint-empty">暂无污点</p>

    <!-- 紧凑两行添加 -->
    <div class="taint-add-area">
      <div class="taint-add-row">
        <el-input
          v-model="newTaint.key"
          placeholder="键"
          size="small"
          clearable
        />
        <el-input
          v-model="newTaint.value"
          placeholder="值"
          size="small"
          clearable
        />
      </div>
      <div class="taint-add-row">
        <el-select
          v-model="newTaint.effect"
          placeholder="效果"
          size="small"
        >
          <el-option label="NoSchedule" value="NoSchedule" />
          <el-option label="PreferNoSchedule" value="PreferNoSchedule" />
          <el-option label="NoExecute" value="NoExecute" />
        </el-select>
        <el-button
          size="small"
          type="primary"
          :disabled="!isTaintValid"
          @click="addTaint"
        >
          <el-icon><Plus /></el-icon> 添加
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { Close, Plus } from '@element-plus/icons-vue'
import type { Taint } from '../../types/k8s-deploy'

interface Props {
  modelValue: Taint[]
  disabled?: boolean
  maxTaints?: number
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: () => ([]),
  disabled: false,
  maxTaints: 10
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: Taint[]): void
}>()

const taints = ref<Taint[]>([...props.modelValue])
const newTaint = ref<Taint>({
  key: '',
  value: '',
  effect: 'NoSchedule'
})

watch(() => props.modelValue, (newValue) => {
  taints.value = [...newValue]
}, { deep: true })

const isFull = computed(() => taints.value.length >= props.maxTaints)

const isTaintValid = computed(() => {
  return (
    newTaint.value.key.trim() &&
    newTaint.value.value.trim() &&
    newTaint.value.effect &&
    !isFull.value
  )
})

const addTaint = () => {
  if (!isTaintValid.value) return
  taints.value.push({
    key: newTaint.value.key.trim(),
    value: newTaint.value.value.trim(),
    effect: newTaint.value.effect
  })
  emitUpdate()
  newTaint.value = { key: '', value: '', effect: 'NoSchedule' }
}

const removeTaint = (index: number) => {
  if (props.disabled) return
  taints.value.splice(index, 1)
  emitUpdate()
}

const emitUpdate = () => {
  emit('update:modelValue', [...taints.value])
}
</script>

<style scoped>
.taint-group {
  padding: 0;
}

.taint-empty {
  margin: 0 0 6px;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

.taint-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 72px;
  overflow-y: auto;
  margin-bottom: 6px;
}

.taint-item {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 2px 6px;
  background: #fff2f0;
  border: 1px solid #ffccc7;
  border-radius: 10px;
  font-size: 11px;
  line-height: 1.4;
}

.taint-key {
  font-weight: 600;
  color: #cf1322;
}

.taint-separator {
  color: #9ca3af;
  margin: 0 2px;
}

.taint-value {
  color: #374151;
}

.taint-effect {
  color: #722ed1;
  margin-left: 4px;
  font-size: 10px;
  background: #f9f0ff;
  border: 1px solid #d3adf7;
  border-radius: 6px;
  padding: 0 4px;
}

.taint-del-btn {
  margin-left: auto;
  padding: 0 !important;
  height: auto !important;
  font-size: 10px;
  color: #9ca3af !important;
}

.taint-del-btn:hover {
  color: #f5222d !important;
}

/* 两行紧凑添加区 */
.taint-add-area {
  display: flex;
  flex-direction: column;
  gap: 4px;
  border-top: 1px solid var(--el-border-color-lighter);
  padding-top: 6px;
}

.taint-add-row {
  display: flex;
  gap: 4px;
  align-items: center;
}

.taint-add-row :deep(.el-input),
.taint-add-row :deep(.el-select) {
  flex: 1;
  min-width: 0;
}
</style>
