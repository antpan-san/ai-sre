<template>
  <div class="init-tools-home page-shell">
    <header v-if="showHeader" class="page-header">
      <div class="page-header-inner">
        <h2 class="page-title">节点初始化</h2>
        <p class="page-desc page-desc--muted">填写参数并生成脚本，在控制机上执行；Ansible 负责多节点并行。</p>
      </div>
    </header>

    <el-alert
      v-if="fromK8sDeploy"
      type="info"
      :closable="false"
      class="from-k8s-banner"
      show-icon
    >
      <template #title>
        <div class="from-k8s-banner-inner">
          <span>
            正在为 Kubernetes 集群
            <strong v-if="k8sCluster">「{{ k8sCluster }}」</strong>
            做部署前的环境优化。完成所需项后可以直接返回部署向导继续下一步。
          </span>
          <el-button type="primary" size="small" @click="backToK8sDeploy">
            返回 K8s 部署
            <el-icon class="el-icon--right"><ArrowRight /></el-icon>
          </el-button>
        </div>
      </template>
    </el-alert>

    <InitToolsSection />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowRight } from '@element-plus/icons-vue'
import InitToolsSection from '../../components/deploy/InitToolsSection.vue'

withDefaults(defineProps<{ showHeader?: boolean }>(), { showHeader: true })

const router = useRouter()
const route = useRoute()

const fromK8sDeploy = computed(() => route.query.from === 'k8s-deploy')
const k8sCluster = computed(() => (route.query.cluster as string) || '')

const backToK8sDeploy = () => {
  router.push({ path: '/service/k8s-deploy' })
}

onMounted(() => {
  if (fromK8sDeploy.value) {
    ElMessage.info('建议顺序：先「时间同步」→ 再「系统参数优化」')
  }
})
</script>

<style scoped>
.init-tools-home {
  max-width: none;
  width: 100%;
  height: 100%;
  min-height: 0;
  padding: 12px 8px 16px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  gap: 0;
}
.page-header {
  flex: 0 0 auto;
  margin-bottom: 8px;
}
.page-header-inner {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.page-title {
  margin: 0;
  font-size: 18px;
  color: #0f172a;
}
.page-desc {
  margin: 0;
  color: #475569;
  font-size: 12px;
}
.from-k8s-banner {
  flex: 0 0 auto;
  margin-bottom: 14px;
}
.from-k8s-banner-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
</style>
