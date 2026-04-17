<template>
  <el-dialog
    v-model="dialogVisible"
    :title="props.title"
    :width="props.width"
    :before-close="handleClose"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
  >
    <div class="confirm-content">
      <el-icon v-if="props.type === 'warning'" class="icon-warning"><WarningFilled /></el-icon>
      <el-icon v-if="props.type === 'info'" class="icon-info"><InfoFilled /></el-icon>
      <el-icon v-if="props.type === 'error'" class="icon-error"><ErrorFilled /></el-icon>
      <el-icon v-if="props.type === 'success'" class="icon-success"><SuccessFilled /></el-icon>
      <span>{{ props.content }}</span>
    </div>
    
    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleCancel">取消</el-button>
        <el-button
          :type="props.confirmType"
          :loading="props.loading"
          @click="handleConfirm"
        >
          {{ props.confirmText }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { WarningFilled, InfoFilled, CircleCloseFilled as ErrorFilled, SuccessFilled } from '@element-plus/icons-vue'

interface Props {
  visible: boolean
  title?: string
  content: string
  type?: 'warning' | 'info' | 'error' | 'success'
  confirmText?: string
  confirmType?: 'primary' | 'success' | 'warning' | 'danger'
  width?: string
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  visible: false,
  title: '确认操作',
  type: 'warning',
  confirmText: '确定',
  confirmType: 'primary',
  width: '30%',
  loading: false
})

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'confirm'): void
  (e: 'cancel'): void
  (e: 'close'): void
}>()

const dialogVisible = ref(props.visible)

watch(() => props.visible, (newVal) => {
  dialogVisible.value = newVal
})

watch(dialogVisible, (newVal) => {
  emit('update:visible', newVal)
})

const handleConfirm = () => {
  emit('confirm')
  dialogVisible.value = false
}

const handleCancel = () => {
  emit('cancel')
  dialogVisible.value = false
}

const handleClose = () => {
  emit('close')
  dialogVisible.value = false
}
</script>

<style scoped>
.confirm-content {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 20px;
  line-height: 1.6;
}

.icon-warning {
  color: #e6a23c;
  font-size: 24px;
}

.icon-info {
  color: #909399;
  font-size: 24px;
}

.icon-error {
  color: #f56c6c;
  font-size: 24px;
}

.icon-success {
  color: #67c23a;
  font-size: 24px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>