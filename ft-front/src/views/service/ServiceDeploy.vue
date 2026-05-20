<template>
  <div class="service-deploy page-shell">
    <div class="split-layout">
      <el-card class="catalog-card" shadow="never">
        <template #header>
          <div class="catalog-header">
            <span class="catalog-title">基础服务</span>
            <span class="catalog-sub">选择后在右侧配置</span>
          </div>
        </template>
        <div class="catalog-list">
          <div
            v-for="item in catalog"
            :key="item.key"
            :class="['catalog-item', { selected: form.service === item.key }]"
            @click="selectService(item.key)"
          >
            <div class="catalog-item-head">
              <span class="catalog-name">{{ item.name }}</span>
              <el-icon v-if="form.service === item.key" class="catalog-check"><Check /></el-icon>
            </div>
            <div class="catalog-desc">{{ item.description }}</div>
            <div class="catalog-tags">
              <el-tag v-for="t in item.tags" :key="t" size="small" type="info" effect="plain">{{ t }}</el-tag>
            </div>
          </div>
        </div>
      </el-card>

      <div class="config-pane">
        <el-empty
          v-if="!selected"
          description="左侧选择一个基础服务后，在此配置参数并生成部署脚本"
          class="empty-pane"
        />

        <ServiceDeployConfigBody v-else :service-key="form.service" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Check } from '@element-plus/icons-vue'
import { useServiceDeploy } from '../../composables/useServiceDeploy'
import ServiceDeployConfigBody from '../../components/deploy/ServiceDeployConfigBody.vue'

const { form, catalog, selected, selectService } = useServiceDeploy()
</script>
<style scoped>
.service-deploy {
  width: 100%;
  max-width: none;
  height: 100%;
  min-height: 0;
  margin: 0;
  padding: 8px var(--page-padding-x, 24px) 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  box-sizing: border-box;
  overflow: hidden;
}

.service-deploy :deep(.el-card) {
  border-radius: 12px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
}

.service-deploy :deep(.el-card__header) {
  padding: 10px 16px;
  font-size: 14px;
}

.service-deploy :deep(.el-card__body) {
  padding: 12px 16px;
}

.catalog-header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
}

.catalog-title {
  font-weight: 600;
}

.catalog-sub {
  color: #94a3b8;
  font-size: 12px;
}

.split-layout {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: 280px 1fr;
  grid-template-rows: minmax(0, 1fr);
  gap: 12px;
  align-items: stretch;
  overflow: hidden;
}

@media (max-width: 960px) {
  .split-layout {
    grid-template-columns: 1fr;
    grid-template-rows: minmax(0, min(320px, 40vh)) minmax(0, 1fr);
  }
}

.catalog-card {
  min-height: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.catalog-card :deep(.el-card__body) {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  padding-top: 10px;
}

.catalog-list {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  overscroll-behavior: contain;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-right: 2px;
}

.catalog-item {
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  padding: 10px 12px;
  cursor: pointer;
  background: var(--el-bg-color);
  transition: border-color .15s, box-shadow .15s, background .15s;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.catalog-item:hover {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 2px rgba(64,158,255,.15);
}

.catalog-item.selected {
  border-color: var(--el-color-primary);
  background: rgba(64,158,255,.08);
}

.catalog-item-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.catalog-check {
  color: var(--el-color-primary);
  font-size: 16px;
}

.catalog-name {
  font-weight: 600;
}

.catalog-desc {
  font-size: 12px;
  color: #6b7280;
}

.catalog-tags {
  margin-top: 2px;
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.config-pane {
  min-height: 0;
  min-width: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow-y: auto;
  overflow-x: hidden;
  overscroll-behavior: contain;
  padding-right: 2px;
}

.empty-pane {
  background: var(--el-bg-color);
  border: 1px dashed var(--el-border-color);
  border-radius: 8px;
  padding: 32px 0;
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

.switch-row--compact {
  margin-top: 0;
}

.switch-row-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: var(--el-text-color-regular);
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.switch-row-tip {
  color: #94a3b8;
  font-size: 14px;
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
  border-radius: 12px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  padding: 0 12px;
}

.advanced-collapse :deep(.el-collapse-item__header) {
  font-weight: 600;
  font-size: 14px;
}

.actions {
  margin: 12px 0 24px;
  display: flex;
  gap: 12px;
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
