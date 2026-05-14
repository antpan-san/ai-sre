<template>
  <div class="feature-billing page-shell page-shell--crud-wide">
    <div class="page-head">
      <div>
        <h2 class="page-title">功能与计费</h2>
        <p class="page-sub">
          功能入口默认可见；关闭展示会隐藏入口，关闭执行会禁止所有角色执行。开启计费后，非超级管理员需拥有对应功能包权益。
        </p>
      </div>
      <el-button type="primary" :loading="saving" :disabled="loading" @click="handleSave">保存</el-button>
    </div>

    <el-card shadow="never" v-loading="loading">
      <el-table :data="rows" stripe border size="small" empty-text="暂无配置">
        <el-table-column prop="feature_key" label="功能键" min-width="200" show-overflow-tooltip />
        <el-table-column label="功能包" min-width="190">
          <template #default="{ row }">
            <el-select v-model="row.pack_key" size="small">
              <el-option v-for="p in packOptions" :key="p" :label="p" :value="p" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="说明" min-width="220">
          <template #default="{ row }">
            <el-input v-model="row.description" type="textarea" :rows="2" maxlength="512" show-word-limit />
          </template>
        </el-table-column>
        <el-table-column label="展示" width="100" align="center">
          <template #default="{ row }">
            <el-switch v-model="row.visible_enabled" />
          </template>
        </el-table-column>
        <el-table-column label="执行" width="100" align="center">
          <template #default="{ row }">
            <el-switch v-model="row.execution_enabled" />
          </template>
        </el-table-column>
        <el-table-column label="计费" width="100" align="center">
          <template #default="{ row }">
            <el-switch v-model="row.billing_enabled" />
          </template>
        </el-table-column>
        <el-table-column label="Stripe Price" min-width="180">
          <template #default="{ row }">
            <el-input v-model="row.stripe_price_id" size="small" placeholder="price_xxx" clearable />
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="172" align="center">
          <template #default="{ row }">
            {{ formatTs(row.updated_at) }}
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { getAdminFeatureBilling, putAdminFeatureBilling, type FeatureBillingRow } from '../../api/billing'

const loading = ref(false)
const saving = ref(false)
const rows = ref<FeatureBillingRow[]>([])
const packOptions = [
  'pack.k8s_delivery',
  'pack.node_ops',
  'pack.monitoring',
  'pack.backup_performance',
  'skillpack.k8s',
  'skillpack.kafka',
  'skillpack.redis',
  'skillpack.nginx',
  'skillpack.mysql',
  'skillpack.elasticsearch'
]

function formatTs(s?: string) {
  if (!s) return '—'
  return String(s).replace('T', ' ').slice(0, 19)
}

const load = async () => {
  loading.value = true
  try {
    const data = await getAdminFeatureBilling()
    rows.value = (data || []).map((r) => ({ ...r }))
  } catch {
    rows.value = []
  } finally {
    loading.value = false
  }
}

const handleSave = async () => {
  saving.value = true
  try {
    const items = rows.value.map((r) => ({
      feature_key: r.feature_key,
      pack_key: r.pack_key,
      visible_enabled: r.visible_enabled,
      execution_enabled: r.execution_enabled,
      billing_enabled: r.billing_enabled,
      stripe_price_id: r.stripe_price_id,
      description: r.description
    }))
    const next = await putAdminFeatureBilling(items)
    rows.value = (next || []).map((r) => ({ ...r }))
    ElMessage.success('已保存')
  } catch {
    /* 拦截器已提示 */
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  void load()
})
</script>

<style scoped>
.page-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 16px;
}

.page-title {
  margin: 0 0 6px;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
}

.page-sub {
  margin: 0;
  max-width: 780px;
  font-size: 13px;
  color: #606266;
  line-height: 1.5;
}

.hint-code {
  font-family: ui-monospace, monospace;
  font-size: 12px;
  background: var(--el-fill-color-light);
  padding: 1px 5px;
  border-radius: 4px;
}
</style>
