<template>
  <div class="time-sync">
    <div class="page-header">
      <h2>时间同步</h2>
      <p>配置服务器时间同步，确保系统时间准确性</p>
    </div>

    <div class="content-container">
      <el-card class="content-card">
        <template #header>
          <div class="card-header">
            <h3>时间同步配置</h3>
            <el-button
              type="primary"
              size="small"
              @click="refreshMachineList"
              :loading="loadingMachineList"
            >
              <el-icon><RefreshRight /></el-icon>
              刷新
            </el-button>
          </div>
        </template>

        <div class="time-sync-container">
          <!-- 主服务器选择 -->
          <div class="section">
            <h4>时间主服务器</h4>
            <el-radio-group v-model="timeSyncConfig.masterType">
              <el-radio label="machine">从机器列表选择</el-radio>
              <el-radio label="custom">手动输入</el-radio>
            </el-radio-group>

            <div v-if="timeSyncConfig.masterType === 'machine'" class="master-machine-select">
              <el-select
                v-model="timeSyncConfig.masterMachine"
                placeholder="请选择主服务器机器"
                style="width: 100%"
              >
                <el-option
                  v-for="machine in timeSyncMachineList"
                  :key="machine.id"
                  :label="`${machine.name} (ID: ${machine.id})`"
                  :value="machine"
                />
              </el-select>
            </div>

            <div v-if="timeSyncConfig.masterType === 'custom'" class="master-custom-input">
              <el-input
                v-model="timeSyncConfig.customMaster"
                placeholder="请输入主服务器地址（IP或域名）"
                style="width: 100%"
                clearable
              />
            </div>
          </div>

          <!-- 从服务器选择 -->
          <div class="section">
            <h4>时间从服务器</h4>
            <el-select
              v-model="timeSyncConfig.slaveMachines"
              placeholder="请选择需要同步时间的机器"
              style="width: 100%"
              multiple
              filterable
              collapse-tags
            >
              <el-option
                v-for="machine in timeSyncMachineList"
                :key="machine.id"
                :label="`${machine.name} (ID: ${machine.id})`"
                :value="machine"
              />
            </el-select>
            <div class="machine-count">已选择 {{ timeSyncConfig.slaveMachines.length }} 台机器</div>
          </div>

          <!-- 同步选项 -->
          <div class="section">
            <h4>同步选项</h4>
            <el-form :model="timeSyncOptions" label-width="100px">
              <el-form-item label="同步间隔">
                <el-input-number
                  v-model="timeSyncOptions.syncInterval"
                  :min="1"
                  :max="60"
                  :step="1"
                  :precision="0"
                  style="width: 150px"
                />
                <span class="unit">分钟</span>
              </el-form-item>
              <el-form-item label="同步时区">
                <el-select v-model="timeSyncOptions.timezone" style="width: 200px">
                  <el-option label="Asia/Shanghai (CST)" value="Asia/Shanghai" />
                  <el-option label="UTC" value="UTC" />
                  <el-option label="Europe/London (GMT)" value="Europe/London" />
                  <el-option label="America/New_York (EST)" value="America/New_York" />
                </el-select>
              </el-form-item>
              <el-form-item label="启用NTP服务">
                <el-switch v-model="timeSyncOptions.enableNtp" />
              </el-form-item>
            </el-form>
          </div>
        </div>

        <div class="card-actions">
          <el-button
            type="success"
            @click="syncTime"
            :disabled="!canSyncTime"
            :loading="syncingTime"
          >
            <el-icon><Timer /></el-icon>
            同步时间
          </el-button>
          <el-button @click="resetTimeSync">
            <el-icon><RefreshRight /></el-icon>
            重置
          </el-button>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { RefreshRight, Timer } from '@element-plus/icons-vue'
import { getMachineList } from '../../api/index'
import type { Machine } from '../../types'

// 时间同步
const loadingMachineList = ref(false)
const syncingTime = ref(false)
const timeSyncMachineList = ref<Machine[]>([])

const timeSyncConfig = reactive({
  masterType: 'machine',
  masterMachine: null as Machine | null,
  customMaster: '',
  slaveMachines: [] as Machine[]
})

const timeSyncOptions = reactive({
  syncInterval: 15,
  timezone: 'Asia/Shanghai',
  enableNtp: true
})

const canSyncTime = computed(() => {
  if (timeSyncConfig.masterType === 'machine') {
    return !!timeSyncConfig.masterMachine && timeSyncConfig.slaveMachines.length > 0
  } else {
    return !!timeSyncConfig.customMaster.trim() && timeSyncConfig.slaveMachines.length > 0
  }
})

// 加载机器列表
const loadMachineList = async () => {
  loadingMachineList.value = true
  try {
    const response = await getMachineList({ page: 1, pageSize: 100 })
    timeSyncMachineList.value = response.list || []
  } catch (error: any) {
    ElMessage.error('获取机器列表失败: ' + (error.msg || error.message))
  } finally {
    loadingMachineList.value = false
  }
}

// 刷新机器列表
const refreshMachineList = () => {
  loadMachineList()
}

// 同步时间
const syncTime = () => {
  syncingTime.value = true
  // 模拟API请求
  setTimeout(() => {
    ElMessage.success('时间同步已完成')
    syncingTime.value = false
  }, 2000)
}

// 重置时间同步配置
const resetTimeSync = () => {
  timeSyncConfig.masterType = 'machine'
  timeSyncConfig.masterMachine = null
  timeSyncConfig.customMaster = ''
  timeSyncConfig.slaveMachines = []
  timeSyncOptions.syncInterval = 15
  timeSyncOptions.timezone = 'Asia/Shanghai'
  timeSyncOptions.enableNtp = true
  ElMessage.info('时间同步配置已重置')
}

// 组件挂载时加载机器列表
loadMachineList()
</script>

<style scoped>
.time-sync {
  padding: 20px;
  box-sizing: border-box;
}

.page-header {
  text-align: center;
  margin-bottom: 30px;
}

.page-header h2 {
  color: #1890ff;
  margin-bottom: 10px;
}

.content-container {
  height: calc(100% - 100px);
  overflow: auto;
}

.content-card {
  max-width: 100%;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-header h3 {
  margin: 0;
  color: #374151;
  font-size: 16px;
  font-weight: 600;
}

/* 时间同步 */
.time-sync-container {
  margin-bottom: 20px;
}

.section {
  margin-bottom: 20px;
}

.section h4 {
  margin: 0 0 15px 0;
  color: #374151;
  font-size: 14px;
  font-weight: 600;
}

.master-machine-select,
.master-custom-input {
  margin-top: 10px;
}

.machine-count {
  margin-top: 10px;
  font-size: 14px;
  color: #6b7280;
}

.unit {
  margin-left: 10px;
  color: #6b7280;
}

/* 卡片操作 */
.card-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid #e5e7eb;
}
</style>