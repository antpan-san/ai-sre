import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { DocumentCopy, Upload, RefreshRight, Check, InfoFilled } from '@element-plus/icons-vue'
import { createServiceDeployment, updateServiceDeployment } from '../api/service'
import type { CreateServiceDeploymentResponse } from '../types/service'
import { copyTextToClipboard } from '../utils/clipboard'

export interface CatalogField {
  key: string
  label: string
  type: 'text' | 'number' | 'select' | 'switch' | 'textarea' | 'autocomplete'
  default: any
  options?: string[]
  min?: number
  max?: number
  rows?: number
  span?: 'quarter' | 'narrow' | 'half' | 'full'
  placeholder?: string
  tip?: string
  visibleIf?: () => boolean
}

export interface CatalogSection {
  key: string
  title: string
  hint?: string
  collapsible?: boolean
  defaultOpen?: boolean
  visibleIf?: () => boolean
  fields: CatalogField[]
  preview?: 'config'
}

export interface CatalogItem {
  key: string
  name: string
  description: string
  tags: string[]
  installMethods: string[]
  fields?: CatalogField[]
  sections?: CatalogSection[]
}

export function useServiceDeploy(options?: { fixedServiceKey?: string }) {
const form = reactive({
  service: '',
  osType: 'ubuntu-debian',
  installMethod: 'package',
  profile: 'default',
  params: {} as Record<string, any>
})

const isMethod = (m: string) => form.installMethod === m
const isOn = (k: string) => form.params[k] === true

const nginxSections: CatalogSection[] = [
  {
    key: 'basic',
    title: '基础',
    fields: [
      {
        key: 'version',
        label: 'Nginx 版本',
        type: 'autocomplete',
        default: '1.24.0',
        options: ['1.24.0', '1.25.5', '1.26.2', '1.27.1', 'stable', 'mainline', 'latest'],
        span: 'quarter',
        tip: 'docker 用作镜像 tag；binary 用于拼源码包 URL；package 仅记录'
      },
      { key: 'http_port', label: 'HTTP 监听端口', type: 'number', default: 80, span: 'quarter' },
      { key: 'server_name', label: 'server_name', type: 'text', default: '_', span: 'quarter', placeholder: '_ 或 example.com' },
      {
        key: 'user',
        label: 'nginx 运行用户',
        type: 'autocomplete',
        default: 'www-data',
        options: ['www-data', 'nginx', 'nobody', 'http'],
        span: 'quarter'
      },
      { key: 'pid_path', label: 'pid 文件路径', type: 'text', default: '/run/nginx.pid' },
      { key: 'worker_processes', label: 'worker_processes', type: 'autocomplete', default: 'auto', options: ['auto', '1', '2', '4', '8', '16'] },
      { key: 'worker_connections', label: 'worker_connections', type: 'number', default: 1024, min: 32, max: 1048576 },
      { key: 'worker_rlimit_nofile', label: 'worker_rlimit_nofile', type: 'number', default: 65535, min: 1024, max: 1048576 },
      {
        key: 'error_log_level',
        label: 'error_log 级别',
        type: 'select',
        default: 'warn',
        options: ['debug', 'info', 'notice', 'warn', 'error', 'crit', 'alert', 'emerg']
      },
      { key: 'daemon', label: 'daemon (后台运行)', type: 'switch', default: true, tip: 'docker 容器内通常应关闭' },
      { key: 'ipv6', label: '同时监听 IPv6 (::)', type: 'switch', default: false },
      { key: 'multi_accept', label: 'multi_accept', type: 'switch', default: true, tip: '一次接受所有新连接，提升高并发吞吐' },
      { key: 'accept_mutex', label: 'accept_mutex', type: 'switch', default: false, tip: 'reuseport 之外的旧惊群保护，关闭可降低延迟' },
      { key: 'server_tokens_hide', label: '隐藏 nginx 版本号 (server_tokens off)', type: 'switch', default: true }
    ]
  },
  {
    key: 'http',
    title: 'HTTP 调优',
    fields: [
      { key: 'keepalive_timeout', label: 'keepalive_timeout (秒)', type: 'number', default: 65, min: 0, max: 3600 },
      { key: 'client_max_body_size', label: 'client_max_body_size', type: 'text', default: '100m' },
      { key: 'sendfile', label: 'sendfile', type: 'switch', default: true },
      { key: 'tcp_nopush', label: 'tcp_nopush', type: 'switch', default: true },
      { key: 'tcp_nodelay', label: 'tcp_nodelay', type: 'switch', default: true },
      { key: 'gzip', label: '启用 gzip 压缩', type: 'switch', default: true },
      {
        key: 'gzip_min_length',
        label: 'gzip_min_length (字节)',
        type: 'number',
        default: 1024,
        min: 0,
        max: 1048576,
        visibleIf: () => isOn('gzip')
      },
      {
        key: 'gzip_types',
        label: 'gzip_types',
        type: 'text',
        default: 'text/plain text/css application/json application/javascript text/xml application/xml',
        span: 'full',
        visibleIf: () => isOn('gzip')
      }
    ]
  },
  {
    key: 'site',
    title: '站点 / 静态资源',
    fields: [
      { key: 'docroot', label: 'root 静态目录', type: 'text', default: '/var/www/html' },
      { key: 'index_files', label: 'index 文件', type: 'text', default: 'index.html index.htm' },
      { key: 'access_log', label: 'access_log 路径', type: 'text', default: '/var/log/nginx/access.log' },
      { key: 'error_log', label: 'error_log 路径', type: 'text', default: '/var/log/nginx/error.log' }
    ]
  },
  {
    key: 'proxy',
    title: '反向代理 (upstream)',
    hint: '关闭则只做静态站点',
    fields: [
      { key: 'reverse_proxy', label: '启用反向代理', type: 'switch', default: false },
      {
        key: 'lb_algorithm',
        label: '负载策略',
        type: 'select',
        default: 'round_robin',
        options: ['round_robin', 'least_conn', 'ip_hash'],
        visibleIf: () => isOn('reverse_proxy')
      },
      {
        key: 'upstreams',
        label: '后端列表（每行 host:port [weight=N]）',
        type: 'textarea',
        default: '10.0.0.1:8080\n10.0.0.2:8080',
        rows: 3,
        span: 'full',
        visibleIf: () => isOn('reverse_proxy')
      },
      {
        key: 'proxy_connect_timeout',
        label: 'proxy_connect_timeout (秒)',
        type: 'number',
        default: 5,
        visibleIf: () => isOn('reverse_proxy')
      },
      {
        key: 'proxy_read_timeout',
        label: 'proxy_read_timeout (秒)',
        type: 'number',
        default: 60,
        visibleIf: () => isOn('reverse_proxy')
      },
      {
        key: 'proxy_send_timeout',
        label: 'proxy_send_timeout (秒)',
        type: 'number',
        default: 60,
        visibleIf: () => isOn('reverse_proxy')
      }
    ]
  },
  {
    key: 'ssl',
    title: 'HTTPS / SSL',
    hint: '开启后渲染 443 server 块',
    fields: [
      { key: 'ssl', label: '启用 HTTPS', type: 'switch', default: false },
      {
        key: 'ssl_port',
        label: 'HTTPS 端口',
        type: 'number',
        default: 443,
        visibleIf: () => isOn('ssl')
      },
      {
        key: 'cert_path',
        label: 'ssl_certificate 路径',
        type: 'text',
        default: '/etc/nginx/ssl/server.crt',
        visibleIf: () => isOn('ssl')
      },
      {
        key: 'key_path',
        label: 'ssl_certificate_key 路径',
        type: 'text',
        default: '/etc/nginx/ssl/server.key',
        visibleIf: () => isOn('ssl')
      },
      {
        key: 'ssl_protocols',
        label: 'ssl_protocols',
        type: 'text',
        default: 'TLSv1.2 TLSv1.3',
        visibleIf: () => isOn('ssl')
      },
      {
        key: 'ssl_ciphers',
        label: 'ssl_ciphers',
        type: 'text',
        default: 'HIGH:!aNULL:!MD5',
        visibleIf: () => isOn('ssl')
      },
      {
        key: 'force_https_redirect',
        label: 'HTTP 强制跳转 HTTPS',
        type: 'switch',
        default: true,
        visibleIf: () => isOn('ssl')
      }
    ]
  },
  {
    key: 'install_path',
    title: '安装路径（仅二进制）',
    visibleIf: () => isMethod('binary'),
    fields: [
      { key: 'install_prefix', label: '--prefix 安装目录', type: 'text', default: '/usr/local/nginx' },
      { key: 'binary_url', label: '源码下载 URL', type: 'text', default: 'https://nginx.org/download/nginx-1.24.0.tar.gz', span: 'full' },
      { key: 'make_jobs', label: 'make 并发数 (-jN)', type: 'number', default: 4, min: 1, max: 64 },
      {
        key: 'configure_args',
        label: 'configure 额外参数',
        type: 'textarea',
        rows: 3,
        span: 'full',
        default: '--with-http_ssl_module --with-http_v2_module --with-http_realip_module --with-http_stub_status_module --with-http_gzip_static_module'
      }
    ]
  },
  {
    key: 'advanced',
    title: '高级 / 自定义',
    hint: '插入自定义指令到 http {} / server {}',
    collapsible: true,
    defaultOpen: false,
    fields: [
      {
        key: 'custom_http',
        label: '附加到 http {} 内的指令',
        type: 'textarea',
        rows: 4,
        span: 'full',
        default: '',
        placeholder: '示例：\nmap $http_upgrade $connection_upgrade { default upgrade; "" close; }'
      },
      {
        key: 'custom_server',
        label: '附加到 server {} 内的指令',
        type: 'textarea',
        rows: 4,
        span: 'full',
        default: '',
        placeholder: '示例：\nlocation /healthz { return 200 "ok"; }'
      }
    ]
  },
  {
    key: 'preview',
    title: '生成的 nginx.conf 预览',
    hint: '只读 / 由上方参数实时渲染',
    collapsible: true,
    defaultOpen: false,
    preview: 'config',
    fields: []
  }
]

const haproxySections: CatalogSection[] = [
  {
    key: 'basic',
    title: '基础',
    fields: [
      { key: 'version', label: 'HAProxy 版本', type: 'autocomplete', default: '2.8', options: ['2.4', '2.6', '2.8', '3.0', 'lts', 'latest'] },
      { key: 'frontend_port', label: '前端端口', type: 'number', default: 80 },
      { key: 'mode', label: '代理模式', type: 'select', default: 'http', options: ['http', 'tcp'] },
      { key: 'maxconn', label: 'global maxconn', type: 'number', default: 4096, min: 1, max: 1048576 },
      { key: 'stats_enabled', label: '启用 stats 页面', type: 'switch', default: true },
      { key: 'httpclose', label: 'option httpclose', type: 'switch', default: false, visibleIf: () => form.params.mode === 'http' }
    ]
  },
  {
    key: 'backend',
    title: '后端与健康检查',
    fields: [
      { key: 'algorithm', label: 'balance', type: 'select', default: 'roundrobin', options: ['roundrobin', 'leastconn', 'source'] },
      { key: 'backends', label: '后端列表（每行 host:port）', type: 'textarea', default: '10.0.0.1:8080\n10.0.0.2:8080', rows: 4, span: 'full' },
      { key: 'check_enabled', label: 'server check', type: 'switch', default: true },
      { key: 'httpchk_enabled', label: 'HTTP 健康检查', type: 'switch', default: false, visibleIf: () => form.params.mode === 'http' },
      { key: 'httpchk_path', label: 'option httpchk 路径', type: 'text', default: 'GET /health', visibleIf: () => form.params.mode === 'http' && isOn('httpchk_enabled') }
    ]
  },
  {
    key: 'timeouts',
    title: '超时',
    fields: [
      { key: 'timeout_connect', label: 'timeout connect', type: 'text', default: '5s' },
      { key: 'timeout_client', label: 'timeout client', type: 'text', default: '30s' },
      { key: 'timeout_server', label: 'timeout server', type: 'text', default: '30s' },
      { key: 'stats_port', label: 'stats 端口', type: 'number', default: 8404, visibleIf: () => isOn('stats_enabled') }
    ]
  }
]

const redisSections: CatalogSection[] = [
  {
    key: 'basic',
    title: '基础',
    fields: [
      { key: 'version', label: 'Redis 版本', type: 'autocomplete', default: '7.2', options: ['6.0', '6.2', '7.0', '7.2', '7.4', 'latest'] },
      { key: 'port', label: '端口', type: 'number', default: 6379 },
      { key: 'bind', label: 'bind 地址', type: 'text', default: '0.0.0.0' },
      { key: 'databases', label: 'databases', type: 'number', default: 16, min: 1, max: 1024 },
      { key: 'dir', label: '数据目录 dir', type: 'text', default: '/var/lib/redis' },
      { key: 'requirepass', label: 'requirepass（可空）', type: 'text', default: '' },
      { key: 'protected_mode', label: 'protected-mode', type: 'switch', default: true },
      { key: 'supervised_systemd', label: 'supervised systemd', type: 'switch', default: true }
    ]
  },
  {
    key: 'memory',
    title: '内存与连接',
    fields: [
      { key: 'maxmemory', label: 'maxmemory', type: 'text', default: '512mb' },
      { key: 'maxmemory_policy', label: 'maxmemory-policy', type: 'select', default: 'allkeys-lru', options: ['noeviction', 'allkeys-lru', 'volatile-lru', 'allkeys-lfu', 'volatile-ttl'] },
      { key: 'timeout', label: 'timeout (秒)', type: 'number', default: 0, min: 0, max: 86400 },
      { key: 'tcp_keepalive', label: 'tcp-keepalive (秒)', type: 'number', default: 300, min: 0, max: 86400 }
    ]
  },
  {
    key: 'persistence',
    title: '持久化',
    fields: [
      { key: 'rdb_enabled', label: '启用 RDB save', type: 'switch', default: true },
      { key: 'dbfilename', label: 'dbfilename', type: 'text', default: 'dump.rdb', visibleIf: () => isOn('rdb_enabled') },
      { key: 'appendonly', label: 'appendonly (AOF)', type: 'switch', default: false },
      { key: 'appendfsync', label: 'appendfsync', type: 'select', default: 'everysec', options: ['always', 'everysec', 'no'], visibleIf: () => isOn('appendonly') }
    ]
  }
]

const kafkaSections: CatalogSection[] = [
  {
    key: 'basic',
    title: '基础（Docker）',
    fields: [
      { key: 'version', label: 'Kafka 版本', type: 'autocomplete', default: '3.6', options: ['3.4', '3.5', '3.6', '3.7', 'latest'] },
      { key: 'port', label: 'broker 端口', type: 'number', default: 9092 },
      { key: 'broker_id', label: 'broker.id', type: 'number', default: 1, min: 0, max: 4096 },
      { key: 'zookeeper', label: 'ZooKeeper 地址', type: 'text', default: 'localhost:2181' },
      { key: 'auto_create_topics', label: 'auto.create.topics.enable', type: 'switch', default: false }
    ]
  },
  {
    key: 'topic_defaults',
    title: 'Topic 默认值与保留策略',
    fields: [
      { key: 'num_partitions', label: 'num.partitions', type: 'number', default: 3, min: 1, max: 10000 },
      { key: 'default_replication_factor', label: 'default.replication.factor', type: 'number', default: 1, min: 1, max: 10 },
      { key: 'log_retention_hours', label: 'log.retention.hours', type: 'number', default: 168, min: 1, max: 87600 },
      { key: 'log_segment_bytes', label: 'log.segment.bytes', type: 'number', default: 1073741824, min: 1048576 },
      { key: 'log_dir', label: 'log.dirs', type: 'text', default: '/var/lib/kafka/logs', span: 'half' }
    ]
  }
]

const mysqlSections: CatalogSection[] = [
  {
    key: 'basic',
    title: '基础',
    fields: [
      { key: 'version', label: 'MySQL 版本', type: 'autocomplete', default: '8.0', options: ['5.7', '8.0', '8.4', 'latest'] },
      { key: 'port', label: '端口', type: 'number', default: 3306 },
      { key: 'root_password', label: 'root 密码', type: 'text', default: 'changeme' },
      { key: 'datadir', label: '数据目录', type: 'text', default: '/var/lib/mysql' },
      { key: 'bind_address', label: 'bind-address', type: 'text', default: '0.0.0.0' },
      { key: 'skip_name_resolve', label: 'skip-name-resolve', type: 'switch', default: true }
    ]
  },
  {
    key: 'server',
    title: '服务参数',
    fields: [
      { key: 'charset', label: 'character-set-server', type: 'text', default: 'utf8mb4' },
      { key: 'collation', label: 'collation-server', type: 'text', default: 'utf8mb4_0900_ai_ci' },
      { key: 'max_connections', label: 'max_connections', type: 'number', default: 500, min: 1, max: 100000 },
      { key: 'innodb_buffer_pool_size', label: 'innodb_buffer_pool_size', type: 'text', default: '512M' },
      { key: 'slow_query_log', label: 'slow_query_log', type: 'switch', default: true },
      { key: 'long_query_time', label: 'long_query_time (秒)', type: 'number', default: 2, min: 0, max: 3600, visibleIf: () => isOn('slow_query_log') }
    ]
  }
]

const elasticsearchSections: CatalogSection[] = [
  {
    key: 'basic',
    title: '基础（生产已避坑：vm.max_map_count、ulimit、heap、wait-ready）',
    fields: [
      { key: 'version', label: 'Elasticsearch 版本', type: 'autocomplete', default: '8.13.4', options: ['7.17.21', '8.11.4', '8.12.2', '8.13.4', '8.14.1', '8.15.0', 'latest'], span: 'quarter' },
      { key: 'http_port', label: 'HTTP 端口', type: 'number', default: 9200, span: 'quarter' },
      { key: 'transport_port', label: '集群传输端口', type: 'number', default: 9300, span: 'quarter' },
      { key: 'cluster_name', label: 'cluster.name', type: 'text', default: 'opsfleet-es', span: 'quarter' },
      { key: 'node_name', label: 'node.name（留空使用 hostname）', type: 'text', default: '', span: 'half', placeholder: '${HOSTNAME}' },
      { key: 'network_host', label: 'network.host', type: 'text', default: '0.0.0.0', span: 'quarter' },
      { key: 'heap_size', label: 'JVM 堆大小（建议 ≤ 物理内存 50% 且 ≤ 30g）', type: 'autocomplete', default: '1g', options: ['512m', '1g', '2g', '4g', '8g', '16g', '30g'], span: 'quarter' },
      { key: 'path_data', label: 'path.data 目录', type: 'text', default: '/var/lib/elasticsearch', span: 'half' },
      { key: 'path_logs', label: 'path.logs 目录', type: 'text', default: '/var/log/elasticsearch', span: 'half' },
      { key: 'vm_max_map_count_setup', label: '自动调高 vm.max_map_count（推荐保持开启）', type: 'switch', default: true, tip: '默认 65530 太小会导致 ES 启动失败，ai-sre 写 sysctl.d 并立即生效' },
      { key: 'bootstrap_memory_lock', label: 'bootstrap.memory_lock', type: 'switch', default: false, tip: '锁定内存避免 swap；启用前确保系统配 LimitMEMLOCK=infinity（systemd 已自动）' }
    ]
  },
  {
    key: 'cluster',
    title: '集群拓扑',
    hint: '默认 single-node；多节点需填写 seed_hosts 与 initial_master_nodes',
    fields: [
      { key: 'discovery_type', label: 'discovery 模式', type: 'select', default: 'single-node', options: ['single-node', 'multi-node'] },
      {
        key: 'seed_hosts',
        label: 'discovery.seed_hosts（逗号或换行分隔 host:port）',
        type: 'textarea',
        rows: 2,
        span: 'full',
        default: '',
        placeholder: '示例：10.0.0.11:9300, 10.0.0.12:9300, 10.0.0.13:9300',
        visibleIf: () => form.params.discovery_type === 'multi-node'
      },
      {
        key: 'initial_master_nodes',
        label: 'cluster.initial_master_nodes（节点 node.name 列表）',
        type: 'textarea',
        rows: 2,
        span: 'full',
        default: '',
        placeholder: '示例：es-1, es-2, es-3',
        visibleIf: () => form.params.discovery_type === 'multi-node'
      }
    ]
  },
  {
    key: 'security',
    title: '安全（xpack）',
    hint: '8.x 默认开启证书+密码；PoC 默认关闭，生产再打开',
    collapsible: true,
    defaultOpen: false,
    fields: [
      { key: 'xpack_security', label: '启用 xpack.security', type: 'switch', default: false, tip: '关闭后 9200 直接 http；开启后 ai-sre 探活会自动改 https + -k' },
      { key: 'xpack_user', label: 'xpack 用户名（探活用）', type: 'text', default: 'elastic', visibleIf: () => isOn('xpack_security') },
      { key: 'xpack_password', label: 'xpack 密码（用于 ai-sre 健康检查）', type: 'text', default: '', visibleIf: () => isOn('xpack_security'), placeholder: '留空时探活只 -k 不带凭据' }
    ]
  },
  {
    key: 'binary_install',
    title: '二进制包安装',
    hint: '官方 Linux tarball；解压到 prefix，ES_PATH_CONF=prefix/config，一键后 wait-ready 直至集群可用',
    visibleIf: () => isMethod('binary'),
    collapsible: true,
    defaultOpen: true,
    fields: [
      {
        key: 'install_prefix',
        label: '安装目录（解压根路径）',
        type: 'text',
        default: '/opt/elasticsearch',
        span: 'full',
        tip: '与包安装隔离：使用独立目录与自管 systemd 单元，避免与 apt 包混用同一实例'
      },
      {
        key: 'binary_url',
        label: 'tarball 完整下载 URL（可选）',
        type: 'textarea',
        rows: 2,
        span: 'full',
        default: '',
        placeholder:
          '留空则按「版本」与本机 CPU 架构自动选择 https://artifacts.elastic.co/downloads/elasticsearch/elasticsearch-<ver>-linux-{x86_64|aarch64}.tar.gz'
      }
    ]
  }
]

const postgresqlSections: CatalogSection[] = [
  {
    key: 'basic',
    title: '基础',
    fields: [
      { key: 'version', label: 'PostgreSQL 版本', type: 'autocomplete', default: '16', options: ['13', '14', '15', '16', '17', 'latest'] },
      { key: 'port', label: '端口', type: 'number', default: 5432 },
      { key: 'password', label: 'postgres 密码', type: 'text', default: 'changeme' },
      { key: 'datadir', label: 'PGDATA 目录', type: 'text', default: '/var/lib/postgresql/data' },
      { key: 'listen_addresses', label: 'listen_addresses', type: 'text', default: '*' },
      { key: 'trust_local_network', label: '允许网段密码访问', type: 'switch', default: true }
    ]
  },
  {
    key: 'tuning',
    title: '连接与内存',
    fields: [
      { key: 'max_connections', label: 'max_connections', type: 'number', default: 200, min: 1, max: 100000 },
      { key: 'shared_buffers', label: 'shared_buffers', type: 'text', default: '512MB' },
      { key: 'work_mem', label: 'work_mem', type: 'text', default: '8MB' },
      { key: 'wal_level', label: 'wal_level', type: 'select', default: 'replica', options: ['minimal', 'replica', 'logical'] },
      { key: 'log_min_duration_statement', label: '慢 SQL 阈值(ms, -1关闭)', type: 'number', default: 1000, min: -1, max: 2147483647 }
    ]
  }
]

const catalog: CatalogItem[] = [
  {
    key: 'nginx',
    name: 'Nginx',
    description: 'Web 服务器 / 反向代理',
    tags: ['gateway', 'web'],
    installMethods: ['package', 'docker', 'binary'],
    sections: nginxSections
  },
  {
    key: 'haproxy',
    name: 'HAProxy',
    description: '高可用 4/7 层负载均衡',
    tags: ['gateway', 'lb'],
    installMethods: ['package', 'docker'],
    sections: haproxySections
  },
  {
    key: 'redis',
    name: 'Redis',
    description: '内存数据库 / 缓存',
    tags: ['cache', 'kv'],
    installMethods: ['package', 'docker'],
    sections: redisSections
  },
  {
    key: 'kafka',
    name: 'Kafka',
    description: '分布式消息队列',
    tags: ['mq'],
    installMethods: ['docker'],
    sections: kafkaSections
  },
  {
    key: 'mysql',
    name: 'MySQL',
    description: '关系型数据库',
    tags: ['db', 'sql'],
    installMethods: ['package', 'docker'],
    sections: mysqlSections
  },
  {
    key: 'postgresql',
    name: 'PostgreSQL',
    description: '关系型数据库',
    tags: ['db', 'sql'],
    installMethods: ['package', 'docker'],
    sections: postgresqlSections
  },
  {
    key: 'elasticsearch',
    name: 'Elasticsearch',
    description: '分布式搜索 / 分析引擎',
    tags: ['search', 'analytics'],
    installMethods: ['package', 'docker', 'binary'],
    sections: elasticsearchSections
  }
]

const osTypeOptions = [
  { label: 'Ubuntu / Debian', value: 'ubuntu-debian' },
  { label: 'CentOS / Rocky / RHEL', value: 'rhel-family' },
  { label: 'openEuler', value: 'openeuler' },
  { label: 'Kylin', value: 'kylin' }
]

const installMethodLabels: Record<string, string> = {
  package: '系统包（apt/yum/dnf 自动适配）',
  docker: 'Docker 容器',
  binary: '官方二进制包（tarball + systemd）'
}

const profileCatalog: Record<string, Array<{ label: string; value: string }>> = {
  nginx: [
    { label: '静态站点', value: 'static' },
    { label: '反向代理', value: 'reverse_proxy' },
    { label: 'HTTPS 站点', value: 'https' }
  ],
  haproxy: [{ label: '负载均衡', value: 'load_balancer' }],
  redis: [
    { label: '单机缓存', value: 'standalone_cache' },
    { label: '持久化单机', value: 'standalone_persistent' }
  ],
  kafka: [{ label: '单 Broker 测试环境', value: 'single_broker' }],
  mysql: [{ label: '单机数据库', value: 'standalone_db' }],
  postgresql: [{ label: '单机数据库', value: 'standalone_db' }],
  elasticsearch: [
    { label: '单节点（PoC / 开发）', value: 'single_node' },
    { label: '多节点集群（生产）', value: 'multi_node_cluster' }
  ]
}

const selected = computed<CatalogItem | null>(() => catalog.find(c => c.key === form.service) || null)

const availableInstallMethods = computed(() => {
  const methods = selected.value?.installMethods || ['package']
  return methods.map(m => ({ value: m, label: installMethodLabels[m] || m }))
})

const profileOptions = computed(() => profileCatalog[form.service] || [{ label: '默认', value: 'default' }])

const selectedProfile = computed(() => form.profile || profileOptions.value[0]?.value || 'default')

const allSections = computed<CatalogSection[]>(() => {
  if (!selected.value) return []
  if (selected.value.sections) return selected.value.sections
  return [{ key: 'config', title: '参数配置', fields: selected.value.fields || [] }]
})

const visibleSections = computed(() =>
  allSections.value.filter(sec => !sec.visibleIf || sec.visibleIf())
)

const regularSections = computed(() => visibleSections.value.filter(s => !s.collapsible && s.key === 'basic'))
const collapsibleSections = computed(() => visibleSections.value.filter(s => s.collapsible || s.key !== 'basic'))

const activeCollapseSections = ref<string[]>([])

const visibleFields = (fields: CatalogField[]) =>
  fields.filter(f => !f.visibleIf || f.visibleIf())

const normalFields = (fields: CatalogField[]) =>
  visibleFields(fields).filter(f => f.type !== 'switch')

const switchFields = (fields: CatalogField[]) =>
  visibleFields(fields).filter(f => f.type === 'switch')

const colMd = (f: CatalogField) => {
  if (f.type === 'textarea' || f.span === 'full') return 24
  if (f.span === 'half') return 12
  if (f.span === 'quarter') return 6
  return 8
}

const sectionNormalColMd = (f: CatalogField) => {
  if (f.type === 'textarea' || f.span === 'full') return 24
  if (f.span === 'half') return 12
  return 8
}

const previewVisible = ref(false)
const activeTab = ref<'bash' | 'cli'>('bash')
const generating = ref(false)
const submittingUpdate = ref(false)
const generatedDeployment = ref<CreateServiceDeploymentResponse | null>(null)
const savedDeploymentSnapshot = ref('')
const curlCommand = computed(() => generatedDeployment.value?.curlCommand || '')
const aiSreInstallCommand = computed(() => generatedDeployment.value?.aiSreCommand || '')
const updatableServices = new Set(['nginx', 'elasticsearch'])
const aiSreUpdateCommand = computed(() => {
  if (generatedDeployment.value?.aiSreUpdateCommand) return generatedDeployment.value.aiSreUpdateCommand
  if (form.service && updatableServices.has(form.service)) return `sudo ai-sre ops ${form.service} update`
  return ''
})
const aiSreUninstallCommand = computed(() => {
  if (generatedDeployment.value?.aiSreUninstallCommand) return generatedDeployment.value.aiSreUninstallCommand
  if (form.service) return `sudo ai-sre ops service uninstall ${form.service}`
  return ''
})
const aiSreRecoverCommand = computed(() => {
  if (generatedDeployment.value?.aiSreRecoverCommand) return generatedDeployment.value.aiSreRecoverCommand
  if (form.service && ['redis', 'mysql', 'postgresql', 'kafka', 'haproxy'].includes(form.service)) {
    return `sudo ai-sre ops service recover ${form.service}`
  }
  return ''
})

const buildDeploymentPayload = () => ({
  service: form.service,
  profile: selectedProfile.value,
  install_method: form.installMethod,
  version: String(form.params.version || ''),
  params: {
    ...form.params,
    osType: form.osType
  }
})

const stableStringify = (value: any): string => {
  if (Array.isArray(value)) return `[${value.map(stableStringify).join(',')}]`
  if (value && typeof value === 'object') {
    return `{${Object.keys(value).sort().map(key => `${JSON.stringify(key)}:${stableStringify(value[key])}`).join(',')}}`
  }
  return JSON.stringify(value) ?? 'null'
}

const currentDeploymentSnapshot = computed(() => stableStringify(buildDeploymentPayload()))
const deploymentDirty = computed(() =>
  !!generatedDeployment.value && savedDeploymentSnapshot.value !== currentDeploymentSnapshot.value
)
const canSubmitUpdate = computed(() =>
  !!generatedDeployment.value && updatableServices.has(form.service) && deploymentDirty.value && !submittingUpdate.value
)
const deploymentStatusDescription = computed(() => {
  if (!generatedDeployment.value) return ''
  const svc = selected.value?.name || form.service
  const cmd = aiSreUpdateCommand.value
  if (deploymentDirty.value) {
    return cmd
      ? `页面配置有新修改，点击"提交配置变更"后，目标机执行 ${cmd} 即可拉取最新配置并重启 ${svc} 生效。`
      : `页面配置有新修改，但当前服务暂不支持 ai-sre 远程更新，请重新生成部署脚本并在目标机执行。`
  }
  return cmd
    ? `状态：${generatedDeployment.value.status}。当前页面配置已保存；后续修改后可提交，并在目标机执行 ${cmd} 更新 ${svc}。`
    : `状态：${generatedDeployment.value.status}。当前页面配置已保存。`
})

const seedParams = (item: CatalogItem) => {
  const out: Record<string, any> = {}
  const sections = item.sections || (item.fields ? [{ key: 'default', title: '', fields: item.fields }] as CatalogSection[] : [])
  sections.forEach(sec => sec.fields.forEach(f => { out[f.key] = f.default }))
  return out
}

const selectService = (key: string) => {
  form.service = key
  const item = catalog.find(c => c.key === key)
  if (!item) return
  generatedDeployment.value = null
  savedDeploymentSnapshot.value = ''
  form.params = seedParams(item)
  if (!item.installMethods.includes(form.installMethod)) {
    form.installMethod = item.installMethods[0] || form.installMethod
  }
  form.profile = profileOptions.value[0]?.value || 'default'
  activeCollapseSections.value = (item.sections || [])
    .filter(s => s.collapsible && s.defaultOpen)
    .map(s => s.key)
}

const onGenerate = async () => {
  if (!selected.value) {
    ElMessage.warning('请先选择基础服务')
    return
  }
  generating.value = true
  try {
    generatedDeployment.value = await createServiceDeployment(buildDeploymentPayload())
    savedDeploymentSnapshot.value = currentDeploymentSnapshot.value
    activeTab.value = 'bash'
    previewVisible.value = true
  } finally {
    generating.value = false
  }
}

const onSubmitUpdate = async () => {
  if (!generatedDeployment.value || !updatableServices.has(form.service)) {
    ElMessage.warning('请先生成支持远程更新的服务部署任务（当前支持：Nginx、Elasticsearch）')
    return
  }
  if (!deploymentDirty.value) {
    ElMessage.info('当前配置已保存，无需提交')
    return
  }
  submittingUpdate.value = true
  try {
    const res = await updateServiceDeployment(generatedDeployment.value.deploymentId, {
      ...buildDeploymentPayload(),
      token: generatedDeployment.value.token
    })
    generatedDeployment.value = {
      ...generatedDeployment.value,
      ...res,
      token: generatedDeployment.value.token,
      curlCommand: generatedDeployment.value.curlCommand,
      aiSreCommand: generatedDeployment.value.aiSreCommand
    }
    savedDeploymentSnapshot.value = currentDeploymentSnapshot.value
    ElMessage.success(`配置已提交，目标机执行 ${aiSreUpdateCommand.value} 后生效`)
  } finally {
    submittingUpdate.value = false
  }
}

const onReset = () => {
  if (!selected.value) return
  form.params = seedParams(selected.value)
  form.installMethod = selected.value.installMethods[0] || form.installMethod
  form.profile = profileOptions.value[0]?.value || 'default'
  activeCollapseSections.value = (selected.value.sections || [])
    .filter(s => s.collapsible && s.defaultOpen)
    .map(s => s.key)
}

const defaultBashFilename = computed(() => `install-${form.service || 'service'}.sh`)

const pkgInstall = (os: string, pkgs: string[]) => {
  const list = pkgs.join(' ')
  switch (os) {
    case 'ubuntu-debian':
      return `sudo apt-get update -y\nsudo DEBIAN_FRONTEND=noninteractive apt-get install -y ${list}`
    default:
      return `(command -v dnf >/dev/null && sudo dnf install -y ${list}) || sudo yum install -y ${list}`
  }
}

const dockerRun = (name: string, image: string, ports: string[], envs: string[] = [], volumes: string[] = []) => {
  const portArgs = ports.map(p => `-p ${p}`).join(' ')
  const envArgs = envs.map(e => `-e ${e}`).join(' ')
  const volArgs = volumes.map(v => `-v ${v}`).join(' ')
  return [
    'sudo docker rm -f ' + name + ' 2>/dev/null || true',
    `sudo docker run -d --name ${name} --restart=always ${portArgs} ${envArgs} ${volArgs} ${image}`.replace(/\s+/g, ' ').trim()
  ].join('\n')
}

const indent = (text: string, pad: string) =>
  text.split('\n').map(l => l.length ? pad + l : l).join('\n')

const renderNginxConf = () => {
  const p = form.params
  const lines: string[] = []
  if (p.user) lines.push(`user ${p.user};`)
  lines.push(`worker_processes ${p.worker_processes || 'auto'};`)
  if (p.worker_rlimit_nofile) lines.push(`worker_rlimit_nofile ${p.worker_rlimit_nofile};`)
  if (p.pid_path) lines.push(`pid ${p.pid_path};`)
  if (p.daemon === false) lines.push(`daemon off;`)
  lines.push(`events {`)
  lines.push(`    worker_connections ${p.worker_connections || 1024};`)
  lines.push(`    multi_accept ${p.multi_accept ? 'on' : 'off'};`)
  lines.push(`    accept_mutex ${p.accept_mutex ? 'on' : 'off'};`)
  lines.push(`}`)
  lines.push(``)
  lines.push(`http {`)
  lines.push(`    include       mime.types;`)
  lines.push(`    default_type  application/octet-stream;`)
  lines.push(``)
  lines.push(`    access_log ${p.access_log || '/var/log/nginx/access.log'};`)
  lines.push(`    error_log  ${p.error_log || '/var/log/nginx/error.log'} ${p.error_log_level || 'warn'};`)
  lines.push(``)
  lines.push(`    sendfile        ${p.sendfile === false ? 'off' : 'on'};`)
  lines.push(`    tcp_nopush      ${p.tcp_nopush === false ? 'off' : 'on'};`)
  lines.push(`    tcp_nodelay     ${p.tcp_nodelay === false ? 'off' : 'on'};`)
  lines.push(`    keepalive_timeout ${p.keepalive_timeout ?? 65};`)
  lines.push(`    client_max_body_size ${p.client_max_body_size || '100m'};`)
  lines.push(`    server_tokens   ${p.server_tokens_hide === false ? 'on' : 'off'};`)

  if (p.gzip) {
    lines.push(``)
    lines.push(`    gzip on;`)
    lines.push(`    gzip_min_length ${p.gzip_min_length ?? 1024};`)
    lines.push(`    gzip_proxied any;`)
    lines.push(`    gzip_types ${p.gzip_types || 'text/plain text/css application/json application/javascript text/xml application/xml'};`)
  }

  if (p.custom_http && String(p.custom_http).trim()) {
    lines.push(``)
    lines.push(`    # ===== custom_http =====`)
    lines.push(indent(String(p.custom_http).trim(), '    '))
  }

  if (p.reverse_proxy) {
    const ups = String(p.upstreams || '')
      .split('\n')
      .map(s => s.trim())
      .filter(Boolean)
    if (ups.length) {
      lines.push(``)
      lines.push(`    upstream backend_app {`)
      if (p.lb_algorithm === 'least_conn') lines.push(`        least_conn;`)
      if (p.lb_algorithm === 'ip_hash') lines.push(`        ip_hash;`)
      ups.forEach(u => lines.push(`        server ${u};`))
      lines.push(`    }`)
    }
  }

  const httpListen = `${p.http_port || 80}${p.ipv6 ? '' : ''}`
  lines.push(``)
  lines.push(`    server {`)
  lines.push(`        listen ${httpListen};`)
  if (p.ipv6) lines.push(`        listen [::]:${p.http_port || 80};`)
  lines.push(`        server_name ${p.server_name || '_'};`)

  if (p.ssl && p.force_https_redirect) {
    lines.push(`        return 301 https://$host$request_uri;`)
  } else {
    lines.push(`        root ${p.docroot || '/var/www/html'};`)
    lines.push(`        index ${p.index_files || 'index.html index.htm'};`)
    lines.push(``)
    if (p.reverse_proxy) {
      lines.push(`        location / {`)
      lines.push(`            proxy_pass http://backend_app;`)
      lines.push(`            proxy_http_version 1.1;`)
      lines.push(`            proxy_set_header Host $host;`)
      lines.push(`            proxy_set_header X-Real-IP $remote_addr;`)
      lines.push(`            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;`)
      lines.push(`            proxy_set_header X-Forwarded-Proto $scheme;`)
      lines.push(`            proxy_connect_timeout ${p.proxy_connect_timeout ?? 5}s;`)
      lines.push(`            proxy_send_timeout ${p.proxy_send_timeout ?? 60}s;`)
      lines.push(`            proxy_read_timeout ${p.proxy_read_timeout ?? 60}s;`)
      lines.push(`        }`)
    } else {
      lines.push(`        location / {`)
      lines.push(`            try_files $uri $uri/ =404;`)
      lines.push(`        }`)
    }
    if (p.custom_server && String(p.custom_server).trim()) {
      lines.push(``)
      lines.push(`        # ===== custom_server =====`)
      lines.push(indent(String(p.custom_server).trim(), '        '))
    }
  }
  lines.push(`    }`)

  if (p.ssl) {
    lines.push(``)
    lines.push(`    server {`)
    lines.push(`        listen ${p.ssl_port || 443} ssl http2;`)
    if (p.ipv6) lines.push(`        listen [::]:${p.ssl_port || 443} ssl http2;`)
    lines.push(`        server_name ${p.server_name || '_'};`)
    lines.push(`        ssl_certificate     ${p.cert_path || '/etc/nginx/ssl/server.crt'};`)
    lines.push(`        ssl_certificate_key ${p.key_path || '/etc/nginx/ssl/server.key'};`)
    lines.push(`        ssl_protocols       ${p.ssl_protocols || 'TLSv1.2 TLSv1.3'};`)
    lines.push(`        ssl_ciphers         ${p.ssl_ciphers || 'HIGH:!aNULL:!MD5'};`)
    lines.push(`        ssl_prefer_server_ciphers on;`)
    lines.push(``)
    lines.push(`        root ${p.docroot || '/var/www/html'};`)
    lines.push(`        index ${p.index_files || 'index.html index.htm'};`)
    if (p.reverse_proxy) {
      lines.push(``)
      lines.push(`        location / {`)
      lines.push(`            proxy_pass http://backend_app;`)
      lines.push(`            proxy_http_version 1.1;`)
      lines.push(`            proxy_set_header Host $host;`)
      lines.push(`            proxy_set_header X-Real-IP $remote_addr;`)
      lines.push(`            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;`)
      lines.push(`            proxy_set_header X-Forwarded-Proto $scheme;`)
      lines.push(`            proxy_connect_timeout ${p.proxy_connect_timeout ?? 5}s;`)
      lines.push(`            proxy_send_timeout ${p.proxy_send_timeout ?? 60}s;`)
      lines.push(`            proxy_read_timeout ${p.proxy_read_timeout ?? 60}s;`)
      lines.push(`        }`)
    } else {
      lines.push(``)
      lines.push(`        location / {`)
      lines.push(`            try_files $uri $uri/ =404;`)
      lines.push(`        }`)
    }
    if (p.custom_server && String(p.custom_server).trim()) {
      lines.push(``)
      lines.push(`        # ===== custom_server (https) =====`)
      lines.push(indent(String(p.custom_server).trim(), '        '))
    }
    lines.push(`    }`)
  }

  lines.push(`}`)
  return lines.join('\n')
}

const confPreview = computed(() => {
  if (form.service === 'nginx') return renderNginxConf()
  return ''
})

const isSemver = (v: string) => /^\d+\.\d+(\.\d+)?$/.test(v)

const dockerImageTag = (image: string, version: string, fallback = 'latest') => {
  const v = String(version || '').trim()
  return v ? `${image}:${v}` : `${image}:${fallback}`
}

const buildNginx = () => {
  const p = form.params
  const conf = renderNginxConf()
  const writeConf = (path: string) => `sudo install -m 0755 -d "$(dirname ${path})"
sudo bash -c 'cat >${path} <<"NGINXCONF"
${conf}
NGINXCONF'`

  const docroot = p.docroot || '/var/www/html'
  const ports = [`${p.http_port || 80}:${p.http_port || 80}`]
  if (p.ssl) ports.push(`${p.ssl_port || 443}:${p.ssl_port || 443}`)

  if (form.installMethod === 'docker') {
    return `${writeConf('/etc/nginx/nginx.conf')}
sudo install -m 0755 -d ${docroot}
${dockerRun(
      'nginx',
      dockerImageTag('nginx', p.version || 'stable', 'stable'),
      ports,
      [],
      ['/etc/nginx/nginx.conf:/etc/nginx/nginx.conf:ro', `${docroot}:${docroot}:ro`]
    )}
sudo ss -lntp | grep -E ":${p.http_port || 80}\\b" || true`
  }

  if (form.installMethod === 'binary') {
    const prefix = p.install_prefix || '/usr/local/nginx'
    const ver = String(p.version || '').trim()
    const url = (p.binary_url && String(p.binary_url).trim())
      || (isSemver(ver) ? `https://nginx.org/download/nginx-${ver}.tar.gz` : 'https://nginx.org/download/nginx-1.24.0.tar.gz')
    const extra = (p.configure_args || '').replace(/\n+/g, ' ').trim()
    return `${pkgInstall(form.osType, ['build-essential', 'libpcre2-dev', 'zlib1g-dev', 'libssl-dev', 'wget', 'tar'])}
sudo install -m 0755 -d /tmp/nginx-build
cd /tmp/nginx-build
sudo wget -O nginx.tar.gz '${url}'
sudo tar -xf nginx.tar.gz --strip-components=1
sudo ./configure --prefix=${prefix} \\
  --conf-path=${prefix}/conf/nginx.conf \\
  --sbin-path=${prefix}/sbin/nginx \\
  --pid-path=${prefix}/logs/nginx.pid \\
  --error-log-path=${prefix}/logs/error.log \\
  --http-log-path=${prefix}/logs/access.log \\
  ${extra}
sudo make -j${p.make_jobs || 4}
sudo make install
${writeConf(`${prefix}/conf/nginx.conf`)}
sudo install -m 0755 -d ${docroot}
sudo bash -c 'cat >/etc/systemd/system/nginx.service <<"UNITEND"
[Unit]
Description=nginx (binary install at ${prefix})
After=network.target

[Service]
Type=forking
PIDFile=${prefix}/logs/nginx.pid
ExecStartPre=${prefix}/sbin/nginx -t
ExecStart=${prefix}/sbin/nginx
ExecReload=/bin/kill -s HUP $MAINPID
ExecStop=/bin/kill -s QUIT $MAINPID
PrivateTmp=true

[Install]
WantedBy=multi-user.target
UNITEND'
sudo systemctl daemon-reload
sudo ${prefix}/sbin/nginx -t
sudo systemctl enable nginx
sudo systemctl restart nginx
sudo ss -lntp | grep -E ":${p.http_port || 80}\\b" || true`
  }

  return `${pkgInstall(form.osType, ['nginx'])}
${writeConf('/etc/nginx/nginx.conf')}
sudo install -m 0755 -d ${docroot}
sudo nginx -t
sudo systemctl enable nginx
sudo systemctl restart nginx
sudo ss -lntp | grep -E ":${p.http_port || 80}\\b" || true`
}

const buildHAProxy = () => {
  const p = form.params
  const backends = String(p.backends || '')
    .split('\n')
    .map((s: string) => s.trim())
    .filter(Boolean)
    .map((s: string, i: number) => `  server srv${i + 1} ${s}${p.check_enabled ? ' check' : ''}`)
    .join('\n')
  const conf = `global
  log /dev/log local0
  maxconn ${p.maxconn || 4096}
defaults
  log     global
  mode    ${p.mode || 'http'}
  option  ${p.mode === 'tcp' ? 'tcplog' : 'httplog'}
${p.httpclose && p.mode === 'http' ? '  option  httpclose\n' : ''}  timeout connect ${p.timeout_connect || '5s'}
  timeout client  ${p.timeout_client || '30s'}
  timeout server  ${p.timeout_server || '30s'}
frontend web
  bind *:${p.frontend_port || p.port || 80}
  default_backend app
backend app
  balance ${p.algorithm}
${p.httpchk_enabled && p.mode === 'http' ? `  option httpchk ${p.httpchk_path || 'GET /health'}\n` : ''}${backends}`
  const stats = p.stats_enabled ? `
listen stats
  bind *:${p.stats_port || 8404}
  mode http
  stats enable
  stats uri /stats
  stats refresh 10s` : ''
  const fullConf = `${conf}${stats}`
  if (form.installMethod === 'docker') {
    return `sudo mkdir -p /etc/haproxy
sudo bash -c 'cat >/etc/haproxy/haproxy.cfg <<"HAPROXYCFG"
${fullConf}
HAPROXYCFG'
${dockerRun('haproxy', dockerImageTag('haproxy', p.version || 'lts', 'lts'), [`${p.frontend_port || 80}:${p.frontend_port || 80}`, ...(p.stats_enabled ? [`${p.stats_port || 8404}:${p.stats_port || 8404}`] : [])], [], ['/etc/haproxy/haproxy.cfg:/usr/local/etc/haproxy/haproxy.cfg:ro'])}`
  }
  return `${pkgInstall(form.osType, ['haproxy'])}
sudo bash -c 'cat >/etc/haproxy/haproxy.cfg <<"HAPROXYCFG"
${fullConf}
HAPROXYCFG'
sudo haproxy -c -f /etc/haproxy/haproxy.cfg
sudo systemctl enable haproxy
sudo systemctl restart haproxy
sudo ss -lntp | grep :${p.frontend_port || 80} || true`
}

const buildRedis = () => {
  const p = form.params
  const conf = [
    `bind ${p.bind || '0.0.0.0'}`,
    `protected-mode ${p.protected_mode ? 'yes' : 'no'}`,
    `port ${p.port || 6379}`,
    `databases ${p.databases || 16}`,
    `dir ${p.dir || '/var/lib/redis'}`,
    `dbfilename ${p.dbfilename || 'dump.rdb'}`,
    `maxmemory ${p.maxmemory || '512mb'}`,
    `maxmemory-policy ${p.maxmemory_policy || 'allkeys-lru'}`,
    `timeout ${p.timeout ?? 0}`,
    `tcp-keepalive ${p.tcp_keepalive ?? 300}`,
    `appendonly ${p.appendonly ? 'yes' : 'no'}`,
    ...(p.appendonly ? [`appendfsync ${p.appendfsync || 'everysec'}`] : []),
    ...(p.requirepass ? [`requirepass ${p.requirepass}`] : []),
    ...(p.rdb_enabled ? ['save 900 1', 'save 300 10', 'save 60 10000'] : ['save ""']),
    ...(p.supervised_systemd && form.installMethod !== 'docker' ? ['supervised systemd'] : [])
  ].join('\n')

  if (form.installMethod === 'docker') {
    return `sudo mkdir -p /etc/redis ${p.dir || '/var/lib/redis'}
sudo bash -c 'cat >/etc/redis/redis.conf <<"REDISCONF"
${conf}
REDISCONF'
sudo docker rm -f redis 2>/dev/null || true
sudo docker run -d --name redis --restart=always -p ${p.port || 6379}:${p.port || 6379} \\
  -v /etc/redis/redis.conf:/usr/local/etc/redis/redis.conf:ro \\
  -v ${p.dir || '/var/lib/redis'}:${p.dir || '/var/lib/redis'} \\
  ${dockerImageTag('redis', p.version || '7', '7')} redis-server /usr/local/etc/redis/redis.conf
sudo ss -lntp | grep :${p.port || 6379} || true`
  }
  return `${pkgInstall(form.osType, ['redis-server'])}
sudo mkdir -p ${p.dir || '/var/lib/redis'}
sudo bash -c 'cat >/etc/redis/redis.conf <<"REDISCONF"
${conf}
REDISCONF'
sudo systemctl enable redis-server || sudo systemctl enable redis
sudo systemctl restart redis-server || sudo systemctl restart redis
sudo ss -lntp | grep :${p.port || 6379} || true`
}

const buildKafka = () => {
  const p = form.params
  return `# 推荐 Docker 方式快速部署 Kafka
${dockerRun(
    'kafka',
    dockerImageTag('bitnami/kafka', p.version || '3.6', '3.6'),
    [`${p.port}:9092`],
    [
      `KAFKA_BROKER_ID=${p.broker_id}`,
      `KAFKA_CFG_ZOOKEEPER_CONNECT=${p.zookeeper}`,
      `KAFKA_CFG_LISTENERS=PLAINTEXT://:9092`,
      `KAFKA_CFG_ADVERTISED_LISTENERS=PLAINTEXT://$(hostname -I | awk '{print $1}'):${p.port || 9092}`,
      `KAFKA_CFG_LOG_DIRS=${p.log_dir}`,
      `KAFKA_CFG_NUM_PARTITIONS=${p.num_partitions || 3}`,
      `KAFKA_CFG_DEFAULT_REPLICATION_FACTOR=${p.default_replication_factor || 1}`,
      `KAFKA_CFG_LOG_RETENTION_HOURS=${p.log_retention_hours || 168}`,
      `KAFKA_CFG_LOG_SEGMENT_BYTES=${p.log_segment_bytes || 1073741824}`,
      `KAFKA_CFG_AUTO_CREATE_TOPICS_ENABLE=${p.auto_create_topics ? 'true' : 'false'}`,
      'ALLOW_PLAINTEXT_LISTENER=yes'
    ],
    ['kafka-data:/bitnami/kafka']
  )}
