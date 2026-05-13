<template>
  <div class="register-page">
    <div class="register-page-bg" aria-hidden="true" />
    <div class="register-card">
      <header class="register-head">
        <h1 class="register-title">创建账号</h1>
        <p class="register-sub">注册为普通用户（<span class="muted">admin 需由管理员分配</span>）</p>
      </header>

      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-position="top"
        class="register-form"
        @submit.prevent="submit"
      >
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" placeholder="3–50 个字符" clearable autocomplete="username" />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="form.email" placeholder="用于找回与通知" clearable autocomplete="email" />
        </el-form-item>
        <el-form-item label="显示名称（可选）" prop="full_name">
          <el-input v-model="form.full_name" placeholder="姓名或备注" clearable maxlength="100" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="至少 6 位"
            show-password
            autocomplete="new-password"
          />
        </el-form-item>
        <el-form-item label="确认密码" prop="password2">
          <el-input
            v-model="form.password2"
            type="password"
            placeholder="再次输入密码"
            show-password
            autocomplete="new-password"
          />
        </el-form-item>

        <el-alert v-if="err" :title="err" type="error" show-icon class="register-alert" @close="err = ''" />

        <el-form-item class="register-actions">
          <el-button
            type="primary"
            class="register-btn"
            :loading="loading"
            :disabled="!regAllowed"
            native-type="submit"
            @click="submit"
          >
            注册
          </el-button>
        </el-form-item>
        <div class="register-footer">
          <router-link class="register-link" to="/login">已有账号？去登录</router-link>
        </div>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getPublicAuthOptions, register } from '../../api/auth'
import type { RegisterForm } from '../../types'

const router = useRouter()
const formRef = ref()
const loading = ref(false)
const err = ref('')
const regAllowed = ref(true)

const form = reactive({
  username: '',
  email: '',
  full_name: '',
  password: '',
  password2: ''
})

const rules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 50, message: '用户名 3–50 字符', trigger: 'blur' }
  ],
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email' as const, message: '邮箱格式不正确', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, max: 128, message: '密码至少 6 位', trigger: 'blur' }
  ],
  password2: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    {
      validator: (_r: unknown, v: string, cb: (e?: Error) => void) => {
        if (v !== form.password) cb(new Error('两次密码不一致'))
        else cb()
      },
      trigger: 'blur'
    }
  ]
}

onMounted(async () => {
  try {
    const o = await getPublicAuthOptions()
    regAllowed.value = o.public_registration_allowed
    if (!regAllowed.value) {
      err.value = '管理员已关闭公开注册'
    }
  } catch {
    regAllowed.value = true
  }
})

const submit = async () => {
  if (!regAllowed.value) return
  if (!formRef.value) return
  err.value = ''
  try {
    await formRef.value.validate()
    loading.value = true
    const payload: RegisterForm = {
      username: form.username.trim(),
      email: form.email.trim(),
      password: form.password,
      full_name: form.full_name.trim() || undefined
    }
    await register(payload)
    ElMessage.success('注册成功，请登录')
    router.push({ path: '/login', query: { registered: '1', u: payload.username } })
  } catch (e: unknown) {
    if (e && typeof e === 'object' && 'message' in e) {
      err.value = String((e as Error).message)
    }
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.register-page {
  position: relative;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px 16px;
  box-sizing: border-box;
  background: var(--layout-page-bg);
  overflow: hidden;
}

.register-page-bg {
  pointer-events: none;
  position: absolute;
  inset: 0;
  background:
    radial-gradient(ellipse 70% 50% at 88% 12%, rgba(255, 105, 0, 0.07) 0%, transparent 50%),
    radial-gradient(ellipse 60% 45% at 10% 88%, rgba(0, 0, 0, 0.03) 0%, transparent 48%);
}

.register-card {
  position: relative;
  z-index: 1;
  width: 100%;
  max-width: 440px;
  padding: 28px 28px 24px;
  background: var(--layout-content-surface);
  border: 1px solid var(--layout-sidebar-border);
  border-radius: 16px;
  box-shadow: var(--layout-shadow-soft), 0 20px 40px rgba(0, 0, 0, 0.06);
}

.register-head {
  margin-bottom: 20px;
}

.register-title {
  margin: 0 0 6px;
  font-size: 20px;
  font-weight: 600;
  color: var(--layout-sidebar-text-strong);
}

.register-sub {
  margin: 0;
  font-size: 13px;
  color: var(--layout-sidebar-text);
}

.muted {
  color: var(--el-text-color-secondary);
}

.register-form :deep(.el-form-item__label) {
  font-weight: 500;
}

.register-alert {
  margin-bottom: 12px;
}

.register-actions {
  margin-bottom: 8px;
}

.register-btn {
  width: 100%;
  height: 44px;
  border-radius: 10px;
  font-weight: 600;
}

.register-footer {
  text-align: center;
}

.register-link {
  font-size: 13px;
  color: var(--el-color-primary);
  text-decoration: none;
}

.register-link:hover {
  text-decoration: underline;
}
</style>
