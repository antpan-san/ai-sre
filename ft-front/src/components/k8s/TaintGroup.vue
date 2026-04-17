<template>
  <div class="taint-group">
    <div class="taint-list">
      <div
        v-for="(taint, index) in taints"
        :key="index"
        class="taint-item"
      >
        <div class="taint-text">
          <span class="taint-key">{{ taint.key }}</span>
          <span class="taint-separator">:</span>
          <span class="taint-value">{{ taint.value }}</span>
          <span class="taint-effect">({{ taint.effect }})</span>
        </div>
        <el-button
          size="small"
          type="danger"
          circle
          @click="removeTaint(index)"
        >
          <el-icon><Delete /></el-icon>
        </el-button>
      </div>
    </div>

    <div class="taint-form">
      <el-form :model="newTaint" label-position="left" label-width="60px" size="small">
        <el-form-item label="键">
          <el-input
            v-model="newTaint.key"
            placeholder="输入键"
            class="taint-input"
          />
        </el-form-item>
        <el-form-item label="值">
          <el-input
            v-model="newTaint.value"
            placeholder="输入值"
            class="taint-input"
          />
        </el-form-item>
        <el-form-item label="效果">
          <el-select
            v-model="newTaint.effect"
            placeholder="选择效果"
            class="taint-input"
          >
            <el-option label="NoSchedule" value="NoSchedule" />
            <el-option label="PreferNoSchedule" value="PreferNoSchedule" />
            <el-option label="NoExecute" value="NoExecute" />
          </el-select>
        </el-form-item>
        <el-form-item label="">
          <el-button
            type="primary"
            @click="addTaint"
            :disabled="!isTaintValid"
          >
            <el-icon><Plus /></el-icon> 添加
          </el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { Delete, Plus } from '@element-plus/icons-vue'
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

  newTaint.value = {
    key: '',
    value: '',
    effect: 'NoSchedule'
  }
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
  padding: 8px;
  background-color: transparent;
  border: none;
  border-radius: 0;
}

.taint-list {
  margin-bottom: 12px;
  max-height: 150px;
  overflow-y: auto;
}

.taint-item {
  display: flex;
  align-items: center;
  padding: 6px 12px;
  background-color: #fff3f0;
  border: 1px solid #ffccc7;
  border-radius: 16px;
  font-size: 12px;
  margin-bottom: 8px;
  transition: all 0.2s ease;
}

.taint-item:hover {
  background-color: #ffe7e3;
  border-color: #ffac9d;
}

.taint-text {
  display: flex;
  align-items: center;
  margin-right: 8px;
}

.taint-key {
  font-weight: bold;
  color: #cf1322;
}

.taint-separator {
  margin: 0 4px;
  color: #6b7280;
}

.taint-value {
  color: #374151;
}

.taint-effect {
  color: #722ed1;
  margin-left: 4px;
}

.taint-form {
  border-top: 1px solid #e5e7eb;
  padding-top: 12px;
}

.taint-input {
  width: 100%;
  border-radius: 4px;
}

.taint-form .el-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.taint-form .el-form-item {
  margin-right: 0;
  margin-bottom: 0;
}

.taint-form .el-form-item__label {
  font-weight: normal;
  color: #6b7280;
  min-width: 60px;
}

.taint-form .el-form-item__content {
  display: flex;
  align-items: center;
}

.taint-form .el-button {
  margin-top: 4px;
  align-self: flex-start;
}
</style>