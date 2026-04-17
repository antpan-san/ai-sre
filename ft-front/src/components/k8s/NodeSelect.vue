<template>
  <div class="node-select-container">
    <el-card class="node-card">
      <template #header>
        <div class="card-header">
          <span>{{ masterTitle }}</span>
          <el-button
            type="primary"
            size="small"
            @click="selectAllMasters"
            :disabled="selectableMasters.length === 0 || masterNodes.length >= selectableMasters.length"
          >
            全选
          </el-button>
          <el-button
            size="small"
            @click="clearAllMasters"
            :disabled="masterNodes.length === 0"
          >
            清空
          </el-button>
        </div>
      </template>
      <el-checkbox-group v-model="masterNodes" @change="handleMasterChange">
        <div v-for="machine in filteredMasters" :key="machine.id" class="machine-checkbox-item">
          <el-checkbox
            :label="machine.id"
            class="machine-checkbox"
            :disabled="!isMachineOnline(machine) || isCollecting(machine)"
          >
            <div class="machine-item machine-grid" :class="{ 'machine-offline': !isMachineOnline(machine), 'machine-collecting': isCollecting(machine) }">
              <span class="col-name">{{ machine.name || '未命名机器' }}</span>
              <span class="col-ip">{{ machine.ip || '未配置IP' }}</span>
              <span class="col-num">{{ displayCpu(machine) }} 核</span>
              <span class="col-num">{{ displayMemory(machine) }} GB</span>
              <span class="col-num">{{ displayDisk(machine) }} GB</span>
              <div class="col-status">
                <el-icon v-if="isCollecting(machine)" class="is-loading status-loading">
                  <Loading />
                </el-icon>
                <el-tag v-else :type="getStatusTagType(machine.status)" size="small">
                  {{ getStatusText(machine.status) }}
                </el-tag>
                <span v-if="isCollecting(machine)" class="status-collecting-text">维护中</span>
              </div>
            </div>
          </el-checkbox>
        </div>
      </el-checkbox-group>
      <div v-if="filteredMasters.length === 0" class="empty-text">
        暂无可用机器
      </div>
    </el-card>

    <el-card class="node-card">
      <template #header>
        <div class="card-header">
          <span>{{ workerTitle }}</span>
          <el-button
            type="primary"
            size="small"
            @click="selectAllWorkers"
            :disabled="selectableWorkers.length === 0 || workerNodes.length >= selectableWorkers.length"
          >
            全选
          </el-button>
          <el-button
            size="small"
            @click="clearAllWorkers"
            :disabled="workerNodes.length === 0"
          >
            清空
          </el-button>
        </div>
      </template>
      <el-checkbox-group v-model="workerNodes" @change="handleWorkerChange">
        <div v-for="machine in filteredWorkers" :key="machine.id" class="machine-checkbox-item">
          <el-checkbox
            :label="machine.id"
            class="machine-checkbox"
            :disabled="!isMachineOnline(machine) || isCollecting(machine)"
          >
            <div class="machine-item machine-grid" :class="{ 'machine-offline': !isMachineOnline(machine), 'machine-collecting': isCollecting(machine) }">
              <span class="col-name">{{ machine.name || '未命名机器' }}</span>
              <span class="col-ip">{{ machine.ip || '未配置IP' }}</span>
              <span class="col-num">{{ displayCpu(machine) }} 核</span>
              <span class="col-num">{{ displayMemory(machine) }} GB</span>
              <span class="col-num">{{ displayDisk(machine) }} GB</span>
              <div class="col-status">
                <el-icon v-if="isCollecting(machine)" class="is-loading status-loading">
                  <Loading />
                </el-icon>
                <el-tag v-else :type="getStatusTagType(machine.status)" size="small">
                  {{ getStatusText(machine.status) }}
                </el-tag>
                <span v-if="isCollecting(machine)" class="status-collecting-text">维护中</span>
              </div>
            </div>
          </el-checkbox>
        </div>
      </el-checkbox-group>
      <div v-if="filteredWorkers.length === 0" class="empty-text">
        暂无可用机器
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Loading } from '@element-plus/icons-vue'
import type { K8sMachineInfo } from '../../types/k8s-deploy'

/** 机器展示用（与机器管理一致：支持 cpu_cores / memory_total / disk_total 等） */
type MachineDisplay = K8sMachineInfo & {
  cpu_cores?: number
  memory_total?: number
  disk_total?: number
}

interface Props {
  machines: (K8sMachineInfo | MachineDisplay)[]
  modelValue: { masterNodes: string[]; workerNodes: string[] }
  masterTitle?: string
  workerTitle?: string
  disabled?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  masterTitle: '主节点',
  workerTitle: '工作节点',
  disabled: false
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: { masterNodes: string[]; workerNodes: string[] }): void
}>()

const masterNodes = ref<string[]>(props.modelValue.masterNodes)
const workerNodes = ref<string[]>(props.modelValue.workerNodes)

