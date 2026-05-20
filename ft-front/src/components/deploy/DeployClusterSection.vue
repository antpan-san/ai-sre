<template>
  <div class="deploy-cluster-section">
    <el-card class="deploy-cluster-section__k8s" shadow="never">
      <template #header>
        <div class="deploy-cluster-section__k8s-head">
          <div>
            <h4 class="deploy-cluster-section__card-title">Kubernetes</h4>
            <p class="deploy-cluster-section__card-desc">离线/在线集群安装、恢复与卸载</p>
          </div>
          <div class="deploy-cluster-section__actions">
            <el-button type="primary" size="small" @click="goK8sDeploy">新建集群</el-button>
            <el-button size="small" @click="goK8sProgress">部署进度</el-button>
          </div>
        </div>
      </template>
      <K8sClusterPanel
        v-if="k8sEntitled"
        :deploy-path="k8sDeployPath"
        :progress-path="k8sProgressPath"
        :max-rows="5"
        :show-toolbar="false"
        hint="最近集群；完整列表见执行记录。"
      />
      <div v-else class="deploy-cluster-section__locked">
        <p>开通 Kubernetes 交付能力后可在此查看集群并新建部署。</p>
        <el-button v-if="k8sCap?.can_subscribe" type="warning" size="small" @click="emit('subscribe-k8s')">
          订阅
        </el-button>
        <el-button v-else type="info" size="small" @click="emit('contact-admin')">联系管理员</el-button>
      </div>
    </el-card>

    <div class="deploy-cluster-section__entries">
      <el-card
        v-for="entry in linkEntries"
        :key="entry.path"
        class="deploy-entry-card"
        shadow="hover"
        @click="router.push(entry.path)"
      >
        <div class="deploy-entry-card__body">
          <el-icon :size="22"><component :is="entry.icon" /></el-icon>
          <div>
            <h4 class="deploy-entry-card__title">{{ entry.title }}</h4>
            <p class="deploy-entry-card__desc">{{ entry.desc }}</p>
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
import { Cpu, Download } from '@element-plus/icons-vue'
import K8sClusterPanel from '../k8s/K8sClusterPanel.vue'
import type { ResolvedCapability } from '../../composables/useCapabilityCatalog'

const props = defineProps<{
  k8sEntitled?: boolean
  k8sCap?: ResolvedCapability | null
  k8sDeployPath?: string
  k8sProgressPath?: string
  linuxPath?: string
  mirrorPath?: string
}>()

const emit = defineEmits<{
  'subscribe-k8s': []
  'contact-admin': []
}>()

const router = useRouter()

const k8sDeployPath = computed(() => props.k8sDeployPath || '/app/service/k8s-deploy')
const k8sProgressPath = computed(() => props.k8sProgressPath || '/app/service/k8s-deploy/progress')

const linkEntries = computed(() => [
  {
    path: props.linuxPath || '/app/service/linux',
    title: 'Linux 主机',
    desc: '主机上的服务状态与运维操作',
    icon: Cpu,
  },
  {
    path: props.mirrorPath || '/app/k8s-mirror',
    title: 'K8s 制品目录',
    desc: '内网制品 manifest 与离线安装包索引',
    icon: Download,
  },
])

const goK8sDeploy = () => router.push(k8sDeployPath.value)
const goK8sProgress = () => router.push(k8sProgressPath.value)
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
.deploy-cluster-section__actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}
.deploy-cluster-section__locked {
  padding: 12px 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.deploy-cluster-section__entries {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
@media (max-width: 640px) {
  .deploy-cluster-section__entries {
    grid-template-columns: 1fr;
  }
}
.deploy-entry-card {
  cursor: pointer;
  border-radius: 10px;
}
.deploy-entry-card__body {
  display: flex;
  align-items: center;
  gap: 12px;
}
.deploy-entry-card__title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}
.deploy-entry-card__desc {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
