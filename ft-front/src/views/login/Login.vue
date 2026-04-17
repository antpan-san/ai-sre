<template>
  <div class="login-container">
    <div class="login-box">
      <h2 class="login-title">OpsFleetPilot</h2>
      <p class="login-subtitle">运维管理平台</p>
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
            @click="handleLogin"
            class="login-btn"
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

// 登录表单
const loginForm = reactive<LoginForm>({
  username: '',
  password: '',
  remember: false
})

// 登录规则
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

// 仅从 localStorage 读取记住的用户名（不存储密码）
onMounted(() => {
  const savedUsername = localStorage.getItem('rememberedUsername')
  if (savedUsername) {
    loginForm.username = savedUsername
    loginForm.remember = true
  }
})

// 处理登录
const handleLogin = async () => {
  if (!loginFormRef.value) return
  loginError.value = ''
  try {
    await loginFormRef.value.validate()
    const result = await userStore.login(loginForm)
    if (result) {
      // 仅记住用户名，不在客户端存储密码
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
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
  background: linear-gradient(135deg, #1E40AF 0%, #3B82F6 50%, #60A5FA 100%);
}

.login-box {
  width: 400px;
  padding: 40px;
  background-color: #fff;
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.15);
}

.login-title {
  text-align: center;
  margin-bottom: 4px;
  color: #1E40AF;
  font-size: 28px;
  font-weight: 700;
}

.login-subtitle {
  text-align: center;
  margin-bottom: 30px;
  color: #909399;
  font-size: 14px;
}

.login-form {
  width: 100%;
}

.remember-item {
  margin-bottom: 20px;
}

.login-error-alert {
  margin-bottom: 16px;
}

.submit-item {
  margin-bottom: 0;
}

.login-btn {
  width: 100%;
  height: 42px;
  font-size: 16px;
}
</style>
