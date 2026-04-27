<template>
  <div class="k8s-deploy-form page-shell">
    <header class="page-header">
      <div class="page-header-inner">
        <div class="page-header-copy">
          <span class="page-kicker">Kubernetes</span>
          <h2 class="page-title">部署 Kubernetes 集群</h2>
          <p class="page-desc">
            按信息类别展开配置，不必按步骤线性跳转；离线生成命令或 zip，在线由 Agent 执行。
          </p>
        </div>
        <div class="install-ai-sre-card">
          <div class="install-ai-sre-card__head">
            <div>
              <h3>安装 ai-sre</h3>
              <p>控制机诊断、清理与离线安装工具</p>
            </div>
          </div>
          <div
            class="install-command-copy"
            role="button"
            tabindex="0"
            title="点击复制安装命令"
            @click="copyInstallAiSreCurl"
            @keyup.enter="copyInstallAiSreCurl"
            @keyup.space="copyInstallAiSreCurl"
          >
            <code>{{ installAiSreCurlCommand }}</code>
            <span class="install-command-copy__hint">点击复制</span>
          </div>
        </div>
      </div>
    </header>

    <div class="deploy-layout">
      <main class="deploy-main">
        <el-collapse v-model="openSections" accordion class="deploy-config-collapse">
        <el-collapse-item name="precheck" class="deploy-config-item">
          <template #title>
            <div class="config-item-title">
              <span class="config-item-icon config-item-icon--precheck"><el-icon><CircleCheck /></el-icon></span>
              <span class="config-item-text">
                <span class="config-item-name">安装预检</span>
                <span class="config-item-desc">{{ getSectionDesc('precheck', '控制机免密 SSH 与节点环境基础检查') }}</span>
              </span>
            </div>
          </template>
          <div class="install-precheck">
            <section class="precheck-block">
              <div class="precheck-block__head">
                <h4>离线安装必读</h4>
                <span>控制机须能免密 SSH 各节点 root</span>
              </div>
              <div class="k8s-prereq-body">
                <p class="k8s-prereq-lead">离线安装由控制机执行 Ansible，控制机必须能以 <strong>root</strong> 免密连接所有节点。</p>
                <ol class="k8s-prereq-ol">
                  <li>控制机：<code>ssh-keygen -t ed25519 -N "" -f ~/.ssh/id_ed25519</code>（若已有密钥可跳过）</li>
                  <li>对每个节点：<code>ssh-copy-id -i ~/.ssh/id_ed25519.pub root@&lt;IP&gt;</code></li>
                  <li>验证：<code>ssh root@&lt;IP&gt;</code> 无密码</li>
                </ol>
                <p class="k8s-prereq-muted">不支持交互式密码；请先完成免密 SSH 再生成安装命令。</p>
              </div>
            </section>

            <section class="precheck-block">
              <div class="precheck-block__head">
                <h4>环境预检</h4>
                <span>建议在生成命令或开始部署前完成</span>
              </div>
              <div class="k8s-prereq-body">
                <p class="k8s-prereq-lead">建议先确认下列项目，避免安装后出现 etcd 慢、CNI 抖动或 CoreDNS 反复重启。需要自动诊断时可在 master 上执行 <code>sudo ai-sre k8s diagnose</code>。</p>
                <el-table :data="preflightRows" size="small" class="k8s-preflight-table" border>
                  <el-table-column prop="item" label="检查项" min-width="170" />
                  <el-table-column prop="why" label="不满足时的症状" min-width="240" />
                  <el-table-column label="在每个节点上执行" min-width="300">
                    <template #default="{ row }">
                      <code class="k8s-prereq-cmd">{{ row.cmd }}</code>
                    </template>
                  </el-table-column>
                  <el-table-column prop="expected" label="期望值" min-width="170" />
                </el-table>
                <p class="k8s-prereq-muted">节点间时钟偏差建议 &lt; 1s；虚拟机环境请先完成 NTP 同步。</p>
              </div>
            </section>
          </div>
        </el-collapse-item>

        <!-- ========== 步骤 1: 基础集群信息 ========== -->
        <el-collapse-item name="basic" class="deploy-config-item">
          <template #title>
            <div class="config-item-title">
              <span class="config-item-icon"><el-icon><component :is="stepsMeta[0].icon" /></el-icon></span>
              <span class="config-item-text">
                <span class="config-item-name">{{ stepsMeta[0].title }}</span>
                <span class="config-item-desc">{{ getSectionDesc('basic', stepsMeta[0].desc) }}</span>
              </span>
            </div>
          </template>
          <div class="step-section step-section--basic">
          <el-divider content-position="left">基础集群信息</el-divider>
          <el-form
            ref="step1FormRef"
            :model="deployConfig.clusterBasicInfo"
            :rules="step1Rules"
            label-position="top"
          >
            <el-row :gutter="16">
              <el-col :xs="24" :sm="12">
                <el-form-item label="集群名称" prop="clusterName">
                  <el-input
                    v-model="deployConfig.clusterBasicInfo.clusterName"
                    placeholder="请输入集群名称"
                    clearable
                  />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :sm="12">
                <el-form-item label="K8s 版本" prop="version">
                  <el-select
                    v-model="deployConfig.clusterBasicInfo.version"
                    placeholder="请选择 K8s 版本"
                    clearable
                    style="width: 100%"
                  >
                    <el-option
                      v-for="ver in k8sVersions"
                      :key="ver.version"
                      :label="ver.version + (ver.recommended ? ' (推荐)' : '')"
                      :value="ver.version"
                    />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :xs="24" :sm="12">
                <el-form-item label="部署模式" prop="deployMode">
                  <el-radio-group v-model="deployConfig.clusterBasicInfo.deployMode">
                    <el-radio value="single">单节点</el-radio>
                    <el-radio value="cluster">多节点</el-radio>
                  </el-radio-group>
                </el-form-item>
              </el-col>
            </el-row>

            <el-row :gutter="16">
              <el-col :xs="24" :sm="12">
                <el-form-item label="运行环境 CPU 架构" prop="cpuArch">
                  <el-select
                    v-model="deployConfig.clusterBasicInfo.cpuArch"
                    placeholder="与 install.sh 所在机及节点一致"
                    style="width: 100%"
                  >
                    <el-option label="amd64 (x86_64)" value="amd64" />
                    <el-option label="arm64 (AArch64 / Apple Silicon 虚拟机)" value="arm64" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :xs="24" :sm="12">
                <el-form-item label="镜像源" prop="imageSource">
                  <el-select
                    v-model="deployConfig.clusterBasicInfo.imageSource"
                    placeholder="请选择镜像源"
                    style="width: 100%"
                  >
                    <el-option label="默认" value="default" />
                    <el-option label="阿里云" value="aliyun" />
                    <el-option label="腾讯云" value="tencent" />
                    <el-option label="自定义" value="custom" />
                  </el-select>
                </el-form-item>
              </el-col>
            </el-row>

            <el-row :gutter="16">
              <el-col :xs="24" :sm="12">
                <el-form-item label="内网制品地址" prop="downloadDomain">
                  <el-input
                    v-model="deployConfig.clusterBasicInfo.downloadDomain"
                    placeholder="留空则用 inventory 默认 download_domain"
                    clearable
                  />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :sm="12">
                <el-form-item label="下载协议" prop="downloadProtocol">
                  <el-input
                    v-model="deployConfig.clusterBasicInfo.downloadProtocol"
                    placeholder="默认 http://"
                    clearable
                  />
                </el-form-item>
              </el-col>
            </el-row>
            <p class="form-hint form-hint--compact">
              留空则用 inventory 默认 <code>download_domain</code>；填写则覆盖。选「阿里云」时二进制多走公网。
            </p>

            <template v-if="deployConfig.clusterBasicInfo.imageSource === 'custom'">
              <el-divider content-position="left">自定义镜像仓库</el-divider>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="仓库地址" prop="customRegistry">
                    <el-input
                      v-model="deployConfig.clusterBasicInfo.customRegistry"
                      placeholder="例如: registry.example.com"
                    />
                  </el-form-item>
                </el-col>
                <el-col :span="6">
                  <el-form-item label="用户名">
                    <el-input
                      v-model="deployConfig.clusterBasicInfo.registryUsername"
                      placeholder="可选"
                    />
                  </el-form-item>
                </el-col>
                <el-col :span="6">
                  <el-form-item label="密码">
                    <el-input
                      v-model="deployConfig.clusterBasicInfo.registryPassword"
                      placeholder="可选"
                      type="password"
                      show-password
                    />
                  </el-form-item>
                </el-col>
              </el-row>
            </template>
          </el-form>
          </div>
        </el-collapse-item>

        <!-- ========== 步骤 2: 节点配置 ========== -->
        <el-collapse-item name="nodes" class="deploy-config-item" v-loading="machineStore.loading">
          <template #title>
            <div class="config-item-title">
              <span class="config-item-icon"><el-icon><component :is="stepsMeta[1].icon" /></el-icon></span>
              <span class="config-item-text">
                <span class="config-item-name">{{ stepsMeta[1].title }}</span>
                <span class="config-item-desc">{{ getSectionDesc('nodes', stepsMeta[1].desc) }}</span>
              </span>
            </div>
          </template>
          <div class="step-section">
          <el-form-item label="部署方式" class="node-mode-form-item">
            <div class="node-mode-row">
              <el-switch
                v-model="offlineBundleMode"
                inline-prompt
                active-text="离线"
                inactive-text="在线 Agent"
              />
              <span class="mode-hint-inline">
                {{
                  offlineBundleMode
                    ? '填写 IP；最后一步可生成一键命令或 zip（须满足步骤 1 中 SSH 免密说明）'
                    : '选择执行机与各节点；须网络互通并已装 Agent'
                }}
              </span>
            </div>
          </el-form-item>

          <template v-if="offlineBundleMode">
            <el-form-item label="控制平面 IP（必填，每行一个）" required>
              <el-input
                v-model="masterHostsText"
                type="textarea"
                :rows="5"
                placeholder="例如：&#10;192.168.1.10&#10;192.168.1.11"
              />
            </el-form-item>
            <el-form-item label="工作节点 IP（可选）">
              <el-input
                v-model="workerHostsText"
                type="textarea"
                :rows="4"
                placeholder="每行一个 Worker IP（可包含控制平面 IP；生成 inventory 时会自动去重）"
              />
            </el-form-item>
          </template>

          <template v-else>
            <el-form-item label="执行节点（Agent 所在机器）" required class="executor-select-item">
              <el-select
                v-model="deployConfig.nodeConfig.executorNode"
                placeholder="选择执行部署任务的机器（需在线且已安装 Agent）"
                clearable
                style="width: 100%"
              >
                <el-option
                  v-for="m in selectableExecutors"
                  :key="m.id"
                  :label="`${m.name || '未命名'} (${m.ip})`"
                  :value="m.id"
                >
                  <span>{{ m.name || '未命名' }}</span>
                  <span style="color: var(--el-text-color-secondary); margin-left: 8px">{{ m.ip }}</span>
                </el-option>
              </el-select>
            </el-form-item>

            <NodeSelect
              :machines="machines"
              :modelValue="{ masterNodes: deployConfig.nodeConfig.masterNodes, workerNodes: deployConfig.nodeConfig.workerNodes }"
              @update:modelValue="(v) => { deployConfig.nodeConfig.masterNodes = v.masterNodes; deployConfig.nodeConfig.workerNodes = v.workerNodes }"
              masterTitle="控制平面节点"
              workerTitle="工作节点（数据平面）"
            />
          </template>

          <el-divider content-position="left">标签与污点</el-divider>

          <div class="label-taint-grid">
            <el-card class="sub-card" shadow="hover">
              <template #header>
                <div class="sub-card-header"><span>主节点标签</span></div>
              </template>
            <LabelGroup v-model="masterLabelsModel" />
            </el-card>
            <el-card class="sub-card" shadow="hover">
              <template #header>
                <div class="sub-card-header"><span>主节点污点</span></div>
              </template>
            <TaintGroup v-model="masterTaintsModel" />
            </el-card>
            <el-card class="sub-card" shadow="hover">
              <template #header>
                <div class="sub-card-header"><span>工作节点标签</span></div>
              </template>
            <LabelGroup v-model="workerLabelsModel" />
            </el-card>
            <el-card class="sub-card" shadow="hover">
              <template #header>
                <div class="sub-card-header"><span>工作节点污点</span></div>
              </template>
            <TaintGroup v-model="workerTaintsModel" />
            </el-card>
          </div>
          </div>
        </el-collapse-item>

        <!-- ========== 步骤 3: 核心组件配置 ========== -->
        <el-collapse-item name="core" class="deploy-config-item">
          <template #title>
            <div class="config-item-title">
              <span class="config-item-icon"><el-icon><component :is="stepsMeta[2].icon" /></el-icon></span>
              <span class="config-item-text">
                <span class="config-item-name">{{ stepsMeta[2].title }}</span>
                <span class="config-item-desc">{{ getSectionDesc('core', stepsMeta[2].desc) }}</span>
              </span>
            </div>
          </template>
          <div class="step-section">
          <el-form
            :model="deployConfig.coreComponentsConfig"
            label-position="top"
          >
            <el-row :gutter="20">
              <el-col :span="8">
                <el-form-item label="kube-proxy 模式">
                  <el-select
                    v-model="deployConfig.coreComponentsConfig.kubeProxyMode"
                    style="width: 100%"
                  >
                    <el-option label="iptables" value="iptables" />
                    <el-option label="ipvs" value="ipvs" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="Pause 镜像">
                  <el-input
                    v-model="deployConfig.coreComponentsConfig.pauseImage"
                    placeholder="自定义 pause 镜像（可选）"
                    clearable
                  />
                </el-form-item>
              </el-col>
            </el-row>

            <el-row :gutter="20">
              <el-col :span="24">
                <el-form-item label="功能开关">
                  <el-checkbox-group v-model="coreFeatures">
                    <el-checkbox value="enableRBAC">启用 RBAC</el-checkbox>
                    <el-checkbox value="enablePodSecurityPolicy">启用 Pod 安全策略</el-checkbox>
                    <el-checkbox value="enableAudit">启用审计日志</el-checkbox>
                  </el-checkbox-group>
                </el-form-item>
              </el-col>
            </el-row>

            <template v-if="coreFeatures.includes('enableAudit')">
              <el-divider content-position="left">审计策略</el-divider>
              <el-form-item label="审计策略文件">
                <el-input
                  v-model="deployConfig.coreComponentsConfig.auditPolicy"
                  type="textarea"
                  :rows="5"
                  placeholder="输入审计策略 YAML 配置"
                />
              </el-form-item>
            </template>
          </el-form>
          </div>
        </el-collapse-item>

        <!-- ========== 步骤 4: 网络配置 ========== -->
        <el-collapse-item name="network" class="deploy-config-item">
          <template #title>
            <div class="config-item-title">
              <span class="config-item-icon"><el-icon><component :is="stepsMeta[3].icon" /></el-icon></span>
              <span class="config-item-text">
                <span class="config-item-name">{{ stepsMeta[3].title }}</span>
                <span class="config-item-desc">{{ getSectionDesc('network', stepsMeta[3].desc) }}</span>
              </span>
            </div>
          </template>
          <div class="step-section">
          <el-form
            ref="step4FormRef"
            :model="deployConfig.networkConfig"
            :rules="step4Rules"
            label-position="top"
          >
            <el-row :gutter="20">
              <el-col :span="8">
                <el-form-item label="网络插件" prop="networkPlugin">
                  <el-select
                    v-model="deployConfig.networkConfig.networkPlugin"
                    style="width: 100%"
                  >
                    <el-option label="Calico" value="calico" />
                    <el-option label="Flannel" value="flannel" />
                    <el-option label="Cilium（未接入）" value="cilium" disabled />
                    <el-option label="Weave（未接入）" value="weave" disabled />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="代理模式" prop="proxyMode">
                  <el-select
                    v-model="deployConfig.networkConfig.proxyMode"
                    style="width: 100%"
                  >
                    <el-option label="iptables" value="iptables" />
                    <el-option label="ipvs" value="ipvs" />
                  </el-select>
                </el-form-item>
              </el-col>
            </el-row>

            <el-row :gutter="20">
              <el-col :span="8">
                <el-form-item label="Pod CIDR" prop="podCIDR">
                  <el-input v-model="deployConfig.networkConfig.podCIDR" placeholder="10.244.0.0/16" />
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="Service CIDR" prop="serviceCIDR">
                  <el-input v-model="deployConfig.networkConfig.serviceCIDR" placeholder="10.96.0.0/12" />
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="DNS Service IP" prop="dnsServiceIP">
                  <el-input v-model="deployConfig.networkConfig.dnsServiceIP" placeholder="10.96.0.10" />
                </el-form-item>
              </el-col>
            </el-row>

            <el-row :gutter="20">
              <el-col :span="8">
                <el-form-item label="集群域名" prop="clusterDomain">
                  <el-input v-model="deployConfig.networkConfig.clusterDomain" placeholder="cluster.local" />
                </el-form-item>
              </el-col>
            </el-row>

            <!-- Calico 特有配置 -->
            <template v-if="deployConfig.networkConfig.networkPlugin === 'calico'">
              <el-divider content-position="left">Calico 参数</el-divider>
              <el-row :gutter="20">
                <el-col :span="8">
                  <el-form-item label="VXLAN 模式">
                    <el-switch v-model="calicoConfigModel.vxlanMode" />
                  </el-form-item>
                </el-col>
                <el-col :span="8">
                  <el-form-item label="MTU 值">
                    <el-input-number
                      v-model="calicoConfigModel.mtu"
                      :min="1200"
                      :max="9000"
                      :step="100"
                    />
                  </el-form-item>
                </el-col>
              </el-row>
            </template>

            <!-- Flannel 特有配置 -->
            <template v-if="deployConfig.networkConfig.networkPlugin === 'flannel'">
              <el-divider content-position="left">Flannel 参数</el-divider>
              <el-row :gutter="20">
                <el-col :span="8">
                  <el-form-item label="后端类型">
                    <el-select v-model="flannelConfigModel.backend" style="width: 100%">
                      <el-option label="VXLAN" value="vxlan" />
                      <el-option label="Host-GW" value="host-gw" />
                      <el-option label="UDP" value="udp" />
                    </el-select>
                  </el-form-item>
                </el-col>
              </el-row>
            </template>

            <el-divider content-position="left">默认容器镜像与版本</el-divider>
            <p class="component-catalog-hint">
              与后端 inventory / 离线包合并逻辑一致。内网无法直连公网时，请预拉下表「镜像」列；全量
              <code>calico.yaml</code> 见下表「说明」或 API 中 docs。
            </p>
            <el-table
              v-loading="k8sComponentCatalogLoading"
              :data="k8sComponentCatalogImages"
              border
              size="small"
              max-height="360"
              class="k8s-component-catalog-table"
            >
              <el-table-column prop="component" label="组件" min-width="120" show-overflow-tooltip />
              <el-table-column prop="version" label="版本" width="100" />
              <el-table-column prop="image" label="镜像（预拉/对照）" min-width="200" show-overflow-tooltip />
              <el-table-column prop="notes" label="说明" min-width="160" show-overflow-tooltip />
            </el-table>
            <p v-if="k8sComponentCatalogDocs.length" class="component-catalog-hint">
              附加：<span v-for="(d, i) in k8sComponentCatalogDocs" :key="d.key"
                >{{ d.description }} — <code>{{ d.value }}</code
                >{{ i < k8sComponentCatalogDocs.length - 1 ? '；' : '' }}</span
              >
            </p>
          </el-form>
          </div>
        </el-collapse-item>

        <!-- ========== 步骤 5: 存储配置 ========== -->
        <el-collapse-item name="storage" class="deploy-config-item">
          <template #title>
            <div class="config-item-title">
              <span class="config-item-icon"><el-icon><component :is="stepsMeta[4].icon" /></el-icon></span>
              <span class="config-item-text">
                <span class="config-item-name">{{ stepsMeta[4].title }}</span>
                <span class="config-item-desc">{{ getSectionDesc('storage', stepsMeta[4].desc) }}</span>
              </span>
            </div>
          </template>
          <div class="step-section">
          <el-form
            :model="deployConfig.storageConfig"
            label-position="top"
          >
            <el-row :gutter="20">
              <el-col :span="8">
                <el-form-item label="默认存储类">
                  <el-switch v-model="deployConfig.storageConfig.defaultStorageClass" />
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="存储供应器">
                  <el-select
                    v-model="deployConfig.storageConfig.storageProvisioner"
                    style="width: 100%"
                  >
                    <el-option label="本地路径" value="local-path" />
                    <el-option label="NFS 客户端" value="nfs-client" />
                    <el-option label="CSI" value="csi" />
                  </el-select>
                </el-form-item>
              </el-col>
            </el-row>

            <!-- local-path -->
            <template v-if="deployConfig.storageConfig.storageProvisioner === 'local-path'">
              <el-divider content-position="left">本地路径配置</el-divider>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="路径">
                    <el-input
                      v-model="localPathConfigModel.path"
                      placeholder="/var/lib/local-path-provisioner"
                    />
                  </el-form-item>
                </el-col>
              </el-row>
            </template>

            <!-- NFS -->
            <template v-if="deployConfig.storageConfig.storageProvisioner === 'nfs-client'">
              <el-divider content-position="left">NFS 配置</el-divider>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="NFS 服务器 IP">
                    <el-input v-model="nfsConfigModel.server" placeholder="NFS 服务器 IP" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="NFS 共享路径">
                    <el-input v-model="nfsConfigModel.path" placeholder="/data/nfs" />
                  </el-form-item>
                </el-col>
              </el-row>
            </template>

            <!-- CSI -->
            <template v-if="deployConfig.storageConfig.storageProvisioner === 'csi'">
              <el-divider content-position="left">CSI 配置</el-divider>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="CSI 驱动名称">
                    <el-input v-model="csiConfigModel.driver" placeholder="csi.aliyun.com" />
                  </el-form-item>
                </el-col>
                <el-col :span="8">
                  <el-form-item label="控制器数量">
                    <el-input-number
                      v-model="csiConfigModel.controllerCount"
                      :min="1"
                      :max="5"
                    />
                  </el-form-item>
                </el-col>
              </el-row>
            </template>
          </el-form>
          </div>
        </el-collapse-item>

        <!-- ========== 步骤 6: 高级配置 ========== -->
        <el-collapse-item name="advanced" class="deploy-config-item">
          <template #title>
            <div class="config-item-title">
              <span class="config-item-icon"><el-icon><component :is="stepsMeta[5].icon" /></el-icon></span>
              <span class="config-item-text">
                <span class="config-item-name">{{ stepsMeta[5].title }}</span>
                <span class="config-item-desc">{{ getSectionDesc('advanced', stepsMeta[5].desc) }}</span>
              </span>
            </div>
          </template>
          <div class="step-section">
          <el-form
            :model="deployConfig.advancedConfig"
            label-position="top"
          >
            <el-form-item label="可选组件">
              <el-checkbox-group v-model="advancedComponents">
                <el-checkbox value="enableNodeLocalDNS">NodeLocal DNS</el-checkbox>
                <el-checkbox value="enableMetricsServer">Metrics Server</el-checkbox>
                <el-checkbox value="enableDashboard">Kubernetes Dashboard</el-checkbox>
                <el-checkbox value="enablePrometheus">Prometheus</el-checkbox>
                <el-checkbox value="enableIngressNginx">Ingress Nginx</el-checkbox>
                <el-checkbox value="enableHelm">Helm</el-checkbox>
              </el-checkbox-group>
            </el-form-item>

            <el-form-item label="部署前环境清理">
              <el-switch v-model="deployConfig.advancedConfig.preDeployCleanup" />
              <p class="pre-cleanup-hint">
                开启后 Step 0 会非交互清理各节点旧 K8s/etcd 数据，便于重复装。执行时可设
                <code>OPSFLEET_OFFLINE_PRE_CLEANUP=0</code> 跳过。
              </p>
            </el-form-item>

            <el-divider content-position="left">额外启动参数</el-divider>

            <div class="extra-args-grid">
              <el-card class="sub-card" shadow="hover">
                <template #header>
                  <div class="sub-card-header"><span>Kubelet 额外参数</span></div>
                </template>
                <KeyValueGroup v-model="extraKubeletArgsModel" />
              </el-card>
              <el-card class="sub-card" shadow="hover">
                <template #header>
                  <div class="sub-card-header"><span>KubeProxy 额外参数</span></div>
                </template>
                <KeyValueGroup v-model="extraKubeProxyArgsModel" />
              </el-card>
              <el-card class="sub-card" shadow="hover">
                <template #header>
                  <div class="sub-card-header"><span>API Server 额外参数</span></div>
                </template>
                <KeyValueGroup v-model="extraAPIServerArgsModel" />
              </el-card>
            </div>
          </el-form>
          </div>
        </el-collapse-item>

        <!-- ========== 步骤 7: 部署确认（精简：核心命令 + 摘要） ========== -->
        <el-collapse-item name="confirm" class="deploy-config-item deploy-config-item--confirm">
          <template #title>
            <div class="config-item-title">
              <span class="config-item-icon"><el-icon><component :is="stepsMeta[6].icon" /></el-icon></span>
              <span class="config-item-text">
                <span class="config-item-name">{{ stepsMeta[6].title }}</span>
                <span class="config-item-desc">{{ getSectionDesc('confirm', stepsMeta[6].desc) }}</span>
              </span>
            </div>
          </template>
          <div class="step-section step-section--confirm">
          <p v-if="offlineBundleMode" class="confirm-lead">
            在<strong>一台控制机</strong>上执行（须已对本单各节点 <strong>root 免密 SSH</strong>）。安装集群任选：先装
            <code>ai-sre</code> 再执行一键命令，或直接用下方「集群安装」的 curl（需 python3）。
          </p>
          <p v-else class="confirm-lead confirm-lead--online">
            核对后点击<strong>开始在线部署</strong>；进度见「部署进度」。
          </p>

          <!-- 离线：集群一键安装（生成后出现） -->
          <div v-if="offlineBundleMode" class="confirm-cmd-card confirm-cmd-card--cluster">
            <div class="confirm-cmd-card__head">
              <span class="confirm-cmd-card__title">安装 Kubernetes 集群（控制机）</span>
              <template v-if="lastInvite">
                <el-button type="primary" size="small" link @click.stop="copyBootstrapCommand">
                  <el-icon class="btn-icon-left"><DocumentCopy /></el-icon>
                  复制推荐命令
                </el-button>
              </template>
            </div>
            <p v-if="!lastInvite" class="offline-install-panel__placeholder">
              点击底部<strong>「生成一键安装命令」</strong>后出现命令（推荐，无需 ai-sre）。亦可下载 zip 后
              <code>sudo bash install.sh</code>。
            </p>
            <template v-else>
              <p class="confirm-cmd-card__meta">
                资源 ID {{ lastInvite.id }} · 有效期至 {{ formatInviteExpiry(lastInvite.expiresAt) }}
              </p>
              <p class="confirm-cmd-card__hint">推荐（curl + python3，与 <code>ai-sre k8s install</code> 等价）。命令含密钥勿外泄。</p>
              <el-input
                type="textarea"
                :rows="3"
                readonly
                :model-value="lastInvite.bootstrapCommand"
                class="install-command-textarea"
              />
              <el-collapse v-model="optionalClusterCmdOpen" class="confirm-optional-collapse">
                <el-collapse-item title="可选：已安装 ai-sre 时" name="a">
                  <el-input
                    type="textarea"
                    :rows="2"
                    readonly
                    :model-value="lastInvite.installCommand"
                    class="install-command-textarea"
                  />
                  <el-button type="primary" size="small" link class="confirm-optional-copy" @click.stop="copyInstallCommand">
                    复制
                  </el-button>
                </el-collapse-item>
                <el-collapse-item title="全节点清理（部署失败或重置环境）" name="cleanup">
                  <p class="confirm-cmd-card__hint">
                    与页面「节点配置」中的 master/worker 一致：重新拉取同一离线包，对 inventory 中全部节点执行
                    <code>pre_cleanup</code>（停止 kubelet/etcd 等并删除数据目录）。引用需在有效期内；须已对各节点 root 免密。
                  </p>
                  <el-input
                    type="textarea"
                    :rows="2"
                    readonly
                    :model-value="lastInvite.cleanupCommand"
                    class="install-command-textarea"
                  />
                  <el-button type="primary" size="small" link class="confirm-optional-copy" @click.stop="copyCleanupCommand">
                    复制清理命令
                  </el-button>
                </el-collapse-item>
              </el-collapse>
            </template>
          </div>

          <el-collapse v-model="confirmAuxOpen" class="confirm-aux-collapse">
            <el-collapse-item title="部署需求说明（复制，不含密钥）" name="doc">
              <el-button type="primary" size="small" @click.stop="copyDeployRequirement">
                <el-icon class="btn-icon-left"><DocumentCopy /></el-icon>
                复制全文
              </el-button>
              <pre class="requirement-pre requirement-pre--compact" tabindex="0">{{ deployRequirementText }}</pre>
            </el-collapse-item>
          </el-collapse>

          <el-descriptions
            v-if="offlineBundleMode"
            title="配置摘要"
            :column="2"
            size="small"
            border
            class="confirm-summary-desc"
          >
            <el-descriptions-item label="集群">
              {{ deployConfig.clusterBasicInfo.clusterName || '—' }}
            </el-descriptions-item>
            <el-descriptions-item label="版本 / 架构">
              {{ deployConfig.clusterBasicInfo.version || '—' }} · {{ deployConfig.clusterBasicInfo.cpuArch || '—' }}
            </el-descriptions-item>
            <el-descriptions-item label="模式">
              {{ deployConfig.clusterBasicInfo.deployMode === 'cluster' ? '多节点' : '单节点' }}
            </el-descriptions-item>
            <el-descriptions-item label="镜像">
              {{ imageSourceText }}
            </el-descriptions-item>
            <el-descriptions-item label="控制平面" :span="2">
              <span class="confirm-value--wrap">{{ executorConfirmText }}</span>
            </el-descriptions-item>
            <el-descriptions-item label="工作节点" :span="2">
              <span class="confirm-value--wrap">{{ confirmWorkerPreview }}</span>
            </el-descriptions-item>
            <el-descriptions-item label="网络" :span="2">
              {{ deployConfig.networkConfig.networkPlugin }} · Pod {{ deployConfig.networkConfig.podCIDR }} · Service
              {{ deployConfig.networkConfig.serviceCIDR }}
            </el-descriptions-item>
            <el-descriptions-item label="存储">
              {{ deployConfig.storageConfig.storageProvisioner }} · 默认 SC
              {{ deployConfig.storageConfig.defaultStorageClass ? '开' : '关' }}
            </el-descriptions-item>
            <el-descriptions-item label="Step0 清理">
              {{ deployConfig.advancedConfig.preDeployCleanup ? '是' : '否' }}
            </el-descriptions-item>
            <el-descriptions-item v-if="enabledComponentsText" label="可选组件" :span="2">
              {{ enabledComponentsText }}
            </el-descriptions-item>
            <el-descriptions-item
              v-if="deployConfig.clusterBasicInfo.downloadDomain?.trim() || deployConfig.clusterBasicInfo.downloadProtocol?.trim()"
              label="制品覆盖"
              :span="2"
            >
              {{ deployConfig.clusterBasicInfo.downloadProtocol?.trim() || '默认' }}
              {{ deployConfig.clusterBasicInfo.downloadDomain?.trim() || '—' }}
            </el-descriptions-item>
          </el-descriptions>

          <el-descriptions v-else title="配置摘要" :column="2" size="small" border class="confirm-summary-desc">
            <el-descriptions-item label="集群">
              {{ deployConfig.clusterBasicInfo.clusterName || '—' }}
            </el-descriptions-item>
            <el-descriptions-item label="版本">
              {{ deployConfig.clusterBasicInfo.version || '—' }}
            </el-descriptions-item>
            <el-descriptions-item label="执行节点" :span="2">
              {{ executorConfirmText }}
            </el-descriptions-item>
            <el-descriptions-item label="控制平面 / 工作节点">
              {{ deployConfig.nodeConfig.masterNodes.length }} / {{ deployConfig.nodeConfig.workerNodes.length }} 台
            </el-descriptions-item>
            <el-descriptions-item label="网络" :span="2">
              {{ deployConfig.networkConfig.networkPlugin }} · Pod {{ deployConfig.networkConfig.podCIDR }}
            </el-descriptions-item>
          </el-descriptions>
          </div>
        </el-collapse-item>
        </el-collapse>
      </main>

    </div>

    <!-- ==================== 底部操作栏 ==================== -->
    <div class="step-actions">
      <div class="action-spacer" />
      <el-button
        v-if="offlineBundleMode"
        type="success"
        size="large"
        class="primary-finish-btn"
        :loading="creatingInvite"
        :disabled="downloadingBundle"
        @click="handleCreateInstallRef"
      >
        <el-icon class="primary-finish-btn-icon"><Promotion /></el-icon>
        生成一键安装命令
      </el-button>
      <el-button
        v-if="offlineBundleMode"
        type="primary"
        size="large"
        class="primary-finish-btn"
        :loading="downloadingBundle"
        :disabled="creatingInvite"
        @click="handleDownloadBundle"
      >
        <el-icon class="primary-finish-btn-icon"><Download /></el-icon>
        下载离线安装包（zip）
      </el-button>
      <el-button
        v-else
        type="success"
        size="large"
        class="primary-finish-btn"
        :loading="submitting"
        @click="submitDeploy"
      >
        开始在线部署
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted, markRaw } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import type { FormInstance } from 'element-plus'
import {
  Monitor,
  Cpu,
  SetUp,
  Connection,
  Coin,
  Operation,
  CircleCheck,
  Download,
  Promotion,
  DocumentCopy
} from '@element-plus/icons-vue'
import NodeSelect from '@/components/k8s/NodeSelect.vue'
import LabelGroup from '@/components/k8s/LabelGroup.vue'
import TaintGroup from '@/components/k8s/TaintGroup.vue'
import KeyValueGroup from '@/components/k8s/KeyValueGroup.vue'
import {
  getK8sVersions,
  getK8sComponentCatalog,
  checkClusterName,
  submitDeployConfig,
  downloadOfflineBundle,
  createK8sBundleInvite
} from '../../../api/k8s-deploy'
import type {
  DeployConfig,
  K8sMachineInfo,
  K8sVersion,
  KeyValuePair,
  Taint
} from '../../../types/k8s-deploy'
import { useK8sDeployStore } from '../../../stores/k8s-deploy'
import { useMachineStore } from '../../../stores/machine'

