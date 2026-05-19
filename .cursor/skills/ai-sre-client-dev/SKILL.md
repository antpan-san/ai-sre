---
name: ai-sre-client-dev
description: >-
  ai-sre CLI client development norms: mandatory fast auto-upgrade, diagnosis contract,
  post-AI skill pack enhancement review, version bump rules. Use when changing internal/cli,
  check/ask/runbook, upgrade flow, or client install/self-update behavior.
---

# ai-sre 客户端开发规范

## 最强铁律：零配置自动完成（写死，高于一切客户端约定）

**产品要求**：用户**不得**为完成主路径而设置环境变量、不得先手动 `upgrade`、不得 `export OPSFLEET_API_URL` / `AI_SRE_*` 才能用功能指令。一条命令即完成：**先自动探测版本并升级（如需）→ 再执行业务**。

| 条目 | 要求 |
|------|------|
| 版本 | 每条功能指令前 `PersistentPreRunE` → `tryAutoUpgradeInPlace`；有新版则下载 + `exec` 同 argv，**用户无感** |
| API 绑定 | `resolveOpsfleetAPIBaseStrict`：install 记录 > `OPSFLEET_API_URL` > 内嵌实验室；**冲突时自动采用 install**，stderr 一行说明，**不报错阻断** |
| 可达性 | `fetchRemoteVersionFast` + `collectOpsfleetAPIBaseCandidates`：**按序探测** install / env / 实验室 / 生产，直到 `GET .../version` 成功 |
| 诊断目标 | `check redis` 等：默认本机 `127.0.0.1`；`check redis <addr>` 支持任意 host:port（见 `check_target.go`） |
| 禁止 | 文档/实现要求用户「先设环境变量」「先 upgrade 再 check」；禁止混环境时直接 `return err` 阻断 |
| 实现 | `opsfleet_env.go`、`upgrade.go`、`check_target.go` |
| 唯一豁免 | `version` / `upgrade` / `help` / `completion`、`-h`/`--help`、显式 `--no-auto-upgrade` / `OPSFLEET_NO_AUTO_UPGRADE=1` |
| 验收 | 旧版二进制仅执行 `ai-sre check redis` → 自动升到服务端版本并跑通；`OPSFLEET_API_URL` 与 install 冲突时仍成功且仅一行提示 |

环境变量仅为**可选高级覆盖**（运维/Debug），**不是**主路径依赖。调试：`OPSFLEET_AUTO_UPGRADE_VERBOSE=1`；隐藏检查失败提示：`OPSFLEET_AUTO_UPGRADE_QUIET=1`。

## 铁律：每次功能指令前必须做版本探测与自动升级

与「零配置」一致；实现细节见下文「自升级」。**禁止**静默跳过探测；**禁止**要求用户额外执行 `ai-sre upgrade`。

## 精简参数原则（强制）

| 原则 | 说明 |
|------|------|
| 主路径 | `ai-sre check <topic>`（目标由 `smartDefaultCheckTarget` + 本机回退自动填充） |
| 可选 | `ai-sre check <topic> <target>` 单参数覆盖 |
| 环境变量 | **不得**作为文档中的必填步骤；仅高级覆盖 |
| `-d` / flag | K8s、密码、`--yes` 等高级场景；不得覆盖用户已 `-d` 的值 |

实现：`applyCheckTargetContext`、`smartDefaultCheckTarget`（`check_target.go`）。

## 诊断契约（强制）

所有 `check <topic>`：**先 probe 采集 → 服务端 AI → 复核 → 纯文本输出**；禁止 AI 让用户手工采集。详见 **`.cursor/skills/ai-sre-diagnosis-contract/SKILL.md`**。

| 环节 | 文件 |
|------|------|
| 采集 | `redis_probe.go`、`check_redis.go` |
| 复核 | `ft-backend/handlers/ai_diagnose_review.go` |
| 展示 | `internal/cli/diagnose_output_format.go`（`check` 默认 `-o text`） |

**输出格式**：终端必须为纯文本小节（`【根因与触发条件】` 等），不得向用户展示 Markdown `##`。

## AI 调用后的技能包增强审查（强制）

