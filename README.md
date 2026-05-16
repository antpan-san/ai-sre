# ai-sre（AI SRE Copilot）

Go 实现的 CLI：**技能包（Skill Pack）+ Prompt 组装 + 可选轻量 RAG + DeepSeek LLM**，对应产品文档中的三条核心能力：

1. **故障诊断** — `analyze`
2. **Runbook 生成** — `runbook`
3. **知识库问答** — `ask`（检索本地 Markdown 片段注入 Prompt）

另：**技能注册** — `skills list`（本地），`skills server`（OpsFleet 服务端已注册技能：builtin + generated），`skills status`（当前 CLI 能力/订阅/计划单状态，只下发状态不下发 YAML/Prompt）。**服务端自迭代** — `skills feedback` 提交诊断反馈，`skills refine` 触发服务端基于最近样本/反馈调用 LLM 产出更优技能包并落盘。

**当前版本**：以运行环境为准，执行 `./ai-sre version`（与源码中 `internal/cli/version.go` 的 `Version` 变量对齐）。**自升级默认基址**：内建 `http://192.168.56.11:9080/ft-api`（代码常量 `internal/cli/EmbeddedOpsfleetAPIBase`）；若设置 **`OPSFLEET_API_URL`** 则优先生效。未关闭时，**每次**执行子命令前（含 `uninstall k8s` 前）会拉 `GET .../api/k8s/deploy/cli/ai-sre/version` 比对，较新则覆盖本机 `ai-sre` 后 **re-exec** 同一命令（非 Windows）。`curl` 安装脚本写入的 **`~/.config/ai-sre/opsfleet_api_url`** 或 **`config.yaml` 的 `opsfleet_api_url`** 亦可作为覆盖来源。关闭：`OPSFLEET_NO_AUTO_UPGRADE=1`；关自动后仅提示：`OPSFLEET_UPGRADE_HINT=1`。**显式一次升级**：`sudo ai-sre upgrade -y`。

本仓库为 **单一 Git 仓库**：根目录 **CLI（ai-sre）**、**OpsFleet 本地执行器（`opsfleet-executor`）** 与 **OpsFleetPilot（Web + API）** 并排共存。**OpsFleetPilot** 包含 `ft-backend/`、`ft-front/`、`deploy/`、`ansible-agent/`；**`opsfleet-executor`** 与 `ai-sre` **共用同一套技能包与执行语义**（`analyze` / `ask` / `runbook` / `job run` 等），用于部署在**受管机**上本地执行。产品总览见 [`PRODUCT_DOC.md`](PRODUCT_DOC.md)，历史说明见 [`docs/opsfleet-README.md`](docs/opsfleet-README.md)。控制台构建：`make build-opsfleet`（产物 `bin/opsfleet-backend`、`dist/web/`）。

---

## 子命令一览

