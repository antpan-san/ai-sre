<template>
  <div class="app-settings page-shell">
    <header class="page-head">
      <div class="page-head-copy">
        <h2 class="page-title">设置</h2>
        <p class="page-desc--muted">CLI 安装、账号信息与帮助入口。</p>
      </div>
    </header>

    <el-card shadow="never" class="settings-card">
      <template #header>安装 ai-sre CLI</template>
      <p class="hint">在目标机器执行以下命令（需登录后生成带 token 的安装脚本）：</p>
      <div class="install-row">
        <el-input v-model="installCmd" readonly type="textarea" :rows="3" />
        <el-button type="primary" :loading="installGenerating" @click="genInstall">生成并复制</el-button>
      </div>
      <p v-if="installVersion" class="meta">当前平台版本：{{ installVersion }}</p>
    </el-card>

    <el-card shadow="never" class="settings-card">
      <template #header>账号</template>
      <dl class="account-dl">
        <dt>用户名</dt>
        <dd>{{ username }}</dd>
        <dt>角色</dt>
        <dd>{{ roleLabel }}</dd>
      </dl>
    </el-card>

    <el-card shadow="never" class="settings-card">
      <template #header>订阅与能力</template>
      <p class="hint">查看全部能力与订阅状态，或发起订阅。</p>
      <el-button type="primary" link @click="goDeploy">打开部署中心</el-button>
    </el-card>

    <el-card shadow="never" class="settings-card">
      <template #header>帮助</template>
      <el-button link type="primary" @click="goErrorCodes">错误码查询</el-button>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { createCLIInstallSession } from '../../api/cli'
import { fetchAiSreCLIVersion } from '../../utils/aiSreCliVersion'
import { copyTextToClipboard } from '../../utils/clipboard'
import { getStoredAuthToken } from '../../utils/installAiSre'

const router = useRouter()
const installCmd = ref('curl -fsSL <平台地址>/ft-api/api/k8s/deploy/install-ai-sre.sh | bash')
const installGenerating = ref(false)
const installVersion = ref('')

const user = computed(() => {
  try {
    return JSON.parse(localStorage.getItem('userInfo') || '{}') as { username?: string; role?: string }
  } catch {
    return {}
  }
})
const username = computed(() => user.value.username || '—')
const roleLabel = computed(() => {
  const r = user.value.role
  if (r === 'super_admin') return '超级管理员'
  if (r === 'admin') return '管理员'
  if (r === 'user') return '普通用户'
  return r || '—'
})

const genInstall = async () => {
  if (!getStoredAuthToken()) {
    ElMessage.warning('请先登录')
    return
  }
  installGenerating.value = true
  try {
    const data = await createCLIInstallSession()
    installCmd.value = data.command
    await copyTextToClipboard(data.command)
    ElMessage.success('已生成并复制安装命令')
  } catch {
    ElMessage.error('生成失败')
  } finally {
    installGenerating.value = false
  }
}

const goDeploy = () => router.push('/app/deploy')
const goErrorCodes = () => router.push('/app/help/error-codes')

onMounted(async () => {
  try {
    const info = await fetchAiSreCLIVersion()
    installVersion.value = info?.version || ''
  } catch {
    installVersion.value = ''
  }
})
</script>

<style scoped>
.settings-card {
  margin-bottom: 16px;
  max-width: 720px;
}
.hint {
  margin: 0 0 12px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.install-row {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.meta {
  margin: 10px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.account-dl {
  display: grid;
  grid-template-columns: 88px 1fr;
  gap: 8px 12px;
  margin: 0;
}
.account-dl dt {
  color: var(--el-text-color-secondary);
}
.account-dl dd {
  margin: 0;
}
</style>