const router = useRouter()
const k8sDeployStore = useK8sDeployStore()
const machineStore = useMachineStore()

/** 非 HTTPS / 部分浏览器下 Clipboard API 不可用，降级到 execCommand */
async function copyTextToClipboard(text: string): Promise<void> {
  const fallback = (): void => {
    const ta = document.createElement('textarea')
    ta.value = text
    ta.setAttribute('readonly', '')
    ta.style.position = 'fixed'
    ta.style.left = '-9999px'
    document.body.appendChild(ta)
    ta.select()
    try {
      if (!document.execCommand('copy')) {
        throw new Error('execCommand copy failed')
      }
    } finally {
      document.body.removeChild(ta)
    }
  }

  if (navigator.clipboard && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(text)
      return
    } catch {
      // 权限或策略失败时再试降级
    }
  }
  fallback()
}

// ---------- 步骤元信息（统一每步的标题、描述、图标） ----------
const stepsMeta = [
  {
    title: '基础集群信息',
    desc: '集群名称、版本、架构、镜像源与可选制品覆盖',
    icon: markRaw(Monitor)
  },
  {
    title: '节点配置',
    desc: '离线填 IP 或在线选 Agent 与节点；标签与污点可选',
    icon: markRaw(Cpu)
  },
  { title: '核心组件配置', desc: 'kube-proxy、RBAC、审计等', icon: markRaw(SetUp) },
  { title: '网络配置', desc: 'CNI、Pod/Service CIDR', icon: markRaw(Connection) },
  { title: '存储配置', desc: '存储类与供应器', icon: markRaw(Coin) },
  { title: '高级配置', desc: '可选组件与额外参数', icon: markRaw(Operation) },
  {
    title: '部署确认',
    desc: '离线：固定 curl 安装 ai-sre、生成集群命令、摘要',
    icon: markRaw(CircleCheck)
  }
] as const