| 命令 | 说明 |
|------|------|
| `ai-sre analyze [topic]` | 故障诊断，`topic`：`kafka` \| `k8s` \| `nginx` \| `redis` \| `elasticsearch` \| `go-runtime`。**k8s**：本机有 `kubectl` 且可连集群时，先自动只读采集再调服务端 AI（默认两轮精炼）。`--pod` 可为 `pending`/`crashloop` 等场景名，或**具体 Pod 名**（将额外采集该 Pod 的 describe/events/logs 并优先参与结论）；可配 `--namespace` |
| `ai-sre analyze code <CODE>` | **错误码 → 根因卡片（0.5.1 新）**：把 `OPSFLEET_K8S_E_PAUSE_MISSING` / `OPSFLEET_DL_E_NETWORK` / `OPSFLEET_K8S_E_APISERVER_TIMEOUT` 等部署/运行错误码翻译成「根因 / 立即恢复一行 / 平台改进 + 文件路径」三段式输出，**不给排查清单**；`--list` 列出全部，`--detail "<paste of last log>"` 把现场原文带给服务端 |
| `ai-sre ask [question]` | 知识问答：默认经 OpsFleet `POST /api/ai/ask`（无需本机 api_key）；服务端失败且本机有凭据时回退本地 LLM+RAG |
| `ai-sre runbook [scenario]` | 生成 Runbook：默认经 `POST /api/ai/runbook`；回退逻辑同 `ask` |
| `ai-sre skills list` | 列出内置 + `--skills-dir` 合并后的技能包 |
| `ai-sre skills server` | 列出 OpsFleet 服务端注册的技能包（builtin + 生成中的 generated 版本） |
| `ai-sre skills status` | 通过 CLI token 同步当前可执行能力、订阅状态、执行模式和是否需要服务端计划单；不返回技能 YAML / Prompt |
| `ai-sre skills feedback --topic <t> -m "…"` | 把本次诊断的有效性反馈给服务端，进入下次精炼样本 |
| `ai-sre skills refine --topic <t>` | 让服务端基于最近 N 次样本 + 反馈调用 LLM 产出新版技能包（生成在 `OPSFLEET_AI_SKILL_DATA_DIR/generated/<topic>.yaml`） |
| `ai-sre analyze … --no-feedback` | 关掉本次诊断后的「是否帮到你」反馈提示（仅 TTY 下有效） |
| `ai-sre doctor` | 自检（凭据、tier、配额计数、技能/知识加载；**不调用 LLM**） |
| `ai-sre version` | 打印版本号 |
| `ai-sre help` | 帮助 |
| `ai-sre kafka diagnose <bootstrap-server>` | Kafka 极简快诊：优先用内置 Go 客户端直连采集（支持 `--config` 的 SASL/TLS）；失败再尝试 Kafka CLI，最后才回退 AI |
| `ai-sre redis diagnose <host:port>` | Redis 极简快诊：只读采集 INFO，定位连接拒绝、淘汰和连接压力 |
| `ai-sre elasticsearch diagnose <http-url-or-host:port>` | Elasticsearch 极简快诊：HTTP 只读 `_cluster/health` + `_cat/nodes`，区分单机/多节点与黄绿红风险；`--json` / `--ai` 与 `kafka diagnose` 语义一致；HTTPS 可用 `--insecure` |
| `ai-sre mysql diagnose <dsn>` | MySQL 极简快诊：只读采集连接、慢查询、线程与只读状态 |
| `ai-sre nginx diagnose` | Nginx 日志统计分析：状态码分布、Top 路径、P95 延迟、5xx/4xx 风险识别 |
| `ai-sre diagnose --pid <pid>` / `--name <name>` / `--pod <ns/pod/container>` | Go 程序智能运行时诊断：一条命令自动采样、分析 RSS/匿名内存/FD/线程/cgroup memory/CPU 风险，CLI 输出结论并上传到当前绑定账号的执行记录与进程观测页；采集器失败时仍输出 K8s 侧结论 |
| `ai-sre nginx uninstall` | 默认仅卸载由 `ai-sre service install` 写入本机状态的 Nginx；`-f/--force` 会强制检测并清理本机 Nginx 相关进程、包、容器、配置、日志和缓存 |
| `ai-sre service install --deploy-id <id> --token <token> --api-url <base>` | 基础服务安装执行器：从 OpsFleet 服务端拉取 Nginx / HAProxy / Redis / Kafka / MySQL / PostgreSQL / Elasticsearch 部署规格，执行安装、写配置、启动与健康检测，并回传步骤状态 |
| `ai-sre nginx update` | 在已通过 OpsFleet 服务部署安装过 Nginx 的目标机上，拉取服务端最新 Nginx 规格，重写配置并重启生效 |
| `ai-sre elasticsearch update` | 同上，作用于 Elasticsearch；自动复跑 system-tune（vm.max_map_count）、写 `elasticsearch.yml` + `jvm.options.d/heap.options`（包安装时另加 systemd drop-in；**binary** 方式则配置在 `install_prefix/config` 且 `ES_PATH_CONF` 指向该目录）、轮询 `_cluster/health` |
| `ai-sre elasticsearch uninstall` | 默认停服并移除 ai-sre 管理的 systemd/配置痕迹；`--purge-package` 在 **package** 下卸载发行版包，在 **binary** 下删除 `install_prefix` 目录；`--purge-data` 清理 data/log；`-f/--force` 端到端清理（容器、包、二进制目录、配置、数据、日志、apt/yum 仓库与 GPG 密钥） |
| `ai-sre k8s …` | 离线包下载、控制机 `install` / `cleanup` / `diagnose` 等（见 `ai-sre k8s --help`） |
| `ai-sre job run --machines <uuid,…> -c '…'` | **（0.5.10）** 经 OpsFleet `POST /api/job/execute` 在多台已在线 Agent 上批量执行命令或脚本；`--timeout`、`--wait` / `--max-wait`、`--print-console-url`（打开带 `?jobId=` 的控制台与同页「执行结果」对齐）；需 `OPSFLEET_API_URL` + 令牌。**`opsfleet-executor`** 亦含该子命令 |
| `ai-sre node tune time-sync …` | 与控制台「初始化工具 → 时间同步」等价的 CLI；本机构建 inventory + chrony / timesyncd playbook 并调用 `ansible-playbook`；缺失 ansible 时按 apt/dnf/yum 自动安装；未填 `--clients` 仅对 localhost 执行 |
| `ai-sre node tune sys-param …` | 与「系统参数优化」等价：sysctl + br_netfilter/overlay 内核模块 + ulimit + 关闭 swap；可用 `--sysctl key=value`（多次）扩展或 `--extra-only` 只用显式提供的项 |
| `ai-sre k8s diagnose` | 本机自检 K8s 常见抖动根因：**时钟跳变 / etcd 慢盘 / kubelet SandboxChanged / 预检缺项（swap/br_netfilter/sysctl）**；`--preflight` 只跑部署前预检，`--json` 输出可直接喂给 `ai-sre analyze k8s --issue instability` |
| `ai-sre upgrade` | 与 OpsFleet 对比版本后覆盖本机 `ai-sre` 二进制（需能访问上表基址） |
| `ai-sre uninstall k8s` | 在控制机 `root` 下用 Ansible `pre_cleanup` 全量清集群；**优先**本机 `/var/lib/opsfleet-k8s/last-bundle`（`install.sh` 预检后同步），无则再试拉 `ofpk8s1` 或 `--workdir` / `--force`（见 `ai-sre uninstall k8s --help`） |

`analyze` / `ask` / `runbook`：在默认内建 OpsFleet 基址（或 `OPSFLEET_API_URL`）可用时，**优先调用控制台公开接口** `POST /ft-api/api/ai/diagnose`、`/api/ai/ask`、`/api/ai/runbook`，**不要求本机配置 DeepSeek api_key**；仅当服务端不可用且本机已配置凭据时才回退到本地 LLM。回退若出现 HTTP 500，多为控制台未配置 **`OPSFLEET_AI_API_KEY`** 或无法访问 DeepSeek；在 **`/etc/opsfleet/backend.env`**（或 systemd 环境）补齐并 **`systemctl restart opsfleet-backend`**，或在运行 `ai-sre` 的机器配置 **`~/.config/ai-sre/config.yaml`** 的 **`api_key`** 作为回退。`analyze k8s` 在能执行 `kubectl` 时会把采集结果一并 POST 给服务端；自 **0.5.0** 起，**所有 topic** 都走「证据驱动」管道——`analyze kafka/redis/mysql/nginx/elasticsearch` 在用户传入 `-d bootstrap=…`、`-d target=host:port`、`-d dsn=…`、`-d access_log=…`、`-d base_url=…` 等参数时，会**就近调用本地** `ai-sre <topic> diagnose --json` 子命令采集指标，作为 `kafka_diagnose_json` / `redis_diagnose_json` / `mysql_diagnose_json` / `nginx_diagnose_json` / `es_diagnose_json` 一并 POST 给服务端，再由服务端提示词约束为「根因 + 证据摘录 + 修复要点」，减少泛泛命令清单；`--pod` 为**具体 Pod 名**时会额外附带该 Pod 的 describe/events/logs（含 previous）并优先参与推理。