sudo ss -lntp | grep :${p.port} || true`
}

const buildMySQL = () => {
  const p = form.params
  const cnf = `[mysqld]
port=${p.port || 3306}
bind-address=${p.bind_address || '0.0.0.0'}
character-set-server=${p.charset || 'utf8mb4'}
collation-server=${p.collation || 'utf8mb4_0900_ai_ci'}
max_connections=${p.max_connections || 500}
innodb_buffer_pool_size=${p.innodb_buffer_pool_size || '512M'}
${p.skip_name_resolve ? 'skip-name-resolve\n' : ''}${p.slow_query_log ? `slow_query_log=ON
long_query_time=${p.long_query_time ?? 2}
slow_query_log_file=/var/log/mysql/mysql-slow.log
` : ''}`
  if (form.installMethod === 'docker') {
    return `sudo mkdir -p /etc/mysql/conf.d ${p.datadir || '/var/lib/mysql'}
sudo bash -c 'cat >/etc/mysql/conf.d/99-ai-sre.cnf <<"MYSQLCONF"
${cnf}
MYSQLCONF'
sudo docker rm -f mysql 2>/dev/null || true
sudo docker run -d --name mysql --restart=always -p ${p.port || 3306}:3306 \\
  -e MYSQL_ROOT_PASSWORD='${p.root_password || 'changeme'}' \\
  -e MYSQL_DATABASE=app \\
  -v ${p.datadir || '/var/lib/mysql'}:/var/lib/mysql \\
  -v /etc/mysql/conf.d/99-ai-sre.cnf:/etc/mysql/conf.d/99-ai-sre.cnf:ro \\
  ${dockerImageTag('mysql', p.version || '8.0', '8.0')}
