# ai-sre（AI SRE Copilot）

Go 实现的 CLI：**技能包（Skill Pack）+ Prompt 组装 + 可选轻量 RAG + DeepSeek LLM**，对应产品文档中的三条核心能力：

1. **故障诊断** — `analyze`
2. **Runbook 生成** — `runbook`
3. **知识库问答** — `ask`（检索本地 Markdown 片段注入 Prompt）

另：**技能注册** — `skills list`（发现已加载技能包）。

**当前版本**：以运行环境为准，执行 `./ai-sre version`（与源码中 `internal/cli/version.go` 的 `Version` 变量对齐）。**自升级默认基址**：内建 `http://192.168.56.11:9080/ft-api`（代码常量 `internal/cli/EmbeddedOpsfleetAPIBase`）；若设置 **`OPSFLEET_API_URL`** 则优先生效。未关闭时，**每次**执行子命令前（含 `uninstall k8s` 前）会拉 `GET .../api/k8s/deploy/cli/ai-sre/version` 比对，较新则覆盖本机 `ai-sre` 后 **re-exec** 同一命令（非 Windows）。`curl` 安装脚本写入的 **`~/.config/ai-sre/opsfleet_api_url`** 或 **`config.yaml` 的 `opsfleet_api_url`** 亦可作为覆盖来源。关闭：`OPSFLEET_NO_AUTO_UPGRADE=1`；关自动后仅提示：`OPSFLEET_UPGRADE_HINT=1`。**显式一次升级**：`sudo ai-sre upgrade -y`。

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
| `ai-sre kafka diagnose <bootstrap-server>` | Kafka 极简快诊：优先用内置 Go 客户端直连采集（支持 `--config` 的 SASL/TLS）；失败再尝试 Kafka CLI，最后才回退 AI |
| `ai-sre redis diagnose <host:port>` | Redis 极简快诊：只读采集 INFO，定位连接拒绝、淘汰和连接压力 |
| `ai-sre mysql diagnose <dsn>` | MySQL 极简快诊：只读采集连接、慢查询、线程与只读状态 |
| `ai-sre nginx diagnose` | Nginx 日志统计分析：状态码分布、Top 路径、P95 延迟、5xx/4xx 风险识别 |
| `ai-sre service install --deploy-id <id> --token <token> --api-url <base>` | 基础服务安装执行器：从 OpsFleet 服务端拉取 Nginx / HAProxy / Redis / Kafka / MySQL / PostgreSQL 部署规格，执行安装、写配置、启动与健康检测，并回传步骤状态 |
| `ai-sre nginx update` | 在已通过 OpsFleet 服务部署安装过 Nginx 的目标机上，拉取服务端最新 Nginx 规格，重写配置并重启生效 |
| `ai-sre k8s …` | 离线包下载、控制机 `install` / `cleanup` / `diagnose` 等（见 `ai-sre k8s --help`） |
| `ai-sre node tune time-sync …` | 与控制台「初始化工具 → 时间同步」等价的 CLI；本机构建 inventory + chrony / timesyncd playbook 并调用 `ansible-playbook`；缺失 ansible 时按 apt/dnf/yum 自动安装；未填 `--clients` 仅对 localhost 执行 |
| `ai-sre node tune sys-param …` | 与「系统参数优化」等价：sysctl + br_netfilter/overlay 内核模块 + ulimit + 关闭 swap；可用 `--sysctl key=value`（多次）扩展或 `--extra-only` 只用显式提供的项 |
| `ai-sre k8s diagnose` | 本机自检 K8s 常见抖动根因：**时钟跳变 / etcd 慢盘 / kubelet SandboxChanged / 预检缺项（swap/br_netfilter/sysctl）**；`--preflight` 只跑部署前预检，`--json` 输出可直接喂给 `ai-sre analyze k8s --issue instability` |
| `ai-sre upgrade` | 与 OpsFleet 对比版本后覆盖本机 `ai-sre` 二进制（需能访问上表基址） |
| `ai-sre uninstall k8s` | 在控制机 `root` 下用 Ansible `pre_cleanup` 全量清集群；**优先**本机 `/var/lib/opsfleet-k8s/last-bundle`（`install.sh` 预检后同步），无则再试拉 `ofpk8s1` 或 `--workdir` / `--force`（见 `ai-sre uninstall k8s --help`） |

`analyze` 新行为（自动编排）：优先本地技能诊断；若本地无凭据或覆盖不足，会自动回退到 OpsFleet 服务端 `POST /api/ai/diagnose`（由服务端 DeepSeek 执行），并可返回技能草案用于自动沉淀。

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

