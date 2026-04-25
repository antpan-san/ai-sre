<template>
  <div class="security-hardening">
    <div class="page-header">
      <div class="page-header-bar">
        <el-button link type="primary" :icon="ArrowLeft" @click="backToHome">返回工具总览</el-button>
      </div>
      <h2>系统安全加固</h2>
      <p>配置系统安全策略，提升服务器安全性。请先选择目标节点与系统类型，再勾选要应用的策略。</p>
    </div>

    <div class="content-container">
      <el-card class="content-card">
        <template #header>
          <div class="card-header">
            <h3>安全加固配置</h3>
          </div>
        </template>

        <NodeSystemSelector v-model="target" class="target-block" />

        <div class="security-container">
          <el-checkbox-group v-model="securityOptions">
            <div class="security-item">
              <el-checkbox label="disable_ssh_root_login">禁用SSH root登录</el-checkbox>
              <el-tooltip content="禁止使用root用户直接SSH登录系统" placement="top">
                <el-icon class="help-icon"><QuestionFilled /></el-icon>
              </el-tooltip>
            </div>
            <div class="security-item">
              <el-checkbox label="change_ssh_port">修改SSH端口</el-checkbox>
              <el-tooltip content="将SSH默认端口22修改为自定义端口，提高安全性" placement="top">
                <el-icon class="help-icon"><QuestionFilled /></el-icon>
              </el-tooltip>
              <el-input-number
                v-if="securityOptions.includes('change_ssh_port')"
                v-model="sshPort"
                :min="1024"
                :max="65535"
                :step="1"
                :precision="0"
                style="width: 120px; margin-left: 10px"
                placeholder="端口号"
              />
            </div>
            <div class="security-item">
              <el-checkbox label="enable_firewall">启用防火墙</el-checkbox>
              <el-tooltip content="启用系统防火墙，并配置基本规则" placement="top">
                <el-icon class="help-icon"><QuestionFilled /></el-icon>
              </el-tooltip>
            </div>
            <div class="security-item">
              <el-checkbox label="disable_unnecessary_services">禁用不必要服务</el-checkbox>
              <el-tooltip content="禁用系统中不需要的服务，减少安全风险" placement="top">
                <el-icon class="help-icon"><QuestionFilled /></el-icon>
              </el-tooltip>
            </div>
            <div class="security-item">
              <el-checkbox label="update_system">系统更新</el-checkbox>
              <el-tooltip content="更新系统到最新版本，修复安全漏洞" placement="top">
                <el-icon class="help-icon"><QuestionFilled /></el-icon>
              </el-tooltip>
            </div>
            <div class="security-item">
              <el-checkbox label="setup_fail2ban">安装Fail2ban</el-checkbox>
              <el-tooltip content="安装并配置Fail2ban，防止暴力破解" placement="top">
                <el-icon class="help-icon"><QuestionFilled /></el-icon>
              </el-tooltip>
            </div>
          </el-checkbox-group>
        </div>

        <div class="card-actions">
          <el-button
            type="success"
            @click="applySecuritySettings"
            :disabled="securityOptions.length === 0 || !targetReady"
            :loading="applyingSecurity"
          >
            <el-icon><CircleCheck /></el-icon>
            应用到所选节点
          </el-button>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CircleCheck, QuestionFilled, ArrowLeft } from '@element-plus/icons-vue'
import NodeSystemSelector, { type NodeSystemValue } from '../../components/init-tools/NodeSystemSelector.vue'

const route = useRoute()
const router = useRouter()

// 系统安全加固
const securityOptions = ref<string[]>([])
const applyingSecurity = ref(false)
const sshPort = ref(2222)

const target = ref<NodeSystemValue>({ nodes: [], osType: '' })
const targetReady = computed(() => target.value.nodes.length > 0 && !!target.value.osType)

onMounted(() => {
  const nodesQ = (route.query.nodes as string) || ''
  const osQ = (route.query.osType as string) || ''
  if (nodesQ) target.value.nodes = nodesQ.split(',').filter(Boolean)
  if (osQ) target.value.osType = osQ as NodeSystemValue['osType']
})

const backToHome = () => {
  const q = { ...route.query }
  delete q.nodes
  delete q.osType
  router.push({ path: '/init-tools', query: q })
}

// 应用安全设置
const applySecuritySettings = () => {
  if (!targetReady.value) {
    ElMessage.warning('请先选择目标节点与系统类型')
    return
  }
  ElMessageBox.confirm(
    `将向 ${target.value.nodes.length} 个节点（${target.value.osType}）应用安全策略，可能影响系统功能，是否继续？`,
    '警告',
    { type: 'warning' }
  ).then(() => {
    applyingSecurity.value = true
    // 后端 API 待补齐，此处先做交互反馈
    setTimeout(() => {
      ElMessage.success('系统安全加固任务已下发')
      applyingSecurity.value = false
    }, 1500)
  }).catch(() => {
    // 取消操作
  })
}
</script>

<style scoped>
.security-hardening {
  padding: 20px;
  box-sizing: border-box;
}

.page-header {
  margin-bottom: 30px;
}

.page-header-bar {
  margin-bottom: 6px;
}

.page-header h2 {
  color: #1890ff;
  margin-bottom: 10px;
}

.target-block {
  margin-bottom: 16px;
}

.content-container {
  height: calc(100% - 100px);
  overflow: auto;
}

.content-card {
  max-width: 100%;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-header h3 {
  margin: 0;
  color: #374151;
  font-size: 16px;
  font-weight: 600;
}

/* 系统安全加固 */
.security-container {
  margin-bottom: 20px;
}

.security-item {
  display: flex;
  align-items: center;
  margin-bottom: 10px;
}

.help-icon {
  margin-left: 5px;
  color: #9ca3af;
  cursor: help;
}

/* 卡片操作 */
.card-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid #e5e7eb;
}
</style>