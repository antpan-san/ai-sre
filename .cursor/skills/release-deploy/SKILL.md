---
name: release-deploy
description: >-
  **Mandatory after every ai-sre monorepo code change batch**: bump Version when ai-sre CLI/instruction
  code changes, run ./scripts/deploy-local.sh (local build), remote deploy + smoke, then git commit + git push.
  Use when finishing development, or when the user says 发布/部署/上线/ship.
---

# 发布部署（总入口）

**每次开发完成必执行（写死，无例外）**：在本仓库内完成**任意**可提交改动后，代理**必须在本回合内**跑完下文检查清单（含 **版本号**、**本机部署**、**远端部署/冒烟**、**`git commit` + `git push`**），**不得**等用户说「发布 / 部署 / 提交 / push」。仅当用户**在本轮用户消息**中明确豁免（如「只改本地、不 SSH、不要 commit、不 push」）时可缩小范围。

**本仓规则优先**于对话里泛化的「不主动提交」偏好（见 **`.cursor/rules/monorepo-release.mdc`**，`alwaysApply`）。

**用户与 192.168.56.11**：实验室 OpsFort 维护由代理在**工作机仓库根**执行 `deploy-local.sh`、`deploy-opsfleet-remote.sh`、`deploy-remote.sh` 等（脚本内 SSH）；**不要求用户**登录 192.168.56.11。

## 仓库与规则

| 项 | 路径或说明 |
|----|------------|
| 仓库根 | 工作区中的 ai-sre 克隆目录（示例：`/root/sre`、开发机上的同仓路径） |
| Cursor 规则（alwaysApply） | `.cursor/rules/monorepo-release.mdc` |
| CLI 同步 + 冒烟 + README + push | `.cursor/skills/ai-sre-ship/SKILL.md` |
| OpsFleet 全栈 | `.cursor/skills/opsfleetpilot-ship/SKILL.md` |
| 生产 opsfleetpilot.com | `.cursor/skills/production-deploy/SKILL.md` |
| K8s 离线 / 控制台 K8s | `.cursor/skills/k8s-offline-deploy-test/SKILL.md` |

## ai-sre 指令代码 → 必须升级版本号

凡触及下列路径之一且变更会影响**子命令行为、参数、输出、技能执行或诊断逻辑**（含修复 bug、改默认、改 prompt/编排），**必须先**递增 **`internal/cli/version.go`** 中的 `Version`（patch 位，如 `0.5.25` → `0.5.26`），并同步 **README** 中的版本说明：

| 路径（相对仓库根） | 说明 |
|--------------------|------|
| `internal/cli/**` | 全部 CLI 子命令与编排（**指令代码**主目录） |
| `internal/skill/**`、`internal/engine/**`、`internal/prompt/**`、`internal/loader/**` | 技能加载与执行引擎 |
| `internal/go_runtime/**` | Go 运行时/K8s 诊断采集 |
| `internal/assets/skills/**` | 随仓发布的内置技能 YAML（非 `ft-backend/skills/builtin`） |
| `main.go`、根 `go.mod`、`go.sum` | 入口与依赖 |

**纯注释/格式/无行为差异的重命名**可不 bump；**不确定时一律 bump**。未 bump 不得进入 commit/push。

版本单一来源：`internal/cli/version.go` → 本机 `./ai-sre version` → 远端二进制 → `GET .../cli/ai-sre/version`（见项 4a）。

## 技能包 vs GitHub push

| 动作 | GitHub | 实验室 192.168.56.11 | 生产 204.44.123.101 |
|------|--------|----------------------|---------------------|
| 技能包 `*.yaml`（`ft-backend/skills/builtin` 等） | **禁止** commit/push | `deploy-skill-packs-lab.sh` | `deploy-skill-packs-production.sh` |
| 代码 | **`git push`（必做）** | `deploy-opsfleet-remote.sh` 等 | `production-deploy`（按需） |

推送前：`./scripts/check-skill-packs-not-in-git.sh`。

## 执行顺序（必须 — 每次开发完成复制勾选）

