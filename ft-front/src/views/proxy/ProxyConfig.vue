<template>
  <div class="proxy-config">
    <div class="page-header">
      <h2>代理配置管理</h2>
      <el-button type="primary" icon="Plus" @click="handleAddConfig">
        新增配置
      </el-button>
    </div>
    
    <!-- 搜索和筛选区域 -->
    <div class="search-filters">
      <el-input
        v-model="proxyStore.filters.name"
        placeholder="搜索配置名称"
        :prefix-icon="Search"
        clearable
        @clear="handleSearch"
        @keyup.enter="handleSearch"
        class="search-input"
      />
      
      <el-select
        v-model="proxyStore.filters.status"
        placeholder="选择状态"
        clearable
        @change="handleSearch"
        class="filter-select"
      >
        <el-option label="活跃" value="active" />
        <el-option label="非活跃" value="inactive" />
        <el-option label="草稿" value="draft" />
      </el-select>
      
      <el-button type="primary" @click="handleSearch">
        <el-icon><Search /></el-icon>
        搜索
      </el-button>
      
      <el-button @click="handleReset">
        <el-icon><RefreshRight /></el-icon>
        重置
      </el-button>
    </div>
    
    <!-- 配置列表表格 -->
    <div class="config-table">
      <el-table
        v-loading="proxyStore.loading"
        :data="proxyStore.proxyConfigList"
        stripe
        border
      >
        <el-table-column prop="name" label="配置名称" min-width="150">
          <template #default="scope">
            <el-link type="primary" @click="handleEditConfig(scope.row)">
              {{ scope.row.name }}
            </el-link>
          </template>
        </el-table-column>
        
        <el-table-column prop="description" label="描述" min-width="200" />
        
        <el-table-column prop="status" label="状态" width="100" align="center">
          <template #default="scope">
            <el-tag
              :type="getStatusType(scope.row.status)"
              size="small"
            >
              {{ getStatusText(scope.row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        
        <el-table-column prop="created_at" label="创建时间" min-width="180" />
        
        <el-table-column prop="updated_at" label="更新时间" min-width="180" />
        
        <el-table-column label="操作" width="200" align="center">
          <template #default="scope">
            <el-button
              type="primary"
              size="small"
              @click="handleEditConfig(scope.row)"
              :icon="Edit"
            >
              编辑
            </el-button>
            
            <el-button
              type="success"
              size="small"
              @click="handleApplyConfig(scope.row.id)"
              :icon="Check"
              :disabled="scope.row.status === 'active'"
            >
              应用
            </el-button>
            
            <el-button
              type="danger"
              size="small"
              @click="handleDeleteConfig(scope.row.id)"
              :icon="Delete"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
    
    <!-- 分页 -->
    <div class="pagination">
      <el-pagination
        v-model:current-page="proxyStore.filters.page"
        v-model:page-size="proxyStore.filters.pageSize"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        :total="proxyStore.total"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>
    
    <!-- 配置编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑配置' : '新增配置'"
      width="80%"
    >
      <div class="config-editor">
        <el-tabs v-model="activeTab">
          <!-- 基本信息 -->
          <el-tab-pane label="基本信息" name="basic">
            <el-form
              ref="basicFormRef"
              :model="configForm.basic"
              :rules="basicRules"
              label-width="120px"
              class="config-form"
            >
              <el-form-item label="配置名称" prop="name">
                <el-input
                  v-model="configForm.basic.name"
                  placeholder="请输入配置名称"
                  clearable
                />
              </el-form-item>
              
              <el-form-item label="描述" prop="description">
                <el-input
                  v-model="configForm.basic.description"
                  placeholder="请输入配置描述"
                  type="textarea"
                  rows="3"
                  clearable
                />
              </el-form-item>
              
              <el-form-item label="状态" prop="status">
                <el-select
                  v-model="configForm.basic.status"
                  placeholder="请选择配置状态"
                >
                  <el-option label="活跃" value="active" />
                  <el-option label="非活跃" value="inactive" />
                  <el-option label="草稿" value="draft" />
                </el-select>
              </el-form-item>
            </el-form>
          </el-tab-pane>
          
          <!-- 全局配置 -->
          <el-tab-pane label="全局配置" name="global">
            <el-form
              :model="configForm.global"
              label-width="150px"
              class="config-form"
            >
              <el-form-item label="工作进程数量">
                <el-select v-model="configForm.global.worker_processes">
                  <el-option label="自动" value="auto" />
                  <el-option label="1" value="1" />
                  <el-option label="2" value="2" />
                  <el-option label="4" value="4" />
                  <el-option label="8" value="8" />
                </el-select>
              </el-form-item>
              
              <el-form-item label="PID文件路径">
                <el-input
                  v-model="configForm.global.pid"
                  placeholder="/var/run/nginx.pid"
                  clearable
                />
              </el-form-item>
              
              <el-form-item label="错误日志路径">
                <el-input
                  v-model="configForm.global.error_log"
                  placeholder="logs/error.log"
                  clearable
                />
              </el-form-item>
            </el-form>
          </el-tab-pane>
          
          <!-- Events配置 -->
          <el-tab-pane label="Events配置" name="events">
            <el-form
              :model="configForm.events"
              label-width="150px"
              class="config-form"
            >
              <el-form-item label="工作进程连接数">
                <el-input-number
                  v-model="configForm.events.worker_connections"
                  :min="1024"
                  :max="65535"
                  :step="1024"
                  placeholder="4096"
                />
              </el-form-item>
              
              <el-form-item label="事件模型">
                <el-select v-model="configForm.events.use">
                  <el-option label="epoll" value="epoll" />
                  <el-option label="kqueue" value="kqueue" />
                  <el-option label="select" value="select" />
                  <el-option label="poll" value="poll" />
                </el-select>
              </el-form-item>
            </el-form>
          </el-tab-pane>
          
          <!-- HTTP配置 -->
          <el-tab-pane label="HTTP配置" name="http">
            <el-form
              :model="configForm.http"
              label-width="150px"
              class="config-form"
            >
              <el-form-item label="启用gzip压缩">
                <el-switch v-model="configForm.http.gzip" />
              </el-form-item>
              
              <el-form-item label="gzip压缩级别" v-if="configForm.http.gzip">
                <el-slider
                  v-model="configForm.http.gzip_comp_level"
                  :min="1"
                  :max="9"
                  :step="1"
                />
                <div class="slider-value">{{ configForm.http.gzip_comp_level }}</div>
              </el-form-item>
              
              <el-form-item label="长连接超时">
                <el-input-number
                  v-model="configForm.http.keepalive_timeout"
                  :min="10"
                  :max="120"
                  :step="10"
                  placeholder="65"
                  suffix="秒"
                />
              </el-form-item>
              
              <el-form-item label="隐藏服务器版本">
                <el-switch v-model="configForm.http.server_tokens" />
              </el-form-item>
              
              <el-form-item label="启用TCP NOPUSH">
                <el-switch v-model="configForm.http.tcp_nopush" />
              </el-form-item>
              
              <el-form-item label="启用TCP NODELAY">
                <el-switch v-model="configForm.http.tcp_nodelay" />
              </el-form-item>
              
              <el-form-item label="启用Sendfile">
                <el-switch v-model="configForm.http.sendfile" />
              </el-form-item>
            </el-form>
          </el-tab-pane>
          
          <!-- 服务器配置 -->
          <el-tab-pane label="服务器配置" name="server">
            <div class="server-configs">
              <div 
                v-for="(server, index) in configForm.server" 
                :key="server.id || index"
                class="server-config-item"
              >
                <div class="server-header">
                  <h3>服务器 {{ index + 1 }}</h3>
                  <el-button 
                    type="danger" 
                    size="small" 
                    icon="Delete" 
                    @click="handleDeleteServer(index)"
                    v-if="configForm.server.length > 1"
                  >
                    删除
                  </el-button>
                </div>
                <el-form
                  :model="server"
                  label-width="150px"
                  class="config-form"
                >
                  <el-form-item label="监听端口">
                    <el-input-number
                      v-model="server.listen"
                      :min="1"
                      :max="65535"
                      placeholder="80"
                    />
                  </el-form-item>
                  
                  <el-form-item label="服务器名称">
                    <el-input
                      v-model="server.server_name"
                      placeholder="example.com"
                      clearable
                    >
                      <template #append>
                        <el-button @click="handleAddServerName(index)">添加</el-button>
                      </template>
                    </el-input>
                    <div class="tag-list">
                      <el-tag 
                        v-for="(name, idx) in server.server_name" 
                        :key="idx"
                        closable
                        @close="handleRemoveServerName(index, idx)"
                      >
                        {{ name }}
                      </el-tag>
                    </div>
                  </el-form-item>
                  
                  <el-form-item label="根目录">
                    <el-input
                      v-model="server.root"
                      placeholder="/var/www/html"
                      clearable
                    />
                  </el-form-item>
                  
                  <el-form-item label="索引文件">
                    <el-input
                      v-model="server.index"
                      placeholder="index.html"
                      clearable
                    >
                      <template #append>
                        <el-button @click="handleAddIndex(server)">添加</el-button>
                      </template>
                    </el-input>
                    <div class="tag-list">
                      <el-tag 
                        v-for="(index, idx) in server.index" 
                        :key="idx"
                        closable
                        @close="handleRemoveIndex(server, idx)"
                      >
                        {{ index }}
                      </el-tag>
                    </div>
                  </el-form-item>
                </el-form>
              </div>
              <el-button type="primary" icon="Plus" @click="handleAddServer">
                新增服务器
              </el-button>
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>
      
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="handleCancel">取消</el-button>
          <el-button
            type="primary"
            :loading="dialogLoading"
            @click="handleSaveConfig"
          >
            保存
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, RefreshRight, Edit, Check, Delete } from '@element-plus/icons-vue'
import { useProxyStore } from '../../stores/proxy'
import type { ProxyConfig, SaveProxyConfigParams } from '../../types/proxy'

const proxyStore = useProxyStore()
const basicFormRef = ref()
const dialogVisible = ref(false)
const dialogLoading = ref(false)
const isEdit = ref(false)
const activeTab = ref('basic')

// 配置表单
const configForm = reactive({
  basic: {
    name: '',
    description: '',
    status: 'draft'
  },
  global: {
    worker_processes: 'auto',
    worker_connections: 4096,
    error_log: 'logs/error.log',
    pid: '/var/run/nginx.pid'
  },
  events: {
    worker_connections: 4096,
    use: 'epoll'
  },
  http: {
    include: ['mime.types'],
    default_type: 'application/octet-stream',
    sendfile: true,
    tcp_nopush: true,
    tcp_nodelay: true,
    keepalive_timeout: 65,
    gzip: true,
    gzip_comp_level: 5,
    gzip_types: ['text/plain', 'text/css', 'application/json', 'application/javascript', 'text/xml', 'text/javascript'],
    server_tokens: false
  },
  server: [
    {
      id: '',
      listen: 80,
      server_name: [] as string[],
      root: '/var/www/html',
      index: ['index.html', 'index.htm']
    }
  ]
})

// 基本信息验证规则
const basicRules = reactive({
  name: [
    { required: true, message: '请输入配置名称', trigger: 'blur' },
    { min: 2, max: 50, message: '配置名称长度在 2 到 50 个字符', trigger: 'blur' }
  ],
  status: [
    { required: true, message: '请选择配置状态', trigger: 'change' }
  ]
})

// 页面加载时获取配置列表
onMounted(() => {
  fetchProxyConfigList()
})

// 获取代理配置列表
const fetchProxyConfigList = () => {
  proxyStore.fetchProxyConfigList()
}

// 处理搜索
const handleSearch = () => {
  proxyStore.filters.page = 1
  fetchProxyConfigList()
}

// 处理重置
const handleReset = () => {
  proxyStore.resetFilters()
  fetchProxyConfigList()
}

// 处理分页大小变化
const handleSizeChange = (size: number) => {
  proxyStore.filters.pageSize = size
  fetchProxyConfigList()
}

// 处理当前页变化
const handleCurrentChange = (current: number) => {
  proxyStore.filters.page = current
  fetchProxyConfigList()
}

// 处理添加配置
const handleAddConfig = () => {
  isEdit.value = false
  resetConfigForm()
  dialogVisible.value = true
  activeTab.value = 'basic'
}

// 处理编辑配置
const handleEditConfig = (config: ProxyConfig) => {
  isEdit.value = true
  proxyStore.setCurrentConfig(config)
  loadConfigForm(config)
  dialogVisible.value = true
  activeTab.value = 'basic'
}

// 加载配置到表单
const loadConfigForm = (config: ProxyConfig) => {
  // 加载基本信息
  configForm.basic.name = config.name
  configForm.basic.description = config.description || ''
  configForm.basic.status = config.status || 'draft'
  
  // 加载全局配置
  if (config.global) {
    configForm.global = { ...configForm.global, ...config.global }
  }
  
  // 加载events配置
  if (config.events) {
    configForm.events = { ...configForm.events, ...config.events }
  }
  
  // 加载http配置
  if (config.http) {
    configForm.http = { ...configForm.http, ...config.http }
  }
  
  // 加载服务器配置
  if (config.server) {
    configForm.server = config.server.map(s => ({
      id: s.id || '',
      listen: s.listen || 80,
      server_name: s.server_name || [],
      root: s.root || '/var/www/html',
      index: s.index || ['index.html', 'index.htm']
    }))
  } else {
    configForm.server = [{
      id: '',
      listen: 80,
      server_name: [],
      root: '/var/www/html',
      index: ['index.html', 'index.htm']
    }]
  }
}

// 重置配置表单
const resetConfigForm = () => {
  // 重置基本信息
  configForm.basic.name = ''
  configForm.basic.description = ''
  configForm.basic.status = 'draft'
  
  // 重置全局配置
  configForm.global = {
    worker_processes: 'auto',
    worker_connections: 4096,
    error_log: 'logs/error.log',
    pid: '/var/run/nginx.pid'
  }
  
  // 重置events配置
  configForm.events = {
    worker_connections: 4096,
    use: 'epoll'
  }
  
  // 重置http配置
  configForm.http = {
    include: ['mime.types'],
    default_type: 'application/octet-stream',
    sendfile: true,
    tcp_nopush: true,
    tcp_nodelay: true,
    keepalive_timeout: 65,
    gzip: true,
    gzip_comp_level: 5,
    gzip_types: ['text/plain', 'text/css', 'application/json', 'application/javascript', 'text/xml', 'text/javascript'],
    server_tokens: false
  }
  
  // 重置服务器配置
  configForm.server = [
    {
      id: '',
      listen: 80,
      server_name: [],
      root: '/var/www/html',
      index: ['index.html', 'index.htm']
    }
  ]
  
  if (basicFormRef.value) {
    basicFormRef.value.resetFields()
  }
}

// 处理保存配置
const handleSaveConfig = async () => {
  if (!basicFormRef.value) return
  
  try {
    await basicFormRef.value.validate()
    dialogLoading.value = true
    
    const saveData: SaveProxyConfigParams = {
      name: configForm.basic.name,
      description: configForm.basic.description,
      status: configForm.basic.status as 'active' | 'inactive' | 'draft',
      config: {
        global: configForm.global,
        events: configForm.events,
        http: configForm.http,
        server: configForm.server
      }
    }
    
    let result
    if (isEdit.value && proxyStore.currentConfig?.id) {
      // 更新配置
      result = await proxyStore.updateExistingProxyConfig(proxyStore.currentConfig.id, saveData)
    } else {
      // 保存新配置
      result = await proxyStore.saveNewProxyConfig(saveData)
    }
    
    if (result) {
      ElMessage.success(isEdit.value ? '更新成功' : '保存成功')
      dialogVisible.value = false
      proxyStore.setCurrentConfig(null)
      resetConfigForm()
      fetchProxyConfigList()
    } else {
      ElMessage.error(isEdit.value ? '更新失败' : '保存失败')
    }
  } catch (error) {
    console.error('表单验证失败:', error)
  } finally {
    dialogLoading.value = false
  }
}

// 处理删除配置
const handleDeleteConfig = (id: string) => {
  ElMessageBox.confirm('确定要删除该配置吗？', '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    const success = await proxyStore.removeProxyConfig(id)
    if (success) {
      ElMessage.success('删除成功')
      fetchProxyConfigList()
    } else {
      ElMessage.error('删除失败')
    }
  }).catch(() => {
    // 取消删除
  })
}

// 处理应用配置
const handleApplyConfig = (id: string) => {
  ElMessageBox.confirm('确定要应用该配置吗？', '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    const success = await proxyStore.applyExistingProxyConfig(id)
    if (success) {
      ElMessage.success('应用成功')
      fetchProxyConfigList()
    } else {
      ElMessage.error('应用失败')
    }
  }).catch(() => {
    // 取消应用
  })
}

