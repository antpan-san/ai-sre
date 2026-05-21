<template>
  <div class="capability-center page-shell page-shell--crud-wide">
    <header class="capability-hero">
      <div>
        <p class="eyebrow">Capability Center</p>
        <h2>能力中心</h2>
        <p>查看平台全部能力、所属订阅包和当前账号状态；已订阅直接打开，未订阅从这里开通。</p>
      </div>
      <div class="hero-actions">
        <el-button size="small" :loading="loading" @click="refresh">刷新状态</el-button>
        <el-button size="small" type="primary" @click="goSettings">安装 CLI</el-button>
      </div>
    </header>

    <section class="summary-row">
      <article class="summary-card">
        <span>全部能力</span>
        <strong>{{ summary.total }}</strong>
      </article>
      <article class="summary-card summary-card--ok">
        <span>已可使用</span>
        <strong>{{ usableCount }}</strong>
      </article>
      <article class="summary-card summary-card--warn">
        <span>可在线订阅</span>
        <strong>{{ summary.subscribeable }}</strong>
      </article>
      <article class="summary-card">
        <span>免费能力</span>
        <strong>{{ summary.free }}</strong>
      </article>
    </section>

    <section class="category-menu" aria-label="能力分类">
      <button
        class="category-menu__item"
        :class="{ 'is-active': activeCategory === 'all' }"
        type="button"
        @click="activeCategory = 'all'"
      >
        <span>全部分类</span>
        <strong>{{ summary.total }}</strong>
      </button>
      <button
        v-for="item in categoryMenuItems"
        :key="item.category"
        class="category-menu__item"
        :class="{ 'is-active': activeCategory === item.category }"
        type="button"
        @click="activeCategory = item.category"
      >
        <span>{{ item.label }}</span>
        <strong>{{ item.count }}</strong>
      </button>
    </section>

    <section class="filter-bar">
      <el-input v-model="keyword" clearable placeholder="搜索能力、订阅包、CLI topic" class="filter-search" />
      <el-radio-group v-model="statusFilter" size="small">
        <el-radio-button value="all">全部</el-radio-button>
        <el-radio-button value="entitled">已可用</el-radio-button>
        <el-radio-button value="unsubscribed">未订阅</el-radio-button>
        <el-radio-button value="free">免费</el-radio-button>
      </el-radio-group>
    </section>

    <section v-for="group in groupedCapabilities" :key="group.category" class="capability-group">
      <div class="group-head">
        <div>
          <h3>{{ group.label }}</h3>
          <p>{{ group.desc }}</p>
        </div>
        <el-tag size="small" effect="plain">{{ group.items.length }} 项</el-tag>
      </div>

      <div class="capability-grid">
        <article v-for="item in group.items" :key="item.id" class="capability-card">
          <header class="capability-card__head">
            <div>
              <h4>{{ item.name }}</h4>
              <p>{{ item.description }}</p>
            </div>
            <el-tag size="small" :type="statusType(item.status)">{{ item.status }}</el-tag>
          </header>

          <div class="capability-card__meta">
            <span>{{ item.pack_display_name || item.pack_key || '免费能力' }}</span>
            <code v-if="item.cli_topic">ai-sre check {{ item.cli_topic }}</code>
          </div>

          <div v-if="item.commands?.length" class="command-list">
            <code v-for="cmd in item.commands.slice(0, 3)" :key="cmd.label">{{ cmd.template }}</code>
          </div>

          <footer class="capability-card__actions">
            <el-button
              size="small"
              :type="primaryActionType(item)"
              :disabled="primaryActionDisabled(item)"
              @click="handlePrimary(item)"
            >
              {{ primaryActionLabel(item) }}
            </el-button>
            <el-button v-if="item.cli_topic" size="small" link type="primary" @click="copyCheckCommand(item)">复制命令</el-button>
          </footer>
        </article>
      </div>
    </section>

    <section v-if="!isSuperAdmin" class="evolution-note">
      <h3>自动进化</h3>
      <p>平台会把诊断样本、能力缺口和执行反馈沉淀为技能增强队列；普通用户可查看能力状态，管理入口仅超级管理员可见。</p>
      <div class="evolution-note__items">
        <span>技能增强审查：管理员可见</span>
        <span>自动迭代：管理员可见</span>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  CAPABILITY_CATEGORY_DESC,
  CAPABILITY_CATEGORY_LABELS,
  type CapabilityCategory,
  type SubscriptionStatusLabel
} from '../../config/capabilityCatalog'
import { useCapabilityCatalog, type ResolvedCapability } from '../../composables/useCapabilityCatalog'
import { copyTextToClipboard } from '../../utils/clipboard'

const route = useRoute()
const router = useRouter()
const {
  loading,
  summary,
  load,
  subscribe,
  filterCapabilities,
  categoryOrder,
  isEntitledStatus
} = useCapabilityCatalog()

const keyword = ref('')
const statusFilter = ref<'all' | 'entitled' | 'unsubscribed' | 'free'>('all')
const activeCategory = ref<CapabilityCategory | 'all'>('all')

const role = computed(() => {
  try {
    return String((JSON.parse(localStorage.getItem('userInfo') || '{}') as { role?: string }).role || '')
  } catch {
    return ''
  }
})
const isSuperAdmin = computed(() => role.value === 'super_admin')
const usableCount = computed(() => summary.value.entitled)

