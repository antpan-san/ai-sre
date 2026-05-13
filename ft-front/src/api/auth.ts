import request from '../utils/request'
import type { LoginForm, LoginResponse, PublicAuthOptions, RegisterForm, User } from '../types'

export const getPublicAuthOptions = (): Promise<PublicAuthOptions> => {
  return request.get('/api/auth/public-options')
}

export const getLoginCaptcha = (): Promise<{
  captcha_id: string
  challenge: string
  captcha_skipped?: boolean
}> => {
  return request.get('/api/auth/login-captcha')
}

export const register = (data: RegisterForm): Promise<{ user: User }> => {
  return request.post('/api/auth/register', data)
}

export const login = (data: LoginForm): Promise<LoginResponse> => {
  return request.post('/api/auth/login', {
    username: data.username,
    password: data.password,
    captcha_id: data.captcha_id,
    captcha_answer: data.captcha_answer
  })
}

export const logout = (): Promise<void> => {
  return request.post('/api/auth/logout')
}

export const getUserInfo = (): Promise<User> => {
  return request.get('/api/auth/info')
}
