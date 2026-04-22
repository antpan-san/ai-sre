---
name: k8s-offline-deploy-test
description: >-
  Kubernetes offline-bundle deployment testing with human-parity (zip from UI or gen-k8s-bundle),
  mandatory checkpoints, cleanup, push preceded by monorepo-release. Mandatory adjunct when changing
  ansible-agent K8s, k8s_bundle, k8s_ansible, K8s deploy UI, gen-k8s-bundle, deploy/k8s-mirror, or k8s_mirror_catalog per .cursor/rules/monorepo-release.mdc.
  When extending the deploy pipeline, update this skill in the same PR (step table, templates, verification).
  Lab VM root@192.168.56.11: optional manifest curl + align with opsfleetpilot-ship deploy.
---

# Kubernetes 离线部署测试（人手等价）

## 与仓库强制工作流的关系

- **总入口**：**`.cursor/skills/release-deploy/SKILL.md`**。
- **`.cursor/rules/monorepo-release.mdc`**（`alwaysApply: true`）要求：凡改动命中 **ansible-agent K8s、`k8s_bundle`、`k8s_ansible`、K8s 部署前端、`gen-k8s-bundle`** 等，在 **`git push` 之前**须完成本 Skill 中与本次变更匹配的步骤（至少 **构建 + 打 zip**；有 SSH 则完整装测）。
- 与 **`.cursor/skills/ai-sre-ship/SKILL.md`**、**`opsfleetpilot-ship`** 叠加时，顺序见 `monorepo-release.mdc`。

---

## 实验室虚拟机 `root@192.168.56.11`（本地 root，与部署 Skill 共用）

本机 VirtualBox/VMware 等内网机 **`192.168.56.11`** 在仓库工作流中承担多角色（**同一台机、同一 root 免密 SSH**）：

| 用途 | 路径或说明 |
|------|------------|
| ai-sre CLI 同步 | `scripts/deploy-remote.sh` → `/root/sre` |
| OpsFleet 全栈 | `scripts/deploy-opsfleet-remote.sh` → 同上目录 + Nginx + systemd |
| K8s 内网制品（可选） | `deploy/k8s-mirror/` → 持久目录默认 `/var/lib/opsfleet-k8s-mirror`，HTTP 提供 `manifest.json` |
| 后端读 manifest | `/etc/opsfleet/backend.env` 中 `OPSFLEET_K8S_MIRROR_BASE_URL`（首次 `deploy-opsfleet-remote.sh` 可自动生成） |

**改 `deploy/k8s-mirror/`、`k8s_mirror_catalog`、或 `download_domain` 相关逻辑时**，在能 SSH 到该 VM 的前提下，**建议**补做：

```bash
ssh -o BatchMode=yes root@192.168.56.11 'curl -sfS --connect-timeout 8 http://127.0.0.1/manifest.json | head -c 200 || echo "manifest 未部署: 见 deploy/k8s-mirror/README.md"'
```

不可达时在测试反馈中注明 **网络假设**。完整发布仍须 **`opsfleetpilot-ship`**（`deploy-opsfleet-remote.sh` + 健康检查）与 **`git push`**，顺序见 `monorepo-release.mdc`。

---

## 随项目演进：维护本 Skill（改流水线必做）

以下任一处变更时，**同一 MR/提交**内应同步更新本文件（至少对照表 + 测试结果模板 + Known issues）：

| 改动类型 | 请更新 Skill 中… |
|----------|------------------|
| 增删/重排 `install.sh` 步骤 | **「离线 `install.sh` 步骤（权威）」**表；**测试结果模板**里的 Step 范围 |
| 改 `generateK8sDeployScript`（在线 Ansible） | 说明在线与离线步骤应对齐；必要时在表内加一列「在线脚本」 |
| 新增 playbook、角色（如 CNI/存储） | 表内步骤名、**分类速查表**、**装通后验证**命令 |
| 改合并 `group_vars` 规则或键名 | **强制检查点**里的 `grep` 示例 |
| 改清理行为 | **Cleanup** 与 `scripts/k8s-offline-test-cleanup.sh`、`playbooks/pre_cleanup.yml` 三者说明一致 |
| 改 `deploy/k8s-mirror/`、manifest 格式、`k8s_mirror_catalog` | **实验室 VM** 小节、`verify-opsfleet-deployment.sh` 中 manifest 检查说明 |

**单一事实来源（改步骤时先改代码再改文档）：**

