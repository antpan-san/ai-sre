# ai-sre（AI SRE Copilot）

Go CLI + 同仓 **OpsFleetPilot** Web/API（`ft-backend/`、`ft-front/`、`deploy/`、`ansible-agent/`）。核心能力：**AI 故障诊断**（`check`）、**只读快采**（`probe`）、**Runbook**（`runbook`）、**问答**（`ask`）；**技能包**（`expert skills`）；**K8s 离线安装**（`k8s`）；**批量作业**（`job run`）。旧命令 `analyze` / `topic diagnose` / 顶层 `diagnose` 仍可用但已弃用。

**版本**：`./ai-sre version`（见 `internal/cli/version.go`）。**自升级**：**每次**执行子命令前快速探测 `GET .../cli/ai-sre/version`（多基址、单址约 1.2s）；有更新则自动下载并 re-exec。基址顺序：`OPSFLEET_API_URL` → `~/.config/ai-sre/opsfleet_api_url` → 内嵌实验室 → **生产 `http://opsfleetpilot.com/ft-api` 回退**。关闭：`OPSFLEET_NO_AUTO_UPGRADE=1` 或 `--no-auto-upgrade`。调试：`OPSFLEET_AUTO_UPGRADE_VERBOSE=1`。开发规范：`.cursor/skills/ai-sre-client-dev/SKILL.md`。构建控制台：`make build-opsfleet`。

---

## 命令分层（记三条即可）

| 层级 | 命令 | 是否调 LLM | 用途 |
|------|------|------------|------|
| AI 诊断 | `check [topic]` | 是 | 故障根因分析（kafka / k8s / redis / nginx / mysql / postgresql / elasticsearch） |
| 只读快采 | `probe <topic> <target>` | 否 | 中间件本地指标/日志采集（JSON 可喂给 `check`） |
| 环境自检 | `doctor` | 否 | CLI 凭据、配额、技能加载自检 |
| Go 运行时 | `check go --pid\|--pod\|…` | 部分 | 进程/K8s 工作负载运行时诊断（原顶层 `diagnose`） |

`analyze`、`kafka diagnose`、`diagnose` 等为兼容别名，执行时会提示弃用。

## 子命令一览

