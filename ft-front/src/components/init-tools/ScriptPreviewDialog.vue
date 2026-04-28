<template>
  <!--
    初始化工具脚本预览弹窗
    - Tab 1: Ansible 脚本（始终可直接执行）— 下载 / 复制 / 在控制机上 bash 执行
    - Tab 2: curl 一键 — roadmap，需后端 /ft-api/api/init-tools/scripts/<name>.sh 接口
    - Tab 3: ai-sre CLI — 是否可执行由 bundle.aiSreCommandExecutable 决定：
        time-sync / sys-param 已在 ai-sre 中实现（>=0.4.7），可直接 sudo bash 执行；
        security / disk 仍是 roadmap 占位。
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
            <strong>「Ansible 执行脚本」</strong> 始终可执行；
            <strong>「ai-sre CLI」</strong>当此卡片在 ai-sre 中已实现时（time-sync / sys-param ≥ 0.4.7）也可 <code>sudo bash</code> 直接运行，
            其它情况（curl 一键 / security / disk）仍是 roadmap 预览。
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
        <el-tab-pane :label="cliTabLabel" name="cli">
          <el-alert
            v-if="cliExecutable"
            type="success"
            :closable="false"
            show-icon
            title="ai-sre CLI 已实现（>=0.4.7）"
            :description="cliReadyHint"
            class="cli-roadmap"
          />
          <el-alert
            v-else
            type="warning"
            :closable="false"
            show-icon
            title="尚未实现：当前卡片对应的 ai-sre node tune 子命令仍是 roadmap"
            :description="cliRoadmapHint"
            class="cli-roadmap"
          />
          <div class="tab-actions">
            <el-tag size="small" :type="cliExecutable ? 'success' : 'warning'">
              {{ cliExecutable ? '在控制机上执行：sudo bash 一行复制即可' : '尚未可执行 · 仅作预览' }}
            </el-tag>
            <el-button size="small" :icon="DocumentCopy" @click="copy(bundle.aiSreCommand)">
              {{ cliExecutable ? '复制命令' : '复制（roadmap）' }}
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
        {{ activePayload.executable ? '下载脚本' : '下载（roadmap Tab 不可用）' }}
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
import { prepareExecutionRecord } from '../../api/execution-records'

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
const cliExecutable = computed(() => Boolean(props.bundle?.aiSreCommandExecutable))
const cliTabLabel = computed(() => (cliExecutable.value ? 'ai-sre CLI' : 'ai-sre CLI（roadmap）'))
const cliRoadmapHint =
  '复制下面命令直接在节点运行会得到 unknown command 错误。' +
  '当前请使用「Ansible 执行脚本」Tab；ai-sre 自动升级（OPSFLEET_NO_AUTO_UPGRADE 未设置时）会拉取最新版本，但本卡片对应子命令在新版本实现前不可用。'
const cliReadyHint =
  'ai-sre 0.4.7 起已内置 time-sync / sys-param 并修正 playbook YAML 解析与「skip 误跳时区」问题；time-sync 只对 --clients 列表的节点动手，不会自动包括跑 ai-sre 的本机；执行前还会跑 ansible-playbook --syntax-check。低版本节点上首次执行命令会自动升级到最新版（OPSFLEET_NO_AUTO_UPGRADE 未设置时）。' +
  '本机用 sudo 执行；未填节点 IP 时仅对控制机 localhost 执行；填节点 IP 时控制机须能 root 免密 SSH 到目标节点。'

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
        filename: defaultFilename.value.replace(/\.sh$/, '') + '-ai-sre.sh',
        executable: cliExecutable.value,
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
    const payload = await buildTrackedPayload(text, activePayload.value.filename, activePayload.value.executable)
    await navigator.clipboard.writeText(payload)
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