// 处理取消
const handleCancel = () => {
  dialogVisible.value = false
  proxyStore.setCurrentConfig(null)
  resetConfigForm()
}

// 处理添加服务器
const handleAddServer = () => {
  configForm.server.push({
    id: '',
    listen: 80,
    server_name: [] as string[],
    root: '/var/www/html',
    index: ['index.html', 'index.htm']
  })
}

// 处理删除服务器
const handleDeleteServer = (index: number) => {
  configForm.server.splice(index, 1)
}

// 处理添加服务器名称
const handleAddServerName = (index: number) => {
  const server = configForm.server[index]
  if (server && server.server_name && server.server_name.length > 0) {
    const name = server.server_name[0]
    if (name) {
      server.server_name.push(name)
      server.server_name.shift()
    }
  }
}

// 处理移除服务器名称
const handleRemoveServerName = (serverIndex: number, nameIndex: number) => {
  const server = configForm.server[serverIndex]
  if (server && server.server_name) {
    server.server_name.splice(nameIndex, 1)
  }
}

// 处理添加索引文件
const handleAddIndex = (server: any) => {
  if (server && server.index && server.index.length > 0) {
    const index = server.index[0]
    if (index) {
      server.index.push(index)
      server.index.shift()
    }
  }
}

// 处理移除索引文件
const handleRemoveIndex = (server: any, index: number) => {
  if (server && server.index) {
    server.index.splice(index, 1)
  }
}