sudo ss -lntp | grep :${p.port || 3306} || true`
  }
  return `${pkgInstall(form.osType, ['mysql-server'])}
sudo mkdir -p /etc/mysql/mysql.conf.d /var/log/mysql
sudo bash -c 'cat >/etc/mysql/mysql.conf.d/99-ai-sre.cnf <<"MYSQLCONF"
${cnf}
MYSQLCONF'
sudo systemctl enable mysql || sudo systemctl enable mysqld
sudo systemctl restart mysql || sudo systemctl restart mysqld
sudo mysql -uroot -e "ALTER USER 'root'@'localhost' IDENTIFIED BY '${p.root_password}'; FLUSH PRIVILEGES;" || true
sudo ss -lntp | grep :${p.port || 3306} || true`
}

const buildPostgres = () => {
  const p = form.params
  const postgresConf = `listen_addresses = '${p.listen_addresses || '*'}'
port = ${p.port || 5432}
max_connections = ${p.max_connections || 200}
shared_buffers = '${p.shared_buffers || '512MB'}'
work_mem = '${p.work_mem || '8MB'}'
wal_level = ${p.wal_level || 'replica'}
log_min_duration_statement = ${p.log_min_duration_statement ?? 1000}`
  const hbaLine = p.trust_local_network ? 'host all all 0.0.0.0/0 scram-sha-256' : 'host all all 127.0.0.1/32 scram-sha-256'
  if (form.installMethod === 'docker') {
    return `sudo mkdir -p /etc/postgresql ${p.datadir || '/var/lib/postgresql/data'}