- 离线：`ft-backend/handlers/k8s_bundle.go` → `renderOfflineInstallScript()` 内 `run "Step …"`。
- 在线：`ft-backend/handlers/k8s_ansible.go` → `generateK8sDeployScript()` 内 `Step …` 与 `ansible-playbook` 调用。

---

## 核心原则：Skill 不得比真人「多一条捷径」

真实用户只会拿到 **OpsFleet 生成的 zip**，解压后得到：

`install.sh` + **`inventory/`（含合并后的 `group_vars/all.yml`）** + `ansible-agent/` + `README.txt`。

若 Agent **只拷贝 `ansible-agent`** 到机器上跑 playbook、或用了**未合并**的 inventory，会得到 **内置默认** 的 `download_domain`（见 `ansible-agent/inventory/group_vars/all.yml`）、`/tmp/ansible-cache` 等，与 **控制台/CLI 打的离线包** 不一致——**这是 Skill「正常」而人工执行失败的首要原因**。  
**本 Skill 要求：默认流程必须与「下载 zip → 解压 → install.sh」等价，禁止该捷径。**

---

## Agent 禁止项（违反则不得声明「与生产验证等价」）

| 禁止 | 原因 |
|------|------|
| 仅在目标机使用**裸 `ansible-agent` 仓库**跑 playbooks | 缺少解压目录下 **已合并** 的 `inventory/group_vars/all.yml`，变量回退到 agent 内默认值 |
| 声称通过但未核对 **zip 内** `inventory/group_vars/all.yml` 与所选镜像源/架构 | 无法用内网源却用公网打出 PASS |
| 忽略用户网络：在 **有公网的 Agent 环境** 测通，却代表 **仅内网** 用户 | 须在反馈中标明 **网络假设** |
| 用陈旧分支打的包却对比最新代码行为 | git SHA 与产物必须一致 |
| 步骤数仍写「1–7」而代码已是 **11 步** | 与 `install.sh` 不一致，结论无效 |

---

## 离线 `install.sh` 步骤（权威，与当前代码一致）

可选 **Step 0**：`OPSFLEET_OFFLINE_PRE_CLEANUP=1`（或打包时勾选预清理）→ `playbooks/pre_cleanup.yml`。

| Step | 名称 | Playbook |
|------|------|----------|
| 1/11 | init | `playbooks/0-init.yml` |
| 2/11 | resources | `playbooks/resources.yml` |
| 3/11 | etcd | `playbooks/etcd.yml` |
| 4/11 | kube-apiserver | `playbooks/kube_apiserver_install.yml` |
| 5/11 | kube-controller-manager | `playbooks/kube_controller_manager_install.yml` |
| 6/11 | kube-scheduler | `playbooks/kube_scheduler_install.yml` |
| 7/11 | kubectl | `playbooks/kubectl.yml` |
| 8/11 | containerd | `playbooks/containerd.yml` |
| 9/11 | kubelet | `playbooks/kubelet.yml` |
| 10/11 | kube-proxy | `playbooks/kube_proxy.yml` |
| 11/11 | addons（Flannel + CoreDNS，受 `network_plugin` 等约束） | `playbooks/k8s_addons.yml` |

**在线部署**（Agent 执行机上的脚本）应与上表 **同序、同 playbook**；若发现漂移，按 **需改代码** 处理。

---

## 两条合法入口（二选一，与人工对齐）

### 入口 A（首选，与最终用户一致）

1. 浏览器登录 **OpsFleet** → K8s 部署向导填：**集群名、版本、CPU 架构、镜像源、节点 IP**。  
2. **生成并下载 zip**（走 `BuildK8sOfflineZip` / 与后端相同逻辑）。  
3. `scp` 到控制机 → **解压到独立空目录** → `sudo bash install.sh`。

### 入口 B（开发/CI，须与 A 数学等价）

```bash
cd /path/to/ft-backend
OPSFLEET_ANSIBLE_DIR=/path/to/repo/ansible-agent \
  go run ./cmd/gen-k8s-bundle -o /tmp/k8s.zip \
  -cluster <NAME> -version <vX.Y.Z> \
  -master '<IP>' -worker '<IP>' \
  -imageSource <aliyun|default|...> -arch <amd64|arm64>
```

`gen-k8s-bundle` 与 HTTP 接口共用 **`BuildK8sOfflineZip`**；**同一组参数** 打出的包应与 UI **一致**（若不一致则属产品 Bug，应记「需改代码」）。

### 入口 C：实验室双节点（本仓库 `scripts/k8s-lab-full-install.sh`）