**服务端诊断任务单与技能树坐标**：CLI 执行时会把现有命令归一化为 `intent`（如 `skill.k8s.workload.pod_pending`、`cap.diagnosis.k8s.workload`、`ops.incident_diagnosis.kubernetes.workload.pod_pending`），服务端以该坐标做订阅、计划单、审计、unlock 与资产沉淀；普通 CLI 命令不变。当 `analyze k8s` 本机没有可用 kubectl 证据、但当前 ai-sre 已绑定控制台 CLI token 时，CLI 会向服务端请求一次性只读任务单，只包含固定 argv 的 `kubectl get/describe/logs/version/config current-context` 采集动作，不下发 skill YAML 或提示词。TTY 会先预览命令并要求输入 `y`；CI/Ansible 等非 TTY 需加 `--yes`。采集结果会回传服务端绑定当前用户，并合并进本次诊断上下文；服务端按问题模式生成待审 skill asset 并给当前用户永久 unlock 该问题模式；超级管理员在控制台 **订阅与计费 → ai-sre 技能包 → 待审资产** 中通过/驳回，通过后写入 `generated/<topic>.yaml`（**默认与注册表已有同 topic 技能合并**，保留原 `analysis_steps` 并追加诊断沉淀）并参与后续 `AIDiagnose` 匹配。任务单除 **k8s**、**go_runtime** 外，亦支持 **redis / kafka / nginx / mysql / elasticsearch**（固定 `ai-sre <topic> diagnose … --json` 只读 argv，不下发凭据与技能 YAML）。控制台 **待审资产** 支持发布预览 diff、合并/独立发布、审核轨迹与下架；**运营统计** 页可查看诊断/AI 调用聚合并导出 CSV。审核前，已解锁用户仍可在诊断 prompt 中合并该资产的观察摘要（只读 overlay）。旧版本地生成 skill 默认不再写入或加载；如需兼容调试，显式设置 `OPSFLEET_ENABLE_LOCAL_SKILL_DRAFT=1`。

**自动迭代（仅 super_admin）**：控制台 **订阅与计费 → 自动迭代**（`/admin/auto-iterations`）管理任务、开关、审批、回滚与 GitHub/钉钉操作；所有 `/api/admin/auto-iterations/**` 仅 `super_admin` JWT 可访问。CLI 用户通过 `POST /api/cli/feedback/analyze`（须 CLI token + 指纹）提交反馈，响应仅含 `feedback_id`、`classification`、`need_iteration`、`user_message`、`next_action`，不含 agent/GitHub/webhook 等内部信息。Code Agent Worker 使用独立 `/api/code-agent/**` 机器 token，不能访问管理接口或批准高风险上线。

**技能树运营视图**：超级管理员可在控制台 **订阅与计费 → ai-sre 技能包 → 能力树** 查看运维能力树，节点会按 `draft/review/approved/deprecated` 汇总资产数量；点击任意大类、能力或叶子 skill 会切到资产审核列表，并按该树节点路径筛选对应沉淀资产。

**技能树数据库化（阶段 1）**：启动时把内置树 seed 到 `skill_tree_versions` / `skill_tree_nodes`（active=`builtin.skill-tree.v1`）。`ActiveSkillTree()` 优先读库，失败时 fallback 代码内置树；节点仅 **停用**（`status=disabled`），不物理删除。管理 API：`GET/POST /api/admin/skill-tree/versions*`、`/api/admin/skill-tree/nodes*`（草稿、发布、编辑、排序）。`GET /api/cli/sync` 返回 `tree_rev`/`tree_source` 与能力坐标，不下发 YAML/Prompt。用户 unlock 粒度为 **`skill_key` + `problem_key`**。

**商业化（阶段 3–4）**：`skill_commercial_products` + `skill_product_node_bindings`（默认 seed `skillpack.k8s/kafka/redis/nginx/mysql/elasticsearch`、`pack.k8s_delivery`、`pack.runtime_observe`）。`GET /api/cli/sync` v2 含 `policy_rev`、`commercial_product_key`、`denial_reason`、`parameter_templates`、`upgrade_required`；CLI `ai-sre skills status [--refresh] [--json]` 本地缓存 10 分钟；`analyze` / 诊断任务单执行前按技能树校验 `can_execute`（不缓存 YAML/Prompt）。

**Go runtime 智能诊断**：推荐使用 `ai-sre diagnose --pid <pid>`、`ai-sre diagnose --name <name>` 或 `ai-sre diagnose --pod <pod|namespace/pod|namespace/pod/container>`（`--pid-name` 仍兼容）。若当前 CLI 已绑定平台，会先校验 CLI token、机器指纹与 `feature.runtime_observe` 权益；平台不可用、鉴权失败或离线时，仍允许执行本地只读采集，并在本机有 LLM 凭据时使用本地 AI 回退，否则用内置规则总结。通过后默认采样 4 次、间隔 10 秒，读取 `/proc/<pid>/status`、`smaps_rollup`、`stat`（含 utime/stime）、`limits`、`fd`、`maps` 以及 cgroup v1/v2 的 memory/cpu 指标，判断 RSS、匿名内存、FD、线程数、cgroup memory/CPU throttling 与趋势风险。`--name` 会扫描 `/proc` 并优先选择 Go binary；`--pod` 会通过 kubectl 定位 Pod 所在节点与容器 ID，创建临时只读 collector 读取宿主机 procfs/cgroup，结束后自动清理。若采集器镜像拉取失败（如离线集群无 busybox），命令仍会基于目标 Pod 与采集器事件给出诊断结论，并提示设置 `OPSFLEET_GO_RUNTIME_COLLECTOR_IMAGE` 或在节点上用 `--pid`。在线且具备权益时，诊断结果会自动上传到控制台「执行记录」与「运行时诊断」（根因/证据可按 Markdown 渲染展示，并可删除历史报告），普通用户只能看到自己的记录。旧入口 `ai-sre diagnose go-process ...` 继续保留，用于离线 fixture 或手动参数调试。

