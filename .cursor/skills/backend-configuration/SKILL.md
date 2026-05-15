---
name: backend-configuration
description: >-
  OpsFleet ft-backend 配置规范：新增或修改运维/AI/K8s/技能相关行为时，必须写入 conf/config.yaml 结构化字段，
  通过 common/config 的 Resolved* 读取；环境变量仅作覆盖。与 security-first-development 一并使用。
---

# 后端配置管理规范

## 原则

1. **单一事实来源（默认）**：`ft-backend/conf/config.yaml`（实验室）或 `deploy/config.production.example.yaml` 复制后的生产 `config.yaml`。
2. **覆盖层**：`/etc/opsfleet/backend.env`（systemd `EnvironmentFile`）中的 `OPSFLEET_*` **仅当需要覆盖 yaml 或存放密钥**时使用。
3. **优先级**：`环境变量 > config.yaml > 代码内硬编码默认值`。
4. **禁止**：在 `handlers/`、`services/` 中新增裸 `os.Getenv("OPSFLEET_…")`；一律经 `ft-backend/common/config` 的 `Resolved*` 或 `EnvOr*`。

## 配置段一览

| YAML 段 | 用途 | 典型环境变量覆盖 |
|---------|------|------------------|
| `ai` | DeepSeek 兼容 LLM | `OPSFLEET_AI_API_KEY`（**仅 env**）、`OPSFLEET_AI_BASE_URL`、`OPSFLEET_AI_MODEL` |
| `opsfleet` | ai-sre 分发路径、技能数据目录 | `OPSFLEET_AISRE_BINARY_PATH*`、`OPSFLEET_AI_SKILL_DATA_DIR` |
| `k8s` | 制品 manifest、relay、ansible 目录 | `OPSFLEET_K8S_MIRROR_*`、`OPSFLEET_K8S_RELAY_BASE_URL`、`OPSFLEET_ANSIBLE_DIR` |
| `skills.auto_refine` | 样本达阈值自动 refine | `OPSFLEET_SKILL_AUTO_REFINE`、`OPSFLEET_SKILL_AUTO_REFINE_*` |

常量名定义在 `ft-backend/common/config/resolved.go`（`EnvAIAPIKey` 等），避免拼写漂移。

## 新增配置项流程

1. 在 `common/config/config.go` 增加 YAML 字段与注释。
2. 在 `common/config/resolved.go` 增加 `ResolvedFoo()`，内部使用 `EnvOrString` / `EnvOrBool` / `EnvOrInt` / `EnvOrDuration` / `EnvOrStringList`（`runtime.go`）。
3. 业务代码只调用 `config.ResolvedFoo()`（`services` 可再包一层如 `LoadServerAIConfig()`）。
4. 同步更新：
   - `ft-backend/conf/config.yaml`（实验室合理默认）
   - `deploy/config.production.example.yaml`
   - `deploy/backend.env.example`（仅列密钥与「临时覆盖」示例）
5. 为 `Resolved*` 增加 `common/config/resolved_test.go` 用例（yaml 默认 + env 覆盖）。
6. 若影响运维文档，更新根 `README.md` 对应小节。

## 密钥与敏感项

- **`ai.api_key`**：生产/实验室均优先 `OPSFLEET_AI_API_KEY`；**不要**将真实 key 提交到 git。
- JWT、数据库密码、Redis 密码：继续只放在 `config.yaml` 或部署机本地文件，勿写入示例仓库。

## 技能自动 refine（实验室）

在 `conf/config.yaml`：

```yaml
skills:
  auto_refine:
    enabled: true
    min_samples: 8
    cooldown: 12h
    topics: [go_runtime, k8s]
    max_per_day: 3
```

生产默认 `enabled: false`（见 `deploy/config.production.example.yaml`）。修改后 `systemctl restart opsfleet-backend`。

## 与发布的关系

- 全栈 `deploy-opsfleet-remote.sh` 仍会向 `backend.env` 写入 **路径类**变量（`OPSFLEET_AISRE_BINARY_PATH`、`OPSFLEET_AI_SKILL_DATA_DIR` 等），与 yaml 中 `opsfleet` 段等价；以 **env 为准**。
- 行为开关（如 auto refine）应主要在 **yaml** 维护，避免仅存在于某台机 `backend.env` 导致漂移。

## 完成检查

- [ ] 无新增裸 `os.Getenv("OPSFLEET_…")`（`common/config` 包除外）
- [ ] `go test ./common/config/...` 通过
- [ ] 示例 yaml 与 `backend.env.example` 已更新
