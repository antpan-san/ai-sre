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
          <el-card class="config-card">
            <template #header>
              <span>{{ selected.name }} · 参数配置</span>
            </template>
            <el-form label-position="top">
              <el-row :gutter="16">
                <el-col v-for="field in selected.fields" :key="field.key" :xs="24" :md="12">
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
                    <el-switch
                      v-else-if="field.type === 'switch'"
                      v-model="form.params[field.key]"
                    />
                    <el-input
                      v-else-if="field.type === 'textarea'"
                      v-model="form.params[field.key]"
                      type="textarea"
                      :rows="3"
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
          </el-card>

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
import { DocumentCopy, Download, Upload, RefreshRight, Check } from '@element-plus/icons-vue'

interface CatalogField {
  key: string
  label: string
  type: 'text' | 'number' | 'select' | 'switch' | 'textarea'
  default: any
  options?: string[]
  min?: number
  max?: number
  placeholder?: string
}

interface CatalogItem {
  key: string
  name: string
  description: string
  tags: string[]
  fields: CatalogField[]
  installMethods: string[]
}

const catalog: CatalogItem[] = [
  {
    key: 'nginx',
    name: 'Nginx',
    description: 'Web 服务器 / 反向代理',
    tags: ['gateway', 'web'],
    installMethods: ['package', 'docker'],
    fields: [
      { key: 'port', label: '监听端口', type: 'number', default: 80 },
      { key: 'worker', label: 'worker_processes', type: 'number', default: 4, min: 1, max: 256 },
      { key: 'ssl', label: '启用 SSL（仅占位）', type: 'switch', default: false },
      { key: 'docroot', label: '静态资源目录', type: 'text', default: '/var/www/html' }
    ]
  },
  {
    key: 'haproxy',
    name: 'HAProxy',
    description: '高可用 4/7 层负载均衡',
    tags: ['gateway', 'lb'],
    installMethods: ['package', 'docker'],
    fields: [
      { key: 'port', label: '前端端口', type: 'number', default: 80 },
      {
        key: 'algorithm',
        label: '负载策略',
        type: 'select',
        default: 'roundrobin',
        options: ['roundrobin', 'leastconn', 'source']
      },
      {
        key: 'backends',
        label: '后端列表（每行 host:port）',
        type: 'textarea',
        default: '10.0.0.1:8080\n10.0.0.2:8080',
        placeholder: '10.0.0.1:8080\n10.0.0.2:8080'
      }
    ]
  },
  {
    key: 'redis',
    name: 'Redis',
    description: '内存数据库 / 缓存',
    tags: ['cache', 'kv'],
    installMethods: ['package', 'docker'],
    fields: [
      { key: 'port', label: '端口', type: 'number', default: 6379 },
      { key: 'password', label: 'requirepass（可空）', type: 'text', default: '' },
      { key: 'maxmemory', label: 'maxmemory', type: 'text', default: '512mb' },
      {
        key: 'persistence',
        label: '持久化',
        type: 'select',
        default: 'rdb',
        options: ['none', 'rdb', 'aof', 'both']
      }
    ]
  },
  {
    key: 'kafka',
    name: 'Kafka',
    description: '分布式消息队列',
    tags: ['mq'],
    installMethods: ['docker'],
    fields: [
      { key: 'port', label: 'broker 端口', type: 'number', default: 9092 },
      { key: 'broker_id', label: 'broker.id', type: 'number', default: 1, min: 0, max: 4096 },
      { key: 'zookeeper', label: 'ZooKeeper 地址', type: 'text', default: 'localhost:2181' },
      { key: 'log_dir', label: 'log.dirs', type: 'text', default: '/var/lib/kafka/logs' }
    ]
  },
  {
    key: 'mysql',
    name: 'MySQL',
    description: '关系型数据库',
    tags: ['db', 'sql'],
    installMethods: ['package', 'docker'],
    fields: [
      { key: 'port', label: '端口', type: 'number', default: 3306 },
      { key: 'root_password', label: 'root 密码', type: 'text', default: 'changeme' },
      { key: 'datadir', label: '数据目录', type: 'text', default: '/var/lib/mysql' },
      { key: 'charset', label: '字符集', type: 'text', default: 'utf8mb4' }
    ]
  },
  {
    key: 'postgresql',
    name: 'PostgreSQL',
    description: '关系型数据库',
    tags: ['db', 'sql'],
    installMethods: ['package', 'docker'],
    fields: [
      { key: 'port', label: '端口', type: 'number', default: 5432 },
      { key: 'password', label: 'POSTGRES_PASSWORD', type: 'text', default: 'changeme' },
      { key: 'datadir', label: 'PGDATA 目录', type: 'text', default: '/var/lib/postgresql/data' }
    ]
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
  binary: '二进制（仅部分服务）'
}

const form = reactive({
  service: '',
  osType: 'ubuntu-debian',
  installMethod: 'package',
  params: {} as Record<string, any>
})

const selected = computed<CatalogItem | null>(() => catalog.find(c => c.key === form.service) || null)

const availableInstallMethods = computed(() => {
  const methods = selected.value?.installMethods || ['package']
  return methods.map(m => ({ value: m, label: installMethodLabels[m] || m }))
})

const previewVisible = ref(false)
const activeTab = ref<'bash' | 'cli'>('bash')

const selectService = (key: string) => {
  form.service = key
  const item = catalog.find(c => c.key === key)
  if (!item) return
  form.params = item.fields.reduce((acc, f) => {
    acc[f.key] = f.default
    return acc
  }, {} as Record<string, any>)
  if (!item.installMethods.includes(form.installMethod)) {
    form.installMethod = item.installMethods[0]
  }
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
  form.params = selected.value.fields.reduce((acc, f) => {
    acc[f.key] = f.default
    return acc
  }, {} as Record<string, any>)
  form.installMethod = selected.value.installMethods[0]
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

const buildNginx = () => {
  const p = form.params
  if (form.installMethod === 'docker') {
    return dockerRun('nginx', 'nginx:stable', [`${p.port}:80`], [], [`${p.docroot}:/usr/share/nginx/html:ro`])
  }
  return `${pkgInstall(form.osType, ['nginx'])}
sudo sed -i 's/^worker_processes.*/worker_processes ${p.worker};/' /etc/nginx/nginx.conf || true
if [ -f /etc/nginx/sites-available/default ]; then
  sudo sed -i 's/listen .* default_server.*/listen ${p.port} default_server;/' /etc/nginx/sites-available/default || true
fi
sudo systemctl enable nginx
sudo systemctl restart nginx
sudo ss -lntp | grep :${p.port} || true`
}

const buildHAProxy = () => {
  const p = form.params
  const backends = String(p.backends || '')
    .split('\n')
    .map((s: string) => s.trim())
    .filter(Boolean)
    .map((s: string, i: number) => `  server srv${i + 1} ${s} check`)
    .join('\n')
  const conf = `global
  log /dev/log local0
defaults
  log     global
  mode    http
  timeout connect 5s
  timeout client  30s
  timeout server  30s
frontend web
  bind *:${p.port}
  default_backend app
backend app
  balance ${p.algorithm}
${backends}`
  if (form.installMethod === 'docker') {
    return `sudo mkdir -p /etc/haproxy
sudo bash -c 'cat >/etc/haproxy/haproxy.cfg <<EOF
${conf}
EOF'
${dockerRun('haproxy', 'haproxy:lts', [`${p.port}:${p.port}`], [], ['/etc/haproxy/haproxy.cfg:/usr/local/etc/haproxy/haproxy.cfg:ro'])}`
  }
  return `${pkgInstall(form.osType, ['haproxy'])}
sudo bash -c 'cat >/etc/haproxy/haproxy.cfg <<EOF
${conf}
EOF'
sudo systemctl enable haproxy
sudo systemctl restart haproxy
sudo ss -lntp | grep :${p.port} || true`
}

const buildRedis = () => {
  const p = form.params
  if (form.installMethod === 'docker') {
    const cmd = ['redis-server', '--port', String(p.port), '--maxmemory', String(p.maxmemory)]
    if (p.password) cmd.push('--requirepass', String(p.password))
    if (p.persistence === 'aof') cmd.push('--appendonly', 'yes')
    if (p.persistence === 'both') cmd.push('--appendonly', 'yes')
    return dockerRun('redis', 'redis:7', [`${p.port}:${p.port}`], [], []) + '\n# 注：如需自定义配置可改用挂载 redis.conf'
  }
  return `${pkgInstall(form.osType, ['redis-server'])}
sudo sed -i 's/^# *requirepass .*/requirepass ${p.password || ''}/' /etc/redis/redis.conf 2>/dev/null || true
sudo sed -i 's/^port .*/port ${p.port}/' /etc/redis/redis.conf 2>/dev/null || true
sudo sed -i 's/^# *maxmemory .*/maxmemory ${p.maxmemory}/' /etc/redis/redis.conf 2>/dev/null || true
sudo systemctl enable redis-server || sudo systemctl enable redis
sudo systemctl restart redis-server || sudo systemctl restart redis
sudo ss -lntp | grep :${p.port} || true`
}

const buildKafka = () => {
  const p = form.params
  return `# 推荐 Docker 方式快速部署 Kafka
${dockerRun(
    'kafka',
    'bitnami/kafka:3.6',
    [`${p.port}:9092`],
    [
      `KAFKA_BROKER_ID=${p.broker_id}`,
      `KAFKA_CFG_ZOOKEEPER_CONNECT=${p.zookeeper}`,
      `KAFKA_CFG_LISTENERS=PLAINTEXT://:9092`,
      `KAFKA_CFG_LOG_DIRS=${p.log_dir}`,
      'ALLOW_PLAINTEXT_LISTENER=yes'
    ],
    ['kafka-data:/bitnami/kafka']
  )}