**诊断结束后的反馈闭环**：TTY 下 `analyze` 答完会追加一行 `本次诊断是否帮你定位了根因？输入 y / n / 自由备注；空行跳过。`；按需写一行后将通过 `POST /api/ai/skills/feedback` 落到服务端 `feedback/<topic>.jsonl`，参与下次 `ai-sre skills refine`。非 TTY、`-o json` 或显式 `--no-feedback` 会自动跳过。

**部署错误码 → 根因卡片（0.5.1 新）**：所有 K8s 部署/下载失败的 ansible / install.sh / bootstrap.sh 都会在 stderr 输出一行机器可读 `[ERROR-CODE] OPSFLEET_* …`：
- `OPSFLEET_K8S_E_PAUSE_MISSING` — containerd 节点缺 `registry.k8s.io/pause:3.10`，静态 Pod sandbox 拉不起来 → 已通过 `ansible-agent/roles/pause_preload` 在所有节点离线 `ctr import` 修复；
- `OPSFLEET_K8S_E_APISERVER_TIMEOUT` — `wait_apiserver.yml` 180s 没等到 6443，pre_tasks 会先 emit 上述细分子码，避免裸 timeout 让运维迷路；
- `OPSFLEET_DL_E_NETWORK` / `OPSFLEET_DL_E_CHECKSUM` — `download-with-progress.sh` 抓 mirror 失败；
- `OPSFLEET_K8S_E_PLAYBOOK_*` — install.sh `run` wrapper 在每个 playbook 失败时按 yml 名生成，可直接 `ai-sre analyze code <CODE>` 让服务端给根因。
- `OPSFLEET_K8S_I_RELAY_ROUTE_APPLIED` — 信息码：选阿里云源时控制机探测公网 tarball 失败后已追加 relay overlay（非致命）；制品中转与运维见 [`deploy/k8s-mirror/README.md`](deploy/k8s-mirror/README.md)。
- `OPSFLEET_K8S_E_EXECUTOR_KUBECTL_BIN_MISSING` — 部署机缓存里尚无解压出的 `server/bin/kubectl`（多半是尚未跑完 resources，或 playbook 顺序被截断）。

控制台：登录后在**顶栏**（「安装 ai-sre」左侧）点击 **错误码**进入（工作台 **`/app/help/error-codes`**；管理端 **`/admin/help/error-codes`**；旧 **`/help/error-codes`** 会重定向）；可浏览与搜索同源目录；接口仍为 `GET /ft-api/api/ai/error-codes`，单条根因卡 `POST /ft-api/api/ai/error-codes/analyze {code,detail}`；客户端等价命令是 `ai-sre analyze code <CODE> [--detail "…"]`，输出三段式：根因 / 立即恢复一行 / 平台改进+文件路径（不给排查清单）。**壳层布局**：左侧固定侧栏 + 右侧主列；顶栏为面包屑与快捷入口；主内容区外侧留白由 `--layout-content-gutter` 与业务页 `page-shell` 分层。**前端样式**：以 **Element Plus 默认主题与组件外观** 为主（`element-plus/dist/index.css`），全局仅保留布局尺寸与少量兼容变量（`src/style.css`），不再整体覆写按钮/表格/卡片的圆角与阴影。**前端页头与纵向留白约定**见仓库 **`.cursor/rules/ft-front-compact-layout.mdc`**。

**错误码扩展（开发门控）**：在 Cursor 中为代理准备 **`.cursor/skills/error-code-development-gate/SKILL.md`**——凡改离线安装/ansible/镜像站等易失败路径时，须同步 **emit `[ERROR-CODE]`**、`ft-backend/skills/builtin/error_codes.yaml` 根因条目与（如适用）`README`/`API`/`analyze code` 自检，保证能力覆盖面随迭代扩大。

**部署机 kubectl 与 kubeconfig（0.5.2）**：执行 `sudo bash install.sh` 的那台机上，在 **`playbooks/kubectl.yml`** 成功后 **`roles/kubectl/tasks/control_host_cli.yml`** 会安装与离线包 **`kubernetes-server` 同版本的** `/usr/local/bin/kubectl`，并把第一份控制面的 admin kubeconfig 下发到 **`$HOME/.kube/config`**（通常为 root：`/root/.kube/config`）。`/etc/profile.d/opsfleet-kubectl.sh` 在未导出 `KUBECONFIG` 时优先使用该文件。**建议始终用 root 跑 install**，与 inventory 默认 `ansible_user=root` 一致；若以普通用户手工跑 `ansible-playbook`，请自行把 **`/root/.kube/config`** 拷到当前用户或使用 `sudo kubectl ... --kubeconfig /root/.kube/config`。

**extension-apiserver-authentication（0.5.3）**：在 **`kube-apiserver` 就绪后、`kube-controller-manager` 安装前**，`install.sh` 会跑 **`playbooks/extension_apiserver_authentication.yml`**，确保 **`kube-system/extension-apiserver-authentication`** 的 **`client-ca-file`** 与 **`requestheader-client-ca-file`** 含有效 PEM（与集群 CA 一致），避免 controller-manager 日志里 **`missing content for CA bundle ... requestheader-client-ca-file`** 并反复重启。已装集群若仍见该错误，可对照该 ConfigMap 是否缺键或为空，必要时用 **`/etc/kubernetes/pki/k8s_ca.crt`**（或你环境中的集群 CA 路径）补全后让 kubelet 重建 controller-manager Pod。

