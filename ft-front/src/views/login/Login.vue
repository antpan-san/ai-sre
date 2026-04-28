<template>
  <div class="login-page">
    <div class="login-page-bg" aria-hidden="true" />
    <div class="login-card">
      <div class="login-brand">
        <div class="login-brand-mark">OP</div>
        <div class="login-brand-text">
          <h1 class="login-title">OpsFleetPilot</h1>
          <p class="login-subtitle">运维控制台</p>
        </div>
      </div>

      <el-form
        ref="loginFormRef"
        :model="loginForm"
        :rules="loginRules"
        label-position="top"
        class="login-form"
      >
        <el-form-item label="用户名" prop="username">
          <el-input
            v-model="loginForm.username"
            placeholder="请输入用户名"
            :prefix-icon="UserIcon"
            clearable
            @keyup.enter="handleLogin"
          />
        </el-form-item>

        <el-form-item label="密码" prop="password">
          <el-input
            v-model="loginForm.password"
            type="password"
            placeholder="请输入密码"
            :prefix-icon="LockIcon"
            clearable
            show-password
            @keyup.enter="handleLogin"
          />
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
            @click="handleLogin"
          >
            登录
          </el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User as UserIcon, Lock as LockIcon } from '@element-plus/icons-vue'
import { useUserStore } from '../../stores/user'
import type { LoginForm } from '../../types'

const router = useRouter()
const userStore = useUserStore()
const loginFormRef = ref()

const loginForm = reactive<LoginForm>({
  username: '',
  password: '',
  remember: false
})

const loginRules = reactive({
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 20, message: '用户名长度在 3 到 20 个字符', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, max: 20, message: '密码长度在 6 到 20 个字符', trigger: 'blur' }
  ]
})

const loginError = ref('')

onMounted(() => {
  const savedUsername = localStorage.getItem('rememberedUsername')
  if (savedUsername) {
    loginForm.username = savedUsername
    loginForm.remember = true
  }
})

const handleLogin = async () => {
  if (!loginFormRef.value) return
  loginError.value = ''
  try {
    await loginFormRef.value.validate()
    const result = await userStore.login(loginForm)
    if (result) {
      if (loginForm.remember) {
        localStorage.setItem('rememberedUsername', loginForm.username)
      } else {
        localStorage.removeItem('rememberedUsername')
      }
      ElMessage.success('登录成功')
      router.push('/dashboard')
    } else {
      loginError.value = '用户名或密码错误，请重试'
    }
  } catch (err: any) {
    loginError.value = err?.message || '登录失败，请检查用户名和密码'
  }
}
</script>

<style scoped>
.login-page {
  position: relative;
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  padding: 24px;
  box-sizing: border-box;
  background-color: var(--layout-page-bg);
  overflow: hidden;
}

.login-page-bg {
  pointer-events: none;
  position: absolute;
  inset: 0;
  background:
    radial-gradient(ellipse 80% 55% at 12% 10%, rgba(255, 105, 0, 0.07) 0%, transparent 56%),
    radial-gradient(ellipse 70% 50% at 90% 16%, rgba(255, 105, 0, 0.05) 0%, transparent 52%),
    radial-gradient(ellipse 55% 42% at 50% 96%, rgba(0, 0, 0, 0.04) 0%, transparent 48%);
}

.login-card {
  position: relative;
  z-index: 1;
  width: 100%;
  max-width: 420px;
  padding: 36px 36px 32px;
  background: var(--layout-content-surface);
  border: 1px solid var(--layout-sidebar-border);
  border-radius: 16px;
  box-shadow:
    var(--layout-shadow-soft),
    0 24px 48px rgba(0, 0, 0, 0.06);
}

.login-brand {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 28px;
  padding-bottom: 24px;
  border-bottom: 1px solid var(--layout-sidebar-border);
}

.login-brand-mark {
  flex-shrink: 0;
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 700;
  letter-spacing: 0.04em;
  color: #fff;
  background: linear-gradient(135deg, var(--el-color-primary) 0%, var(--el-color-primary-light-3) 100%);
  box-shadow: 0 4px 16px rgba(255, 105, 0, 0.2);
}

.login-brand-text {
  min-width: 0;
}

.login-title {
  margin: 0 0 4px;
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.02em;
  color: var(--layout-sidebar-text-strong);
  line-height: 1.3;
}

.login-subtitle {
  margin: 0;
  font-size: 13px;
  font-weight: 400;
  color: var(--layout-sidebar-text);
}

.login-form {
  width: 100%;
}

.login-form :deep(.el-form-item__label) {
  color: var(--layout-sidebar-text-strong);
  font-weight: 500;
}

.remember-item {
  margin-bottom: 18px;
}

.login-error-alert {
  margin-bottom: 16px;
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
</style>
