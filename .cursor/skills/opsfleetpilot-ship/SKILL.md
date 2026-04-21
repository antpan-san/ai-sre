---
name: opsfleetpilot-ship
description: >-
  When OpsFleet paths change in ai-sre monorepo (ft-backend, ft-front, deploy, ansible-agent, opsfleet scripts):
  update README/docs, run deploy-opsfleet-remote.sh, nginx+systemd, health, push. Invoked from release-deploy / monorepo-release.mdc.
---

# OpsFleetPilot 发布与全栈部署（强制工作流）

**总入口**：**`.cursor/skills/release-deploy/SKILL.md`**。本文件在触及 OpsFleet 路径时**叠加**在 `ai-sre-ship` 之上执行。

## 触发条件（在 ai-sre-ship 之上叠加）

在已满足 **`.cursor/rules/monorepo-release.mdc`** 与 **`.cursor/skills/ai-sre-ship/SKILL.md`** 的前提下，若本次变更触及 **OpsFleet** 相关路径或脚本，**另须**完整执行本文件：`ft-backend/`、`ft-front/`、`deploy/`、`ansible-agent/`，或 `scripts/deploy-opsfleet-remote.sh`、`scripts/build-all.sh`、`scripts/verify-opsfleet-deployment.sh`、`PRODUCT_DOC.md` 中与控制台部署强相关的内容。

OpsFleetPilot 与 **ai-sre** CLI **同仓**，仓库根目录：**`/Users/panshuai/Documents/work/code/ai-sre`**。默认远程 **`root@192.168.56.11`**（本地虚拟机、**root** 免密，与 **ai-sre-ship** 相同主机），远端目录 **`/root/sre`**（与 `scripts/deploy-opsfleet-remote.sh` 中 `OPSFLEET_REMOTE_DIR` 默认一致；**不要**与仅 CLI 的 `scripts/deploy-remote.sh` 混淆）。**同一 VM** 还可部署 **K8s 内网制品站**（`deploy/k8s-mirror/`，持久目录 `/var/lib/opsfleet-k8s-mirror`）；后端通过 **`/etc/opsfleet/backend.env`** 的 **`OPSFLEET_K8S_MIRROR_BASE_URL`** 拉取 `manifest.json`（首次全栈部署脚本可创建该文件）。

## 服务范围（「所有服务」）

部署为 **本机二进制 + Nginx + systemd**，**不使用** Docker Compose。

| 组件 | 说明 |
|------|------|
| PostgreSQL / Redis | 由运维在主机上安装与维护；配置见 `ft-backend/conf/config.yaml`（可参考 `deploy/config.production.example.yaml`） |
| 后端 | `bin/opsfleet-backend`，由 systemd 启动，`WorkingDirectory` 为 `ft-backend` |
| 前端静态资源 | 构建在仓库 `dist/web/`，部署时同步到 **`/var/www/opsfleetpilot/`**（`OPSFLEET_WEB_ROOT`），由 Nginx `root` 提供（**勿**用 `/root/...` 作 `root`，`www-data` 无法读） |
| Nginx | 由 `deploy/nginx.opsfleet.conf.template` 生成站点配置，反代 `/ft-api`、`/api`、`/ws`、`/uploads`、`/health` 等 |

**历史**：原 **ft-client**（Go Agent）源码曾在本仓，现已移除；在线部署若仍依赖 Agent 协议，需由运维侧自行提供兼容二进制或从其他分支构建。

**GitHub**：仅源码；`bin/`、`dist/` 等在 `.gitignore` 中，禁止提交构建产物。

## README 维护（Push 前）

在 **`git push` 前** 必须核对或更新根目录 **`README.md`**（及 `docs/opsfleet-README.md` 若涉及历史路径）：

- [ ] 与 **ai-sre 同仓** 的布局描述、`make build-opsfleet`/`scripts/build-all.sh`、产物路径与 **Nginx 端口**（默认 **9080**）是否一致  
- [ ] **`scripts/deploy-opsfleet-remote.sh`**（非 `deploy-remote.sh`）与环境变量（含 `OPSFLEET_BACKEND_PORT`）是否写清  
- [ ] **Agent 与 Web 进程边界**（若文档仍提及在线 Agent）；**仓库不存二进制**  

若变更影响部署或访问方式，**必须**在同一批提交中更新文档。

## 发布后 README 复核

在 **`./scripts/deploy-opsfleet-remote.sh` 成功且健康检查通过** 后，再快速核对 README 与线上行为。

## 执行顺序（上线必做）

与 **`.cursor/skills/release-deploy/SKILL.md`** 中「OpsFleet 上线顺序」表一致；代理须**实际执行**下列命令（除非用户豁免 SSH），不得只写说明。