| 命令 | 说明 |
|------|------|
| `ai-sre check [topic]` | **（推荐）** AI 故障诊断，同原 `analyze`；`topic`：`kafka` \| `k8s` \| `nginx` \| `redis` \| `elasticsearch` \| `domain` \| `dns`。**domain**：`check domain <fqdn>`（DNS/HTTP/TLS 只读采集 + AI）。**k8s**：本机有 `kubectl` 时可先只读采集再调服务端 AI；`--pod` 可为场景名或具体 Pod 名 |
| `ai-sre probe domain <fqdn>` | 域名只读快采（无 AI）；配合 `check domain` 使用 |
| `ai-sre check code <CODE>` | 错误码 → 根因卡片：`OPSFLEET_*` 等；`--list` / `--detail` |
| `ai-sre probe kafka <bootstrap>` | Kafka 只读快采（原 `kafka diagnose`） |
| `ai-sre probe redis <host:port>` | Redis 只读快采 |
| `ai-sre probe elasticsearch <url>` | ES 只读快采；`--insecure` / `--ai` 等同旧 diagnose |
| `ai-sre probe mysql <dsn>` | MySQL 只读快采 |
| `ai-sre probe nginx` | Nginx access log 统计 |
| `ai-sre check go --pid <pid>` / `--pod <ns/pod>` | Go 运行时诊断（原 `diagnose`） |
| `ai-sre analyze [topic]` | **已弃用**，请改用 `check` |
| `ai-sre ask [question]` | 知识问答：默认经 OpsFleet `POST /api/ai/ask`（无需本机 api_key）；服务端失败且本机有凭据时回退本地 LLM+RAG |
| `ai-sre runbook [scenario]` | 生成 Runbook：默认经 `POST /api/ai/runbook`；回退逻辑同 `ask` |
| `ai-sre expert skills list` | 列出内置 + `--skills-dir` 合并后的技能包 |
| `ai-sre expert skills server` | 列出 OpsFleet 服务端注册的技能包（builtin + 生成中的 generated 版本） |
| `ai-sre expert skills status` | 通过 CLI token 同步当前可执行能力、订阅状态、执行模式和是否需要服务端计划单；不返回技能 YAML / Prompt |
| CLI 参数层纠错 | 未知命令/flag 或 flag 作用域错误时，仅本地给出最多 3 条建议（含 `auto_iteration_created=false`），**不**触发自动迭代 |
| CLI 能力层 fulfillment | 命令与参数正确但能力不足时，调用 `POST /api/cli/fulfillment/plan`（须带 `command_catalog_digest`）；服务端返回 `subscription_required` / `granted_retry` / `auto_iteration_created` / `awaiting_approval` 等公开字段 |
| `ai-sre expert skills feedback --topic <t> -m "…"` | 提交诊断反馈样本；可经 `POST /api/cli/feedback/analyze` 触发自动迭代研判 |
| `ai-sre expert skills refine --topic <t>` | 让服务端基于最近 N 次样本 + 反馈调用 LLM 产出新版技能包（生成在 `OPSFLEET_AI_SKILL_DATA_DIR/generated/<topic>.yaml`） |
| `ai-sre doctor` | 自检（凭据、tier、配额计数、技能/知识加载；**不调用 LLM**） |
| `ai-sre version` | 打印版本号 |
| `ai-sre help` | 帮助 |
| `ai-sre nginx uninstall` | 默认仅卸载由 `ai-sre service install` 写入本机状态的 Nginx；`-f/--force` 会强制检测并清理本机 Nginx 相关进程、包、容器、配置、日志和缓存 |
| `ai-sre service install --deploy-id <id> --token <token> --api-url <base>` | 基础服务安装执行器：从 OpsFleet 服务端拉取 Nginx / HAProxy / Redis / Kafka / MySQL / PostgreSQL / Elasticsearch 部署规格，执行安装、写配置、启动与健康检测，并回传步骤状态 |
| `ai-sre nginx update` | 在已通过 OpsFleet 服务部署安装过 Nginx 的目标机上，拉取服务端最新 Nginx 规格，重写配置并重启生效 |
| `ai-sre elasticsearch update` | 同上，作用于 Elasticsearch；自动复跑 system-tune（vm.max_map_count）、写 `elasticsearch.yml` + `jvm.options.d/heap.options`（包安装时另加 systemd drop-in；**binary** 方式则配置在 `install_prefix/config` 且 `ES_PATH_CONF` 指向该目录）、轮询 `_cluster/health` |
| `ai-sre elasticsearch uninstall` | 默认停服并移除 ai-sre 管理的 systemd/配置痕迹；`--purge-package` 在 **package** 下卸载发行版包，在 **binary** 下删除 `install_prefix` 目录；`--purge-data` 清理 data/log；`-f/--force` 端到端清理（容器、包、二进制目录、配置、数据、日志、apt/yum 仓库与 GPG 密钥） |
| `ai-sre k8s …` | 离线包下载、控制机 `install` / `cleanup` / `diagnose` 等（见 `ai-sre k8s --help`） |
| `ai-sre job run --machines <uuid,…> -c '…'` | **（0.5.10）** 经 OpsFleet `POST /api/job/execute` 在多台已在线 Agent 上批量执行命令或脚本；`--timeout`、`--wait` / `--max-wait`、`--print-console-url`（打开带 `?jobId=` 的控制台与同页「执行结果」对齐）；需 `OPSFLEET_API_URL` + 令牌。**`opsfleet-executor`** 亦含该子命令 |
| `ai-sre node tune time-sync …` | 与控制台「初始化工具 → 时间同步」等价的 CLI；本机构建 inventory + chrony / timesyncd playbook 并调用 `ansible-playbook`；缺失 ansible 时按 apt/dnf/yum 自动安装；未填 `--clients` 仅对 localhost 执行 |
| `ai-sre node tune sys-param …` | 与「系统参数优化」等价：sysctl + br_netfilter/overlay 内核模块 + ulimit + 关闭 swap；可用 `--sysctl key=value`（多次）扩展或 `--extra-only` 只用显式提供的项 |
| `ai-sre k8s diagnose` | 本机自检 K8s 常见抖动根因（**非** `probe`/`check`）；`--json` 可喂给 `ai-sre check k8s --issue instability` |
| `ai-sre upgrade` | 与 OpsFleet 对比版本后覆盖本机 `ai-sre` 二进制（需能访问上表基址） |
| `ai-sre uninstall k8s` | 在控制机 `root` 下用 Ansible `pre_cleanup` 全量清集群；**优先**本机 `/var/lib/opsfleet-k8s/last-bundle`（`install.sh` 预检后同步），无则再试拉 `ofpk8s1` 或 `--workdir` / `--force`（见 `ai-sre uninstall k8s --help`） |

