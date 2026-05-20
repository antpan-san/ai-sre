<template>
  <div class="capability-center page-shell page-shell--crud-wide">
    <AppPageHeader
      title="能力中心"
      description="浏览平台全部能力、订阅状态与入口。未订阅能力可查看说明并发起订阅。"
    >
      <template #actions>
        <el-button size="small" link type="primary" @click="router.push('/app/workloads')">工作负载</el-button>
        <el-button size="small" :loading="loading" @click="load(true)">刷新</el-button>
      </template>
    </AppPageHeader>

    <section v-loading="loading" class="app-stats-row">
      <article class="app-stat-tile">
        <span class="app-stat-label">已开通</span>
        <strong>{{ summary.entitled }}</strong>
      </article>
      <article class="app-stat-tile">
        <span class="app-stat-label">可订阅</span>
        <strong>{{ summary.subscribeable }}</strong>
      </article>
      <article class="app-stat-tile app-stat-tile--ok">
        <span class="app-stat-label">免费可用</span>
        <strong>{{ summary.free }}</strong>
      </article>
      <article class="app-stat-tile">
        <span class="app-stat-label">能力总数</span>
        <strong>{{ summary.total }}</strong>
      </article>
    </section>

    <section v-if="packsWithCapabilities.length" id="packs" class="cap-packs">
      <h3 class="section-title">我的功能包</h3>
      <div class="app-pack-row">
        <PackCard
          v-for="pack in packsWithCapabilities"
          :key="pack.pack_key"
          :pack="pack"
          @manage="scrollToPack"
          @subscribe="subscribePack"
        />
      </div>
    </section>

    <div class="app-toolbar">
      <el-input v-model="searchQ" size="small" clearable placeholder="搜索能力名称、关键词…" style="width: 220px" />
      <el-select v-model="statusFilter" size="small" style="width: 120px">
        <el-option label="全部状态" value="all" />
        <el-option label="已开通" value="entitled" />
        <el-option label="未订阅" value="unsubscribed" />
        <el-option label="免费" value="free" />
      </el-select>
      <div class="app-category-pills">
        <el-button
          v-for="cat in visibleCategories"
          :key="cat"
          size="small"
          :type="activeCategory === cat ? 'primary' : 'default'"
          round
          @click="scrollToCategory(cat)"
        >
          {{ categoryLabels[cat] }}
        </el-button>
      </div>
    </div>

    <section v-loading="loading" class="cap-sections">
      <div
        v-for="cat in visibleCategories"
        :id="`cap-${cat}`"
        :key="cat"
        class="cap-section"
      >
        <h3 class="section-title">{{ categoryLabels[cat] }}</h3>
        <div class="app-cap-grid">
          <CapabilityCard
            v-for="item in filteredByCategory(cat)"
            :key="item.id"
            :item="item"
            @open="openItem"
            @subscribe="subscribeItem"
          />
        </div>
      </div>
      <el-empty v-if="!filteredAll.length && !loading" description="没有匹配的能力" />
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppPageHeader from '../../components/app/AppPageHeader.vue'
import CapabilityCard from '../../components/app/CapabilityCard.vue'
import PackCard from '../../components/app/PackCard.vue'
import '../../assets/app-workbench.css'
import {
  useCapabilityCatalog,
  type PackWithCapabilities,
  type ResolvedCapability,
  type StatusFilter
} from '../../composables/useCapabilityCatalog'
import type { CapabilityCategory } from '../../config/capabilityCatalog'
import { openCapability } from '../../utils/capabilityNavigation'

const router = useRouter()
const route = useRoute()
const {
  loading,
  summary,
  packsWithCapabilities,
  categoryLabels,
  categoryOrder,
  load,
  subscribe,
  filterCapabilities
} = useCapabilityCatalog()

const searchQ = ref('')
const statusFilter = ref<StatusFilter>('all')
const activeCategory = ref<CapabilityCategory | 'all'>('all')

const filteredAll = computed(() =>
  filterCapabilities({ q: searchQ.value, status: statusFilter.value, category: 'all' })
)

const visibleCategories = computed(() => {
  const cats = categoryOrder().filter((cat) => {
    if (cat === 'evolution') {
      return filteredByCategory(cat).length > 0
    }
    return filteredByCategory(cat).length > 0
  })
  return cats
})

const filteredByCategory = (cat: CapabilityCategory) => {
  return filterCapabilities({ q: searchQ.value, status: statusFilter.value, category: cat })
}

const openItem = (item: ResolvedCapability) => openCapability(router, item)

const subscribeItem = (item: ResolvedCapability) => {
  void subscribe(item)
}

const subscribePack = (pack: PackWithCapabilities) => {
  void subscribe({ pack_key: pack.pack_key, can_subscribe: pack.can_subscribe })
}

const scrollToCategory = async (cat: CapabilityCategory) => {
  activeCategory.value = cat
  await nextTick()
  document.getElementById(`cap-${cat}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

const scrollToPack = (pack: PackWithCapabilities) => {
  const first = pack.capabilities[0]
  if (first) void scrollToCategory(first.category)
}

watch(
  () => route.hash,
  (hash) => {
    if (hash === '#packs') {
      nextTick(() => document.getElementById('packs')?.scrollIntoView({ behavior: 'smooth' }))
    }
  },
  { immediate: true }
)

onMounted(() => {
  void load()
})
</script>

<style scoped>
.section-title {
  margin: 0 0 12px;
  font-size: 16px;
  font-weight: 600;
}
.cap-packs {
  margin-bottom: 20px;
}
.cap-sections {
  display: flex;
  flex-direction: column;
  gap: 28px;
}
</style>
