# ai-sre（AI SRE Copilot）

Go 实现的 CLI：**技能包（Skill Pack）+ Prompt 组装 + 可选轻量 RAG + DeepSeek LLM**，对应产品文档中的三条核心能力：

1. **故障诊断** — `analyze`
2. **Runbook 生成** — `runbook`
3. **知识库问答** — `ask`（检索本地 Markdown 片段注入 Prompt）

另：**技能注册** — `skills list`（发现已加载技能包）。

**当前版本**：以运行环境为准，执行 `./ai-sre version`（与源码中 `internal/cli/version.go` 的 `Version` 变量对齐）。**自升级**：若存在 **`OPSFLEET_API_URL`**、或 `curl` 安装脚本写入的 **`~/.config/ai-sre/opsfleet_api_url`**、或 **`config.yaml` 的 `opsfleet_api_url` 字段**（如 `http://<host>:9080/ft-api`），**每次**执行子命令前都会拉取 `GET .../api/k8s/deploy/cli/ai-sre/version` 比对，较新则自动下载并覆盖后 **重新执行同一命令**（非 Windows）。关闭：`OPSFLEET_NO_AUTO_UPGRADE=1`；关自动后仅提示：`OPSFLEET_UPGRADE_HINT=1`。**显式一次升级**：`sudo ai-sre upgrade -y`（同上基址来源）。

本仓库为 **单一 Git 仓库**：根目录 **CLI（ai-sre）**、**OpsFleet 本地执行器（`opsfleet-executor`）** 与 **OpsFleetPilot（Web + API）** 并排共存。**OpsFleetPilot** 包含 `ft-backend/`、`ft-front/`、`deploy/`、`ansible-agent/`；**`opsfleet-executor`** 与 `ai-sre` **共用同一套技能包与执行语义**（`analyze` / `ask` / `runbook` 等），用于部署在**受管机**上本地执行。产品总览见 [`PRODUCT_DOC.md`](PRODUCT_DOC.md)，历史说明见 [`docs/opsfleet-README.md`](docs/opsfleet-README.md)。控制台构建：`make build-opsfleet`（产物 `bin/opsfleet-backend`、`dist/web/`）。

---

## 子命令一览

| 命令 | 说明 |
|------|------|
| `ai-sre analyze [topic]` | 故障诊断，`topic`：`kafka` \| `k8s` \| `nginx` \| `redis` |
| `ai-sre ask [question]` | 知识问答（可选 RAG） |
| `ai-sre runbook [scenario]` | 生成 Runbook |
| `ai-sre skills list` | 列出内置 + `--skills-dir` 合并后的技能包 |
| `ai-sre doctor` | 自检（凭据、tier、配额计数、技能/知识加载；**不调用 LLM**） |
| `ai-sre version` | 打印版本号 |
| `ai-sre help` | 帮助 |
| `ai-sre k8s …` | 离线包下载、控制机 `install` / `cleanup` 等（见 `ai-sre k8s --help`） |
| `ai-sre upgrade` | 与 OpsFleet 对比版本后覆盖本机 `ai-sre` 二进制（需能访问上表基址） |
| `ai-sre uninstall k8s` | 在控制机 `root` 下按已记录的安装引用全量清理（与 `k8s cleanup` 同路径，无需再填 workdir） |

别名：`ops-ai`（与 `ai-sre` 等价）。

---

## 全局参数（根命令）

| 参数 | 说明 |
|------|------|
| `--config` | 凭据 YAML 路径（默认 `~/.config/ai-sre/config.yaml`） |
| `--key-file` | 仅含 API Key 的文件路径 |
| `-v` / `--verbose` | 打印凭据文件路径、技能数、知识片段数等 |
| `--no-rag` | 关闭知识库检索 |
| `-o` / `--output` | 输出格式：`text`（默认）或 `json`（适用于 `analyze` / `ask` / `runbook`） |
| `--skills-dir` | 额外技能包目录（`*.yaml`，与内置合并；同名覆盖） |
| `--knowledge-dir` | 额外知识库目录（`*.md`，与内置合并参与 RAG） |