# --- 自动技能迭代（可选）---
# 在 ~/.config/ai-sre/evolution.yaml 中设置：
# mode: full_pipeline
# target_branch: main
# max_auto_commits: 1
# pre_push_test_cmd: "go test ./..."
# auto_commit_msg: "chore(skills): auto-evolve generated skill"
# fail_fast_streak: 3
# enable_generated: true
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
./ai-sre kafka diagnose 10.0.0.1:9092
./ai-sre kafka diagnose 'b1:9092,b2:9092,b3:9092'
./ai-sre redis diagnose 10.0.0.2:6379
./ai-sre mysql diagnose 'user:pass@tcp(10.0.0.3:3306)/mysql?timeout=5s'
./ai-sre nginx diagnose --access-log /var/log/nginx/access.log --tail 10000
./ai-sre service install --api-url http://192.168.56.11:9080/ft-api --deploy-id <id> --token <token>
sudo ai-sre nginx update
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
- **服务端 AI 回退**：若要启用无本地 key 的 `analyze` 自动回退，请在 OpsFleet 后端环境配置 `OPSFLEET_AI_API_KEY`（可选 `OPSFLEET_AI_BASE_URL`、`OPSFLEET_AI_MODEL`）。

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
| `deploy/k8s-mirror/` | K8s 内网制品同步脚本、manifest、Nginx 示例、**opsfleet-k8s-mirror-serve**（未命中缓存时从公网拉取并落盘，见 `deploy/k8s-mirror/README.md`；部署在制品机，常与 192.168.56.11 同机） |
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

**执行记录**：左侧菜单最下方提供 **执行记录** 页面，统一持久化展示 `ai-sre` CLI、K8s 一键安装 / bootstrap、初始化工具复制脚本、作业中心任务等执行历史。记录包含来源、目标主机/资源、命令或脚本摘要、开始/结束时间、退出码、输出摘要、执行效果 JSON、回滚能力与回滚状态。目标机侧上报为 **best-effort**：脚本或 CLI 能连到 OpsFleet API 时会调用 `/api/execution-records/report/*` 写入开始、事件和结束状态；上报失败不会改变原命令退出码。页面回滚采用保守策略：同目标或同资源在该记录之后存在成功执行时，会先提示关联影响；可自动或半自动回滚的记录会创建一条关联的 rollback 记录，不可验证恢复的命令会显示人工回滚建议。

**Kubernetes 部署（推荐）**：在 **Kubernetes 部署** 页按折叠配置项填写安装预检、**控制平面部署方式**（二进制 + systemd 或 kubelet **静态 Pod**）、基础集群信息、节点、核心组件、网络、存储、高级配置与部署确认，不再使用「下一步」线性向导。默认仅展开 **安装预检**，其它配置项默认折叠；标题区域右侧提供 **安装或升级 ai-sre**（已安装则覆盖）：`curl -fsSL '<publicApiBase>/api/k8s/deploy/install-ai-sre.sh' | sudo bash`（会写入 `~/.config/ai-sre/opsfleet_api_url` 供后续**自动比对升级**）。全栈机执行 **`./scripts/deploy-opsfleet-remote.sh`** 时，远端 **`build-all.sh` 会生成 `bin/ai-sre`**，并在 **`/etc/opsfleet/backend.env`** 写入 **`OPSFLEET_AISRE_BINARY_PATH=<仓库>/bin/ai-sre`**（**systemd 优先于 config.yaml**），故每次发布控制台分发的 CLI 与源码一致；仅当**未用该脚本部署**时，才需在 `conf/config.yaml` 配置 **`opsfleet.ai_sre_binary_path`**。集群安装：**①** `sudo ai-sre k8s install 'ofpk8s1.…'`；**②** `curl -fsSL '<publicApiBase>/api/k8s/deploy/bootstrap.sh' | sudo bash -s -- 'ofpk8s1.…'`（需 `python3`）；**③** zip 解压后 **`sudo bash install.sh`**。**控制机须能免密 SSH 各节点 `root`**。离线配置里若 worker 填了与 master 相同 IP，后端会在生成 inventory 时自动去重（master 本身已安装 kubelet 并注册为 Node，无需重复声明）。同一角色列表内（master 或 worker 自身）仍不允许重复 IP。**卸载**（在曾安装过并记录了引用的控制机上）：`sudo ai-sre uninstall k8s` 或 `sudo ai-sre k8s cleanup 'ofpk8s1.…'`。

**部署前的节点初始化（可选但推荐）**：Kubernetes 部署页最前面的 **安装预检** 配置项合并展示「离线安装必读」与「环境预检」，用于提前确认 root 免密 SSH、NTP、br_netfilter、sysctl、swap、节点架构与主机名等风险；也可手动进入 **初始化工具**（`/init-tools`，**单页、无子菜单、内容区满宽、4 列固定卡片**）优化节点环境，避免 calico-node / coredns 在 NTP 漂移、br_netfilter 缺失等情况下反复 Killing。所有优化项以紧凑卡片形式集中：**时间同步 / 系统参数优化 / 系统安全加固 / 磁盘分区优化**。卡片大小一致、始终一行 4 个：顶部标题与底部操作按钮固定，配置项较多时仅卡片中部区域纵向滚动；卡片不出现横向滚动条，页面在内容未溢出时也不出现整页滚动条。每张卡片自包含 **目标节点（多选）**、**系统类型**（Ubuntu/Debian/CentOS/Rocky/RHEL/openEuler/Kylin/其它 Linux）与对应工具的关键参数：

