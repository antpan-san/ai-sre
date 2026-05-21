<template>
  <div :class="['service-deploy-body', { 'service-deploy-body--compact': compact }]">
    <el-card v-if="!compact" class="install-card" shadow="never">
      <template #header><span>系统与安装方式</span></template>
      <el-form label-position="top">
        <el-row :gutter="16">
          <el-col :xs="24" :md="8">
            <el-form-item label="目标系统类型">
              <el-select v-model="deploy.form.osType" style="width: 100%">
                <el-option v-for="os in deploy.osTypeOptions" :key="os.value" :label="os.label" :value="os.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="8">
            <el-form-item label="安装方式">
              <el-select v-model="deploy.form.installMethod" style="width: 100%">
                <el-option
                  v-for="method in deploy.availableInstallMethods"
                  :key="method.value"
                  :label="method.label"
                  :value="method.value"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :md="8">
            <el-form-item label="部署场景">
              <el-select v-model="deploy.form.profile" style="width: 100%">
                <el-option v-for="profile in deploy.profileOptions" :key="profile.value" :label="profile.label" :value="profile.value" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </el-card>
    <el-form v-else label-position="top" class="service-deploy-body__compact-install">
      <el-row :gutter="12">
        <el-col :xs="24" :sm="8">
          <el-form-item label="系统">
            <el-select v-model="deploy.form.osType" size="small" style="width: 100%">
              <el-option v-for="os in deploy.osTypeOptions" :key="os.value" :label="os.label" :value="os.value" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :xs="24" :sm="8">
          <el-form-item label="安装方式">
            <el-select v-model="deploy.form.installMethod" size="small" style="width: 100%">
              <el-option
                v-for="method in deploy.availableInstallMethods"
                :key="method.value"
                :label="method.label"
                :value="method.value"
              />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :xs="24" :sm="8">
          <el-form-item label="场景">
            <el-select v-model="deploy.form.profile" size="small" style="width: 100%">
              <el-option v-for="profile in deploy.profileOptions" :key="profile.value" :label="profile.label" :value="profile.value" />
            </el-select>
          </el-form-item>
        </el-col>
      </el-row>
    </el-form>

    <el-card
      v-for="sec in deploy.regularSections"
      :key="sec.key"
      class="config-card"
      shadow="never"
    >
      <template #header>
        <div class="section-header">
          <span>{{ deploy.selected?.name }} · {{ sec.title }}</span>
          <span v-if="sec.hint" class="section-hint">{{ sec.hint }}</span>
        </div>
      </template>
      <el-form label-position="top">
        <div class="section-field-grid">
          <div class="section-normal-fields">
            <el-row :gutter="16">
              <el-col
                v-for="field in deploy.normalFields(sec.fields)"
                :key="field.key"
                :xs="24"
                :md="deploy.sectionNormalColMd(field)"
              >
                <el-form-item :label="field.label">
                  <el-input-number
                    v-if="field.type === 'number'"
                    v-model="deploy.form.params[field.key]"
                    :min="field.min ?? 1"
                    :max="field.max ?? 65535"
                    style="width: 100%"
                  />
                  <el-select
                    v-else-if="field.type === 'select'"
                    v-model="deploy.form.params[field.key]"
                    style="width: 100%"
                  >
                    <el-option v-for="opt in field.options" :key="opt" :label="opt" :value="opt" />
                  </el-select>
                  <el-select
                    v-else-if="field.type === 'autocomplete'"
                    v-model="deploy.form.params[field.key]"
                    filterable
                    allow-create
                    default-first-option
                    :placeholder="field.placeholder || '选择或输入自定义值'"
                    style="width: 100%"
                  >
                    <el-option v-for="opt in field.options" :key="opt" :label="opt" :value="opt" />
                  </el-select>
                  <el-input
                    v-else-if="field.type === 'textarea'"
                    v-model="deploy.form.params[field.key]"
                    type="textarea"
                    :rows="field.rows ?? 3"
                    :placeholder="field.placeholder || ''"
                  />
                  <el-input v-else v-model="deploy.form.params[field.key]" :placeholder="field.placeholder || ''" />
                </el-form-item>
              </el-col>
            </el-row>
          </div>
          <div v-if="deploy.switchFields(sec.fields).length" class="section-switch-fields">
            <div v-for="field in deploy.switchFields(sec.fields)" :key="field.key" class="switch-row switch-row--compact">
              <span class="switch-row-label">
                {{ field.label }}
                <el-tooltip v-if="field.tip" :content="field.tip" placement="top">
                  <el-icon class="switch-row-tip"><InfoFilled /></el-icon>
                </el-tooltip>
              </span>
              <el-switch v-model="deploy.form.params[field.key]" inline-prompt active-text="开" inactive-text="关" />
            </div>
          </div>
        </div>
      </el-form>
    </el-card>

    <el-collapse
      v-if="deploy.collapsibleSections.length"
      v-model="deploy.activeCollapseSections"
      class="advanced-collapse"
    >
      <el-collapse-item
        v-for="sec in deploy.collapsibleSections"
        :key="sec.key"
        :name="sec.key"
        :title="`${deploy.selected?.name} · ${sec.title}${sec.hint ? '（' + sec.hint + '）' : ''}`"
      >
        <el-form v-if="sec.fields.length" label-position="top">
          <el-row :gutter="16">
            <el-col
              v-for="field in deploy.visibleFields(sec.fields)"
              :key="field.key"
              :xs="24"
              :md="deploy.colMd(field)"
            >
              <el-form-item :label="field.label">
                <el-input-number
                  v-if="field.type === 'number'"
                  v-model="deploy.form.params[field.key]"
                  :min="field.min ?? 1"
                  :max="field.max ?? 65535"
                  style="width: 100%"
                />
                <el-select v-else-if="field.type === 'select'" v-model="deploy.form.params[field.key]" style="width: 100%">
                  <el-option v-for="opt in field.options" :key="opt" :label="opt" :value="opt" />
                </el-select>
                <el-select
                  v-else-if="field.type === 'autocomplete'"
                  v-model="deploy.form.params[field.key]"
                  filterable
                  allow-create
                  default-first-option
                  style="width: 100%"
                >
                  <el-option v-for="opt in field.options" :key="opt" :label="opt" :value="opt" />
                </el-select>
                <el-switch
                  v-else-if="field.type === 'switch'"
                  v-model="deploy.form.params[field.key]"
                  inline-prompt
                  active-text="开"
                  inactive-text="关"
                />
                <el-input
                  v-else-if="field.type === 'textarea'"
                  v-model="deploy.form.params[field.key]"
                  type="textarea"
                  :rows="field.rows ?? 3"
                />
                <el-input v-else v-model="deploy.form.params[field.key]" />
              </el-form-item>
            </el-col>
          </el-row>
        </el-form>
        <pre v-if="sec.preview === 'config'" class="code-block code-block--inline">{{ deploy.confPreview }}</pre>
      </el-collapse-item>
    </el-collapse>

    <div class="actions">
      <el-button type="primary" :icon="Upload" :loading="deploy.generating" size="small" @click="deploy.onGenerate">
        生成部署脚本
      </el-button>
      <el-button
        v-if="!compact"
        type="success"
        :icon="Check"
        :loading="deploy.submittingUpdate"
        :disabled="!deploy.canSubmitUpdate"
        @click="deploy.onSubmitUpdate"
      >
        提交配置变更
      </el-button>
      <el-button :icon="RefreshRight" size="small" @click="deploy.onReset">重置</el-button>
    </div>
    <el-alert
      v-if="deploy.generatedDeployment"
      class="deploy-status"
      :type="deploy.deploymentDirty ? 'warning' : 'success'"
      :closable="false"
      show-icon
      :title="`部署任务已保存：${deploy.generatedDeployment.deploymentId}`"
      :description="deploy.deploymentStatusDescription"
    />

    <el-dialog
      v-model="deploy.previewVisible"
      :title="`${deploy.selected?.name || ''} 部署脚本`"
      width="860px"
      :close-on-click-modal="false"
    >
      <el-alert type="info" :closable="false" show-icon title="使用方式">
        <template #default>
          <div>
            页面只保存服务端部署规格，目标机执行 <code>curl | sudo bash</code> 后会安装/升级 ai-sre，
            再由 <code>ai-sre ops service install</code> 从服务端拉取完整参数并回传执行状态。
          </div>
        </template>
      </el-alert>
      <el-tabs v-model="deploy.activeTab" class="dialog-tabs">
        <el-tab-pane label="curl + bash（推荐）" name="bash">
          <div class="tab-actions">
            <el-tag size="small" type="success">目标机执行：复制后直接运行</el-tag>
            <el-button size="small" :icon="DocumentCopy" :disabled="!deploy.generatedDeployment" @click="deploy.copy(deploy.curlCommand)">
              复制
            </el-button>
          </div>
          <pre class="code-block">{{ deploy.curlCommand }}</pre>
        </el-tab-pane>
        <el-tab-pane label="ai-sre CLI" name="cli">
          <div class="tab-actions">
            <el-button size="small" :icon="DocumentCopy" :disabled="!deploy.generatedDeployment" @click="deploy.copy(deploy.aiSreInstallCommand)">
              复制命令
            </el-button>
          </div>
          <pre class="code-block">{{ deploy.aiSreInstallCommand }}</pre>
        </el-tab-pane>
      </el-tabs>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { InfoFilled } from '@element-plus/icons-vue'
