<template>
  <div class="login-page">
    <div class="login-page-bg" aria-hidden="true" />
    <div class="login-shell">
      <aside class="login-hero" aria-hidden="true">
        <div class="login-hero-inner">
          <div class="login-hero-mark">OP</div>
          <h2 class="login-hero-title">OpsFleetPilot</h2>
          <p class="login-hero-lead">统一运维控制台：交付、K8s、可观测与自动化技能编排。</p>
          <ul class="login-hero-points">
            <li>角色与权限隔离</li>
            <li>JWT 会话与限流保护</li>
            <li>可选自助注册（由管理员配置）</li>
          </ul>
        </div>
      </aside>

      <main class="login-main">
        <div class="login-card">
          <header class="login-card-head">
            <h1 class="login-title">欢迎回来</h1>
            <p class="login-subtitle">使用账号登录运维控制台</p>
          </header>

          <el-form
            ref="loginFormRef"
            :model="loginForm"
            :rules="loginRules"
            label-position="top"
            class="login-form"
            @submit.prevent="handleLogin"
          >
            <el-form-item label="用户名" prop="username">
              <el-input
                v-model="loginForm.username"
                placeholder="用户名"
                :prefix-icon="UserIcon"
                clearable
                autocomplete="username"
                @keyup.enter="handleLogin"
              />
            </el-form-item>

            <el-form-item label="密码" prop="password">
              <el-input
                v-model="loginForm.password"
                type="password"
                placeholder="密码"
                :prefix-icon="LockIcon"
                clearable
                show-password
                autocomplete="current-password"
                @keyup.enter="handleLogin"
              />
            </el-form-item>

            <el-form-item
              v-if="authOptions?.login_captcha_required"
              label="验证码"
              prop="captcha_answer"
            >
              <div class="captcha-row">
                <el-input
                  v-model="loginForm.captcha_answer"
                  placeholder="计算结果"
                  clearable
                  class="captcha-input"
                  @keyup.enter="handleLogin"
                />
                <el-button class="captcha-chip" @click="loadCaptcha">
                  {{ captchaChallenge || '加载验证码' }}
                </el-button>
              </div>
              <p class="captcha-hint">点击算式可刷新</p>
            </el-form-item>

            <el-alert
              v-if="loginError"
              :title="loginError"
              type="error"
              show-icon
              :closable="true"
              class="login-error-alert"
              @close="loginError = ''"
            />

            <el-form-item class="remember-item">
              <el-checkbox v-model="loginForm.remember">记住用户名</el-checkbox>
            </el-form-item>

            <el-form-item class="submit-item">
              <el-button
                type="primary"
                :loading="userStore.loading"
                class="login-btn"
                native-type="submit"
                @click="handleLogin"
              >
                登录
              </el-button>
            </el-form-item>

            <div v-if="authOptions?.public_registration_allowed" class="login-footer-row">
              <router-link class="login-link" to="/register">没有账号？立即注册</router-link>
            </div>
          </el-form>
        </div>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User as UserIcon, Lock as LockIcon } from '@element-plus/icons-vue'
import { useUserStore } from '../../stores/user'
import { getPublicAuthOptions, getLoginCaptcha } from '../../api/auth'
import type { LoginForm, PublicAuthOptions } from '../../types'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const loginFormRef = ref()

const authOptions = ref<PublicAuthOptions | null>(null)
const captchaChallenge = ref('')
const captchaId = ref('')

const loginForm = reactive<LoginForm>({
  username: '',
  password: '',
  remember: false,
  captcha_id: '',
  captcha_answer: ''
})

const loginRules = reactive({
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 50, message: '用户名长度在 3 到 50 个字符', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, max: 128, message: '密码至少 6 位', trigger: 'blur' }
  ],
  captcha_answer: [
    {
      validator: (_r: unknown, v: string, cb: (e?: Error) => void) => {
        if (!authOptions.value?.login_captcha_required) return cb()
        if (!v || !String(v).trim()) return cb(new Error('请输入验证码结果'))
        cb()
      },
      trigger: 'blur'
    }
  ]
})

const loginError = ref('')

const loadCaptcha = async () => {
  if (!authOptions.value?.login_captcha_required) return
  try {
    const data = await getLoginCaptcha()
    if (data.captcha_skipped) {
      captchaChallenge.value = ''
      captchaId.value = ''
      loginForm.captcha_id = ''
      return
    }
    captchaId.value = data.captcha_id
    captchaChallenge.value = data.challenge
    loginForm.captcha_id = data.captcha_id
    loginForm.captcha_answer = ''
  } catch {
    captchaChallenge.value = '点击重试'
  }
}

onMounted(async () => {
  const savedUsername = localStorage.getItem('rememberedUsername')
  if (savedUsername) {
    loginForm.username = savedUsername
    loginForm.remember = true
  }
  try {
    authOptions.value = await getPublicAuthOptions()
  } catch {
    authOptions.value = { public_registration_allowed: true, login_captcha_required: true }
  }
  if (authOptions.value.login_captcha_required) {
    await loadCaptcha()
  }
  if (route.query.registered === '1') {
    ElMessage.success('注册成功，请登录')
  }
  const u = route.query.u
  if (typeof u === 'string' && u && !loginForm.username) {
    loginForm.username = u
  }
})