`analyze` 常用 flag：`--lag`、`--topic`、`--pod`、`--namespace`、`--issue`、`--code`、`--upstream`、`--latency`、`-d`/`--set key=value`。

---

## 配置（LLM 凭据，来自文件；OpsFleet 自升级可附加文件/环境）

在运行机器上放置密钥，**二选一**即可：

**方式 A — YAML（推荐）**  
默认路径：`~/.config/ai-sre/config.yaml`（若设置 `XDG_CONFIG_HOME`，则为 `$XDG_CONFIG_HOME/ai-sre/config.yaml`）

```yaml
api_key: "你的 DeepSeek API Key"
# 可选
base_url: "https://api.deepseek.com/v1"
model: "deepseek-chat"

# --- 前期「变现 / 免费版」MVP（产品文档：限制技能与调用次数）---
# tier: free          # 设为 free 时忽略 --skills-dir / --knowledge-dir，仅使用内置技能与知识
# max_llm_calls_per_day: 20   # 每日 LLM 调用上限；0 或不写表示不限制。计数文件在 ~/.cache/ai-sre/llm_usage.json
# 可选，供自升级/默认与控制台通信（同 OPSFLEET_API_URL 语义，含 /ft-api）
# opsfleet_api_url: "http://<host>:9080/ft-api"
```

若仅使用 **`api_key` 纯文件** 存密钥，仍可在同目录增加 **`config.yaml`**（可只含 `tier` / `max_llm_calls_per_day`，不含 `api_key`），程序会自动合并限额配置。

**方式 B — 仅密钥文件**  
默认路径：`~/.config/ai-sre/api_key`（纯文本，第一行为密钥；`#` 开头行为注释）

命令行覆盖：`--config`、`--key-file`。

```bash
mkdir -p ~/.config/ai-sre
chmod 700 ~/.config/ai-sre
printf '%s\n' '你的密钥' > ~/.config/ai-sre/api_key
chmod 600 ~/.config/ai-sre/api_key
```

---

## 构建与常用示例

```bash
go build -o ai-sre .

./ai-sre analyze kafka --lag 100000
./ai-sre analyze k8s --pod pending
./ai-sre ask "kafka lag 高怎么办"
./ai-sre runbook "pod频繁重启"

./ai-sre skills list
./ai-sre -o json analyze kafka --lag 1
./ai-sre --no-rag ask "redis 慢查询怎么查"

./ai-sre --skills-dir ./my-skills --knowledge-dir ./my-docs analyze redis --latency 10ms

./ai-sre doctor
```

技能 YAML 中可使用占位符 `{{lag}}`、`{{topic}}` 等（与 flag / `--set` 注入的 context 键一致）。

---

## 开发（前期工程化）

```bash
make vet          # go vet ./...
make test         # go test ./...
make build        # 生成 ./ai-sre
make build-executor   # 生成 bin/opsfleet-executor（与 ai-sre 同引擎，供受管机使用）
make clean        # 删除本机 ai-sre、bin/、dist/、OpsFleet 常见构建产物
```

CI 或发布前建议：`go test ./... && go vet ./...`（`scripts/remote-e2e.sh` 的静态阶段已包含 `go test`）。

---

## 结构化输出（`-o json`）

`analyze` / `ask` / `runbook` 在 `-o json` 下输出 JSON，包含 `answer`、`skill`（命中技能包元数据）、`duration_ms`、`context`、`rag` 等，便于流水线与自动化。

---

## 远程部署与冒烟（团队环境）

默认将本仓库同步到 **`root@192.168.56.11:/root/sre`**；冒烟脚本还会在远程执行 `go vet`、`go test`、`go build`、`./ai-sre doctor` 等（见 `scripts/remote-e2e.sh`）。

在**仓库根目录**执行：

```bash
./scripts/deploy-remote.sh
```

可选环境变量（覆盖默认主机与目录）：

- `DEPLOY_REMOTE`：例如 `root@其它IP`
- `DEPLOY_REMOTE_DIR`：远程目录，默认 `/root/sre`

