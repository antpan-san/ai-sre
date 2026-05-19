---
name: ai-sre-diagnosis-contract
description: >-
  Contract for ai-sre check/probe diagnosis: param contract, automatic evidence
  collection, evidence-first AI, no delegating collection to users. Use when
  adding or changing check <topic>, probe, server diagnose prompts, or middleware skills.
---

# ai-sre 诊断契约（强制）

## 采集优先，AI 后置

1. `ai-sre check <topic> [target]` 必须先完成对应 `probe` 只读采集，再调用服务端 AI。
2. AI **禁止**输出「请用户执行 redis-cli / probe / kubectl / shell」补采集。
3. 证据不足时，只说明**缺失原因**（认证失败、ACL、网络），不给命令清单。

## 服务端复核与客户端纯文本（强制）

| 步骤 | 要求 |
|------|------|
| Prompt | 中间件 topic 使用 `middleware_evidence`：`【根因与触发条件】` `【关键指标证据】` `【缓解与根治建议】`，禁止 Markdown `##` |
| 复核 | `finalizeDiagnoseAnswer`：启发式校验因果（如 Redis 低内存+高碎片不得单独解释少量 rejected_connections）→ 有问题则二次 LLM 修订 |
| 客户端 | `formatCheckAnswerText`：默认 **text** 输出，剥离 `##`、`**`、代码块 |
| 禁止 | 未经复核直接把初稿返回用户；禁止 check 默认输出 Markdown |

实现：`ft-backend/handlers/ai_diagnose_review.go`、`ai_diagnose_middleware_prompt.go`；`internal/cli/diagnose_output_format.go`。

## 参数合同 vs 能力层

| 层 | 触发 | 行为 |
|----|------|------|
| 参数 | 未知命令、flag 错误、参数个数错误 | 本地纠错建议，`auto_iteration_created=false` |
| 认证 | Redis 等需要密码且未提供 | TTY 交互输入；非 TTY 返回 `auth_required=true`，不伪分析 |
| 能力 | 采集器 bug、字段缺失、AI 仍甩锅用户采集 | feedback / fulfillment / 自动迭代 |

**不触发自动迭代**：用户未输入密码、地址不可达、ACL 拒绝部分只读命令、参数错误。

## 新增中间件诊断 checklist

- [ ] `probe <topic> [target] --json` 结构化字段完整
- [ ] `check <topic> [target]` 进程内采集 + 证据注入 `*_diagnose_json`
- [ ] 密码不落日志、不进上传 context（`stripSensitiveCheckContext`）
- [ ] 服务端 `evidence_root_cause` prompt 收录 `redis_*` / `kafka_*` 等证据键
- [ ] 单测：无密码、NOAUTH+AUTH、ACL 部分失败
- [ ] 技能包 ExtraGuidance 禁止「列出采集命令」

## Redis 样板

实现见 `internal/cli/redis_probe.go`、`check_redis.go`；推广至 Kafka/MySQL/PostgreSQL/Nginx/ES 时复用同一契约。