// 监听父组件modelValue变化，更新本地状态
watch(() => props.modelValue, (newValue) => {
  masterNodes.value = [...newValue.masterNodes]
  workerNodes.value = [...newValue.workerNodes]
}, { deep: true })

// 主节点与工作节点均展示所有受控机器，同一机器可同时作为主节点和工作节点（非互斥）
const filteredMasters = computed(() => props.machines)
const filteredWorkers = computed(() => props.machines)

// 机器是否在线（在线才可选）
const isMachineOnline = (machine: { status: string }) => machine.status === 'online'

// 是否处于采集中：cpu/内存/磁盘全为 0，表示尚未采集到真实数据（初始进入页面时可能如此）
const isCollecting = (m: MachineDisplay): boolean => {
  const cpu = m.cpu_cores ?? m.cpu ?? 0
  const mem = m.memory ?? 0
  const disk = m.disk ?? 0
  return cpu === 0 && mem === 0 && disk === 0
}

// 主节点中可选的机器：在线即可（K8s 控制平面节点，与 Agent 所在节点解耦）
const selectableMasters = computed(() =>
  filteredMasters.value.filter(m => isMachineOnline(m))
)

// 工作节点中可选的机器（在线）
const selectableWorkers = computed(() =>
  filteredWorkers.value.filter(m => isMachineOnline(m))
)

// 与机器管理列表展示一致：CPU 优先 cpu_cores，其次 cpu；内存/磁盘使用配置值 memory/disk（GB）
// 采集中时显示 0
function displayCpu(m: MachineDisplay): number {
  return isCollecting(m) ? 0 : (m.cpu_cores ?? m.cpu ?? 0)
}
function displayMemory(m: MachineDisplay): number {
  return isCollecting(m) ? 0 : (m.memory ?? 0)
}
function displayDisk(m: MachineDisplay): number {
  return isCollecting(m) ? 0 : (m.disk ?? 0)
}

const getStatusTagType = (status: string): 'success' | 'danger' | 'warning' => {
  switch (status) {
    case 'online':
      return 'success'
    case 'offline':
      return 'danger'
    case 'maintenance':
      return 'warning'
    default:
      return 'danger'
  }
}

const getStatusText = (status: string): string => {
  switch (status) {
    case 'online':
      return '在线'
    case 'offline':
      return '离线'
    case 'maintenance':
      return '维护中'
    default:
      return '未知'
  }
}

const handleMasterChange = () => {
  emitUpdate()
}

const handleWorkerChange = () => {
  emitUpdate()
}

const emitUpdate = () => {
  emit('update:modelValue', {
    masterNodes: masterNodes.value,
    workerNodes: workerNodes.value
  })
}

const selectAllMasters = () => {
  masterNodes.value = selectableMasters.value.map(machine => String(machine.id))
  emitUpdate()
}

const clearAllMasters = () => {
  masterNodes.value = []
  emitUpdate()
}

const selectAllWorkers = () => {
  workerNodes.value = selectableWorkers.value.map(machine => String(machine.id))
  emitUpdate()
}

const clearAllWorkers = () => {
  workerNodes.value = []
  emitUpdate()
}
</script>

<style scoped>
.node-select-container {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

.node-card {
  max-height: 300px;
  overflow-y: auto;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.machine-checkbox-item {
  margin-bottom: 10px;
  display: flex;
  align-items: center;
}

.machine-checkbox {
  width: 100%;
  height: auto;
  align-items: center;
  margin-right: 0;
}

.machine-checkbox :deep(.el-checkbox__input) {
  align-self: center;
}

.machine-offline {
  opacity: 0.75;
}

.machine-checkbox :deep(.el-checkbox__label) {
  width: 100%;
  line-height: 1.4;
  padding-left: 8px;
}

.machine-item {
  padding: 10px 12px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background-color: var(--el-fill-color-blank);
  width: 100%;
  box-sizing: border-box;
}

/* 表格式网格：每列纵向对齐 */
.machine-grid {
  display: grid;
  grid-template-columns: minmax(100px, 1fr) minmax(100px, 1fr) 72px 72px 72px auto;
  align-items: center;
  gap: 12px 16px;
  font-size: 13px;
}

.col-name {
  font-weight: 600;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.col-ip {
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.col-num {
  color: var(--el-text-color-secondary);
  text-align: right;
  white-space: nowrap;
}

.col-status {
  justify-self: end;
  display: flex;
  align-items: center;
  gap: 6px;
}

.status-loading {
  font-size: 16px;
  color: var(--el-color-primary);
}

.status-collecting-text {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.machine-collecting {
  opacity: 0.9;
}

.empty-text {
  text-align: center;
  color: #9ca3af;
  padding: 20px;
}

@media (max-width: 1200px) {
  .node-select-container {
    grid-template-columns: 1fr;
  }
}
</style>