import { useServiceDeploy } from '../../composables/useServiceDeploy'

const props = defineProps<{
  serviceKey: string
  compact?: boolean
}>()

const deploy = useServiceDeploy({ fixedServiceKey: props.serviceKey }) as any
const { Upload, RefreshRight, Check, DocumentCopy } = deploy
</script>

<style scoped>
.service-deploy-body--compact .config-card :deep(.el-card__header) {
  padding: 8px 12px;
  font-size: 13px;
}
.service-deploy-body--compact .config-card :deep(.el-card__body) {
  padding: 8px 12px;
}
.service-deploy-body__compact-install {
  margin-bottom: 8px;
}
.section-header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
}
.section-hint {
  color: #94a3b8;
  font-size: 12px;
}
.section-field-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  column-gap: 16px;
  align-items: start;
}
.section-normal-fields {
  grid-column: 1 / span 3;
  min-width: 0;
}
.section-switch-fields {
  grid-column: 4 / span 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 32px;
  padding: 4px 8px;
  margin: 4px 0 18px;
  border-radius: 6px;
  border: 1px dashed var(--el-border-color);
  background: var(--el-fill-color-lighter);
}
.switch-row-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.switch-row-tip {
  color: #94a3b8;
  cursor: help;
}
@media (max-width: 1200px) {
  .section-field-grid {
    grid-template-columns: 1fr;
  }
  .section-normal-fields,
  .section-switch-fields {
    grid-column: 1;
  }
}
.advanced-collapse {
  border-radius: 8px;
  border: 1px solid var(--el-border-color);
  padding: 0 12px;
  margin-top: 8px;
}
.actions {
  margin-top: 12px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.deploy-status {
  margin-top: 8px;
}
.tab-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.code-block {
  background: #0f172a;
  color: #e2e8f0;
  padding: 12px 14px;
  border-radius: 6px;
  max-height: 480px;
  overflow: auto;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
}
.code-block--inline {
  margin-top: 4px;
  max-height: 360px;
}
.dialog-tabs {
  margin-top: 8px;
}
</style>