远程需已安装 Go；目标机密钥仍放在 `~/.config/ai-sre/`（root 用户即为 `/root/.config/ai-sre/`）。

冒烟脚本（在**已部署的远程目录**或本地均可测）：

```bash
SHORT=1 bash scripts/remote-e2e.sh   # vet/test/build/version/doctor/skills list/无凭证负例
bash scripts/remote-e2e.sh         # 含 LLM（需有效 api_key）
```

**发布总入口（Cursor Agent）**：**`.cursor/skills/release-deploy/SKILL.md`** → 再按场景执行 **`.cursor/skills/ai-sre-ship/SKILL.md`**（CLI 同步与 push）、**`.cursor/skills/opsfleetpilot-ship/SKILL.md`**（全栈）、**`.cursor/skills/k8s-offline-deploy-test/SKILL.md`**（K8s 离线）。**README 须在 push 前、发布后保持更新**。**任意本仓文件变更**还应遵循 **`.cursor/rules/monorepo-release.mdc`**（`alwaysApply`）。

---

## 标准约定（同仓）

- **OpsFleet 后端仅使用** `ft-backend/conf/config.yaml`（由 `deploy/config.production.example.yaml` 复制编辑）；仓库内**不得**再保留根路径 `ft-backend/config.yaml` 等重复配置，以免误用。
- **勿提交**：本机编译产物（`ai-sre`、`bin/`、`dist/`、`ft-backend/opsfleet-backend`）、`node_modules`、vim `*.swp`（见根目录 `.gitignore`）。
- **CLI 凭据**：仅用 `~/.config/ai-sre/`，与 OpsFleet 的 PostgreSQL/JWT 配置无关。

---

## 仓库布局

| 路径 | 说明 |
|------|------|
| `internal/cli` | 命令路由 |
| `internal/engine` | AI 编排 |
| `internal/skill` | 技能包 |
| `internal/prompt` | Prompt |
| `internal/rag` | 轻量 RAG |
| `internal/output` | 文本 / JSON |
| `internal/llm` | DeepSeek（OpenAI 兼容 API） |
| `internal/loader` | 内置资源 + 可选目录合并 |
| `internal/config` | 凭据与 tier / 限额 |
| `internal/quota` | 每日 LLM 调用计数（`~/.cache/ai-sre`） |
| `internal/assets/skills/*.yaml` | 内置技能 |
| `internal/assets/knowledge/*.md` | 内置知识片段 |
| `cmd/opsfleet-executor` | OpsFleet 本地执行器入口（调用 `internal/cli`，与 `ai-sre` 同子命令） |
| `ft-backend/` | OpsFleetPilot API（Gin），独立 `go.mod` |
| `ft-front/` | OpsFleetPilot Web（Vue3 + Vite） |
| `deploy/` | Nginx / systemd 模板与生产配置示例 |
| `ansible-agent/` | K8s/Ansible 相关 playbook |
| `PRODUCT_DOC.md` | OpsFleetPilot 产品文档 |
| `docs/` | 归档说明、客户端相关 PRD 等（如 `docs/opsfleet-README.md`、`docs/ft-client-prd-machines.txt`） |
| `scripts/deploy-remote.sh` | 同步本仓并编译 **ai-sre CLI**（默认远端 `/root/sre`） |
| `scripts/deploy-opsfleet-remote.sh` | 同步本仓并构建 **OpsFleet**（Nginx + systemd，`build-all.sh`；可创建 `/etc/opsfleet/backend.env`） |
| `deploy/k8s-mirror/` | K8s 内网制品同步脚本、manifest、Nginx 示例（部署在制品机，常与 192.168.56.11 同机） |
| `scripts/build-all.sh` | 仅构建 OpsFleet 后端 + 前端静态资源 |
| `scripts/remote-e2e.sh` | CLI 端到端冒烟 |

---

## OpsFleetPilot（同仓，与 CLI 并列）

