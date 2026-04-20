package handlers

import (
	"encoding/json"
	"fmt"
	"strings"

	"ft-backend/common/logger"
	"ft-backend/common/redis"
	"ft-backend/common/response"
	"ft-backend/database"
	"ft-backend/models"
	"ft-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// K8sDeployRequest matches the frontend K8s deploy form (Ansible 流水线多步).
type K8sDeployRequest struct {
	// Step 1 — Basic
	ClusterName      string `json:"clusterName" binding:"required"`
	Version          string `json:"version"     binding:"required"`
	DeployMode       string `json:"deployMode"`    // single | ha
	ImageSource      string `json:"imageSource"`   // default | aliyun | tencent | custom
	// ArchVersion 节点 CPU 架构（linux：amd64 | arm64），与 kubernetes/etcd 二进制一致；空则默认 amd64。
	ArchVersion      string `json:"archVersion"`
	CustomRegistry   string `json:"customRegistry"`
	RegistryUsername string `json:"registryUsername"`
	RegistryPassword string `json:"registryPassword"`

	// Step 2 — Nodes（UUID，来自机器管理 + Agent）
	ExecutorNode string                 `json:"executorNode"` // 执行部署的 Agent 节点（可选，不填则回退到首个 Master）
	MasterNodes  []string               `json:"masterNodes"`
	WorkerNodes  []string               `json:"workerNodes"`
	// 离线安装包：直接填节点 IP/主机名，无需 Agent（与 MasterNodes 二选一）
	MasterHosts []string `json:"masterHosts"`
	WorkerHosts []string `json:"workerHosts"`
	MasterLabels map[string]string      `json:"masterLabels"`
	WorkerLabels map[string]string      `json:"workerLabels"`

	// Step 3 — Core components
	KubeProxyMode           string `json:"kubeProxyMode"` // iptables | ipvs
	EnableRBAC              bool   `json:"enableRBAC"`
	EnablePodSecurityPolicy bool   `json:"enablePodSecurityPolicy"`
	EnableAudit             bool   `json:"enableAudit"`
	AuditPolicy             string `json:"auditPolicy"`
	PauseImage              string `json:"pauseImage"`

	// Step 4 — Network
	NetworkPlugin string          `json:"networkPlugin"` // calico | flannel | cilium | weave
	PodCIDR       string          `json:"podCidr"`
	ServiceCIDR   string          `json:"serviceCidr"`
	DNSServiceIP  string          `json:"dnsServiceIP"`
	ClusterDomain string          `json:"clusterDomain"`
	CalicoConfig  json.RawMessage `json:"calicoConfig"`
	FlannelConfig json.RawMessage `json:"flannelConfig"`

	// Step 5 — Storage
	DefaultStorageClass bool            `json:"defaultStorageClass"`
	StorageProvisioner  string          `json:"storageProvisioner"` // local-path | nfs-client | csi
	StorageConfig       json.RawMessage `json:"storageConfig"`

	// Step 6 — Advanced
	EnableNodeLocalDNS  bool             `json:"enableNodeLocalDNS"`
	EnableMetricsServer bool             `json:"enableMetricsServer"`
	EnableDashboard     bool             `json:"enableDashboard"`
	EnablePrometheus    bool             `json:"enablePrometheus"`
	EnableIngressNginx  bool             `json:"enableIngressNginx"`
	EnableHelm          bool             `json:"enableHelm"`
	ExtraKubeletArgs    []kvPair         `json:"extraKubeletArgs"`
	ExtraKubeProxyArgs  []kvPair         `json:"extraKubeProxyArgs"`
	ExtraAPIServerArgs  []kvPair         `json:"extraAPIServerArgs"`

	// Legacy catch-all (kept for backward compat)
	Config json.RawMessage `json:"config"`

	// 离线 install.sh / 在线执行脚本：是否在 Step 0 运行 playbooks/pre_cleanup.yml（非交互）
	PreDeployCleanup bool `json:"preDeployCleanup"`

	// 内网制品下载（可选）：覆盖 ansible-agent/inventory/group_vars/all.yml 中的 download_domain / download_protocol。
	// 留空则使用 zip 内合并后的 inventory 默认值（仅改机房时编辑 all.yml 即可，无需改代码）。
	DownloadDomain   string `json:"downloadDomain"`
	DownloadProtocol string `json:"downloadProtocol"`
}

// kvPair is a generic key-value pair from the frontend form.
type kvPair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// normalizeK8sCPUArch maps UI / uname 常见写法到 ansible 使用的 linux 架构名。
func normalizeK8sCPUArch(s string) string {
	a := strings.ToLower(strings.TrimSpace(s))
	switch a {
	case "", "x86_64", "amd64":
		return "amd64"
	case "aarch64", "arm64":
		return "arm64"
	default:
		return "amd64"
	}
}

// normalizeImageSource 统一镜像源字段，避免 JSON 首尾空格、大小写导致「选了阿里云却未命中 dl.k8s.io 覆盖」。
func normalizeImageSource(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return "default"
	}
	return s
}