**与 OpsFleet 联动（摘要）**

- **AI**：`check` / `ask` / `runbook` 优先走控制台 API；本机可配 `api_key` 作回退。`check` 可附带本机 `kubectl` 或 `ai-sre probe <topic> … --json` 采集结果。
- **诊断任务单**：已绑定 CLI 时，可向控制台申请只读采集计划（k8s / redis / kafka 等）；结果用于诊断与技能沉淀。
- **技能包**：控制台审核「待审资产」；`expert skills feedback` / `expert skills refine` 参与技能更新。
- **自动迭代**（仅 `super_admin`）：控制台 **订阅与计费 → 自动迭代** 单页管理任务与审批；**任务列表**支持按状态、来源、Topic 与关键词（标题/需求）筛选，并展示创建时间。页面创建与 `code-agent-worker` 拉取任务时，Agent 须遵守 **省 Token 开发规范**（`.cursor/skills/auto-iteration-dev/SKILL.md`；提交需求建议按「目标/范围/验收」模板填写）。本机 Worker 完成后（含页面手动创建的开发任务）会向钉钉推送 **Markdown** 通知（**已完成 / 失败 / 待审批**）；创建、批准等节点也会通知。在部署机 `/etc/opsfleet/backend.env` 配置 `OPSFLEET_AUTO_ITERATION_DINGTALK_WEBHOOK`（勿提交 Git）；机器人若启用关键词安全，设置 `OPSFLEET_AUTO_ITERATION_DINGTALK_KEYWORD`（默认 `操`，须与钉钉机器人一致）。
- **登录有效期**：JWT 访问令牌默认 **24 小时**（`jwt.access_token_exp` 或 `OPSFLEET_JWT_ACCESS_TOKEN_EXP`）。
- **错误码**：`ai-sre check code <CODE>` 或控制台「错误码」页。
- **Go 运行时**：`ai-sre check go --pid|--name|--pod` 采样 proc/cgroup，可上传至控制台「运行时诊断」。
- **反馈**：使用 `expert skills feedback` 或能力层 fulfillment 链路提交；`check` 结束不再交互询问。
- **普通用户工作台**：登录后进入 `/app/dashboard`，侧栏收敛为概览、执行记录、工作负载、能力中心、问题排查、作业中心、设置；`admin` / `super_admin` 仍进入 `/admin/dashboard` 管理后台。
- **订阅包**：控制台普通用户侧通过「能力中心」展示独立订阅包卡片，不对外展示等级分层。订阅某个 `pack.*` 或 `skillpack.*` 后即可使用对应能力；未订阅时后端返回 `PAYWALL_<pack_key>`，前端弹出对应订阅包。Stripe Price 未配置时仍展示订阅包，但按钮显示“联系管理员开通”。

