<template>
  <div class="user-list page-shell page-shell--crud-wide">
    <div class="user-page-head">
      <div class="user-page-head-left">
        <h2 class="user-page-title">用户管理</h2>
        <el-tag size="small" type="info" effect="plain">共 {{ userStore.total }} 人</el-tag>
        <el-popover placement="bottom-start" :width="280" trigger="click">
          <template #reference>
            <el-button text type="primary" class="user-help-btn">说明</el-button>
          </template>
          <p class="user-help-text">支持按用户名、角色筛选；UUID 为主键。删除与批量删除会跳过管理员账号。</p>
        </el-popover>
      </div>
      <el-button type="primary" :icon="Plus" @click="handleAdd">新增用户</el-button>
    </div>

    <el-card class="user-data-card" shadow="never">
      <div class="search-filters">
        <el-input
          v-model="userStore.filters.username"
          placeholder="搜索用户名"
          :prefix-icon="Search"
          clearable
          class="search-input"
          @keyup.enter="handleSearch"
          @clear="handleSearch"
        />
        <el-select
          v-model="userStore.filters.role"
          placeholder="角色"
          clearable
          class="filter-select"
          @change="handleSearch"
        >
          <el-option label="管理员" value="admin" />
          <el-option label="普通用户" value="user" />
        </el-select>
        <div class="search-filters-actions">
          <el-button type="primary" :icon="Search" @click="handleSearch">搜索</el-button>
          <el-button :icon="RefreshRight" @click="handleReset">重置</el-button>
        </div>
      </div>

      <div class="user-table-wrap">
        <el-table
          v-loading="userStore.loading"
          :data="userStore.userList"
          stripe
          border
          row-key="id"
          style="width: 100%"
          table-layout="auto"
          @selection-change="handleSelectionChange"
        >
          <el-table-column type="selection" width="52" :selectable="rowSelectable" />
          <el-table-column prop="id" label="用户 ID" min-width="200" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="user-id-cell">{{ row.id }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="username" label="用户名" min-width="140" show-overflow-tooltip />
          <el-table-column prop="email" label="邮箱" min-width="200" show-overflow-tooltip />
          <el-table-column prop="full_name" label="显示名" min-width="120" show-overflow-tooltip />
          <el-table-column prop="phone" label="手机" min-width="130" show-overflow-tooltip />
          <el-table-column prop="role" label="角色" width="112" align="center">
            <template #default="{ row }">
              <el-tag :type="row.role === 'admin' ? 'success' : 'info'" size="small">
                {{ row.role === 'admin' ? '管理员' : '普通用户' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="创建时间" min-width="172" align="center">
            <template #default="{ row }">
              {{ formatTime(row.created_at) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="260" align="center" fixed="right">
            <template #default="{ row }">
              <el-button type="primary" link size="small" :icon="Edit" @click="handleEdit(row)">编辑</el-button>
              <el-button type="primary" link size="small" :icon="SwitchButton" @click="handleChangeRole(row)">
                角色
              </el-button>
              <el-button
                type="danger"
                link
                size="small"
                :icon="Delete"
                :disabled="row.role === 'admin'"
                @click="handleDelete(row.id)"
              >
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div class="table-footer">
        <div v-if="selectedIds.length" class="batch-bar">
          <span>已选 {{ selectedIds.length }} 项</span>
          <el-button type="danger" size="small" :icon="Delete" @click="handleBatchDelete">批量删除</el-button>
        </div>
        <el-pagination
          v-model:current-page="userStore.filters.page"
          v-model:page-size="userStore.filters.pageSize"
          class="pagination"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          :total="userStore.total"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑用户' : '新增用户'" width="520px" destroy-on-close>
      <el-form ref="userFormRef" :model="userForm" :rules="userRules" label-position="top">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="userForm.username" placeholder="登录名" clearable :disabled="isEdit" />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="userForm.email" placeholder="name@example.com" clearable />
        </el-form-item>
        <el-form-item label="显示名" prop="full_name">
          <el-input v-model="userForm.full_name" placeholder="可选" clearable maxlength="100" />
        </el-form-item>
        <el-form-item label="手机" prop="phone">
          <el-input v-model="userForm.phone" placeholder="可选" clearable maxlength="20" />
        </el-form-item>
        <el-form-item v-if="!isEdit" label="初始密码" prop="password">
          <el-input v-model="userForm.password" type="password" show-password placeholder="至少 6 位" />
        </el-form-item>
        <el-form-item v-else label="重置密码（可选）" prop="password">
          <el-input v-model="userForm.password" type="password" show-password placeholder="留空则不修改" />
        </el-form-item>
        <el-form-item label="角色" prop="role">
          <el-select v-model="userForm.role" placeholder="角色" style="width: 100%">
            <el-option label="管理员" value="admin" />
            <el-option label="普通用户" value="user" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="dialogLoading" @click="handleDialogSubmit">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="roleDialogVisible" title="设置角色" width="400px" destroy-on-close>
      <div class="role-dialog-content">
        <p><strong>用户</strong> {{ selectedUser?.username }}</p>
        <p>
          当前角色
          <el-tag :type="selectedUser?.role === 'admin' ? 'success' : 'info'" size="small">
            {{ selectedUser?.role === 'admin' ? '管理员' : '普通用户' }}
          </el-tag>
        </p>
        <el-form-item label="新角色" class="role-select-item">
          <el-select v-model="newRole" placeholder="请选择" style="width: 100%">
            <el-option label="管理员" value="admin" />
            <el-option label="普通用户" value="user" />
          </el-select>
        </el-form-item>
      </div>
      <template #footer>
        <el-button @click="roleDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="roleDialogLoading" @click="handleRoleSubmit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, RefreshRight, Plus, Edit, Delete, SwitchButton } from '@element-plus/icons-vue'
import { useUserManagementStore } from '../../stores/userManagement'
import type { User, UserForm } from '../../types'

const userStore = useUserManagementStore()
const userFormRef = ref()
const dialogVisible = ref(false)
const dialogLoading = ref(false)
const roleDialogVisible = ref(false)
const roleDialogLoading = ref(false)
const selectedIds = ref<string[]>([])
const isEdit = ref(false)
const selectedUser = ref<User | null>(null)
const newRole = ref<string>('')

const userForm = reactive<UserForm>({
  id: undefined,
  username: '',
  email: '',
  phone: '',
  full_name: '',
  password: '',
  role: 'user'
})

const userRules = reactive({
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 50, message: '用户名 3–50 字符', trigger: 'blur' }
  ],
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email' as const, message: '邮箱格式不正确', trigger: 'blur' }
  ],
  phone: [
    {
      validator: (_r: unknown, v: string, cb: (e?: Error) => void) => {
        if (!v || !v.trim()) return cb()
        if (!/^1[3-9]\d{9}$/.test(v)) return cb(new Error('请输入正确手机号'))
        cb()
      },
      trigger: 'blur'
    }
  ],
  password: [
    {
      validator: (_r: unknown, v: string, cb: (e?: Error) => void) => {
        if (!isEdit.value) {
          if (!v || v.length < 6) return cb(new Error('密码至少 6 位'))
        } else if (v && v.length < 6) {
          return cb(new Error('密码至少 6 位'))
        }
        cb()
      },
      trigger: 'blur'
    }
  ],
  role: [{ required: true, message: '请选择角色', trigger: 'change' }]
})

function formatTime(s?: string) {
  if (!s) return '—'
  return String(s).replace('T', ' ').slice(0, 19)
}

function rowSelectable(row: User) {
  return row.role !== 'admin'
}

onMounted(() => {
  userStore.fetchUserList()
})

const fetchUserList = () => {
  userStore.fetchUserList()
}

const handleSearch = () => {
  userStore.filters.page = 1
  fetchUserList()
}

const handleReset = () => {
  userStore.resetFilters()
  fetchUserList()
}

const handleSizeChange = () => {
  fetchUserList()
}

const handleCurrentChange = () => {
  fetchUserList()
}

const handleSelectionChange = (selection: User[]) => {
  selectedIds.value = selection.map((item) => item.id)
}

const handleAdd = () => {
  isEdit.value = false
  resetUserForm()
  dialogVisible.value = true
}

const handleEdit = (user: User) => {
  isEdit.value = true
  Object.assign(userForm, {
    id: user.id,
    username: user.username,
    email: user.email || '',
    phone: user.phone || '',
    full_name: user.full_name || '',
    password: '',
    role: user.role
  })
  dialogVisible.value = true
}

const handleDelete = (id: string) => {
  ElMessageBox.confirm('确定删除该用户？', '提示', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning'
  })
    .then(async () => {
      const ok = await userStore.removeUser(id)
      if (ok) {
        ElMessage.success('已删除')
        selectedIds.value = selectedIds.value.filter((x) => x !== id)
      } else {
        ElMessage.error('删除失败')
      }
    })
    .catch(() => {})
}

