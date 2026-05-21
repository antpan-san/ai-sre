<template>
  <div class="deploy-center deploy-config-page page-shell page-shell--crud-wide">
    <AppPageHeader
      title="工作负载"
      description="只展示当前账号已开通的可执行工作流；未开通能力请到能力中心订阅后使用。"
    >
      <template #actions>
        <el-button size="small" :loading="loading" @click="refresh">刷新</el-button>
        <el-button size="small" @click="router.push('/app/capabilities')">能力中心</el-button>
      </template>
    </AppPageHeader>

    <el-empty
      v-if="!loading && !hasVisibleWorkflows"
      class="workload-empty"
      description="暂无已开通工作流，请到能力中心订阅后使用。"
    >
      <el-button type="primary" @click="router.push('/app/capabilities')">前往能力中心</el-button>
    </el-empty>

    <template v-else>
      <section v-if="k8sEntitled || k8sMirrorEntitled" id="cluster" class="deploy-config-category">
        <h3 class="deploy-config-category__title">Kubernetes 交付</h3>
        <p class="deploy-config-category__desc">集群安装、恢复/卸载指引与离线制品目录；执行记录请到执行记录页查看。</p>
        <DeployClusterSection
          :show-k8s="k8sEntitled"
          :show-mirror="k8sMirrorEntitled"
        />
      </section>

      <section v-if="serviceDeployEntitled" id="services" class="deploy-config-category">
        <h3 class="deploy-config-category__title">应用服务部署</h3>
        <p class="deploy-config-category__desc">中间件与应用服务按领域分类展示，展开后配置参数并生成部署脚本。</p>
        <ServiceDeployGrid grouped />
      </section>

      <section v-if="linuxHostsEntitled" id="linux-hosts" class="deploy-config-category">
        <h3 class="deploy-config-category__title">Linux 主机</h3>
        <p class="deploy-config-category__desc">进入主机服务管理，执行 systemd 查询与服务启停等主机运维工作流。</p>
        <div class="workload-action-grid">
          <el-card class="workload-action-card" shadow="hover" @click="router.push('/app/service/linux')">
            <div class="workload-action-card__body">
              <el-icon :size="22"><Cpu /></el-icon>
              <div>
                <h4>Linux 服务管理</h4>
                <p>查看服务状态，执行 start / stop / restart / enable / disable 等操作。</p>
              </div>
              <el-button type="primary" link size="small">进入</el-button>
            </div>
          </el-card>
        </div>
      </section>

      <section v-if="initToolsEntitled" id="init-tools" class="deploy-config-category">
        <h3 class="deploy-config-category__title">节点初始化</h3>
        <p class="deploy-config-category__desc">部署前环境准备：时间同步、系统参数、安全加固与磁盘分区优化。</p>
        <InitToolsSection />
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Cpu } from '@element-plus/icons-vue'
import AppPageHeader from '../../components/app/AppPageHeader.vue'
import DeployClusterSection from '../../components/deploy/DeployClusterSection.vue'
import ServiceDeployGrid from '../../components/deploy/ServiceDeployGrid.vue'
import InitToolsSection from '../../components/deploy/InitToolsSection.vue'
import '../../assets/app-workbench.css'
import '../../assets/deploy-config.css'
import { useCapabilityCatalog } from '../../composables/useCapabilityCatalog'

const route = useRoute()
const router = useRouter()
const { loading, load: loadCaps, filterCapabilities, isEntitledStatus } = useCapabilityCatalog()

const deliveryCaps = computed(() => filterCapabilities({ category: 'delivery', status: 'all' }))

const isCapEntitled = (id: string) => {
  const cap = deliveryCaps.value.find((c) => c.id === id)
  return cap ? isEntitledStatus(cap.status) : false
}

const k8sEntitled = computed(() => isCapEntitled('k8s_delivery'))
const k8sMirrorEntitled = computed(() => isCapEntitled('k8s_mirror'))
const serviceDeployEntitled = computed(() => isCapEntitled('service_deploy'))
const linuxHostsEntitled = computed(() => isCapEntitled('linux_hosts'))
const initToolsEntitled = computed(() => isCapEntitled('init_tools'))

const hasVisibleWorkflows = computed(() =>
  k8sEntitled.value ||
  k8sMirrorEntitled.value ||
  serviceDeployEntitled.value ||
  linuxHostsEntitled.value ||
  initToolsEntitled.value
)

const scrollToHash = async () => {
  const hash = route.hash?.replace('#', '')
  if (!hash) return
  await nextTick()
  document.getElementById(hash)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

watch(
  () => route.hash,
  () => {
    void scrollToHash()
  }
)

const refresh = async () => {
  await loadCaps(true)
}

onMounted(async () => {
  await loadCaps()
  await scrollToHash()
})
</script>

<style scoped>
.workload-empty {
  padding: 36px 0;
  border: 1px dashed var(--el-border-color);
  border-radius: 14px;
  background: var(--el-fill-color-extra-light);
}
.workload-action-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
}
.workload-action-card {
  cursor: pointer;
  border-radius: 12px;
}
.workload-action-card__body {
  display: flex;
  align-items: center;
  gap: 12px;
}
.workload-action-card__body h4 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}
.workload-action-card__body p {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.45;
}
.workload-action-card__body .el-button {
  margin-left: auto;
  flex-shrink: 0;
}
@media (max-width: 640px) {
  .workload-action-card__body {
    align-items: flex-start;
  }
}
</style>