K8s 安装细节、节点初始化、制品镜像等见 [`deploy/k8s-mirror/README.md`](deploy/k8s-mirror/README.md)、[`PRODUCT_DOC.md`](PRODUCT_DOC.md)。

别名：`ops-ai`（与 `ai-sre` 等价）。

---

## 全局参数（根命令）

| 参数 | 说明 |
|------|------|
| `--config` | 凭据 YAML 路径（默认 `~/.config/ai-sre/config.yaml`） |
| `--key-file` | 仅含 API Key 的文件路径 |
| `-v` / `--verbose` | 打印凭据文件路径、技能数、知识片段数等 |
| `--no-rag` | 关闭知识库检索 |
| `-o` / `--output` | 输出格式：`text`（默认）或 `json`（适用于 `check` / `ask` / `runbook`） |
| `--skills-dir` | 额外技能包目录（`*.yaml`，与内置合并；同名覆盖） |
| `--knowledge-dir` | 额外知识库目录（`*.md`，与内置合并参与 RAG） |

`check [topic]` 常用 flag：`--lag`、`--topic`、`--pod`、`--namespace`、`--issue`、`--code`、`--upstream`、`--latency`、`-d`/`--set key=value`、`--yes`（确认服务端只读诊断任务单）。

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

# --- 本地限额（遗留本地配置；商业能力以控制台订阅包为准）---
# tier: free          # 仅本地运行时限额用途；控制台商业能力只使用独立订阅包
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

./ai-sre check kafka --lag 100000
./ai-sre probe kafka 10.0.0.1:9092
./ai-sre probe kafka 'b1:9092,b2:9092,b3:9092' --config ./client.properties
./ai-sre probe redis 10.0.0.2:6379
./ai-sre probe elasticsearch 127.0.0.1:9200
./ai-sre probe elasticsearch https://es:9200 --insecure --user elastic --password '***'
./ai-sre check elasticsearch -d base_url=http://127.0.0.1:9200
./ai-sre probe mysql 'user:pass@tcp(10.0.0.3:3306)/mysql?timeout=5s'
./ai-sre probe nginx --access-log /var/log/nginx/access.log --tail 10000
./ai-sre check go --pid "$(pgrep -n my-go-service)"
./ai-sre check go --name my-go-service
./ai-sre check go --pod default/api-0/app
./ai-sre check go -o json --pod default/api-0/app
./ai-sre service install --api-url http://192.168.56.11:9080/ft-api --deploy-id <id> --token <token>
sudo ai-sre nginx update
sudo ai-sre nginx uninstall
# 如确认该目标机上的 nginx 包也由 ai-sre 安装且要一并移除:
sudo ai-sre nginx uninstall --purge-package
# 强制清理本机所有 Nginx 相关环境（不要求 ai-sre 安装状态）:
sudo ai-sre nginx uninstall -f

# Elasticsearch（OpsFleet 控制台「服务部署」选 package / docker / binary，生成 deploy_id/token 后在目标机一键执行）：
# binary：官方 Linux tarball 解压到 install_prefix（默认 /opt/elasticsearch），systemd 拉起，装完即可 curl 本机 http 端口。
./ai-sre service install --api-url http://192.168.56.11:9080/ft-api --deploy-id <id> --token <token>
sudo ai-sre elasticsearch update
sudo ai-sre elasticsearch uninstall                        # 停服并清理 ai-sre 单元/配置痕迹，保留数据与安装目录
sudo ai-sre elasticsearch uninstall --purge-package        # 另移除 apt/yum 包，或 binary 时删除 install_prefix
sudo ai-sre elasticsearch uninstall --purge-data           # 同时清理 data/log 目录
sudo ai-sre elasticsearch uninstall -f                     # 强制端到端清理（不要求 ai-sre 安装状态）
./ai-sre check k8s --pod pending
./ai-sre check k8s --pod kube-controller-manager-k8s-master-0 -n kube-system
./ai-sre ask "kafka lag 高怎么办"
./ai-sre runbook "pod频繁重启"

