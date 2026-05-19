import axios, { type AxiosInstance, type InternalAxiosRequestConfig, type AxiosResponse } from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import NProgress from 'nprogress'
import type { ApiResponse } from '../types'

/** POST /api/auth/login（排除 login-captcha，避免 URL 误判） */
function isAuthLoginPostURL(url: string | undefined): boolean {
  if (!url) return false
  return url.includes('/auth/login') && !url.includes('login-captcha')
}

// 请求计数器，用于处理并发请求
let requestCount = 0
let paywallPromptVisible = false

function isPaywall(data: unknown): boolean {
  return String((data as any)?.biz || '').startsWith('PAYWALL_')
}

async function promptSubscription(baseURL: string, data: any): Promise<void> {
  if (paywallPromptVisible) return
  paywallPromptVisible = true
  const packKey = String(data?.pack_key || '').trim()
  const title = packKey ? `订阅 ${packKey}` : '功能订阅'
  const message = data?.msg || (packKey ? `当前功能需要开通 ${packKey} 后使用。是否前往 Stripe 收银台？` : '当前功能需要订阅后使用。是否前往 Stripe 收银台？')
  try {
    await ElMessageBox.confirm(message, title, {
      confirmButtonText: '去订阅',
      cancelButtonText: '取消',
      type: 'warning'
    })
    const token = localStorage.getItem('token')
    const resp = await axios.post(
      `${baseURL.replace(/\/$/, '')}/api/billing/checkout-session`,
      packKey ? { pack_key: packKey } : {},
      { headers: token ? { Authorization: `Bearer ${token}` } : {} }
    )
    const url = resp.data?.data?.url
    if (url) {
      window.location.href = url
    } else {
      ElMessage.info(resp.data?.data?.message || '当前账号无需订阅')
    }
  } catch {
    // user canceled or checkout failed; interceptor/default axios message handles network failures elsewhere
  } finally {
    paywallPromptVisible = false
  }
}

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
        
        // 根据 code 判断请求是否成功（201 用于资源创建，如手动创建自动迭代任务）
        if (res.code !== 200 && res.code !== 201) {
          const msg = (res as any).msg || (res as any).message || '请求失败'
          if (!isAuthLoginPostURL(response.config?.url)) {
            ElMessage.error(msg)
          }
          // 处理401未授权（登录接口的 401 不跳转，由登录页自己处理）
          if (res.code === 401) {
            const isLogin = isAuthLoginPostURL(response.config?.url)
            if (!isLogin) {
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
          const isLoginPost = isAuthLoginPostURL(error.config?.url)
          switch (error.response.status) {
            case 401:
              errorMsg = (error.response.data as any)?.message ?? (error.response.data as any)?.msg ?? '用户名或密码错误'
              if (!isLoginPost) {
                localStorage.removeItem('token')
                localStorage.removeItem('userInfo')
                window.location.href = '/login'
              }
              break
            case 403:
              errorMsg = (error.response.data as any)?.msg ?? '拒绝访问'
              if (isPaywall(error.response.data)) {
                void promptSubscription(this.instance.defaults.baseURL || '/ft-api', error.response.data)
              }
              break
            case 404:
              errorMsg = '请求地址不存在'
              break
            case 429:
              errorMsg = (error.response.data as any)?.msg ?? '请求过于频繁，请稍后再试'
              break
            case 500:
              errorMsg = '服务器内部错误'
              break
            default:
              errorMsg = (error.response.data as any)?.msg || '请求失败'
          }
          if (!isLoginPost) {
            ElMessage.error(errorMsg)
          }
        } else if (error.request) {
          errorMsg = '网络连接失败，请检查网络设置'
          ElMessage.error(errorMsg)
        } else {
          errorMsg = error.message || '请求失败'
          ElMessage.error(errorMsg)
        }
        return Promise.reject(new Error(errorMsg))
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
