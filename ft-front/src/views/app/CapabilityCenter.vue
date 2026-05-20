<template>
  <div class="capability-center page-shell">
    <header class="page-head">
      <div class="page-head-copy">
        <h2 class="page-title">能力中心</h2>
        <p class="page-desc--muted">浏览平台全部能力、订阅状态与入口。未订阅能力可查看说明并发起订阅。</p>
      </div>
      <el-button size="small" :loading="loading" @click="load">刷新</el-button>
    </header>

    <section v-loading="loading" class="cap-sections">
      <div v-for="cat in categoryOrder()" :key="cat" class="cap-section">
        <h3 class="cap-section__title">{{ categoryLabels[cat] }}</h3>
        <div class="cap-grid">
          <CapabilityCard
            v-for="item in byCategory.get(cat) || []"
            :key="item.id"
            :item="item"
            @open="openItem"
            @subscribe="subscribe"
          />
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import CapabilityCard from '../../components/app/CapabilityCard.vue'
import { useCapabilityCatalog, type ResolvedCapability } from '../../composables/useCapabilityCatalog'

const router = useRouter()
const { loading, byCategory, categoryLabels, categoryOrder, load, subscribe } = useCapabilityCatalog()

const openItem = (item: ResolvedCapability) => {
  if (!item.can_open || !item.open_path) return
  const [path, queryStr] = item.open_path.split('?')
  const query: Record<string, string> = {}
  if (queryStr) {
    for (const part of queryStr.split('&')) {
      const [k, v] = part.split('=')
      if (k) query[k] = decodeURIComponent(v || '')
    }
  }
  router.push({ path, query: Object.keys(query).length ? query : undefined })
}
</script>

<style scoped>
.cap-sections {
  display: flex;
  flex-direction: column;
  gap: 28px;
}
.cap-section__title {
  margin: 0 0 12px;
  font-size: 16px;
  font-weight: 600;
}
.cap-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 14px;
}
</style>
