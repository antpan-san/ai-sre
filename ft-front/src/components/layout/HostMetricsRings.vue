<template>
  <div
    v-if="visible"
    class="host-metrics-rings"
    :class="{ 'host-metrics-rings--loading': dashboardStore.loading && !hasSample }"
    role="group"
    aria-label="服务端资源占用"
  >
    <el-tooltip v-for="ring in rings" :key="ring.key" placement="bottom" :show-after="200">
      <template #content>
        <div>{{ ring.tipTitle }}</div>
        <div v-if="hostTip">{{ hostTip }}</div>
        <div v-if="collectError" class="host-metrics-rings__err">{{ collectError }}</div>
      </template>
      <div class="host-metric-ring">
        <el-progress
          type="circle"
          :percentage="ring.pct"
          :color="ring.color"
          :width="44"
          :stroke-width="5"
          :format="() => ring.labelShort"
        />
        <span class="host-metric-ring__caption">{{ ring.caption }}</span>
      </div>
    </el-tooltip>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useDashboardStore } from '../../stores/dashboard'
import { clampPct, usageRingColor } from '../../utils/hostMetrics'

const props = withDefaults(
  defineProps<{
    enabled?: boolean
  }>(),
  { enabled: true }
)

const dashboardStore = useDashboardStore()
const visible = computed(() => props.enabled)
const dash = computed(() => dashboardStore.dashboardData)

const hasSample = computed(() => {
  const u = dash.value?.resourceUsage
  return u != null && (u.cpu > 0 || u.memory > 0 || u.disk > 0 || !!dash.value?.hostRuntime)
})

const collectError = computed(() => dash.value?.hostRuntime?.error?.trim() || '')

const hostTip = computed(() => {
  const h = dash.value?.hostRuntime
  if (!h?.hostname) return '本机（运行 opsfleet-backend）'
  const bits = [h.hostname]
  if (h.sampledAt) {
    const d = new Date(h.sampledAt)
    if (!Number.isNaN(d.getTime())) bits.push(d.toLocaleString())
  }
  if (h.os) bits.push(h.os)
  return bits.join(' · ')
})

function ringPct(v: number | undefined): number {
  return Math.round(clampPct(Number(v ?? 0)))
}

const rings = computed(() => {
  const u = dash.value?.resourceUsage
  const cpu = ringPct(u?.cpu)
  const mem = ringPct(u?.memory)
  const disk = ringPct(u?.disk)
  return [
    {
      key: 'cpu',
      caption: 'CPU',
      labelShort: `${cpu}%`,
      pct: cpu,
      color: usageRingColor(cpu),
      tipTitle: `服务端 CPU ${cpu}%`
    },
    {
      key: 'memory',
      caption: '内存',
      labelShort: `${mem}%`,
      pct: mem,
      color: usageRingColor(mem),
      tipTitle: `服务端内存 ${mem}%`
    },
    {
      key: 'disk',
      caption: '磁盘',
      labelShort: `${disk}%`,
      pct: disk,
      color: usageRingColor(disk),
      tipTitle: `服务端磁盘（根分区）${disk}%`
    }
  ]
})

let pollTimer: ReturnType<typeof setInterval> | null = null

const refresh = () => {
  if (!visible.value) return
  void dashboardStore.fetchDashboardData()
}

onMounted(() => {
  if (!visible.value) return
  void refresh()
  pollTimer = setInterval(refresh, 45_000)
})

onUnmounted(() => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
})
</script>

<style scoped>
.host-metrics-rings {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 14px;
  flex-shrink: 0;
  padding: 0 8px;
}

.host-metrics-rings--loading {
  opacity: 0.72;
}

.host-metric-ring {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  cursor: default;
}

.host-metric-ring__caption {
  font-size: 11px;
  line-height: 1.2;
  color: var(--el-text-color-secondary);
  font-weight: 500;
}

.host-metrics-rings__err {
  margin-top: 4px;
  color: var(--el-color-danger);
  max-width: 280px;
  word-break: break-word;
}
</style>