**Calico 镜像预拉（0.5.4+）**：选 **Calico** 时，`playbooks/k8s_addons.yml` 会先在 **`k8s_cluster` 全部节点**用 **`ctr -n k8s.io images pull`** 带重试拉齐三镜像；默认前缀 **`quay.io`**，可在 **`inventory/group_vars/all.yml`** 设 **`calico_image_pull_registry: quay.m.daocloud.io`**（DaoCloud 等），预拉成功后 **`ctr images tag`** 回 **`quay.io/calico/...`** 与上游 Calico YAML 一致，再 **`kubectl apply`** 并 **`kubectl wait` calico-node Ready**（失败即中止）。仍失败时 stderr 含 **`[ERROR-CODE] OPSFLEET_K8S_E_CALICO_PREFETCH_FAILED`**；纯离线需制品站 **`ctr import`**（见 **`ft-backend/skills/builtin/error_codes.yaml`**）。

**下载进度条（0.5.0 新）**：在 **TTY 交互终端** 下，客户端涉及下载二进制 / 离线包的路径可输出**进度条 + 已下/总量 + 速度 + ETA**（非 TTY 自动退化为摘要）：
- Go 侧：`ai-sre upgrade`、`ai-sre k8s install`、`ai-sre k8s download-bundle` 等使用统一的 `progressReader`（TTY 下绘 `[====-----] 42.3% 120MiB/284MiB 25.3MiB/s eta 7s`，非 TTY 自动退化为每秒一行可解析的摘要）。
- 服务端动态生成的 `install-ai-sre.sh`、`bootstrap.sh`：TTY 下 `curl --progress-bar` / Python 内置流式进度；非 TTY 退化为静默 + 完成后摘要。
- `ansible-agent`：`group_vars` 为 `download-with-progress.sh` 传入 `OPSFLEET_NO_PROGRESS=1`（`curl -sS`），stderr 仍为起止时间与完成一行摘要。
- 其它路径静音：`OPSFLEET_NO_PROGRESS=1`（如 CI）。

**服务端自迭代技能注册表**：服务端为每个 topic（`k8s` / `go_runtime` / kafka / redis / mysql / nginx / elasticsearch 等）内嵌一份 YAML 技能包（`ft-backend/skills/builtin/*.yaml`）。`analyze` 与 **`ai-sre diagnose`** 成功后均异步追加脱敏样本到 `samples/<topic>.jsonl`。配置在 **`ft-backend/conf/config.yaml`** 的 `skills.auto_refine`（实验室默认可 `enabled: true`）；环境变量 `OPSFLEET_SKILL_AUTO_REFINE_*` 可覆盖。LLM 密钥用 **`OPSFLEET_AI_API_KEY`**。规范见 **`.cursor/skills/backend-configuration/SKILL.md`** 与 `deploy/config.production.example.yaml`。

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

`analyze` 常用 flag：`--lag`、`--topic`、`--pod`、`--namespace`、`--issue`、`--code`、`--upstream`、`--latency`、`-d`/`--set key=value`、`--yes`（确认服务端只读诊断任务单）。

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
./ai-sre kafka diagnose 'b1:9092,b2:9092,b3:9092' --config ./client.properties
./ai-sre redis diagnose 10.0.0.2:6379
./ai-sre elasticsearch diagnose 127.0.0.1:9200
./ai-sre elasticsearch diagnose https://es:9200 --insecure --user elastic --password '***'
./ai-sre analyze elasticsearch -d base_url=http://127.0.0.1:9200
./ai-sre mysql diagnose 'user:pass@tcp(10.0.0.3:3306)/mysql?timeout=5s'
./ai-sre nginx diagnose --access-log /var/log/nginx/access.log --tail 10000
./ai-sre diagnose --pid "$(pgrep -n my-go-service)"
./ai-sre diagnose --name my-go-service
./ai-sre diagnose --pod default/api-0/app
./ai-sre diagnose -o json --pod default/api-0/app
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
./ai-sre analyze k8s --pod pending
./ai-sre analyze k8s --pod kube-controller-manager-k8s-master-0 -n kube-system
./ai-sre ask "kafka lag 高怎么办"
./ai-sre runbook "pod频繁重启"

./ai-sre skills list
./ai-sre skills server                           # 看服务端注册了哪些 builtin/generated
./ai-sre skills feedback --topic k8s -m "事件链条没串好"
./ai-sre skills refine --topic k8s --hint "对前一轮答案不准确"
./ai-sre -o json analyze kafka --lag 1
./ai-sre analyze kafka -d bootstrap=broker1:9092 -d topic=orders
./ai-sre analyze redis -d target=127.0.0.1:6379
./ai-sre analyze mysql -d dsn='user:pass@tcp(127.0.0.1:3306)/db?charset=utf8mb4'
./ai-sre analyze elasticsearch -d base_url=http://127.0.0.1:9200
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
- `OPSFLEET_SKIP_REMOTE=1`：不把 OpsFleet 当作 API 基址（不调服务端 `analyze`/`ask`/`runbook`、也不用内建基址做自升级比对）。**仅推荐** `scripts/remote-e2e.sh` 无凭证负例或离线单测；生产环境勿随意开启。

远程需已安装 Go；目标机密钥仍放在 `~/.config/ai-sre/`（root 用户即为 `/root/.config/ai-sre/`）。

冒烟脚本（在**已部署的远程目录**或本地均可测）：

```bash
SHORT=1 bash scripts/remote-e2e.sh   # vet/test/build/version/doctor/skills list/无凭证负例
bash scripts/remote-e2e.sh         # 含 LLM（需有效 api_key）
```

**发布总入口（Cursor Agent）**：**`.cursor/skills/release-deploy/SKILL.md`** → 再按场景执行 **`.cursor/skills/ai-sre-ship/SKILL.md`**（CLI 同步与 push）、**`.cursor/skills/opsfleetpilot-ship/SKILL.md`**（实验室全栈）、**`.cursor/skills/production-deploy/SKILL.md`**（生产 `opsfleetpilot.com`，保留生产 Nginx 与 `config.yaml`）、**`.cursor/skills/k8s-offline-deploy-test/SKILL.md`**（K8s 离线）。**README 须在 push 前、发布后保持更新**。**任意本仓文件变更**还应遵循 **`.cursor/rules/monorepo-release.mdc`**（`alwaysApply`）。

