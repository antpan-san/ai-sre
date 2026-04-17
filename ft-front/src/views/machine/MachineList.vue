<template>
  <div class="machine-list">
    <div class="page-header">
      <h2>机器管理</h2>
    </div>

    <!-- 搜索和筛选区域 -->
    <div class="search-filters">
      <el-input
        v-model="machineStore.filters.name"
        placeholder="搜索机器名称 / IP"
        :prefix-icon="Search"
        clearable
        @clear="handleSearch"
        @keyup.enter="handleSearch"
        class="search-input"
      />

      <el-select
        v-model="machineStore.filters.status"
        placeholder="选择状态"
        clearable
        @change="handleSearch"
        class="filter-select"
      >
        <el-option label="在线" value="online" />
        <el-option label="离线" value="offline" />
        <el-option label="维护中" value="maintenance" />
      </el-select>

      <el-select
        v-model="machineStore.filters.cluster_id"
        placeholder="选择集群"
        clearable
        filterable
        @change="handleSearch"
        class="filter-select"
      >
        <el-option
          v-for="cid in machineStore.clusterList"
          :key="cid"
          :label="cid.slice(0, 8) + '...'"
          :value="cid"
        />
      </el-select>

      <el-date-picker
        v-model="dateRange"
        type="daterange"
        range-separator="至"
        start-placeholder="开始日期"
        end-placeholder="结束日期"
        format="YYYY-MM-DD"
        value-format="YYYY-MM-DD"
        @change="handleDateChange"
        class="date-picker"
      />

      <el-button type="primary" @click="handleSearch">
        <el-icon><Search /></el-icon>
        搜索
      </el-button>

      <el-button @click="handleReset">
        <el-icon><RefreshRight /></el-icon>
        重置
      </el-button>

      <el-button type="success" :disabled="!isAdmin" @click="handleAdd">
        <el-icon><Plus /></el-icon>
        新增机器
      </el-button>
    </div>

    <!-- 工具栏 -->
    <div class="toolbar">
      <el-button size="small" @click="toggleExpandAll">
        <el-icon>
          <FolderOpened v-if="allExpanded" />
          <Folder v-else />
        </el-icon>
        {{ allExpanded ? '全部折叠' : '全部展开' }}
      </el-button>

      <el-button
        v-if="selectedIds.length > 0"
        type="danger"
        size="small"
        :disabled="!isAdmin"
        @click="handleBatchDelete"
      >
        <el-icon><Delete /></el-icon>
        批量删除 ({{ selectedIds.length }})
      </el-button>
    </div>

    <!-- 机器列表表格 - 使用 Element Plus 树形表格 -->
    <div class="machine-table">
      <el-table
        v-loading="machineStore.loading"
        :data="machineStore.treeData"
        stripe
        border
        row-key="id"
        :default-expand-all="allExpanded"
        :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
        @selection-change="handleSelectionChange"
        style="width: 100%"
      >
        <el-table-column type="selection" width="45" />

        <!-- 机器名称（含角色标签） -->
        <el-table-column prop="name" label="机器名称" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">
            <div class="machine-name-cell">
              <el-tag
                :type="getRoleBadgeType(row.node_role)"
                size="small"
                effect="dark"
                class="role-tag"
              >
                {{ getRoleLabel(row.node_role) }}
              </el-tag>
              <span class="machine-name-text">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="ip" label="IP 地址" min-width="130" align="center" />

        <el-table-column prop="os_version" label="系统版本" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">
            <span v-if="row.os_version">{{ row.os_version }}</span>
            <span v-else class="no-heartbeat">--</span>
          </template>
        </el-table-column>

        <el-table-column label="CPU" min-width="90" align="center">
          <template #default="{ row }">
            <span>{{ row.cpu_cores || row.cpu }} 核</span>
          </template>
        </el-table-column>

        <el-table-column label="内存" min-width="120" align="center">
          <template #default="{ row }">
            <div v-if="row.status === 'online' && row.memory_usage !== undefined && row.memory_usage !== null">
              <el-progress
                :percentage="Math.round(row.memory_usage)"
                :stroke-width="14"
                :color="getUsageColor(row.memory_usage)"
                :text-inside="true"
              />
              <span class="metric-detail">{{ formatBytes(row.memory_used) }} / {{ formatBytes(row.memory_total) }}</span>
            </div>
            <span v-else>{{ row.memory }} GB</span>
          </template>
        </el-table-column>

        <el-table-column label="磁盘" min-width="120" align="center">
          <template #default="{ row }">
            <div v-if="row.status === 'online' && row.disk_usage !== undefined && row.disk_usage !== null">
              <el-progress
                :percentage="Math.round(row.disk_usage)"
                :stroke-width="14"
                :color="getUsageColor(row.disk_usage)"
                :text-inside="true"
              />
              <span class="metric-detail">{{ formatBytes(row.disk_used) }} / {{ formatBytes(row.disk_total) }}</span>
            </div>
            <span v-else>{{ row.disk }} GB</span>
          </template>
        </el-table-column>

        <el-table-column prop="status" label="状态" min-width="100" align="center">
          <template #default="{ row }">
            <div class="status-cell">
              <span
                v-if="isRecentlyHeartbeated(row)"
                class="heartbeat-pulse"
              ></span>
              <el-tag :type="getStatusType(row.status)" size="small">
                {{ getStatusText(row.status) }}
              </el-tag>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="探测信息" min-width="100" align="center">
          <template #default="{ row }">
            <span v-if="row.node_role !== 'worker'" class="no-heartbeat">--</span>
            <el-tooltip
              v-else-if="getWorkerProbeIssue(row)"
              :content="getWorkerProbeIssue(row)"
              placement="top"
              effect="light"
              popper-class="probe-error-tooltip"
            >
              <el-tag type="danger" size="small">探测失败</el-tag>
            </el-tooltip>
            <el-tag v-else type="success" size="small" effect="plain">正常</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="心跳" min-width="130" align="center">
          <template #default="{ row }">
            <span v-if="row.last_heartbeat_at" class="heartbeat-text">
              {{ formatTime(row.last_heartbeat_at) }}
            </span>
            <span v-else class="no-heartbeat">--</span>
          </template>
        </el-table-column>

        <el-table-column prop="created_at" label="创建时间" min-width="140" align="center">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>

        <el-table-column label="操作" min-width="200" align="center" fixed="right">
          <template #default="{ row }">
            <div class="action-btns">
              <!-- Master: 注册受控节点按钮 (核心操作) -->
              <el-button
                v-if="row.node_role === 'master'"
                size="small"
                type="success"
                :disabled="!isAdmin || row.status !== 'online'"
                @click.stop="handleRegister(row)"
              >
                注册节点
              </el-button>

              <!-- 通用操作按钮组（文本化，减少视觉噪音） -->
              <el-button
                size="small"
                link
                type="primary"
                :disabled="!isAdmin"
                @click.stop="handleEdit(row)"
              >
                编辑
              </el-button>
              <el-button
                size="small"
                link
                type="danger"
                :disabled="!isAdmin"
                @click.stop="handleDelete(row.id)"
              >
                删除
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 分页 -->
    <div class="pagination">
      <el-pagination
        v-model:current-page="machineStore.filters.page"
        v-model:page-size="machineStore.filters.pageSize"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        :total="machineStore.total"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>

    <!-- 添加/编辑机器对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑机器' : '新增机器'"
      width="560px"
      destroy-on-close
    >
      <el-form
        ref="machineFormRef"
        :model="machineForm"
        :rules="machineRules"
        label-position="top"
      >
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="机器名称" prop="name">
              <el-input v-model="machineForm.name" placeholder="请输入机器名称" clearable />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="IP 地址" prop="ip">
              <el-input v-model="machineForm.ip" placeholder="请输入 IP 地址" clearable />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="16">
          <el-col :span="8">
            <el-form-item label="CPU（核）" prop="cpu">
              <el-input-number v-model="machineForm.cpu" :min="1" :max="128" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="内存（GB）" prop="memory">
              <el-input-number v-model="machineForm.memory" :min="1" :max="1024" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="磁盘（GB）" prop="disk">
              <el-input-number v-model="machineForm.disk" :min="10" :max="10240" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="节点角色" prop="node_role">
              <el-select v-model="machineForm.node_role" placeholder="请选择角色" style="width: 100%">
                <el-option label="Master" value="master" />
                <el-option label="Worker" value="worker" />
                <el-option label="Standalone" value="standalone" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="状态" prop="status">
              <el-select v-model="machineForm.status" placeholder="请选择状态" style="width: 100%">
                <el-option label="在线" value="online" />
                <el-option label="离线" value="offline" />
                <el-option label="维护中" value="maintenance" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <!-- Worker 关联 Master -->
        <el-form-item
          v-if="machineForm.node_role === 'worker'"
          label="所属 Master"
          prop="master_machine_id"
        >
          <el-select
            v-model="machineForm.master_machine_id"
            placeholder="请选择所属 Master"
            filterable
            style="width: 100%"
          >
            <el-option
              v-for="m in masterOptions"
              :key="m.id"
              :label="`${m.name} (${m.ip})`"
              :value="m.id"
            />
          </el-select>
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" :loading="dialogLoading" @click="handleDialogSubmit">
            确认
          </el-button>
        </div>
      </template>
    </el-dialog>
    <!-- 注册受控节点对话框 -->
    <el-dialog
      v-model="registerDialogVisible"
      title="注册受控节点"
      width="720px"
      destroy-on-close
      :close-on-click-modal="false"
    >
      <!-- Master 信息展示 -->
      <div class="register-master-info">
        <el-descriptions :column="3" border size="small">
          <el-descriptions-item label="主控节点">{{ registerMasterMachine?.name }}</el-descriptions-item>
          <el-descriptions-item label="IP 地址">{{ registerMasterMachine?.ip }}</el-descriptions-item>
          <el-descriptions-item label="集群">{{ registerMasterMachine?.cluster_id ? registerMasterMachine.cluster_id.slice(0, 8) + '...' : '未分配' }}</el-descriptions-item>
        </el-descriptions>
      </div>

      <!-- 受控节点列表 -->
      <div class="register-worker-list">
        <div
          v-for="(worker, index) in registerWorkers"
          :key="index"
          class="register-worker-card"
        >
          <div class="register-worker-header">
            <span class="register-worker-title">受控节点 #{{ index + 1 }}</span>
            <el-button
              v-if="registerWorkers.length > 1"
              type="danger"
              text
              size="small"
              :icon="Delete"
              @click="removeRegisterWorker(index)"
            >
              移除
            </el-button>
          </div>
          <el-form :model="worker" label-position="top" size="small">
            <el-row :gutter="12">
              <el-col :span="8">
                <el-form-item label="IP 地址" required>
                  <el-input v-model="worker.ip" placeholder="如 192.168.1.10" />
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="主机名">
                  <el-input v-model="worker.hostname" placeholder="选填" />
                </el-form-item>
              </el-col>
              <el-col :span="4">
                <el-form-item label="SSH 端口">
                  <el-input-number v-model="worker.ssh_port" :min="1" :max="65535" controls-position="right" style="width: 100%" />
                </el-form-item>
              </el-col>
              <el-col :span="4">
                <el-form-item label="SSH 用户">
                  <el-input v-model="worker.ssh_user" placeholder="root" />
                </el-form-item>
              </el-col>
            </el-row>

            <!-- 认证方式切换 -->
            <el-form-item label="认证方式" required>
              <el-radio-group v-model="worker.auth_type" class="auth-type-radio">
                <el-radio value="password">密码认证</el-radio>
                <el-radio value="key">密钥认证</el-radio>
              </el-radio-group>
            </el-form-item>

            <!-- 密码认证 -->
            <el-form-item
              v-if="worker.auth_type === 'password'"
              label="SSH 密码"
              required
            >
              <el-input
                v-model="worker.ssh_password"
                type="password"
                placeholder="输入 SSH 登录密码"
                show-password
              />
            </el-form-item>

            <!-- 密钥认证 -->
            <el-form-item
              v-if="worker.auth_type === 'key'"
              label="SSH 密钥路径"
              required
            >
              <el-input v-model="worker.ssh_key" placeholder="如 /root/.ssh/id_rsa" />
            </el-form-item>

            <el-alert
              v-if="worker.auth_type === 'password'"
              type="info"
              :closable="false"
              show-icon
              class="auth-tip"
            >
              <template #title>
                首次连接时将自动接受主机密钥（等同于 SSH 的 "yes" 确认）
              </template>
            </el-alert>
          </el-form>
        </div>
      </div>

      <el-button type="primary" text @click="addRegisterWorker" class="add-worker-btn">
        <el-icon><CirclePlus /></el-icon>
        添加受控节点
      </el-button>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="registerDialogVisible = false">取消</el-button>
          <el-button
            type="success"
            :loading="registerLoading"
            :disabled="!isRegisterFormValid"
            @click="submitRegisterWorkers"
          >
            提交注册
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance } from 'element-plus'
import {
  Search,
  RefreshRight,
  Plus,
  Delete,
  CirclePlus,
  FolderOpened,
  Folder
} from '@element-plus/icons-vue'
import { useMachineStore } from '../../stores/machine'
import { useUserStore } from '../../stores/user'
import { getTaskDetail } from '../../api/machine'
import type { Machine, MachineForm, MachineTreeNode, NodeRole, RegisterWorkerNode } from '../../types'