sudo bash -c 'cat >/etc/postgresql/postgresql.conf <<"PGCONF"
${postgresConf}
PGCONF'
sudo bash -c 'cat >/etc/postgresql/pg_hba.conf <<"PGHBA"
local all all trust
host all all 127.0.0.1/32 scram-sha-256
${hbaLine}
PGHBA'
sudo docker rm -f postgres 2>/dev/null || true
sudo docker run -d --name postgres --restart=always -p ${p.port || 5432}:5432 \\
  -e POSTGRES_PASSWORD='${p.password || 'changeme'}' \\
  -e PGDATA=${p.datadir || '/var/lib/postgresql/data'} \\
  -v ${p.datadir || '/var/lib/postgresql/data'}:${p.datadir || '/var/lib/postgresql/data'} \\
  -v /etc/postgresql/postgresql.conf:/etc/postgresql/postgresql.conf:ro \\
  -v /etc/postgresql/pg_hba.conf:/etc/postgresql/pg_hba.conf:ro \\
  ${dockerImageTag('postgres', p.version || '16', '16')} -c config_file=/etc/postgresql/postgresql.conf -c hba_file=/etc/postgresql/pg_hba.conf
sudo ss -lntp | grep :${p.port || 5432} || true`
  }
  return `${pkgInstall(form.osType, ['postgresql', 'postgresql-contrib'])}
