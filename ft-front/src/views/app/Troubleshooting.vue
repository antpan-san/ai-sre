<template>
  <div class="troubleshooting page-shell page-shell--crud-wide">
    <header class="page-head">
      <div class="page-head-copy">
        <h2 class="page-title">问题排查</h2>
        <p class="page-desc--muted">以 <code>ai-sre check &lt;topic&gt; [target]</code> 为核心；先采集证据，本地规则优先，AI 兜底。</p>
      </div>
      <el-button size="small" link type="primary" @click="goSettings">CLI 安装</el-button>
    </header>

    <div class="topic-grid">
      <article v-for="topic in topics" :key="topic.cli_topic" class="topic-card">
        <header class="topic-card__head">
          <h3>{{ topic.name }}</h3>
          <el-tag size="small" :type="statusType(topic)">{{ topicStatus(topic) }}</el-tag>
        </header>
        <p class="topic-card__desc">{{ topic.description }}</p>
        <div class="topic-card__cmd">
          <code>{{ commandFor(topic) }}</code>
          <el-button size="small" link type="primary" @click="copyCmd(topic)">复制</el-button>
        </div>
        <el-input v-model="targets[topic.cli_topic!]" size="small" placeholder="目标（可选，如 host:6379）" class="topic-target" />
      </article>
    </div>

    <el-collapse class="advanced-collapse">
      <el-collapse-item title="高级用法（probe / ask / runbook）" name="advanced">
        <p class="page-desc--muted">日常排查推荐使用 check。高级场景可在 CLI 中使用：</p>
        <pre class="cmd-line">ai-sre probe &lt;topic&gt; [target] --json</pre>
        <pre class="cmd-line">ai-sre ask "问题描述" --topic &lt;topic&gt;</pre>
        <pre class="cmd-line">ai-sre runbook &lt;topic&gt;</pre>
      </el-collapse-item>
    </el-collapse>
  </div>
</template>

<script setup lang="ts">
import { reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { TROUBLESHOOT_TOPICS } from '../../config/capabilityCatalog'
import { useCapabilityCatalog } from '../../composables/useCapabilityCatalog'
import { copyTextToClipboard } from '../../utils/clipboard'

const router = useRouter()
const { resolved, shellPrefix } = useCapabilityCatalog()
const targets = reactive<Record<string, string>>({})

const topics = TROUBLESHOOT_TOPICS

const topicCap = (cliTopic: string) => resolved.value.find((c) => c.cli_topic === cliTopic)

const topicStatus = (topic: (typeof topics)[0]) => topicCap(topic.cli_topic || '')?.status || '—'

const statusType = (topic: (typeof topics)[0]) => {
  const s = topicStatus(topic)
  if (s === '已订阅' || s === '免费可用' || s === '管理员已开通') return 'success'
  if (s === '未订阅') return 'warning'
  return 'info'
}

const commandFor = (topic: (typeof topics)[0]) => {
  const t = topic.cli_topic || 'topic'
  const target = (targets[t] || '').trim()
  return target ? `ai-sre check ${t} ${target}` : `ai-sre check ${t}`
}

const copyCmd = async (topic: (typeof topics)[0]) => {
  await copyTextToClipboard(commandFor(topic))
  ElMessage.success('已复制命令')
}

const goSettings = () => {
  router.push(`${shellPrefix.value}/settings`)
}
</script>

<style scoped>
.topic-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 14px;
  margin-bottom: 20px;
}
.topic-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  padding: 14px;
  background: var(--layout-content-surface);
}
.topic-card__head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 8px;
  margin-bottom: 8px;
}
.topic-card__head h3 {
  margin: 0;
  font-size: 15px;
}
.topic-card__desc {
  margin: 0 0 10px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.topic-card__cmd {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
  margin-bottom: 8px;
}
.topic-card__cmd code {
  font-size: 12px;
  word-break: break-all;
}
.topic-target {
  width: 100%;
}
.advanced-collapse {
  max-width: 640px;
}
.cmd-line {
  margin: 0 0 8px;
  padding: 8px 10px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
  font-size: 12px;
}
</style>
