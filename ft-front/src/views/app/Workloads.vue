<template>
  <div class="workloads page-shell">
    <header class="page-head">
      <div class="page-head-copy">
        <h2 class="page-title">工作负载</h2>
        <p class="page-desc--muted">Kubernetes、应用服务、Linux 主机与初始化工具；执行受订阅状态控制。</p>
      </div>
    </header>

    <el-tabs v-model="activeTab" class="workload-tabs">
      <el-tab-pane label="Kubernetes" name="k8s">
        <WorkloadPanel
          title="Kubernetes 集群"
          description="安装、恢复与卸载离线/在线 K8s 集群。"
          :cap="capById('k8s_delivery')"
          :commands="k8sCommands"
          @open="go('/service/k8s-deploy')"
          @subscribe="sub"
        />
      </el-tab-pane>
      <el-tab-pane label="应用服务" name="services">
        <WorkloadPanel
          title="应用服务部署"
          description="中间件与应用组件的安装、更新与卸载。"
          :cap="capById('service_deploy')"
          :commands="serviceCommands"
          @open="go('/service/deploy')"
          @subscribe="sub"
        />
      </el-tab-pane>
      <el-tab-pane label="Linux 主机" name="linux">
        <WorkloadPanel
          title="Linux 主机"
          description="查看与管理主机上的服务状态。"
          :cap="capById('linux_hosts')"
          @open="go('/service/linux')"
          @subscribe="sub"
        />
      </el-tab-pane>
      <el-tab-pane label="初始化工具" name="init">
        <WorkloadPanel
          title="节点初始化"
          description="系统参数、时间同步、安全加固等脚本。"
          :cap="capById('init_tools')"
          @open="go('/init-tools')"
          @subscribe="sub"
        />
      </el-tab-pane>
      <el-tab-pane label="出口代理" name="proxy">
        <WorkloadPanel
          title="出口代理"
          description="配置主机与集群访问外网。"
          :cap="capById('proxy')"
          @open="go('/proxy/config')"
          @subscribe="sub"
        />
      </el-tab-pane>
      <el-tab-pane label="制品目录" name="mirror">
        <WorkloadPanel
          title="K8s 制品目录"
          description="内网 manifest 与离线包索引。"
          :cap="capById('k8s_mirror')"
          @open="go('/k8s-mirror')"
          @subscribe="sub"
        />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import WorkloadPanel from '../../components/app/WorkloadPanel.vue'
import { useCapabilityCatalog, type ResolvedCapability } from '../../composables/useCapabilityCatalog'

const route = useRoute()
const router = useRouter()
const { resolved, shellPrefix, subscribe } = useCapabilityCatalog()

const activeTab = ref(String(route.query.tab || 'k8s'))

watch(
  () => route.query.tab,
  (t) => {
    if (typeof t === 'string' && t) activeTab.value = t
  }
)

const capById = (id: string) => resolved.value.find((c) => c.id === id)

const k8sCommands = [
  'ai-sre ops k8s install --help',
  'ai-sre ops k8s recover --cluster <name>',
  'ai-sre ops k8s uninstall --cluster <name>'
]
const serviceCommands = [
  'ai-sre ops service install <service> --target <host>',
  'ai-sre ops service update <service> --target <host>',
  'ai-sre ops service uninstall <service> --target <host>'
]

const go = (suffix: string) => {
  router.push(`${shellPrefix.value}${suffix}`)
}

const sub = (item: ResolvedCapability) => {
  void subscribe(item)
}
</script>

<style scoped>
.workload-tabs {
  margin-top: 8px;
}
</style>