PG_CONF_DIR="$(sudo -u postgres psql -tAc 'show config_file' 2>/dev/null | xargs dirname || true)"
if [ -z "$PG_CONF_DIR" ]; then PG_CONF_DIR="/etc/postgresql"; fi
sudo mkdir -p "\${PG_CONF_DIR}/conf.d" 2>/dev/null || true
if [ -d "\${PG_CONF_DIR}/conf.d" ]; then
  sudo tee "\${PG_CONF_DIR}/conf.d/99-ai-sre.conf" >/dev/null <<"PGCONF"
${postgresConf}
PGCONF
else
  sudo tee -a "\${PG_CONF_DIR}/postgresql.conf" >/dev/null <<"PGCONF"
${postgresConf}
PGCONF
fi
echo "${hbaLine}" | sudo tee -a "\${PG_CONF_DIR}/pg_hba.conf" >/dev/null || true
sudo systemctl enable postgresql
sudo systemctl restart postgresql
sudo -u postgres psql -c "ALTER USER postgres WITH PASSWORD '${p.password}';" || true
sudo ss -lntp | grep :${p.port || 5432} || true`
}

const buildElasticsearch = () => {
  const p = form.params
  const method = form.installMethod
  const prefix = p.install_prefix || '/opt/elasticsearch'
  return `# Elasticsearch 部署较为复杂（vm.max_map_count、ulimits、heap、wait-ready），