---

## 标准约定（同仓）

- **OpsFleet 后端仅使用** `ft-backend/conf/config.yaml`（由 `deploy/config.production.example.yaml` 复制编辑）；仓库内**不得**再保留根路径 `ft-backend/config.yaml` 等重复配置，以免误用。
- **勿提交**：本机编译产物（`ai-sre`、`bin/`、`dist/`、`ft-backend/opsfleet-backend`）、`node_modules`、vim `*.swp`（见根目录 `.gitignore`）。
- **CLI 凭据**：仅用 `~/.config/ai-sre/`，与 OpsFleet 的 PostgreSQL/JWT 配置无关。
- **服务端 AI 回退**：若要启用无本地 key 的 `analyze` 自动回退，请在 OpsFleet 后端环境配置 `OPSFLEET_AI_API_KEY`（可选 `OPSFLEET_AI_BASE_URL`、`OPSFLEET_AI_MODEL`）。
- **服务端技能数据目录**：可选 `OPSFLEET_AI_SKILL_DATA_DIR`（默认尝试 `/var/lib/opsfleet/ai-skills`，否则 `./data/ai-skills`）。诊断样本会以 JSONL 形式追加到该目录的 `samples/<topic>.jsonl`，反馈到 `feedback/<topic>.jsonl`；`ai-sre skills refine` 产出的新版技能包落到 `generated/<topic>.yaml`，旧版本归档到 `generated/<topic>.history/<ts>.yaml`。**全栈部署脚本会自动 mkdir + 写入 backend.env。**

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

**实验室虚拟机 `root@192.168.56.11`（本地 root 免密）**：与 **ai-sre** `deploy-remote.sh`、**OpsFleet** `deploy-opsfleet-remote.sh` 使用**同一默认主机**；可在该机上另部署 **K8s 内网制品站**（`deploy/k8s-mirror/README.md`，持久目录默认 `/var/lib/opsfleet-k8s-mirror`）。全栈部署脚本**首次**可在远端创建 **`/etc/opsfleet/backend.env`**，设置 **`OPSFLEET_K8S_MIRROR_BASE_URL`**（默认 `http://192.168.56.11`），供控制台 **「制品目录」** 页（路由 **`/admin/k8s-mirror`**；旧路径 **`/admin/service/k8s-mirror`** 会重定向）代理展示 `manifest.json` 与 SHA512。**发布顺序**见 **`.cursor/skills/release-deploy/SKILL.md`** 与 **`.cursor/rules/monorepo-release.mdc`**：**release-deploy** 总清单 → **ai-sre-ship** → 若改 OpsFleet 则 **opsfleetpilot-ship** → 若改 K8s 离线/制品 则 **k8s-offline-deploy-test** → 最后 **`git push`**。

本地开发：在 `ft-backend` 配置 `conf/config.yaml` 后 `go run .`；在 `ft-front` 执行 `npm install && npm run dev`（Vite 代理 `/ft-api`）。

**OpsFleet 控制台登录**：数据库迁移脚本初始化时默认用户名为 **`admin`**、密码为明文 **`password`**（bcrypt），角色为 **`super_admin`**。生产环境请修改；忘记密码可在数据库所在机执行 **`ft-backend/database/reset_admin_password_pg.sql`**（将 **`admin`** 重置为 **`123456`** 并确保角色为 `super_admin`，与当前运维约定一致）。登录页在默认配置下会要求**一次性算术验证码**（防撞库/脚本爆破，与 `POST /api/auth/login` 的限流并存）；可在 **`ft-backend/conf/config.yaml`** 的 **`security.disable_login_captcha: true`** 关闭（纯内网场景）。**反代与限流**：Nginx 模板已传 **`X-Forwarded-For`**；后端信任私网/回环上游并按真实客户端 IP 限流，避免所有请求在 Gin 侧表现为 **`127.0.0.1`** 而误触 **429「请求过于频繁」**（此前易被误认为「登录拒绝」）。**CORS**：浏览器对 API 的 `POST` 会带 **`Origin`（须与地址栏完全一致，含 `http`/`https` 与端口）**；若未写入 **`ft-backend/conf/config.yaml`** 的 **`security.cors_allowed_origins`**，中间件会直接 **403**，页面常显示「拒绝访问」且**不会执行登录逻辑**——请把实际访问控制台用的完整 Origin（例如 `http://192.168.56.11:9080`、`http://内网IP:9080`）加入该列表并 **`systemctl restart opsfleet-backend`**。**自助注册**：`GET /api/auth/public-options` 与注册页受 **`security.disable_public_registration`** 控制；注册账号默认角色为 **`user`**，**`admin` / `super_admin`** 仍仅由管理员在用户管理页分配，其中 `super_admin` 只能由现有 `super_admin` 分配或撤销。

