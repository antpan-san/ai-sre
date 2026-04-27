<template>
  <!--
    初始化工具脚本预览弹窗
    - Tab 1: Ansible 脚本（**当前唯一可直接执行**）— 下载 / 复制 / 在控制机上 bash 执行
    - Tab 2: curl 一键 — roadmap，需后端 /ft-api/api/init-tools/scripts/<name>.sh 接口
    - Tab 3: ai-sre CLI — roadmap，ai-sre node tune 子命令尚未实现
    底部「复制 / 下载」按钮跟随当前 Tab；roadmap Tab 上禁用「下载」、并标注复制内容暂不可执行。
  -->
  <el-dialog
    :model-value="modelValue"
    :title="title"
    width="820px"
    :close-on-click-modal="false"
    @update:model-value="(v: boolean) => emit('update:modelValue', v)"
  >
    <div v-if="bundle" class="script-dialog">
      <el-alert
        v-if="subtitle"
        :title="subtitle"
        type="success"
        :closable="false"
        show-icon
        class="dialog-subtitle"
      />

      <el-alert type="info" :closable="false" class="dialog-hint">
        <template #title>
          <span>
            脚本将在<strong>控制机</strong>上运行，Ansible 会自动连接所有目标节点执行操作。
            未填写节点 IP 时，Ansible 仅对<code>localhost</code>执行（即控制机本身）。
            脚本已内置 Ansible 自动安装，目标节点须允许控制机 <strong>root 免密 SSH</strong>。
            目前仅 <strong>「Ansible 执行脚本」</strong> 可直接运行；其它两个 Tab 为 roadmap 预览。
          </span>
        </template>
      </el-alert>

      <el-tabs v-model="activeTab" class="dialog-tabs">

        <!-- ── Tab 1: Ansible 执行脚本 ── -->
        <el-tab-pane label="Ansible 执行脚本" name="ansible">
          <div class="tab-actions">
            <el-tag size="small" type="success">在控制机上执行：bash {{ defaultFilename }}</el-tag>
            <el-button size="small" :icon="DocumentCopy" @click="copy(bundle.fullScript)">
              复制脚本
            </el-button>
            <el-button size="small" :icon="Download" @click="download(bundle.fullScript, defaultFilename)">
              下载 .sh
            </el-button>
          </div>
          <pre class="code-block">{{ bundle.fullScript }}</pre>
        </el-tab-pane>

        <!-- ── Tab 2: curl 一键 ── -->
        <el-tab-pane label="curl 一键（roadmap）" name="curl">
          <el-alert
            type="warning"
            :closable="false"
            show-icon
            title="尚未实现：需后端 /ft-api/api/init-tools/scripts/<name>.sh 接口（roadmap）"
            description="下面只是未来通过 ai-sre 控制台直接 curl 执行的示意；当前请使用「Ansible 执行脚本」Tab。"
            class="curl-roadmap"
          />
          <div class="tab-actions">
            <el-tag size="small" type="warning">尚未可执行 · 仅作预览</el-tag>
            <el-button size="small" :icon="DocumentCopy" @click="copy(bundle.curlOneLiner)">
              复制（roadmap）
            </el-button>
          </div>
          <pre class="code-block">{{ bundle.curlOneLiner }}</pre>
        </el-tab-pane>

        <!-- ── Tab 3: ai-sre CLI ── -->
        <el-tab-pane label="ai-sre CLI（roadmap）" name="cli">
          <el-alert
            type="warning"
            :closable="false"
            show-icon
            title="尚未实现：ai-sre 0.4.x 暂无 node tune 子命令（roadmap）"
            :description="cliRoadmapHint"
            class="cli-roadmap"
          />
          <div class="tab-actions">
            <el-tag size="small" type="warning">尚未可执行 · 仅作预览</el-tag>
            <el-button size="small" :icon="DocumentCopy" @click="copy(bundle.aiSreCommand)">
              复制（roadmap）
            </el-button>
          </div>
          <pre class="code-block">{{ bundle.aiSreCommand }}</pre>
        </el-tab-pane>
      </el-tabs>
    </div>

    <template #footer>
      <el-button @click="emit('update:modelValue', false)">关闭</el-button>
      <el-button
        v-if="bundle"
        type="primary"
        :icon="Download"
        :disabled="!activePayload.executable"
        @click="download(activePayload.text, activePayload.filename)"
      >
        {{ activePayload.executable ? '下载脚本' : '下载（仅 Ansible Tab 可用）' }}
      </el-button>
      <el-button
        v-if="bundle"
        type="primary"
        :icon="DocumentCopy"
        @click="copy(activePayload.text)"
      >
        {{ activePayload.executable ? '复制脚本' : '复制（roadmap）' }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { DocumentCopy, Download } from '@element-plus/icons-vue'
import type { ScriptBundle } from '../../views/init-tools/scripts'

const props = defineProps<{
  modelValue: boolean
  title: string
  bundle: ScriptBundle | null
  defaultFilename?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
}>()

type TabKey = 'ansible' | 'curl' | 'cli'
const activeTab = ref<TabKey>('ansible')
const subtitle = computed(() => props.bundle?.subtitle || '')
const defaultFilename = computed(() => props.defaultFilename || 'init.sh')
const cliRoadmapHint =
  '复制下面命令直接在节点运行会得到 unknown command "node" for "ai-sre" 错误。' +
  '当前请使用「Ansible 执行脚本」Tab；ai-sre 自动升级（OPSFLEET_NO_AUTO_UPGRADE 未设置时）会拉取最新版本，但 node tune 在新版本实现前都不可用。'

interface ActivePayload {
  text: string
  filename: string
  executable: boolean
}

const activePayload = computed<ActivePayload>(() => {
  const b = props.bundle
  if (!b) return { text: '', filename: defaultFilename.value, executable: false }
  switch (activeTab.value) {
    case 'curl':
      return {
        text: b.curlOneLiner,
        filename: defaultFilename.value.replace(/\.sh$/, '') + '-curl.txt',
        executable: false,
      }
    case 'cli':
      return {
        text: b.aiSreCommand,
        filename: defaultFilename.value.replace(/\.sh$/, '') + '-ai-sre.txt',
        executable: false,
      }
    case 'ansible':
    default:
      return {
        text: b.fullScript,
        filename: defaultFilename.value,
        executable: true,
      }
  }
})

watch(
  () => props.modelValue,
  (v) => { if (v) activeTab.value = 'ansible' },
)

const copy = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制到剪贴板')
  } catch {
    const ta = document.createElement('textarea')
    ta.value = text
    ta.style.position = 'fixed'
    ta.style.left = '-9999px'
    document.body.appendChild(ta)
    ta.select()
    try {
      document.execCommand('copy')
      ElMessage.success('已复制到剪贴板')
    } catch {
      ElMessage.error('复制失败，请手动选中后复制')
    } finally {
      document.body.removeChild(ta)
    }
  }
}

const download = (text: string, filename: string) => {
  const blob = new Blob([text], { type: 'text/x-shellscript;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
  ElMessage.success(`已下载: ${filename}`)
}
</script>

<style scoped>
.script-dialog {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.dialog-subtitle :deep(.el-alert__title),
.dialog-hint :deep(.el-alert__title) {
  font-size: 13px;
  line-height: 1.6;
}

.dialog-hint code {
  background: #f1f5f9;
  padding: 0 4px;
  border-radius: 3px;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  font-size: 12px;
}

.dialog-hint strong {
  color: #1e40af;
}

.dialog-tabs {
  margin-top: 4px;
}

.tab-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  flex-wrap: wrap;
}

.code-block {
  margin: 0;
  padding: 12px 14px;
  background: #0f172a;
  color: #e2e8f0;
  border-radius: 8px;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  font-size: 12.5px;
  line-height: 1.55;
  max-height: 480px;
  overflow: auto;
  white-space: pre;
}

.curl-roadmap,
.cli-roadmap {
  margin-bottom: 8px;
}
</style>
