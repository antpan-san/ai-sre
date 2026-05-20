export type CapabilityCategory =
  | 'delivery'
  | 'troubleshoot'
  | 'observe'
  | 'monitoring'
  | 'data'
  | 'evolution'

export type SubscriptionStatusLabel =
  | '已订阅'
  | '未订阅'
  | '免费可用'
  | '管理员已开通'
  | '暂不可用'
  | '联系管理员开通'

export type WorkloadTier = 'primary' | 'secondary'

export interface CatalogCommand {
  label: string
  template: string
}

export interface CatalogCapability {
  id: string
  name: string
  description: string
  category: CapabilityCategory
  feature_key?: string
  pack_key?: string
  route_suffix?: string
  direct_route?: string
  cli_topic?: string
  cli_hint?: string
  icon?: string
  keywords?: string[]
  workload_tier?: WorkloadTier
  commands?: CatalogCommand[]
  execution_source?: string
  admin_only?: boolean
  super_admin_only?: boolean
  always_free?: boolean
}

export const CAPABILITY_CATEGORY_LABELS: Record<CapabilityCategory, string> = {
  delivery: '交付部署',
  troubleshoot: '问题排查',
  observe: '运行观测',
  monitoring: '可观测性',
  data: '数据与性能',
  evolution: '自动进化'
}

export const CAPABILITY_CATEGORY_SHORT: Record<CapabilityCategory, string> = {
  delivery: '交付',
  troubleshoot: '排查',
  observe: '观测',
  monitoring: '监控',
  data: '数据',
  evolution: '进化'
}

export const CAPABILITY_CATEGORY_DESC: Record<CapabilityCategory, string> = {
  delivery: '安装与管理 K8s、应用服务、主机与初始化工具',
  troubleshoot: 'ai-sre check 证据驱动诊断（详细表单见问题排查）',
  observe: '进程与运行时持续观测',
  monitoring: 'Prometheus 与 Exporter 监控栈',
  data: '备份恢复与性能分析',
  evolution: '平台自动进化（管理员）'
}

export const CAPABILITY_CATEGORY_ICON: Record<CapabilityCategory, string> = {
  delivery: 'Box',
  troubleshoot: 'Search',
  observe: 'View',
  monitoring: 'Monitor',
  data: 'FolderOpened',
  evolution: 'MagicStick'
}

/** 普通用户 Hub 左栏可见分类（不含 evolution 除非有项） */
export const HUB_CATEGORY_ORDER: CapabilityCategory[] = [
  'delivery',
  'troubleshoot',
  'observe',
  'monitoring',
  'data',
  'evolution'
]