// ---------- 表单 Refs ----------
const step1FormRef = ref<FormInstance>()
const step4FormRef = ref<FormInstance>()

type SectionName = 'precheck' | 'basic' | 'nodes' | 'core' | 'network' | 'storage' | 'advanced' | 'confirm'

// ---------- 状态 ----------
const activeStep = ref(0)
const submitting = ref(false)
const downloadingBundle = ref(false)
const creatingInvite = ref(false)
const openSections = ref<SectionName>('precheck')
const k8sComponentCatalogLoading = ref(false)
const k8sComponentCatalogImages = ref<
  { component: string; versionKey: string; version: string; image: string; notes: string }[]
>([])
const k8sComponentCatalogDocs = ref<{ key: string; value: string; description: string }[]>([])
/**
 * 部署 K8s 前的环境预检清单。来自本仓库踩坑：
 *   1) 虚拟机 RTC 漂移 → systemd-timesyncd/chrony 跳变 → kubelet 误判 sandbox 过期 → calico-node / coredns 60s 左右被 Killing；
 *   2) swap 未关 / br_netfilter 未加载 / ip_forward=0 → kubelet 拒绝启动或 pod 跨节点不通；
 *   3) hostname 冲突或 cgroup v1（systemd driver 不一致）→ kubelet+containerd 频繁崩溃。
 * 前端只做静态展示（真正校验在 CLI `ai-sre k8s diagnose` 中）。
 */
