export interface WorkloadSpecialLink {
  label: string
  path: string
  primary?: boolean
}

export interface WorkloadCapabilityMeta {
  id: string
  title: string
  description: string
  tags: string[]
  accent: string
  capabilityId?: string
  links: WorkloadSpecialLink[]
  commands?: { label: string; template: string }[]
}

export interface WorkloadServiceGroup {
  id: string
  title: string
  desc: string
  accent: string
  services: string[]
}

export const WORKLOAD_CAPABILITY_META: Record<string, WorkloadCapabilityMeta> = {
  k8s_delivery: {
    id: 'k8s_delivery',
    title: 'Kubernetes 交付',
    description: '离线 / 在线集群安装、恢复与卸载。',
    tags: ['K8s', 'Bundle', '安装'],
    accent: '#f97316',
    links: [
      { label: '打开 K8s 页面', path: '/app/service/k8s-deploy', primary: true },
      { label: '查看部署进度', path: '/app/service/k8s-deploy/progress' }
    ],
    commands: [
      { label: '安装', template: 'ai-sre ops k8s install --help' },
      { label: '恢复', template: 'ai-sre ops k8s recover --cluster <name>' },
      { label: '卸载', template: 'ai-sre ops k8s uninstall --cluster <name>' }
    ]
  },
  k8s_mirror: {
    id: 'k8s_mirror',
    title: 'K8s 制品目录',
    description: '内网制品 manifest 与离线安装包索引。',
    tags: ['Manifest', '离线包', '制品'],
    accent: '#8b5cf6',
    links: [{ label: '打开制品目录', path: '/app/k8s-mirror', primary: true }],
    commands: [{ label: '制品目录', template: 'ai-sre ops k8s mirror --help' }]
  },
  linux_hosts: {
    id: 'linux_hosts',
    title: 'Linux 主机',
    description: '主机服务状态、systemd 查询与运维操作。',
    tags: ['Linux', 'Systemd', '主机'],
    accent: '#06b6d4',
    links: [{ label: '打开主机页面', path: '/app/service/linux', primary: true }],
    commands: [
      { label: '服务查询', template: 'ai-sre ops host service list --host <ip>' },
      { label: '服务操作', template: 'ai-sre ops host service restart --host <ip> --name <service>' }
    ]
  },
  init_tools: {
    id: 'init_tools',
    title: '节点初始化',
    description: '时间同步、系统参数、安全加固与磁盘优化。',
    tags: ['初始化', 'Ansible', '基线'],
    accent: '#10b981',
    links: [{ label: '打开初始化页面', path: '/app/init-tools', primary: true }]
  },
  init_time_sync: {
    id: 'init_time_sync',
    capabilityId: 'init_tools',
    title: '时间同步',
    description: '统一时区、NTP 源与节点时钟偏差，作为集群和数据库部署前置检查。',
    tags: ['NTP', '时区', 'Chrony'],
    accent: '#f59e0b',
    links: [{ label: '打开时间同步页面', path: '/app/init-tools#time-sync', primary: true }],
    commands: [{ label: '时间同步', template: 'ai-sre node tune time-sync --help' }]
  },
  init_sys_param: {
    id: 'init_sys_param',
    capabilityId: 'init_tools',
    title: '系统参数优化',
    description: '优化文件句柄、内核参数与基础系统资源阈值，支撑中间件与 K8s 运行。',
    tags: ['sysctl', 'ulimit', '内核参数'],
    accent: '#2563eb',
    links: [{ label: '打开系统参数页面', path: '/app/init-tools#sys-param', primary: true }],
    commands: [{ label: '系统参数优化', template: 'ai-sre node tune sys-param --help' }]
  },
  init_security_hardening: {
    id: 'init_security_hardening',
    capabilityId: 'init_tools',
    title: '系统安全加固',
    description: '生成基础安全基线脚本，覆盖 ssh、权限与常见风险项治理。',
    tags: ['安全', 'SSH', '加固'],
    accent: '#dc2626',
    links: [{ label: '打开安全加固页面', path: '/app/init-tools#security', primary: true }],
    commands: [{ label: '系统安全加固', template: 'ai-sre node tune security --help' }]
  },
  init_disk_optimize: {
    id: 'init_disk_optimize',
    capabilityId: 'init_tools',
    title: '磁盘分区优化',
    description: '针对数据盘、日志盘和挂载目录生成标准化磁盘布局与初始化脚本。',
    tags: ['磁盘', '分区', '挂载'],
    accent: '#0f766e',
    links: [{ label: '打开磁盘优化页面', path: '/app/init-tools#disk', primary: true }],
    commands: [{ label: '磁盘分区优化', template: 'ai-sre node tune disk --help' }]
  }
}

export const WORKLOAD_SERVICE_GROUPS: WorkloadServiceGroup[] = [
  {
    id: 'gateway',
    title: '网关与负载',
    desc: 'Web 网关、反向代理与四 / 七层负载均衡。',
    accent: '#2563eb',
    services: ['nginx', 'haproxy']
  },
  {
    id: 'cache-mq',
    title: '缓存与消息',
    desc: '缓存、键值存储与消息队列基础组件。',
    accent: '#7c3aed',
    services: ['redis', 'kafka']
  },
  {
    id: 'database',
    title: '数据库',
    desc: '关系型数据库单机部署与基础参数生成。',
    accent: '#0f766e',
    services: ['mysql', 'postgresql']
  },
  {
    id: 'search',
    title: '搜索与分析',
    desc: '搜索引擎与分析组件部署配置。',
    accent: '#d97706',
    services: ['elasticsearch']
  }
]

export const WORKLOAD_DETAIL_SERVICE_LINK = (serviceKey: string) =>
  `/app/service/deploy/${encodeURIComponent(serviceKey)}`

export const WORKLOAD_DETAIL_CAPABILITY_ROUTE = (capId: string) =>
  `/app/workloads/${encodeURIComponent(capId)}`
