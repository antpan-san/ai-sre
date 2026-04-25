<template>
  <!--
    安装脚本预览弹窗
    - 三个 Tab：Bash 脚本 / ai-sre CLI / 多节点批量
    - 每个 Tab 都有「复制」按钮
    - 顶部展示目标节点 / 系统 / 操作摘要 (subtitle)
    - 强调脚本本身已含「已存在则跳过」的探测逻辑（ON_CONFLICT）
  -->
  <el-dialog
    :model-value="modelValue"
    :title="title"
    width="780px"
    :close-on-click-modal="false"
    @update:model-value="(v) => emit('update:modelValue', v)"
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

      <el-alert
        type="info"
        :closable="false"
        class="dialog-hint"
      >
        <template #title>
          <span>
            脚本已内置探测逻辑：检测到节点上已存在同类配置（如 chrony / sshd_config.d / sysctl.d/99-ai-sre.conf 等）时
            <el-tag size="small" type="warning" effect="plain">默认 ON_CONFLICT=skip</el-tag>
            直接退出并打印当前状态，不会覆盖。需要强制覆盖请设置 <code>ON_CONFLICT=force</code>。
          </span>
        </template>
      </el-alert>

      <el-tabs v-model="activeTab" class="dialog-tabs">
        <el-tab-pane label="Bash 脚本" name="bash">
          <div class="tab-actions">
            <el-tag size="small" type="info">直接 SSH 到目标节点 bash -s 执行</el-tag>
            <el-button size="small" :icon="DocumentCopy" @click="copy(bundle.fullScript)">
              复制脚本
            </el-button>
            <el-button size="small" :icon="Download" @click="download(bundle.fullScript, defaultFilename)">
              下载 .sh
            </el-button>
          </div>
          <pre class="code-block">{{ bundle.fullScript }}</pre>
        </el-tab-pane>

        <el-tab-pane label="ai-sre CLI" name="cli">
          <el-alert
            type="warning"
            :closable="false"
            show-icon
            title="ai-sre node tune 子命令规划中（roadmap）"
            description="下面命令展示了未来 ai-sre 客户端工具的等价调用方式，参数与脚本一致。当前可使用 Bash 脚本 / 多节点批量来落地。"
            class="cli-roadmap"
          />
          <div class="tab-actions">
            <el-button size="small" :icon="DocumentCopy" @click="copy(bundle.aiSreCommand)">
              复制命令
            </el-button>
          </div>
          <pre class="code-block">{{ bundle.aiSreCommand }}</pre>
        </el-tab-pane>

        <el-tab-pane label="多节点批量" name="batch">
          <div class="tab-actions">
            <el-tag size="small" type="info">前置：先把上方 Bash 脚本保存为 .sh，再运行此循环</el-tag>
            <el-button size="small" :icon="DocumentCopy" @click="copy(bundle.batchOneLiner)">
              复制循环命令
            </el-button>
          </div>
          <pre class="code-block">{{ bundle.batchOneLiner }}</pre>

          <el-divider content-position="left">curl 一键执行（roadmap）</el-divider>
          <div class="tab-actions">
            <el-tag size="small" type="warning">需后端 /ft-api/api/init-tools/scripts/&lt;name&gt;.sh 接口配合</el-tag>
            <el-button size="small" :icon="DocumentCopy" @click="copy(bundle.curlOneLiner)">
              复制 curl 命令
            </el-button>
          </div>
          <pre class="code-block">{{ bundle.curlOneLiner }}</pre>
        </el-tab-pane>
      </el-tabs>
    </div>

    <template #footer>
      <el-button @click="emit('update:modelValue', false)">关闭</el-button>
      <el-button v-if="bundle" type="primary" :icon="DocumentCopy" @click="copy(bundle.fullScript)">
        复制 Bash 脚本
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

const activeTab = ref<'bash' | 'cli' | 'batch'>('bash')
const subtitle = computed(() => props.bundle?.subtitle || '')
const defaultFilename = computed(() => props.defaultFilename || 'init.sh')

watch(
  () => props.modelValue,
  (v) => {
    if (v) activeTab.value = 'bash'
  },
)

const copy = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制到剪贴板')
  } catch {
    // 兼容不支持 clipboard 的环境（http、firefox 旧版等）
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
  max-height: 460px;
  overflow: auto;
  white-space: pre;
}

.cli-roadmap {
  margin-bottom: 8px;
}
</style>
