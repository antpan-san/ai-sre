<template>
  <div class="job-center">
    <div class="page-header">
      <h2>作业中心</h2>
    </div>

    <el-card class="main-card">
      <!-- 机器列表区域 -->
      <div class="machine-section">
        <div class="section-header">
            <h3>机器列表</h3>
            <el-button
              type="primary"
              size="small"
              @click="refreshMachines"
              :loading="machineStore.loading"
            >
              <el-icon><RefreshRight /></el-icon>
              刷新
            </el-button>
          </div>
        
        <div class="machine-list-container">
          <el-transfer
            v-loading="machineStore.loading"
            v-model="transferValue"
            :data="transferData"
            :titles="['待选机器', '已选机器']"
            :filterable="true"
            filter-placeholder="搜索机器ID或IP"
            :format="{
              noMatch: '无匹配数据',
              noData: '无数据',
              all: '全部',
              confirm: '确认'
            }"
            @change="handleTransferChange"
          >
            <template #default="{ option }">
              <div 
              class="transfer-item" 
              :class="{
                'status-online': option.status === 'online',
                'status-offline': option.status === 'offline',
                'status-maintenance': option.status === 'maintenance'
              }"
            >
                <span class="item-id">ID: {{ option.id }}</span>
                <span class="item-ip">IP: {{ option.ip }}</span>
                <el-tag
                  v-if="option.status"
                  :type="option.status === 'online' ? 'success' : option.status === 'offline' ? 'danger' : 'warning'"
                  size="small"
                  class="item-status"
                >
                  {{ option.status === 'online' ? '在线' : option.status === 'offline' ? '离线' : '维护中' }}
                </el-tag>
              </div>
            </template>
          </el-transfer>
        </div>
        
        <div class="selection-info">
          <span>已选择 {{ selectedMachines.length }} 台机器</span>
        </div>
      </div>

      <!-- 执行命令和结果区域（左右结构） -->
      <div class="command-result-container">
        <!-- 命令输入区域（左） -->
        <div class="command-section">
          <div class="section-header">
            <h3>执行命令</h3>
          </div>
          
          <div class="command-input-container">
            <div class="terminal-content">
              <div class="terminal-prompt">$</div>
              <el-input
                v-model="commandText"
                type="textarea"
                :rows="12"
                placeholder="请输入要执行的命令，多条命令请用换行分隔"
                clearable
                class="command-textarea"
              />
            </div>
          </div>
          
          <!-- 命令语法检查错误显示 -->
          <div v-if="commandErrors.length > 0" class="command-errors-container">
            <div v-for="(error, index) in commandErrors" :key="index" class="command-error-item">
              <el-icon class="error-icon"><Warning /></el-icon>
              <span>{{ error }}</span>
            </div>
          </div>
          
          <div class="command-actions">
            <el-dropdown v-if="commandHistory.length > 0" trigger="click">
              <el-button type="info">
                <el-icon><Clock /></el-icon>
                命令历史
                <el-icon class="el-icon--right"><ArrowDown /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item
                    v-for="(command, index) in commandHistory"
                    :key="index"
                    @click="selectFromHistory(command)"
                    class="history-item"
                  >
                    <pre>{{ command }}</pre>
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
            <el-button
              type="primary"
              :disabled="selectedMachines.length === 0 || !commandText.trim()"
              @click="executeCommands"
              :loading="executing"
            >
              <el-icon><CirclePlus /></el-icon>
              执行命令
            </el-button>
            <el-button @click="clearCommand">
              <el-icon><Delete /></el-icon>
              清空
            </el-button>
          </div>
        </div>

        <!-- 执行结果区域（右） -->
        <div class="result-section">
          <div class="section-header">
            <h3>执行结果</h3>
            <div class="result-header-actions">
              <el-select 
                v-model="resultFilter"
                placeholder="过滤结果"
                size="small"
                style="width: 120px; margin-right: 10px;"
              >
                <el-option label="全部" value="all" />
                <el-option label="成功" value="success" />
                <el-option label="失败" value="failed" />
              </el-select>
              <el-button
                type="info"
                size="small"
                @click="clearResult"
              >
                <el-icon><Delete /></el-icon>
                清空结果
              </el-button>
            </div>
          </div>
          
          <div class="result-container">
            <div class="filter-status" v-if="resultFilter !== 'all'">
              当前显示: {{ resultFilter === 'success' ? '成功' : '失败' }} 的执行结果 (共 {{ filteredResults.length }} 条)
            </div>
            <div v-if="!filteredResults.length" class="empty-result">
              {{ executionResults.length > 0 ? '没有匹配的执行结果' : '执行结果将显示在这里' }}
            </div>
            
            <div
              v-for="(result, index) in filteredResults"
              :key="index"
              class="execution-result-item"
            >
              <div class="result-header">
                <div>
                  <span class="machine-name">{{ result.machineName }} ({{ result.machineId }})</span>
                  <el-tag
                    :type="result.success ? 'success' : 'danger'"
                    size="small"
                  >
                    {{ result.success ? '执行成功' : '执行失败' }}
                  </el-tag>
                </div>
                <span class="execution-time">{{ formatDate(result.executionTime) }}</span>
              </div>
              
              <div class="result-content">
                <div v-if="result.stdout" class="stdout">
                  <div class="content-label">标准输出:</div>
                  <pre>{{ result.stdout }}</pre>
                </div>
                <div v-if="result.stderr" class="stderr">
                  <div class="content-label">错误输出:</div>
                  <pre>{{ result.stderr }}</pre>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { RefreshRight, CirclePlus, Delete, Clock, ArrowDown, Warning } from '@element-plus/icons-vue'
