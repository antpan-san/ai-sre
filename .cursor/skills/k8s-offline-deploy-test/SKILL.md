---
name: k8s-offline-deploy-test
description: >-
  Kubernetes offline-bundle deployment testing with human-parity (zip from UI or gen-k8s-bundle),
  mandatory checkpoints, cleanup, push preceded by monorepo-release. Mandatory adjunct when changing
  ansible-agent K8s, k8s_bundle, k8s_ansible, K8s deploy UI, or gen-k8s-bundle per .cursor/rules/monorepo-release.mdc.
---

# Kubernetes 离线部署测试（人手等价）

## 与仓库强制工作流的关系

- **`.cursor/rules/monorepo-release.mdc`**（`alwaysApply: true`）要求：凡改动命中 **ansible-agent K8s、`k8s_bundle`、`k8s_ansible`、K8s 部署前端、`gen-k8s-bundle`** 等，在 **`git push` 之前**须完成本 Skill 中与本次变更匹配的步骤（至少 **构建 + 打 zip**；有 SSH 则完整装测）。
- 与 **`.cursor/skills/ai-sre-ship/SKILL.md`**、**`opsfleetpilot-ship`** 叠加时，顺序见 `monorepo-release.mdc`。

## 核心原则：Skill 不得比真人「多一条捷径」

真实用户只会拿到 **OpsFleet 生成的 zip**，解压后得到：

`install.sh` + **`inventory/`（含合并后的 `group_vars/all.yml`）** + `ansible-agent/` + `README.txt`。

若 Agent **只拷贝 `ansible-agent`** 到机器上跑 playbook、或用了**未合并**的 inventory，会得到 **内置默认** 的 `10.10.120.144`、`/tmp/ansible-cache` 等，与 **控制台/CLI 打的离线包** 不一致——**这是 Skill「正常」而人工执行失败的首要原因**。  
**本 Skill 要求：默认流程必须与「下载 zip → 解压 → install.sh」等价，禁止该捷径。**

---

## Agent 禁止项（违反则不得声明「与生产验证等价」）

| 禁止 | 原因 |
|------|------|
| 仅在目标机使用**裸 `ansible-agent` 仓库**跑 playbooks | 缺少解压目录下 **已合并** 的 `inventory/group_vars/all.yml`，变量回退到 agent 内默认值 |
| 声称通过但未核对 **zip 内** `inventory/group_vars/all.yml` 与所选镜像源/架构 | 无法用内网源却用公网打出 PASS |
| 忽略用户网络：在 **有公网的 Agent 环境** 测通，却代表 **仅内网** 用户 | 须在反馈中标明 **网络假设** |
| 用陈旧分支打的包却对比最新代码行为 | git SHA 与产物必须一致 |

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

---

## 强制检查点（解压后、执行 install.sh 之前，人手也会做／Agent 必须做）

在**控制机**上进入解压目录，执行或核对：

```bash
# 1) 必须使用解压根下的 inventory，而非 ansible-agent 自带的默认 inventory
test -f inventory/group_vars/all.yml && test -f inventory/hosts.ini || { echo FAIL; exit 1; }

# 2) 确认缓存目录与镜像行为（当前仓库：持久目录应为 /var/cache/opsfleet-k8s，而非 /tmp/ansible-cache）
grep -E "local_cache_dir|image_source|arch_version|k8s_server_tarball_url|download_domain" inventory/group_vars/all.yml | head -40

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
| F | 写测试反馈与问题分类 | 同左 |
| G | 停服务、删解压目录、按需删缓存 | `scripts/k8s-offline-test-cleanup.sh` |
| H | 回流 issue/文档 | 同左 |

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
- **时间** / **git**:
- **参数**: cluster, version, arch, imageSource, nodes

### 人手等价声明
- **解压后已检查** `inventory/group_vars/all.yml`：是 / 否（若否，本次结论不得写「与用户场景等价」）
- **网络假设**: 控制机能否访问 dl.k8s.io / GitHub / 内网 mirror：

### 总结果
- Step 1–7: PASS | FAIL

### 失败点与分类（需改代码 / 非代码 / 待确认）
（同前版结构）

### 缓存与清理
- Cleanup: done / skipped
```

---

## 分类速查表

| 现象 | 倾向 | 说明 |
|------|------|------|
| `10.10.120.144` / `/tmp/ansible-cache` 且未选内网部署 | **流程错误** | 未使用合并 zip 或旧包；回到 **入口 A/B + 检查点 C.1** |
| `urlopen timed out` 向内网 IP | **非代码** | 与用户机房网络一致；换 mirror 或修网络 |
| `duplicate mapping key` | 一般不阻塞 | 合并策略可优化属「需代码」 |
| `Exec format error` | 非代码优先 | arch 与 `uname -m` |
| controller-manager health | 已加重试 | 仍失败查 `journalctl` |

---

## Cleanup

见仓库 **`scripts/k8s-offline-test-cleanup.sh`**（默认不深删 `/etc/kubernetes`）。

---

## Known issues

- **仅内网**：镜像选「默认」并配 `download_domain`；不要用公网专用链。  
- **`vars_files`**：`playbooks/resources.yml` 须指向 `../../inventory/group_vars/all.yml`。  
- **与人手不一的常见根因**：只跑 ansible-agent、未用 zip 内 **合并后的** `group_vars`。  
- **kube-controller-manager**：已对 `10257/healthz` 做轮询重试。