const preflightRows = [
  {
    item: '时钟同步 (chrony/NTP)',
    why: 'RTC 漂移会让 kubelet 误判 sandbox 过期，calico-node/coredns 会每 60-70s 被 Killing',
    cmd: 'timedatectl show -p NTPSynchronized --value',
    expected: 'yes',
  },
  {
    item: '历史启动的时钟跳变',
    why: '同一 boot 中时钟后退几小时是本仓库最常见的 calico 循环重启根因',
    cmd: 'journalctl --list-boots | head -5',
    expected: '最近一次开机后无 hours 级回拨',
  },
  {
    item: '关闭 swap',
    why: 'swap 打开时 kubelet 默认拒启动；或 eviction 异常',
    cmd: 'swapon --show',
    expected: '空输出',
  },
  {
    item: '内核模块 br_netfilter / overlay',
    why: '缺失则 iptables 规则对桥接流量不生效，Service 不通',
    cmd: 'lsmod | grep -E "^(br_netfilter|overlay) "',
    expected: '两行都存在',
  },
  {
    item: 'sysctl 网络参数',
    why: 'ip_forward=0 或 bridge-nf-call-iptables=0 → 跨节点 Pod 流量被丢弃',
    cmd: 'sysctl net.ipv4.ip_forward net.bridge.bridge-nf-call-iptables',
    expected: '都等于 1',
  },
  {
    item: '可用内存 ≥ 4GiB (master 推荐 ≥ 8GiB)',
    why: '内存不足会导致 etcd fsync 抖动、kubelet OOM、sandbox 反复重建',
    cmd: 'free -g | awk "/Mem:/ {print $2\\\" GiB total, \\\"$7\\\" GiB available\\\"}"',
    expected: '充足',
  },
  {
    item: '架构与离线包一致 (amd64 / arm64)',
    why: '架构不匹配会在 pause/etcd 镜像加载时直接报 exec format error',
    cmd: 'uname -m',
    expected: '与 archVersion 一致',
  },
  {
    item: '节点间主机名唯一',
    why: '同 hostname 时 kubelet 会互相抢同一个 Node 对象',
    cmd: 'hostname',
    expected: '每节点不同',
  },
  {
    item: '控制机 → 各节点 root 免密 SSH',
    why: 'Ansible 以 ansible_user=root 连接；未免密 install.sh 将 Permission denied',
    cmd: 'ssh -o BatchMode=yes root@<节点IP> true',
    expected: '无提示直接返回',
  },
]
/** 部署确认：需求说明折叠，默认收起 */
const confirmAuxOpen = ref<string[]>([])
/** 已生成邀请时，「可选 ai-sre」命令折叠 */
const optionalClusterCmdOpen = ref<string[]>([])
/** 离线一键安装接口返回，用于最后一步展示 */
const lastInvite = ref<{
  id: string
  expiresAt: string
  installRef: string
  installCommand: string
  bootstrapCommand: string
  cleanupCommand: string
} | null>(null)
/** true：离线 zip（推荐）；false：经 Agent 在线部署 */
const offlineBundleMode = ref(true)
const masterHostsText = ref('')
const workerHostsText = ref('')
const k8sVersions = ref<K8sVersion[]>([])