export const CAPABILITY_CATALOG: CatalogCapability[] = [
  {
    id: 'k8s_delivery',
    name: 'Kubernetes 交付',
    description: '离线/在线 K8s 集群安装、恢复与卸载',
    category: 'delivery',
    feature_key: 'feature.k8s_delivery',
    pack_key: 'pack.k8s_delivery',
    direct_route: '/service/k8s-deploy',
    icon: 'Connection',
    keywords: ['k8s', 'kubernetes', '集群', '安装'],
    workload_tier: 'primary',
    execution_source: 'k8s',
    commands: [
      { label: '安装', template: 'ai-sre ops k8s install --help' },
      { label: '恢复', template: 'ai-sre ops k8s recover --cluster <name>' },
      { label: '卸载', template: 'ai-sre ops k8s uninstall --cluster <name>' }
    ]
  },
  {
    id: 'service_deploy',
    name: '应用服务部署',
    description: '中间件与应用服务安装、更新与卸载',
    category: 'delivery',
    feature_key: 'feature.node_ops',
    pack_key: 'pack.node_ops',
    direct_route: '/service/deploy',
    icon: 'Box',
    keywords: ['服务', '部署', 'redis', 'mysql', 'nginx'],
    workload_tier: 'primary',
    execution_source: 'cli',
    commands: [
      { label: '安装', template: 'ai-sre ops service install <service> --target <host>' },
      { label: '更新', template: 'ai-sre ops service update <service> --target <host>' },
      { label: '卸载', template: 'ai-sre ops service uninstall <service> --target <host>' }
    ]
  },
  {
    id: 'linux_hosts',
    name: 'Linux 主机管理',
    description: '主机上的服务状态与运维操作',
    category: 'delivery',
    feature_key: 'feature.node_ops',
    pack_key: 'pack.node_ops',
    direct_route: '/service/linux',
    icon: 'Cpu',
    keywords: ['linux', '主机', 'systemd'],
    workload_tier: 'primary'
  },
  {
    id: 'init_tools',
    name: '节点初始化',
    description: '系统参数、时间同步、安全加固等初始化脚本',
    category: 'delivery',
    feature_key: 'feature.node_ops',
    pack_key: 'pack.node_ops',
    direct_route: '/init-tools',
    icon: 'Tools',
    keywords: ['初始化', '加固', 'ntp'],
    workload_tier: 'primary',
    execution_source: 'init-tools'
  },
  {
    id: 'proxy',
    name: '出口代理',
    description: '集群与主机访问外网的代理配置',
    category: 'delivery',
    feature_key: 'feature.node_ops',
    pack_key: 'pack.node_ops',
    direct_route: '/proxy/config',
    icon: 'Link',
    keywords: ['代理', 'proxy', '出口'],
    workload_tier: 'secondary'
  },
  {
    id: 'k8s_mirror',
    name: 'K8s 制品目录',
    description: '内网制品 manifest 与离线安装包索引',
    category: 'delivery',
    feature_key: 'feature.k8s_delivery',
    pack_key: 'pack.k8s_delivery',
    direct_route: '/k8s-mirror',
    icon: 'Download',
    keywords: ['制品', 'manifest', '离线包'],
    workload_tier: 'secondary'
  },
  {
    id: 'check_redis',
    name: 'Redis 排查',
    description: '采集 + 本地规则 / AI 诊断 Redis 实例',
    category: 'troubleshoot',
    feature_key: 'feature.ai_diagnosis',
    pack_key: 'skillpack.redis',
    cli_topic: 'redis',
    icon: 'Coin',
    keywords: ['redis', '缓存', '内存'],
    route_suffix: '/troubleshooting?topic=redis'
  },
  {
    id: 'check_linux',
    name: 'Linux 性能排查',
    description: 'CPU、内存、IO 等主机性能证据诊断',
    category: 'troubleshoot',
    feature_key: 'feature.ai_diagnosis',
    pack_key: 'skillpack.k8s',
    cli_topic: 'linux',
    icon: 'Monitor',
    keywords: ['linux', 'cpu', 'io', '性能'],
    route_suffix: '/troubleshooting?topic=linux'
  },
  {
    id: 'check_k8s',
    name: 'Kubernetes 排查',
    description: 'Pod、工作负载与集群事件证据驱动根因',
    category: 'troubleshoot',
    feature_key: 'feature.ai_diagnosis',
    pack_key: 'skillpack.k8s',
    cli_topic: 'k8s',
    icon: 'Connection',
    keywords: ['k8s', 'pod', 'kubernetes'],
    route_suffix: '/troubleshooting?topic=k8s'
  },
  {
    id: 'check_kafka',
    name: 'Kafka 排查',
    description: 'Broker、Topic 与消费者 Lag 快诊',
    category: 'troubleshoot',
    feature_key: 'feature.ai_diagnosis',
    pack_key: 'skillpack.kafka',
    cli_topic: 'kafka',
    icon: 'Message',
    keywords: ['kafka', '消息', 'lag'],
    route_suffix: '/troubleshooting?topic=kafka'
  },
  {
    id: 'check_mysql',
    name: 'MySQL 排查',
    description: '连接、慢 SQL 与锁等待证据分析',
    category: 'troubleshoot',
    feature_key: 'feature.ai_diagnosis',
    pack_key: 'skillpack.mysql',
    cli_topic: 'mysql',
    icon: 'Coin',
    keywords: ['mysql', '数据库', '慢sql'],
    route_suffix: '/troubleshooting?topic=mysql'
  },
  {
    id: 'check_postgresql',
    name: 'PostgreSQL 排查',
    description: '连接、事务与慢查询诊断',
    category: 'troubleshoot',
    feature_key: 'feature.ai_diagnosis',
    pack_key: 'skillpack.postgresql',
    cli_topic: 'postgresql',
    icon: 'Coin',
    keywords: ['postgresql', 'postgres', 'pg'],
    route_suffix: '/troubleshooting?topic=postgresql'
  },
  {
    id: 'check_nginx',
    name: 'Nginx 排查',
    description: '访问日志与 upstream 健康分析',
    category: 'troubleshoot',
    feature_key: 'feature.ai_diagnosis',
    pack_key: 'skillpack.nginx',
    cli_topic: 'nginx',
    icon: 'Share',
    keywords: ['nginx', '网关', '502'],
    route_suffix: '/troubleshooting?topic=nginx'
  },
  {
    id: 'check_elasticsearch',
    name: 'Elasticsearch 排查',
    description: '集群健康、分片与节点状态',
    category: 'troubleshoot',
    feature_key: 'feature.ai_diagnosis',
    pack_key: 'skillpack.elasticsearch',
    cli_topic: 'elasticsearch',
    icon: 'Search',
    keywords: ['elasticsearch', 'es', '分片'],
    route_suffix: '/troubleshooting?topic=elasticsearch'
  },
  {
    id: 'check_domain',
    name: '域名 / DNS 排查',
    description: '解析链路与连通性诊断',
    category: 'troubleshoot',
    feature_key: 'feature.ai_diagnosis',
    pack_key: 'skillpack.domain',
    cli_topic: 'domain',
    icon: 'Link',
    keywords: ['dns', '域名', '解析'],
    route_suffix: '/troubleshooting?topic=domain'
  },
  {
    id: 'check_go_runtime',
    name: 'Go Runtime 排查',
    description: 'Go 应用 Pod 运行时与 K8s 工作负载诊断',
    category: 'troubleshoot',
    feature_key: 'feature.runtime_observe',
    pack_key: 'pack.runtime_observe',
    cli_topic: 'go_runtime',
    icon: 'Cpu',
    keywords: ['go', 'runtime', 'golang'],
    route_suffix: '/troubleshooting?topic=go_runtime'
  },
  {
    id: 'error_codes',
    name: '错误码查询',
    description: '部署与平台错误码根因库',
    category: 'troubleshoot',
    always_free: true,
    icon: 'Reading',
    keywords: ['错误码', 'error', '部署失败'],
    direct_route: '/help/error-codes'
  },
  {
    id: 'runtime_observe',
    name: 'Go Runtime 观测',
    description: '持续采集进程指标并上报平台',
    category: 'observe',
    feature_key: 'feature.runtime_observe',
    pack_key: 'pack.runtime_observe',
    direct_route: '/advanced/runtime-observe',
    icon: 'View',
    keywords: ['观测', '进程', 'runtime']
  },
  {
    id: 'prometheus',
    name: 'Prometheus',
    description: '监控服务端安装与配置',
    category: 'monitoring',
    feature_key: 'feature.monitoring',
    pack_key: 'pack.monitoring',
    direct_route: '/monitoring/prometheus',
    icon: 'TrendCharts',
    keywords: ['prometheus', '监控', 'metrics']
  },
  {
    id: 'exporters',
    name: 'Exporter 套件',
    description: 'Node / Redis / JMX / MongoDB / Blackbox Exporter',
    category: 'monitoring',
    feature_key: 'feature.monitoring',
    pack_key: 'pack.monitoring',
    direct_route: '/monitoring/node-exporter',
    icon: 'DataLine',
    keywords: ['exporter', 'node', 'jmx']
  },
  {
    id: 'backup',
    name: '备份与恢复',
    description: '平台数据备份与一键恢复',
    category: 'data',
    feature_key: 'feature.backup_performance',
    pack_key: 'pack.backup_performance',
    direct_route: '/advanced/backup-restore',
    icon: 'FolderOpened',
    keywords: ['备份', '恢复', 'backup']
  },
  {
    id: 'performance',
    name: '性能分析',
    description: '主机与应用性能报告',
    category: 'data',
    feature_key: 'feature.backup_performance',
    pack_key: 'pack.backup_performance',
    direct_route: '/advanced/performance-analysis',
    icon: 'Odometer',
    keywords: ['性能', '报告', '分析']
  },
  {
    id: 'skill_refinement',
    name: '技能增强审查',
    description: '诊断样本驱动的技能包增强队列（管理）',
    category: 'evolution',
    super_admin_only: true,
    icon: 'MagicStick',
    direct_route: '/ai-sre/skill-refinement'
  },
  {
    id: 'auto_iterations',
    name: '自动迭代',
    description: '平台能力缺口与 CLI 自动改进任务（管理）',
    category: 'evolution',
    super_admin_only: true,
    icon: 'Refresh',
    direct_route: '/auto-iterations'
  }
]

export const TROUBLESHOOT_TOPICS = CAPABILITY_CATALOG.filter((c) => c.cli_topic)

export const DELIVERY_CAPABILITIES = CAPABILITY_CATALOG.filter((c) => c.category === 'delivery')

export function categoryOrder(): CapabilityCategory[] {
  return ['delivery', 'troubleshoot', 'observe', 'monitoring', 'data', 'evolution']
}

export function catalogRoutePath(item: CatalogCapability): string {
  return item.direct_route || item.route_suffix || ''
}

export function capabilitiesForPack(packKey: string): CatalogCapability[] {
  return CAPABILITY_CATALOG.filter((c) => c.pack_key === packKey)
}
