---
name: release-deploy
description: >-
  Publishes ai-sre monorepo after code changes: run remote deploy scripts, verify health, README, git push.
  Use after any edit under /Users/panshuai/Documents/work/code/ai-sre, when the user says 发布/部署/上线/ship, or when finishing a task that modified this repo.
---

# 发布部署（总入口）

本 skill 是 **ai-sre 同仓** 发布流程的**唯一总入口**。代理在**完成本仓库内任意代码或配置变更**后，**必须**按本节执行，**不要**等用户再说「发布」——除非用户在本轮对话中**明确**豁免（不 SSH / 不 push / 仅本地）。

## 仓库与规则

| 项 | 路径或说明 |
|----|------------|
| 仓库根 | `/Users/panshuai/Documents/work/code/ai-sre` |
| Cursor 规则（alwaysApply） | `.cursor/rules/monorepo-release.mdc` |
| CLI 同步 + 冒烟 + README + push | `.cursor/skills/ai-sre-ship/SKILL.md` |
| OpsFleet 全栈（Nginx、前端 dist、后端 systemd） | `.cursor/skills/opsfleetpilot-ship/SKILL.md` |
| K8s 离线包 / 控制台 K8s 页 / 制品镜像 | `.cursor/skills/k8s-offline-deploy-test/SKILL.md` |

## 执行顺序（必须）

复制并逐项完成：

```
发布部署检查清单
- [ ] 1. 用 Read 打开 monorepo-release.mdc，确认无用户豁免
- [ ] 2. 若变更触及 OpsFleet → 先执行 opsfleetpilot-ship 全栈（见下「OpsFleet 上线顺序」），再执行 ai-sre-ship 的 CLI 与 push；仅 CLI 改动可只跑 ai-sre-ship
- [ ] 3. 用 Read 打开并完整执行 ai-sre-ship：`deploy-remote.sh`、`SHORT=1 bash scripts/remote-e2e.sh`（或全量 remote-e2e）、README 核对、push 相关步骤
- [ ] 4. OpsFleet 全栈（触及 ft-backend/、ft-front/、deploy/、ansible-agent/、OpsFleet 脚本时）：`./scripts/deploy-opsfleet-remote.sh` → SSH 执行 `bash scripts/verify-opsfleet-deployment.sh`。验证须含：**/health**、**GET /ft-api/api/k8s/deploy/cli/ai-sre**、**GET /ft-api/api/k8s/deploy/install-ai-sre.sh**（脚本以 `#!` 开头）。**deploy-opsfleet-remote** 须在远端生成 **`bin/ai-sre`** 并写入 **`/etc/opsfleet/backend.env`** 的 **`OPSFLEET_AISRE_BINARY_PATH`**
- [ ] 5. 若变更触及 K8s 离线/控制台 K8s/制品镜像 → 另执行 k8s-offline-deploy-test（见 monorepo-release 第 3 条）
- [ ] 6. git：确认未提交 bin/、dist/；commit；push origin main
- [ ] 7. 向用户汇报：各脚本退出码、verify 输出摘要、访问 URL（如 http://<host>:9080/）、提交哈希
```

### OpsFleet 上线顺序（与子 skill 一致）

| 步骤 | 动作 |
|------|------|
| A | 仓库根：`./scripts/deploy-opsfleet-remote.sh`（rsync → 远端 `build-all.sh` → Nginx → systemd → 本机 /health） |
| B | SSH 部署机：`bash scripts/verify-opsfleet-deployment.sh`（含 install-ai-sre.sh 探测） |
| C | 仓库根：`./scripts/deploy-remote.sh`（仅 ai-sre CLI 同步构建，与全栈独立但同主机同目录时常规仍执行） |
| D | 仓库根：`SHORT=1 bash scripts/remote-e2e.sh`（本地 vet + 远程 CLI 冒烟） |
| E | `git add` / `commit` / `push`（勿纳入 bin/、dist/） |

**后端说明**：`ft-backend` 已挂载 **`StripOptionalFtAPIPrefix`**，Nginx 将 **`/ft-api/api/...`** 整段转发时也能命中路由；模板仍要求 **`proxy_pass .../`** 带尾斜杠（见 `deploy/nginx.opsfleet.conf.template` 注释）。

**顺序约束**：适用时先跑完 **opsfleetpilot-ship / k8s-offline** 中的构建与验证，再与 **ai-sre-ship** 的 push 步骤对齐（与 `monorepo-release.mdc` 第 4 条一致：勿在未跑完适用子 skill 时宣称完成）。

## 与本 skill 的迭代（不断完善）

当发布流程有**新步骤、新脚本、新环境变量或新故障模式**时，代理应**同时**更新：

1. **本文件** — 检查清单或表格中增加一条可执行项；
2. **对应子 skill** — 具体命令与失败处理（`ai-sre-ship`、`opsfleetpilot-ship`、`k8s-offline-deploy-test`）；
3. **`README.md`** — 若影响用户或运维操作，必须在同一批变更中更新；
4. **`monorepo-release.mdc`** — 仅当触发条件或顺序变化时精简同步。

## 快速命令参考（细节以子 skill 为准）

| 场景 | 命令（仓库根或 SSH 内） |
|------|----------------|
| 仅 CLI / 通用同步 | `./scripts/deploy-remote.sh` |
| OpsFleet 全栈 | `./scripts/deploy-opsfleet-remote.sh` |
| 远端自检（在部署机上） | `bash scripts/verify-opsfleet-deployment.sh`（含 **install-ai-sre.sh**、cli/ai-sre、manifest、/health） |
| CLI 冒烟（本地触发 SSH） | `SHORT=1 bash scripts/remote-e2e.sh` |

环境变量默认值见 `ai-sre-ship` 与 `opsfleetpilot-ship` 中的表格。
