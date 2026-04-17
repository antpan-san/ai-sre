import axios, { type AxiosInstance, type InternalAxiosRequestConfig, type AxiosResponse } from 'axios'
import { ElMessage } from 'element-plus'
import NProgress from 'nprogress'
import type { ApiResponse } from '../types'

// 请求计数器，用于处理并发请求
let requestCount = 0

// Trae Library - 自定义API请求库
class Trae {
  private instance: AxiosInstance

  constructor() {
    // 创建axios实例
    this.instance = axios.create({
      baseURL: import.meta.env.VITE_BASE_API || '/ft-api',
      timeout: 10000
    })

    // 配置请求拦截器
    this.setupRequestInterceptor()
    
    // 配置响应拦截器
    this.setupResponseInterceptor()
  }

  // 请求拦截器
  private setupRequestInterceptor() {
    this.instance.interceptors.request.use(
      (config: InternalAxiosRequestConfig) => {
        // 增加请求计数器
        requestCount++
        // 启动进度条
        NProgress.start()
        
        // 从localStorage获取token
        const token = localStorage.getItem('token')
        if (token && config.headers) {
          config.headers.Authorization = `Bearer ${token}`
        }
        return config
      },
      (error) => {
        // 请求失败时减少计数器
        requestCount--
        if (requestCount <= 0) {
          NProgress.done()
        }
        return Promise.reject(error)
      }
    )
  }

  // 响应拦截器
  private setupResponseInterceptor() {
    this.instance.interceptors.response.use(
      (response: AxiosResponse<ApiResponse>) => {
        const res = response.data
        // 减少请求计数器
        requestCount--
        // 如果所有请求都完成，停止进度条
        if (requestCount <= 0) {
          NProgress.done()
        }
        
        // 根据code判断请求是否成功
        if (res.code !== 200) {
          const msg = (res as any).msg || (res as any).message || '请求失败'
          ElMessage.error(msg)
          // 处理401未授权（登录接口的 401 不跳转，由登录页自己处理）
          if (res.code === 401) {
            const isLoginRequest = (response.config?.url ?? '').includes('/auth/login')
            if (!isLoginRequest) {
              localStorage.removeItem('token')
              localStorage.removeItem('userInfo')
              window.location.href = '/login'
            }
          }
          return Promise.reject(new Error(msg))
        } else {
          // 直接返回业务数据
          return res.data
        }
      },
      (error) => {
        // 减少请求计数器
        requestCount--
        // 如果所有请求都完成，停止进度条
        if (requestCount <= 0) {
          NProgress.done()
        }
        
        let errorMsg = '网络错误'
        if (error.response) {
          const isLoginRequest = (error.config?.url ?? '').includes('/auth/login')
          switch (error.response.status) {
            case 401:
              errorMsg = (error.response.data as any)?.message ?? (error.response.data as any)?.msg ?? '用户名或密码错误'
              if (!isLoginRequest) {
                localStorage.removeItem('token')
                localStorage.removeItem('userInfo')
                window.location.href = '/login'
              }
              break
            case 403:
              errorMsg = '拒绝访问'
              break
            case 404:
              errorMsg = '请求地址不存在'
              break
            case 500:
              errorMsg = '服务器内部错误'
              break
            default:
              errorMsg = error.response.data?.msg || '请求失败'
          }
        } else if (error.request) {
          errorMsg = '网络连接失败，请检查网络设置'
        } else {
          errorMsg = error.message || '请求失败'
        }
        ElMessage.error(errorMsg)
        return Promise.reject(error)
      }
    )
  }

  // GET请求 - config 支持 { params, responseType } 等 axios 配置
  get<T = any>(url: string, config?: Record<string, any>): Promise<T> {
    return this.instance.get(url, config)
  }

  // POST请求
  post<T = any>(url: string, data?: any, config?: Record<string, any>): Promise<T> {
    return this.instance.post(url, data, config)
  }

  // PUT请求
  put<T = any>(url: string, data?: any, config?: Record<string, any>): Promise<T> {
    return this.instance.put(url, data, config)
  }

  // DELETE请求
  delete<T = any>(url: string, config?: Record<string, any>): Promise<T> {
    return this.instance.delete(url, config)
  }

  // PATCH请求
  patch<T = any>(url: string, data?: any, config?: Record<string, any>): Promise<T> {
    return this.instance.patch(url, data, config)
  }

  // HEAD请求
  head<T = any>(url: string, config?: Record<string, any>): Promise<T> {
    return this.instance.head(url, config)
  }

  // OPTIONS请求
  options<T = any>(url: string, config?: Record<string, any>): Promise<T> {
    return this.instance.options(url, config)
  }
}

// 创建Trae实例
const trae = new Trae()

export default trae
