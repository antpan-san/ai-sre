<template>
  <!--
    通用「目标节点 + 系统类型」选择器
    - 节点列表来自 useMachineStore，已上线机器优先
    - 系统类型与 ansible-agent / opsfleet 实际支持的发行版对齐
    - 子组件不持久化数据；由父组件通过 v-model 接管
  -->
  <div class="node-system-selector">
    <div class="nss-row">
      <label class="nss-label">
        目标节点
        <el-tag v-if="modelValue.nodes.length > 0" type="info" size="small">
          已选 {{ modelValue.nodes.length }}
        </el-tag>
      </label>
      <div class="nss-control">
        <el-select
          :model-value="modelValue.nodes"
          multiple
          filterable
          collapse-tags
          collapse-tags-tooltip
          :placeholder="machineLoading ? '加载机器列表...' : (machineList.length === 0 ? '暂无可用机器，请先纳管' : '请选择需要执行的目标节点')"
          :loading="machineLoading"
          :disabled="disabled"
          style="width: 100%"
          @update:model-value="onNodesChange"
        >
          <el-option
            v-for="m in machineOptions"
            :key="m.id || m.ip"
            :label="`${m.name || m.ip}（${m.ip}）`"
            :value="m.id || m.ip"
          >
            <div class="nss-option">
              <span class="nss-option-name">{{ m.name || m.ip }}</span>
              <span class="nss-option-meta">
                <el-tag :type="m.status === 'online' ? 'success' : 'info'" size="small" effect="plain">
                  {{ m.status === 'online' ? '在线' : '离线' }}
                </el-tag>
                <span class="nss-option-ip">{{ m.ip }}</span>
                <span v-if="m.os_version" class="nss-option-os">{{ m.os_version }}</span>
              </span>
            </div>
          </el-option>
        </el-select>
        <div v-if="machineList.length === 0 && !machineLoading" class="nss-tip">
          未发现可用机器，可以
          <el-link type="primary" @click="goMachineList">前往机器纳管</el-link>
          后再回来执行此优化。
        </div>
      </div>
    </div>

    <div class="nss-row">
      <label class="nss-label">系统类型</label>
      <div class="nss-control">
        <el-select
          :model-value="modelValue.osType"
          :disabled="disabled"
          placeholder="请选择目标节点的发行版"
          style="width: 100%"
          @update:model-value="onOsChange"
        >
          <el-option-group
            v-for="grp in OS_GROUPS"
            :key="grp.label"
            :label="grp.label"
          >
            <el-option
              v-for="opt in grp.options"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </el-option-group>
        </el-select>
        <el-tooltip
          content="系统类型决定后端实际执行的 Ansible playbook 与包管理器（apt / dnf / zypper）"
          placement="top"
        >
          <el-icon class="nss-help"><QuestionFilled /></el-icon>
        </el-tooltip>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { QuestionFilled } from '@element-plus/icons-vue'
import { useMachineStore } from '../../stores/machine'

export type OsType =
  | 'ubuntu-24.04'
  | 'ubuntu-22.04'
  | 'ubuntu-20.04'
  | 'debian-12'
  | 'debian-11'
  | 'centos-7'
  | 'centos-stream-9'
  | 'rocky-9'
  | 'rhel-9'
  | 'openeuler-22.03'
  | 'kylin-v10'
  | 'other-linux'

export interface NodeSystemValue {
  // 节点 ID 或 IP（兼容尚未注册的机器）
  nodes: string[]
  osType: OsType | ''
}

const props = defineProps<{
  modelValue: NodeSystemValue
  disabled?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', val: NodeSystemValue): void
}>()

const router = useRouter()
const machineStore = useMachineStore()

const machineList = computed(() => machineStore.machineList)
const machineLoading = computed(() => machineStore.loading)

const machineOptions = computed(() => {
  // 在线节点排前面，便于一键勾选；离线但已纳管的也允许选（实际下发由后端校验）
  const list = [...machineList.value]
  return list.sort((a, b) => {
    if (a.status === b.status) return 0
    return a.status === 'online' ? -1 : 1
  })
})

const OS_GROUPS: { label: string; options: { label: string; value: OsType }[] }[] = [
  {
    label: 'Ubuntu / Debian (apt)',
    options: [
      { label: 'Ubuntu 24.04 LTS (Noble)', value: 'ubuntu-24.04' },
      { label: 'Ubuntu 22.04 LTS (Jammy)', value: 'ubuntu-22.04' },
      { label: 'Ubuntu 20.04 LTS (Focal)', value: 'ubuntu-20.04' },
      { label: 'Debian 12 (Bookworm)', value: 'debian-12' },
      { label: 'Debian 11 (Bullseye)', value: 'debian-11' },
    ],
  },
  {
    label: 'RHEL 系 (dnf / yum)',
    options: [
      { label: 'CentOS 7', value: 'centos-7' },
      { label: 'CentOS Stream 9', value: 'centos-stream-9' },
      { label: 'Rocky Linux 9', value: 'rocky-9' },
      { label: 'RHEL 9', value: 'rhel-9' },
    ],
  },
  {
    label: '国产化 / 其它',
    options: [
      { label: 'openEuler 22.03 LTS', value: 'openeuler-22.03' },
      { label: 'Kylin V10', value: 'kylin-v10' },
      { label: '其它 Linux（保守模式）', value: 'other-linux' },
    ],
  },
]

const onNodesChange = (nodes: string[]) => {
  emit('update:modelValue', { ...props.modelValue, nodes: [...nodes] })
}

const onOsChange = (osType: OsType | '') => {
  emit('update:modelValue', { ...props.modelValue, osType })
}

const goMachineList = () => {
  router.push('/service/k8s/clusters')
}

onMounted(async () => {
  if (machineList.value.length === 0 && !machineLoading.value) {
    await machineStore.fetchMachineList({ page: 1, pageSize: 100 })
  }
})
</script>

<style scoped>
.node-system-selector {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
  padding: 10px 12px;
  background: #f8fafc;
  border: 1px dashed #d6dee9;
  border-radius: 8px;
}

.nss-row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.nss-label {
  flex: 0 0 70px;
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  color: #1f2937;
  font-weight: 500;
  font-size: 12.5px;
}

.nss-control {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.nss-control :deep(.el-select) {
  min-width: 0;
}

.nss-control :deep(.el-select__wrapper) {
  min-width: 0;
}

.nss-tip {
  margin-top: 6px;
  font-size: 12px;
  color: #6b7280;
}

.nss-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.nss-option-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  color: #6b7280;
  font-size: 12px;
}

.nss-option-name,
.nss-option-ip {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.nss-option-os {
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.nss-help {
  flex: 0 0 auto;
  color: #94a3b8;
  cursor: help;
}
</style>