./ai-sre expert skills list
./ai-sre expert skills server                           # 看服务端注册了哪些 builtin/generated
./ai-sre expert skills feedback --topic k8s -m "事件链条没串好"
./ai-sre expert skills refine --topic k8s --hint "对前一轮答案不准确"
./ai-sre -o json check kafka --lag 1
./ai-sre check redis
./ai-sre check redis 192.168.56.11:6379
./ai-sre check kafka
./ai-sre check kafka 10.0.0.1:9092
./ai-sre --no-rag ask "redis 慢查询怎么查"

./ai-sre --skills-dir ./my-skills --knowledge-dir ./my-docs check redis --latency 10ms

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

受管机执行面首选 `opsfleet-executor agent`：

```bash
sudo bin/opsfleet-executor agent --api-url http://<opsfleet-host>:9080/ft-api
# 测试可用：bin/opsfleet-executor agent --allow-no-root --once --api-url http://127.0.0.1:9080/ft-api
```

Agent 复用 `/api/v1/heartbeats`、`/api/v1/task/log`、`/api/v1/task/report`，首批支持 `run_shell`、`sys_init`、`time_sync`、`security_harden`、`disk_optimize`、`install_monitor`、`sync_nodes`。本地日志默认写入 `/var/log/opsfleet-executor`。

CI 或发布前建议：`go test ./... && go vet ./...`（`scripts/remote-e2e.sh` 的静态阶段已包含 `go test`）。

---

## 结构化输出（`-o json`）

`analyze` / `ask` / `runbook` 在 `-o json` 下输出 JSON，包含 `answer`、`skill`（命中技能包元数据）、`duration_ms`、`context`、`rag` 等，便于流水线与自动化。

---

## 本机与远程部署

**本机（每次改代码后必做，见 `.cursor/skills/release-deploy/SKILL.md`）**：

```bash
./scripts/deploy-local.sh
```

触及 OpsFleet 路径时：`DEPLOY_LOCAL_OPSFLEET=1 ./scripts/deploy-local.sh`（本机构建，勿提交 `bin/`、`dist/`）。

**远程（团队实验室）**：默认同步到 **`root@192.168.56.11:/root/sre`**；冒烟脚本还会在远程执行 `go vet`、`go test`、`go build`、`./ai-sre doctor` 等（见 `scripts/remote-e2e.sh`）。

```bash
./scripts/deploy-remote.sh
```

可选环境变量（覆盖默认主机与目录）：

- `DEPLOY_REMOTE`：例如 `root@其它IP`
- `DEPLOY_REMOTE_DIR`：远程目录，默认 `/root/sre`
- `OPSFLEET_SKIP_REMOTE=1`：不把 OpsFleet 当作 API 基址（不调服务端 `analyze`/`ask`/`runbook`、也不用内建基址做自升级比对）。**仅推荐** `scripts/remote-e2e.sh` 无凭证负例或离线单测；生产环境勿随意开启。

远程需已安装 Go；目标机密钥仍放在 `~/.config/ai-sre/`（root 用户即为 `/root/.config/ai-sre/`）。

冒烟脚本（在**已部署的远程目录**或本地均可测）：

```bash
SHORT=1 bash scripts/remote-e2e.sh   # vet/test/build/version/doctor/expert skills list/无凭证负例
bash scripts/remote-e2e.sh         # 含 LLM（需有效 api_key）
```

实验环境固定验收（默认 `root@192.168.56.11`）：

```bash
ssh root@192.168.56.11 'systemctl is-active opsfleet-backend nginx'
ssh root@192.168.56.11 'curl -fsS http://127.0.0.1:9080/health'
ssh root@192.168.56.11 'cd /root/sre && ./ai-sre doctor'
ssh root@192.168.56.11 'cd /root/sre && ./ai-sre expert skills list'
ssh root@192.168.56.11 'cd /root/sre && bash scripts/verify-opsfleet-deployment.sh'
ssh root@192.168.56.11 'cd /root/sre && SHORT=1 bash scripts/remote-e2e.sh'
```

