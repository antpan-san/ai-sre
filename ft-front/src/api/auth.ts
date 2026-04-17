import request from '../utils/request'
import type { LoginForm, LoginResponse, User } from '../types'

// 用户登录
export const login = (data: LoginForm): Promise<LoginResponse> => {
  return request.post('/api/auth/login', data)
}

// 用户登出
export const logout = (): Promise<void> => {
  return request.post('/api/auth/logout')
}

// 获取用户信息
export const getUserInfo = (): Promise<User> => {
  return request.get('/api/auth/info')
}
