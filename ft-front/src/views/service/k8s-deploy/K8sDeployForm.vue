<template>
  <div class="k8s-deploy-form page-shell page-shell--wizard">
    <header class="page-header">
      <div class="page-header-inner">
        <span class="page-kicker">Kubernetes</span>
        <h2 class="page-title">部署 Kubernetes 集群</h2>
        <p class="page-desc">
          离线：生成<strong>一键命令</strong>或 zip，在 Ubuntu 24.04 控制机执行；在线：由 Agent 执行 Ansible。
        </p>
      </div>
    </header>

    <!-- 步骤条（简洁横条，减少视觉噪音） -->
    <el-steps
      :active="activeStep"
      class="deploy-steps"
      align-center
      finish-status="success"
      simple
    >
      <el-step v-for="(s, i) in stepsMeta" :key="i" :title="s.title" :icon="s.icon" />
    </el-steps>

    <!-- ==================== 步骤内容 ==================== -->
    <div class="step-card">
      <!-- 步骤标题区域（固定风格） -->
      <div class="step-card-header">
        <div class="step-card-indicator">{{ activeStep + 1 }}</div>
        <div class="step-card-meta">
          <h3 class="step-card-title">{{ stepsMeta[activeStep].title }}</h3>
          <p class="step-card-desc">{{ stepsMeta[activeStep].desc }}</p>
        </div>
      </div>

      <div class="step-card-body">
        <!-- ========== 步骤 1: 基础集群信息 ========== -->
        <div v-show="activeStep === 0" class="step-section">
          <el-collapse v-model="stepAuxOpen" class="step-aux-collapse">
            <el-collapse-item name="ssh" title="离线安装必读：控制机须能免密 SSH 各节点 root（展开查看操作）">
              <div class="k8s-prereq-body">
                <p class="k8s-prereq-lead">
                  <code>install.sh</code> 在<strong>你执行命令的 Ubuntu 机</strong>上跑 Ansible，并以 <strong>root</strong> 连所有节点 IP。若报
                  <code>Permission denied</code>，先完成下列步骤。
                </p>
                <ol class="k8s-prereq-ol">
                  <li>
                    控制机：<code>ssh-keygen -t ed25519 -N "" -f ~/.ssh/id_ed25519</code>（若已有密钥可跳过）
                  </li>
                  <li>对每个节点：<code>ssh-copy-id -i ~/.ssh/id_ed25519.pub root@&lt;IP&gt;</code></li>
                  <li>验证：<code>ssh root@&lt;IP&gt;</code> 无密码</li>
                </ol>
                <p class="k8s-prereq-muted">
                  脚本会另建 <code>ansible</code> 用户与密钥；清单为 <code>ansible_user=root</code>，不支持交互式密码。
                </p>
              </div>
            </el-collapse-item>
            <el-collapse-item
              :title="`部署记录（${k8sDeployStore.deployRecords.length} 条）`"
              name="records"
            >
              <div class="deploy-records-inner">
                <div class="deploy-records-header">
                  <el-button type="primary" link size="small" :loading="k8sDeployStore.loadingRecords" @click="k8sDeployStore.fetchDeployRecords">
                    刷新
                  </el-button>
                </div>
                <el-table
                  :data="k8sDeployStore.deployRecords"
                  stripe
                  size="small"
                  max-height="200"
                  class="deploy-records-table"
                >
                  <el-table-column prop="clusterName" label="集群" min-width="100" show-overflow-tooltip />
                  <el-table-column prop="status" label="状态" width="80" align="center">
                    <template #default="{ row }">
                      <el-tag
                        :type="row.status === 'success' ? 'success' : row.status === 'failed' ? 'danger' : row.status === 'running' || row.status === 'pending' ? 'warning' : 'info'"
                        size="small"
                      >
                        {{ statusLabel(row.status) }}
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column prop="progress" label="进度" width="72" align="center">
                    <template #default="{ row }">{{ row.progress }}%</template>
                  </el-table-column>
                  <el-table-column prop="createdAt" label="时间" width="140">
                    <template #default="{ row }">{{ formatRecordTime(row.createdAt) }}</template>
                  </el-table-column>
                  <el-table-column label="" width="64" align="center" fixed="right">
                    <template #default="{ row }">
                      <el-button type="primary" link size="small" @click="goToProgress(row.deployId)">查看</el-button>
                    </template>
                  </el-table-column>
                </el-table>
                <div v-if="k8sDeployStore.deployRecords.length === 0 && !k8sDeployStore.loadingRecords" class="deploy-records-empty">
                  暂无记录
                </div>
              </div>
            </el-collapse-item>
          </el-collapse>

          <!-- 正在部署：通过 WS 实时展示当前进行中的部署 -->
          <div v-if="runningDeploy" class="deploy-status-block">
            <div class="deploy-status-header">
              <span class="deploy-status-title">正在部署</span>
              <el-tag type="warning" size="small">进行中</el-tag>
              <el-button type="primary" link size="small" @click="goToProgress(runningDeploy.deployId)">
                查看详情
              </el-button>
            </div>
            <div class="deploy-status-body">
              <div class="deploy-status-meta">
                <span>{{ runningDeploy.clusterName }}</span>
                <span class="deploy-status-step">{{ runningDeploy.currentStep || '准备中...' }}</span>
              </div>
              <el-progress
                :percentage="runningDeploy.progress"
                :stroke-width="10"
                status=""
              />
            </div>
          </div>

          <!-- 部署记录：与机器管理同源，展示历史与状态 -->
          <el-divider content-position="left">基础集群信息</el-divider>
          <el-form
            ref="step1FormRef"
            :model="deployConfig.clusterBasicInfo"
            :rules="step1Rules"
            label-position="top"
          >
            <el-row :gutter="20">
              <el-col :span="8">
                <el-form-item label="集群名称" prop="clusterName">
                  <el-input
                    v-model="deployConfig.clusterBasicInfo.clusterName"
                    placeholder="请输入集群名称"
                    clearable
                  />
                </el-form-item>
              </el-col>
              <el-col :span="8">
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
              <el-col :span="8">
                <el-form-item label="部署模式" prop="deployMode">
                  <el-radio-group v-model="deployConfig.clusterBasicInfo.deployMode">
                    <el-radio value="single">单节点</el-radio>
                    <el-radio value="cluster">多节点</el-radio>
                  </el-radio-group>
                </el-form-item>
              </el-col>
            </el-row>

            <el-row :gutter="20">
              <el-col :span="8">
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
              <el-col :span="8">
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

            <el-row :gutter="20">
              <el-col :span="8">
                <el-form-item label="内网制品地址" prop="downloadDomain">
                  <el-input
                    v-model="deployConfig.clusterBasicInfo.downloadDomain"
                    placeholder="留空则用 inventory 默认 download_domain"
                    clearable
                  />
                </el-form-item>
              </el-col>
              <el-col :span="8">
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

        <!-- ========== 步骤 2: 节点配置 ========== -->
        <div v-show="activeStep === 1" class="step-section" v-loading="activeStep === 1 && machineStore.loading">
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
                placeholder="每行一个 Worker IP"
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
              <LabelGroup v-model="deployConfig.nodeConfig.masterLabels" />
            </el-card>
            <el-card class="sub-card" shadow="hover">
              <template #header>
                <div class="sub-card-header"><span>主节点污点</span></div>
              </template>
              <TaintGroup v-model="deployConfig.nodeConfig.masterTaints" />
            </el-card>
            <el-card class="sub-card" shadow="hover">
              <template #header>
                <div class="sub-card-header"><span>工作节点标签</span></div>
              </template>
              <LabelGroup v-model="deployConfig.nodeConfig.workerLabels" />
            </el-card>
            <el-card class="sub-card" shadow="hover">
              <template #header>
                <div class="sub-card-header"><span>工作节点污点</span></div>
              </template>
              <TaintGroup v-model="deployConfig.nodeConfig.workerTaints" />
            </el-card>
          </div>
        </div>

        <!-- ========== 步骤 3: 核心组件配置 ========== -->
        <div v-show="activeStep === 2" class="step-section">
          <el-form
            ref="step3FormRef"
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

        <!-- ========== 步骤 4: 网络配置 ========== -->
        <div v-show="activeStep === 3" class="step-section">
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
                    <el-option label="Cilium" value="cilium" />
                    <el-option label="Weave" value="weave" />
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
                    <el-switch v-model="deployConfig.networkConfig.calicoConfig.vxlanMode" />
                  </el-form-item>
                </el-col>
                <el-col :span="8">
                  <el-form-item label="MTU 值">
                    <el-input-number
                      v-model="deployConfig.networkConfig.calicoConfig.mtu"
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
                    <el-select v-model="deployConfig.networkConfig.flannelConfig.backend" style="width: 100%">
                      <el-option label="VXLAN" value="vxlan" />
                      <el-option label="Host-GW" value="host-gw" />
                      <el-option label="UDP" value="udp" />
                    </el-select>
                  </el-form-item>
                </el-col>
              </el-row>
            </template>
          </el-form>
        </div>

        <!-- ========== 步骤 5: 存储配置 ========== -->
        <div v-show="activeStep === 4" class="step-section">
          <el-form
            ref="step5FormRef"
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
                      v-model="deployConfig.storageConfig.localPathConfig.path"
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
                    <el-input v-model="deployConfig.storageConfig.nfsConfig.server" placeholder="NFS 服务器 IP" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="NFS 共享路径">
                    <el-input v-model="deployConfig.storageConfig.nfsConfig.path" placeholder="/data/nfs" />
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
                    <el-input v-model="deployConfig.storageConfig.csiConfig.driver" placeholder="csi.aliyun.com" />
                  </el-form-item>
                </el-col>
                <el-col :span="8">
                  <el-form-item label="控制器数量">
                    <el-input-number
                      v-model="deployConfig.storageConfig.csiConfig.controllerCount"
                      :min="1"
                      :max="5"
                    />
                  </el-form-item>
                </el-col>
              </el-row>
            </template>
          </el-form>
        </div>

        <!-- ========== 步骤 6: 高级配置 ========== -->
        <div v-show="activeStep === 5" class="step-section">
          <el-form
            ref="step6FormRef"
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
                <KeyValueGroup v-model="deployConfig.advancedConfig.extraKubeletArgs" />
              </el-card>
              <el-card class="sub-card" shadow="hover">
                <template #header>
                  <div class="sub-card-header"><span>KubeProxy 额外参数</span></div>
                </template>
                <KeyValueGroup v-model="deployConfig.advancedConfig.extraKubeProxyArgs" />
              </el-card>
              <el-card class="sub-card" shadow="hover">
                <template #header>
                  <div class="sub-card-header"><span>API Server 额外参数</span></div>
                </template>
                <KeyValueGroup v-model="deployConfig.advancedConfig.extraAPIServerArgs" />
              </el-card>
            </div>
          </el-form>
        </div>

        <!-- ========== 步骤 7: 部署确认 ========== -->
        <div v-show="activeStep === 6" class="step-section step-section--confirm">
          <div
            class="confirm-hero"
            :class="offlineBundleMode ? 'confirm-hero--offline' : 'confirm-hero--online'"
          >
            <div class="confirm-hero-icon" aria-hidden="true">
              <el-icon v-if="offlineBundleMode" :size="28"><FolderOpened /></el-icon>
              <el-icon v-else :size="28"><Promotion /></el-icon>
            </div>
            <div class="confirm-hero-text">
              <h4 class="confirm-hero-title">
                {{ offlineBundleMode ? '离线交付' : '在线部署' }}
              </h4>
              <p class="confirm-hero-desc">
                <template v-if="offlineBundleMode">
                  核对摘要后，可<strong>生成一键安装命令</strong>（推荐）或<strong>下载 zip</strong>。命令在控制机执行，须已安装
                  <code>ai-sre</code>。
                </template>
                <template v-else>
                  核对无误后点击底部<strong>开始在线部署</strong>，由 Agent 执行 Ansible。
                </template>
              </p>
            </div>
          </div>

          <!-- 离线：安装命令展示区（生成后常驻，可复制） -->
          <div v-if="offlineBundleMode" class="offline-install-panel">
            <div class="offline-install-panel__head">
              <h4 class="offline-install-panel__title">一键安装命令</h4>
              <el-button
                v-if="lastInvite"
                type="primary"
                size="small"
                @click="copyInstallCommand"
              >
                <el-icon class="btn-icon-left"><DocumentCopy /></el-icon>
                复制命令
              </el-button>
            </div>
            <p v-if="!lastInvite" class="offline-install-panel__placeholder">
              点击下方<strong>「生成一键安装命令」</strong>后，将在此显示完整命令与资源 ID（无需仅依赖弹窗或下载 zip）。
            </p>
            <template v-else>
              <el-descriptions :column="2" size="small" border class="invite-meta-desc">
                <el-descriptions-item label="资源 ID">
                  <span class="mono-ellipsis" :title="lastInvite.id">{{ lastInvite.id }}</span>
                </el-descriptions-item>
                <el-descriptions-item label="有效期至">
                  {{ formatInviteExpiry(lastInvite.expiresAt) }}
                </el-descriptions-item>
              </el-descriptions>
              <el-input
                type="textarea"
                :rows="3"
                readonly
                :model-value="lastInvite.installCommand"
                class="install-command-textarea"
              />
              <p class="offline-install-panel__warn">
                命令含下载密钥，请勿泄露；过期后请在本页重新生成。
              </p>
            </template>
          </div>

          <!-- 根据表单自动生成的需求说明（评审 / 工单 / 交接） -->
          <div class="requirement-doc-panel">
            <div class="requirement-doc-panel__head">
              <h4 class="requirement-doc-panel__title">部署需求说明</h4>
              <el-button type="primary" size="small" @click="copyDeployRequirement">
                <el-icon class="btn-icon-left"><DocumentCopy /></el-icon>
                复制全文
              </el-button>
            </div>
            <p class="requirement-doc-panel__hint">
              由当前向导配置自动生成，随表单变化更新；可复制到邮件、工单或文档。<strong>不含</strong>一键命令中的下载密钥。
            </p>
            <el-input
              type="textarea"
              :rows="16"
              readonly
              :model-value="deployRequirementText"
              class="requirement-textarea"
            />
          </div>

          <div class="confirm-grid">
            <!-- 集群基础 -->
            <div class="confirm-block">
              <h4 class="confirm-block-title">集群基本信息</h4>
              <div class="confirm-row">
                <span class="confirm-label">集群名称</span>
                <span class="confirm-value">{{ deployConfig.clusterBasicInfo.clusterName }}</span>
              </div>
              <div class="confirm-row">
                <span class="confirm-label">K8s 版本</span>
                <span class="confirm-value">{{ deployConfig.clusterBasicInfo.version }}</span>
              </div>
              <div class="confirm-row">
                <span class="confirm-label">部署模式</span>
                <span class="confirm-value">{{ deployConfig.clusterBasicInfo.deployMode === 'cluster' ? '多节点' : '单节点' }}</span>
              </div>
              <div class="confirm-row">
                <span class="confirm-label">CPU 架构</span>
                <span class="confirm-value">{{ deployConfig.clusterBasicInfo.cpuArch || 'arm64' }}</span>
              </div>
              <div class="confirm-row">
                <span class="confirm-label">镜像源</span>
                <span class="confirm-value">{{ imageSourceText }}</span>
              </div>
              <div
                v-if="deployConfig.clusterBasicInfo.downloadDomain?.trim() || deployConfig.clusterBasicInfo.downloadProtocol?.trim()"
                class="confirm-row"
              >
                <span class="confirm-label">制品覆盖</span>
                <span class="confirm-value">
                  {{ deployConfig.clusterBasicInfo.downloadProtocol?.trim() || '（默认）' }}
                  {{ deployConfig.clusterBasicInfo.downloadDomain?.trim() || '—' }}
                </span>
              </div>
            </div>

            <!-- 节点 -->
            <div class="confirm-block">
              <h4 class="confirm-block-title">节点配置</h4>
              <div class="confirm-row">
                <span class="confirm-label">{{ offlineBundleMode ? '控制平面 IP' : '执行节点' }}</span>
                <span class="confirm-value confirm-value--wrap">{{ executorConfirmText }}</span>
              </div>
              <template v-if="!offlineBundleMode">
                <div class="confirm-row">
                  <span class="confirm-label">控制平面</span>
                  <span class="confirm-value">{{ deployConfig.nodeConfig.masterNodes.length }} 台</span>
                </div>
                <div class="confirm-row">
                  <span class="confirm-label">工作节点</span>
                  <span class="confirm-value">{{ deployConfig.nodeConfig.workerNodes.length }} 台</span>
                </div>
              </template>
              <template v-else>
                <div class="confirm-row">
                  <span class="confirm-label">工作节点 IP</span>
                  <span class="confirm-value confirm-value--wrap">{{ confirmWorkerPreview }}</span>
                </div>
              </template>
            </div>

            <!-- 网络 -->
            <div class="confirm-block">
              <h4 class="confirm-block-title">网络配置</h4>
              <div class="confirm-row">
                <span class="confirm-label">网络插件</span>
                <span class="confirm-value">{{ deployConfig.networkConfig.networkPlugin }}</span>
              </div>
              <div class="confirm-row">
                <span class="confirm-label">Pod CIDR</span>
                <span class="confirm-value">{{ deployConfig.networkConfig.podCIDR }}</span>
              </div>
              <div class="confirm-row">
                <span class="confirm-label">Service CIDR</span>
                <span class="confirm-value">{{ deployConfig.networkConfig.serviceCIDR }}</span>
              </div>
            </div>

            <!-- 存储 -->
            <div class="confirm-block">
              <h4 class="confirm-block-title">存储配置</h4>
              <div class="confirm-row">
                <span class="confirm-label">存储供应器</span>
                <span class="confirm-value">{{ deployConfig.storageConfig.storageProvisioner }}</span>
              </div>
              <div class="confirm-row">
                <span class="confirm-label">默认存储类</span>
                <span class="confirm-value">{{ deployConfig.storageConfig.defaultStorageClass ? '是' : '否' }}</span>
              </div>
            </div>

            <!-- 高级 -->
            <div class="confirm-block confirm-block-full">
              <h4 class="confirm-block-title">高级配置</h4>
              <div class="confirm-row">
                <span class="confirm-label">部署前清理</span>
                <span class="confirm-value">{{ deployConfig.advancedConfig.preDeployCleanup ? '是（Step 0 非交互）' : '否' }}</span>
              </div>
              <div class="confirm-row">
                <span class="confirm-label">启用组件</span>
                <span class="confirm-value">{{ enabledComponentsText || '无' }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ==================== 底部操作栏（最后一步仅展示与当前模式匹配的主操作） ==================== -->
    <div class="step-actions">
      <el-button
        v-if="activeStep > 0"
        @click="prevStep"
        :disabled="submitting || downloadingBundle || creatingInvite"
      >
        上一步
      </el-button>
      <div class="action-spacer" />
      <el-button
        v-if="activeStep < 6"
        type="primary"
        class="step-next-btn"
        @click="nextStep"
        :loading="validating"
      >
        下一步
      </el-button>
      <template v-if="activeStep === 6">
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
      </template>
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
  FolderOpened,
  Promotion,
  DocumentCopy
} from '@element-plus/icons-vue'
import NodeSelect from '@/components/k8s/NodeSelect.vue'
import LabelGroup from '@/components/k8s/LabelGroup.vue'
import TaintGroup from '@/components/k8s/TaintGroup.vue'
import KeyValueGroup from '@/components/k8s/KeyValueGroup.vue'
import {
  getK8sVersions,
  checkClusterName,
  submitDeployConfig,
  downloadOfflineBundle,
  createK8sBundleInvite
} from '../../../api/k8s-deploy'
import type {
  DeployConfig,
  K8sMachineInfo,
  K8sVersion
} from '../../../types/k8s-deploy'
import { useK8sDeployStore } from '../../../stores/k8s-deploy'
import { useMachineStore } from '../../../stores/machine'