**发布总入口（Cursor Agent）**：**`.cursor/skills/release-deploy/SKILL.md`** → 再按场景执行 **`.cursor/skills/ai-sre-ship/SKILL.md`**（CLI 同步与 push）、**`.cursor/skills/opsfleetpilot-ship/SKILL.md`**（实验室全栈）、**`.cursor/skills/production-deploy/SKILL.md`**（生产 `opsfleetpilot.com`，保留生产 Nginx 与 `config.yaml`）、**`.cursor/skills/k8s-offline-deploy-test/SKILL.md`**（K8s 离线）。**README 须在 push 前、发布后保持更新**。**任意本仓文件变更**还应遵循 **`.cursor/rules/monorepo-release.mdc`**（`alwaysApply`）。

---

## 标准约定（同仓）

- **OpsFleet 后端仅使用** `ft-backend/conf/config.yaml`（由 `deploy/config.production.example.yaml` 复制编辑）；仓库内**不得**再保留根路径 `ft-backend/config.yaml` 等重复配置，以免误用。
- **勿提交**：本机编译产物（`ai-sre`、`bin/`、`dist/`、`ft-backend/opsfleet-backend`）、`node_modules`、vim `*.swp`（见根目录 `.gitignore`）。
- **CLI 凭据**：仅用 `~/.config/ai-sre/`，与 OpsFleet 的 PostgreSQL/JWT 配置无关。
- **服务端 AI 回退**：若要启用无本地 key 的 `analyze` 自动回退，请在 OpsFleet 后端环境配置 `OPSFLEET_AI_API_KEY`（可选 `OPSFLEET_AI_BASE_URL`、`OPSFLEET_AI_MODEL`）。
- **服务端技能数据目录**：可选 `OPSFLEET_AI_SKILL_DATA_DIR`（默认尝试 `/var/lib/opsfleet/ai-skills`，否则 `./data/ai-skills`）。诊断样本会以 JSONL 形式追加到该目录的 `samples/<topic>.jsonl`，反馈到 `feedback/<topic>.jsonl`；`ai-sre expert skills refine` 产出的新版技能包落到 `generated/<topic>.yaml`，旧版本归档到 `generated/<topic>.history/<ts>.yaml`。**全栈部署脚本会自动 mkdir + 写入 backend.env。**

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
| `scripts/deploy-local.sh` | **本机** `go vet` + `go build` + 版本与 `version.go` 一致校验 |
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

**实验室部署**：默认主机 `root@192.168.56.11`，目录 `/root/sre`。密钥与覆盖项写在 **`/etc/opsfleet/backend.env`**（见 `deploy/backend.env.example`）。

**控制台**：`ft-backend/conf/config.yaml` + `npm run dev`（前端代理 `/ft-api`）。默认账号 `admin` / `password`（`super_admin`），生产请修改。CORS 须在 `security.cors_allowed_origins` 填写浏览器访问的完整 Origin。

**主要功能页**：概览、K8s 部署、制品目录、执行记录、作业中心、技能包与订阅（`super_admin`）、**自动迭代**（`super_admin`）、错误码帮助、节点初始化工具。

**CLI 安装**：`GET .../install-ai-sre.sh` 或顶栏「安装 ai-sre」绑定账号后，使用 `opsfleet_token` + 指纹调用 AI 与作业 API。

**K8s 安装**：控制台表单生成离线包 → 控制机 `install.sh` / `ai-sre k8s install`；控制机需对各节点 `root` 免密 SSH。卸载：`ai-sre uninstall k8s`。

更多产品说明见 [`PRODUCT_DOC.md`](PRODUCT_DOC.md)。

---

## 安全说明

密钥仅从本机文件读取，勿将真实 `api_key` / `config.yaml` 提交到 Git。建议目录权限 `700`、密钥文件 `600`。

---

## Git 远程

```text
git@github.com:antpan-san/ai-sre.git
```
