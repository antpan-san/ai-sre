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

export interface CatalogCapability {
  id: string
  name: string
  description: string
  category: CapabilityCategory
  feature_key?: string
  pack_key?: string
  route_suffix?: string
  cli_topic?: string
  cli_hint?: string
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

export const CAPABILITY_CATALOG: CatalogCapability[] = [
  {
    id: 'k8s_delivery',
    name: 'Kubernetes 交付',
    description: '离线/在线 K8s 集群安装、恢复与卸载',
    category: 'delivery',
    feature_key: 'feature.k8s_delivery',
    pack_key: 'pack.k8s_delivery',
    route_suffix: '/workloads?tab=k8s'
  },
  {
    id: 'service_deploy',
    name: '应用服务部署',
    description: '中间件与应用服务安装、更新与卸载',
    category: 'delivery',
    feature_key: 'feature.node_ops',
    pack_key: 'pack.node_ops',
    route_suffix: '/workloads?tab=services'
  },
  {
    id: 'linux_hosts',
    name: 'Linux 主机管理',
    description: '主机上的服务状态与运维操作',
    category: 'delivery',
    feature_key: 'feature.node_ops',
    pack_key: 'pack.node_ops',
    route_suffix: '/workloads?tab=linux'
  },
  {
    id: 'init_tools',
    name: '节点初始化',
    description: '系统参数、时间同步、安全加固等初始化脚本',
    category: 'delivery',
    feature_key: 'feature.node_ops',
    pack_key: 'pack.node_ops',
    route_suffix: '/workloads?tab=init'
  },
  {
    id: 'proxy',
    name: '出口代理',
    description: '集群与主机访问外网的代理配置',
    category: 'delivery',
    feature_key: 'feature.node_ops',
    pack_key: 'pack.node_ops',
    route_suffix: '/workloads?tab=proxy'
  },
  {
    id: 'k8s_mirror',
    name: 'K8s 制品目录',
    description: '内网制品 manifest 与离线安装包索引',
    category: 'delivery',
    feature_key: 'feature.k8s_delivery',
    pack_key: 'pack.k8s_delivery',
    route_suffix: '/workloads?tab=mirror'
  },
  {
    id: 'check_redis',
    name: 'Redis 排查',
    description: '采集 + 本地规则 / AI 诊断 Redis 实例',
    category: 'troubleshoot',
    feature_key: 'feature.ai_diagnosis',
    pack_key: 'skillpack.redis',
    cli_topic: 'redis',
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
    route_suffix: '/troubleshooting?topic=go_runtime'
  },
  {
    id: 'error_codes',
    name: '错误码查询',
    description: '部署与平台错误码根因库',
    category: 'troubleshoot',
    always_free: true,
    route_suffix: '/help/error-codes'
  },
  {
    id: 'runtime_observe',
    name: 'Go Runtime 观测',
    description: '持续采集进程指标并上报平台',
    category: 'observe',
    feature_key: 'feature.runtime_observe',
    pack_key: 'pack.runtime_observe',
    route_suffix: '/advanced/runtime-observe'
  },
  {
    id: 'prometheus',
    name: 'Prometheus',
    description: '监控服务端安装与配置',
    category: 'monitoring',
    feature_key: 'feature.monitoring',
    pack_key: 'pack.monitoring',
    route_suffix: '/monitoring/prometheus'
  },
  {
    id: 'exporters',
    name: 'Exporter 套件',
    description: 'Node / Redis / JMX / MongoDB / Blackbox Exporter',
    category: 'monitoring',
    feature_key: 'feature.monitoring',
    pack_key: 'pack.monitoring',
    route_suffix: '/monitoring/node-exporter'
  },
  {
    id: 'backup',
    name: '备份与恢复',
    description: '平台数据备份与一键恢复',
    category: 'data',
    feature_key: 'feature.backup_performance',
    pack_key: 'pack.backup_performance',
    route_suffix: '/advanced/backup-restore'
  },
  {
    id: 'performance',
    name: '性能分析',
    description: '主机与应用性能报告',
    category: 'data',
    feature_key: 'feature.backup_performance',
    pack_key: 'pack.backup_performance',
    route_suffix: '/advanced/performance-analysis'
  },
  {
    id: 'skill_refinement',
    name: '技能增强审查',
    description: '诊断样本驱动的技能包增强队列（管理）',
    category: 'evolution',
    super_admin_only: true,
    route_suffix: '/ai-sre/skill-refinement'
  },
  {
    id: 'auto_iterations',
    name: '自动迭代',
    description: '平台能力缺口与 CLI 自动改进任务（管理）',
    category: 'evolution',
    super_admin_only: true,
    route_suffix: '/auto-iterations'
  }
]

export const TROUBLESHOOT_TOPICS = CAPABILITY_CATALOG.filter((c) => c.cli_topic)

export function categoryOrder(): CapabilityCategory[] {
  return ['delivery', 'troubleshoot', 'observe', 'monitoring', 'data', 'evolution']
}