const download = async (text: string, filename: string) => {
  const payload = await buildTrackedPayload(text, filename, activePayload.value.executable)
  const blob = new Blob([payload], { type: 'text/x-shellscript;charset=utf-8' })
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

const buildTrackedPayload = async (text: string, filename: string, executable: boolean) => {
  if (!executable || !text.trim()) return text
  try {
    const prepared = await prepareExecutionRecord({
      source: 'init-tools',
      category: filename.replace(/\.sh$/, ''),
      name: props.title || filename,
      command: text.slice(0, 4000),
      rollback_capability: 'manual',
      rollback_plan: {
        mode: 'manual',
        advice: '初始化脚本可能修改系统配置，请结合执行输出、备份文件和目标节点状态人工恢复。',
      },
      rollback_advice: '初始化脚本当前提供人工回滚建议；如脚本输出包含备份路径，请优先使用对应备份恢复。',
      metadata: {
        filename,
        tab: activeTab.value,
      },
    })
    return wrapShellScriptWithReporting(text, prepared, filename)
  } catch {
    ElMessage.warning('执行记录初始化失败，已保留原脚本内容')
    return text
  }
}

const shellSingleQuote = (value: string) => `'${value.replace(/'/g, `'\\''`)}'`

const getApiBase = () => {
  const raw = import.meta.env.VITE_BASE_API || '/ft-api'
  return new URL(raw, window.location.origin).toString().replace(/\/$/, '')
}

const wrapShellScriptWithReporting = (
  original: string,
  prepared: { id: string; correlationId: string; reportToken: string },
  filename: string,
) => {
  const apiBase = getApiBase()
  const safeName = (props.title || filename).replace(/"/g, '\\"')
  return `#!/usr/bin/env bash
# OpsFleet execution-record wrapper. Reporting is best-effort and never changes the script result.
set +e
OPSFLEET_REPORT_API=${shellSingleQuote(apiBase)}
OPSFLEET_EXECUTION_ID=${shellSingleQuote(prepared.id)}
OPSFLEET_EXECUTION_CORRELATION_ID=${shellSingleQuote(prepared.correlationId)}
OPSFLEET_EXECUTION_TOKEN=${shellSingleQuote(prepared.reportToken)}

opsfleet_report_start() {
  curl -fsS -m 2 -X POST "$OPSFLEET_REPORT_API/api/execution-records/report/start" \\
    -H 'Content-Type: application/json' --data-binary @- >/dev/null 2>&1 <<JSON || true
{"correlation_id":"$OPSFLEET_EXECUTION_CORRELATION_ID","token":"$OPSFLEET_EXECUTION_TOKEN","source":"init-tools","category":"${filename.replace(/"/g, '\\"')}","name":"${safeName}","status":"running","rollback_capability":"manual","rollback_advice":"初始化脚本当前提供人工回滚建议；请结合输出和备份文件恢复。"}
JSON
}

opsfleet_report_finish() {
  local code="$1"
  local status="success"
  if [[ "$code" != "0" ]]; then status="failed"; fi
  curl -fsS -m 2 -X POST "$OPSFLEET_REPORT_API/api/execution-records/report/finish" \\
    -H 'Content-Type: application/json' --data-binary @- >/dev/null 2>&1 <<JSON || true
{"record_id":"$OPSFLEET_EXECUTION_ID","correlation_id":"$OPSFLEET_EXECUTION_CORRELATION_ID","token":"$OPSFLEET_EXECUTION_TOKEN","status":"$status","exit_code":$code}
JSON
}

opsfleet_report_start
tmp_script="$(mktemp /tmp/opsfleet-init-XXXXXX.sh)"
cat > "$tmp_script" <<'OPSFLEET_ORIGINAL_SCRIPT'
${original}
OPSFLEET_ORIGINAL_SCRIPT
bash "$tmp_script"
exit_code="$?"
rm -f "$tmp_script"
opsfleet_report_finish "$exit_code"
exit "$exit_code"
`
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
  background: #f5f5f5;
  padding: 0 4px;
  border-radius: 3px;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  font-size: 12px;
}

.dialog-hint strong {
  color: var(--el-color-primary);
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