const groupedCapabilities = computed(() => {
  const list = filterCapabilities({ q: keyword.value, status: statusFilter.value, category: activeCategory.value })
  return categoryOrder()
    .map((category: CapabilityCategory) => ({
      category,
      label: CAPABILITY_CATEGORY_LABELS[category],
      desc: CAPABILITY_CATEGORY_DESC[category],
      items: list.filter((item) => item.category === category)
    }))
    .filter((group) => group.items.length > 0)
})

const categoryMenuItems = computed(() =>
  categoryOrder()
    .map((category: CapabilityCategory) => ({
      category,
      label: CAPABILITY_CATEGORY_LABELS[category],
      count: filterCapabilities({ category, status: statusFilter.value, q: keyword.value }).length
    }))
    .filter((item) => item.count > 0)
)

const refresh = async () => {
  await load(true)
}

const statusType = (status: SubscriptionStatusLabel) => {
  if (status === '已订阅' || status === '免费可用' || status === '管理员已开通') return 'success'
  if (status === '未订阅' || status === '联系管理员开通') return 'warning'
  return 'info'
}

const primaryActionLabel = (item: ResolvedCapability) => {
  if (isEntitledStatus(item.status)) return item.open_path ? '打开' : '已可用'
  if (item.can_subscribe) return '订阅'
  if (item.status === '联系管理员开通') return '联系管理员'
  return '暂不可用'
}

const primaryActionType = (item: ResolvedCapability) => {
  if (isEntitledStatus(item.status)) return 'primary'
  if (item.can_subscribe) return 'warning'
  return 'info'
}

const primaryActionDisabled = (item: ResolvedCapability) => {
  if (isEntitledStatus(item.status)) return !item.open_path
  return item.status === '暂不可用'
}

const handlePrimary = (item: ResolvedCapability) => {
  if (isEntitledStatus(item.status)) {
    if (item.open_path) void router.push(item.open_path)
    return
  }
  if (item.can_subscribe) {
    void subscribe(item)
    return
  }
  ElMessage.info('请联系管理员开通此能力')
}

const copyCheckCommand = async (item: ResolvedCapability) => {
  if (!item.cli_topic) return
  await copyTextToClipboard(`ai-sre check ${item.cli_topic} <target>`)
  ElMessage.success('已复制命令模板')
}

const goSettings = () => {
  router.push('/app/settings')
}

watch(
  () => route.query.status,
  (value) => {
    const v = String(value || '')
    if (['all', 'entitled', 'unsubscribed', 'free'].includes(v)) {
      statusFilter.value = v as typeof statusFilter.value
    }
  },
  { immediate: true }
)

onMounted(() => {
  void load()
})
</script>

<style scoped>
.capability-center {
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.capability-hero {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  padding: 18px;
  border-radius: 18px;
  border: 1px solid var(--el-border-color-lighter);
  background:
    linear-gradient(135deg, rgba(255, 105, 0, 0.10), rgba(255, 255, 255, 0.78)),
    radial-gradient(circle at 90% 0%, rgba(64, 158, 255, 0.14), transparent 34%);
}
.eyebrow {
  margin: 0 0 6px;
  font-size: 11px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--el-color-primary);
}
.capability-hero h2 {
  margin: 0 0 8px;
  font-size: 24px;
}
.capability-hero p {
  margin: 0;
  max-width: 680px;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}
.hero-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}
.summary-row {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}
.summary-card {
  padding: 14px;
  border-radius: 14px;
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
}
.summary-card span {
  display: block;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.summary-card strong {
  display: block;
  margin-top: 4px;
  font-size: 24px;
}
.summary-card--ok strong {
  color: var(--el-color-success);
}
.summary-card--warn strong {
  color: var(--el-color-warning);
}
.category-menu {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding: 6px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 16px;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.92), rgba(248, 250, 252, 0.9));
}
.category-menu__item {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex: 0 0 auto;
  padding: 9px 12px;
  border: 1px solid transparent;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-regular);
  cursor: pointer;
}
.category-menu__item strong {
  min-width: 22px;
  padding: 1px 7px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  font-size: 12px;
}
.category-menu__item.is-active {
  border-color: var(--el-color-primary-light-5);
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}
.filter-bar {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.filter-search {
  max-width: 360px;
}
.capability-group {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.group-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 12px;
}
.group-head h3 {
  margin: 0 0 4px;
  font-size: 17px;
}
.group-head p {
  margin: 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.capability-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
}
.capability-card {
  min-height: 188px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px;
  border-radius: 14px;
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
}
.capability-card__head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 10px;
}
.capability-card__head h4 {
  margin: 0 0 6px;
  font-size: 15px;
}
.capability-card__head p {
  margin: 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}
.capability-card__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.capability-card__meta span,
.capability-card__meta code,
.command-list code {
  padding: 4px 7px;
  border-radius: 7px;
  background: var(--el-fill-color-light);
}
.command-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.command-list code {
  font-size: 12px;
  overflow-wrap: anywhere;
}
.capability-card__actions {
  margin-top: auto;
  display: flex;
  align-items: center;
  gap: 8px;
}
.evolution-note {
  padding: 16px;
  border-radius: 14px;
  border: 1px dashed var(--el-border-color);
  background: var(--el-fill-color-light);
}
.evolution-note h3 {
  margin: 0 0 6px;
}
.evolution-note p {
  margin: 0 0 10px;
  color: var(--el-text-color-secondary);
}
.evolution-note__items {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.evolution-note__items span {
  padding: 5px 8px;
  border-radius: 999px;
  background: var(--el-bg-color);
  font-size: 12px;
}
@media (max-width: 760px) {
  .capability-hero,
  .filter-bar {
    flex-direction: column;
  }
  .summary-row {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .hero-actions,
  .filter-search {
    width: 100%;
  }
}
</style>