// 获取状态类型
const getStatusType = (status: string) => {
  switch (status) {
    case 'active':
      return 'success'
    case 'inactive':
      return 'warning'
    case 'draft':
      return 'info'
    default:
      return 'info'
  }
}

// 获取状态文本
const getStatusText = (status: string) => {
  switch (status) {
    case 'active':
      return '活跃'
    case 'inactive':
      return '非活跃'
    case 'draft':
      return '草稿'
    default:
      return status
  }
}
</script>

<style scoped>
.proxy-config {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 0;
  color: #303133;
}

/* 搜索和筛选区域 */
.search-filters {
  display: flex;
  gap: 15px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}

.search-input {
  width: 250px;
}

.filter-select {
  width: 150px;
}

/* 配置表格 */
.config-table {
  margin-bottom: 20px;
}

/* 分页 */
.pagination {
  text-align: right;
  margin-bottom: 20px;
}

/* 配置编辑器 */
.config-editor {
  max-height: calc(100vh - 200px);
  overflow-y: auto;
  padding: 10px;
  min-height: 400px;
}

/* 表单样式 */
.config-form {
  margin-bottom: 20px;
}

/* 服务器配置项 */
.server-configs {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.server-config-item {
  border: 1px solid #e5e7eb;
  border-radius: 4px;
  padding: 15px;
}

.server-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.server-header h3 {
  margin: 0;
  color: #303133;
}

/* 标签列表 */
.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}

/* 滑块值 */
.slider-value {
  text-align: center;
  margin-top: 10px;
  color: #606266;
}
</style>