// 使用 machineStore 作为数据源，与机器管理页共享状态，WebSocket 心跳会实时更新 status
const BYTES_TO_GB = 1024 ** 3
const machines = computed<K8sMachineInfo[]>(() => {
  const list = machineStore.machineList
  return list.map((m): K8sMachineInfo => {
    const memoryGb = (m.memory != null && m.memory > 0)
      ? m.memory
      : (m.memory_total != null ? Math.round((m.memory_total / BYTES_TO_GB) * 10) / 10 : 0)
    const diskGb = (m.disk != null && m.disk > 0)
      ? m.disk
      : (m.disk_total != null ? Math.round((m.disk_total / BYTES_TO_GB) * 10) / 10 : 0)
    return {
      id: m.id,
      name: m.name,
      ip: m.ip,
      cpu: m.cpu_cores ?? m.cpu ?? 0,
      memory: memoryGb,
      disk: diskGb,
      status: m.status,
      node_role: m.node_role,
      client_id: m.client_id
    }
  })
})

// 可选的执行节点：在线 + 有 client_id（已安装 Agent）
const selectableExecutors = computed(() =>
  machines.value.filter(m => m.status === 'online' && !!m.client_id)
)

// ---------- 部署配置（完整结构） ----------
const deployConfig = reactive<DeployConfig>({
  clusterBasicInfo: {
    clusterName: '',
    version: '',
    deployMode: 'cluster',
    cpuArch: 'arm64',
    imageSource: 'aliyun',
    downloadDomain: '',
    downloadProtocol: ''
  },
  nodeConfig: {
    executorNode: '' as string,
    masterNodes: [] as string[],
    workerNodes: [] as string[],
    masterHosts: [] as string[],
    workerHosts: [] as string[],
    masterLabels: {},
    workerLabels: {},
    masterTaints: [],
    workerTaints: []
  },
  coreComponentsConfig: {
    kubeProxyMode: 'iptables',
    enablePodSecurityPolicy: false,
    enableRBAC: true,
    enableAudit: false
  },
  networkConfig: {
    networkPlugin: 'calico',
    podCIDR: '10.244.0.0/16',
    serviceCIDR: '10.96.0.0/12',
    dnsServiceIP: '10.96.0.10',
    clusterDomain: 'cluster.local',
    proxyMode: 'iptables',
    calicoConfig: {
      vxlanMode: true,
      mtu: 1450
    },
    flannelConfig: {
      backend: 'vxlan'
    }
  },
  storageConfig: {
    defaultStorageClass: true,
    storageProvisioner: 'local-path',
    localPathConfig: { path: '/var/lib/local-path-provisioner' },
    nfsConfig: { server: '', path: '' },
    csiConfig: { driver: '', controllerCount: 1 }
  },
  advancedConfig: {
    enableNodeLocalDNS: false,
    enableMetricsServer: true,
    enableDashboard: false,
    enablePrometheus: false,
    enableIngressNginx: false,
    enableHelm: true,
    preDeployCleanup: false,
    extraKubeletArgs: [],
    extraKubeProxyArgs: [],
    extraAPIServerArgs: []
  }
})

const masterLabelsModel = computed<Record<string, string>>({
  get: () => deployConfig.nodeConfig.masterLabels ?? {},
  set: value => { deployConfig.nodeConfig.masterLabels = value }
})

const workerLabelsModel = computed<Record<string, string>>({
  get: () => deployConfig.nodeConfig.workerLabels ?? {},
  set: value => { deployConfig.nodeConfig.workerLabels = value }
})

const masterTaintsModel = computed<Taint[]>({
  get: () => deployConfig.nodeConfig.masterTaints ?? [],
  set: value => { deployConfig.nodeConfig.masterTaints = value }
})

const workerTaintsModel = computed<Taint[]>({
  get: () => deployConfig.nodeConfig.workerTaints ?? [],
  set: value => { deployConfig.nodeConfig.workerTaints = value }
})

const calicoConfigModel = computed({
  get: () => {
    deployConfig.networkConfig.calicoConfig ??= { vxlanMode: true, mtu: 1450 }
    return deployConfig.networkConfig.calicoConfig
  },
  set: value => { deployConfig.networkConfig.calicoConfig = value }
})

const flannelConfigModel = computed({
  get: () => {
    deployConfig.networkConfig.flannelConfig ??= { backend: 'vxlan' }
    return deployConfig.networkConfig.flannelConfig
  },
  set: value => { deployConfig.networkConfig.flannelConfig = value }
})

const localPathConfigModel = computed({
  get: () => {
    deployConfig.storageConfig.localPathConfig ??= { path: '/var/lib/local-path-provisioner' }
    return deployConfig.storageConfig.localPathConfig
  },
  set: value => { deployConfig.storageConfig.localPathConfig = value }
})

const nfsConfigModel = computed({
  get: () => {
    deployConfig.storageConfig.nfsConfig ??= { server: '', path: '' }
    return deployConfig.storageConfig.nfsConfig
  },
  set: value => { deployConfig.storageConfig.nfsConfig = value }
})

const csiConfigModel = computed({
  get: () => {
    deployConfig.storageConfig.csiConfig ??= { driver: '', controllerCount: 1 }
    return deployConfig.storageConfig.csiConfig
  },
  set: value => { deployConfig.storageConfig.csiConfig = value }
})

const extraKubeletArgsModel = computed<KeyValuePair[]>({
  get: () => deployConfig.advancedConfig.extraKubeletArgs ?? [],
  set: value => { deployConfig.advancedConfig.extraKubeletArgs = value }
})

const extraKubeProxyArgsModel = computed<KeyValuePair[]>({
  get: () => deployConfig.advancedConfig.extraKubeProxyArgs ?? [],
  set: value => { deployConfig.advancedConfig.extraKubeProxyArgs = value }
})

const extraAPIServerArgsModel = computed<KeyValuePair[]>({
  get: () => deployConfig.advancedConfig.extraAPIServerArgs ?? [],
  set: value => { deployConfig.advancedConfig.extraAPIServerArgs = value }
})

// ---------- checkbox ↔ boolean 双向绑定 ----------
const coreFeatures = computed({
  get: () => {
    const f: string[] = []
    if (deployConfig.coreComponentsConfig.enableRBAC) f.push('enableRBAC')
    if (deployConfig.coreComponentsConfig.enablePodSecurityPolicy) f.push('enablePodSecurityPolicy')
    if (deployConfig.coreComponentsConfig.enableAudit) f.push('enableAudit')
    return f
  },
  set: (v: string[]) => {
    deployConfig.coreComponentsConfig.enableRBAC = v.includes('enableRBAC')
    deployConfig.coreComponentsConfig.enablePodSecurityPolicy = v.includes('enablePodSecurityPolicy')
    deployConfig.coreComponentsConfig.enableAudit = v.includes('enableAudit')
  }
})

const advancedComponents = computed({
  get: () => {
    const c: string[] = []
    if (deployConfig.advancedConfig.enableNodeLocalDNS) c.push('enableNodeLocalDNS')
    if (deployConfig.advancedConfig.enableMetricsServer) c.push('enableMetricsServer')
    if (deployConfig.advancedConfig.enableDashboard) c.push('enableDashboard')
    if (deployConfig.advancedConfig.enablePrometheus) c.push('enablePrometheus')
    if (deployConfig.advancedConfig.enableIngressNginx) c.push('enableIngressNginx')
    if (deployConfig.advancedConfig.enableHelm) c.push('enableHelm')
    return c
  },
  set: (v: string[]) => {
    deployConfig.advancedConfig.enableNodeLocalDNS = v.includes('enableNodeLocalDNS')
    deployConfig.advancedConfig.enableMetricsServer = v.includes('enableMetricsServer')
    deployConfig.advancedConfig.enableDashboard = v.includes('enableDashboard')
    deployConfig.advancedConfig.enablePrometheus = v.includes('enablePrometheus')
    deployConfig.advancedConfig.enableIngressNginx = v.includes('enableIngressNginx')
    deployConfig.advancedConfig.enableHelm = v.includes('enableHelm')
  }
})