const handleLogin = async () => {
  if (!loginFormRef.value) return
  loginError.value = ''
  if (authOptions.value?.login_captcha_required) {
    loginForm.captcha_id = captchaId.value
  }
  try {
    await loginFormRef.value.validate()
    try {
      const result = await userStore.login(loginForm)
      if (result) {
        if (loginForm.remember) {
          localStorage.setItem('rememberedUsername', loginForm.username)
        } else {
          localStorage.removeItem('rememberedUsername')
        }
        ElMessage.success('登录成功')
        router.push('/admin/dashboard')
        return
      }
      loginError.value = '用户名或密码错误，请重试'
      await loadCaptcha()
    } catch (loginErr: unknown) {
      const msg = loginErr instanceof Error ? loginErr.message : '登录失败，请检查用户名和密码'
      loginError.value = msg
      await loadCaptcha()
    }
  } catch {
    /* 表单校验失败 */
  }
}
</script>

<style scoped>
.login-page {
  position: relative;
  min-height: 100vh;
  padding: 0;
  box-sizing: border-box;
  background-color: var(--layout-page-bg);
  overflow: hidden;
}

.login-page-bg {
  pointer-events: none;
  position: absolute;
  inset: 0;
  background:
    radial-gradient(ellipse 80% 55% at 12% 10%, rgba(255, 105, 0, 0.08) 0%, transparent 56%),
    radial-gradient(ellipse 70% 50% at 92% 18%, rgba(255, 105, 0, 0.05) 0%, transparent 52%),
    radial-gradient(ellipse 55% 42% at 50% 96%, rgba(0, 0, 0, 0.04) 0%, transparent 48%);
}

.login-shell {
  position: relative;
  z-index: 1;
  display: grid;
  grid-template-columns: minmax(280px, 1fr) minmax(360px, 440px);
  min-height: 100vh;
  max-width: 1080px;
  margin: 0 auto;
  align-items: stretch;
}

@media (max-width: 880px) {
  .login-shell {
    grid-template-columns: 1fr;
    max-width: 480px;
    padding: 24px 16px 32px;
  }
  .login-hero {
    display: none;
  }
}

.login-hero {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 40px 48px 24px;
}

.login-hero-inner {
  max-width: 360px;
}

.login-hero-mark {
  width: 52px;
  height: 52px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 15px;
  font-weight: 700;
  letter-spacing: 0.04em;
  color: #fff;
  background: linear-gradient(135deg, var(--el-color-primary) 0%, var(--el-color-primary-light-3) 100%);
  box-shadow: 0 6px 20px rgba(255, 105, 0, 0.22);
  margin-bottom: 20px;
}

.login-hero-title {
  margin: 0 0 10px;
  font-size: 26px;
  font-weight: 700;
  letter-spacing: -0.03em;
  color: var(--layout-sidebar-text-strong);
  line-height: 1.2;
}

.login-hero-lead {
  margin: 0 0 20px;
  font-size: 14px;
  line-height: 1.6;
  color: var(--layout-sidebar-text);
}

.login-hero-points {
  margin: 0;
  padding-left: 18px;
  font-size: 13px;
  line-height: 1.75;
  color: var(--layout-sidebar-text);
}

.login-main {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px 24px 40px;
}

.login-card {
  width: 100%;
  max-width: 420px;
  padding: 32px 32px 28px;
  background: var(--layout-content-surface);
  border: 1px solid var(--layout-sidebar-border);
  border-radius: 16px;
  box-shadow:
    var(--layout-shadow-soft),
    0 24px 48px rgba(0, 0, 0, 0.06);
}

.login-card-head {
  margin-bottom: 22px;
}

.login-title {
  margin: 0 0 6px;
  font-size: 20px;
  font-weight: 600;
  letter-spacing: -0.02em;
  color: var(--layout-sidebar-text-strong);
}

.login-subtitle {
  margin: 0;
  font-size: 13px;
  color: var(--layout-sidebar-text);
}

.login-form {
  width: 100%;
}

.login-form :deep(.el-form-item__label) {
  color: var(--layout-sidebar-text-strong);
  font-weight: 500;
}

.captcha-row {
  display: flex;
  gap: 10px;
  width: 100%;
}

.captcha-input {
  flex: 1;
  min-width: 0;
}

.captcha-chip {
  flex-shrink: 0;
  min-width: 120px;
  font-variant-numeric: tabular-nums;
}

.captcha-hint {
  margin: 6px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.remember-item {
  margin-bottom: 14px;
}

.login-error-alert {
  margin-bottom: 14px;
}

.submit-item {
  margin-bottom: 0;
}

.login-btn {
  width: 100%;
  height: 44px;
  font-size: 15px;
  font-weight: 600;
  border-radius: 10px;
}

.login-footer-row {
  text-align: center;
  margin-top: 4px;
}

.login-link {
  font-size: 13px;
  color: var(--el-color-primary);
  text-decoration: none;
}

.login-link:hover {
  text-decoration: underline;
}
</style>
