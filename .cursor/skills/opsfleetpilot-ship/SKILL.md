---
name: opsfleetpilot-ship
description: >-
  When OpsFleet paths change in ai-sre monorepo (ft-backend, ft-front, deploy, ansible-agent, opsfleet scripts):
  update README/docs, run deploy-opsfleet-remote.sh, nginx+systemd, health, push. Trigger per monorepo-release.mdc.
---

# OpsFleetPilot 发布与全栈部署（强制工作流）

## 触发条件（在 ai-sre-ship 之上叠加）

在已满足 **`.cursor/rules/monorepo-release.mdc`** 与 **`.cursor/skills/ai-sre-ship/SKILL.md`** 的前提下，若本次变更触及 **OpsFleet** 相关路径或脚本，**另须**完整执行本文件：`ft-backend/`、`ft-front/`、`ft-client/`、`deploy/`、`ansible-agent/`，或 `scripts/deploy-opsfleet-remote.sh`、`scripts/build-all.sh`、`scripts/verify-opsfleet-deployment.sh`、`PRODUCT_DOC.md` 中与控制台部署强相关的内容。

OpsFleetPilot 与 **ai-sre** CLI **同仓**，仓库根目录：**`/Users/panshuai/Documents/work/code/ai-sre`**。默认远程 **`root@172.16.195.128`**，远端目录 **`/root/sre`**（与 `scripts/deploy-opsfleet-remote.sh` 中 `OPSFLEET_REMOTE_DIR` 默认一致；**不要**与仅 CLI 的 `scripts/deploy-remote.sh` 混淆）。

## 服务范围（「所有服务」）

部署为 **本机二进制 + Nginx + systemd**，**不使用** Docker Compose。

| 组件 | 说明 |
|------|------|
| PostgreSQL / Redis | 由运维在主机上安装与维护；配置见 `ft-backend/conf/config.yaml`（可参考 `deploy/config.production.example.yaml`） |
| 后端 | `bin/opsfleet-backend`，由 systemd 启动，`WorkingDirectory` 为 `ft-backend` |
| 前端静态资源 | 构建在仓库 `dist/web/`，部署时同步到 **`/var/www/opsfleetpilot/`**（`OPSFLEET_WEB_ROOT`），由 Nginx `root` 提供（**勿**用 `/root/...` 作 `root`，`www-data` 无法读） |
| Nginx | 由 `deploy/nginx.opsfleet.conf.template` 生成站点配置，反代 `/ft-api`、`/api`、`/ws`、`/uploads`、`/health` 等 |

**不包含**：`ft-client`（Agent 需装在受管机，独立发布）。

**GitHub**：仅源码；`bin/`、`dist/` 等在 `.gitignore` 中，禁止提交构建产物。

## README 维护（Push 前）

在 **`git push` 前** 必须核对或更新根目录 **`README.md`**（及 `docs/opsfleet-README.md` 若涉及历史路径）：

- [ ] 与 **ai-sre 同仓** 的布局描述、`make build-opsfleet`/`scripts/build-all.sh`、产物路径与 **Nginx 端口**（默认 **9080**）是否一致  
- [ ] **`scripts/deploy-opsfleet-remote.sh`**（非 `deploy-remote.sh`）与环境变量（含 `OPSFLEET_BACKEND_PORT`）是否写清  
- [ ] **ft-client** 与 Web 进程边界；**仓库不存二进制**  

若变更影响部署或访问方式，**必须**在同一批提交中更新文档。

## 发布后 README 复核

在 **`./scripts/deploy-opsfleet-remote.sh` 成功且健康检查通过** 后，再快速核对 README 与线上行为。

## 执行顺序

1. **更新 README**（见上）并暂存相关文档。  
2. **本地快速校验**（可选）：`make vet-opsfleet`；`make build-opsfleet` 确认能生成 `bin/` 与 `dist/web/`（勿提交）。  
3. **远程部署**：`./scripts/deploy-opsfleet-remote.sh`（rsync → 远端 `scripts/build-all.sh` → Nginx + systemd → `curl /health`）。  
4. **失败处理**：若构建失败，在远端或本地 `make build-opsfleet` 复现；修复后再推送。  
5. **Git**：`git add -A && git commit && git push`。**确认未误加 `bin/`、`dist/`。**  
6. **汇报**：说明访问地址 `http://<host>:9080/`、服务状态、提交哈希。

## 固定参数（可覆盖）

| 环境变量 | 默认 |
|----------|------|
| `OPSFLEET_REMOTE` | `root@172.16.195.128` |
| `OPSFLEET_REMOTE_DIR` | `/root/sre` |
| `OPSFLEET_UI_PORT` | `9080` |
| `OPSFLEET_BACKEND_PORT` | `8080` |

## 远程前提

- 已安装 **Go**、**Node/npm**、**Nginx**、**systemd**；主机上已有 **PostgreSQL** 与 **Redis**（或等价连接信息写入 `ft-backend/conf/config.yaml`）。  
- 本机到远程 **SSH 免密**。  
- 防火墙放行 **9080**（或自定义 UI 端口）。

## 与 ai-sre CLI

同仓根目录另有 **ai-sre** 子模块（`main.go`、`internal/`）；其远程同步脚本为 **`scripts/deploy-remote.sh`**（只构建 CLI，不跑 OpsFleet 全栈）。变更 OpsFleet 时使用 **`deploy-opsfleet-remote.sh`**。