**产品要求**：ai-sre 的 AI 调用必须反哺技能包。凡是 CLI 在**已支持的 topic / skill pack** 下调用 AI（`check` / `ask` / `runbook` / 已弃用 `diagnose` 别名，以及安装恢复触发的 `install` topic），**完成响应后**都必须审查当前技能包是否需要增强。目标是让**下次相同命令、相同问题、相似证据**尽量不再依赖 AI，从而降低 AI 成本并提升稳定性。

**原则**：AI 调用不是终点，而是技能包沉淀入口。

### 必做审查（每次 AI 成功返回后）

开发者或 agent 必须判断（**不得**再次调用 AI 做审查）：

| 审查项 | 动作 |
|--------|------|
| 根因是否可确定性化 | 能否写成 local rule / pattern / error code |
| 证据是否不完整 | 是否应扩展 `probe`、`gatherTopicEvidence`、`diagnostic plan` |
| skill YAML 是否不足 | `analysis_steps`、`extra_guidance`、输出小节是否缺约束 |
| 是否让用户手工采集 | 若 AI 或初稿出现「请执行 top/redis-cli/kubectl」→ **优先补 CLI 自动采集** |
| 是否可复用 | 相同 topic+problem+证据形态是否应进入 skill asset / builtin YAML |

### 沉淀优先级（CLI 侧实现顺序）

1. **本地只读采集 + 确定性规则**（`probe`、evidence parser、`param_contract`、本地复核启发式）。
2. **skill pack YAML**（`extra_guidance`、输出格式、禁止甩锅采集）；遵守 **`.cursor/skills/skill-pack-assets/SKILL.md`**，**禁止**把核心 YAML 提交 GitHub。
3. **skill asset review**（可复用故障模式、需 super_admin 审核的沉淀）。
4. **自动迭代**（产品能力缺口、bug）；见 **`.cursor/skills/auto-iteration-dev/SKILL.md`**。
5. **高风险**（新 CLI 参数影响计费/权限、破坏性自动修复）→ super_admin 或自动迭代高风险审批。

### CLI 开发时的强制动作

- 新增或修改 `check <topic>`：**同时**评估对应 `ft-backend/skills/builtin/<topic>_*.yaml`（本地维护 + 实验室/生产脚本发布）。
- 合并前在 PR/回复中写明：**本次是否增强技能包**；若否，**必须说明原因**（一次性问题、证据不足、待样本等）。
- 允许记录诊断样本路径（后续 `DiagnoseSample` / `skill_enhancement_review` 由服务端实现）；开发阶段至少在 issue/PR 留 **证据 JSON 摘要 + 拟增强点**。

### 禁止

- 禁止 AI 调用后只展示答案，不记录样本、不评估技能包增强。
- 禁止把完整 Prompt、核心 skill YAML 或权益配置下发给 CLI。
- 禁止为降低 AI 成本而跳过必要诊断；**先保证结论可靠**，再谈沉淀。
- 禁止通过随意新增 CLI 参数绕过技能包能力不足（参数须符合 `param_contract` 与诊断契约）。

### 与现有模块关系

| 模块 | 沉淀类型 |
|------|----------|
| `topic_evidence.go` / `*_probe.go` | evidence 采集规则 |
| `diagnose_output_format.go` / `ai_diagnose_review.go` | 输出结构、本地复核规则 |
| `execution_intent.go` / skill tree | topic、pack_key、problem_key 坐标 |
| `install_recovery.go` | install topic 技能与 error pattern |

### 后续实现（代码层，规范已约束行为）

- AI 成功后统一记录 `DiagnoseSample`；`skill_enhancement_review` 元数据。
- 高频相似调用触发 `MaybeAutoRefine`。
- 控制台展示「AI 成本节省潜力 / 待增强技能包」。

## 安装/下载失败 → 服务端 AI（强制）

**产品要求**：`install` / `upgrade` / 自动升级任一步失败时，**禁止**仅输出本地 error 后结束；**必须** `POST /api/ai/diagnose`（topic=`install`）→ 失败则 `/api/ai/ask` → 再输出内置 `curl` 手工步骤。