// ---------- 表单验证规则 ----------
const CIDR_PATTERN = /^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\/\d{1,2}$/
const IP_PATTERN = /^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$/

const step1Rules = computed(() => ({
  clusterName: [
    { required: true, message: '请输入集群名称', trigger: 'blur' },
    { min: 2, max: 30, message: '长度在 2 到 30 个字符', trigger: 'blur' }
  ],
  version: [
    { required: true, message: '请选择 K8s 版本', trigger: 'change' }
  ],
  cpuArch: [
    { required: true, message: '请选择 CPU 架构', trigger: 'change' }
  ],
  imageSource: [
    { required: true, message: '请选择镜像源', trigger: 'change' }
  ],
  customRegistry: deployConfig.clusterBasicInfo.imageSource === 'custom'
    ? [{ required: true, message: '请输入自定义镜像仓库地址', trigger: 'blur' }]
    : []
}))

const step4Rules = {
  networkPlugin: [{ required: true, message: '请选择网络插件', trigger: 'change' }],
  podCIDR: [
    { required: true, message: '请输入 Pod CIDR', trigger: 'blur' },
    { pattern: CIDR_PATTERN, message: '格式: x.x.x.x/xx', trigger: 'blur' }
  ],
  serviceCIDR: [
    { required: true, message: '请输入 Service CIDR', trigger: 'blur' },
    { pattern: CIDR_PATTERN, message: '格式: x.x.x.x/xx', trigger: 'blur' }
  ],
  dnsServiceIP: [
    { required: true, message: '请输入 DNS Service IP', trigger: 'blur' },
    { pattern: IP_PATTERN, message: '请输入有效 IP 地址', trigger: 'blur' }
  ]
}

// ---------- 确认页计算属性 ----------
const executorConfirmText = computed(() => {
  if (offlineBundleMode.value) {
    const m = deployConfig.nodeConfig.masterHosts?.length
      ? deployConfig.nodeConfig.masterHosts.join(', ')
      : masterHostsText.value.trim().split(/\r?\n/).map(s => s.trim()).filter(Boolean).join(', ')
    return m || '（请在节点步骤填写 IP）'
  }
  const id = deployConfig.nodeConfig.executorNode
  if (!id) return '未选择'
  const mm = machines.value.find(x => x.id === id)
  return mm ? `${mm.name || '未命名'} (${mm.ip})` : id.slice(0, 8) + '…'
})

const imageSourceText = computed(() => {
  const m: Record<string, string> = { default: '默认源', aliyun: '阿里云', tencent: '腾讯云', custom: '自定义' }
  return m[deployConfig.clusterBasicInfo.imageSource] || deployConfig.clusterBasicInfo.imageSource
})

const enabledComponentsText = computed(() => {
  const items: string[] = []
  if (deployConfig.advancedConfig.enableNodeLocalDNS) items.push('NodeLocal DNS')
  if (deployConfig.advancedConfig.enableMetricsServer) items.push('Metrics Server')
  if (deployConfig.advancedConfig.enableDashboard) items.push('Dashboard')
  if (deployConfig.advancedConfig.enablePrometheus) items.push('Prometheus')
  if (deployConfig.advancedConfig.enableIngressNginx) items.push('Ingress Nginx')
  if (deployConfig.advancedConfig.enableHelm) items.push('Helm')
  return items.join('、')
})

const sectionSummary = computed<Record<SectionName, string>>(() => {
  const basic = deployConfig.clusterBasicInfo
  const node = deployConfig.nodeConfig
  const core = deployConfig.coreComponentsConfig
  const network = deployConfig.networkConfig
  const storage = deployConfig.storageConfig
  const advanced = deployConfig.advancedConfig
  const masters = offlineBundleMode.value ? parseHostLines(masterHostsText.value).length : node.masterNodes.length
  const workers = offlineBundleMode.value ? parseHostLines(workerHostsText.value).length : node.workerNodes.length
  const advancedEnabled = [
    advanced.enableNodeLocalDNS,
    advanced.enableMetricsServer,
    advanced.enableDashboard,
    advanced.enablePrometheus,
    advanced.enableIngressNginx,
    advanced.enableHelm
  ].filter(Boolean).length

  return {
    precheck: '已提供免密 SSH 与环境检查清单',
    basic: `${basic.clusterName || '未命名'} · ${basic.version || '未选版本'} · ${basic.cpuArch}`,
    nodes: offlineBundleMode.value
      ? `离线清单：Master ${masters} 台 / Worker ${workers} 台`
      : `在线部署：${node.executorNode ? '执行机已选' : '未选执行机'} · Master ${masters} / Worker ${workers}`,
    core: `kube-proxy=${core.kubeProxyMode} · RBAC=${core.enableRBAC ? '开' : '关'} · 审计=${core.enableAudit ? '开' : '关'}`,
    network: `${network.networkPlugin} · Pod ${network.podCIDR} · Svc ${network.serviceCIDR}`,
    storage: `${storage.storageProvisioner} · 默认存储类${storage.defaultStorageClass ? '开启' : '关闭'}`,
    advanced: `可选组件 ${advancedEnabled} 项 · Step0 ${advanced.preDeployCleanup ? '清理' : '不清理'}`,
    confirm: lastInvite.value?.installRef
      ? `安装引用已生成：${lastInvite.value.installRef}`
      : '待确认后可生成一键安装命令'
  }
})

function getSectionDesc(section: SectionName, fallback: string): string {
  return openSections.value === section ? fallback : sectionSummary.value[section]
}

const confirmWorkerPreview = computed(() => {
  const fromCfg = deployConfig.nodeConfig.workerHosts?.length
    ? deployConfig.nodeConfig.workerHosts
    : workerHostsText.value
        .split(/\r?\n/)
        .map((x) => x.trim())
        .filter(Boolean)
  return fromCfg.length ? fromCfg.join('、') : '无'
})

/** 与生成 bundle-invite 时使用的 publicApiBase 一致，供确认页展示 */
const publicApiBasePreview = computed(() =>
  `${window.location.origin}${import.meta.env.VITE_BASE_API || '/ft-api'}`.replace(/\/$/, '')
)

/** 全站固定：curl 安装 ai-sre（同源 API，脚本内再拉二进制） */
const installAiSreCurlCommand = computed(
  () => `curl -fsSL '${publicApiBasePreview.value}/api/k8s/deploy/install-ai-sre.sh' | sudo bash`
)

/** 部署确认页：精简需求说明（不含一键命令密钥） */
const deployRequirementText = computed(() => {
  const now = new Date().toLocaleString('zh-CN', { hour12: false })
  const name = deployConfig.clusterBasicInfo.clusterName?.trim() || '（未填写）'
  const ver = deployConfig.clusterBasicInfo.version || '—'
  const arch = deployConfig.clusterBasicInfo.cpuArch || 'amd64'
  const mode = deployConfig.clusterBasicInfo.deployMode === 'cluster' ? '多节点' : '单节点'
  const masters = offlineBundleMode.value
    ? deployConfig.nodeConfig.masterHosts?.length
      ? deployConfig.nodeConfig.masterHosts
      : parseHostLines(masterHostsText.value)
    : []
  const workers = offlineBundleMode.value
    ? deployConfig.nodeConfig.workerHosts?.length
      ? deployConfig.nodeConfig.workerHosts
      : parseHostLines(workerHostsText.value)
    : []
  const L: string[] = [
    `K8s 部署需求（OpsFleet ${now}）`,
    '',
    `集群「${name}」 ${ver} ${arch} ${mode} · 镜像 ${imageSourceText.value}`,
    '',
    offlineBundleMode.value
      ? `节点：控制平面 ${masters.length ? masters.join('、') : '（未填）'}；工作 ${workers.length ? workers.join('、') : '无'}`
      : `在线：执行 ${executorConfirmText.value}；控制/工作 ${deployConfig.nodeConfig.masterNodes.length}/${deployConfig.nodeConfig.workerNodes.length} 台`,
    '',
    `网络 ${deployConfig.networkConfig.networkPlugin} Pod ${deployConfig.networkConfig.podCIDR} Svc ${deployConfig.networkConfig.serviceCIDR}`,
    `存储 ${deployConfig.storageConfig.storageProvisioner} · Step0 ${deployConfig.advancedConfig.preDeployCleanup ? '清理' : '不清理'} · ${enabledComponentsText.value || '无可选组件'}`,
    '',
    '离线：控制机 root 免密 SSH 各节点；安装命令见页面（含密钥勿写入本文）。',
    '—— 表单生成，请业务与执行双方确认。',
  ]
  return L.join('\n')
})

// ---------- 状态持久化：跳转后返回仍保留已填写的步骤与数据 ----------
watch(
  () => ({ config: deployConfig, step: activeStep.value }),
  () => {
    k8sDeployStore.saveState(deployConfig as DeployConfig, activeStep.value)
  },
  { deep: true }
)

function formatInviteExpiry(iso: string): string {
  if (!iso) return '—'
  try {
    return new Date(iso).toLocaleString('zh-CN')
  } catch {
    return iso
  }
}

function copyInstallCommand() {
  const cmd = lastInvite.value?.installCommand
  if (!cmd) return
  void copyTextToClipboard(cmd).then(
    () => ElMessage.success('已复制 ai-sre 集群命令'),
    () => ElMessage.error('复制失败，请手动选择文本复制')
  )
}

