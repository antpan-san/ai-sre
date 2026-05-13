import request from '../utils/request'
import type { User, UserListParams, UserListResponse, UserForm } from '../types'

export const getUserList = (params: UserListParams): Promise<UserListResponse> => {
  return request.get('/api/user', { params })
}

export const getUserDetail = (id: string): Promise<User> => {
  return request.get(`/api/user/${id}`)
}

export const addUser = (data: UserForm): Promise<User> => {
  return request.post('/api/user', data)
}

export const updateUser = (id: string, data: UserForm): Promise<User> => {
  return request.put(`/api/user/${id}`, data)
}

export const deleteUser = (id: string): Promise<void> => {
  return request.delete(`/api/user/${id}`)
}

export const batchDeleteUser = (ids: string[]): Promise<void> => {
  return request.delete('/api/user/batch', { data: { ids } })
}

export const updateUserRole = (id: string, role: string): Promise<User> => {
  return request.patch(`/api/user/${id}/role`, { role })
}
