<template>
  <div class="user-list">
    <div class="page-header">
      <h2>用户管理</h2>
    </div>
    
    <!-- 搜索和筛选区域 -->
    <div class="search-filters">
      <el-input
        v-model="userStore.filters.username"
        placeholder="搜索用户名"
        :prefix-icon="Search"
        clearable
        @keyup.enter="handleSearch"
        class="search-input"
      />
      
      <el-select
        v-model="userStore.filters.role"
        placeholder="选择角色"
        clearable
        @change="handleSearch"
        class="filter-select"
      >
        <el-option label="管理员" value="admin" />
        <el-option label="普通用户" value="user" />
      </el-select>
      
      <el-button type="primary" @click="handleSearch">
        <el-icon><Search /></el-icon>
        搜索
      </el-button>
      
      <el-button @click="handleReset">
        <el-icon><RefreshRight /></el-icon>
        重置
      </el-button>
      
      <el-button type="success" @click="handleAdd">
        <el-icon><Plus /></el-icon>
        新增用户
      </el-button>
    </div>
    
    <!-- 用户列表表格 -->
    <div class="user-table">
      <el-table
        v-loading="userStore.loading"
        :data="userStore.userList"
        stripe
        border
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" />
        
        <el-table-column prop="id" label="ID" width="80" align="center" />
        
        <el-table-column prop="username" label="用户名" min-width="150" />
        
        <el-table-column prop="phone" label="手机号" min-width="150" />
        
        <el-table-column prop="role" label="角色" width="120" align="center">
          <template #default="scope">
            <el-tag
              :type="scope.row.role === 'admin' ? 'success' : 'info'"
              size="small"
            >
              {{ scope.row.role === 'admin' ? '管理员' : '普通用户' }}
            </el-tag>
          </template>
        </el-table-column>
        
        <el-table-column prop="createTime" label="创建时间" width="180" align="center" />
        
        <el-table-column label="操作" width="250" align="center">
          <template #default="scope">
            <el-button
              type="primary"
              size="small"
              @click="handleEdit(scope.row)"
              :icon="Edit"
            >
              编辑
            </el-button>
            
            <el-button
              type="warning"
              size="small"
              @click="handleChangeRole(scope.row)"
              :icon="SwitchButton"
            >
              角色设置
            </el-button>
            
            <el-button
              type="danger"
              size="small"
              @click="handleDelete(scope.row.id)"
              :icon="Delete"
              :disabled="scope.row.id === 1"
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
        v-model:current-page="userStore.filters.page"
        v-model:page-size="userStore.filters.pageSize"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        :total="userStore.total"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>
    
    <!-- 批量操作 -->
    <div class="batch-actions" v-if="selectedIds.length > 0">
      <el-button type="danger" @click="handleBatchDelete">
        <el-icon><Delete /></el-icon>
        批量删除 ({{ selectedIds.length }})
      </el-button>
    </div>
    
    <!-- 添加/编辑用户对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑用户' : '新增用户'"
      width="500px"
    >
      <el-form
        ref="userFormRef"
        :model="userForm"
        :rules="userRules"
        label-position="top"
      >
        <el-form-item label="用户名" prop="username">
          <el-input
            v-model="userForm.username"
            placeholder="请输入用户名"
            clearable
          />
        </el-form-item>
        
        <el-form-item label="手机号" prop="phone">
          <el-input
            v-model="userForm.phone"
            placeholder="请输入手机号"
            clearable
          />
        </el-form-item>
        
        <el-form-item v-if="!isEdit" label="密码" prop="password">
          <el-input
            v-model="userForm.password"
            type="password"
            placeholder="请输入密码"
            show-password
          />
        </el-form-item>
        
        <el-form-item label="角色" prop="role">
          <el-select
            v-model="userForm.role"
            placeholder="请选择角色"
          >
            <el-option label="管理员" value="admin" />
            <el-option label="普通用户" value="user" />
          </el-select>
        </el-form-item>
      </el-form>
      
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button
            type="primary"
            :loading="dialogLoading"
            @click="handleDialogSubmit"
          >
            确认
          </el-button>
        </div>
      </template>
    </el-dialog>
    
    <!-- 角色设置对话框 -->
    <el-dialog
      v-model="roleDialogVisible"
      title="设置角色"
      width="400px"
    >
      <div class="role-dialog-content">
        <p>当前用户: {{ selectedUser?.username }}</p>
        <p>当前角色: <el-tag :type="selectedUser?.role === 'admin' ? 'success' : 'info'">
          {{ selectedUser?.role === 'admin' ? '管理员' : '普通用户' }}
        </el-tag></p>
        <el-form-item label="新角色">
          <el-select
            v-model="newRole"
            placeholder="请选择新角色"
            style="width: 100%"
          >
            <el-option label="管理员" value="admin" />
            <el-option label="普通用户" value="user" />
          </el-select>
        </el-form-item>
      </div>
      
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="roleDialogVisible = false">取消</el-button>
          <el-button
            type="primary"
            :loading="roleDialogLoading"
            @click="handleRoleSubmit"
          >
            确认修改
          </el-button>
        </div>
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
const selectedIds = ref<number[]>([])
const isEdit = ref(false)
const selectedUser = ref<User | null>(null)
const newRole = ref<string>('')

