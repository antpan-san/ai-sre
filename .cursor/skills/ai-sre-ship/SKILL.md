---
name: ai-sre-ship
description: >-
  Mandatory after ANY change under /Users/panshuai/Documents/work/code/ai-sre (entire monorepo):
  README check, deploy-remote.sh to root@172.16.195.128:/root/sre, remote smoke, README re-check, push.
  Also see .cursor/rules/monorepo-release.mdc and opsfleetpilot-ship when touching ft-*/deploy/ansible.
---

# ai-sre 发布与同步（强制工作流）

## 触发条件（必须）

在 **`/Users/panshuai/Documents/work/code/ai-sre` 仓库内对任意路径、任意类型文件的修改**（代码、配置、脚本、文档、`.cursor` 规则与 skill 等）完成后，代理**必须**按本文件执行发布流程，**不要**等待用户再说「发布」。若用户**明确**声明豁免（例如仅本地试验、禁止 SSH），则按其声明缩小步骤，但仍须完成可执行的 README 与自检。

仓库同仓参数与 **Cursor 规则**见 **`.cursor/rules/monorepo-release.mdc`**（`alwaysApply`）；本 skill 为具体步骤。

在 **ai-sre** 项目（`/Users/panshuai/Documents/work/code/ai-sre`）中**每次完成上述修改后**，代理必须按顺序执行本流程；用户已配置 **root@172.16.195.128** 免密 SSH。

## 固定参数

| 项 | 值 |
|----|-----|
| 远程主机 | `root@172.16.195.128` |
| 远程目录 | `/root/sre`（不存在则创建） |
| GitHub 远程 | `git@github.com:antpan-san/ai-sre.git` |
| 本地项目根 | `/Users/panshuai/Documents/work/code/ai-sre` |
| 产品使用文档 | 仓库根目录 **`README.md`**（唯一权威用户手册，须与代码同步） |

**同仓说明**：根目录另含 **OpsFleetPilot**（`ft-backend/`、`ft-front/` 等）。本工作流 **`scripts/deploy-remote.sh`** 仅同步并构建 **ai-sre CLI**；若变更 OpsFleet 全栈部署，使用 **`scripts/deploy-opsfleet-remote.sh`**（见 **`.cursor/skills/opsfleetpilot-ship/SKILL.md`**）。

---

## README 维护（强制）

### Push 代码前（必须）

在 **`git commit` / `git push` 之前**，必须**完善或核对** `README.md` 中的**产品使用**相关内容，确保与当前代码一致，包括但不限于：

- [ ] **子命令与用途**：`analyze` / `ask` / `runbook` / `skills list` / `doctor` / `version`
- [ ] **全局参数**：`--config`、`--key-file`、`-v`、`--no-rag`、`-o`/`--output`、`--skills-dir`、`--knowledge-dir`
- [ ] **典型示例**：与产品文档 CLI 示例一致或可覆盖之
- [ ] **配置方式**：密钥文件路径、`config.yaml` / `api_key` 格式（不含真实密钥）
- [ ] **结构化输出**：`-o json` 的用途说明（若有变更须更新）
- [ ] **远程部署与冒烟**：`./scripts/deploy-remote.sh`、`DEPLOY_REMOTE`、`DEPLOY_REMOTE_DIR`、`scripts/remote-e2e.sh`（含 `go test`、`doctor`）
- [ ] **免费版 / 配额**：`tier`、`max_llm_calls_per_day`、`doctor`、缓存路径说明是否与 `internal/config`、`internal/quota` 一致
- [ ] **当前版本号**：与 `internal/cli` 中 `version` 子命令输出一致（或注明以 `./ai-sre version` 为准）

若本次变更**影响用户可见行为**（新 flag、新子命令、配置路径、脚本行为），**必须**在本次提交中**同步更新 README**。仅内部重构时，至少快速通读 README，修正已过时表述。

### 每次发布后（必须）

在 **`./scripts/deploy-remote.sh` 成功且远程冒烟通过之后**，**再次核对** `README.md`：

