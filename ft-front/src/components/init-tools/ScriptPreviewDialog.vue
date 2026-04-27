<template>
  <!--
    Ansible 执行脚本预览弹窗
    - Tab 1: Ansible 脚本（主 tab）— 下载 / 复制 / 在控制机上 bash 执行
    - Tab 2: curl 一键 — 未来通过 ai-sre 控制台接口获取
    - Tab 3: ai-sre CLI — 未来 CLI 等价命令（roadmap）
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
        <el-tab-pane label="curl 一键" name="curl">
          <el-alert
            type="warning"
            :closable="false"
            show-icon
            title="需后端 /ft-api/api/init-tools/scripts/<name>.sh 接口（roadmap）"
            description="下面展示了通过 ai-sre 控制台直接 curl 执行的方式，无需手动下载脚本。当前请使用「Ansible 执行脚本」Tab。"
            class="curl-roadmap"
          />
          <div class="tab-actions">
            <el-button size="small" :icon="DocumentCopy" @click="copy(bundle.curlOneLiner)">
              复制命令
            </el-button>
          </div>
          <pre class="code-block">{{ bundle.curlOneLiner }}</pre>
        </el-tab-pane>

        <!-- ── Tab 3: ai-sre CLI ── -->
        <el-tab-pane label="ai-sre CLI" name="cli">
          <el-alert
            type="warning"
            :closable="false"
            show-icon
            title="ai-sre node tune 子命令规划中（roadmap）"
            description="下面命令展示了未来 ai-sre CLI 的等价调用方式，参数与脚本一致。"
            class="cli-roadmap"
          />
          <div class="tab-actions">
            <el-button size="small" :icon="DocumentCopy" @click="copy(bundle.aiSreCommand)">
              复制命令
            </el-button>
          </div>
          <pre class="code-block">{{ bundle.aiSreCommand }}</pre>
        </el-tab-pane>
      </el-tabs>
    </div>

    <template #footer>
      <el-button @click="emit('update:modelValue', false)">关闭</el-button>
      <el-button v-if="bundle" type="primary" :icon="Download" @click="download(bundle.fullScript, defaultFilename)">
        下载脚本
      </el-button>
      <el-button v-if="bundle" type="primary" :icon="DocumentCopy" @click="copy(bundle.fullScript)">
        复制脚本
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

const activeTab = ref<'ansible' | 'curl' | 'cli'>('ansible')
const subtitle = computed(() => props.bundle?.subtitle || '')
const defaultFilename = computed(() => props.defaultFilename || 'init.sh')

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