const handleBatchDelete = () => {
  ElMessageBox.confirm(`确定删除选中的 ${selectedIds.value.length} 个用户？`, '提示', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning'
  })
    .then(async () => {
      const ok = await userStore.batchRemoveUser(selectedIds.value)
      if (ok) {
        ElMessage.success('批量删除完成')
        selectedIds.value = []
      } else {
        ElMessage.error('批量删除失败')
      }
    })
    .catch(() => {})
}

const handleChangeRole = (user: User) => {
  selectedUser.value = user
  newRole.value = user.role
  roleDialogVisible.value = true
}

const handleRoleSubmit = async () => {
  if (!selectedUser.value) return
  roleDialogLoading.value = true
  try {
    const ok = await userStore.changeUserRole(selectedUser.value.id, newRole.value)
    if (ok) {
      ElMessage.success('角色已更新')
      roleDialogVisible.value = false
    } else {
      ElMessage.error('更新失败')
    }
  } finally {
    roleDialogLoading.value = false
  }
}

const handleDialogSubmit = async () => {
  if (!userFormRef.value) return
  try {
    await userFormRef.value.validate()
    dialogLoading.value = true
    let ok = false
    if (isEdit.value && userForm.id) {
      const payload: UserForm = {
        id: userForm.id,
        username: userForm.username,
        email: userForm.email,
        phone: userForm.phone || '',
        full_name: userForm.full_name || '',
        role: userForm.role
      }
      if (userForm.password?.trim()) {
        payload.password = userForm.password
      }
      ok = !!(await userStore.updateExistingUser(userForm.id, payload))
    } else {
      ok = !!(await userStore.addNewUser({ ...userForm }))
    }
    if (ok) {
      ElMessage.success(isEdit.value ? '已保存' : '已创建')
      dialogVisible.value = false
      resetUserForm()
    } else {
      ElMessage.error(isEdit.value ? '保存失败' : '创建失败')
    }
  } catch {
    /* validate */
  } finally {
    dialogLoading.value = false
  }
}