// ---------- stores ----------
const machineStore = useMachineStore()
const userStore = useUserStore()

// ---------- refs ----------
const machineFormRef = ref<FormInstance>()
const dialogVisible = ref(false)
const dialogLoading = ref(false)
const selectedIds = ref<string[]>([])
const isEdit = ref(false)
const dateRange = ref<[string, string] | null>(null)
const allExpanded = ref(true)

// ---------- 权限 ----------
const isAdmin = computed(() => {
  return userStore.currentUser?.role === 'admin'
})

// ---------- master 下拉选项（新增/编辑 worker 时使用） ----------
const masterOptions = computed(() => {
  return machineStore.machineList.filter(m => m.node_role === 'master')
})

// ---------- 表单 ----------
const machineForm = reactive<MachineForm>({
  id: undefined,
  name: '',
  ip: '',
  cpu: 1,
  memory: 4,
  disk: 100,
  status: 'online',
  node_role: 'standalone',
  cluster_id: null,
  master_machine_id: null
})

watch(
  () => machineForm.node_role,
  (role) => {
    if (role === 'master') {
      machineForm.master_machine_id = null
    }
    if (role === 'standalone') {
      machineForm.master_machine_id = null
      machineForm.cluster_id = null
    }
  }
)

const machineRules = reactive({
  name: [
    { required: true, message: '请输入机器名称', trigger: 'blur' },
    { min: 2, max: 50, message: '机器名称长度在 2 到 50 个字符', trigger: 'blur' }
  ],
  ip: [
    { required: true, message: '请输入 IP 地址', trigger: 'blur' },
    {
      pattern: /^((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$/,
      message: '请输入正确的 IP 地址格式',
      trigger: 'blur'
    }
  ],
  cpu: [
    { required: true, message: '请输入 CPU 核数', trigger: 'blur' },
    { type: 'number' as const, min: 1, message: 'CPU 核数至少为 1', trigger: 'blur' }
  ],
  memory: [
    { required: true, message: '请输入内存大小', trigger: 'blur' },
    { type: 'number' as const, min: 1, message: '内存大小至少为 1GB', trigger: 'blur' }
  ],
  disk: [
    { required: true, message: '请输入磁盘大小', trigger: 'blur' },
    { type: 'number' as const, min: 10, message: '磁盘大小至少为 10GB', trigger: 'blur' }
  ],
  status: [{ required: true, message: '请选择机器状态', trigger: 'change' }],
  node_role: [{ required: true, message: '请选择节点角色', trigger: 'change' }]
})

// ---------- 注册受控节点相关 ----------
const registerDialogVisible = ref(false)
const registerLoading = ref(false)
const registerMasterMachine = ref<Machine | null>(null)
const registerWorkers = ref<RegisterWorkerNode[]>([])

const createEmptyWorker = (): RegisterWorkerNode => ({
  ip: '',
  hostname: '',
  ssh_port: 22,
  ssh_user: 'root',
  auth_type: 'password',
  ssh_password: '',
  ssh_key: '/root/.ssh/id_rsa'
})

const addRegisterWorker = () => {
  registerWorkers.value.push(createEmptyWorker())
}

const removeRegisterWorker = (index: number) => {
  registerWorkers.value.splice(index, 1)
}

const isRegisterFormValid = computed(() => {
  if (registerWorkers.value.length === 0) return false
  return registerWorkers.value.every(w => {
    if (w.ip.trim() === '') return false
    if (w.auth_type === 'password') return w.ssh_password.trim() !== ''
    return w.ssh_key.trim() !== ''
  })
})

// ==================== 生命周期 ====================
onMounted(async () => {
  await fetchMachineList()
})

// ==================== 数据操作 ====================
const fetchMachineList = () => {
  return machineStore.fetchMachineList()
}

// 搜索
const handleSearch = () => {
  machineStore.filters.page = 1
  fetchMachineList()
}

// 日期范围
const handleDateChange = (val: [string, string] | null) => {
  if (val) {
    machineStore.filters.startDate = val[0]
    machineStore.filters.endDate = val[1]
  } else {
    machineStore.filters.startDate = ''
    machineStore.filters.endDate = ''
  }
  machineStore.filters.page = 1
  fetchMachineList()
}

// 重置
const handleReset = () => {
  machineStore.resetFilters()
  dateRange.value = null
  fetchMachineList()
}

// 分页
const handleSizeChange = (size: number) => {
  machineStore.filters.pageSize = size
  machineStore.filters.page = 1
  fetchMachineList()
}

const handleCurrentChange = (current: number) => {
  machineStore.filters.page = current
  fetchMachineList()
}

// 多选
const handleSelectionChange = (selection: MachineTreeNode[]) => {
  selectedIds.value = selection.map(item => item.id)
}

// 展开 / 折叠
const toggleExpandAll = () => {
  allExpanded.value = !allExpanded.value
  // 通过重新请求触发表格重新渲染，或使用 key 强制刷新
  fetchMachineList()
}

// ==================== CRUD ====================

// 新增
const handleAdd = () => {
  isEdit.value = false
  resetMachineForm()
  dialogVisible.value = true
}

// 编辑
const handleEdit = (row: Machine) => {
  isEdit.value = true
  Object.assign(machineForm, {
    id: row.id,
    name: row.name,
    ip: row.ip,
    cpu: row.cpu,
    memory: row.memory,
    disk: row.disk,
    status: row.status,
    node_role: row.node_role,
    cluster_id: row.cluster_id || null,
    master_machine_id: row.master_machine_id || null
  })
  dialogVisible.value = true
}

// 删除
const handleDelete = (id: string) => {
  ElMessageBox.confirm('确定要删除该机器吗？', '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
    .then(async () => {
      const success = await machineStore.removeMachine(id)
      if (success) {
        ElMessage.success('删除成功')
        selectedIds.value = selectedIds.value.filter(item => item !== id)
      } else {
        ElMessage.error('删除失败')
      }
    })
    .catch(() => {
      // 取消
    })
}

// 批量删除
const handleBatchDelete = () => {
  if (selectedIds.value.length === 0) {
    ElMessage.warning('请选择需要删除的机器')
    return
  }

  ElMessageBox.confirm(`确定要删除选中的 ${selectedIds.value.length} 台机器吗？`, '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
    .then(async () => {
      const success = await machineStore.batchRemoveMachine(selectedIds.value)
      if (success) {
        ElMessage.success('批量删除成功')
        selectedIds.value = []
      } else {
        ElMessage.error('批量删除失败')
      }
    })
    .catch(() => {
      // 取消
    })
}

// 注册受控节点 — 打开弹窗
const handleRegister = (row: Machine) => {
  registerMasterMachine.value = row
  registerWorkers.value = [createEmptyWorker()]
  registerDialogVisible.value = true
}

// 提交注册受控节点，并轮询任务执行状态
const submitRegisterWorkers = async () => {
  if (!registerMasterMachine.value) return
  registerLoading.value = true
  try {
    const result = await machineStore.registerWorkerNodes(
      registerMasterMachine.value.id,
      registerWorkers.value
    )
    if (!result) {
      ElMessage.error('注册失败')
      return
    }
    const taskId: string | undefined = (result as any)?.task_id
    registerDialogVisible.value = false

    if (taskId) {
      ElMessage.info('注册请求已提交，正在等待客户端响应…')
      pollRegisterTask(taskId)
    } else {
      ElMessage.success('注册请求已提交，受控节点将在下次心跳时同步')
    }
  } catch {
    ElMessage.error('注册失败')
  } finally {
    registerLoading.value = false
  }
}

// 轮询任务状态，最多等待 90 秒（心跳间隔 ≤30s，加 SSH 探测时间）
const pollRegisterTask = (taskId: string) => {
  const MAX_POLLS = 30
  const INTERVAL_MS = 3000
  let count = 0

  const poll = async () => {
    count++
    try {
      const detail = await getTaskDetail(taskId) as any
      const status: string = detail?.task?.status ?? ''
      if (status === 'success') {
        ElMessage.success('受控节点已与 Master 客户端建立连接，指标采集中…')
        // Refresh list to show latest worker statuses
        machineStore.fetchMachineList()
        return
      }
      if (status === 'failed' || status === 'timeout' || status === 'cancelled') {
        ElMessage.warning(`注册任务执行${status === 'failed' ? '失败' : status === 'timeout' ? '超时' : '已取消'}，请检查客户端连接及 SSH 凭证`)
        return
      }
    } catch {
      // non-fatal — keep polling
    }

    if (count < MAX_POLLS) {
      setTimeout(poll, INTERVAL_MS)
    } else {
      ElMessage.warning('等待超时：受控节点将在下次客户端心跳后完成同步，请稍后刷新列表')
    }
  }

  setTimeout(poll, INTERVAL_MS)
}

// 提交表单
const handleDialogSubmit = async () => {
  if (!machineFormRef.value) return
  try {
    await machineFormRef.value.validate()
    dialogLoading.value = true

    let result
    if (isEdit.value && machineForm.id) {
      result = await machineStore.updateExistingMachine(machineForm.id, machineForm)
    } else {
      result = await machineStore.addNewMachine(machineForm)
    }

    if (result) {
      ElMessage.success(isEdit.value ? '更新成功' : '添加成功')
      dialogVisible.value = false
      resetMachineForm()
    } else {
      ElMessage.error(isEdit.value ? '更新失败' : '添加失败')
    }
  } catch (error) {
    console.error('表单验证失败:', error)
  } finally {
    dialogLoading.value = false
  }
}

// 重置表单
const resetMachineForm = () => {
  Object.assign(machineForm, {
    id: undefined,
    name: '',
    ip: '',
    cpu: 1,
    memory: 4,
    disk: 100,
    status: 'online',
    node_role: 'standalone',
    cluster_id: null,
    master_machine_id: null
  })
  if (machineFormRef.value) {
    machineFormRef.value.resetFields()
  }
}

// ==================== 辅助函数 ====================

const getStatusType = (status: string) => {
  const map: Record<string, string> = {
    online: 'success',
    offline: 'danger',
    maintenance: 'warning'
  }
  return map[status] || 'info'
}

const getStatusText = (status: string) => {
  const map: Record<string, string> = {
    online: '在线',
    offline: '离线',
    maintenance: '维护中'
  }
  return map[status] || status
}

const getRoleBadgeType = (role: NodeRole) => {
  const map: Record<NodeRole, string> = {
    master: '',
    worker: 'success',
    standalone: 'info'
  }
  return map[role] || 'info'
}

const getRoleLabel = (role: NodeRole) => {
  const map: Record<NodeRole, string> = {
    master: 'Master',
    worker: 'Worker',
    standalone: 'Standalone'
  }
  return map[role] || role
}

const getWorkerProbeIssue = (row: any) => {
  const fromMetadata = row?.metadata?.probe_error
  if (typeof fromMetadata === 'string' && fromMetadata.trim() !== '') {
    return fromMetadata.trim()
  }
  const fromRealtime = row?.probe_error
  if (typeof fromRealtime === 'string' && fromRealtime.trim() !== '') {
    return fromRealtime.trim()
  }
  return ''
}

const formatTime = (time: string | null | undefined) => {
  if (!time) return '--'
  try {
    return new Date(time).toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    })
  } catch {
    return time
  }
}

// ==================== 实时指标工具函数 ====================

/**
 * 根据使用率返回进度条颜色
 */
const getUsageColor = (usage: number) => {
  if (usage >= 90) return '#F56C6C'  // 危险: 红色
  if (usage >= 70) return '#E6A23C'  // 警告: 橙色
  return '#67C23A'                    // 正常: 绿色
}

/**
 * 格式化字节为人类可读单位 (KB, MB, GB, TB)
 */
const formatBytes = (bytes: number | undefined | null) => {
  if (!bytes || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  const val = bytes / Math.pow(1024, i)
  return `${val.toFixed(1)} ${units[i]}`
}

/**
 * 判断机器是否在最近 15 秒内收到了心跳 (用于脉冲动画)
 */
const isRecentlyHeartbeated = (row: any) => {
  if (row.status !== 'online') return false
  if (!row.last_heartbeat_at) return false
  const lastHB = new Date(row.last_heartbeat_at).getTime()
  return Date.now() - lastHB < 15000
}
</script>

<style scoped>
.machine-list {
  padding: 0 20px 20px 20px;
}

.page-header h2 {
  margin: 0 0 20px 0;
  color: #1890ff;
  font-size: 30px;
  font-weight: 600;
}

.search-filters {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
  align-items: center;
}

.search-input {
  width: 250px;
}

.filter-select {
  width: 160px;
}

.date-picker {
  width: 300px;
}

.toolbar {
  display: flex;
  gap: 10px;
  margin-bottom: 12px;
  align-items: center;
}

.machine-table {
  margin-bottom: 20px;
  width: 100%;
}

.machine-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.role-tag {
  flex-shrink: 0;
}

.machine-name-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.heartbeat-text {
  color: #67c23a;
  font-size: 12px;
}

.no-heartbeat {
  color: #c0c4cc;
}

/* 探测失败悬浮提示：白色背景，具体原因仅悬浮显示，避免与表格 tooltip 重叠 */

.pagination {
  text-align: center;
  margin-bottom: 20px;
  width: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

/* ---- 实时指标相关样式 ---- */
.metric-detail {
  font-size: 11px;
  color: #909399;
  display: block;
  margin-top: 2px;
}

.status-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

/* 脉冲动画 - 标识最近收到心跳的机器 */
.heartbeat-pulse {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: #67C23A;
  display: inline-block;
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0% {
    box-shadow: 0 0 0 0 rgba(103, 194, 58, 0.6);
  }
  70% {
    box-shadow: 0 0 0 8px rgba(103, 194, 58, 0);
  }
  100% {
    box-shadow: 0 0 0 0 rgba(103, 194, 58, 0);
  }
}

/* ---- 操作列按钮布局 ---- */
.action-btns {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex-wrap: nowrap;
}

/* ---- 注册受控节点弹窗样式 ---- */
.register-master-info {
  margin-bottom: 20px;
}

.register-worker-list {
  max-height: 440px;
  overflow-y: auto;
  padding-right: 4px;
}

.register-worker-card {
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  padding: 14px 16px 4px;
  margin-bottom: 12px;
  background: #fafafa;
  transition: border-color 0.2s;
}
.register-worker-card:hover {
  border-color: #c0c4cc;
}

.register-worker-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.register-worker-title {
  font-weight: 600;
  font-size: 13px;
  color: #303133;
}

.add-worker-btn {
  margin-top: 8px;
}

.auth-type-radio {
  margin-top: 4px;
}

.auth-tip {
  margin-bottom: 12px;
  padding: 6px 12px;
}
</style>

<style>
/* 探测失败悬浮提示：白色背景（tooltip 渲染在 body，需全局样式） */
.probe-error-tooltip {
  background: #fff !important;
  border: 1px solid var(--el-border-color-light);
  color: var(--el-text-color-primary);
  max-width: 320px;
}
</style>