import { executeCommand } from '../../api/index'
import { useMachineStore } from '../../stores/machine'
import type { Machine } from '../../types'

// 页面状态
const executing = ref(false)

// 初始化机器Store
const machineStore = useMachineStore()

// 机器列表数据
const selectedMachines = ref<Machine[]>([])
const transferValue = ref<string[]>([])

// 穿梭框数据转换
const transferData = computed(() => {
  return machineStore.machineList.map(machine => {
    const isOnline = machine.status === 'online'
    return {
      key: machine.id,
      label: `${machine.id} - ${machine.ip} (${isOnline ? '在线' : machine.status === 'offline' ? '离线' : '维护中'})`,
      id: machine.id,
      ip: machine.ip,
      status: machine.status,
      // 只有在线机器可选择
      disabled: !isOnline
    }
  })
})

// 命令输入
const commandText = ref('')
const commandErrors = ref<string[]>([])

// 命令语法检查
const checkCommandSyntax = (command: string) => {
  const errors: string[] = []
  const lines = command.trim().split('\n')
  
  // 基本语法检查规则
  lines.forEach((line, index) => {
    const trimmedLine = line.trim()
    
    // 检查空行
    if (trimmedLine === '') return
    
    // 检查危险命令
    const dangerousCommands = ['rm -rf', 'format', 'mkfs', 'dd if=/dev/zero']
    dangerousCommands.forEach(dangerousCmd => {
      if (trimmedLine.includes(dangerousCmd)) {
        errors.push(`第${index + 1}行: 检测到危险命令 "${dangerousCmd}"，请谨慎执行`)
      }
    })
    
    // 检查不支持的命令类型
    const unsupportedCommands = ['vi', 'vim', 'nano', 'emacs', 'top', 'htop']
    unsupportedCommands.forEach(cmd => {
      if (trimmedLine.startsWith(cmd)) {
        errors.push(`第${index + 1}行: 交互式命令 "${cmd}" 不被支持，请使用非交互式命令`)
      }
    })
  })
  
  commandErrors.value = errors
  return errors
}

// 命令历史记录
const commandHistory = ref<string[]>([])
const MAX_HISTORY_COUNT = 20

// 从localStorage加载命令历史
const loadCommandHistory = () => {
  const history = localStorage.getItem('commandHistory')
  if (history) {
    try {
      commandHistory.value = JSON.parse(history)
    } catch (error) {
      console.error('Failed to parse command history:', error)
      commandHistory.value = []
    }
  }
}

