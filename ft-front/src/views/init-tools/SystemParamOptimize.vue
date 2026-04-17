<template>
  <div class="system-param-optimize">
    <div class="page-header">
      <h2>系统参数优化</h2>
    </div>

    <div class="content-container">
      <el-card class="content-card">
        <template #header>
          <div class="card-header">
            <h3>系统参数配置</h3>
            <el-button
              type="primary"
              size="small"
              @click="refreshSystemParams"
              :loading="loadingSystemParams"
            >
              <el-icon><RefreshRight /></el-icon>
              刷新
            </el-button>
          </div>
        </template>

        <div class="system-params-container">
          <el-table
            v-loading="loadingSystemParams"
            :data="systemParams"
            style="width: 100%"
            stripe
          >
            <el-table-column
              prop="key"
              label="参数名称"
              min-width="200"
            >
              <template #default="scope">
                <div class="param-name">
                  <span class="key">{{ scope.row.key }}</span>
                  <el-tag v-if="scope.row.required" type="danger" size="small">必填</el-tag>
                </div>
              </template>
            </el-table-column>
            <el-table-column
              prop="value"
              label="参数值"
              min-width="250"
            >
              <template #default="scope">
                <el-input
                  v-model="scope.row.value"
                  placeholder="请输入参数值"
                  clearable
                />
              </template>
            </el-table-column>
            <el-table-column
              prop="description"
              label="描述"
              min-width="300"
            >
              <template #default="scope">
                <el-tooltip
                  :content="scope.row.description"
                  placement="top"
                >
                  <div class="param-description">{{ scope.row.description }}</div>
                </el-tooltip>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <div class="card-actions">
          <el-button
            type="success"
            @click="applySystemParams"
            :loading="applyingSystemParams"
          >
            <el-icon><Check /></el-icon>
            确认需求
          </el-button>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { RefreshRight, Check } from '@element-plus/icons-vue'

// 系统参数优化
const loadingSystemParams = ref(false)
const applyingSystemParams = ref(false)

interface SystemParam {
  key: string
  value: string
  description: string
  required: boolean
}

const systemParams = ref<SystemParam[]>([
  { key: 'vm.swappiness', value: '10', description: '内存交换策略，降低值减少交换', required: true },
  { key: 'net.core.somaxconn', value: '65535', description: '最大连接数', required: true },
  { key: 'net.ipv4.tcp_max_tw_buckets', value: '6000', description: 'TIME_WAIT状态的最大连接数', required: true },
  { key: 'net.ipv4.tcp_slow_start_after_idle', value: '0', description: '关闭TCP慢启动', required: true },
  { key: 'net.ipv4.tcp_syncookies', value: '1', description: '启用SYN cookies', required: true },
  { key: 'fs.file-max', value: '655350', description: '系统最大文件句柄数', required: true },
  { key: 'kernel.sysrq', value: '1', description: '启用SysRq功能', required: false },
  { key: 'kernel.panic', value: '60', description: '系统崩溃后自动重启时间', required: false }
])

// 刷新系统参数
const refreshSystemParams = () => {
  loadingSystemParams.value = true
  // 模拟API请求
  setTimeout(() => {
    ElMessage.success('系统参数已刷新')
    loadingSystemParams.value = false
  }, 1000)
}

// 应用系统参数
const applySystemParams = () => {
  applyingSystemParams.value = true
  // 检查必填参数
  const missingParams = systemParams.value.filter(p => p.required && !p.value.trim())
  if (missingParams.length > 0) {
    ElMessage.error(`缺少必填参数: ${missingParams.map(p => p.key).join(', ')}`)
    applyingSystemParams.value = false
    return
  }

  // 模拟API请求
  setTimeout(() => {
    ElMessage.success('系统参数优化已完成')
    applyingSystemParams.value = false
  }, 2000)
}
</script>

<style scoped>
.system-param-optimize {
  padding: 20px;
  box-sizing: border-box;
}

.page-header {
  margin-bottom: 20px;
  padding-bottom: 10px;
  border-bottom: 1px solid #e5e7eb;
}

.page-header h2 {
  color: #1890ff;
  margin: 0;
  font-size: 20px;
  font-weight: 600;
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

/* 系统参数优化 */
.system-params-container {
  margin-bottom: 20px;
}

.param-name {
  display: flex;
  align-items: center;
  gap: 8px;
}

.param-name .key {
  font-weight: 500;
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