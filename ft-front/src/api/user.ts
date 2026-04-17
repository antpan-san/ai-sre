import request from '../utils/request'
import type { User, UserListParams, UserListResponse, UserForm } from '../types'

// 获取用户列表
export const getUserList = (params: UserListParams): Promise<UserListResponse> => {
  return request.get('/api/user', { params })
}

// 获取用户详情
export const getUserDetail = (id: number): Promise<User> => {
  return request.get(`/api/user/${id}`)
}

// 添加用户
export const addUser = (data: UserForm): Promise<User> => {
  return request.post('/api/user', data)
}

// 更新用户
export const updateUser = (id: number, data: UserForm): Promise<User> => {
  return request.put(`/api/user/${id}`, data)
}

// 删除用户
export const deleteUser = (id: number): Promise<void> => {
  return request.delete(`/api/user/${id}`)
}

// 批量删除用户
export const batchDeleteUser = (ids: number[]): Promise<void> => {
  return request.delete('/api/user/batch', { data: { ids } })
}

// 更新用户角色
export const updateUserRole = (id: number, role: string): Promise<User> => {
  return request.patch(`/api/user/${id}/role`, { role })
}