function copyCleanupCommand() {
  const cmd = lastInvite.value?.cleanupCommand
  if (!cmd) return
  void copyTextToClipboard(cmd).then(
    () => ElMessage.success('已复制全节点清理命令'),
    () => ElMessage.error('复制失败，请手动选择文本复制')
  )
}

function copyBootstrapCommand() {
  const cmd = lastInvite.value?.bootstrapCommand
  if (!cmd) return
  void copyTextToClipboard(cmd).then(
    () => ElMessage.success('已复制集群安装命令'),
    () => ElMessage.error('复制失败，请手动选择文本复制')
  )
}

function copyInstallAiSreCurl() {
  const cmd = installAiSreCurlCommand.value
  if (!cmd?.trim()) return
  void copyTextToClipboard(cmd).then(
    () => ElMessage.success('已复制安装 ai-sre 命令'),
    () => ElMessage.error('复制失败')
  )
}

function copyDeployRequirement() {
  const text = deployRequirementText.value
  if (!text?.trim()) return
  void copyTextToClipboard(text).then(
    () => ElMessage.success('已复制部署需求说明'),
    () => ElMessage.error('复制失败，请手动全选文本复制')
  )
}

// ---------- 初始化 ----------
onMounted(() => {
  k8sDeployStore.restoreInto(deployConfig as DeployConfig, activeStep)
  if (deployConfig.nodeConfig.masterHosts?.length) {
    masterHostsText.value = deployConfig.nodeConfig.masterHosts.join('\n')
  }
  if (deployConfig.nodeConfig.workerHosts?.length) {
    workerHostsText.value = deployConfig.nodeConfig.workerHosts.join('\n')
  }
  loadK8sVersions()
  loadK8sComponentCatalog()
  loadMachines()
})

watch(offlineBundleMode, () => {
  lastInvite.value = null
})

const loadK8sComponentCatalog = async () => {
  k8sComponentCatalogLoading.value = true
  try {
    const data = await getK8sComponentCatalog()
    k8sComponentCatalogImages.value = data.images || []
    k8sComponentCatalogDocs.value = data.docs || []
  } catch (e: any) {
    k8sComponentCatalogImages.value = []
    k8sComponentCatalogDocs.value = []
    ElMessage.error('获取组件版本清单失败: ' + (e?.message || e))
  } finally {
    k8sComponentCatalogLoading.value = false
  }
}

const loadK8sVersions = async () => {
  try {
    const res = await getK8sVersions()
    k8sVersions.value = res as K8sVersion[]
    const recommended = (res as K8sVersion[]).find(v => v.recommended)
    if (recommended) {
      deployConfig.clusterBasicInfo.version = recommended.version
    } else {
      const firstVersion = (res as K8sVersion[])[0]
      if (firstVersion) {
        deployConfig.clusterBasicInfo.version = firstVersion.version
      }
    }
  } catch (e: any) {
    ElMessage.error('获取 K8s 版本列表失败: ' + (e.msg || e.message))
  }
}

const loadMachines = async () => {
  try {
    // 拉取所有受控机器（显式清除 status 筛选，避免沿用机器管理页的筛选导致只显示在线机器）
    await machineStore.fetchMachineList({ page: 1, pageSize: 500, status: '' })
  } catch (e: any) {
    ElMessage.error('获取机器列表失败: ' + (e.msg || e.message))
  }
}

function parseHostLines(s: string): string[] {
  return s.split(/\r?\n/).map(x => x.trim()).filter(Boolean)
}

async function validateDeployInputs(): Promise<boolean> {
  try {
    await step1FormRef.value?.validate()
    await step4FormRef.value?.validate()
  } catch {
    ElMessage.warning('请先完善基础信息与网络配置')
    openSections.value = 'basic'
    return false
  }

  const res = await checkClusterName({ clusterName: deployConfig.clusterBasicInfo.clusterName })
  if (!(res as any)?.isAvailable) {
    ElMessage.error('集群名称已存在，请换一个名称')
    openSections.value = 'basic'
    return false
  }

  if (offlineBundleMode.value) {
    const masters = parseHostLines(masterHostsText.value)
    const workers = parseHostLines(workerHostsText.value)
    deployConfig.nodeConfig.masterHosts = masters
    deployConfig.nodeConfig.workerHosts = workers
    if (masters.length === 0) {
      ElMessage.warning('请在「节点配置」填写至少一行控制平面 IP')
      openSections.value = 'nodes'
      return false
    }
    return true
  }

  if (!deployConfig.nodeConfig.executorNode) {
    ElMessage.warning('请选择执行节点（Agent 所在机器）')
    openSections.value = 'nodes'
    return false
  }
  if (deployConfig.nodeConfig.masterNodes.length === 0) {
    ElMessage.warning('请至少选择一个 K8s 控制平面节点')
    openSections.value = 'nodes'
    return false
  }
  return true
}

// ---------- 下载离线包 ----------
const handleCreateInstallRef = async () => {
  if (!(await validateDeployInputs())) {
    return
  }
  creatingInvite.value = true
  try {
    const publicApiBase = `${window.location.origin}${import.meta.env.VITE_BASE_API || '/ft-api'}`.replace(
      /\/$/,
      ''
    )
    const data = await createK8sBundleInvite(deployConfig as DeployConfig, publicApiBase)
    lastInvite.value = {
      id: data.id,
      expiresAt: data.expiresAt,
      installRef: data.installRef,
      installCommand: data.installCommand,
      bootstrapCommand: data.bootstrapCommand,
      cleanupCommand: data.cleanupCommand || `sudo ai-sre k8s cleanup '${data.installRef}'`
    }
    try {
      await navigator.clipboard.writeText(data.bootstrapCommand)
      ElMessage.success('已生成；已复制「方式 B」命令（无需 ai-sre）。已装 CLI 可选用方式 A')
    } catch {
      ElMessage.success('已生成命令，请在上方案块内复制方式 A 或 B')
    }
  } catch (e: any) {
    ElMessage.error(e?.message || '生成失败')
  } finally {
    creatingInvite.value = false
  }
}

const handleDownloadBundle = async () => {
  if (!(await validateDeployInputs())) {
    return
  }
  downloadingBundle.value = true
  try {
    await downloadOfflineBundle(deployConfig as DeployConfig)
    ElMessage.success('已开始下载 zip 安装包')
  } catch (e: any) {
    ElMessage.error(e?.message || '生成失败')
  } finally {
    downloadingBundle.value = false
  }
}