// normalizeDownloadProtocol 将 UI/CLI 输入规范为 ansible 使用的前缀（如 http://、https://）。
func normalizeDownloadProtocol(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	low := strings.ToLower(p)
	switch low {
	case "http", "http://":
		return "http://"
	case "https", "https://":
		return "https://"
	}
	if strings.HasSuffix(p, "://") {
		return p
	}
	return p + "://"
}

// generateAnsibleInventory produces a dynamic Ansible inventory based on the deploy request.
func generateAnsibleInventory(req K8sDeployRequest, machines map[string]models.Machine) string {
	inv := "[control]\nlocalhost ansible_connection=local\n\n"

	inv += "[kube_control_plane]\n"
	for _, id := range req.MasterNodes {
		if m, ok := machines[id]; ok {
			inv += fmt.Sprintf("%s ansible_host=%s\n", m.Name, m.IP)
		}
	}

	inv += "\n[kube_node]\n"
	for _, id := range req.WorkerNodes {
		if m, ok := machines[id]; ok {
			inv += fmt.Sprintf("%s ansible_host=%s\n", m.Name, m.IP)
		}
	}

	inv += "\n[etcd]\n"
	// etcd runs on control plane nodes
	for _, id := range req.MasterNodes {
		if m, ok := machines[id]; ok {
			inv += fmt.Sprintf("%s ansible_host=%s\n", m.Name, m.IP)
		}
	}

	inv += "\n[k8s_cluster:children]\nkube_control_plane\nkube_node\n\n"
	inv += "[all:vars]\nansible_user=root\nansible_ssh_common_args='-o StrictHostKeyChecking=no'\n"

	return inv
}