sudo ss -lntp | grep :${p.port} || true`
}

const buildMySQL = () => {
  const p = form.params
  if (form.installMethod === 'docker') {
    return `${dockerRun(
      'mysql',
      'mysql:8.0',
      [`${p.port}:3306`],
      [
        `MYSQL_ROOT_PASSWORD=${p.root_password}`,
        `MYSQL_DATABASE=app`
      ],
      [`${p.datadir}:/var/lib/mysql`]
    )}
sudo ss -lntp | grep :${p.port} || true`
  }
  return `${pkgInstall(form.osType, ['mysql-server'])}
sudo systemctl enable mysql || sudo systemctl enable mysqld
sudo systemctl restart mysql || sudo systemctl restart mysqld
sudo mysql -uroot -e "ALTER USER 'root'@'localhost' IDENTIFIED BY '${p.root_password}'; FLUSH PRIVILEGES;" || true
sudo ss -lntp | grep :${p.port} || true`
}

const buildPostgres = () => {
  const p = form.params
  if (form.installMethod === 'docker') {
    return `${dockerRun(
      'postgres',
      'postgres:16',
      [`${p.port}:5432`],
      [`POSTGRES_PASSWORD=${p.password}`, `PGDATA=${p.datadir}`],
      [`${p.datadir}:${p.datadir}`]
    )}
sudo ss -lntp | grep :${p.port} || true`
  }
  return `${pkgInstall(form.osType, ['postgresql', 'postgresql-contrib'])}
sudo systemctl enable postgresql
sudo systemctl restart postgresql
sudo -u postgres psql -c "ALTER USER postgres WITH PASSWORD '${p.password}';" || true
sudo ss -lntp | grep :${p.port} || true`
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
    .filter(([_, v]) => v !== undefined && v !== '' && v !== null)
    .map(([k, v]) => `--${k}=${typeof v === 'string' ? `'${v}'` : v}`)
    .join(' ')
  const installCmd = `# 规划中（ai-sre 0.5+）
ai-sre install ${form.service} --os=${form.osType} --method=${form.installMethod} ${params}

# 当前可用：让 ai-sre 给出可执行步骤（基于本机 LLM/服务端 AI）
ai-sre runbook "在 ${osTypeOptions.find(x => x.value === form.osType)?.label} 上以 ${installMethodLabels[form.installMethod]} 安装并配置 ${selected.value?.name}：${params}"
`
  return installCmd
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

.dialog-tabs {
  margin-top: 8px;
}
</style>