在**本机有 Go、能 `ssh` 到控制机**的前提下，**一键**完成：**打 zip → scp 到控制机 → 空目录解压 → `install.sh`（可配置失败重试）**，与入口 A/B **同一套 zip 内容**。

- **默认约定（可按环境变量改）**  
  - 集群名 **`111111`**，master **`192.168.56.101`**，worker **`192.168.56.102`**。  
  - 控制机（跑 `install.sh` 的机器、须对 inventory 内各节点 `root` 免密）**默认** `K8S_LAB_SSH=root@192.168.56.101`（与 master 同机是常见跑法；若你的控制机是 `192.168.56.11` 等，请设 `K8S_LAB_SSH`）。  
  - 架构 **默认 `arm64`**（与 `gen-k8s-bundle` 一致；x86 实验室请设 **`K8S_LAB_ARCH=amd64`**，与节点 `uname -m` 一致）。  
  - 版本默认 **`v1.28.15`**，与 CLI `gen-k8s-bundle` 一致。

- **执行**

```bash
cd /path/to/ai-sre
./scripts/k8s-lab-full-install.sh
# 或（示例：控制机为 11，多 worker 逗号分隔）
# K8S_LAB_SSH=root@192.168.56.11 K8S_LAB_MASTERS=192.168.56.101 K8S_LAB_WORKERS=192.168.56.102,192.168.56.103 ./scripts/k8s-lab-full-install.sh
```

- **与「不要一步一坎」的对应**  
  - 仍须满足 **网络 + 免密 + 架构** 等 Preconditions；脚本负责 **固定参数、固定 OPSFLEET_ANSIBLE_DIR、可重复解压目录、可重试 install、统一日志**（控制机上 `~/opsfleet-k8s-lab-install.log`）。  
  - 装通后验证见下文 **装通后验证**；未改 skill 的禁止项：禁止仅在目标机**裸跑 ansible-agent 仓库**替代 zip。

---

## 强制检查点（解压后、执行 install.sh 之前，人手也会做／Agent 必须做）

在**控制机**上进入解压目录，执行或核对：

```bash
# 1) 必须使用解压根下的 inventory，而非 ansible-agent 自带的默认 inventory
test -f inventory/group_vars/all.yml && test -f inventory/hosts.ini || { echo FAIL; exit 1; }

# 2) 确认缓存目录与镜像行为（当前仓库：持久目录应为 /var/cache/opsfleet-k8s，而非 /tmp/ansible-cache）
grep -E "local_cache_dir|image_source|arch_version|k8s_server_tarball_url|download_domain|pod_cluster_cidr|dns_service_ip|network_plugin" inventory/group_vars/all.yml | head -50

# 3) 若选「阿里云」镜像源：应出现 dl.k8s.io 的 tarball/sha512 覆盖；若仍是纯内网 IP 且无合并块，说明包不对
```

**网络预检（与镜像源一致）：**

- `imageSource: aliyun`（公网）：在控制机上 `curl -sI --connect-timeout 10 https://dl.k8s.io/` 或即将使用的版本 URL 应可达。  
- `imageSource: default`（内网 mirror）：`curl -sI --connect-timeout 10 http://<download_domain>/` 必须可达，否则与人工在同一机房的结果一致：**会超时失败**——反馈中标 **非代码（网络）**，不得标 Skill 逻辑错误。

---

## 标准步骤序列（A–H，含人手等价说明）

| 步骤 | 人手做什么 | Agent 必须同样做 |
|------|------------|------------------|
| A | 确认分支、参数与目标机 `uname -m` | 同左；记录 git SHA |
| B | UI 下载 zip **或** CLI `gen-k8s-bundle` | **禁止**只同步 ansible-agent 代替 zip |
| C | scp + unzip 到**空目录** | 同左 |
| **C.1** | 打开 `inventory/group_vars/all.yml` 看镜像与缓存 | **强制检查点**（上一节命令） |
| D | `sudo bash install.sh \| tee log` | 同左；保存完整日志 |
| E | 复测时一般不删 `/var/cache/opsfleet-k8s` | 同左 |
| F | 写测试反馈与问题分类 | 使用下方 **测试结果模板**（含 Step 1–11） |
| G | 停服务、删解压目录、按需删缓存 | `scripts/k8s-offline-test-cleanup.sh` |
| H | 回流 issue/文档 | 同左；若改了流水线则更新 **本 Skill** |

---

## 装通后验证（建议写入测试反馈）

在**装有 `kubectl` 与 `admin.conf` 的控制平面节点**（或与 API 可达的机器）：