// 保存命令历史到localStorage
const saveCommandHistory = () => {
  localStorage.setItem('commandHistory', JSON.stringify(commandHistory.value))
}

// 添加命令到历史记录
const addToHistory = (command: string) => {
  if (!command.trim()) return
  
  // 移除重复的命令
  const index = commandHistory.value.indexOf(command)
  if (index > -1) {
    commandHistory.value.splice(index, 1)
  }
  
  // 添加到历史记录开头
  commandHistory.value.unshift(command)
  
  // 限制历史记录数量
  if (commandHistory.value.length > MAX_HISTORY_COUNT) {
    commandHistory.value.pop()
  }
  
  // 保存到localStorage
  saveCommandHistory()
}

// 从历史记录中选择命令
const selectFromHistory = (command: string) => {
  commandText.value = command
}

// 监听命令输入变化，自动进行语法检查
watch(commandText, (newValue) => {
  checkCommandSyntax(newValue)
})

// 执行结果
interface ExecutionResult {
  machineId: number
  machineName: string
  success: boolean
  stdout: string
  stderr: string
  executionTime: string
}

const executionResults = ref<ExecutionResult[]>([])
const resultFilter = ref<string>('all')

// 过滤后的执行结果
const filteredResults = computed(() => {
  if (resultFilter.value === 'all') {
    return executionResults.value
  } else if (resultFilter.value === 'success') {
    return executionResults.value.filter(result => result.success)
  } else if (resultFilter.value === 'failed') {
    return executionResults.value.filter(result => !result.success)
  }
  return executionResults.value
})

// 键盘快捷键
const setupKeyboardShortcuts = () => {
  const handleKeydown = (e: KeyboardEvent) => {
    // Ctrl+Enter 执行命令
    if (e.ctrlKey && e.key === 'Enter') {
      e.preventDefault()
      if (selectedMachines.value.length > 0 && commandText.value.trim()) {
        executeCommands()
      }
    }
    
    // Ctrl+K 清空命令
    if (e.ctrlKey && e.key === 'k') {
      e.preventDefault()
      clearCommand()
    }
    
    // Ctrl+R 刷新机器列表
    if (e.ctrlKey && e.key === 'r') {
      e.preventDefault()
      refreshMachines()
    }
    
    // Ctrl+L 清空结果
    if (e.ctrlKey && e.key === 'l') {
      e.preventDefault()
      clearResult()
    }
  }
  
  window.addEventListener('keydown', handleKeydown)
  
  return () => {
    window.removeEventListener('keydown', handleKeydown)
  }
}

// 存放键盘快捷键清理函数
let cleanupKeyboardShortcuts: (() => void) | null = null

// 初始化数据
onMounted(() => {
  loadMachineList()
  loadCommandHistory()
  cleanupKeyboardShortcuts = setupKeyboardShortcuts()
})

// 组件卸载时清理事件监听器，防止内存泄漏
onUnmounted(() => {
  if (cleanupKeyboardShortcuts) {
    cleanupKeyboardShortcuts()
    cleanupKeyboardShortcuts = null
  }
})

// 加载机器列表
const loadMachineList = async () => {
  try {
    await machineStore.fetchMachineList({ page: 1, pageSize: 1000 })
    
    // 清空已选机器，防止加载新数据后选中状态不正确
    transferValue.value = []
    selectedMachines.value = []
  } catch (error: any) {
    ElMessage.error('获取机器列表失败: ' + (error.msg || error.message))
  }
}

// 刷新机器列表
const refreshMachines = () => {
  loadMachineList()
}

// 处理穿梭框选择变化
const handleTransferChange = (value: string[]) => {
  transferValue.value = value
  selectedMachines.value = machineStore.machineList.filter(machine => 
    value.includes(machine.id)
  )
}

