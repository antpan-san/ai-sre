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
        <div v-if="hostLine" class="host-rings-tip host-rings-tip--sub">{{ hostLine }}</div>
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
              :stroke-dasharray="arcDash(m.value)"
            />
          </svg>
          <span class="host-ring__num">{{ Math.round(m.value) }}</span>
        </div>
        <span class="host-ring__lbl">{{ m.short }}</span>
      </div>
    </el-tooltip>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, shallowRef } from 'vue'
import { getDashboardHostResources } from '../../api/dashboard'
import type { HostRuntimeMeta, ResourceUsage } from '../../types/dashboard'

const POLL_MS = 5000
const RING_R = 12
const RING_C = 2 * Math.PI * RING_R

const loading = ref(false)
const resourceUsage = shallowRef<ResourceUsage | null>(null)
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

const hostLine = computed(() => {
  if (!visible.value) return ''
  const h = hostRuntime.value
  if (!h?.hostname) return ''
  const bits = [h.hostname]
  if (h.os) bits.push(h.os)
  return bits.join(' · ')
})

const hostError = computed(() => {
  if (!visible.value) return ''
  return hostRuntime.value?.error || ''
})

const meters = computed(() => {
  if (!visible.value) return []
  const u = resourceUsage.value
  const items = [
    { key: 'cpu', short: 'CPU', label: 'CPU', value: u?.cpu ?? 0 },
    { key: 'mem', short: '内存', label: '内存', value: u?.memory ?? 0 },
    { key: 'disk', short: '磁盘', label: '磁盘（/）', value: u?.disk ?? 0 }
  ]
  return items.map((m) => ({
    ...m,
    level: levelOf(m.value),
    tip: `${m.label} ${m.value.toFixed(1)}%`
  }))
})

const refreshHostResources = async () => {
  if (!visible.value || inFlight) return
  inFlight = true
  if (!resourceUsage.value) loading.value = true
  try {
    const data = await getDashboardHostResources()
    resourceUsage.value = data.resourceUsage ?? null
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
  gap: 10px;
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

.host-rings-tip--sub {
  margin-top: 4px;
  opacity: 0.85;
  font-size: 11px;
}

.host-rings-tip--err {
  margin-top: 4px;
  color: var(--el-color-danger);
  font-size: 11px;
}
</style>
