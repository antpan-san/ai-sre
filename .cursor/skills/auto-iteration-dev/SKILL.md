---
name: auto-iteration-dev
description: >-
  Code-agent worker task playbook: implement CLI/skill gaps from auto-iteration queue,
  verify with SHORT=1 remote-e2e, push via github-push-safe only, deploy skill packs separately.
---

# 自动迭代开发（Code Agent）

## 目标

完成控制台/CLI 反馈创建的自动迭代任务，使 Worker 冒烟 **SHORT=1 bash scripts/remote-e2e.sh** 通过。

## 必须遵守

1. **范围**：只改与任务 Topic/摘要相关的文件；勿改 plan 附件、无关重构。
2. **技能包 YAML**：`ft-backend/skills/builtin/*.yaml` **禁止** `git add`/commit；本地可维护，发布用 `deploy-skill-packs-production.sh`。
3. **推送**：仅 `bash scripts/github-push-safe.sh`（或先 `./scripts/check-skill-packs-not-in-git.sh` 再 push）；勿提交 `backend.env`、`code-agent-worker.env`、密钥。
4. **验证**：开发期 `go test`/`go vet` 针对改动包；收尾前 `SHORT=1 bash scripts/remote-e2e.sh`（禁止无 SHORT 全量 LLM 冒烟）。
5. **版本**：动 CLI 时 bump `internal/cli/version.go`。

## 中间件 Topic 模板（如 postgresql/general）

- CLI：`internal/cli/<topic>.go` + `execution_intent.go` + `topic_evidence.go` + `root.go` 注册
- 后端：`skill_tree.go`、`skill_commercial.go`、`models/billing.go`（SkillPack*）、`skill_asset.go`
- 技能包：本地 `ft-backend/skills/builtin/<topic>.yaml`（不进 git）
- 单测：`internal/cli/*_test.go` 覆盖 intent 坐标

## 发布顺序（任务要求上线时）

1. Read `.cursor/skills/release-deploy/SKILL.md`
2. 实验室：`deploy-remote.sh`、`deploy-opsfleet-remote.sh`、`SHORT=1 remote-e2e`（若可达）
3. `git commit` + `bash scripts/github-push-safe.sh`
4. 生产：`.cursor/skills/production-deploy/SKILL.md`（勿对生产跑 deploy-opsfleet-remote.sh）
5. 若改了技能包 YAML：`./scripts/deploy-skill-packs-production.sh`
