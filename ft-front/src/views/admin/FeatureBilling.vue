<template>
  <div class="feature-billing page-shell page-shell--crud-wide">
    <div class="page-head">
      <div>
        <h2 class="page-title">功能与计费</h2>
        <p class="page-sub">
          控制各功能是否对<strong>非管理员</strong>启用计费校验。关闭时与历史行为一致；开启后需用户具备对应权益或有效订阅（Stripe Webhook 同步）。
        </p>
      </div>
      <el-button type="primary" :loading="saving" :disabled="loading" @click="handleSave">保存</el-button>
    </div>

    <el-card shadow="never" v-loading="loading">
      <el-table :data="rows" stripe border size="small" empty-text="暂无配置">
        <el-table-column prop="feature_key" label="功能键" min-width="200" show-overflow-tooltip />
        <el-table-column prop="description" label="说明" min-width="220">
          <template #default="{ row }">
            <el-input v-model="row.description" type="textarea" :rows="2" maxlength="512" show-word-limit />
          </template>
        </el-table-column>
        <el-table-column label="启用计费" width="120" align="center">
          <template #default="{ row }">
            <el-switch v-model="row.billing_enabled" />
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
      billing_enabled: r.billing_enabled,
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
  max-width: 720px;
  font-size: 13px;
  color: #606266;
  line-height: 1.5;
}
</style>