// 执行命令
const executeCommands = async () => {
  if (selectedMachines.value.length === 0) {
    ElMessage.warning('请至少选择一台机器')
    return
  }
  
  if (!commandText.value.trim()) {
    ElMessage.warning('请输入要执行的命令')
    return
  }
  
  // 执行命令前进行语法检查
  const errors = checkCommandSyntax(commandText.value)
  if (errors.length > 0) {
    // 显示确认对话框
    const confirmResult = await ElMessageBox.confirm(
      `检测到${errors.length}个潜在问题，是否继续执行？\n\n${errors.join('\n')}`,
      '命令执行警告',
      {
        confirmButtonText: '继续执行',
        cancelButtonText: '取消',
        type: 'warning',
        dangerouslyUseHTMLString: true
      }
    )
    
    if (confirmResult !== 'confirm') {
      return
    }
  }
  
  // 保存命令到历史记录
  addToHistory(commandText.value.trim())
  
  executing.value = true
  try {
    // 响应拦截器已解包，response 即为 data 部分
    const response = await executeCommand({
      machine_ids: selectedMachines.value.map(machine => machine.id),
      command: commandText.value.trim()
    }) as any
    
    // 处理执行结果
    const results = response?.results || []
    const newResults: ExecutionResult[] = results.map((result: any) => ({
      machineId: result.machineId,
      machineName: selectedMachines.value.find(m => m.id === result.machineId)?.name || `机器${result.machineId}`,
      success: result.success,
      stdout: result.stdout || '',
      stderr: result.stderr || '',
      executionTime: new Date().toISOString()
    }))
    
    executionResults.value.unshift(...newResults)
    ElMessage.success(`命令已在${selectedMachines.value.length}台机器上执行`)
  } catch (error: any) {
    ElMessage.error('执行命令失败: ' + (error.msg || error.message))
  } finally {
    executing.value = false
  }
}

// 清空命令
const clearCommand = () => {
  commandText.value = ''
}

// 清空结果
const clearResult = () => {
  executionResults.value = []
}

// 格式化日期
const formatDate = (dateString: string): string => {
  if (!dateString) return ''
  const date = new Date(dateString)
  return date.toLocaleString('zh-CN')
}
</script>

<style scoped>
.job-center {
  padding: 0 20px 20px 20px;
}

.page-header {
  margin-bottom: 20px;
  padding-bottom: 10px;
}

.page-header h2 {
  color: #1890ff;
  margin: 0;
  font-size: 30px;
  font-weight: 600;
}

.main-card {
  max-width: 100%;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 10px;
  border-bottom: 1px solid #e5e7eb;
}

.result-header-actions {
  display: flex;
  align-items: center;
}

.section-header h3 {
  margin: 0;
  color: #374151;
  font-size: 16px;
  font-weight: 600;
}

/* 机器列表区域 */
.machine-section {
  margin-bottom: 20px;
}

.machine-list-container {
  background-color: #ffffff;
  border-radius: 8px;
  padding: 10px;
  border: 1px solid #e5e7eb;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  min-height: 250px;
  width: 100%;
}

/* 最彻底的Element Plus样式重置 */
.machine-list-container :deep(.el-transfer) {
  width: 100%;
}

.machine-list-container :deep(.el-transfer-panel) {
  height: auto !important;
}

.machine-list-container :deep(.el-transfer-panel__header) {
  padding: 8px 12px !important;
}

.machine-list-container :deep(.el-transfer-panel__body-wrapper) {
  overflow: hidden;
}

/* 完全重置列表项的所有间距 */
.machine-list-container :deep(.el-transfer-panel__body .el-checkbox) {
  position: relative;
  display: block;
  height: 28px !important;
  line-height: 28px !important;
  margin: 0 !important;
  padding: 0 12px !important;
  cursor: pointer;
}

.machine-list-container :deep(.el-transfer-panel__body .el-checkbox__input) {
  float: left;
  margin-top: 5px !important;
  margin-right: 8px !important;
  vertical-align: middle;
  line-height: 1;
}

.machine-list-container :deep(.el-transfer-panel__body .el-checkbox__label) {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  height: 28px !important;
  line-height: 28px !important;
  margin-left: 28px !important;
  vertical-align: middle;
}

/* 无数据状态居中显示 */
.machine-list-container :deep(.el-transfer-panel__empty) {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
  min-height: 200px;
  margin: 0;
  padding: 0;
  font-size: 14px;
  color: #909399;
}

