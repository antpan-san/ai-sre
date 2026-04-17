import { defineStore } from 'pinia'
import { ref } from 'vue'
import { login as apiLogin, logout as apiLogout, getUserInfo as apiGetUserInfo } from '../api/auth'
import type { LoginForm, User } from '../types'

export const useUserStore = defineStore('user', () => {
  // 状态
  const currentUser = ref<User | null>(null)
  const loading = ref<boolean>(false)
  const token = ref<string>(localStorage.getItem('token') || '')

  // 登录
  const login = async (data: LoginForm) => {
    loading.value = true
    try {
      const res = await apiLogin(data) as any
      // 响应拦截器已解包，res 直接是 { token, user }
      if (res && res.token) {
        token.value = res.token
        currentUser.value = res.user
        localStorage.setItem('token', res.token)
        localStorage.setItem('userInfo', JSON.stringify(res.user))
        return res
      }
      return null
    } catch (error) {
      console.error('登录失败:', error)
      return null
    } finally {
      loading.value = false
    }
  }

  // 登出
  const logout = async () => {
    try {
      await apiLogout()
    } catch (error) {
      // 即使接口失败也清除本地状态
      console.error('登出接口调用失败:', error)
    } finally {
      token.value = ''
      currentUser.value = null
      localStorage.removeItem('token')
      localStorage.removeItem('userInfo')
    }
  }

  // 获取用户信息
  const fetchUserInfo = async () => {
    loading.value = true
    try {
      const res = await apiGetUserInfo()
      currentUser.value = res as unknown as User
      localStorage.setItem('userInfo', JSON.stringify(currentUser.value))
      return currentUser.value
    } catch (error) {
      console.error('获取用户信息失败:', error)
      return null
    } finally {
      loading.value = false
    }
  }

  // 从 localStorage 恢复用户信息
  const restoreUser = () => {
    const userStr = localStorage.getItem('userInfo')
    if (userStr) {
      try {
        currentUser.value = JSON.parse(userStr)
      } catch {
        currentUser.value = null
      }
    }
  }

  // 初始化时恢复
  restoreUser()

  return {
    currentUser,
    loading,
    token,
    login,
    logout,
    fetchUserInfo,
    restoreUser
  }
})