// 用户表单
const userForm = reactive<UserForm>({
  id: undefined,
  username: '',
  phone: '',
  password: '',
  role: 'user'
})

// 用户表单验证规则
const userRules = reactive({
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 20, message: '用户名长度在 3 到 20 个字符', trigger: 'blur' }
  ],
  phone: [
    { required: true, message: '请输入手机号', trigger: 'blur' },
    { pattern: /^1[3-9]\d{9}$/, message: '请输入正确的手机号格式', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, max: 20, message: '密码长度在 6 到 20 个字符', trigger: 'blur' }
  ],
  role: [
    { required: true, message: '请选择用户角色', trigger: 'change' }
  ]
})

// 加载用户列表
onMounted(() => {
  fetchUserList()
})

// 获取用户列表
const fetchUserList = () => {
  userStore.fetchUserList()
}

// 处理搜索
const handleSearch = () => {
  userStore.filters.page = 1
  fetchUserList()
}

// 处理重置
const handleReset = () => {
  userStore.resetFilters()
  fetchUserList()
}

// 处理分页大小变化
const handleSizeChange = (size: number) => {
  userStore.filters.pageSize = size
  fetchUserList()
}

// 处理当前页变化
const handleCurrentChange = (current: number) => {
  userStore.filters.page = current
  fetchUserList()
}

// 处理选择变化
const handleSelectionChange = (selection: User[]) => {
  selectedIds.value = selection.map(item => item.id)
}

// 处理添加用户
const handleAdd = () => {
  isEdit.value = false
  resetUserForm()
  dialogVisible.value = true
}

// 处理编辑用户
const handleEdit = (user: User) => {
  isEdit.value = true
  // 复制用户数据到表单
  Object.assign(userForm, user)
  dialogVisible.value = true
}

// 处理删除用户
const handleDelete = (id: number) => {
  ElMessageBox.confirm('确定要删除该用户吗？', '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    const success = await userStore.removeUser(id)
    if (success) {
      ElMessage.success('删除成功')
      selectedIds.value = selectedIds.value.filter(item => item !== id)
    } else {
      ElMessage.error('删除失败')
    }
  }).catch(() => {
    // 取消删除
  })
}

// 处理批量删除
const handleBatchDelete = () => {
  ElMessageBox.confirm(`确定要删除选中的 ${selectedIds.value.length} 个用户吗？`, '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    const success = await userStore.batchRemoveUser(selectedIds.value)
    if (success) {
      ElMessage.success('批量删除成功')
      selectedIds.value = []
    } else {
      ElMessage.error('批量删除失败')
    }
  }).catch(() => {
    // 取消删除
  })
}

// 处理角色设置
const handleChangeRole = (user: User) => {
  selectedUser.value = user
  newRole.value = user.role
  roleDialogVisible.value = true
}

// 处理角色提交
const handleRoleSubmit = async () => {
  if (!selectedUser.value) return
  
  try {
    roleDialogLoading.value = true
    const success = await userStore.changeUserRole(selectedUser.value.id, newRole.value)
    if (success) {
      ElMessage.success('角色修改成功')
      roleDialogVisible.value = false
    } else {
      ElMessage.error('角色修改失败')
    }
  } catch (error) {
    ElMessage.error('角色修改失败')
  } finally {
    roleDialogLoading.value = false
  }
}

// 处理对话框提交
const handleDialogSubmit = async () => {
  if (!userFormRef.value) return
  
  try {
    await userFormRef.value.validate()
    dialogLoading.value = true
    
    let result
    if (isEdit.value) {
      // 更新用户
      if (userForm.id) {
        result = await userStore.updateExistingUser(userForm.id, userForm)
      }
    } else {
      // 添加用户
      result = await userStore.addNewUser(userForm)
    }
    
    if (result) {
      ElMessage.success(isEdit.value ? '更新成功' : '添加成功')
      dialogVisible.value = false
      resetUserForm()
    } else {
      ElMessage.error(isEdit.value ? '更新失败' : '添加失败')
    }
  } catch (error) {
    console.error('表单验证失败:', error)
  } finally {
    dialogLoading.value = false
  }
}

// 重置用户表单
const resetUserForm = () => {
  Object.assign(userForm, {
    id: undefined,
    username: '',
    phone: '',
    password: '',
    role: 'user'
  })
  
  if (userFormRef.value) {
    userFormRef.value.resetFields()
  }
}
</script>

<style scoped>
.user-list {
  padding: 20px;
}

.page-header h2 {
  margin: 0 0 20px 0;
  color: #303133;
}

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

.user-table {
  margin-bottom: 20px;
}

.pagination {
  text-align: right;
  margin-bottom: 20px;
}

.batch-actions {
  margin-top: 20px;
  padding: 15px;
  background-color: #f5f7fa;
  border-radius: 4px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.role-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.role-dialog-content p {
  margin: 0;
  font-size: 14px;
}
</style>
