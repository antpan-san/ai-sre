---
name: ai-sre-client-dev
description: >-
  ai-sre CLI client development norms: mandatory fast auto-upgrade before every subcommand,
  OPSFLEET_API_URL bases, version bump rules. Use when changing internal/cli, upgrade flow,
  or client install/self-update behavior.
---

# ai-sre 客户端开发规范

## 精简参数原则（强制）

**产品要求**：用户侧命令以**最少位置参数**完成主路径；禁止把 `-d key=value` / `--set` 当作日常必选项。

| 原则 | 说明 |
|------|------|
| 主路径 | `ai-sre check <topic>` 即可启动诊断（中间件 topic 自动填充本机默认地址，见 `check_target.go`） |
| 可选覆盖 | `ai-sre check <topic> <target>` 一个位置参数覆盖连接目标（与 `probe <topic> <target>` 对齐） |
| 环境变量 | 非本机默认时用 `AI_SRE_*`（如 `AI_SRE_REDIS_ADDR`），**不算**「额外 CLI 参数」 |
| `-d` / flag | 仅用于 K8s 场景、Kafka lag/topic、密码、非 TTY `--yes` 等**高级**场景；已有 `-d` 时**不得**被默认值覆盖 |
| 新增 topic | 必须在 `checkTargetSpecs`（或 domain 专用逻辑）登记默认目标；README / `check` Long 须给出一行最简示例 |

实现入口：`applyCheckTargetContext`、`checkTopicAcceptsOptionalTarget`（`internal/cli/check_target.go`）。

## 安装/下载失败 → 服务端 AI（强制）

**产品要求**：`install` / `upgrade` / 自动升级任一步失败时，**禁止**仅输出本地 error 后结束；**必须** `POST /api/ai/diagnose`（topic=`install`）→ 失败则 `/api/ai/ask` → 再输出内置 `curl` 手工步骤。

实现：`recoverInstallDownloadFailure`（`install_recovery.go`）。调用方勿将 cause 直接 `return err` 给最终用户。

豁免：`OPSFLEET_SKIP_INSTALL_AI_RECOVERY=1`。技能包：`cli_install_recovery.yaml`。

## 自升级（强制，每次命令）

**产品要求**：用户执行 **任意** `ai-sre` 子命令（除下文豁免）前，必须先做 **快速** 版本探测；若 OpsFleet 上版本更高，则 **自动下载并 re-exec** 同一参数（Linux/macOS）。

### 实现入口（勿绕过）

| 位置 | 作用 |
|------|------|
| `root.PersistentPreRunE` → `opsfleetPersistentPreRun` | 每个子命令前执行 |
| `preflightAutoUpgradeIfUnknown` | 未知子命令时先尝试升级再报错 |
| `fetchRemoteVersionFast` | 多基址、单址 ~1.2s 超时、总预算 ~3.5s |
| `tryAutoUpgradeInPlace` | 比对版本、下载、校验、exec |

### 豁免（不得扩大）

仅以下情况 **不** 做升级探测：

- `ai-sre version` / `upgrade` / `help` / `completion`
- `-h` / `--help`
- `OPSFLEET_NO_AUTO_UPGRADE=1` 或全局 `--no-auto-upgrade`

**不得** 为其它子命令（含 `doctor`、`probe`、`check`、`k8s`）默认跳过。

### API 基址优先级（`resolveOpsfleetAPIBasesForUpgrade`）

1. `OPSFLEET_API_URL`
2. `~/.config/ai-sre/opsfleet_api_url`（install-ai-sre 写入）
3. `config.yaml` 中 `opsfleet_api_url`
4. 内嵌实验室 `http://192.168.56.11:9080/ft-api`
5. 内嵌生产回退 `http://opsfleetpilot.com/ft-api`

**禁止** 只内嵌实验室 IP 而不做生产回退——外网 ARM 控制机否则永远探测失败、表现为「从不自动升级」。

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
| `OPSFLEET_AUTO_UPGRADE_VERBOSE=1` | 打印探测/失败原因 |
| `OPSFLEET_NO_AUTO_UPGRADE=1` | 关闭自动升级 |
| `OPSFLEET_API_URL` | 覆盖探测基址（应含 `/ft-api`） |
| `OPSFLEET_UPGRADE_DOWNLOAD_TIMEOUT` | 下载超时（如 `10m`） |

## 与发布 skill 关系

客户端变更合并前须满足 **`.cursor/skills/release-deploy/SKILL.md`**（版本号、本机/远端部署、push）。
