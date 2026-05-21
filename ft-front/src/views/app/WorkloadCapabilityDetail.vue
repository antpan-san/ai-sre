<template>
  <div class="workload-detail page-shell page-shell--crud-wide">
    <AppPageHeader :title="pageTitle" :description="pageDescription">
      <template #actions>
        <el-button size="small" @click="router.push('/app/workloads')">返回工作负载</el-button>
        <el-button v-if="primaryLink" size="small" type="primary" @click="router.push(primaryLink.path)">
          打开专门页面
        </el-button>
      </template>
    </AppPageHeader>

    <el-empty v-if="!detail" class="workload-detail__empty" description="未找到可展示的工作负载能力">
      <el-button type="primary" @click="router.push('/app/workloads')">返回工作负载</el-button>
    </el-empty>

    <template v-else>
      <el-card class="workload-detail__hero" shadow="never" :style="{ '--workload-accent': detail.accent }">
        <div class="workload-detail__hero-main">
          <div>
            <p class="workload-detail__eyebrow">{{ detail.groupLabel }}</p>
            <h2>{{ detail.title }}</h2>
            <p class="workload-detail__desc">{{ detail.description }}</p>
            <div class="workload-detail__tags">
              <el-tag v-for="tag in detail.tags" :key="tag" size="small" effect="plain">{{ tag }}</el-tag>
            </div>
          </div>
          <div class="workload-detail__status">
            <el-tag :type="detail.statusType">{{ detail.status }}</el-tag>
            <span>{{ detail.packLabel }}</span>
          </div>
        </div>

        <div class="workload-detail__actions">
          <el-button v-for="link in specialLinks" :key="link.path" :type="link.primary ? 'primary' : 'default'" @click="router.push(link.path)">
            {{ link.label }}
          </el-button>
          <el-button @click="router.push('/app/workloads')">回到工作负载</el-button>
        </div>
      </el-card>

      <section v-if="detail.commands.length" class="workload-detail__section">
        <h3>常用命令</h3>
        <div class="workload-detail__commands">
          <pre v-for="cmd in detail.commands" :key="cmd.label" class="workload-detail__command"><code>{{ cmd.label }}\n{{ cmd.template }}</code></pre>
        </div>
      </section>

      <section class="workload-detail__section">
        <h3>专门页面</h3>
        <div class="workload-detail__links">
          <el-card v-for="link in specialLinks" :key="link.path" class="workload-detail__link-card" shadow="hover" @click="router.push(link.path)">
            <div>
              <h4>{{ link.label }}</h4>
              <p>{{ link.hint }}</p>
            </div>
            <el-button type="primary" link>进入</el-button>
          </el-card>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppPageHeader from '../../components/app/AppPageHeader.vue'
import { useCapabilityCatalog } from '../../composables/useCapabilityCatalog'
import { useServiceDeploy } from '../../composables/useServiceDeploy'
import { CAPABILITY_CATEGORY_LABELS } from '../../config/capabilityCatalog'
import {
  WORKLOAD_CAPABILITY_META,
  WORKLOAD_DETAIL_SERVICE_LINK,
  WORKLOAD_SERVICE_GROUPS,
} from '../../config/workloadPresentation'
import '../../assets/app-workbench.css'

const route = useRoute()
const router = useRouter()
const { load: loadCaps, filterCapabilities, isEntitledStatus } = useCapabilityCatalog()
const { catalog: serviceCatalog } = useServiceDeploy()

const serviceKey = computed(() => String(route.params.serviceKey || '').trim())
const capId = computed(() => String(route.params.capId || '').trim())

const deliveryCaps = computed(() => filterCapabilities({ category: 'delivery', status: 'all' }))
const serviceGroup = computed(() => WORKLOAD_SERVICE_GROUPS.find((group) => group.services.includes(serviceKey.value)))

const statusType = (status: string) => {
  if (status === '已订阅' || status === '免费可用' || status === '管理员已开通') return 'success'
  if (status === '未订阅' || status === '联系管理员开通') return 'warning'
  return 'info'
}

