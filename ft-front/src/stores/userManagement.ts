import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'
import { getUserList, addUser, updateUser, deleteUser, batchDeleteUser, updateUserRole } from '../api/user'
import type { User, UserListParams, UserListResponse, UserForm } from '../types'

export const useUserManagementStore = defineStore('userManagement', () => {
  // 状态
  const userList = ref<User[]>([])
  const total = ref<number>(0)
  const loading = ref<boolean>(false)
  const filters = reactive<UserListParams>({
    page: 1,
    pageSize: 10,
    username: '',
    role: ''
  })

  // 获取用户列表
  const fetchUserList = async (params?: Partial<UserListParams>) => {
    loading.value = true
    try {
      const queryParams = { ...filters, ...params }
      const res: UserListResponse = await getUserList(queryParams)
      userList.value = res.list
      total.value = res.total
      return res
    } catch (error) {
      return null
    } finally {
      loading.value = false
    }
  }

  // 添加用户
  const addNewUser = async (data: UserForm) => {
    try {
      const res = await addUser(data)
      await fetchUserList()
      return res
    } catch (error) {
      return null
    }
  }

  // 更新用户
  const updateExistingUser = async (id: number, data: UserForm) => {
    try {
      const res = await updateUser(id, data)
      await fetchUserList()
      return res
    } catch (error) {
      return null
    }
  }

  // 删除用户
  const removeUser = async (id: number) => {
    try {
      await deleteUser(id)
      await fetchUserList()
      return true
    } catch (error) {
      return false
    }
  }

  // 批量删除用户
  const batchRemoveUser = async (ids: number[]) => {
    try {
      await batchDeleteUser(ids)
      await fetchUserList()
      return true
    } catch (error) {
      return false
    }
  }

  // 更新用户角色
  const changeUserRole = async (id: number, role: string) => {
    try {
      const res = await updateUserRole(id, role)
      await fetchUserList()
      return res
    } catch (error) {
      return null
    }
  }

  // 设置过滤器
  const setFilters = (newFilters: Partial<UserListParams>) => {
    Object.assign(filters, newFilters)
    if (newFilters.page) {
      filters.page = newFilters.page
    }
  }

  // 重置过滤器
  const resetFilters = () => {
    filters.page = 1
    filters.pageSize = 10
    filters.username = ''
    filters.role = ''
  }

  return {
    userList,
    total,
    loading,
    filters,
    fetchUserList,
    addNewUser,
    updateExistingUser,
    removeUser,
    batchRemoveUser,
    changeUserRole,
    setFilters,
    resetFilters
  }
})