# 强烈推荐使用 "curl + bash（推荐）" 标签或 "ai-sre CLI" 命令，
# 让 ai-sre 在目标机统一处理系统调优、配置、启动与健康等待。
# 下方仅展示关键步骤摘要：
echo "[reminder] 安装方式=${method}；目标机 vm.max_map_count 建议 >= 262144；JVM heap=${p.heap_size || '1g'}"
echo "[reminder] 数据目录：${p.path_data || '/var/lib/elasticsearch'}；日志目录：${p.path_logs || '/var/log/elasticsearch'}"
echo "[reminder] HTTP 端口：${p.http_port || 9200}；Transport：${p.transport_port || 9300}"
echo "[reminder] 集群模式：${p.discovery_type || 'single-node'}；xpack.security=${p.xpack_security ? 'true' : 'false'}"
${method === 'binary' ? `echo "[reminder] 二进制安装目录：${prefix}；配置目录：${prefix}/config（ES_PATH_CONF）"` : ''}
echo "[reminder] 复制使用上方 ai-sre 命令以触发完整安装流水线"`
}

const bashScript = computed(() => {
  if (!selected.value) return ''
  const header = `#!/usr/bin/env bash
set -euo pipefail
echo "[ai-sre] service=${form.service} os=${form.osType} method=${form.installMethod}"`
  let body = ''
  switch (form.service) {
    case 'nginx': body = buildNginx(); break
    case 'haproxy': body = buildHAProxy(); break
    case 'redis': body = buildRedis(); break
    case 'kafka': body = buildKafka(); break
    case 'mysql': body = buildMySQL(); break
    case 'postgresql': body = buildPostgres(); break
    case 'elasticsearch': body = buildElasticsearch(); break
  }
  return `${header}\n${body}\n`
})

const aiSreCommand = computed(() => {
  if (!selected.value) return ''
  const params = Object.entries(form.params)
    .filter(([_, v]) => v !== undefined && v !== '' && v !== null && typeof v !== 'object')
    .map(([k, v]) => `--${k}=${typeof v === 'string' ? `'${String(v).replace(/'/g, "'\\''")}'` : v}`)
    .join(' ')
  return `# 规划中（ai-sre 0.5+）
