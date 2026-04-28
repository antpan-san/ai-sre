<template>
  <div class="permission-management">
    <div class="page-header">
      <h2>权限管理</h2>
    </div>
    
    <!-- 搜索和筛选区域 -->
    <div class="search-filters">
      <el-input
        v-model="filters.name"
        placeholder="搜索权限名称"
        :prefix-icon="Search"
        clearable
        @clear="handleSearch"
        @keyup.enter="handleSearch"
        class="search-input"
      />
      
      <el-input
        v-model="filters.code"
        placeholder="搜索权限代码"
        :prefix-icon="Search"
        clearable
        @clear="handleSearch"
        @keyup.enter="handleSearch"
        class="search-input"
      />
      
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
        新增权限
      </el-button>
    </div>
    
    <!-- 权限列表表格 -->
    <div class="permission-table">
      <el-table
        v-loading="loading"
        :data="permissionsList"
        stripe
        border
        @selection-change="handleSelectionChange"
        style="width: 100%"
        row-key="id"
      >
        <el-table-column type="selection" min-width="40" />
        
        <el-table-column prop="id" label="ID" min-width="60" align="center" />
        
        <el-table-column prop="name" label="权限名称" min-width="150" align="left" />
        
        <el-table-column prop="code" label="权限代码" min-width="200" align="center">
          <template #default="scope">
            <el-tag size="small" type="info">{{ scope.row.code }}</el-tag>
          </template>
        </el-table-column>
        
        <el-table-column prop="description" label="描述" min-width="250" align="left">
          <template #default="scope">
            <el-tooltip effect="dark" :content="scope.row.description" placement="top">
              <span class="description" style="display: inline-block; width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">
                {{ scope.row.description || '无描述' }}
              </span>
            </el-tooltip>
          </template>
        </el-table-column>
        
        <el-table-column prop="createTime" label="创建时间" min-width="180" align="center" />
        
        <el-table-column prop="updateTime" label="更新时间" min-width="180" align="center" />
        
        <el-table-column label="操作" min-width="120" align="center">
          <template #default="scope">
            <el-button
              size="small"
              type="primary"
              @click="handleEdit(scope.row)"
              :icon="Edit"
              title="编辑"
            >
            </el-button>
            
            <el-button
              type="danger"
              size="small"
              @click="handleDelete(scope.row.id)"
              :icon="Delete"
              title="删除"
            >
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
    
    <!-- 批量操作 -->
    <div class="batch-actions" v-if="selectedIds.length > 0">
      <el-button type="danger" @click="handleBatchDelete">
        <el-icon><Delete /></el-icon>
        批量删除 ({{ selectedIds.length }})
      </el-button>
    </div>
    
    <!-- 分页 -->
    <div class="pagination">
      <el-pagination
        v-model:current-page="filters.page"
        v-model:page-size="filters.pageSize"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        :total="total"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>
    
    <!-- 添加/编辑权限对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑权限' : '新增权限'"
      width="500px"
    >
      <el-form
        ref="permissionFormRef"
        :model="permissionForm"
        :rules="permissionRules"
        label-position="top"
      >
        <el-form-item label="权限名称" prop="name">
          <el-input
            v-model="permissionForm.name"
            placeholder="请输入权限名称"
            clearable
          />
        </el-form-item>
        
        <el-form-item label="权限代码" prop="code">
          <el-input
            v-model="permissionForm.code"
            placeholder="请输入权限代码（如：user:create）"
            clearable
          />
        </el-form-item>
        
        <el-form-item label="描述" prop="description">
          <el-input
            v-model="permissionForm.description"
            placeholder="请输入权限描述"
            type="textarea"
            :rows="3"
            clearable
          />
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
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, RefreshRight, Plus, Delete, Edit } from '@element-plus/icons-vue'
import { getPermissions, addPermission, updatePermission, deletePermission, batchDeletePermissions } from '../../api/security-audit'
import type { Permission, PermissionListParams, PermissionListResponse } from '../../types'

// 响应式状态
const loading = ref(false)
const dialogVisible = ref(false)
const dialogLoading = ref(false)
const permissionFormRef = ref()
const selectedIds = ref<number[]>([])
const isEdit = ref(false)

// 筛选条件
const filters = reactive<PermissionListParams>({
  page: 1,
  pageSize: 20,
  name: '',
  code: ''
})

// 权限列表数据
const permissionsList = ref<Permission[]>([])
const total = ref(0)

// 权限表单
const permissionForm = reactive<Permission>({
  id: 0,
  name: '',
  code: '',
  description: '',
  createTime: '',
  updateTime: ''
})