const resetUserForm = () => {
  Object.assign(userForm, {
    id: undefined,
    username: '',
    email: '',
    phone: '',
    full_name: '',
    password: '',
    role: 'user'
  })
  userFormRef.value?.resetFields?.()
}
</script>

<style scoped>
.user-data-card {
  border-radius: 12px;
  border: 1px solid var(--el-border-color-lighter);
}

.user-data-card :deep(.el-card__body) {
  padding: 20px 22px 18px;
}

.user-page-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.user-page-head-left {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  min-width: 0;
}

.user-page-title {
  margin: 0;
  font-size: var(--page-header-title-max);
  font-weight: 600;
  color: var(--layout-sidebar-text-strong);
}

.user-help-btn {
  padding: 0 6px;
  font-size: var(--page-header-desc-size);
}

.user-help-text {
  margin: 0;
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-regular);
}

.search-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 12px 14px;
  align-items: center;
  margin-bottom: 18px;
}

.search-input {
  flex: 1 1 220px;
  min-width: 200px;
  max-width: 420px;
}

.filter-select {
  width: 132px;
  flex-shrink: 0;
}

.search-filters-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  margin-left: auto;
}

.user-table-wrap {
  width: 100%;
  overflow-x: auto;
  border-radius: 8px;
}

.user-table-wrap :deep(.el-table .cell) {
  padding-top: 10px;
  padding-bottom: 10px;
}

.user-id-cell {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.table-footer {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  margin-top: 18px;
  padding-top: 4px;
}

.batch-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: var(--el-text-color-regular);
}

.pagination {
  margin-left: auto;
}

.role-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 10px;
  font-size: 14px;
}

.role-dialog-content p {
  margin: 0;
}

.role-select-item {
  margin-bottom: 0;
  margin-top: 4px;
}
</style>