1. **更新 README**（见上）并暂存相关文档（若本次影响用户可见部署方式）。  
2. **本地快速校验**（可选）：`make vet-opsfleet` 或 `cd ft-backend && go build -o /dev/null .`；`make build-opsfleet` 仅用于本地复现（**勿提交** `bin/`、`dist/`）。  
3. **全栈远程部署**（仓库根）：`./scripts/deploy-opsfleet-remote.sh`  
   - rsync 源码 → 远端 **`scripts/build-all.sh`** → **`bin/opsfleet-backend`**、**`bin/ai-sre`**、**`dist/web/`** → 同步 **`/var/www/opsfleetpilot/`** → 渲染 Nginx → **`systemctl restart opsfleet-backend`**  
   - 每次向 **`/etc/opsfleet/backend.env`** 追加/刷新 **`OPSFLEET_AISRE_BINARY_PATH=<REMOTE_DIR>/bin/ai-sre`**（控制台 **curl 安装 ai-sre**、**GET /ft-api/api/k8s/deploy/cli/ai-sre** 依赖此项）  
4. **部署后自检**（SSH 到部署机，在 **`$OPSFLEET_REMOTE_DIR`** 下）：`bash scripts/verify-opsfleet-deployment.sh`  
   - 必看：**systemd active**、**/health**、**静态 index.html**、**GET /ft-api/api/k8s/deploy/cli/ai-sre?arch=amd64 → 200**、**GET /ft-api/api/k8s/deploy/install-ai-sre.sh** 返回以 **`#!`** 开头的 shell（动态脚本；若 404/HTML 检查 Nginx **`location /ft-api/`** 与后端 **`StripOptionalFtAPIPrefix`**）  
   - **manifest.json**：未部署 `deploy/k8s-mirror` 时 WARN 可接受  
5. **CLI 同步**（同主机常规仍执行）：仓库根 **`./scripts/deploy-remote.sh`**（仅 **ai-sre** 二进制构建，与全栈共用目录时不冲突）。  
6. **冒烟**：仓库根 **`SHORT=1 bash scripts/remote-e2e.sh`**（见 **ai-sre-ship**）。  
7. **失败处理**：构建或 health 失败 → **`journalctl -u opsfleet-backend -n 120`**、**`nginx -t`**、远端 **`go build`** 复现；修复后从步骤 3 重跑。  
8. **Git**：**确认未误加 `bin/`、`dist/`** → `git add` → `commit` → **`git push origin main`**。  
9. **汇报**：`http://<host>:9080/`、verify 摘要、**install-ai-sre.sh** 是否 OK、提交哈希。

**与 K8s 离线 Skill**：若变更命中 `ansible-agent`、`k8s_bundle`、`deploy/k8s-mirror` 等，在 **`git push` 前**还须满足 **`.cursor/skills/k8s-offline-deploy-test/SKILL.md`** 最低限度（`go build`、`gen-k8s-bundle`；能 SSH **192.168.56.11** 时建议验证 manifest）。

**后端路由**：`ft-backend` 使用 **`middleware.StripOptionalFtAPIPrefix()`**，即使 Nginx **`proxy_pass`** 未去掉 **`/ft-api`** 前缀，**`/ft-api/api/...`** 仍可命中 Gin 路由；生产仍推荐模板中带尾斜杠的 **`proxy_pass`**（见 **`deploy/nginx.opsfleet.conf.template`** 注释）。

## 固定参数（可覆盖）

| 环境变量 | 默认 |
|----------|------|
| `OPSFLEET_REMOTE` | `root@192.168.56.11` |
| `OPSFLEET_REMOTE_DIR` | `/root/sre` |
| `OPSFLEET_UI_PORT` | `9080` |
| `OPSFLEET_BACKEND_PORT` | `8080` |
| `OPSFLEET_AISRE_BINARY_PATH` | 由 `deploy-opsfleet-remote.sh` 在远端设为 **`$OPSFLEET_REMOTE_DIR/bin/ai-sre`**（勿手改除非自定义路径） |

## 远程前提

- 已安装 **Go**、**Node/npm**、**Nginx**、**systemd**；主机上已有 **PostgreSQL** 与 **Redis**（或等价连接信息写入 `ft-backend/conf/config.yaml`）。  
- 本机到远程 **SSH 免密**。  
- 防火墙放行 **9080**（或自定义 UI 端口）。

## 与 ai-sre CLI

同仓根目录另有 **ai-sre** 子模块（`main.go`、`internal/`）；其远程同步脚本为 **`scripts/deploy-remote.sh`**（只构建 CLI，不跑 OpsFleet 全栈）。变更 OpsFleet 时使用 **`deploy-opsfleet-remote.sh`**。
