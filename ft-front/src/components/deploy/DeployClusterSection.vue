<template>
  <div class="deploy-cluster-section">
    <el-card v-if="showK8s" class="deploy-cluster-section__k8s" shadow="never">
      <template #header>
        <div class="deploy-cluster-section__k8s-head">
          <div>
            <h4 class="deploy-cluster-section__card-title">Kubernetes 集群交付</h4>
            <p class="deploy-cluster-section__card-desc">离线/在线安装、恢复与卸载指引</p>
          </div>
          <el-button type="primary" size="small" @click="goK8sDeploy">新建集群</el-button>
        </div>
      </template>
      <div class="deploy-workflow-grid">
        <article
          v-for="item in k8sWorkflows"
          :key="item.title"
          class="deploy-workflow-card"
          @click="router.push(item.path)"
        >
          <el-icon :size="22"><component :is="item.icon" /></el-icon>
          <div>
            <h4>{{ item.title }}</h4>
            <p>{{ item.desc }}</p>
          </div>
          <el-button type="primary" link size="small">进入</el-button>
        </article>
      </div>
    </el-card>

    <div v-if="showMirror" class="deploy-cluster-section__entries">
      <el-card class="deploy-entry-card" shadow="hover" @click="router.push(mirrorPath)">
        <div class="deploy-entry-card__body">
          <el-icon :size="22"><Download /></el-icon>
          <div>
            <h4 class="deploy-entry-card__title">K8s 制品目录</h4>
            <p class="deploy-entry-card__desc">内网制品 manifest 与离线安装包索引</p>
          </div>
          <el-button type="primary" link size="small">进入</el-button>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { Connection, Download, RefreshRight, Delete } from '@element-plus/icons-vue'

const props = withDefaults(defineProps<{
  showK8s?: boolean
  showMirror?: boolean
  k8sDeployPath?: string
  k8sProgressPath?: string
  mirrorPath?: string
}>(), {
  showK8s: true,
  showMirror: true,
  k8sDeployPath: '/app/service/k8s-deploy',
  k8sProgressPath: '/app/service/k8s-deploy/progress',
  mirrorPath: '/app/k8s-mirror'
})

const router = useRouter()

const mirrorPath = computed(() => props.mirrorPath)
const k8sWorkflows = computed(() => [
  {
    title: '新建集群',
    desc: '生成安装参数与 bundle，复制到控制机执行安装。',
    path: props.k8sDeployPath,
    icon: Connection,
  },
  {
    title: '恢复引导',
    desc: '查看 recover 命令模板与恢复前检查说明。',
    path: props.k8sDeployPath,
    icon: RefreshRight,
  },
  {
    title: '卸载引导',
    desc: '查看 uninstall 命令模板，避免默认执行破坏性清理。',
    path: props.k8sDeployPath,
    icon: Delete,
  },
])

const goK8sDeploy = () => router.push(props.k8sDeployPath)
</script>

<style scoped>
.deploy-cluster-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.deploy-cluster-section__k8s-head {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}
.deploy-cluster-section__card-title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
}
.deploy-cluster-section__card-desc {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.deploy-workflow-grid,
.deploy-cluster-section__entries {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 12px;
}
.deploy-workflow-card,
.deploy-entry-card {
  cursor: pointer;
  border-radius: 10px;
}
.deploy-workflow-card {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 92px;
  padding: 14px;
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
}
.deploy-workflow-card h4,
.deploy-entry-card__title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}
.deploy-workflow-card p,
.deploy-entry-card__desc {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.45;
}
.deploy-workflow-card .el-button,
.deploy-entry-card .el-button {
  margin-left: auto;
  flex-shrink: 0;
}
.deploy-entry-card__body {
  display: flex;
  align-items: center;
  gap: 12px;
}
@media (max-width: 640px) {
  .deploy-workflow-card,
  .deploy-entry-card__body {
    align-items: flex-start;
  }
}
</style>
