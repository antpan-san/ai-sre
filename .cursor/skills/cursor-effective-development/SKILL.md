---
name: cursor-effective-development
description: >-
  Guides Cursor agent collaboration for this user: concise task scoping, @-narrowed context,
  token-efficient messages, and ai-sre monorepo habits (releases default to full ship per
  monorepo-release.mdc). Apply at the start of every implementation request, plan execution,
  or when the user asks how to phrase tasks or save tokens in Cursor.
---

# Cursor 高效开发协作（用户习惯）

本 skill 约束**代理**如何理解需求、控制范围、省 token，并与本仓其它 skill 衔接。用户无需每次重复说明。

## 代理：收到实现类需求时先做

1. **解析用户消息**是否包含下表五元；缺项则在动手前用**一条简短追问**补齐（不要长问卷）：
   - 做什么 / 不做什么 / 验收标准 / 环境（实验室 192.168.56.11 或仅本地）/ **是否豁免发布**（默认不豁免；仅当用户本回合明确写「只改本地 / 不 SSH / 不 commit / 不 push」等）
2. **本仓（ai-sre monorepo）强制**：凡改动可追溯进 git 的工件，**默认完整**执行 **`monorepo-release.mdc` → `release-deploy` → `ai-sre-ship`（及适用的 `opsfleetpilot-ship`）**，含远端 deploy、冒烟、**`git commit`（每一批代码改动必须 commit，不得留未提交工作区）+ `git push`**。**不要**等用户说「发布」或「提交」。**不要**以其它对话中的泛化「不主动提交」偏好跳过本仓发布流程。
3. **豁免**：仅当用户**在本回合用户消息中**明确写出不 deploy、不 push、不 commit 等时方可缩小范围；否则须在**同一助手回合内**跑完适用脚本与 git 收尾。
4. **严禁**仅用「已通过 `npm run build` / `go build`」收官：只要触及 `ft-backend`、`ft-front`、根 CLI 等与线上一致的路径，必须以**远端构建与冒烟结果**作为「可做发布结论」的依据（见 `monorepo-release.mdc`）。
5. **用 @ 收窄上下文**：优先用户 @ 的文件/目录；禁止无必要全仓 `@Codebase` 式探索。
6. 任务结束回复须包含 **发布自检段落**：列出已执行的脚本命令、摘要结果（exit）；若豁免须逐条写明「跳过 deploy 的原因」。未执行发布亦须承认并说明是否属违规，不得以「如需发布请说」推脱。

### 为何会漏跑发布（复盘模板）

| 原因 | 说明 |
|------|------|
| 把「改完并通过本地编译」误判为做完 | Skill 定义的完成含远端 deploy + 冒烟 + 默认可用的 **commit + push** |
| 与全局「不主动 git commit」混淆 | **在本仓库以 `monorepo-release.mdc` 为准**：测试通过后默认 **commit + push**，除非用户本回合明确豁免 |
| 对话在此处结束（无下一轮） | 代理应在**同一条助手回复的工具调用中**顺序跑完脚本与 git，再写总结 |

用户未写【发布】时：实现完成后仍须默认跑 **`deploy-remote.sh`** + 适用 **`deploy-opsfleet-remote.sh`** + **`SHORT=1 remote-e2e`** +（适用时）verify，然后 **`git commit` + `git push`**（无豁免时）。

## 推荐用户消息模板（可提示用户复制）

```text
【做】<一句话目标>
【不做】<明确排除项，如不改前端 / 不合并历史报告>
【验收】<测试或检查项，如 go test、verify-opsfleet、三门版本一致>
【环境】实验室 192.168.56.11 / 仅本地
【发布】默认不写即**完整 release（含 commit+push）**；仅当本回合要豁免时写「不 SSH / 不 commit / 不 push」
【其它】<版本号是否 bump、是否改 config.yaml 等>
```


| 做法 | 说明 |
|------|------|
| 小步 @ | 只读/改相关路径，跨模块时分轮并写清「本轮只改 backend」 |
| Plan 一次 | Plan 定稿后实现轮写「按 plan，勿改 plan 文件」；避免每轮重贴长 plan |
| 少先要长文 | 边界清楚时直接实现+测试，除非用户要「先讲现状」 |
| 新 thread 承接 | 上下文变长时建议用户开新对话，附 5 行「已完成/待做/不要」 |
| 区分问与做 | 「为什么没发布」= 审计；「继续实现」= 写代码+deploy，勿只解释 |

## 快速理解意图（代理行为）

- **产品决策**以用户最新消息为准；与 plan 冲突时以用户「变更：…」为准。
- **ai-sre 同仓**常见意图映射：
  - 动 `internal/cli` / `main.go` → 必须 `deploy-opsfleet-remote.sh`（更新 `bin/ai-sre`），不能只 `deploy-remote.sh`
  - 动 `ft-backend/skills/builtin/**` 等技能包 → `git push` 后 **`./scripts/deploy-skill-packs-production.sh`**（生产）；实验室不算技能包发布
  - 动 `ft-backend` 配置 → `conf/config.yaml` + `backend-configuration` skill，密钥仅 env
  - 动 ansible/K8s 离线/UI 部署 → 另加 `k8s-offline-deploy-test`
- 用户说「实现 plan」→ 打开 plan 中 todos，逐项完成并标记，**不编辑 plan 文件本身**。

## 与本仓其它 skill 的关系

| 场景 | 同时遵循 |
|------|----------|
| 任意文件变更发布 | `release-deploy` → `ai-sre-ship` / `opsfleetpilot-ship` |
| 后端 OPSFLEET 配置 | `backend-configuration` |
| API/安全面 | `security-first-development` |
| 错误码/部署失败 | `error-code-development-gate` |

## 用户可选用法（Cursor 产品层）

- **大功能**：Plan 定边界 → Agent 分模块实现（每轮带【发布】声明）。
- **小改/答疑**：Ask + 单文件 @。
- **全仓陌生代码**：Explore 子任务并**限定目录**。

## 示例

**高效：**

```text
【做】auto refine 写入 conf/config.yaml，走 Resolved*
【不做】不改 ft-front；不 git push
【验收】go test ./ft-backend/...；deploy-opsfleet + verify
【环境】192.168.56.11
【发布】deploy + 冒烟，本回合不 commit（豁免示例）
```

**低效（代理应主动要求补全）：**

```text
帮我把配置整理好。
```

## 完成汇报模板（代理结尾可选用）

- 变更摘要（1–3 句）
- 测试/部署：命令与结果
- 发布 skill：已跑 / 跳过及原因
- Git：未提交 / 已 commit+push（哈希）
- 残留风险或需用户手动项（如 `OPSFLEET_AI_API_KEY`）