/* 当无数据时隐藏滚动条 */
.machine-list-container :deep(.el-transfer-panel) {
  position: relative;
}

/* 无数据状态时移除所有滚动条 */
.machine-list-container :deep(.el-transfer-panel__empty) {
  overflow: hidden !important;
}

.machine-list-container :deep(.el-transfer-panel__empty) + .el-scrollbar {
  display: none !important;
}

/* 修复Element Plus内部样式导致的滚动条问题 */
.machine-list-container :deep(.el-transfer-panel__body-wrapper) {
  overflow: hidden !important;
}

.machine-list-container :deep(.el-transfer-panel__body-wrapper) .el-scrollbar {
  overflow: hidden !important;
}

/* 只有当有列表项时才显示垂直滚动条 */
.machine-list-container :deep(.el-transfer-panel__list-item) ~ .el-scrollbar {
  overflow-y: auto !important;
}

/* 确保整个复选框元素垂直对齐 */
.machine-list-container :deep(.el-transfer-panel__body .el-checkbox) {
  display: flex !important;
  align-items: center !important;
  height: 28px !important;
  line-height: 28px !important;
  padding: 0 12px !important;
}

/* 调整复选框内部样式，确保垂直对齐 */
.machine-list-container :deep(.el-transfer-panel__body .el-checkbox__input) {
  margin: 0 !important;
  vertical-align: middle;
  display: flex;
  align-items: center;
  justify-content: center;
}

.machine-list-container :deep(.el-transfer-panel__body .el-checkbox__input input) {
  margin: 0 !important;
  padding: 0 !important;
  vertical-align: middle;
}

.machine-list-container :deep(.el-transfer) {
  display: flex;
  width: 100%;
  justify-content: space-between;
}

.machine-list-container :deep(.el-transfer-panel) {
  width: 48%;
  min-width: 200px;
}

.machine-list-container :deep(.el-transfer-panel__body) {
  max-height: 280px;
  overflow-x: hidden;
  overflow-y: auto;
}

/* 彻底去除横向滚动条 */
.machine-list-container :deep(.el-transfer-panel__body),
.machine-list-container :deep(.el-transfer-panel__body-wrapper),
.machine-list-container :deep(.el-transfer-panel__body-wrapper .el-scrollbar),
.machine-list-container :deep(.el-transfer-panel__body-wrapper .el-scrollbar__wrap),
.machine-list-container :deep(.el-transfer-panel__body-wrapper .el-scrollbar__view) {
  overflow-x: hidden !important;
  overflow-y: auto !important;
}

