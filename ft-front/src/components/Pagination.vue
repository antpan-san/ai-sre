<template>
  <div class="pagination-container">
    <el-pagination
      :current-page="currentPage"
      :page-size="pageSize"
      :page-sizes="pageSizes"
      :total="total"
      :layout="layout"
      :hide-on-single-page="hideOnSinglePage"
      @size-change="handleSizeChange"
      @current-change="handleCurrentChange"
    />
  </div>
</template>

<script setup lang="ts">
interface Props {
  currentPage: number
  pageSize: number
  total: number
  pageSizes?: number[]
  layout?: string
  hideOnSinglePage?: boolean
}

withDefaults(defineProps<Props>(), {
  currentPage: 1,
  pageSize: 10,
  total: 0,
  pageSizes: () => [10, 20, 50, 100],
  layout: 'total, sizes, prev, pager, next, jumper',
  hideOnSinglePage: false
})

const emit = defineEmits<{
  (e: 'update:currentPage', value: number): void
  (e: 'update:pageSize', value: number): void
  (e: 'size-change', value: number): void
  (e: 'current-change', value: number): void
}>()

const handleSizeChange = (size: number) => {
  emit('update:pageSize', size)
  emit('size-change', size)
}

const handleCurrentChange = (current: number) => {
  emit('update:currentPage', current)
  emit('current-change', current)
}
</script>

<style scoped>
.pagination-container {
  text-align: right;
  margin-top: 20px;
  margin-bottom: 20px;
}
</style>
