<template>
  <div class="k8s-mirror-page page-shell">
    <div class="mirror-toolbar">
      <h2 class="page-title">制品目录</h2>
      <el-popover placement="bottom-start" :width="360" trigger="click">
        <template #reference>
          <el-button text type="primary" size="small">数据来源</el-button>
        </template>
        <p class="page-desc--muted" style="margin: 0">
          读取制品机 manifest（<code>k8s-mirror-generate-manifest.sh</code>）。后端经
          <code>OPSFLEET_K8S_MIRROR_BASE_URL</code> / <code>_MANIFEST_URL</code> 拉取。
        </p>
      </el-popover>
    </div>

    <div v-if="loading" class="loading-wrap">
      <el-skeleton :rows="6" animated />
    </div>

    <template v-else>
      <el-alert v-if="catalog?.fetchError" type="error" :title="catalog.fetchError" :closable="false" show-icon />

      <el-descriptions v-if="catalog && !catalog.fetchError" :column="2" border class="meta">
        <el-descriptions-item label="manifest 地址">{{ catalog.manifestUrl || '—' }}</el-descriptions-item>
        <el-descriptions-item label="生成时间">{{ catalog.generatedAt || '—' }}</el-descriptions-item>
        <el-descriptions-item label="制品根目录（远端）">{{ catalog.mirrorRoot || '—' }}</el-descriptions-item>
        <el-descriptions-item label="对外 Base URL">{{ catalog.publicBaseUrl || '—' }}</el-descriptions-item>
      </el-descriptions>

      <el-table
        v-if="catalog?.files?.length"
        :data="catalog.files"
        stripe
        border
        style="width: 100%; margin-top: 16px"
        max-height="560"
      >
        <el-table-column prop="relativePath" label="相对路径" min-width="280" show-overflow-tooltip />
        <el-table-column prop="sizeBytes" label="大小 (bytes)" width="130" />
        <el-table-column label="SHA512" min-width="200">
          <template #default="{ row }">
            <span class="sha">{{ row.sha512 }}</span>
            <el-button type="primary" link size="small" @click="copyText(row.sha512)">复制</el-button>
          </template>
        </el-table-column>
        <el-table-column label="下载 URL" min-width="240">
          <template #default="{ row }">
            <el-link v-if="row.downloadUrl" :href="row.downloadUrl" target="_blank" type="primary">
              打开
            </el-link>
            <span v-else>—</span>
          </template>
        </el-table-column>
      </el-table>

      <el-empty v-else-if="catalog && !catalog.fetchError" description="manifest 中暂无文件条目" />
    </template>

    <div class="footer-actions">
      <el-button type="primary" :loading="loading" @click="load">刷新</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { getK8sMirrorCatalog, type K8sMirrorCatalog } from '../../../api/k8s-mirror'

const loading = ref(true)
const catalog = ref<K8sMirrorCatalog | null>(null)

async function load() {
  loading.value = true
  try {
    catalog.value = await getK8sMirrorCatalog()
  } catch {
    catalog.value = { fetchError: '请求失败' }
  } finally {
    loading.value = false
  }
}

function copyText(s: string) {
  navigator.clipboard.writeText(s).then(
    () => ElMessage.success('已复制'),
    () => ElMessage.error('复制失败')
  )
}

onMounted(() => {
  load()
})
</script>

<style scoped>
.k8s-mirror-page {
  padding: var(--page-padding-y) var(--page-padding-x) 24px;
  box-sizing: border-box;
}

.mirror-toolbar {
  display: flex;
  align-items: baseline;
  gap: 10px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.page-title {
  margin: 0;
  font-size: var(--page-header-title-max);
  font-weight: 600;
}
.meta {
  margin-top: 12px;
}
.sha {
  font-family: ui-monospace, monospace;
  font-size: 12px;
  word-break: break-all;
}
.loading-wrap {
  margin-top: 16px;
}
.footer-actions {
  margin-top: 16px;
}
</style>