// generateAnsibleGroupVars produces group_vars/all.yml from the full deploy request.
func generateAnsibleGroupVars(req K8sDeployRequest, masterIP string) string {
	podCIDR := req.PodCIDR
	if podCIDR == "" {
		podCIDR = "10.244.0.0/16"
	}
	serviceCIDR := req.ServiceCIDR
	if serviceCIDR == "" {
		serviceCIDR = "10.96.0.0/12"
	}
	dnsServiceIP := req.DNSServiceIP
	if dnsServiceIP == "" {
		dnsServiceIP = "10.96.0.10"
	}
	clusterDomain := req.ClusterDomain
	if clusterDomain == "" {
		clusterDomain = "cluster.local"
	}
	networkPlugin := req.NetworkPlugin
	if networkPlugin == "" {
		// Ansible 流水线当前内置 Flannel + CoreDNS；Calico 等需后续接入。
		networkPlugin = "flannel"
	}
	kubeProxyMode := req.KubeProxyMode
	if kubeProxyMode == "" {
		kubeProxyMode = "iptables"
	}
	imageSource := normalizeImageSource(req.ImageSource)
	storageProvisioner := req.StorageProvisioner
	if storageProvisioner == "" {
		storageProvisioner = "local-path"
	}
	cpuArch := normalizeK8sCPUArch(req.ArchVersion)

	// Build optional component flags as YAML booleans.
	boolStr := func(b bool) string {
		if b {
			return "true"
		}
		return "false"
	}

	// Build extra-args YAML block.
	extraArgsYAML := func(pairs []kvPair, varName string) string {
		if len(pairs) == 0 {
			return ""
		}
		out := fmt.Sprintf("\n%s:\n", varName)
		for _, p := range pairs {
			out += fmt.Sprintf("  %s: \"%s\"\n", p.Key, p.Value)
		}
		return out
	}

	vars := fmt.Sprintf(`---
# Auto-generated by OpsFleetPilot — do not edit manually.
# 持久化缓存目录（与 install.sh mkdir、resources 角色一致；写入 overlay 防止仅解析部分 YAML 时缺失）
local_cache_dir: "/var/cache/opsfleet-k8s"
remote_dest: "/usr/local"

kubernetes_version: "%s"
cluster_name: "%s"
master_ip: "%s"

# Network
pod_cluster_cidr: "%s"
service_cluster_ip_range: "%s"
dns_service_ip: "%s"
cluster_domain: "%s"
network_plugin: "%s"
kube_proxy_mode: "%s"

# Image source
image_source: "%s"

# CPU / binary arch (amd64=x86_64, arm64=AArch64/Apple Silicon 虚拟机常见)
arch_version: "%s"
`,
		req.Version, req.ClusterName, masterIP,
		podCIDR, serviceCIDR, dnsServiceIP, clusterDomain,
		networkPlugin, kubeProxyMode,
		imageSource,
		cpuArch,
	)

	if d := strings.TrimSpace(req.DownloadDomain); d != "" {
		vars += fmt.Sprintf("download_domain: \"%s\"\n", d)
	}
	if p := strings.TrimSpace(req.DownloadProtocol); p != "" {
		vars += fmt.Sprintf("download_protocol: \"%s\"\n", normalizeDownloadProtocol(p))
	}

	if req.CustomRegistry != "" {
		vars += fmt.Sprintf("custom_registry: \"%s\"\n", req.CustomRegistry)
		if req.RegistryUsername != "" {
			vars += fmt.Sprintf("registry_username: \"%s\"\n", req.RegistryUsername)
		}
	}

	// Security / RBAC
	vars += fmt.Sprintf(`
# Security
enable_rbac: %s
enable_pod_security_policy: %s
enable_audit: %s
`,
		boolStr(req.EnableRBAC),
		boolStr(req.EnablePodSecurityPolicy),
		boolStr(req.EnableAudit),
	)

	// Storage
	vars += fmt.Sprintf(`
# Storage
default_storage_class: %s
storage_provisioner: "%s"
`,
		boolStr(req.DefaultStorageClass),
		storageProvisioner,
	)

	// Optional components
	vars += fmt.Sprintf(`
# Optional components
enable_node_local_dns: %s
enable_metrics_server: %s
enable_dashboard: %s
enable_prometheus: %s
enable_ingress_nginx: %s
enable_helm: %s
`,
		boolStr(req.EnableNodeLocalDNS),
		boolStr(req.EnableMetricsServer),
		boolStr(req.EnableDashboard),
		boolStr(req.EnablePrometheus),
		boolStr(req.EnableIngressNginx),
		boolStr(req.EnableHelm),
	)

	// Extra args
	vars += extraArgsYAML(req.ExtraKubeletArgs, "extra_kubelet_args")
	vars += extraArgsYAML(req.ExtraKubeProxyArgs, "extra_kube_proxy_args")
	vars += extraArgsYAML(req.ExtraAPIServerArgs, "extra_apiserver_args")

	// 阿里云镜像源：Kubernetes 官方分发域（与阿里云文档常用做法一致：二进制 tarball + 远程校验收据；镜像仓库前缀供 sandbox/coredns 等拉取）
	if imageSource == "aliyun" {
		vars += `
# Aliyun image source — public download URLs (Ansible 会展开 {{ }} 占位符)
k8s_server_tarball_url: "https://dl.k8s.io/{{ kubernetes_version }}/{{ k8s_package_name }}.tar.gz"
k8s_server_tarball_checksum: "sha512:https://dl.k8s.io/{{ kubernetes_version }}/{{ k8s_package_name }}.tar.gz.sha512"
etcd_download_url: "https://github.com/etcd-io/etcd/releases/download/{{ etcd_version }}/{{ etcd_package_name }}.tar.gz"
k8s_image_repository: "registry.aliyuncs.com/google_containers"
# CNI 插件走公网（与内网 download_domain 解耦）
cni_plugins_download_url: "https://github.com/containernetworking/plugins/releases/download/{{ cni_plugins_version }}/cni-plugins-linux-{{ arch_version }}-{{ cni_plugins_version }}.tgz"
`
	}

	if req.PreDeployCleanup {
		vars += "\npre_deploy_cleanup: true\n"
	} else {
		vars += "\npre_deploy_cleanup: false\n"
	}

	return vars
}