// 权限表单验证规则
const permissionRules = reactive({
  name: [
    { required: true, message: '请输入权限名称', trigger: 'blur' },
    { min: 2, max: 50, message: '权限名称长度在 2 到 50 个字符', trigger: 'blur' }
  ],
  code: [
    { required: true, message: '请输入权限代码', trigger: 'blur' },
    { min: 2, max: 100, message: '权限代码长度在 2 到 100 个字符', trigger: 'blur' },
    { pattern: /^[a-zA-Z0-9_:]+$/, message: '权限代码只能包含字母、数字、下划线和冒号', trigger: 'blur' }
  ]
})

// 加载权限列表
onMounted(() => {
  fetchPermissions()
})

// 获取权限列表
const fetchPermissions = async () => {
  loading.value = true
  try {
    const response: PermissionListResponse = await getPermissions(filters)
    permissionsList.value = response.list
    total.value = response.total
  } catch (error) {
    console.error('获取权限列表失败:', error)
    ElMessage.error('获取权限列表失败')
  } finally {
    loading.value = false
  }
}

// 处理搜索
const handleSearch = () => {
  filters.page = 1
  fetchPermissions()
}

// 处理重置
const handleReset = () => {
  Object.assign(filters, {
    page: 1,
    pageSize: 20,
    name: '',
    code: ''
  })
  fetchPermissions()
}

// 处理分页大小变化
const handleSizeChange = (size: number) => {
  filters.pageSize = size
  filters.page = 1
  fetchPermissions()
}

// 处理当前页变化
const handleCurrentChange = (current: number) => {
  filters.page = current
  fetchPermissions()
}

// 处理选择变化
const handleSelectionChange = (selection: Permission[]) => {
  selectedIds.value = selection.map(item => item.id)
}

// 处理添加权限
const handleAdd = () => {
  isEdit.value = false
  resetPermissionForm()
  dialogVisible.value = true
}

// 处理编辑权限
const handleEdit = (permission: Permission) => {
  isEdit.value = true
  Object.assign(permissionForm, permission)
  dialogVisible.value = true
}

// 处理删除权限
const handleDelete = (id: number) => {
  ElMessageBox.confirm('确定要删除该权限吗？', '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await deletePermission(id)
      ElMessage.success('删除成功')
      selectedIds.value = selectedIds.value.filter(item => item !== id)
      fetchPermissions()
    } catch (error) {
      console.error('删除权限失败:', error)
      ElMessage.error('删除权限失败')
    }
  }).catch(() => {
    // 取消删除
  })
}

// 处理批量删除
const handleBatchDelete = () => {
  ElMessageBox.confirm(`确定要删除选中的 ${selectedIds.value.length} 个权限吗？`, '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await batchDeletePermissions(selectedIds.value)
      ElMessage.success('批量删除成功')
      selectedIds.value = []
      fetchPermissions()
    } catch (error) {
      console.error('批量删除权限失败:', error)
      ElMessage.error('批量删除权限失败')
    }
  }).catch(() => {
    // 取消删除
  })
}

// 处理对话框提交
const handleDialogSubmit = async () => {
  if (!permissionFormRef.value) return
  
  try {
    await permissionFormRef.value.validate()
    dialogLoading.value = true
    
    if (isEdit.value) {
      // 更新权限
      await updatePermission(permissionForm.id, permissionForm)
      ElMessage.success('更新成功')
    } else {
      // 添加权限
      await addPermission(permissionForm)
      ElMessage.success('添加成功')
    }
    
    dialogVisible.value = false
    resetPermissionForm()
    fetchPermissions()
  } catch (error) {
    console.error('表单验证失败:', error)
  } finally {
    dialogLoading.value = false
  }
}

// 重置权限表单
const resetPermissionForm = () => {
  Object.assign(permissionForm, {
    id: 0,
    name: '',
    code: '',
    description: '',
    createTime: '',
    updateTime: ''
  })
  
  if (permissionFormRef.value) {
    permissionFormRef.value.resetFields()
  }
}
</script>

<style scoped>
.permission-management {
  padding: 0 20px 20px 20px;
}

.page-header h2 {
  margin: 0 0 20px 0;
  color: var(--el-color-primary);
  font-size: 30px;
  font-weight: 600;
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

.permission-table {
  margin-bottom: 20px;
  width: 100%;
}

.batch-actions {
  margin-top: 20px;
  padding: 15px;
  background-color: #f5f7fa;
  border-radius: 4px;
}

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
</style>