- 远程路径、脚本名称、环境变量是否与文档一致；
- `./ai-sre version` 与 README 中的版本说明是否一致；
- 若发布过程中发现文档与线上行为不符，**立即修订 README**，并与代码变更**一并提交**（或紧随其后的文档提交），再执行 `git push`。

**禁止**：在 README 明显过时的情况下直接 push；禁止将仅含代码、不含 README 核对说明的发布标记为完成。

---

## 执行顺序（必须完整）

### 1. 更新 README（Push 前）

编辑 `README.md`，完成上文 **Push 代码前** 检查清单。将 `README.md` 纳入本次提交的暂存区。

### 2. 部署到远程

在本地项目根执行：

```bash
./scripts/deploy-remote.sh
```

脚本行为：确保远程目录存在、`rsync` 同步源码（排除 `.git` 与二进制）、远程 `go mod download`、`go vet ./...`、`go build`、运行 `./ai-sre version`。

若脚本不存在，则等价手动执行（见下文「手动等价命令」）。

**不要使用 `--delete` rsync**（除非用户明确要求），以免误删服务器上额外文件。

### 3. 远程功能测试（必须通过）

- **必测**：远程构建成功、`./ai-sre version` 正常；建议执行 `SHORT=1 bash scripts/remote-e2e.sh`（vet/build/version/skills list/无凭证负例）。
- **可选（若远程已配置密钥文件）**：`bash scripts/remote-e2e.sh` 全量 LLM 联调。

若任一步失败：**停止推送 GitHub**，先修复代码并重复步骤 1–3。

### 4. 发布后 README 复核

按上文 **每次发布后** 小节核对并必要时修订 `README.md`。

### 5. 推送到 GitHub

在**本地**项目根：

```bash
cd /Users/panshuai/Documents/work/code/ai-sre
git status
git add -A
git commit -m "feat: <简明说明，若含文档可写 docs: 更新 README>"
git push -u origin main
```

- 若尚未初始化仓库：`git init`，`git remote add origin git@github.com:antpan-san/ai-sre.git`，首次推送 `git push -u origin main`。

**注意**：若步骤 4 修改了 README，须**再次** `git add README.md && git commit --amend` 或**追加一次 commit** 后再 `git push`。

### 6. 向用户汇报

说明：README 是否已更新、远程部署路径、测试结果、Git 提交哈希或「已 push」。

---

## 手动等价命令（无 deploy-remote.sh 时）

```bash
ssh root@172.16.195.128 "mkdir -p /root/sre"
rsync -avz \
  --exclude '.git' \
  --exclude 'ai-sre' \
  --exclude '.DS_Store' \
  /Users/panshuai/Documents/work/code/ai-sre/ \
  root@172.16.195.128:/root/sre/
ssh root@172.16.195.128 'cd /root/sre && go mod download && go vet ./... && go build -o ai-sre . && ./ai-sre version'
```

---

## 远程环境前提

- 目标机需可执行 `go`（Ubuntu 示例：`apt-get install -y golang-go`）。
- 本机已配置到 `root@172.16.195.128` 的免密 SSH。

## 失败处理

- **SSH/rsync 失败**：检查网络与连通性。
- **远程 go build 失败**：本地复现 `go build` 并修复。
- **git push 失败**：按错误处理；不强行覆盖远程。

## 与用户提示的关系

只要本轮对话中**改动了本仓库（`/Users/panshuai/Documents/work/code/ai-sre`）内任意文件**，在完成修改后**自动应用本 skill**（含 README 维护、`deploy-remote.sh`、发布后复核与 push），除非用户明确说「仅本地、不要部署/不要改 README/不要 push」。

若变更同时涉及 OpsFleet 路径，**还须**执行 **`.cursor/skills/opsfleetpilot-ship/SKILL.md`**（见 **`.cursor/rules/monorepo-release.mdc`**）。

若变更涉及 **Kubernetes 离线包 / ansible-agent K8s / `k8s_bundle` / 控制台 K8s 部署页**，**还须**执行 **`.cursor/skills/k8s-offline-deploy-test/SKILL.md`**（最低限度：`ft-backend` 可构建、`gen-k8s-bundle` 可打 zip；有测试机则完整 `install.sh`）。详见 **`.cursor/rules/monorepo-release.mdc`** 第 3 条。
