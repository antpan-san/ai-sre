<template>
  <div class="service-deploy page-shell">
    <div class="split-layout">
      <el-card class="catalog-card" shadow="never">
        <template #header>
          <div class="catalog-header">
            <span class="catalog-title">基础服务</span>
            <span class="catalog-sub">选择 → 配参 → 生成脚本</span>
          </div>
        </template>
        <div class="catalog-list">
          <div
            v-for="item in catalog"
            :key="item.key"
            :class="['catalog-item', { selected: form.service === item.key }]"
            @click="selectService(item.key)"
          >
            <div class="catalog-item-head">
              <span class="catalog-name">{{ item.name }}</span>
              <el-icon v-if="form.service === item.key" class="catalog-check"><Check /></el-icon>
            </div>
            <div class="catalog-desc">{{ item.description }}</div>
            <div class="catalog-tags">
              <el-tag v-for="t in item.tags" :key="t" size="small" type="info" effect="plain">{{ t }}</el-tag>
            </div>
          </div>
        </div>
      </el-card>

      <div class="config-pane">
        <el-empty
          v-if="!selected"
          description="左侧选择一个基础服务后，在此配置参数并生成部署脚本"
          class="empty-pane"
        />

        <template v-else>
          <el-card class="install-card">
            <template #header>
              <span>系统与安装方式</span>
            </template>
            <el-form label-position="top">
              <el-row :gutter="16">
                <el-col :xs="24" :md="12">
                  <el-form-item label="目标系统类型">
                    <el-select v-model="form.osType" style="width: 100%">
                      <el-option v-for="os in osTypeOptions" :key="os.value" :label="os.label" :value="os.value" />
                    </el-select>
                  </el-form-item>
                </el-col>
                <el-col :xs="24" :md="12">
                  <el-form-item label="安装方式">
                    <el-select v-model="form.installMethod" style="width: 100%">
                      <el-option
                        v-for="method in availableInstallMethods"
                        :key="method.value"
                        :label="method.label"
                        :value="method.value"
                      />
                    </el-select>
                  </el-form-item>
                </el-col>
              </el-row>
            </el-form>
          </el-card>

          <el-card
            v-for="sec in regularSections"
            :key="sec.key"
            class="config-card"
          >
            <template #header>
              <div class="section-header">
                <span>{{ selected.name }} · {{ sec.title }}</span>
                <span v-if="sec.hint" class="section-hint">{{ sec.hint }}</span>
              </div>
            </template>
            <el-form label-position="top">
              <div class="section-field-grid">
                <div class="section-normal-fields">
                  <el-row :gutter="16">
                    <el-col
                      v-for="field in normalFields(sec.fields)"
                      :key="field.key"
                      :xs="24"
                      :md="sectionNormalColMd(field)"
                    >
                      <el-form-item :label="field.label">
                        <el-input-number
                          v-if="field.type === 'number'"
                          v-model="form.params[field.key]"
                          :min="field.min ?? 1"
                          :max="field.max ?? 65535"
                          style="width: 100%"
                        />
                        <el-select
                          v-else-if="field.type === 'select'"
                          v-model="form.params[field.key]"
                          style="width: 100%"
                        >
                          <el-option v-for="opt in field.options" :key="opt" :label="opt" :value="opt" />
                        </el-select>
                        <el-select
                          v-else-if="field.type === 'autocomplete'"
                          v-model="form.params[field.key]"
                          filterable
                          allow-create
                          default-first-option
                          :placeholder="field.placeholder || '选择或输入自定义值'"
                          style="width: 100%"
                        >
                          <el-option v-for="opt in field.options" :key="opt" :label="opt" :value="opt" />
                        </el-select>
                        <el-input
                          v-else-if="field.type === 'textarea'"
                          v-model="form.params[field.key]"
                          type="textarea"
                          :rows="field.rows ?? 3"
                          :placeholder="field.placeholder || ''"
                        />
                        <el-input
                          v-else
                          v-model="form.params[field.key]"
                          :placeholder="field.placeholder || ''"
                        />
                      </el-form-item>
                    </el-col>
                  </el-row>
                </div>
                <div
                  v-if="switchFields(sec.fields).length"
                  class="section-switch-fields"
                >
                  <div
                    v-for="field in switchFields(sec.fields)"
                    :key="field.key"
                    class="switch-row switch-row--compact"
                  >
                    <span class="switch-row-label">
                      {{ field.label }}
                      <el-tooltip v-if="field.tip" :content="field.tip" placement="top">
                        <el-icon class="switch-row-tip"><InfoFilled /></el-icon>
                      </el-tooltip>
                    </span>
                    <el-switch
                      v-model="form.params[field.key]"
                      inline-prompt
                      active-text="开"
                      inactive-text="关"
                    />
                  </div>
                </div>
              </div>
            </el-form>
          </el-card>

          <el-collapse
            v-if="collapsibleSections.length"
            v-model="activeCollapseSections"
            class="advanced-collapse"
          >
            <el-collapse-item
              v-for="sec in collapsibleSections"
              :key="sec.key"
              :name="sec.key"
              :title="`${selected.name} · ${sec.title}${sec.hint ? '（' + sec.hint + '）' : ''}`"
            >
              <el-form v-if="sec.fields.length" label-position="top">
                <el-row :gutter="16">
                  <el-col
                    v-for="field in visibleFields(sec.fields)"
                    :key="field.key"
                    :xs="24"
                    :md="colMd(field)"
                  >
                    <el-form-item :label="field.label">
                      <el-input-number
                        v-if="field.type === 'number'"
                        v-model="form.params[field.key]"
                        :min="field.min ?? 1"
                        :max="field.max ?? 65535"
                        style="width: 100%"
                      />
                      <el-select
                        v-else-if="field.type === 'select'"
                        v-model="form.params[field.key]"
                        style="width: 100%"
                      >
                        <el-option v-for="opt in field.options" :key="opt" :label="opt" :value="opt" />
                      </el-select>
                      <el-select
                        v-else-if="field.type === 'autocomplete'"
                        v-model="form.params[field.key]"
                        filterable
                        allow-create
                        default-first-option
                        :placeholder="field.placeholder || '选择或输入自定义值'"
                        style="width: 100%"
                      >
                        <el-option v-for="opt in field.options" :key="opt" :label="opt" :value="opt" />
                      </el-select>
                      <el-switch
                        v-else-if="field.type === 'switch'"
                        v-model="form.params[field.key]"
                        inline-prompt
                        active-text="开"
                        inactive-text="关"
                      />
                      <el-input
                        v-else-if="field.type === 'textarea'"
                        v-model="form.params[field.key]"
                        type="textarea"
                        :rows="field.rows ?? 3"
                        :placeholder="field.placeholder || ''"
                      />
                      <el-input
                        v-else
                        v-model="form.params[field.key]"
                        :placeholder="field.placeholder || ''"
                      />
                    </el-form-item>
                  </el-col>
                </el-row>
              </el-form>
              <pre
                v-if="sec.preview === 'config'"
                class="code-block code-block--inline"
              >{{ confPreview }}</pre>
            </el-collapse-item>
          </el-collapse>

          <div class="actions">
            <el-button type="primary" :icon="Upload" @click="onGenerate">生成部署脚本</el-button>
            <el-button :icon="RefreshRight" @click="onReset">重置</el-button>
          </div>
        </template>
      </div>
    </div>

    <el-dialog
      v-model="previewVisible"
      :title="`${selected?.name || ''} 部署脚本`"
      width="860px"
      :close-on-click-modal="false"
    >
      <el-alert
        type="info"
        :closable="false"
        show-icon
        title="使用方式"
      >
        <template #default>
          <div>
            「bash 脚本」可直接在目标节点执行 <code>sudo bash 文件</code>，立即可用；
            「ai-sre CLI」展示控制台等价命令，<code>ai-sre install &lt;service&gt;</code> 子命令在规划中，
            当前可用 <code>ai-sre runbook</code> 让 AI 给出详细步骤。
          </div>
        </template>
      </el-alert>

      <el-tabs v-model="activeTab" class="dialog-tabs">
        <el-tab-pane label="bash 脚本（立即可用）" name="bash">
          <div class="tab-actions">
            <el-tag size="small" type="success">目标机执行：sudo bash {{ defaultBashFilename }}</el-tag>
            <el-button size="small" :icon="DocumentCopy" @click="copy(bashScript)">复制</el-button>
            <el-button size="small" :icon="Download" @click="download(bashScript, defaultBashFilename)">下载 .sh</el-button>
          </div>
          <pre class="code-block">{{ bashScript }}</pre>
        </el-tab-pane>
        <el-tab-pane label="ai-sre CLI" name="cli">
          <el-alert
            type="warning"
            :closable="false"
            show-icon
            title="`ai-sre install <service>` 即将提供（0.5+）；当前推荐用 ai-sre runbook 让 AI 给详细步骤"
          />
          <div class="tab-actions">
            <el-button size="small" :icon="DocumentCopy" @click="copy(aiSreCommand)">复制命令</el-button>
          </div>
          <pre class="code-block">{{ aiSreCommand }}</pre>
        </el-tab-pane>
      </el-tabs>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { DocumentCopy, Download, Upload, RefreshRight, Check, InfoFilled } from '@element-plus/icons-vue'