- **时间同步**：NTP 工具（chrony / timesyncd）、主源、备用源、时区、同步间隔、`ON_CONFLICT` 策略
- **系统参数**：sysctl 参数表（K8s 必填项默认勾选）、关 swap、提升 ulimit、`ON_CONFLICT`
- **安全加固**：禁 root SSH、改 SSH 端口、防火墙、Fail2ban、自动更新、`ON_CONFLICT`
- **磁盘**：SSD TRIM、文件系统挂载优化（noatime）、Swap 大小（auto / 1G–16G）、`ON_CONFLICT`

点击「**生成执行脚本**」会弹出脚本预览对话框，含三个 Tab：**Ansible 执行脚本**（始终可直接运行）、**curl 一键（roadmap）**、**ai-sre CLI**（在 ai-sre 中已实现的卡片显示为可执行，否则仍标 `roadmap`）。底部「复制 / 下载」按钮跟随当前选中 Tab：可执行 Tab 显示「复制脚本 / 下载脚本」，roadmap Tab 显示「复制（roadmap）」并禁用下载。`ai-sre 0.4.5+` 起 **时间同步** 与 **系统参数优化** 两张卡片对应的 `ai-sre node tune time-sync` / `ai-sre node tune sys-param` 子命令已落地（见 `internal/cli/node.go`；`0.4.6` 修正 `when:` 以双引号开头导致 Ansible YAML 解析失败、并在执行前 `--syntax-check`；`0.4.8` 修正「`--on-conflict skip` 在目标已有 NTP 服务时会连同时区也被跳过」的子任务次序，并在 stderr 显式打印「将变更：A, B；本机不在内」）。注意：`time-sync` / `sys-param` 只对 `--clients` / `--master-node` / `--nodes` 列出的节点动手，**不会**自动包含跑 ai-sre 的那台控制机；如需顺带配置控制机本身，请把它的 IP 加进列表。低版本节点首次执行命令会触发自动升级（`internal/cli/upgrade.go`）。系统安全加固 / 磁盘分区优化两张卡片仍是 roadmap 占位，复制运行仍会得到 `unknown command`：

- **Bash 脚本**：完整可执行 bash，含 `set -euo pipefail`、自动备份至 `/var/backups/ai-sre/<ts>/`、写入幂等的 drop-in 配置文件（如 `/etc/sysctl.d/99-ai-sre.conf`、`/etc/ssh/sshd_config.d/99-ai-sre.conf`），并在末尾打印验证状态与回滚命令；支持「复制」与「下载 .sh」
- **存在检测**：脚本默认 `ON_CONFLICT=skip`，检测到节点已运行其他时间同步服务（chrony/ntpd/systemd-timesyncd 等）或已存在 ai-sre drop-in 时直接退出并打印当前状态，**不进行任何写入或重启**；需要覆盖请改用 `ON_CONFLICT=force`
- **ai-sre CLI**：`ai-sre node tune <subcmd>` 子命令在控制机上 `sudo` 执行（按 `--clients` / `--nodes` 列表 SSH 到目标节点）；当前已实现 **time-sync** 与 **sys-param**（>= 0.4.8），命令在 Go 内构建与 Ansible Tab 等价的 inventory + playbook，**先跑 `ansible-playbook --syntax-check`** 再正式执行；`time-sync` 中「设置时区」固定排在 NTP 处理之前，因此即使 `--on-conflict skip` 触发 `meta: end_play`，时区也已生效。缺失 ansible 时按 apt/dnf/yum 自动安装（可用 `--auto-install-ansible=false` 关闭、`--dry-run` 仅打印）。**security** 与 **disk** 仍是 roadmap，等子命令落地后 ai-sre 自动升级会随版本带过去
- **多节点批量**：`for ip in <ips>; do ssh root@$ip "bash -s" < script.sh; done` 与 curl 一键模式（curl 模式需后端 `/ft-api/api/init-tools/scripts/<name>.sh` 配合，列在 roadmap）

完成所需项后点顶部「返回 K8s 部署」回到折叠配置页。旧地址 `/init-tools/system-param` 等会被路由自动重定向到该单页。

**机器与作业**：已移除「机器管理」独立页面；后端 `/api/machine` 与作业中心仍用于在线机器列表与任务目标（见 [`PRODUCT_DOC.md`](PRODUCT_DOC.md)）。

---

## 安全说明

密钥仅从本机文件读取，勿将真实 `api_key` / `config.yaml` 提交到 Git。建议目录权限 `700`、密钥文件 `600`。

---

## Git 远程

```text
git@github.com:antpan-san/ai-sre.git
```
