package handlers

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ft-backend/common/logger"
	"ft-backend/common/response"

	"github.com/gin-gonic/gin"
)

// ErrK8sBundleMissingMasters 表示离线包请求缺少 masterHosts。
var ErrK8sBundleMissingMasters = errors.New("k8s offline bundle: no master hosts")

// ErrK8sBundleAnsibleDir 表示未找到 ansible-agent 目录。
var ErrK8sBundleAnsibleDir = errors.New("k8s offline bundle: ansible-agent directory not found")

// BuildK8sOfflineZip 生成与 HTTP 接口相同的 zip 字节流（供 CLI/CI/脚本调用）。
func BuildK8sOfflineZip(req K8sDeployRequest) ([]byte, error) {
	masters := normalizeHostList(req.MasterHosts)
	workers := normalizeHostList(req.WorkerHosts)
	if len(masters) == 0 {
		return nil, ErrK8sBundleMissingMasters
	}
	ansibleDir := resolveAnsibleAgentDir()
	if ansibleDir == "" {
		return nil, ErrK8sBundleAnsibleDir
	}
	masterIP := masters[0]
	inventoryContent := generateAnsibleInventoryFromHosts(masters, workers)
	groupVarsContent := generateAnsibleGroupVars(req, masterIP)
	mergedGroupVars, err := mergeK8sInventoryGroupVars(ansibleDir, groupVarsContent)
	if err != nil {
		return nil, err
	}
	buf, err := buildK8sOfflineZip(req.ClusterName, ansibleDir, inventoryContent, mergedGroupVars)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GenerateK8sOfflineBundle 根据表单配置打包 ansible-agent + inventory + install.sh，供 Ubuntu 24.04 上一键执行。
// 不依赖 ft-client / 机器管理；需用户填写 masterHosts（及 workerHosts）。
func GenerateK8sOfflineBundle(c *gin.Context) {
	var req K8sDeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}
	data, err := BuildK8sOfflineZip(req)
	if err != nil {
		switch {
		case errors.Is(err, ErrK8sBundleMissingMasters):
			response.BadRequest(c, "请至少填写一个 control plane 节点 IP（masterHosts）")
			return
		case errors.Is(err, ErrK8sBundleAnsibleDir):
			logger.Error("ansible-agent directory not found (set OPSFLEET_ANSIBLE_DIR or run from repo with ../ansible-agent)")
			response.ServerError(c, "服务器未找到 ansible-agent 目录，无法生成离线包")
			return
		default:
			logger.Error("build offline zip: %v", err)
			response.ServerError(c, "打包失败: "+err.Error())
			return
		}
	}
	safeName := sanitizeBundleFilePrefix(req.ClusterName)
	filename := fmt.Sprintf("opsfleet-k8s-%s-%s.zip", safeName, time.Now().Format("20060102-150405"))
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(data)))
	c.Data(http.StatusOK, "application/zip", data)
}

func normalizeHostList(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

func sanitizeBundleFilePrefix(name string) string {
	b := strings.Builder{}
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else if r == ' ' || r == '_' {
			b.WriteRune('-')
		}
	}
	s := b.String()
	if s == "" {
		return "cluster"
	}
	return s
}

