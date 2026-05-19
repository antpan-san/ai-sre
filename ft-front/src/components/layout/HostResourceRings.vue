<template>
  <div
    v-if="visible"
    class="host-rings"
    role="group"
    aria-label="服务端资源"
    :class="{ 'host-rings--loading': loading && !resourceUsage }"
  >
    <el-tooltip v-for="m in meters" :key="m.key" placement="bottom" :show-after="200">
      <template #content>
        <div class="host-rings-tip">{{ m.tip }}</div>
        <div v-if="hostError" class="host-rings-tip host-rings-tip--err">{{ hostError }}</div>
      </template>
      <div class="host-ring" :class="`host-ring--${m.level}`">
        <div class="host-ring__viz">
          <svg class="host-ring__svg" viewBox="0 0 32 32" aria-hidden="true">
            <circle class="host-ring__track" cx="16" cy="16" :r="RING_R" />
            <circle
              class="host-ring__arc"
              cx="16"
              cy="16"
              :r="RING_R"
              :stroke-dasharray="arcDash(m.arcValue)"
            />
          </svg>
          <span class="host-ring__num">{{ m.display }}</span>
        </div>
        <span class="host-ring__lbl">{{ m.short }}</span>
      </div>
    </el-tooltip>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, shallowRef } from 'vue'
import { getDashboardHostResources } from '../../api/dashboard'
import type { HostResourceDetail, HostRuntimeMeta, ResourceUsage } from '../../types/dashboard'

const POLL_MS = 5000
const RING_R = 12
const RING_C = 2 * Math.PI * RING_R

const loading = ref(false)
const resourceUsage = shallowRef<ResourceUsage | null>(null)
const resourceDetail = shallowRef<HostResourceDetail | null>(null)
const hostRuntime = shallowRef<HostRuntimeMeta | null>(null)
let pollTimer: ReturnType<typeof setInterval> | null = null
let inFlight = false

const visible = computed(() => {
  try {
    const role = String(
      (JSON.parse(localStorage.getItem('userInfo') || '{}') as { role?: string }).role ?? ''
    )
    return role === 'super_admin'
  } catch {
    return false
  }
})

const arcDash = (pct: number) => {
  const p = Math.min(100, Math.max(0, pct))
  const filled = (p / 100) * RING_C
  return `${filled} ${RING_C}`
}

const levelOf = (pct: number) => {
  if (pct >= 80) return 'danger'
  if (pct >= 60) return 'warn'
  return 'ok'
}

const formatBytes = (bytes: number) => {
  if (!Number.isFinite(bytes) || bytes < 0) return '?'
  const gb = bytes / 1024 ** 3
  if (gb >= 1) return `${gb.toFixed(1)}G`
  const mb = bytes / 1024 ** 2
  if (mb >= 1) return `${mb.toFixed(0)}M`
  return `${Math.max(0, Math.round(bytes / 1024))}K`
}

const formatLoad = (v: number) => {
  if (!Number.isFinite(v)) return '0'
  if (v < 10) return v.toFixed(1)
  return String(Math.round(v))
}

const hostError = computed(() => {
  if (!visible.value) return ''
  return hostRuntime.value?.error || ''
})

const meters = computed(() => {
  if (!visible.value) return []
  const u = resourceUsage.value
  const d = resourceDetail.value
  const cores = d?.cpuCores ?? 1
  const load1 = u?.load ?? 0
  const loadArc = Math.min(100, (load1 / Math.max(cores, 1)) * 100)

  const items = [
    {
      key: 'cpu',
      short: 'CPU',
      display: String(Math.round(u?.cpu ?? 0)),
      arcValue: u?.cpu ?? 0,
      tip: `CPU ${(u?.cpu ?? 0).toFixed(0)}% / 100%`
    },
    {
      key: 'load',
      short: '负载',
      display: formatLoad(load1),
      arcValue: loadArc,
      tip: `负载 ${formatLoad(load1)} / ${cores}`
    },
    {
      key: 'mem',
      short: '内存',
      display: String(Math.round(u?.memory ?? 0)),
      arcValue: u?.memory ?? 0,
      tip: `内存 ${formatBytes(d?.memUsedBytes ?? 0)} / ${formatBytes(d?.memTotalBytes ?? 0)}`
    },
    {
      key: 'disk',
      short: '磁盘',
      display: String(Math.round(u?.disk ?? 0)),
      arcValue: u?.disk ?? 0,
      tip: `磁盘 ${formatBytes(d?.diskUsedBytes ?? 0)} / ${formatBytes(d?.diskTotalBytes ?? 0)}`
    },
    {
      key: 'diskio',
      short: 'IO',
      display: String(Math.round(u?.diskIo ?? 0)),
      arcValue: u?.diskIo ?? 0,
      tip: `IO ${(u?.diskIo ?? 0).toFixed(0)}% / 100%`
    }
  ]

  return items.map((m) => ({
    ...m,
    level: levelOf(m.arcValue)
  }))
})

const refreshHostResources = async () => {
  if (!visible.value || inFlight) return
  inFlight = true
  if (!resourceUsage.value) loading.value = true
  try {
    const data = await getDashboardHostResources()
    resourceUsage.value = data.resourceUsage ?? null
    resourceDetail.value = data.resourceDetail ?? null
    hostRuntime.value = data.hostRuntime ?? null
  } catch {
    // 静默轮询：不打断页面、不弹 toast
  } finally {
    loading.value = false
    inFlight = false
  }
}

const startPolling = () => {
  if (!visible.value || pollTimer) return
  void refreshHostResources()
  pollTimer = setInterval(() => {
    void refreshHostResources()
  }, POLL_MS)
}

const stopPolling = () => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

onMounted(() => {
  if (visible.value) startPolling()
})

onUnmounted(() => {
  stopPolling()
})
</script>

<style scoped>
.host-rings {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 0 4px;
}

.host-rings--loading {
  opacity: 0.55;
}

.host-ring {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1px;
  cursor: default;
}

.host-ring__viz {
  position: relative;
  width: 34px;
  height: 34px;
}

.host-ring__svg {
  width: 34px;
  height: 34px;
  transform: rotate(-90deg);
}

.host-ring__track {
  fill: none;
  stroke: var(--el-border-color-lighter);
  stroke-width: 2.5;
}

.host-ring__arc {
  fill: none;
  stroke-width: 2.5;
  stroke-linecap: round;
  transition: stroke-dasharray 0.35s ease;
}

.host-ring--ok .host-ring__arc {
  stroke: #67c23a;
}

.host-ring--warn .host-ring__arc {
  stroke: #e6a23c;
}

.host-ring--danger .host-ring__arc {
  stroke: #f56c6c;
}

.host-ring__num {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  font-weight: 700;
  line-height: 1;
  color: var(--el-text-color-primary);
}

.host-ring__lbl {
  font-size: 9px;
  line-height: 1;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}

.host-rings-tip {
  font-size: 12px;
  line-height: 1.4;
}

.host-rings-tip--err {
  margin-top: 4px;
  color: var(--el-color-danger);
  font-size: 11px;
}
</style>
