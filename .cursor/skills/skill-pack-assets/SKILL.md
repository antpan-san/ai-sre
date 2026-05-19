---
name: skill-pack-assets
description: >-
  Skill pack assets: no YAML on GitHub; lab/production deploy; mandatory post-AI
  enhancement review and沉淀路径. Use when changing builtin/generated skills or after
  any AI diagnose/ask/runbook flow.
---

# 技能包核心资产（禁止上 GitHub）

## 策略（用户指定）

| 环境 | 技能包 YAML | GitHub |
|------|-------------|--------|
| **GitHub 仓库** | **禁止** 提交/推送 `*.yaml` 技能包正文 | 仅代码、README 占位、部署脚本 |
| **本地** | 允许维护 `ft-backend/skills/builtin/`、`internal/assets/skills/` | 不 push |
| **实验室 192.168.56.11** | **允许** rsync 部署与联调 | — |
| **生产 204.44.123.101** | **权威存放** + 对外服务注册表 | — |

`git push` **只推代码**；**不得**把「技能包已发布」等同于 push 成功。

## AI 调用后的技能包增强审查（强制）

**产品要求**：每次在已支持 topic / skill pack 下完成 AI 诊断（CLI `check` / 服务端 `POST /api/ai/diagnose` 等），都必须审查**当前技能包是否可增强**，使下次相同问题尽量**不再次调用 AI**。

**原则**：技能包是 AI 调用的沉淀出口；审查**不再调用 AI**，而是改 YAML、采集规则或本地判断。

### 可增强的沉淀形态（优先顺序）

| 类型 | 存放位置 | 说明 |
|------|----------|------|
| evidence 采集规则 | CLI `probe` / `gatherTopicEvidence`（**代码**，git push） | 补全 JSON 字段，禁止让用户手工执行命令 |
| 本地判断规则 | CLI 启发式、服务端 `ai_diagnose_review`（**代码**） | 确定性复核，减少二次 LLM |
| skill YAML 约束 | `ft-backend/skills/builtin/*.yaml`（**禁止 commit**） | `analysis_steps`、`extra_guidance`、输出小节 |
| error code / pattern | `error_codes.yaml` 或 topic 技能（**禁止 commit**） | 与 `ai-sre analyze code`、控制台同源 |
| 诊断模板 / 输出结构 | YAML `output_format` + CLI `diagnose_output_format` | 纯文本小节名写死，避免模型漂移 |
| 可复用自动修复建议 | skill asset → review → publish | 高风险须 super_admin |

### 沉淀优先级

1. 能用**本地只读采集 + 确定性规则**解决的 → 先改 CLI/服务端代码。
2. 需要经验表达、不需改逻辑的 → 增强 **builtin YAML**（本目录策略下本地改，脚本发布）。
3. 可复用故障模式 → **skill asset** 审核流（`ft-backend` SkillAsset）。
4. 产品缺口或 bug → **自动迭代**（见 **`.cursor/skills/auto-iteration-dev/SKILL.md`**）。
5. 涉及权限/计费/新 CLI 参数 → **super_admin** 或自动迭代高风险审批。

### 发布与验收（增强后必做）

```bash
./scripts/check-skill-packs-not-in-git.sh   # commit 前，有代码提交时
./scripts/deploy-skill-packs-lab.sh         # 联调
./scripts/deploy-skill-packs-production.sh # 权威；对外以 curl .../api/ai/skills 为准
```

- **禁止**把「已 `git push`」当作技能包已更新。
- **禁止**为沉淀而把完整 Prompt 或权益配置写入 YAML 下发给 CLI。
- **禁止**把「请用户再执行 top/redis-cli/kubectl」写进 `extra_guidance`；应推动 **probe 自动采集**（见 **`.cursor/skills/ai-sre-client-dev/SKILL.md`**）。

### 禁止

- 禁止 AI 返回后只展示答案、不评估技能包是否可增强。
- 禁止将 `ft-backend/skills/builtin/*.yaml`、`internal/assets/skills/*.yaml` 纳入 GitHub commit。
- 禁止跳过高风险沉淀的审核门槛。

### 后续实现（平台）

- `DiagnoseSample` 与 `skill_enhancement_review` 事件。
- `MaybeAutoRefine` 对高频样本自动起草 YAML。
- 控制台「待增强技能包 / AI 成本节省潜力」。

## 路径

| 用途 | 本地（gitignore） | 服务端运行时 |
|------|-------------------|--------------|
| Builtin | `ft-backend/skills/builtin/*.yaml` | `/var/lib/opsfleet/ai-skills/builtin/` |
| Generated | — | `.../ai-skills/generated/` |
| 样本/反馈 | — | `.../samples/`、`.../feedback/` |

## 部署命令（代理必须会）

```bash
# 实验室联调
./scripts/deploy-skill-packs-lab.sh

# 生产权威环境
./scripts/deploy-skill-packs-production.sh
```

推送前自检（仓库根）：

```bash
./scripts/check-skill-packs-not-in-git.sh
```

## 与 release 的关系

- 触及 **仅** 技能包 YAML：实验室脚本 + 生产脚本；**不要**把 YAML 放进 `git commit`。
- 触及 **SkillRegistry / API** 代码：走 `release-deploy` → 实验室代码部署 → `git push` → 若同时改了本地 YAML，再跑上述技能包脚本。

## 禁止

- 将 `ft-backend/skills/builtin/*.yaml` 或 `internal/assets/skills/*.yaml` 加入 commit。
- 在回复中宣称「已 push 到 GitHub」即等于技能包已安全发布（生产 `curl .../api/ai/skills` 才算）。