interface CatalogField {
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

interface CatalogSection {
  key: string
  title: string
  hint?: string
  collapsible?: boolean
  defaultOpen?: boolean
  visibleIf?: () => boolean
  fields: CatalogField[]
  preview?: 'config'
}

interface CatalogItem {
  key: string
  name: string
  description: string
  tags: string[]
  installMethods: string[]
  fields?: CatalogField[]
  sections?: CatalogSection[]
}

const form = reactive({
  service: '',
  osType: 'ubuntu-debian',
  installMethod: 'package',
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
  binary: '二进制 / 源码编译'
}

const selected = computed<CatalogItem | null>(() => catalog.find(c => c.key === form.service) || null)

const availableInstallMethods = computed(() => {
  const methods = selected.value?.installMethods || ['package']
  return methods.map(m => ({ value: m, label: installMethodLabels[m] || m }))
})

const allSections = computed<CatalogSection[]>(() => {
  if (!selected.value) return []
  if (selected.value.sections) return selected.value.sections
  return [{ key: 'config', title: '参数配置', fields: selected.value.fields || [] }]
})

const visibleSections = computed(() =>
  allSections.value.filter(sec => !sec.visibleIf || sec.visibleIf())
)

const regularSections = computed(() => visibleSections.value.filter(s => !s.collapsible))
const collapsibleSections = computed(() => visibleSections.value.filter(s => s.collapsible))

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
  form.params = seedParams(item)
  if (!item.installMethods.includes(form.installMethod)) {
    form.installMethod = item.installMethods[0]
  }
  activeCollapseSections.value = (item.sections || [])
    .filter(s => s.collapsible && s.defaultOpen)
    .map(s => s.key)
}

const onGenerate = () => {
  if (!selected.value) {
    ElMessage.warning('请先选择基础服务')
    return
  }
  activeTab.value = 'bash'
  previewVisible.value = true
}

const onReset = () => {
  if (!selected.value) return
  form.params = seedParams(selected.value)
  form.installMethod = selected.value.installMethods[0]
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
sudo mkdir -p "${PG_CONF_DIR}/conf.d" 2>/dev/null || true
if [ -d "${PG_CONF_DIR}/conf.d" ]; then
  sudo tee "${PG_CONF_DIR}/conf.d/99-ai-sre.conf" >/dev/null <<"PGCONF"
${postgresConf}
PGCONF
else
  sudo tee -a "${PG_CONF_DIR}/postgresql.conf" >/dev/null <<"PGCONF"
${postgresConf}
PGCONF
fi
echo "${hbaLine}" | sudo tee -a "${PG_CONF_DIR}/pg_hba.conf" >/dev/null || true
sudo systemctl enable postgresql
sudo systemctl restart postgresql
sudo -u postgres psql -c "ALTER USER postgres WITH PASSWORD '${p.password}';" || true
sudo ss -lntp | grep :${p.port || 5432} || true`
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
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败，请手动选择复制')
  }
}

const download = (text: string, filename: string) => {
  const blob = new Blob([text], { type: 'text/x-shellscript' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}
</script>

<style scoped>
.service-deploy {
  width: 100%;
  max-width: none;
  margin: 0;
  padding: 8px var(--page-padding-x, 24px) 24px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  box-sizing: border-box;
  overflow-y: auto;
  overflow-x: hidden;
}

.service-deploy :deep(.el-card) {
  border-radius: 12px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
}

.service-deploy :deep(.el-card__header) {
  padding: 10px 16px;
  font-size: 14px;
}

.service-deploy :deep(.el-card__body) {
  padding: 12px 16px;
}

.catalog-header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
}

.catalog-title {
  font-weight: 600;
}

.catalog-sub {
  color: #94a3b8;
  font-size: 12px;
}

.split-layout {
  display: grid;
  grid-template-columns: 280px 1fr;
  gap: 12px;
  align-items: start;
}

@media (max-width: 960px) {
  .split-layout {
    grid-template-columns: 1fr;
  }
}

.catalog-card {
  position: sticky;
  top: 0;
}

.catalog-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.catalog-item {
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  padding: 10px 12px;
  cursor: pointer;
  background: var(--el-bg-color);
  transition: border-color .15s, box-shadow .15s, background .15s;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.catalog-item:hover {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 2px rgba(64,158,255,.15);
}

.catalog-item.selected {
  border-color: var(--el-color-primary);
  background: rgba(64,158,255,.08);
}

.catalog-item-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.catalog-check {
  color: var(--el-color-primary);
  font-size: 16px;
}

.catalog-name {
  font-weight: 600;
}

.catalog-desc {
  font-size: 12px;
  color: #6b7280;
}

.catalog-tags {
  margin-top: 2px;
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.config-pane {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
}

.empty-pane {
  background: var(--el-bg-color);
  border: 1px dashed var(--el-border-color);
  border-radius: 8px;
  padding: 32px 0;
}

.section-header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
}

.section-hint {
  color: #94a3b8;
  font-size: 12px;
}

.section-field-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  column-gap: 16px;
  align-items: start;
}

.section-normal-fields {
  grid-column: 1 / span 3;
  min-width: 0;
}

.section-switch-fields {
  grid-column: 4 / span 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 32px;
  padding: 4px 8px;
  margin: 4px 0 18px;
  border-radius: 6px;
  border: 1px dashed var(--el-border-color);
  background: var(--el-fill-color-lighter);
}

.switch-row--compact {
  margin-top: 0;
}

.switch-row-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: var(--el-text-color-regular);
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.switch-row-tip {
  color: #94a3b8;
  font-size: 14px;
  cursor: help;
}

@media (max-width: 1200px) {
  .section-field-grid {
    grid-template-columns: 1fr;
  }

  .section-normal-fields,
  .section-switch-fields {
    grid-column: 1;
  }
}

.advanced-collapse {
  border-radius: 12px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  padding: 0 12px;
}

.advanced-collapse :deep(.el-collapse-item__header) {
  font-weight: 600;
  font-size: 14px;
}

.actions {
  margin: 12px 0 24px;
  display: flex;
  gap: 12px;
}

.tab-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.code-block {
  background: #0f172a;
  color: #e2e8f0;
  padding: 12px 14px;
  border-radius: 6px;
  max-height: 480px;
  overflow: auto;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
}

.code-block--inline {
  margin-top: 4px;
  max-height: 360px;
}

.dialog-tabs {
  margin-top: 8px;
}
</style>