**前端路由与功能包订阅**：全体登录租户成员（含 `user`）默认进入 **`/admin/...`** 并使用同一套侧边栏（**概览 → 工作负载（应用服务 / Kubernetes / Linux 主机）→ 制品目录（独立一级，路由 `/admin/k8s-mirror`）→ 出口代理 → 可观测性 → 任务（作业中心 / 执行记录；控制台启用 K8s 交付时在「执行记录」页内另含「K8s 集群」标签，URL 可用 `?tab=k8s`）→ 安全 → 数据 → 工具** 等信息架构；短名称与 **`ft-front/src/router`** 标题一致）；入口默认可见，付费状态通过菜单「订阅」标签、页面提示、按钮锁定和 Paywall 弹窗呈现，真正安全边界始终在后端/CLI/Agent 执行校验。路由级禁区：**用户**（`/admin/user/list`）、**权限** 仅限 **`admin` / `super_admin`**；**订阅与计费** 仅限 **`super_admin`**。**`super_admin`** 永久豁免订阅计费，并独占 **订阅与计费** 与手动授予权益。当前功能包包括 **`pack.k8s_delivery`**、**`pack.node_ops`**、**`pack.monitoring`**、**`pack.backup_performance`** 与 **`skillpack.k8s/kafka/redis/nginx/mysql/elasticsearch`**；旧功能键 **`feature.k8s_ops/service_ops/infra_ops/advanced`** 会兼容映射到对应功能包。对外接口：`GET /ft-api/api/billing/capabilities`（Web/CLI/菜单/按钮统一能力清单）、`GET /ft-api/api/billing/packages`（档位列表）、`GET /ft-api/api/billing/me`（账号订阅与权益）、`POST /ft-api/api/billing/checkout-session`（Body 使用 `pack_key`，兼容 `package_id`）、`POST /ft-api/api/billing/stripe/webhook`。**`GET/PUT /ft-api/api/admin/billing/features`** 可由 `super_admin` 配置 `visible_enabled`、`execution_enabled`、`billing_enabled` 与 Stripe Price。默认计费开关关闭以兼容旧部署。详细设计见 **`docs/advanced-feature-billing-design.md`**。

**概览**：控制台不把「业务侧上报托管机器」做成全租户机器大盘；概览中的机器类 KPI 不再从 `Machine` 表聚合。**仅 `super_admin`** 在首行 KPI 下方**另起一行**展示控制台机 **CPU、内存、磁盘** 三卡（服务端即时采样）；其它角色不展示该区块。首行 KPI 从左到右：**`console_admin` 为 K8s 集群**，否则为 **业务服务（台账）**（脚注汇总各运行态计数），其次 **进行中作业**、**近 24h 执行**；**`super_admin`** 在行末追加 **用户 / 审计日志**（最多四列）；同行卡片等高对齐。**近 24h 执行** KPI 与 **近 24h 执行健康** 条展示新建条数、成功/已取消/失败及终态失败率，并按 **source** 汇总 `cli` / `k8s` / `job` 条数（与列表同一角色可见范围；`GET /api/dashboard` 的 `platformSummary.executionsSuccessLast24h`、`executionsCancelledLast24h`、`executionsFailedLast24h`、`executionsBySourceLast24h`）。下方列表区各卡标题均为 **「最近 …」** 句式；**最近 Linux / 业务服务** 与 **最近执行记录** 等表格卡**单列纵向排列**（整行宽），表头与卡身统一白底与固定阴影，避免双列并排时「一卡有悬停阴影、一卡仅描边」的视觉不一致；表头内边距与 KPI 区一致。**`services` 台账**若旧数据长期处于 **部署中** 且超过 **24 小时**无任何行更新，在请求 `GET /api/dashboard`、服务列表/详情或再次 `POST /api/service/deploy` 前会自动将状态纠为 **已停止**，以免与真实任务脱节（与 **`service_deployments`** 安装向导流水无关）。**最近 Linux / 业务服务** 表列含 **服务名称**、**功能名称**（`services.config` 中 `productName` / `service_key` 等约定键，否则 `description`，再否则按 `type` 给出缺省文案）、**资源**（镜像、关联 `machine_id`、端口等摘要）、副本、状态与更新时间。

**执行记录**：在侧栏 **任务 → 执行记录**（`/admin/execution-records`；工作台 **`/app/execution-records`**）统一持久化展示 `ai-sre` CLI、AI 调用、安装 ai-sre、K8s 一键安装 / bootstrap、初始化工具复制脚本、作业中心任务等执行历史；**控制台启用 K8s 交付能力（`feature.k8s_delivery`）时**，同一页以 **「K8s 集群」** 标签展示已登记集群列表（原独立「集群」菜单并入此处，可用 **`?tab=k8s`** 直达）。记录包含来源、账号、功能包/技能包、目标主机/资源、命令或脚本摘要、开始/结束时间、退出码、输出摘要、执行效果 JSON、回滚能力与回滚状态。目标机侧上报为 **best-effort**：脚本或 CLI 能连到 OpsFleet API 时会调用 `/api/execution-records/report/*` 写入开始、事件和结束状态；上报失败不会改变原命令退出码。**角色与列表**：`admin` / `super_admin` 可见租户内全部记录；**`user`** 仅可见本人相关记录（`created_by` / `trigger_user` 与登录用户名一致），且限于 **失败/已取消**、**AI 类子命令**（如 `analyze` / `ask` / `runbook`）或 **安装 ai-sre**。AI 记录只保存摘要、上下文 key/大小、技能包、额度和权益来源，不保存敏感全文。凭 **K8s 安装引用**（invite id + token）上报时，服务端会把 **`created_by` / `trigger_user`** 回填为邀请创建者；也可在目标机设置 **`OPSFLEET_EXECUTION_USERNAME=<控制台用户名>`** 与 **`OPSFLEET_API_URL`**、**`OPSFLEET_EXECUTION_TOKEN`** 一并上报以便与账号对齐。页面回滚采用保守策略：同目标或同资源在该记录之后存在成功执行时，会先提示关联影响；可自动或半自动回滚的记录会创建一条关联的 rollback 记录，不可验证恢复的命令会显示人工回滚建议。

**ai-sre 安装绑定**：公开安装入口 `GET /ft-api/api/k8s/deploy/install-ai-sre.sh` 只安装/升级 CLI 并写入 `opsfleet_api_url`，不绑定账号。顶栏 **安装 ai-sre** 会先调用 `POST /ft-api/api/me/cli/install-session` 生成 15 分钟有效的一次性命令，脚本通过 `X-OpsFleet-Install-Token` 获取并在目标机上采集 `/etc/machine-id`（或 dbus machine-id）、hostname、OS、arch 后计算机器指纹，`POST /ft-api/api/cli/install-bind` 换取专用 CLI token。服务端只保存 token 哈希和指纹哈希；安装成功后本机写入 `~/.config/ai-sre/opsfleet_token`、`opsfleet_binding_id`、`opsfleet_fingerprint` 与 `opsfleet_username`。后续 `ai-sre analyze/ask/runbook` 会带 `Authorization: Bearer <cli_token>`、`X-OpsFleet-CLI-Fingerprint` 和 `X-OpsFleet-CLI-Version`，按绑定账号识别订阅与 AI 免费额度；token 无效或指纹不匹配返回 401，不降级匿名。