```bash
export KUBECONFIG=/etc/kubernetes/admin.conf   # 与当前 ansible 角色路径一致
kubectl get nodes -o wide
kubectl get pods -A
kubectl -n kube-system get svc kube-dns
```

- **节点 NotReady**：常见为 CNI 未就绪、`network_plugin` 与已下发清单不一致、或镜像拉取失败（标 **非代码** 或 **需改代码** 视日志）。  
- **CoreDNS Pending**：常为网络插件或 kube-proxy 未就绪。

---

## Preconditions

| 项 | 要求 |
|----|------|
| 控制机 SSH | 对 inventory 内所有节点 `root` 免密 |
| 架构 | 与 UI/`-arch` 一致（`uname -m`） |
| 网络 | 与所选镜像源一致；**内网-only 勿选依赖公网的 aliyun/dl.k8s 链** |
| 构建 | `OPSFLEET_ANSIBLE_DIR` 指向当前仓库 `ansible-agent`（仅 **B 入口**需要） |

---

## 持久化缓存

- `local_cache_dir`：`/var/cache/opsfleet-k8s`（与旧文档 `/tmp/ansible-cache` **不同**——若日志仍是后者，说明 **未使用当前合并包或旧代码**）。
- K8s：`get_url` + sha512，命中则跳过下载。

---

## 测试结果模板（必填）

```markdown
## K8s 离线部署测试反馈

### 元数据
- **入口**: UI | CLI-gen-k8s-bundle
- **时间** / **git SHA**:
- **参数**: cluster, version, arch, imageSource, nodes, preDeployCleanup (Y/N)

### 人手等价声明
- **解压后已检查** `inventory/group_vars/all.yml`：是 / 否（若否，本次结论不得写「与用户场景等价」）
- **网络假设**: 控制机能否访问 dl.k8s.io / GitHub / 内网 mirror：

### 总结果
- Step 0（预清理，可选）: PASS | SKIP | FAIL
- Step 1–11（见 Skill 内「离线 install.sh 步骤」表）: PASS | FAIL
- **若 FAIL**：失败于 Step __ / playbook ____________

### 验证命令（建议粘贴输出摘要）
- `kubectl get nodes` :
- `kubectl get pods -A` :

### 失败点与分类（需改代码 / 非代码 / 待确认）


### 缓存与清理
- Cleanup: done / skipped（脚本路径与参数）
```

---

## 分类速查表

| 现象 | 倾向 | 说明 |
|------|------|------|
| 合并包内 `download_domain` 与机房不符 / `/tmp/ansible-cache` 且未选内网部署 | **流程错误** | 未使用合并 zip 或旧包；回到 **入口 A/B + 检查点 C.1** |
| `urlopen timed out` 向内网 IP | **非代码** | 与用户机房网络一致；换 mirror 或修网络 |
| `duplicate mapping key` | 一般不阻塞 | 合并策略可优化属「需代码」 |
| `Exec format error` | 非代码优先 | arch 与 `uname -m` |
| controller-manager `10257/healthz` | 已加重试 | 仍失败查 `journalctl` |
| containerd / kubelet 启动失败 | 需查 journal | cgroup、CRI socket、`/opt/cni/bin` |
| Flannel / CoreDNS 镜像拉取失败 | 多为 **非代码** | 内网需镜像仓库或预拉取 |
| `network_plugin: calico` 但仅实现 Flannel | **待确认/需代码** | 见 `group_vars` 与 `k8s_addons` 行为 |

---

## Cleanup

- 仓库脚本：**`scripts/k8s-offline-test-cleanup.sh`**（可选 `--purge-cache`、`--deep`）。  
- 与 **`playbooks/pre_cleanup.yml`** 的关系：预清理在 **install 前**、面向重复部署；本脚本面向 **测试后** 收实验室，二者服务单元列表应随组件演进**偶尔对齐**（改 playbook 预清理时检查脚本 stop 列表）。

---

## Known issues

- **仅内网**：镜像选「默认」并配 `download_domain`；不要用公网专用链。  
- **group_vars 加载**：多数 play 使用 `inventory_dir` + `include_vars` 或 inventory 与 `group_vars` 同目录；**禁止**在未合并包上依赖 ansible-agent 内置默认值。  
- **与人手不一的常见根因**：只跑 ansible-agent、未用 zip 内 **合并后的** `group_vars`。  
- **kube-controller-manager**：已对 `10257/healthz` 做轮询重试。  
- **CNI 与 `network_plugin`**：当前自动 addons 以 **Flannel + CoreDNS** 为主；非 flannel 需自行安装对应 CNI 或扩展 playbook。