// resolveAnsibleAgentDir 查找仓库内 ansible-agent（相对 ft-backend 工作目录一般为 ..）。
func resolveAnsibleAgentDir() string {
	if d := os.Getenv("OPSFLEET_ANSIBLE_DIR"); d != "" {
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			return d
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	candidates := []string{
		filepath.Join(wd, "..", "ansible-agent"),
		filepath.Join(wd, "ansible-agent"),
	}
	for _, cand := range candidates {
		if st, err := os.Stat(cand); err == nil && st.IsDir() {
			if _, err := os.Stat(filepath.Join(cand, "playbooks")); err == nil {
				return filepath.Clean(cand)
			}
		}
	}
	return ""
}

// mergeK8sInventoryGroupVars 将控制台生成的 YAML 片段追加到 ansible-agent 内 inventory/group_vars/all.yml 之后。
// 使用追加而非整文件反序列化，避免破坏内含 Jinja（如 groups['kube_control_plane']）的字符串。
// Ansible 解析同一文件内重复顶级键时以后者为准（与大多数 YAML/Ansible 行为一致）。
func mergeK8sInventoryGroupVars(ansibleRoot, overlayYAML string) (string, error) {
	basePath := filepath.Join(ansibleRoot, "inventory", "group_vars", "all.yml")
	baseBytes, err := os.ReadFile(basePath)
	if err != nil {
		return "", fmt.Errorf("read base group_vars: %w", err)
	}
	over := strings.TrimSpace(overlayYAML)
	for strings.HasPrefix(over, "---") {
		over = strings.TrimSpace(over[3:])
	}
	base := strings.TrimRight(string(baseBytes), "\n\r")
	var b strings.Builder
	b.WriteString("# Merged: ansible-agent inventory/group_vars + OpsFleet UI overrides (块在后，覆盖同名键)\n")
	b.WriteString(base)
	b.WriteString("\n\n# --- OpsFleetPilot UI overrides ---\n")
	b.WriteString(over)
	b.WriteString("\n")
	return b.String(), nil
}

func generateAnsibleInventoryFromHosts(masters, workers []string) string {
	inv := "[control]\nlocalhost ansible_connection=local\n\n"
	inv += "[kube_control_plane]\n"
	for i, ip := range masters {
		name := fmt.Sprintf("k8s-master-%d", i)
		inv += fmt.Sprintf("%s ansible_host=%s\n", name, ip)
	}
	inv += "\n[kube_node]\n"
	for i, ip := range workers {
		name := fmt.Sprintf("k8s-worker-%d", i)
		inv += fmt.Sprintf("%s ansible_host=%s\n", name, ip)
	}
	inv += "\n[etcd]\n"
	for i, ip := range masters {
		name := fmt.Sprintf("k8s-master-%d", i)
		inv += fmt.Sprintf("%s ansible_host=%s\n", name, ip)
	}
	inv += "\n[k8s_cluster:children]\nkube_control_plane\nkube_node\n\n"
	inv += "[all:vars]\nansible_user=root\nansible_ssh_common_args='-o StrictHostKeyChecking=no'\n"
	return inv
}

func buildK8sOfflineZip(clusterName, ansibleRoot, inventory, groupVars string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	addString := func(name, content string) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, strings.NewReader(content))
		return err
	}

	readme := fmt.Sprintf(`OpsFleetPilot Kubernetes 离线安装包
集群名称: %s
生成时间(UTC): %s

【必须先满足】在将要执行 install.sh 的那台机器上，能对 inventory 里每个节点 IP 免密 SSH 登录 root。
若 ansible 报错 Permission denied (publickey,password)，说明未完成本项。

最少操作示例（在「执行 install.sh 的机器」上，对每个节点 IP 各执行一次）:
  ssh-keygen -t ed25519 -N "" -f ~/.ssh/id_ed25519
  ssh-copy-id -i ~/.ssh/id_ed25519.pub root@<节点IP>
  ssh root@<节点IP>    # 确认无密码即可登录

说明: install.sh 会自动准备 /root/.ssh/ansible_id_rsa.pub（优先复制已有的 id_ed25519.pub / id_rsa.pub），
      供第 1 步 playbook 在节点上创建 ansible 用户并写入 authorized_keys。
      若曾使用旧版 zip，请重新下载离线包以更新 ansible-agent（旧包 init role 会报 lookup file 类错误）。

inventory 已设 ansible_user=root；当前不支持交互式输入 SSH 密码。

步骤:
 1. 解压: unzip opsfleet-k8s-*.zip && cd 解压目录
 2. 执行: sudo bash install.sh（脚本开头会预检 SSH，失败会提示）

说明:
 - ansible-agent/ 为内置 Playbook；inventory/ 为根据控制台表单生成的清单与变量。
 - apt 输出里若出现 “No VM guests are running outdated hypervisor (qemu)” 可忽略。
 - 若仅单机 All-in-One，可只填一个 master IP；多节点需提前能 SSH 到各 IP。
 - 网络与镜像参数已写入 inventory/group_vars/all.yml。
 - 控制台选择「阿里云」镜像源时，Kubernetes 服务端二进制使用 https://dl.k8s.io 与发布页 sha512；etcd 使用 GitHub Release；pause/coredns 等容器前缀见 k8s_image_repository。
 - 验证：解压后打开 inventory/group_vars/all.yml，在「OpsFleetPilot UI overrides」段应含 image_source: aliyun 与 k8s_server_tarball_url: https://dl.k8s.io/...；若仍为内网 IP，请升级 OpsFleet 后端至含 normalizeImageSource 的版本并重新生成 zip。
 - 内网自建二进制镜像站请选「默认」镜像源并配置 inventory 中 download_domain。
`, clusterName, time.Now().UTC().Format(time.RFC3339))

	if err := addString("README.txt", readme); err != nil {
		zw.Close()
		return nil, err
	}
	if err := addString("inventory/hosts.ini", inventory); err != nil {
		zw.Close()
		return nil, err
	}
	if err := addString("inventory/group_vars/all.yml", groupVars); err != nil {
		zw.Close()
		return nil, err
	}

	// 嵌入 ansible-agent 目录树
	err := filepath.Walk(ansibleRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(ansibleRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		// 跳过编辑器临时文件与隐藏目录
		base := filepath.Base(path)
		if strings.HasPrefix(base, ".") && base != ".gitignore" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".swp") {
			return nil
		}
		zipPath := filepath.Join("ansible-agent", filepath.ToSlash(rel))
		if info.IsDir() {
			return nil
		}
		w, err := zw.Create(zipPath)
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(w, f)
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if err != nil {
		zw.Close()
		return nil, err
	}

	installSh := renderOfflineInstallScript()
	if err := addString("install.sh", installSh); err != nil {
		zw.Close()
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf, nil
}

func renderOfflineInstallScript() string {
	// 与 generateK8sDeployScript 步骤一致，但使用包内路径；在控制节点本机执行（与 inventory 中 control 段 localhost 一致时需能 SSH 到各节点）。
	return `#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STATE_FILE="$ROOT/.opsfleet-k8s-state"
rm -f "$STATE_FILE"

echo "=== OpsFleetPilot K8s 离线安装（Ubuntu 24.04+） ==="
if [[ -r /etc/os-release ]]; then
  # shellcheck source=/dev/null
  . /etc/os-release
  echo "检测到: $PRETTY_NAME"
  if [[ "${VERSION_ID:-}" != "24.04" ]]; then
    echo "提示: 建议在 Ubuntu 24.04 LTS 上运行；继续执行..."
  fi
fi

if [[ "${EUID:-0}" -ne 0 ]]; then
  echo "请使用 root 或 sudo 运行: sudo bash install.sh"
  exit 1
fi

export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y python3 python3-pip ansible sshpass openssh-client rsync

ANSIBLE_DIR="$ROOT/ansible-agent"
INV="$ROOT/inventory/hosts.ini"
VARS_DIR="$ROOT/inventory/group_vars"
mkdir -p "$VARS_DIR"
# group_vars 已随包携带
export ANSIBLE_HOST_KEY_CHECKING=False

# ansible-agent roles/init 在控制机上读取 $PUB 并通过 slurp 写入各节点的 ansible 用户 authorized_keys（与 root 免密无关）
ensure_ansible_controller_keypair() {
  mkdir -p /root/.ssh
  chmod 700 /root/.ssh
  local pub=/root/.ssh/ansible_id_rsa.pub
  local priv=/root/.ssh/ansible_id_rsa
  if [[ -f "$pub" ]]; then
    echo "=== 已存在 $pub，供 init playbook 使用 ==="
    return 0
  fi
  if [[ -f /root/.ssh/id_ed25519.pub ]]; then
    echo "=== 复制已有 /root/.ssh/id_ed25519.pub -> $pub（init playbook 需要此固定文件名）==="
    cp -a /root/.ssh/id_ed25519.pub "$pub"
    if [[ -f /root/.ssh/id_ed25519 ]] && [[ ! -f "$priv" ]]; then
      cp -a /root/.ssh/id_ed25519 "$priv"
      chmod 600 "$priv"
    fi
    return 0
  fi
  if [[ -f /root/.ssh/id_rsa.pub ]]; then
    echo "=== 复制已有 /root/.ssh/id_rsa.pub -> $pub ==="
    cp -a /root/.ssh/id_rsa.pub "$pub"
    if [[ -f /root/.ssh/id_rsa ]] && [[ ! -f "$priv" ]]; then
      cp -a /root/.ssh/id_rsa "$priv"
      chmod 600 "$priv"
    fi
    return 0
  fi
  echo "=== 生成 $priv（init playbook 写入各节点 ansible 用户）==="
  ssh-keygen -t ed25519 -N "" -f "$priv" -C "opsfleet-k8s-ansible"
}
ensure_ansible_controller_keypair

# 旧离线包内 init role 曾用 lookup('file')；若仍存在则必须换用含 slurp 的新 ansible-agent
if grep -q "lookup('file'" "$ANSIBLE_DIR/roles/init/tasks/main.yml" 2>/dev/null; then
  echo "ERROR: 当前解压目录里的 ansible-agent 过旧（roles/init 仍含 lookup('file')）。"
  echo "请从 OpsFleet 控制台重新「生成并下载离线安装包」，或 git 拉取最新仓库后重新解压再执行。"
  exit 1
fi

preflight_ssh_roots() {
  if [[ ! -f "$INV" ]]; then
    echo "ERROR: 缺少 $INV"
    exit 1
  fi
  local ips
  ips=$(grep -oE 'ansible_host=[0-9.]+' "$INV" 2>/dev/null | cut -d= -f2 | sort -u) || true
  if [[ -z "${ips// }" ]]; then
    echo "WARN: 未从 inventory 解析到节点 IP，跳过 SSH 预检"
    return 0
  fi
  echo "=== SSH 预检：本机须能免密登录 root@各节点（与 inventory 中 IP 一致）==="
  local ip failed=0
  for ip in $ips; do
    if ssh -o BatchMode=yes -o StrictHostKeyChecking=no -o ConnectTimeout=12 "root@${ip}" "true" 2>/dev/null; then
      echo "  OK  root@${ip}"
    else
      echo "  FAIL root@${ip} — 常见原因: 未配置免密。请先在同一台机器执行:"
      echo "       ssh-copy-id -i ~/.ssh/id_ed25519.pub root@${ip}   # 或你的公钥路径"
      echo "       再试: ssh root@${ip}"
      failed=1
    fi
  done
  if [[ "$failed" -ne 0 ]]; then
    echo "ERROR: SSH 预检未通过，已中止。修正后请重新运行: sudo bash install.sh"
    exit 1
  fi
}

preflight_ssh_roots

# 与 inventory/group_vars 中 local_cache_dir 一致，跨多次测试复用已下载 tarball。
mkdir -p /var/cache/opsfleet-k8s
chmod 0755 /var/cache/opsfleet-k8s

cd "$ANSIBLE_DIR"

run() {
  local step="$1"
  local pb="$2"
  echo "=== ${step} ==="
  ansible-playbook -i "$INV" "$pb" || { echo "FAILED at ${pb}"; exit 1; }
  echo "${step}" >> "$STATE_FILE"
}

run "Step 1/7: init" "playbooks/0-init.yml"
run "Step 2/7: resources" "playbooks/resources.yml"
run "Step 3/7: etcd" "playbooks/etcd.yml"
run "Step 4/7: kube-apiserver" "playbooks/kube_apiserver_install.yml"
run "Step 5/7: kube-controller-manager" "playbooks/kube_controller_manager_install.yml"
run "Step 6/7: kube-scheduler" "playbooks/kube_scheduler_install.yml"
run "Step 7/7: kubectl" "playbooks/kubectl.yml"

echo "=== 完成 ==="
kubectl get nodes 2>/dev/null || echo "请登录各节点检查 kubelet / 网络插件（Calico 等）是否已按 group_vars 后续部署"
`
}
