<template>
  <el-tag size="small" :type="tagType">{{ status }}</el-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { SubscriptionStatusLabel } from '../../config/capabilityCatalog'

const props = defineProps<{ status: SubscriptionStatusLabel }>()

const tagType = computed(() => {
  switch (props.status) {
    case '已订阅':
    case '管理员已开通':
    case '免费可用':
      return 'success'
    case '未订阅':
      return 'warning'
    case '联系管理员开通':
      return 'info'
    default:
      return 'info'
  }
})
</script>