ai-sre install ${form.service} --os=${form.osType} --method=${form.installMethod} ${params}

# 当前可用：让 ai-sre 给出可执行步骤（基于本机 LLM/服务端 AI）
ai-sre runbook "在 ${osTypeOptions.find(x => x.value === form.osType)?.label} 上以 ${installMethodLabels[form.installMethod]} 安装并配置 ${selected.value?.name}：${params}"
`
})

const copy = async (text: string) => {
  if (!text) {
    ElMessage.warning('没有可复制内容')
    return
  }
  try {
    await copyTextToClipboard(text)
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败：请用鼠标选中下方文本后 Ctrl/Cmd+C，或在浏览器设置中允许剪贴板权限')
  }
}

if (options?.fixedServiceKey) {
  onMounted(() => selectService(options.fixedServiceKey!))
}

return {
  form,
  catalog,
  selected,
  selectService,
  osTypeOptions,
  availableInstallMethods,
  profileOptions,
  regularSections,
  collapsibleSections,
  activeCollapseSections,
  visibleFields,
  normalFields,
  switchFields,
  colMd,
  sectionNormalColMd,
  previewVisible,
  activeTab,
  generating,
  submittingUpdate,
  generatedDeployment,
  curlCommand,
  aiSreInstallCommand,
  aiSreUpdateCommand,
  aiSreUninstallCommand,
  aiSreRecoverCommand,
  deploymentDirty,
  canSubmitUpdate,
  deploymentStatusDescription,
  confPreview,
  onGenerate,
  onSubmitUpdate,
  onReset,
  copy,
  defaultBashFilename,
  bashScript,
  aiSreCommand,
  Check,
  Upload,
  RefreshRight,
  DocumentCopy,
  InfoFilled,
}
}