实现：`recoverInstallDownloadFailure`（`install_recovery.go`）。调用方勿将 cause 直接 `return err` 给最终用户。

豁免：`OPSFLEET_SKIP_INSTALL_AI_RECOVERY=1`。技能包：`cli_install_recovery.yaml`。

## 自升级（实现细节，与上文铁律一致）

### 实现入口（勿绕过）

| 位置 | 作用 |
|------|------|
| `root.PersistentPreRunE` → `opsfleetPersistentPreRun` | 每个子命令前执行 |
| `preflightAutoUpgradeIfUnknown` | 未知子命令时先尝试升级再报错 |
| `fetchRemoteVersionFast` | `collectOpsfleetAPIBaseCandidates` 顺序探测直至成功，总预算 ~3.5s |
| `tryAutoUpgradeInPlace` | 比对版本、下载、校验、exec |

### 豁免（不得扩大）

仅以下情况 **不** 做升级探测：

- `ai-sre version` / `upgrade` / `help` / `completion`
- `-h` / `--help`
- `OPSFLEET_NO_AUTO_UPGRADE=1` 或全局 `--no-auto-upgrade`

**不得** 为其它子命令（含 `doctor`、`probe`、`check`、`k8s`）默认跳过。

### API 基址（自动绑定 + 探测链）

1. **业务与升级共用** `resolveOpsfleetAPIBaseStrict`：install 文件优先；与 `OPSFLEET_API_URL` 冲突时**自动忽略 env**（`autoBindingWarn`）
2. **版本探测** `collectOpsfleetAPIBaseCandidates`：install → env → 实验室 → 生产（去重），`fetchRemoteVersionFast` 逐个尝试直至成功
3. **禁止** 返回空候选列表；**禁止** 因冲突向用户抛「禁止混用」错误（改为自动绑定 + 一行 stderr）

### 双架构分发（与自升级联动）

- `GET .../cli/ai-sre/version` 须与 **当前 arch 下载包** 一致（见 `aiSreAdvertisedVersion`、部署 `sync-aisre-backend-env.sh`）。
- 部署时 amd64/arm64 **版本不一致必须失败**，避免客户端下载后版本不变而死循环。

### 改动客户端逻辑时

1. 触及 `internal/cli` 行为变更 → **递增** `internal/cli/version.go`。
2. 本地：`./scripts/deploy-local.sh`；生产/实验室：全栈 deploy + `sync-aisre-backend-env.sh`。
3. 验收：`ai-sre version`；`OPSFLEET_AUTO_UPGRADE_VERBOSE=1 ai-sre doctor` 应能看到探测日志；故意降版本二进制后执行任意子命令应触发升级。

### 下载超时（勿与版本快检混淆）

| 场景 | 超时 |
|------|------|
| 版本快检 `fetchRemoteVersionFast` | 单址 ~1.2s，总 ~3.5s |
| **二进制下载** `upgrade` / 自动升级 | 默认 **10 分钟**（`upgradeDownloadTimeout`） |

**禁止** 在 `upgrade` 交互确认（`输入 y`）之前创建下载用 `context`——否则用户输入时间会计入 deadline（曾导致 15s 内下不完 ~10MB arm64 包）。

环境变量：`OPSFLEET_UPGRADE_DOWNLOAD_TIMEOUT=15m` 可覆盖下载时限。

### 调试环境变量

| 变量 | 含义 |
|------|------|
| `OPSFLEET_AUTO_UPGRADE_VERBOSE=1` | 打印探测/失败详情 |
| `OPSFLEET_AUTO_UPGRADE_QUIET=1` | 隐藏「版本检查未完成」单行提示（不关闭探测） |
| `OPSFLEET_NO_AUTO_UPGRADE=1` | 关闭自动升级 |
| `OPSFLEET_API_URL` | 可选覆盖（与 install 冲突时自动忽略 env，勿写入用户主路径文档） |
| `OPSFLEET_UPGRADE_DOWNLOAD_TIMEOUT` | 下载超时（如 `10m`） |

## 与发布 skill 关系

客户端变更合并前须满足 **`.cursor/skills/release-deploy/SKILL.md`**（版本号、本机/远端部署、push）。