```
发布部署检查清单（完成开发后必跑）
- [ ] 0. 确认用户本轮未豁免；由代理在工作机执行，不要求用户 SSH 到 192.168.56.11
- [ ] 1. **版本号**：若触及「ai-sre 指令代码」表内路径 → 已递增 `internal/cli/version.go`；README 版本段已核对
- [ ] 2. **本机部署（必做）**：仓库根 `./scripts/deploy-local.sh` 通过（`go vet` + `go build` + `./ai-sre version` 与 version.go 一致）
      - 若本轮还改了 ft-backend/、ft-front/、deploy/、ansible-agent/ → `DEPLOY_LOCAL_OPSFLEET=1 ./scripts/deploy-local.sh`（本机 build-all，勿提交 bin/、dist/）
- [ ] 3. **ai-sre 可执行 / 指令逻辑变更**：除本机外，**必须** `./scripts/deploy-opsfleet-remote.sh`（更新 `$REMOTE_DIR/bin/ai-sre`），**不得**仅 `deploy-remote.sh`；并执行 ai-sre-ship 其余项
- [ ] 4. **ai-sre-ship**：`./scripts/deploy-remote.sh` → `SHORT=1 bash scripts/remote-e2e.sh` **通过** → README 复核
- [ ] 5. **OpsFleet 全栈**（触及 ft-*、deploy、ansible-agent 或需对外 ai-sre 版本时）：`deploy-opsfleet-remote.sh` → SSH `bash scripts/verify-opsfleet-deployment.sh`
- [ ] 5a. **版本三门一致**（有 OpsFort 时）：`version.go` = 远端 `$OPSFLEET_AISRE_BINARY_PATH` 的 `version` = `curl -sS http://192.168.56.11:9080/ft-api/api/k8s/deploy/cli/ai-sre/version`；不一致则重跑 deploy-opsfleet-remote
- [ ] 6. **K8s 离线**（若适用）→ `k8s-offline-deploy-test`
- [ ] 7. **git commit + push（必做，在 2–6 全部通过后）**：`git add` → **`git commit`**（本轮全部改动，禁止留脏工作区）→ **`git push origin main`**（远程测试未通过禁止 push）
- [ ] 7b. 技能包 YAML 有本地改动：`git push` 后 `deploy-skill-packs-lab.sh` + `deploy-skill-packs-production.sh`（勿把 YAML 纳入 commit）
- [ ] 8. 汇报：本机/远端 exit、版本三门、verify 摘要、提交哈希
```

### 推荐命令顺序（与子 skill 一致）

| 步骤 | 动作 |
|------|------|
| L | **`./scripts/deploy-local.sh`**（本机环境；OpsFleet 路径加 `DEPLOY_LOCAL_OPSFLEET=1`） |
| A | `./scripts/deploy-opsfleet-remote.sh`（适用时） |
| B | SSH：`bash scripts/verify-opsfleet-deployment.sh` |
| C | `./scripts/deploy-remote.sh` |
| D | `SHORT=1 bash scripts/remote-e2e.sh` |
| E | **`git commit` + `git push`**（仅 L–D 及适用 k8s-offline 全部通过后） |

**禁止**：仅用本机 `go build` / `npm run build` 收官而不跑 `deploy-local.sh` 与适用远端脚本；禁止未 commit/push 结束回合（用户豁免除外）。

## 快速命令参考

| 场景 | 命令 |
|------|------|
| **本机部署（每次必做）** | `./scripts/deploy-local.sh` |
| **服务端自愈（OpsFleet 机上）** | `bash scripts/fix-aisre-upgrade-loop-hotfix.sh`（= build-all + sync-aisre-backend-env + restart） |
| **版本 env 同步** | `bash scripts/sync-aisre-backend-env.sh`（deploy-opsfleet-remote 已自动调用） |
| 本机 + OpsFleet 构建 | `DEPLOY_LOCAL_OPSFLEET=1 ./scripts/deploy-local.sh` |
| 远端 CLI 同步 | `./scripts/deploy-remote.sh` |
| OpsFleet 全栈 | `./scripts/deploy-opsfleet-remote.sh` |
| 远端自检 | `bash scripts/verify-opsfleet-deployment.sh` |
| CLI 冒烟 | `SHORT=1 bash scripts/remote-e2e.sh` |
| 推 GitHub | `git commit` → `git push origin main` |

环境变量默认值见 `ai-sre-ship`、`opsfleetpilot-ship`。

## 与本 skill 的迭代

新步骤/脚本/故障模式时，同步更新：本文件、对应子 skill、`README.md`、必要时 `monorepo-release.mdc`。
