/**
 * API 统一入口 - 从各模块重新导出所有 API 函数
 *
 * 使用方式：
 *   import { login, getMachineList, ... } from '@/api'
 * 或直接从子模块导入：
 *   import { login } from '@/api/auth'
 */

// 认证相关
export { login, logout, getUserInfo } from './auth'

// 仪表盘相关
export { getDashboardData } from './dashboard'

// 机器管理相关
export {
  getMachineList,
  getMachineDetail,
  addMachine,
  updateMachine,
  deleteMachine,
  batchDeleteMachine,
  updateMachineStatus,
} from './machine'

// 用户管理相关
export {
  getUserList,
  getUserDetail,
  addUser,
  updateUser,
  deleteUser,
  batchDeleteUser,
  updateUserRole,
} from './user'

// 服务管理相关
export {
  deployService,
  getServiceList,
  getServiceDetail,
  startService,
  stopService,
  restartService,
  deleteService,
  batchDeleteServices,
  getLinuxServiceList,
  operateLinuxService,
} from './service'

// Kubernetes 部署相关
export {
  getK8sVersions,
  getMachines,
  checkClusterName,
  submitDeployConfig,
  getDeployProgress,
  getDeployLogs,
} from './k8s-deploy'

// 监控告警相关
export {
  getMonitoringConfigList,
  getMonitoringConfig,
  createMonitoringConfig,
  updateMonitoringConfig,
  deleteMonitoringConfig,
  getAlertRules,
  createAlertRule,
  updateAlertRule,
  deleteAlertRule,
} from './monitoring'

// 代理配置相关
export {
  getProxyConfigList,
  getProxyConfigDetail,
  saveProxyConfig,
  deleteProxyConfig,
  applyProxyConfig,
} from './proxy'

// 安全与审计相关
export {
  getOperationLogs,
  getOperationLogDetail,
  exportOperationLogs,
  getPermissions,
  getPermissionDetail,
  addPermission,
  updatePermission,
  deletePermission,
  batchDeletePermissions,
  getRolePermissions,
  assignRolePermissions,
} from './security-audit'

// 高级功能相关
export {
  getBackups,
  getBackupDetail,
  createBackup,
  restoreBackup,
  deleteBackup,
  batchDeleteBackups,
  getBackupProgress,
  getPerformanceData,
  getPerformanceReport,
  generatePerformanceReport,
  exportPerformanceReport,
  getSystemPerformanceMetrics,
} from './advanced'

// 作业中心相关
export {
  getAvailableMachines,
  executeCommand,
  getExecutionResult,
} from './job'

// 执行记录相关
export {
  getExecutionRecords,
  getExecutionRecordDetail,
  prepareExecutionRecord,
  previewExecutionRollback,
  rollbackExecutionRecord,
} from './execution-records'

// 初始化工具相关
export {
  getSystemParams,
  updateSystemParams,
  syncTime,
  hardenSecurity,
  optimizeDisk,
} from './init-tools'