const detail = computed(() => {
  if (serviceKey.value) {
    const service = serviceCatalog.find((item) => item.key === serviceKey.value)
    const base = deliveryCaps.value.find((item) => item.id === 'service_deploy')
    if (!service || !base || !isEntitledStatus(base.status)) return null
    return {
      title: service.name,
      description: service.description,
      groupLabel: serviceGroup.value?.title || CAPABILITY_CATEGORY_LABELS.delivery,
      tags: service.tags,
      status: base.status,
      statusType: statusType(base.status),
      packLabel: base.pack_display_name || base.pack_key || '节点运维包',
      accent: serviceGroup.value?.accent || '#2563eb',
      commands: [
        { label: '服务安装', template: `ai-sre ops service install ${service.key} --target <host>` },
        { label: '服务更新', template: `ai-sre ops service update ${service.key} --target <host>` },
        { label: '服务卸载', template: `ai-sre ops service uninstall ${service.key} --target <host>` }
      ],
      links: [
        {
          label: '打开服务页面',
          path: WORKLOAD_DETAIL_SERVICE_LINK(service.key),
          primary: true,
          hint: '进入该服务的独立部署页面并直接生成脚本。'
        }
      ]
    }
  }

  if (!capId.value) return null
  const meta = WORKLOAD_CAPABILITY_META[capId.value]
  const requiredCapId = meta?.capabilityId || capId.value
  const cap = deliveryCaps.value.find((item) => item.id === requiredCapId)
  if (!meta || !cap || !isEntitledStatus(cap.status)) return null
  return {
    title: meta.title,
    description: meta.description,
    groupLabel:
      meta.capabilityId === 'init_tools'
        ? '节点初始化'
        : meta.capabilityId === 'k8s_mirror' || meta.id === 'k8s_mirror'
          ? 'K8s 制品目录'
          : CAPABILITY_CATEGORY_LABELS.delivery,
    tags: meta.tags,
    status: cap.status,
    statusType: statusType(cap.status),
    packLabel: cap.pack_display_name || cap.pack_key || '免费能力',
    accent: meta.accent,
    commands: meta.commands || cap.commands || [],
    links: meta.links.map((link) => ({
      ...link,
      hint:
        link.path === '/app/service/k8s-deploy'
          ? '进入 K8s 安装与 bundle 页面。'
          : link.path === '/app/service/k8s-deploy/progress'
            ? '查看安装进度与历史任务。'
            : link.path === '/app/k8s-mirror'
              ? '查看制品 manifest 与离线包。'
              : link.path === '/app/service/linux'
                ? '查看主机服务状态与操作。'
                : link.path.startsWith('/app/init-tools')
                  ? '查看时间同步、参数优化、安全加固与磁盘分区脚本。'
                  : '进入对应的专门页面。'
    }))
  }
})

const pageTitle = computed(() => detail.value?.title || '工作负载详情')
const pageDescription = computed(() => detail.value?.description || '请选择一个已开通的工作负载能力查看详情。')
const specialLinks = computed(() => detail.value?.links || [])
const primaryLink = computed(() => specialLinks.value[0] || null)

watch(pageTitle, (title) => {
  document.title = title
}, { immediate: true })

onMounted(async () => {
  await loadCaps()
})
</script>

<style scoped>
.workload-detail {
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.workload-detail__empty {
  padding: 36px 0;
  border: 1px dashed var(--el-border-color);
  border-radius: 14px;
  background: var(--el-fill-color-extra-light);
}
.workload-detail__hero {
  border-radius: 18px;
  border: 1px solid var(--el-border-color-lighter);
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.98), rgba(248, 250, 252, 0.96)),
    radial-gradient(circle at top right, color-mix(in srgb, var(--workload-accent) 18%, transparent), transparent 34%);
}
.workload-detail__hero-main {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
}
.workload-detail__eyebrow {
  margin: 0 0 6px;
  color: var(--workload-accent);
  font-size: 11px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}
.workload-detail h2 {
  margin: 0 0 8px;
  font-size: 24px;
}
.workload-detail__desc {
  margin: 0;
  color: var(--el-text-color-secondary);
  line-height: 1.7;
  max-width: 720px;
}
.workload-detail__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 12px;
}
.workload-detail__status {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.workload-detail__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 16px;
}
.workload-detail__section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.workload-detail__section h3 {
  margin: 0;
  font-size: 16px;
}
.workload-detail__commands {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 12px;
}
.workload-detail__command {
  margin: 0;
  padding: 14px;
  border-radius: 14px;
  background: var(--el-fill-color-extra-light);
  border: 1px solid var(--el-border-color-lighter);
  overflow: auto;
  white-space: pre-wrap;
  line-height: 1.55;
  font-size: 12px;
}
.workload-detail__links {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
}
.workload-detail__link-card {
  cursor: pointer;
  border-radius: 14px;
}
.workload-detail__link-card :deep(.el-card__body) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.workload-detail__link-card h4 {
  margin: 0;
  font-size: 14px;
}
.workload-detail__link-card p {
  margin: 4px 0 0;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.5;
}
@media (max-width: 720px) {
  .workload-detail__hero-main {
    flex-direction: column;
  }
  .workload-detail__status {
    align-items: flex-start;
  }
}
</style>