// DEPLOY_STATE_FILE 用于记录已完成的部署步骤，终止清理时按逆序执行对应清理。
const deployStateFile = "/tmp/ofp-k8s-deploy-state"

// generateK8sDeployScript creates a shell script that runs the Ansible playbooks in order.
// 每完成一步即写入 DEPLOY_STATE_FILE，便于终止时按步骤做清理。
func generateK8sDeployScript(inventoryContent, groupVarsContent string, preDeployCleanup bool) string {
	preBlock := ""
	if preDeployCleanup {
		preBlock = `
echo "=== Step 0: Pre-deploy cleanup (non-interactive) ==="
ansible-playbook -i "$TEMP_INVENTORY" playbooks/pre_cleanup.yml || { echo "FAILED: pre_cleanup"; exit 1; }
`
	}
	return fmt.Sprintf(`#!/bin/bash
set -e
STATE_FILE="%s"
rm -f "$STATE_FILE"

ANSIBLE_DIR="/opt/opsfleetpilot/ansible-agent"
TEMP_INVENTORY="/tmp/ofp-inventory-$$.ini"
TEMP_VARS="/tmp/ofp-group-vars-$$.yml"

echo "=== OpsFleetPilot K8s Deployment ==="

if [ ! -d "$ANSIBLE_DIR" ]; then
    echo "ERROR: ansible-agent directory not found at $ANSIBLE_DIR"
    exit 1
fi

cat > "$TEMP_INVENTORY" << 'INVENTORY_EOF'
%s
INVENTORY_EOF

mkdir -p "$ANSIBLE_DIR/inventory/group_vars"
cat > "$TEMP_VARS" << 'VARS_EOF'
%s
VARS_EOF
cp "$TEMP_VARS" "$ANSIBLE_DIR/inventory/group_vars/all.yml"

cd "$ANSIBLE_DIR"
%s
echo "=== Step 1/11: System Initialization ==="
ansible-playbook -i "$TEMP_INVENTORY" playbooks/0-init.yml || { echo "FAILED: init"; exit 1; }
echo "init" >> "$STATE_FILE"

echo "=== Step 2/11: Download Resources ==="
ansible-playbook -i "$TEMP_INVENTORY" playbooks/resources.yml || { echo "FAILED: resources"; exit 1; }
echo "resources" >> "$STATE_FILE"

echo "=== Step 3/11: Deploy etcd Cluster ==="
ansible-playbook -i "$TEMP_INVENTORY" playbooks/etcd.yml || { echo "FAILED: etcd"; exit 1; }
echo "etcd" >> "$STATE_FILE"

echo "=== Step 4/11: Deploy kube-apiserver ==="
ansible-playbook -i "$TEMP_INVENTORY" playbooks/kube_apiserver_install.yml || { echo "FAILED: apiserver"; exit 1; }
echo "apiserver" >> "$STATE_FILE"

echo "=== Step 5/11: Deploy kube-controller-manager ==="
ansible-playbook -i "$TEMP_INVENTORY" playbooks/kube_controller_manager_install.yml || { echo "FAILED: controller-manager"; exit 1; }
echo "controller_manager" >> "$STATE_FILE"

echo "=== Step 6/11: Deploy kube-scheduler ==="
ansible-playbook -i "$TEMP_INVENTORY" playbooks/kube_scheduler_install.yml || { echo "FAILED: scheduler"; exit 1; }
echo "scheduler" >> "$STATE_FILE"

echo "=== Step 7/11: Install kubectl ==="
ansible-playbook -i "$TEMP_INVENTORY" playbooks/kubectl.yml || { echo "FAILED: kubectl"; exit 1; }
echo "kubectl" >> "$STATE_FILE"

echo "=== Step 8/11: Install containerd ==="
ansible-playbook -i "$TEMP_INVENTORY" playbooks/containerd.yml || { echo "FAILED: containerd"; exit 1; }
echo "containerd" >> "$STATE_FILE"

echo "=== Step 9/11: Install kubelet ==="
ansible-playbook -i "$TEMP_INVENTORY" playbooks/kubelet.yml || { echo "FAILED: kubelet"; exit 1; }
echo "kubelet" >> "$STATE_FILE"

echo "=== Step 10/11: Install kube-proxy ==="
ansible-playbook -i "$TEMP_INVENTORY" playbooks/kube_proxy.yml || { echo "FAILED: kube-proxy"; exit 1; }
echo "kube_proxy" >> "$STATE_FILE"

echo "=== Step 11/11: Apply addons (Flannel + CoreDNS) ==="
ansible-playbook -i "$TEMP_INVENTORY" playbooks/k8s_addons.yml || { echo "FAILED: k8s_addons"; exit 1; }
echo "k8s_addons" >> "$STATE_FILE"

echo "=== K8s Deployment Completed Successfully! ==="
rm -f "$TEMP_INVENTORY" "$TEMP_VARS"
kubectl get nodes || echo "WARNING: kubectl check failed"
kubectl get pods -A || true
`, deployStateFile, inventoryContent, groupVarsContent, preBlock)
}

