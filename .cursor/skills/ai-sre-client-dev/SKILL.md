---
name: ai-sre-client-dev
description: >-
  ai-sre CLI client development norms: mandatory fast auto-upgrade before every subcommand,
  OPSFLEET_API_URL bases, version bump rules. Use when changing internal/cli, upgrade flow,
  or client install/self-update behavior.
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

所有 `check <topic>`：**先 probe 采集，再 AI**；禁止 AI 让用户手工采集。详见 **`.cursor/skills/ai-sre-diagnosis-contract/SKILL.md`**。Redis 样板：`redis_probe.go`、`check_redis.go`。

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