const router = useRouter()
const k8sDeployStore = useK8sDeployStore()
const machineStore = useMachineStore()

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
    desc: '核对摘要、复制部署需求说明；离线可生成命令或 zip',
    icon: markRaw(CircleCheck)
  }
]

// ---------- 表单 Refs ----------
const step1FormRef = ref<FormInstance>()
const step3FormRef = ref<FormInstance>()
const step4FormRef = ref<FormInstance>()
const step5FormRef = ref<FormInstance>()
const step6FormRef = ref<FormInstance>()

// ---------- 状态 ----------
const activeStep = ref(0)
const validating = ref(false)
const submitting = ref(false)
const downloadingBundle = ref(false)
const creatingInvite = ref(false)
/** 步骤 1：折叠区默认收起，减少首屏噪音 */
const stepAuxOpen = ref<string[]>([])
/** 离线一键安装接口返回，用于最后一步展示 */
const lastInvite = ref<{
  id: string
  expiresAt: string
  installRef: string
  installCommand: string
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

const confirmWorkerPreview = computed(() => {
  const fromCfg = deployConfig.nodeConfig.workerHosts?.length
    ? deployConfig.nodeConfig.workerHosts
    : workerHostsText.value
        .split(/\r?\n/)
        .map((x) => x.trim())
        .filter(Boolean)
  return fromCfg.length ? fromCfg.join('、') : '无'
})

/** 部署确认页：可复制的需求说明全文（不含 installRef 密钥） */
const deployRequirementText = computed(() => {
  const L: string[] = []
  const now = new Date().toLocaleString('zh-CN', { hour12: false })
  const name = deployConfig.clusterBasicInfo.clusterName?.trim() || '（未填写）'
  const ver = deployConfig.clusterBasicInfo.version || '—'
  const arch = deployConfig.clusterBasicInfo.cpuArch || 'amd64'
  const mode = deployConfig.clusterBasicInfo.deployMode === 'cluster' ? '多节点' : '单节点'

  L.push('Kubernetes 集群部署需求说明（OpsFleet 控制台生成）')
  L.push(`生成时间：${now}`)
  L.push('')
  L.push('【1. 目标】')
  L.push(`部署集群「${name}」，Kubernetes ${ver}，节点 CPU 架构 ${arch}，规划模式 ${mode}。`)
  L.push('')
  L.push('【2. 交付方式与前置条件】')
  if (offlineBundleMode.value) {
    L.push('- 交付形态：离线（一键安装命令 ai-sre，或 zip + install.sh）。')
    L.push('- 控制机：建议 Ubuntu 24.04 LTS；一键命令方式须已安装 ai-sre。')
    L.push('- 连通性：控制机须以 root 免密 SSH 登录所有节点（见步骤 1 折叠说明）。')
    L.push('- 制品：镜像源为「' + imageSourceText.value + '」；若填写内网制品地址则覆盖 inventory 默认 download_domain。')
  } else {
    L.push('- 交付形态：在线（由已注册 Agent 的执行节点执行 Ansible）。')
    L.push('- 执行机与集群节点须网络互通，SSH 可达。')
    L.push('- 镜像源：「' + imageSourceText.value + '」。')
  }
  L.push('')
  L.push('【3. 节点清单】')
  if (offlineBundleMode.value) {
    const masters = deployConfig.nodeConfig.masterHosts?.length
      ? deployConfig.nodeConfig.masterHosts
      : parseHostLines(masterHostsText.value)
    const workers = deployConfig.nodeConfig.workerHosts?.length
      ? deployConfig.nodeConfig.workerHosts
      : parseHostLines(workerHostsText.value)
    L.push('- 控制平面 IP：' + (masters.length ? masters.join('、') : '（请在「节点配置」填写）'))
    L.push('- 工作节点 IP：' + (workers.length ? workers.join('、') : '无'))
  } else {
    L.push('- 执行节点：' + executorConfirmText.value)
    L.push(`- 控制平面：已选 ${deployConfig.nodeConfig.masterNodes.length} 台（机器以控制台为准）。`)
    L.push(`- 工作节点：已选 ${deployConfig.nodeConfig.workerNodes.length} 台。`)
  }
  L.push('')
  L.push('【4. 网络与存储】')
  L.push(`- 网络插件：${deployConfig.networkConfig.networkPlugin}；Pod CIDR ${deployConfig.networkConfig.podCIDR}；Service CIDR ${deployConfig.networkConfig.serviceCIDR}；DNS ${deployConfig.networkConfig.dnsServiceIP}。`)
  L.push(
    `- 存储：供应器 ${deployConfig.storageConfig.storageProvisioner}；默认 StorageClass ${deployConfig.storageConfig.defaultStorageClass ? '开启' : '关闭'}。`
  )
  L.push('')
  L.push('【5. 其他】')
  L.push(`- kube-proxy：${deployConfig.coreComponentsConfig.kubeProxyMode}；RBAC ${deployConfig.coreComponentsConfig.enableRBAC ? '开启' : '关闭'}。`)
  L.push(`- 部署前环境清理（Step 0）：${deployConfig.advancedConfig.preDeployCleanup ? '开启' : '关闭'}。`)
  L.push(`- 可选组件：${enabledComponentsText.value || '无'}。`)
  if (offlineBundleMode.value && lastInvite.value) {
    L.push('')
    L.push('【6. 已登记的离线安装资源（仅元数据）】')
    L.push(`- 资源 ID：${lastInvite.value.id}`)
    L.push(`- 有效期至：${formatInviteExpiry(lastInvite.value.expiresAt)}`)
    L.push('- 完整安装命令请在本页「一键安装命令」区查看；勿将命令粘贴进不受控渠道。')
  }
  L.push('')
  L.push('—— 以上由向导根据当前表单生成，提交部署或下载前请业务方与执行方共同确认。')
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

// ---------- 部署记录与正在部署 ----------
const runningDeploy = computed(() => k8sDeployStore.getRunningDeploy())

function statusLabel(s: string): string {
  const m: Record<string, string> = {
    pending: '待执行',
    running: '进行中',
    success: '成功',
    failed: '失败',
    cancelled: '已取消'
  }
  return m[s] ?? s
}

function formatRecordTime(iso: string): string {
  if (!iso) return '--'
  try {
    return new Date(iso).toLocaleString('zh-CN')
  } catch {
    return iso
  }
}

function goToProgress(deployId: string) {
  router.push({ path: '/service/k8s-deploy/progress', query: { deployId } })
}

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
  navigator.clipboard.writeText(cmd).then(
    () => ElMessage.success('已复制安装命令'),
    () => ElMessage.error('复制失败，请手动选择文本复制')
  )
}

function copyDeployRequirement() {
  const text = deployRequirementText.value
  if (!text?.trim()) return
  navigator.clipboard.writeText(text).then(
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
  loadMachines()
  k8sDeployStore.fetchDeployRecords()
})

// 切换到节点配置步骤时刷新机器列表，确保与机器管理页状态一致
watch(activeStep, (step) => {
  if (step === 1) loadMachines()
})

watch(offlineBundleMode, () => {
  lastInvite.value = null
})

const loadK8sVersions = async () => {
  try {
    const res = await getK8sVersions()
    k8sVersions.value = res as K8sVersion[]
    const recommended = (res as K8sVersion[]).find(v => v.recommended)
    if (recommended) {
      deployConfig.clusterBasicInfo.version = recommended.version
    } else if ((res as K8sVersion[]).length > 0) {
      deployConfig.clusterBasicInfo.version = (res as K8sVersion[])[0].version
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

// ---------- 步骤切换 ----------
const prevStep = () => { activeStep.value-- }

const nextStep = async () => {
  validating.value = true
  try {
    // 步骤 1 校验
    if (activeStep.value === 0) {
      await step1FormRef.value?.validate()
      // 校验集群名称唯一性
      const res = await checkClusterName({
        clusterName: deployConfig.clusterBasicInfo.clusterName
      })
      if (!(res as any)?.isAvailable) {
        ElMessage.error('集群名称已存在，请换一个名称')
        return
      }
    }
    // 步骤 2 校验
    else if (activeStep.value === 1) {
      if (offlineBundleMode.value) {
        const masters = parseHostLines(masterHostsText.value)
        const workers = parseHostLines(workerHostsText.value)
        deployConfig.nodeConfig.masterHosts = masters
        deployConfig.nodeConfig.workerHosts = workers
        if (masters.length === 0) {
          ElMessage.warning('请至少填写一行控制平面节点 IP')
          return
        }
      } else {
        if (!deployConfig.nodeConfig.executorNode) {
          ElMessage.warning('请选择执行节点（Agent 所在机器）')
          return
        }
        if (deployConfig.nodeConfig.masterNodes.length === 0) {
          ElMessage.warning('请至少选择一个 K8s 控制平面节点')
          return
        }
      }
    }
    // 步骤 4 校验
    else if (activeStep.value === 3) {
      await step4FormRef.value?.validate()
    }
    activeStep.value++
  } catch {
    // validate() 抛错表示校验失败，不切步骤
  } finally {
    validating.value = false
  }
}

function parseHostLines(s: string): string[] {
  return s.split(/\r?\n/).map(x => x.trim()).filter(Boolean)
}

// ---------- 下载离线包 ----------
const handleCreateInstallRef = async () => {
  if (!deployConfig.clusterBasicInfo.clusterName?.trim()) {
    ElMessage.warning('请填写集群名称')
    return
  }
  if (!deployConfig.clusterBasicInfo.version) {
    ElMessage.warning('请选择 K8s 版本')
    return
  }
  deployConfig.nodeConfig.masterHosts = parseHostLines(masterHostsText.value)
  deployConfig.nodeConfig.workerHosts = parseHostLines(workerHostsText.value)
  if (deployConfig.nodeConfig.masterHosts.length === 0) {
    ElMessage.warning('请在「节点配置」填写至少一行控制平面 IP')
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
      installCommand: data.installCommand
    }
    try {
      await navigator.clipboard.writeText(data.installCommand)
      ElMessage.success('已生成并复制安装命令，可在上方卡片再次查看或复制')
    } catch {
      ElMessage.success('已生成安装命令，请在上方案块内复制')
    }
  } catch (e: any) {
    ElMessage.error(e?.message || '生成失败')
  } finally {
    creatingInvite.value = false
  }
}

const handleDownloadBundle = async () => {
  if (!deployConfig.clusterBasicInfo.clusterName?.trim()) {
    ElMessage.warning('请填写集群名称')
    return
  }
  if (!deployConfig.clusterBasicInfo.version) {
    ElMessage.warning('请选择 K8s 版本')
    return
  }
  deployConfig.nodeConfig.masterHosts = parseHostLines(masterHostsText.value)
  deployConfig.nodeConfig.workerHosts = parseHostLines(workerHostsText.value)
  if (deployConfig.nodeConfig.masterHosts.length === 0) {
    ElMessage.warning('请在「节点配置」填写至少一行控制平面 IP')
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
.step-aux-collapse {
  margin-bottom: 16px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  overflow: hidden;
  --el-collapse-header-height: 44px;
}

.step-aux-collapse :deep(.el-collapse-item__header) {
  font-weight: 600;
  font-size: 13px;
  padding: 0 14px;
  background: var(--el-fill-color-light);
}

.step-aux-collapse :deep(.el-collapse-item__wrap) {
  border-top: 1px solid var(--el-border-color-lighter);
}

.step-aux-collapse :deep(.el-collapse-item__content) {
  padding: 12px 14px 14px;
}

.deploy-records-inner .deploy-records-header {
  justify-content: flex-end;
  margin-bottom: 8px;
}

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

.k8s-deploy-form {
  width: 100%;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.page-header {
  margin: 0;
}

.page-header-inner {
  text-align: center;
  max-width: 640px;
  margin: 0 auto;
  padding: 20px 20px 18px;
  border-radius: 14px;
  background: linear-gradient(165deg, var(--el-color-primary-light-9) 0%, #fff 55%);
  border: 1px solid var(--el-border-color-lighter);
  box-shadow: 0 1px 2px rgba(30, 64, 175, 0.06);
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
  margin: 0 0 10px;
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

.deploy-status-block {
  padding: 14px 16px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 10px;
  background: linear-gradient(135deg, #fffbf0 0%, #fff 100%);
  margin-bottom: 16px;
}

.deploy-status-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.deploy-status-title {
  font-weight: 600;
  font-size: 14px;
  color: var(--el-text-color-primary);
}

.deploy-status-body .deploy-status-meta {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  margin-bottom: 8px;
}

.deploy-status-step {
  margin-left: 12px;
  color: var(--el-text-color-regular);
}

.deploy-records-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.deploy-records-title {
  font-weight: 600;
  font-size: 14px;
  color: var(--el-text-color-primary);
}

.deploy-records-table {
  font-size: 13px;
}

.deploy-records-empty {
  text-align: center;
  color: var(--el-text-color-placeholder);
  padding: 24px;
  font-size: 13px;
}

.deploy-steps {
  margin-bottom: 4px;
  padding: 2px 0 8px;
}

.deploy-steps :deep(.el-step__title) {
  font-size: 12px;
  line-height: 1.35;
}

/* 确认页：模式说明 + 摘要 */
.step-section--confirm {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.confirm-hero {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  padding: 18px 20px;
  border-radius: 12px;
  border: 1px solid var(--el-border-color-lighter);
}

.confirm-hero--offline {
  background: linear-gradient(120deg, #ecfdf5 0%, #f0fdf4 40%, #fff 100%);
  border-color: #a7f3d0;
}

.confirm-hero--online {
  background: linear-gradient(120deg, #eff6ff 0%, #f0f7ff 45%, #fff 100%);
  border-color: var(--el-color-primary-light-9);
}

.confirm-hero-icon {
  flex-shrink: 0;
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.85);
  color: var(--el-color-primary);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
}

.confirm-hero--offline .confirm-hero-icon {
  color: #059669;
}

.confirm-hero-text {
  min-width: 0;
  flex: 1;
}

.confirm-hero-title {
  margin: 0 0 6px;
  font-size: 16px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}

.confirm-hero-desc {
  margin: 0;
  font-size: 13px;
  line-height: 1.65;
  color: var(--el-text-color-regular);
}

.confirm-hero-desc code {
  font-size: 12px;
  padding: 2px 6px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.9);
  border: 1px solid var(--el-border-color-lighter);
}

.offline-install-panel {
  padding: 18px 20px;
  border-radius: 12px;
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
}

.offline-install-panel__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.offline-install-panel__title {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}

.offline-install-panel__placeholder {
  margin: 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.65;
}

.invite-meta-desc {
  margin-bottom: 12px;
}

.install-command-textarea :deep(textarea) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  font-size: 12px;
  line-height: 1.5;
}

.offline-install-panel__warn {
  margin: 10px 0 0;
  font-size: 12px;
  color: var(--el-color-warning-dark-2);
}

.mono-ellipsis {
  font-family: ui-monospace, monospace;
  font-size: 12px;
  word-break: break-all;
}

.confirm-value--wrap {
  flex: 1;
  min-width: 0;
  text-align: right;
  line-height: 1.5;
}

.form-hint--compact {
  margin: -6px 0 12px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.requirement-doc-panel {
  padding: 18px 20px;
  border-radius: 12px;
  border: 1px dashed var(--el-border-color);
  background: var(--el-fill-color-light);
}

.requirement-doc-panel__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 8px;
}

.requirement-doc-panel__title {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}

.requirement-doc-panel__hint {
  margin: 0 0 12px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.55;
}

.requirement-textarea :deep(textarea) {
  font-family: ui-monospace, 'SF Mono', Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 1.5;
  background: var(--el-bg-color);
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
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
  max-height: 380px;
  overflow-y: auto;
  padding-right: 4px;
}

/* ==================== 额外参数网格 ==================== */
.extra-args-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
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

/* ==================== 确认页网格 ==================== */
.confirm-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
}

.confirm-block {
  background: var(--el-fill-color-blank);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  padding: 18px 20px;
  transition: box-shadow 0.2s ease, border-color 0.2s ease;
}

.confirm-block:hover {
  border-color: var(--el-border-color);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.confirm-block-full {
  grid-column: 1 / -1;
}

.confirm-block-title {
  margin: 0 0 12px 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-color-primary);
  padding-bottom: 8px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.confirm-row {
  display: flex;
  margin-bottom: 8px;
}

.confirm-row:last-child {
  margin-bottom: 0;
}

.confirm-label {
  font-weight: 500;
  min-width: 110px;
  color: #6b7280;
  font-size: 13px;
}

.confirm-value {
  color: #374151;
  font-size: 13px;
  word-break: break-all;
}

/* ==================== 底部操作（sticky，便于长确认页仍能看到主操作） ==================== */
.step-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 4px 8px;
  margin-top: 8px;
  position: sticky;
  bottom: 0;
  z-index: 20;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0) 0%, #fff 18%);
  border-top: 1px solid var(--el-border-color-lighter);
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
@media (max-width: 1200px) {
  .extra-args-grid {
    grid-template-columns: 1fr;
  }
  .confirm-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 900px) {
  .label-taint-grid {
    grid-template-columns: 1fr;
  }
}
</style>