| 操作 | 命令或说明 |
|------|------------|
| 构建 Web + 后端产物 | `make build-opsfleet` 或 `bash scripts/build-all.sh` |
| 仅 vet 后端 Go | `make vet-opsfleet` |
| 远程全栈部署（无 Docker） | `./scripts/deploy-opsfleet-remote.sh`（默认远端目录与 `DEPLOY_REMOTE_DIR` 一致：`/root/sre`；可用 `OPSFLEET_REMOTE_DIR` 覆盖） |
| 部署后自检（在服务器上） | `bash scripts/verify-opsfleet-deployment.sh` |

**实验室虚拟机 `root@192.168.56.11`（本地 root 免密）**：与 **ai-sre** `deploy-remote.sh`、**OpsFleet** `deploy-opsfleet-remote.sh` 使用**同一默认主机**；可在该机上另部署 **K8s 内网制品站**（`deploy/k8s-mirror/README.md`，持久目录默认 `/var/lib/opsfleet-k8s-mirror`）。全栈部署脚本**首次**可在远端创建 **`/etc/opsfleet/backend.env`**，设置 **`OPSFLEET_K8S_MIRROR_BASE_URL`**（默认 `http://192.168.56.11`），供控制台 **「K8s 制品镜像」** 页代理展示 `manifest.json` 与 SHA512。**发布顺序**见 **`.cursor/skills/release-deploy/SKILL.md`** 与 **`.cursor/rules/monorepo-release.mdc`**：**release-deploy** 总清单 → **ai-sre-ship** → 若改 OpsFleet 则 **opsfleetpilot-ship** → 若改 K8s 离线/制品 则 **k8s-offline-deploy-test** → 最后 **`git push`**。

本地开发：在 `ft-backend` 配置 `conf/config.yaml` 后 `go run .`；在 `ft-front` 执行 `npm install && npm run dev`（Vite 代理 `/ft-api`）。

**OpsFleet 控制台登录**：数据库迁移脚本初始化时默认用户名为 **`admin`**、密码为明文 **`password`**（bcrypt）。生产环境请修改；忘记密码可在数据库所在机执行 **`ft-backend/database/reset_admin_password_pg.sql`**（将 **`admin`** 重置为 **`123456`**，与当前运维约定一致）。

**Kubernetes 部署（推荐）**：在 **Kubernetes 部署** 向导中填写参数与节点 IP。离线控制机：**安装或升级 ai-sre**（已安装则覆盖）：`curl -fsSL '<publicApiBase>/api/k8s/deploy/install-ai-sre.sh' | sudo bash`（会写入 `~/.config/ai-sre/opsfleet_api_url` 供后续**自动比对升级**）。全栈机执行 **`./scripts/deploy-opsfleet-remote.sh`** 时，远端 **`build-all.sh` 会生成 `bin/ai-sre`**，并在 **`/etc/opsfleet/backend.env`** 写入 **`OPSFLEET_AISRE_BINARY_PATH=<仓库>/bin/ai-sre`**（**systemd 优先于 config.yaml**），故每次发布控制台分发的 CLI 与源码一致；仅当**未用该脚本部署**时，才需在 `conf/config.yaml` 配置 **`opsfleet.ai_sre_binary_path`**。集群安装：**①** `sudo ai-sre k8s install 'ofpk8s1.…'`；**②** `curl -fsSL '<publicApiBase>/api/k8s/deploy/bootstrap.sh' | sudo bash -s -- 'ofpk8s1.…'`（需 `python3`）；**③** zip 解压后 **`sudo bash install.sh`**。**控制机须能免密 SSH 各节点 `root`**。**卸载**（在曾安装过并记录了引用的控制机上）：`sudo ai-sre uninstall k8s` 或 `sudo ai-sre k8s cleanup 'ofpk8s1.…'`。

**机器与作业**：已移除「机器管理」独立页面；后端 `/api/machine` 与作业中心仍用于在线机器列表与任务目标（见 [`PRODUCT_DOC.md`](PRODUCT_DOC.md)）。

---

## 安全说明

密钥仅从本机文件读取，勿将真实 `api_key` / `config.yaml` 提交到 Git。建议目录权限 `700`、密钥文件 `600`。

---

## Git 远程

```text
git@github.com:antpan-san/ai-sre.git
```