// generateK8sCleanupScript 生成清理脚本：按部署步骤的逆序执行清理，严格恢复到部署前状态。
// 各步骤清理幂等，未执行过的步骤清理也无害。
func generateK8sCleanupScript() string {
	return `#!/bin/bash
set +e
STATE_FILE="` + deployStateFile + `"
echo "=== OpsFleetPilot K8s Deploy Cleanup (restore to pre-deploy state) ==="

cleanup_kubectl() {
  systemctl stop kube-proxy kubelet 2>/dev/null || true
  rm -f /usr/local/bin/kubectl /usr/local/bin/kubelet /usr/local/bin/kube-proxy
  rm -rf /root/.kube /etc/kubernetes/kubelet.conf /etc/kubernetes/kube-proxy.conf 2>/dev/null || true
}
cleanup_scheduler() {
  systemctl stop kube-scheduler 2>/dev/null || true
  systemctl disable kube-scheduler 2>/dev/null || true
  rm -f /etc/systemd/system/kube-scheduler.service /etc/kubernetes/kube-scheduler*.kubeconfig 2>/dev/null || true
}
cleanup_controller_manager() {
  systemctl stop kube-controller-manager 2>/dev/null || true
  systemctl disable kube-controller-manager 2>/dev/null || true
  rm -f /etc/systemd/system/kube-controller-manager.service /etc/kubernetes/kube-controller-manager*.kubeconfig 2>/dev/null || true
}
cleanup_apiserver() {
  systemctl stop kube-apiserver 2>/dev/null || true
  systemctl disable kube-apiserver 2>/dev/null || true
  rm -f /etc/systemd/system/kube-apiserver.service /etc/kubernetes/kube-apiserver*.kubeconfig /etc/kubernetes/token.csv 2>/dev/null || true
}
cleanup_etcd() {
  systemctl stop etcd 2>/dev/null || true
  systemctl disable etcd 2>/dev/null || true
  rm -rf /var/lib/etcd /etc/etcd /etc/systemd/system/etcd.service 2>/dev/null || true
}
cleanup_resources() {
  rm -rf /opt/kubernetes/bin /opt/kubernetes/pkg /tmp/kubernetes 2>/dev/null || true
}
cleanup_init() {
  rm -rf /etc/kubernetes /var/lib/kubelet 2>/dev/null || true
}

# 按部署逆序执行清理（kubectl -> ... -> init）
cleanup_kubectl
cleanup_scheduler
cleanup_controller_manager
cleanup_apiserver
cleanup_etcd
cleanup_resources
cleanup_init

rm -f "$STATE_FILE" /tmp/ofp-inventory-*.ini /tmp/ofp-group-vars-*.yml 2>/dev/null || true
systemctl daemon-reload 2>/dev/null || true
echo "=== K8s cleanup finished ==="
`
}

