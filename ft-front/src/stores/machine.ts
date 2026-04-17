import { defineStore } from 'pinia'
import { ref, reactive, computed } from 'vue'
import {
  getMachineList,
  getMachineDetail,
  addMachine,
  updateMachine,
  deleteMachine,
  batchDeleteMachine,
  registerWorkerNodes
} from '../api/machine'
import type {
  Machine,
  MachineTreeNode,
  MachineListParams,
  MachineForm,
  MachineHeartbeatData,
  MachineStatusUpdateData,
  RegisterWorkerNode
} from '../types'

export const useMachineStore = defineStore('machine', () => {
  // ==================== 状态 ====================
  const machineList = ref<Machine[]>([])
  const total = ref<number>(0)
  const loading = ref<boolean>(false)
  const filters = reactive<MachineListParams>({
    page: 1,
    pageSize: 10,
    name: '',
    status: '',
    startDate: '',
    endDate: '',
    cluster_id: '',
    node_role: undefined
  })
  const pendingOfflineTimers = new Map<string, ReturnType<typeof setTimeout>>()
  const OFFLINE_APPLY_DELAY_MS = 1500

  // ==================== 计算属性 ====================

  /**
   * 将扁平的机器列表构建为 master → worker 的二级树结构
   * - master / standalone 作为一级行
   * - worker 挂载到对应 master 的 children 下
   * - 若 worker 的 master 不在当前分页中，则作为独立行展示
   */
  const treeData = computed<MachineTreeNode[]>(() => {
    const list = machineList.value
    if (!list.length) return []

    // 按 id 建立索引
    const machineMap = new Map<string, MachineTreeNode>()
    list.forEach(m => {
      machineMap.set(m.id, { ...m, children: [] })
    })

    const roots: MachineTreeNode[] = []

    machineMap.forEach(node => {
      if (node.node_role === 'worker' && node.master_machine_id) {
        // 尝试挂载到 master
        const parent = machineMap.get(node.master_machine_id)
        if (parent) {
          parent.children!.push(node)
        } else {
          // master 不在当前数据集中，作为独立行
          roots.push(node)
        }
      } else {
        // master 或 standalone 作为一级行
        roots.push(node)
      }
    })

    // 清理无 children 的节点，保持数据干净
    roots.forEach(node => {
      if (node.children && node.children.length === 0) {
        delete node.children
      }
    })
    machineMap.forEach(node => {
      if (node.children && node.children.length === 0) {
        delete node.children
      }
    })

    return roots
  })

  /**
   * 提取不重复的集群 ID 列表，用于筛选下拉框
   */
  const clusterList = computed<string[]>(() => {
    const set = new Set<string>()
    machineList.value.forEach(m => {
      if (m.cluster_id) set.add(m.cluster_id)
    })
    return Array.from(set)
  })

  // ==================== Actions ====================

  // 获取机器列表
  const fetchMachineList = async (params?: Partial<MachineListParams>) => {
    loading.value = true
    try {
      const queryParams = { ...filters, ...params }
      // 清理空值
      Object.keys(queryParams).forEach(key => {
        const k = key as keyof MachineListParams
        if (queryParams[k] === '' || queryParams[k] === undefined || queryParams[k] === null) {
          delete queryParams[k]
        }
      })
      const res = await getMachineList(queryParams) as any
      machineList.value = res.list || []
      clearAllPendingOfflineTimers()
      total.value = res.total || 0
      return res
    } catch (error) {
      console.error('获取机器列表失败:', error)
      return null
    } finally {
      loading.value = false
    }
  }

  // 获取机器详情
  const getMachineDetailById = async (id: string) => {
    try {
      const res = await getMachineDetail(id)
      return res
    } catch (error) {
      console.error('获取机器详情失败:', error)
      return null
    }
  }

  // 添加机器
  const addNewMachine = async (data: MachineForm) => {
    try {
      const res = await addMachine(data)
      await fetchMachineList()
      return res
    } catch (error) {
      console.error('添加机器失败:', error)
      return null
    }
  }

  // 更新机器
  const updateExistingMachine = async (id: string, data: MachineForm) => {
    try {
      const res = await updateMachine(id, data)
      await fetchMachineList()
      return res
    } catch (error) {
      console.error('更新机器失败:', error)
      return null
    }
  }

  // 删除机器
  const removeMachine = async (id: string) => {
    try {
      await deleteMachine(id)
      await fetchMachineList()
      return true
    } catch (error) {
      console.error('删除机器失败:', error)
      return false
    }
  }

  // 批量删除机器
  const batchRemoveMachine = async (ids: string[]) => {
    try {
      await batchDeleteMachine(ids)
      await fetchMachineList()
      return true
    } catch (error) {
      console.error('批量删除机器失败:', error)
      return false
    }
  }

  // 注册受控节点 (为 master 添加 worker 节点)
  const registerWorkerNodesById = async (masterId: string, workers: RegisterWorkerNode[]) => {
    try {
      const res = await registerWorkerNodes(masterId, workers)
      await fetchMachineList()
      return res
    } catch (error) {
      console.error('注册受控节点失败:', error)
      return null
    }
  }

  // ==================== 实时状态更新 (WebSocket) ====================

  /**
   * 处理来自 WebSocket 的机器心跳数据，实时合并到 machineList 中。
   * 如果机器不在当前列表中（新注册的），追加到列表末尾。
   */
  const handleMachineHeartbeat = (data: MachineHeartbeatData) => {
    if (!data || !data.ip) return

    // Find the machine by client_id or IP
    const idx = machineList.value.findIndex(
      m => (data.client_id && m.client_id === data.client_id) || m.ip === data.ip
    )

    if (idx >= 0) {
      // Update existing machine with real-time metrics
      const machine = machineList.value[idx]
      if (!machine) return
      cancelPendingOffline(getMachineRealtimeKey(machine))
      machine.status = data.status === 'online' ? 'online' : 'offline'
      machine.cpu_usage = data.cpu_usage
      machine.memory_usage = data.memory_usage
      machine.disk_usage = data.disk_usage
      machine.os_version = data.os_version
      machine.kernel_version = data.kernel_version
      machine.cpu_cores = data.cpu_cores
      machine.memory_total = data.memory_total
      machine.memory_used = data.memory_used
      machine.disk_total = data.disk_total
      machine.disk_used = data.disk_used
      machine.last_heartbeat_at = new Date().toISOString()
    }

    // Update / append worker machines from the heartbeat's workers array.
    // Workers are SSH-probed by the master client (NodeManager) and reported
    // back in each heartbeat.  When a worker is newly discovered (not yet in
    // the local list), we need to add it so the tree-view updates without a
    // full page refresh.
    if (data.workers && data.workers.length > 0) {
      let hasNewWorkers = false
      for (const worker of data.workers) {
        const workerIdx = machineList.value.findIndex(m => m.ip === worker.ip)
        if (workerIdx >= 0) {
          // Update existing row
          const wm = machineList.value[workerIdx]
          if (!wm) continue
          cancelPendingOffline(getMachineRealtimeKey(wm))
          wm.status = worker.status === 'up' ? 'online' : 'offline'
          wm.cpu_usage = worker.cpu_usage
          wm.memory_total = worker.memory_total
          wm.memory_used = worker.memory_used
          wm.memory_usage = worker.memory_usage
          wm.disk_total = worker.disk_total
          wm.disk_used = worker.disk_used
          wm.disk_usage = worker.disk_usage
          wm.os_version = worker.os_version
          wm.kernel_version = worker.kernel_version
          wm.cpu_cores = worker.cpu_cores
          wm.last_heartbeat_at = new Date().toISOString()
          // Persist latest worker probe diagnostics for table display.
          const metadata = { ...(wm.metadata || {}) }
          metadata.probe_status = worker.status
          if (worker.probe_error && worker.probe_error.trim() !== '') {
            metadata.probe_error = worker.probe_error
          } else {
            delete metadata.probe_error
          }
          wm.metadata = metadata
        } else {
          // New worker discovered via SSH probing — schedule a list refresh so
          // the server-assigned DB record (created by processSecondaryHosts)
          // is loaded with its full metadata (id, master_machine_id, etc.).
          hasNewWorkers = true
        }
      }
      // Debounce the refresh: only fire once even if multiple new workers arrive
      if (hasNewWorkers) {
        scheduleListRefresh()
      }
    }
  }

  // Debounce timer for refreshing the machine list when new workers are detected.
  let _listRefreshTimer: ReturnType<typeof setTimeout> | null = null
  const scheduleListRefresh = () => {
    if (_listRefreshTimer) return
    _listRefreshTimer = setTimeout(async () => {
      _listRefreshTimer = null
      await fetchMachineList()
    }, 2000)
  }

  const handleMachineStatusUpdate = (updates: MachineStatusUpdateData[]) => {
    if (!Array.isArray(updates) || updates.length === 0) return
    for (const item of updates) {
      const idx = machineList.value.findIndex(
        m =>
          (item.id && m.id === item.id) ||
          (item.client_id && m.client_id === item.client_id) ||
          m.ip === item.ip
      )
      if (idx >= 0) {
        const machine = machineList.value[idx]
        if (!machine) continue
        const nextStatus = item.status === 'online' ? 'online' : 'offline'
        if (nextStatus === 'online') {
          cancelPendingOffline(getMachineRealtimeKey(machine))
          machine.status = 'online'
        } else {
          // Delay offline apply to absorb noisy/competing offline events.
          scheduleOfflineApply(machine, item)
        }
        if (item.last_heartbeat_at !== undefined) {
          machine.last_heartbeat_at = item.last_heartbeat_at
        }
      }
    }
  }

  const getMachineRealtimeKey = (m: Partial<Machine>) => m.client_id || m.ip || m.id || ''

  const clearRealtimeMetrics = (m: Machine) => {
    m.cpu_usage = undefined
    m.memory_usage = undefined
    m.disk_usage = undefined
    m.memory_total = undefined
    m.memory_used = undefined
    m.disk_total = undefined
    m.disk_used = undefined
  }

  const scheduleOfflineApply = (machine: Machine, item: MachineStatusUpdateData) => {
    const key = getMachineRealtimeKey(machine)
    if (!key) return
    cancelPendingOffline(key)
    const timer = setTimeout(() => {
      const current = machineList.value.find(
        m =>
          (item.id && m.id === item.id) ||
          (item.client_id && m.client_id === item.client_id) ||
          m.ip === item.ip
      )
      if (!current) return
      current.status = 'offline'
      clearRealtimeMetrics(current)
      if (item.last_heartbeat_at !== undefined) {
        current.last_heartbeat_at = item.last_heartbeat_at
      }
      pendingOfflineTimers.delete(key)
    }, OFFLINE_APPLY_DELAY_MS)
    pendingOfflineTimers.set(key, timer)
  }

  const cancelPendingOffline = (key: string) => {
    const timer = pendingOfflineTimers.get(key)
    if (timer) {
      clearTimeout(timer)
      pendingOfflineTimers.delete(key)
    }
  }

  const clearAllPendingOfflineTimers = () => {
    pendingOfflineTimers.forEach(timer => clearTimeout(timer))
    pendingOfflineTimers.clear()
  }

  // 设置过滤器
  const setFilters = (newFilters: Partial<MachineListParams>) => {
    Object.assign(filters, newFilters)
  }

  // 重置过滤器
  const resetFilters = () => {
    filters.page = 1
    filters.pageSize = 10
    filters.name = ''
    filters.status = ''
    filters.startDate = ''
    filters.endDate = ''
    filters.cluster_id = ''
    filters.node_role = undefined
  }

  return {
    // 状态
    machineList,
    total,
    loading,
    filters,
    // 计算属性
    treeData,
    clusterList,
    // Actions
    fetchMachineList,
    getMachineDetail: getMachineDetailById,
    addNewMachine,
    updateExistingMachine,
    removeMachine,
    batchRemoveMachine,
    registerWorkerNodes: registerWorkerNodesById,
    setFilters,
    resetFilters,
    handleMachineHeartbeat,
    handleMachineStatusUpdate
  }
})