// ---------- 提交部署（在线 Agent） ----------
const submitDeploy = async () => {
  if (offlineBundleMode.value) {
    ElMessage.warning('当前为离线模式，请使用最后一步的「一键安装命令」或「下载 zip」')
    return
  }
  if (!(await validateDeployInputs())) {
    return
  }
  submitting.value = true
  try {
    const res = await submitDeployConfig(deployConfig as DeployConfig)
    k8sDeployStore.clearState()
    ElMessage.success('部署任务已创建')
    router.push({
      path: '/service/k8s-deploy/progress',
      query: { deployId: res.deployId }
    })
  } catch (e: any) {
    ElMessage.error('提交部署失败: ' + (e.msg || e.message))
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
/* ==================== 页面布局 ==================== */
.k8s-prereq-body {
  font-size: 13px;
  line-height: 1.6;
  color: #374151;
}

.k8s-prereq-lead {
  margin: 0 0 10px;
}

.k8s-prereq-body p {
  margin: 0 0 10px;
}

.k8s-prereq-ol {
  margin: 6px 0 12px 1.25rem;
  padding-left: 0.25rem;
}

.k8s-prereq-ol li {
  margin-bottom: 8px;
}

.k8s-prereq-body code {
  font-size: 12px;
  padding: 1px 5px;
  border-radius: 4px;
  background: #f3f4f6;
  color: #1f2937;
}

.k8s-prereq-muted {
  margin-top: 8px !important;
  margin-bottom: 0 !important;
  font-size: 12px;
  color: #6b7280;
}

.k8s-preflight-table {
  margin-top: 4px;
  margin-bottom: 8px;
  font-size: 12px;
}

.k8s-prereq-cmd {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  padding: 1px 5px;
  border-radius: 4px;
  background: #f3f4f6;
  color: #1f2937;
  white-space: pre-wrap;
  word-break: break-all;
}

.k8s-deploy-form {
  width: 100%;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow-x: hidden;
  background:
    radial-gradient(circle at top left, rgba(30, 64, 175, 0.08), transparent 32%),
    linear-gradient(180deg, #f8fafc 0%, #fff 36%);
}

.page-header {
  margin: 0;
}

.page-header-inner {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(320px, 420px);
  align-items: center;
  gap: 18px;
  text-align: left;
  max-width: none;
  margin: 0 auto;
  padding: 16px 18px;
  border-radius: 14px;
  background: linear-gradient(165deg, var(--el-color-primary-light-9) 0%, #fff 55%);
  border: 1px solid var(--el-border-color-lighter);
  box-shadow: 0 1px 2px rgba(30, 64, 175, 0.06);
}

.page-header-copy {
  min-width: 0;
}

.page-kicker {
  display: inline-block;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--el-color-primary);
  margin-bottom: 8px;
}

.page-title {
  color: var(--el-text-color-primary);
  margin: 0 0 8px;
  font-size: 24px;
  font-weight: 700;
  letter-spacing: -0.02em;
  line-height: 1.25;
}

.page-desc {
  color: var(--el-text-color-secondary);
  font-size: 14px;
  line-height: 1.65;
  margin: 0;
}

.page-desc strong {
  color: var(--el-text-color-regular);
  font-weight: 600;
}

.deploy-layout {
  display: block;
  width: 100%;
}

.deploy-main {
  min-width: 0;
}

.deploy-config-collapse {
  display: flex;
  flex-direction: column;
  gap: 12px;
  border: none;
}

.deploy-config-collapse :deep(.el-collapse-item) {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 14px;
  overflow: hidden;
  background: rgba(255, 255, 255, 0.94);
  box-shadow: 0 10px 28px rgba(15, 23, 42, 0.05);
}

.deploy-config-collapse :deep(.el-collapse-item__header) {
  min-height: 56px;
  height: auto;
  padding: 9px 16px;
  border-bottom: none;
  background: linear-gradient(135deg, #fff 0%, #f8fbff 100%);
}

.deploy-config-collapse :deep(.el-collapse-item__wrap) {
  border-top: 1px solid var(--el-border-color-lighter);
}

.deploy-config-collapse :deep(.el-collapse-item__content) {
  padding: 18px 20px 20px;
}

.config-item-title {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  width: 100%;
}

.config-item-icon {
  width: 38px;
  height: 38px;
  border-radius: 12px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  color: #fff;
  background: linear-gradient(135deg, var(--el-color-primary), var(--el-color-primary-light-3));
  box-shadow: 0 8px 16px rgba(30, 64, 175, 0.18);
}

.config-item-icon--precheck {
  background: linear-gradient(135deg, #0f766e, #14b8a6);
  box-shadow: 0 8px 16px rgba(15, 118, 110, 0.16);
}

.config-item-text {
  display: flex;
  flex-direction: column;
  min-width: 0;
  line-height: 1.35;
}

.config-item-name {
  color: var(--el-text-color-primary);
  font-size: 15px;
  font-weight: 700;
}

.config-item-desc {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 400;
  white-space: normal;
}

.install-precheck {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.precheck-block {
  padding: 14px 16px;
  border: 1px solid rgba(30, 64, 175, 0.08);
  border-radius: 12px;
  background: linear-gradient(180deg, rgba(248, 250, 252, 0.92), rgba(255, 255, 255, 0.96));
}

.precheck-block__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.precheck-block__head h4 {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 14px;
  font-weight: 700;
}

.precheck-block__head span {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.precheck-block__head--actions {
  align-items: center;
}

.install-ai-sre-card {
  min-width: 0;
  padding: 12px 14px;
  border: 1px solid rgba(30, 64, 175, 0.1);
  border-radius: 12px;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.82), rgba(219, 234, 254, 0.34));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.9);
}

.install-ai-sre-card__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}

.install-ai-sre-card h3 {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 15px;
  line-height: 1.25;
}

.install-ai-sre-card p {
  margin: 3px 0 0;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.4;
}

.install-command-copy {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  padding: 9px 10px;
  border: 1px solid rgba(30, 64, 175, 0.08);
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.74);
  color: var(--el-text-color-primary);
  cursor: pointer;
  transition: border-color 0.16s ease, background 0.16s ease, box-shadow 0.16s ease;
}

.install-command-copy:hover,
.install-command-copy:focus-visible {
  border-color: rgba(30, 64, 175, 0.22);
  background: #fff;
  box-shadow: 0 6px 16px rgba(30, 64, 175, 0.08);
  outline: none;
}

.install-command-copy code {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  font-size: 12px;
  line-height: 1.45;
  color: #1e3a8a;
  background: transparent;
}

.install-command-copy__hint {
  color: var(--el-color-primary);
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}

.node-mode-form-item {
  margin-bottom: 16px;
}

.node-mode-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px 12px;
}

.mode-hint-inline {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.5;
  max-width: min(560px, 100%);
}

.mode-hint {
  margin-left: 12px;
  color: #6b7280;
  font-size: 13px;
}
.mode-hint code {
  font-size: 12px;
  padding: 2px 6px;
  background: #f3f4f6;
  border-radius: 4px;
}

.btn-icon-left {
  margin-right: 4px;
  vertical-align: -0.12em;
}

.pre-cleanup-hint {
  margin: 8px 0 0;
  max-width: 720px;
  font-size: 13px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}

.pre-cleanup-hint code {
  font-size: 12px;
  padding: 1px 5px;
  border-radius: 4px;
  background: var(--el-fill-color-light);
}

.deploy-steps {
  margin-bottom: 4px;
  padding: 2px 0 8px;
}

.deploy-steps :deep(.el-step__title) {
  font-size: 12px;
  line-height: 1.35;
}

/* 确认页：精简核心信息 */
.step-section--confirm {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding-bottom: 0;
}

.confirm-lead {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--el-text-color-regular);
}

.confirm-lead--online {
  color: var(--el-text-color-secondary);
}

.confirm-cmd-card {
  padding: 16px 18px;
  border-radius: 10px;
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
}

.confirm-cmd-card--cluster {
  border-color: var(--el-color-success-light-5);
  background: linear-gradient(180deg, var(--el-color-success-light-9) 0%, var(--el-bg-color) 55%);
}

.confirm-cmd-card__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 8px;
  flex-wrap: wrap;
}

.confirm-cmd-card__title {
  font-size: 14px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}

.confirm-cmd-card__hint {
  margin: 0 0 10px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}

.confirm-cmd-card__hint code {
  font-size: 11px;
}

.confirm-cmd-card__meta {
  margin: 0 0 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  word-break: break-all;
}

.offline-install-panel__placeholder {
  margin: 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.65;
}

.install-command-textarea :deep(textarea) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  font-size: 12px;
  line-height: 1.5;
}

.confirm-value--wrap {
  line-height: 1.5;
  word-break: break-all;
}

.confirm-aux-collapse {
  border: 1px dashed var(--el-border-color);
  border-radius: 8px;
  padding: 0 12px;
  background: var(--el-fill-color-lighter);
}

.confirm-aux-collapse :deep(.el-collapse-item__header) {
  font-size: 13px;
  font-weight: 600;
}

.requirement-pre--compact {
  margin-top: 10px;
  max-height: 220px;
}

.requirement-pre {
  margin: 0;
  padding: 12px 14px;
  max-height: 320px;
  overflow: auto;
  font-family: ui-monospace, 'SF Mono', Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
}

.confirm-summary-desc {
  margin-top: 4px;
}

.confirm-summary-desc :deep(.el-descriptions__title) {
  font-size: 14px;
  font-weight: 700;
  margin-bottom: 10px;
}

.confirm-optional-collapse {
  margin-top: 12px;
  border: none;
}

.confirm-optional-collapse :deep(.el-collapse-item__wrap) {
  border: none;
}

.confirm-optional-copy {
  margin-top: 6px;
}

.form-hint--compact {
  margin: -6px 0 12px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

/* ==================== 步骤卡片（统一风格） ==================== */
.step-card {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
  overflow: hidden;
}

.step-card-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px 28px;
  border-bottom: 1px solid #f0f0f0;
  background: linear-gradient(135deg, #f8fbff 0%, #f0f7ff 100%);
}

.step-card-indicator {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  background: linear-gradient(135deg, #1890ff, #096dd9);
  color: #fff;
  font-size: 18px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.step-card-meta {
  flex: 1;
}

.step-card-title {
  margin: 0 0 2px 0;
  font-size: 17px;
  font-weight: 600;
  color: #1f2937;
}

.step-card-desc {
  margin: 0;
  font-size: 13px;
  color: #9ca3af;
}

.step-card-body {
  padding: 28px;
}

/* ==================== 步骤内容区域（fadeIn 动画） ==================== */
.step-section {
  animation: fadeIn 0.25s ease-out;
}

.step-section--basic {
  max-width: 980px;
}

.step-section--basic :deep(.el-row) {
  row-gap: 2px;
}

.step-section--basic :deep(.el-form-item) {
  margin-bottom: 14px;
}

.step-section--basic :deep(.el-input),
.step-section--basic :deep(.el-select),
.step-section--basic :deep(.el-input-number) {
  max-width: 460px;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}

.step-alert {
  margin-bottom: 20px;
}

.executor-select-item {
  margin-bottom: 20px;
}

/* ==================== 标签/污点 网格 ==================== */
.label-taint-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

/* ==================== 额外参数网格 ==================== */
.extra-args-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}

/* ==================== 子卡片（统一样式） ==================== */
.sub-card {
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  overflow: hidden;
  transition: box-shadow 0.2s;
}

.sub-card:hover {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
}

.sub-card-header {
  font-weight: 600;
  font-size: 14px;
  color: #1e40af;
  display: flex;
  align-items: center;
}

.sub-card-header::before {
  content: '';
  width: 4px;
  height: 14px;
  background: #1e40af;
  border-radius: 2px;
  margin-right: 8px;
  flex-shrink: 0;
}

/* ==================== 底部操作（sticky，便于长确认页仍能看到主操作） ==================== */
.step-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  padding: 12px 14px;
  z-index: 20;
  background: rgba(255, 255, 255, 0.92);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 14px;
  box-shadow: 0 8px 22px rgba(15, 23, 42, 0.05);
}
.action-spacer {
  flex: 1;
}

.primary-finish-btn {
  min-width: 220px;
  font-weight: 600;
  padding-left: 22px;
  padding-right: 22px;
}

.primary-finish-btn-icon {
  margin-right: 6px;
  vertical-align: -0.15em;
}

.step-next-btn {
  min-width: 108px;
}

/* ==================== 响应式 ==================== */
@media (max-width: 1280px) {
  .page-header-inner {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 1200px) {
  .extra-args-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 900px) {
  .k8s-deploy-form {
    padding: 0;
  }

  .label-taint-grid {
    grid-template-columns: 1fr;
  }

  .step-actions {
    justify-content: stretch;
  }

  .action-spacer {
    display: none;
  }

  .primary-finish-btn {
    flex: 1 1 100%;
  }
}

.component-catalog-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
  margin: 0 0 10px;
}
.k8s-component-catalog-table {
  margin-bottom: 12px;
}

.k8s-preflight-table,
.k8s-component-catalog-table {
  width: 100%;
}

.k8s-preflight-table :deep(.cell),
.k8s-component-catalog-table :deep(.cell) {
  white-space: normal;
  word-break: break-word;
}
</style>