**Kubernetes 部署（推荐）**：在 **Kubernetes 部署** 页按折叠配置项填写安装预检、**控制平面部署方式**（二进制 + systemd 或 kubelet **静态 Pod**）、基础集群信息、节点、核心组件、网络、存储、高级配置与部署确认，不再使用「下一步」线性向导。默认仅展开 **安装预检**，其它配置项默认折叠；标题区域右侧提供 **安装或升级 ai-sre**（已安装则覆盖）：`curl -fsSL '<publicApiBase>/api/k8s/deploy/install-ai-sre.sh' | sudo bash`（会写入 `~/.config/ai-sre/opsfleet_api_url` 供后续**自动比对升级**；**不**写入令牌，服务端 AI 仍按匿名 IP 限额）。`ai-sre` 侧优先读环境变量 **`OPSFLEET_TOKEN`**，否则读 `opsfleet_token` 文件；**`OPSFLEET_API_URL`** 未设置时亦会读取 `opsfleet_api_url` 文件后再回退内建默认基址。全栈机执行 **`./scripts/deploy-opsfleet-remote.sh`** 时，远端 **`build-all.sh` 会生成 `bin/ai-sre`**，并在 **`/etc/opsfleet/backend.env`** 写入 **`OPSFLEET_AISRE_BINARY_PATH=<仓库>/bin/ai-sre`**，并在 **`build-all.sh` 成功交叉编译 `bin/ai-sre.arm64`** 时写入 **`OPSFLEET_AISRE_BINARY_PATH_ARM64`**（供 aarch64 控制机 `install-ai-sre.sh` 按 `uname -m` 拉取 ARM ELF；`GET .../cli/ai-sre?arch=` 与文件 ELF 不一致时返回 **400** 而非下发错误架构）（**systemd 优先于 config.yaml**），故每次发布控制台分发的 CLI 与源码一致；仅当**未用该脚本部署**时，才需在 `conf/config.yaml` 配置 **`opsfleet.ai_sre_binary_path`**。集群安装：**①** `sudo ai-sre k8s install 'ofpk8s1.…'`；**②** `curl -fsSL '<publicApiBase>/api/k8s/deploy/bootstrap.sh' | sudo bash -s -- 'ofpk8s1.…'`（需 `python3`）；**③** zip 解压后 **`sudo bash install.sh`**。**控制机须能免密 SSH 各节点 `root`**。离线配置里若 worker 填了与 master 相同 IP，后端会在生成 inventory 时自动去重（master 本身已安装 kubelet 并注册为 Node，无需重复声明）。同一角色列表内（master 或 worker 自身）仍不允许重复 IP。**卸载**（在曾安装过并记录了引用的控制机上）：`sudo ai-sre uninstall k8s` 或 `sudo ai-sre k8s cleanup 'ofpk8s1.…'`。

**部署前的节点初始化（可选但推荐）**：Kubernetes 部署页最前面的 **安装预检** 配置项合并展示「离线安装必读」与「环境预检」，用于提前确认 root 免密 SSH、NTP、br_netfilter、sysctl、swap、节点架构与主机名等风险；也可手动进入 **工具 → 节点初始化**（**`/admin/init-tools`**，旧 **`/init-tools`** 会重定向；侧栏为 **工具** 分组下入口；**单页、内容区满宽、4 列固定卡片**）优化节点环境，避免 calico-node / coredns 在 NTP 漂移、br_netfilter 缺失等情况下反复 Killing。所有优化项以紧凑卡片形式集中：**时间同步 / 系统参数优化 / 系统安全加固 / 磁盘分区优化**。卡片大小一致、始终一行 4 个：顶部标题与底部操作按钮固定，配置项较多时仅卡片中部区域纵向滚动；卡片不出现横向滚动条，页面在内容未溢出时也不出现整页滚动条。每张卡片自包含 **目标节点（多选）**、**系统类型**（Ubuntu/Debian/CentOS/Rocky/RHEL/openEuler/Kylin/其它 Linux）与对应工具的关键参数：

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

**机器与作业**：已移除「机器管理」独立页面；后端 `/api/machine` 仍用于台账与拓扑。**作业中心**（`/admin/job/center`、`/app/job/center`）支持：**目标文本框**（UUID / IP / 名称按在线快照自动解析，执行与生成脚本前会再即时对齐）、折叠 **「执行选项」（超时秒数等）**、**命令或脚本**，以及 **一键脚本**（推荐 `ai-sre job run --machines … -c … --print-console-url`）。页面 **URL `?jobId=<uuid>`** 或 **`ai-sre job run --print-console-url` 输出的链接** 可把同一任务的各机输出拉回「执行结果」区。API：**`POST /api/job/execute`**（JSON：`machine_ids`、`command`、`timeout`，超时由后端夹在 10～3600 秒）；**`GET /api/job/result/:jobId`** 轮询子任务。**CLI**：`ai-sre job run --machines uuid1,uuid2 -c '…' [--timeout 120] [--wait] [--max-wait 15m] [--print-console-url]`（需 `OPSFLEET_API_URL` + `OPSFLEET_TOKEN`/文件令牌）；**executor** 亦注册同一子命令。详见 [`PRODUCT_DOC.md`](PRODUCT_DOC.md)。

---

## 安全说明

密钥仅从本机文件读取，勿将真实 `api_key` / `config.yaml` 提交到 Git。建议目录权限 `700`、密钥文件 `600`。

---

## Git 远程

```text
git@github.com:antpan-san/ai-sre.git
```