// SubmitK8sDeployWithAnsible handles K8s deploy submission with Ansible integration.
func SubmitK8sDeployWithAnsible(c *gin.Context) {
	var req K8sDeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}
	if len(req.MasterHosts) > 0 {
		response.BadRequest(c, "在线部署不能填写 masterHosts；请使用控制台「生成离线安装包」下载后在目标机执行")
		return
	}
	if len(req.MasterNodes) == 0 {
		response.BadRequest(c, "至少选择一个 K8s 控制平面节点")
		return
	}

	username, _ := c.Get("username")

	// Look up all machines (executor + master + worker)
	allNodeIDs := append([]string{}, req.MasterNodes...)
	allNodeIDs = append(allNodeIDs, req.WorkerNodes...)
	if req.ExecutorNode != "" {
		allNodeIDs = append(allNodeIDs, req.ExecutorNode)
	}
	machineMap := make(map[string]models.Machine)
	for _, id := range allNodeIDs {
		uid, err := uuid.Parse(id)
		if err != nil {
			continue
		}
		var m models.Machine
		if err := database.DB.Where("id = ?", uid).First(&m).Error; err == nil {
			machineMap[id] = m
		}
	}

	// Get master IP for Ansible vars (K8s API Server 地址，来自首个控制平面节点)
	masterIP := ""
	if len(req.MasterNodes) > 0 {
		if m, ok := machineMap[req.MasterNodes[0]]; ok {
			masterIP = m.IP
		}
	}
	if masterIP == "" {
		response.BadRequest(c, "K8s 控制平面节点 IP 无法解析")
		return
	}

	// 确定执行节点：优先使用 executorNode，否则回退到首个 Master（兼容旧前端）
	executorID := req.ExecutorNode
	if executorID == "" {
		executorID = req.MasterNodes[0]
	}
	executorMachine, ok := machineMap[executorID]
	if !ok {
		response.BadRequest(c, "执行节点不存在")
		return
	}
	// 执行节点只需：在线 + 有 client_id（Agent），与 K8s 集群节点无关联
	if executorMachine.Status != "online" {
		response.BadRequest(c, "所选执行节点离线，无法接收部署任务")
		return
	}
	if executorMachine.ClientID == "" {
		response.BadRequest(c, "所选执行节点尚未上报 client_id，请等待 Agent 心跳成功后再部署")
		return
	}

	// Generate Ansible artifacts
	inventoryContent := generateAnsibleInventory(req, machineMap)
	groupVarsContent := generateAnsibleGroupVars(req, masterIP)
	deployScript := generateK8sDeployScript(inventoryContent, groupVarsContent, req.PreDeployCleanup)

	// Create K8s cluster record — store the full request config for auditability.
	workerNodesJSON, _ := json.Marshal(req.WorkerNodes)
	configJSON, _ := json.Marshal(map[string]interface{}{
		"deploy_mode":          req.DeployMode,
		"network_plugin":       req.NetworkPlugin,
		"pod_cidr":             req.PodCIDR,
		"service_cidr":         req.ServiceCIDR,
		"dns_service_ip":       req.DNSServiceIP,
		"cluster_domain":       req.ClusterDomain,
		"image_source":         req.ImageSource,
		"kube_proxy_mode":      req.KubeProxyMode,
		"enable_rbac":          req.EnableRBAC,
		"storage_provisioner":  req.StorageProvisioner,
		"enable_metrics_server": req.EnableMetricsServer,
		"enable_dashboard":     req.EnableDashboard,
		"enable_prometheus":    req.EnablePrometheus,
		"enable_ingress_nginx": req.EnableIngressNginx,
		"enable_helm":          req.EnableHelm,
	})

	cluster := models.K8sCluster{
		ClusterName: req.ClusterName,
		Status:      "deploying",
		Version:     req.Version,
		MasterNode:  masterIP,
		WorkerNodes: models.JSONB(workerNodesJSON),
		Config:      models.JSONB(configJSON),
	}

	tx := database.DB.Begin()
	if err := tx.Create(&cluster).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "创建集群记录失败")
		return
	}

	// Create the deployment task targeting the executor node (Agent 所在机器，执行 Ansible)
	payload, _ := json.Marshal(map[string]interface{}{
		"script":       deployScript,
		"cluster_id":   cluster.ID.String(),
		"cluster_name": req.ClusterName,
	})
	targetIDsJSON, _ := json.Marshal([]string{executorID})

	task := models.Task{
		Name:        "K8s部署: " + req.ClusterName,
		Type:        string(models.TaskTypeK8sDeploy),
		Status:      string(models.TaskStatusPending),
		CreatedBy:   username.(string),
		Description: "Kubernetes " + req.Version + " 集群部署 (Ansible)",
		Payload:     models.JSONB(payload),
		TargetIDs:   models.JSONB(targetIDsJSON),
		TotalCount:  1, // Ansible 在 executor 节点执行，SSH 到 Master/Worker 节点
		TimeoutSec:  3600,
	}
	if err := tx.Create(&task).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "创建部署任务失败")
		return
	}

	// Create sub-task for the executor (Agent 所在机器执行 Ansible，与 K8s 集群节点解耦)
	dispatchID := executorMachine.ClientID
	if dispatchID == "" {
		dispatchID = executorMachine.IP
	}
	subTask := models.SubTask{
		TaskID:    task.ID,
		MachineID: executorMachine.ID,
		ClientID:  dispatchID,
		Command:   "install_k8s",
		Status:    string(models.TaskStatusPending),
		Payload:   models.JSONB(payload),
		MaxRetry:  1,
	}
	if err := tx.Create(&subTask).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "创建子任务失败")
		return
	}

	tx.Create(&models.TaskLog{
		TaskID:  task.ID,
		Level:   "info",
		Message: fmt.Sprintf("K8s集群 %s 部署任务已创建 (Ansible集成)", req.ClusterName),
	})

	if err := tx.Commit().Error; err != nil {
		logger.Error("K8s deploy commit failed: %v", err)
		response.ServerError(c, "创建部署任务失败")
		return
	}

	// Broadcast initial progress so deploy form / progress page can show the new task via WS.
	if utils.GlobalWebSocketManager != nil {
		go BroadcastK8sDeployProgress(task.ID)
	}

	// 入队 Redis 以便下次心跳立即下发（与 register_nodes 一致）；未配置 Redis 时仍靠 DB 查询下发。
	if redis.IsConnected() {
		cmd := models.Command{
			TaskID:    task.ID.String(),
			SubTaskID: subTask.ID.String(),
			Command:   "install_k8s",
			Payload:   json.RawMessage(payload),
			Timeout:   task.TimeoutSec,
		}
		for _, key := range []string{executorMachine.ClientID, executorMachine.IP} {
			if key == "" {
				continue
			}
			if err := redis.EnqueueTask(key, cmd); err != nil {
				logger.Warn("K8s deploy enqueue Redis failed for key=%s: %v", key, err)
			}
		}
		logger.Info("K8s deploy task enqueued to Redis for client_id=%s ip=%s", executorMachine.ClientID, executorMachine.IP)
	}

	response.OK(c, gin.H{
		"deployId":    task.ID.String(),
		"clusterName": req.ClusterName,
		"status":      "deploying",
	})
}
