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
- [ ] 2. 用 Read 打开并完整执行 ai-sre-ship（deploy-remote.sh、远程冒烟、README、push 相关步骤）
- [ ] 3. 若变更触及 ft-backend/、ft-front/、deploy/、ansible-agent/ 或 OpsFleet 脚本 → 另执行 opsfleetpilot-ship（deploy-opsfleet-remote.sh、verify-opsfleet-deployment.sh）。**deploy-opsfleet-remote** 须在远端生成 **`bin/ai-sre`** 并刷新 **`OPSFLEET_AISRE_BINARY_PATH`**（否则控制台「curl 安装 ai-sre」会 404）
- [ ] 4. 若变更触及 K8s 离线/控制台 K8s/制品镜像相关路径 → 另执行 k8s-offline-deploy-test（见 monorepo-release 第 3 条）
- [ ] 5. git：确认未提交 bin/、dist/；commit；push origin main
- [ ] 6. 向用户汇报：部署脚本结果、健康检查、访问 URL（如 http://<host>:9080/）、提交哈希
```

**顺序约束**：适用时先跑完 **opsfleetpilot-ship / k8s-offline** 中的构建与验证，再与 **ai-sre-ship** 的 push 步骤对齐（与 `monorepo-release.mdc` 第 4 条一致：勿在未跑完适用子 skill 时宣称完成）。

## 与本 skill 的迭代（不断完善）

当发布流程有**新步骤、新脚本、新环境变量或新故障模式**时，代理应**同时**更新：

1. **本文件** — 检查清单或表格中增加一条可执行项；
2. **对应子 skill** — 具体命令与失败处理（`ai-sre-ship`、`opsfleetpilot-ship`、`k8s-offline-deploy-test`）；
3. **`README.md`** — 若影响用户或运维操作，必须在同一批变更中更新；
4. **`monorepo-release.mdc`** — 仅当触发条件或顺序变化时精简同步。

## 快速命令参考（细节以子 skill 为准）

| 场景 | 命令（仓库根） |
|------|----------------|
| 仅 CLI / 通用同步 | `./scripts/deploy-remote.sh` |
| OpsFleet 全栈 | `./scripts/deploy-opsfleet-remote.sh` |
| 远端自检（SSH 登录后） | `bash scripts/verify-opsfleet-deployment.sh` |

环境变量默认值见 `ai-sre-ship` 与 `opsfleetpilot-ship` 中的表格。