/* 隐藏滚动条轨道但保留滚动功能 */
.machine-list-container :deep(.el-scrollbar__wrap) {
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.machine-list-container :deep(.el-scrollbar__wrap::-webkit-scrollbar) {
  display: none;
}

/* 确保内容不会溢出 */
.machine-list-container :deep(.el-transfer-panel__list-item) {
  width: 100% !important;
  box-sizing: border-box !important;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.machine-list-container :deep(.el-transfer__buttons) {
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  margin: 0 12px;
  width: 40px;
  gap: 6px;
  height: 100%;
  /* 更精确的垂直居中 */
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  align-self: center;
  margin: 0 12px;
  width: 40px;
  gap: 6px;
  height: auto;
}

.machine-list-container :deep(.el-transfer__buttons button) {
  width: 36px;
  height: 32px;
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 0;
  margin: 0;
  border: none;
}

.machine-list-container :deep(.el-transfer__buttons button .el-icon) {
  font-size: 16px;
  margin: 0;
}

/* 确保按钮容器在垂直方向居中 */
.machine-list-container :deep(.el-transfer) {
  align-items: stretch;
}

.machine-list-container :deep(.el-transfer__buttons) {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
}

.transfer-item {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 15px;
  padding: 2px 0 !important;
  border-radius: 0;
  height: auto;
  box-sizing: border-box;
  width: 100%;
  margin: 0 !important;
  background-color: transparent;
  border: none;
  transition: all 0.3s;
  position: relative;
  z-index: 1;
  line-height: 1.2;
  min-height: 24px;
}

.transfer-item > * {
  margin: 0 !important;
  padding: 0 !important;
  line-height: 1.2;
}

.transfer-item:hover {
  background-color: #f0f9ff;
  border-radius: 0;
  z-index: 2;
}

/* 修复Element Plus内部样式冲突 */
.machine-list-container :deep(.el-transfer-panel__list) {
  list-style: none;
  padding: 0;
  margin: 0;
  line-height: 1.2;
}

.machine-list-container :deep(.el-transfer-panel__list-item) {
  padding: 0 !important;
  margin: 0 !important;
  line-height: 1.2 !important;
  height: auto !important;
  min-height: 24px !important;
  box-sizing: border-box !important;
  overflow: hidden !important;
  border-bottom: 1px solid #f5f5f5;
}

/* 直接覆盖Element Plus的默认样式 */
.machine-list-container :deep(.el-transfer-panel__body) {
  padding: 0 !important;
}

.machine-list-container :deep(.el-transfer-panel__body .el-checkbox) {
  margin: 0 !important;
  padding: 0 12px !important;
  height: auto !important;
  min-height: 24px !important;
  line-height: 1.2 !important;
}

/* 移除所有可能导致间距的样式 */
.machine-list-container :deep(.el-transfer-panel__body *),
.machine-list-container :deep(.el-transfer-panel__list *),
.machine-list-container :deep(.el-transfer-panel__list-item *) {
  margin-top: 0 !important;
  margin-bottom: 0 !important;
  padding-top: 0 !important;
  padding-bottom: 0 !important;
  line-height: 1.2 !important;
}

.machine-list-container :deep(.el-transfer-panel__list-item:hover) {
  background-color: transparent;
}

.machine-list-container :deep(.el-transfer-panel__body .el-checkbox) {
  margin: 0;
  display: flex;
  align-items: center;
  height: 100%;
  padding: 2px 12px;
  width: 100%;
  box-sizing: border-box;
  line-height: 1.2;
}

.machine-list-container :deep(.el-transfer-panel__body .el-checkbox__input) {
  margin-right: 8px;
  flex-shrink: 0;
  margin-top: 0;
}

.machine-list-container :deep(.el-transfer-panel__body .el-checkbox__label) {
  display: block;
  width: 100%;
  margin: 0;
  padding: 0;
  line-height: 1.2;
}

.machine-list-container :deep(.el-transfer-panel__list) {
  line-height: 1.2;
}

.machine-list-container :deep(.el-transfer-panel__list-item) {
  line-height: 1.2;
  min-height: 28px;
}

.transfer-item:has(.status-offline) {
  opacity: 0.6;
  cursor: not-allowed;
}

.item-id {
  font-weight: 500;
  color: #374151;
  font-size: 14px;
  line-height: 1.5;
  width: 60px;
}

.item-ip {
  color: #6b7280;
  font-size: 13px;
  line-height: 1.5;
  width: 120px;
}

.item-status {
  font-size: 12px;
  margin: 0;
  width: 60px;
  text-align: center;
  flex-shrink: 0;
}

.transfer-item.status-online {
  background-color: rgba(212, 252, 231, 0.3);
}

.transfer-item.status-offline {
  background-color: rgba(254, 226, 226, 0.3);
}

.transfer-item.status-maintenance {
  background-color: rgba(252, 231, 186, 0.3);
}

.selection-info {
  margin-top: 10px;
  font-size: 14px;
  color: #6b7280;
}

/* 命令和结果容器 */
.command-result-container {
  display: flex;
  gap: 20px;
  margin-bottom: 20px;
  min-height: 300px;
}

/* 命令输入区域 */
.command-section {
  flex: 4;
  min-width: 300px;
  display: flex;
  flex-direction: column;
}

.command-input-container {
  margin-bottom: 15px;
  position: relative;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  overflow: hidden;
  transition: all 0.3s;
  flex: 1;
  min-height: 200px;
  background-color: #ffffff;
  color: #333333;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.command-input-container:hover {
  border-color: #409eff;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.command-input-container:focus-within {
  border-color: #409eff;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08), 0 0 0 2px rgba(64, 158, 255, 0.2);
}

/* 终端内容区域 */
.terminal-content {
  position: relative;
  padding: 8px;
  height: 100%;
  display: flex;
  flex-direction: row;
  align-items: flex-start;
}

.terminal-prompt {
  position: relative;
  top: 0;
  left: 0;
  margin-right: 8px;
  margin-left: 4px;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 14px;
  color: #1890ff;
  font-weight: bold;
  pointer-events: none;
  z-index: 1;
  flex-shrink: 0;
  line-height: 24px;
  padding-top: 2px;
}

.command-textarea {
  flex: 1;
  height: 100%;
}

.command-textarea .el-textarea__inner {
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 14px;
  line-height: 1.6;
  padding: 4px 8px;
  min-height: 184px;
  resize: vertical;
  background-color: transparent;
  border: none;
  color: #333333;
  box-shadow: none;
  overflow-y: auto;
}

.command-textarea .el-textarea__inner::placeholder {
  color: #909399;
}

.command-textarea .el-textarea__inner:focus {
  box-shadow: none;
  border: none;
  background-color: transparent;
}

/* 终端样式增强 */
.command-input-container {
  border-color: #dcdfe6;
}

.command-input-container:focus-within {
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

/* 清除按钮样式 */
.command-textarea .el-input__clear {
  color: #c0c4cc;
}

.command-textarea .el-input__clear:hover {
  color: #909399;
}

/* 命令语法错误显示样式 */
.command-errors-container {
  margin-bottom: 15px;
  padding: 10px 15px;
  background-color: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 8px;
}

.command-error-item {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 5px;
  color: #b91c1c;
  font-size: 13px;
  line-height: 1.4;
}

.command-error-item:last-child {
  margin-bottom: 0;
}

.error-icon {
  color: #ef4444;
  font-size: 14px;
}

/* 基本的命令语法高亮模拟 */
.command-textarea .el-textarea__inner::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  pointer-events: none;
  background-image: 
    linear-gradient(to right, transparent calc(2em + 15px), transparent 100%);
  z-index: 0;
}

.command-textarea .el-textarea__inner:focus {
  box-shadow: none;
}

.command-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.history-item {
  max-width: 500px;
  white-space: normal;
  overflow: hidden;
  text-overflow: ellipsis;
}

.history-item pre {
  margin: 0;
  padding: 0;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.4;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 80px;
  overflow-y: auto;
}

/* 执行结果区域 */
.result-section {
  flex: 5;
  min-width: 300px;
  display: flex;
  flex-direction: column;
}

.result-container {
  background-color: #f9fafb;
  border-radius: 8px;
  padding: 10px;
  border: 1px solid #e5e7eb;
  flex: 1;
  min-height: 200px;
  overflow-y: auto;
  scrollbar-width: thin;
  scrollbar-color: #d1d5db #f3f4f6;
}

.result-container::-webkit-scrollbar {
  width: 8px;
}

.result-container::-webkit-scrollbar-track {
  background: #f3f4f6;
  border-radius: 4px;
}

.result-container::-webkit-scrollbar-thumb {
  background: #d1d5db;
  border-radius: 4px;
}

.result-container::-webkit-scrollbar-thumb:hover {
  background: #9ca3af;
}

.filter-status {
  text-align: center;
  color: #6b7280;
  font-size: 13px;
  padding: 8px;
  background-color: #f3f4f6;
  border-radius: 4px;
  margin-bottom: 10px;
}

.empty-result {
  text-align: center;
  color: #9ca3af;
  padding: 40px 0;
  font-style: italic;
}

.execution-result-item {
  margin-bottom: 20px;
  padding: 15px;
  background-color: #ffffff;
  border-radius: 8px;
  border: 1px solid #e5e7eb;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  transition: all 0.3s;
}

.execution-result-item:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  border-color: #d1d5db;
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
  padding-bottom: 10px;
  border-bottom: 1px solid #f3f4f6;
  flex-wrap: wrap;
  gap: 10px;
}

.result-header > div:first-child {
  display: flex;
  align-items: center;
  gap: 8px;
}

.machine-name {
  font-weight: 600;
  color: #374151;
}

.execution-time {
  font-size: 12px;
  color: #9ca3af;
}

.result-content {
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
}

.content-label {
  font-weight: 600;
  margin-bottom: 5px;
  display: inline-block;
  color: #6b7280;
}

.stdout pre {
  margin: 0 0 10px 0;
  padding: 12px;
  background-color: #f3f4f6;
  border-radius: 6px;
  color: #1f2937;
  white-space: pre-wrap;
  word-break: break-all;
  border-left: 3px solid #409eff;
  box-shadow: inset 0 1px 2px rgba(0, 0, 0, 0.05);
}

.stderr pre {
  margin: 0;
  padding: 12px;
  background-color: #fee2e2;
  border-radius: 6px;
  color: #dc2626;
  white-space: pre-wrap;
  word-break: break-all;
  border-left: 3px solid #f56c6c;
  box-shadow: inset 0 1px 2px rgba(0, 0, 0, 0.05);
}

/* 响应式布局 */
@media (max-width: 992px) {
  /* 在中等屏幕上改为上下布局 */
  .command-result-container {
    flex-direction: column;
  }
  
  .command-section,
  .result-section {
    min-width: 100%;
  }
  
  .result-container {
    max-height: 400px;
  }
  
  .command-textarea .el-textarea__inner {
    font-size: 13px;
  }
  
  .result-header-actions {
    gap: 8px;
  }
  
  .result-header-actions .el-select {
    width: 110px;
  }
}

@media (max-width: 768px) {
  /* 小屏幕设备优化 */
  .section-header {
    gap: 8px;
    flex-wrap: wrap;
  }
  
  .section-header h3 {
    font-size: 15px;
  }
  
  .section-header .el-button {
    font-size: 12px;
    padding: 6px 10px;
  }
  
  /* 优化穿梭框在小屏幕上的显示 */
  .machine-list-container :deep(.el-transfer) {
    flex-direction: column;
    gap: 10px;
  }
  
  .machine-list-container :deep(.el-transfer-panel) {
    width: 100%;
    max-width: 100%;
  }
  
  .machine-list-container :deep(.el-transfer-panel__body) {
    max-height: 180px;
  }
  
  .machine-list-container :deep(.el-transfer__buttons) {
    flex-direction: row;
    gap: 10px;
    margin: 0;
    position: static;
    top: auto;
    transform: none;
  }
  
  .machine-list-container :deep(.el-transfer__buttons button) {
    padding: 6px 12px;
    margin: 0;
  }
  
  /* 确保按钮容器居中 */
  .machine-list-container :deep(.el-transfer__buttons) {
    justify-content: center;
    align-items: center;
  }
  
  .command-actions {
    gap: 8px;
  }
  
  .command-actions .el-button {
    font-size: 12px;
    padding: 8px 12px;
  }
  
  .command-textarea .el-textarea__inner {
    font-size: 12px;
    line-height: 1.4;
    min-height: 100px;
  }
  
  .terminal-prompt {
    font-size: 13px;
  }
  
  .result-header {
    gap: 8px;
  }
  
  .machine-name {
    font-size: 14px;
  }
  
  .execution-time {
    font-size: 11px;
  }
  
  .execution-result-item {
    padding: 12px;
  }
  
  .stdout pre,
  .stderr pre {
    padding: 8px;
    font-size: 12px;
    line-height: 1.4;
  }
  
  .history-item {
    max-width: 250px;
  }
  
  .history-item pre {
    font-size: 12px;
  }
  
  .filter-status {
    font-size: 12px;
    padding: 6px;
  }
}

/* 平板设备优化 */
@media (max-width: 1024px) and (min-width: 769px) {
  .command-actions {
    flex-wrap: wrap;
  }
  
  .history-item {
    max-width: 400px;
  }
  
  .machine-list-container :deep(.el-transfer-panel) {
    width: 45%;
  }
}
</style>