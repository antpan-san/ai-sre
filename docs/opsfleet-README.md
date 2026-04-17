> **归档说明**：原独立仓库 `opsfleetpilot` 已并入 **[ai-sre 仓库根目录](../README.md)**。下列为迁移前 README，便于对照；**命令与默认路径以根目录 `README.md` 为准**（例如远程部署脚本现名为 `scripts/deploy-opsfleet-remote.sh`，默认远端目录 **`/root/sre`**；自检脚本为 `scripts/verify-opsfleet-deployment.sh`）。

---

# OpsFleetPilot

企业级服务器运维管理与 Kubernetes 集群部署平台：**Web 前端（Vue3）+ API（Golang/Gin）**；历史 **ft-client** Agent 源码曾在此仓，**现已移除**，在线能力仍以服务端 API 与（若部署）兼容 Agent 二进制为准。可作为整体运维 / AIOps 产品体系的 **Web UI 控制台**（与 CLI 类工具如 ai-sre 等配合使用）。

详细功能与模块说明见 **[PRODUCT_DOC.md](../PRODUCT_DOC.md)**。

## 子项目

| 目录 | 说明 |
|------|------|
| `ft-front` | Vue3 + Element Plus + Vite |
| `ft-backend` | Gin + PostgreSQL + Redis + WebSocket |
| ~~`ft-client`~~ | （已自本仓移除）原 Go Agent，部署在受管机器 |

## 仓库与构建产物

本仓库 **只存放源代码**。以下内容在 **`.gitignore`** 中，**不要提交到 GitHub**：

- `bin/` — 后端编译产物（如 `bin/opsfleet-backend`）
- `dist/` — 前端构建产物（如 `dist/web/`）
- `ft-front/dist/`、`ft-front/node_modules/` 等

在**仓库根目录**从源码生成二进制与静态资源：

```bash
make build-opsfleet
# 或
bash scripts/build-all.sh
```

产物路径：

- 后端可执行文件：`bin/opsfleet-backend`（在 `ft-backend` 目录下配合 `conf/config.yaml` 运行）
- 前端静态文件：`dist/web/`（由 Nginx 提供）

## 本地开发

- **后端**：进入 `ft-backend`，配置 `conf/config.yaml`，`go run .`（默认 `:8080`）。
- **前端**：进入 `ft-front`，`npm install && npm run dev`（Vite 代理 `/ft-api` → 后端）。

## 生产部署（二进制 + Nginx + systemd）

不使用 Docker：在目标机上安装 **Go、Node/npm、Nginx、systemd**，并自行运行 **PostgreSQL** 与 **Redis**（监听本机，如 `127.0.0.1`）。

1. 参考 **`deploy/config.production.example.yaml`** 编写 `ft-backend/conf/config.yaml`（数据库、Redis、JWT 密钥等）。
2. 在仓库根目录执行 **`make build-opsfleet`**，得到 `bin/opsfleet-backend` 与 `dist/web/`。
3. 将 **`dist/web/`** 同步到 **`/var/www/opsfleetpilot/`**（或自定义目录）并 `chown -R www-data:www-data`，再使用 **`deploy/nginx.opsfleet.conf.template`** 生成站点配置（`@OPSFLEET_WEB_ROOT@` 指静态目录，勿指向 `/root/...`，否则 Nginx 无法读文件）。反代 `/ft-api`、`/api`、`/ws`、`/uploads`、`/health` 等到后端。
4. 使用 **`deploy/opsfleet-backend.service.example`** 生成 systemd 单元，`WorkingDirectory` 指向 `ft-backend`，`ExecStart` 指向 `bin/opsfleet-backend`。

远程一键发布（**仅同步源码**，在远端编译；需本机 SSH 免密，远端已安装上述依赖）：

```bash
chmod +x scripts/deploy-opsfleet-remote.sh
./scripts/deploy-opsfleet-remote.sh
```

环境变量：

| 变量 | 默认 | 说明 |
|------|------|------|
| `OPSFLEET_REMOTE` | `root@172.16.195.128` | SSH 目标 |
| `OPSFLEET_REMOTE_DIR` | `/root/sre`（与 ai-sre 同仓部署目录一致） | 远端仓库根路径 |
| `OPSFLEET_UI_PORT` | `9080` | Nginx 对外端口 |
| `OPSFLEET_BACKEND_PORT` | `8080` | 本机后端监听端口 |
| `OPSFLEET_WEB_ROOT` | `/var/www/opsfleetpilot` | Nginx 静态根目录（`www-data` 可读） |

脚本会：rsync 源码（排除 `bin`、`dist`、`node_modules` 等）→ 远端执行 `scripts/build-all.sh` → **`dist/web` 同步到 `OPSFLEET_WEB_ROOT` 并改属主** → 写入 Nginx 与 systemd → **尽力放行防火墙端口**（ufw / firewalld）→ **SELinux 下允许 Nginx 反代** → 探测本机 `http://127.0.0.1:<UI_PORT>/health`。若后端未起来，脚本会 **打印 `journalctl` 并以非零退出**，避免“部署成功但无法访问”。

部署完成后若 **浏览器仍无法打开**，请逐项确认：

1. **访问地址** 使用 `http://<服务器 IP>:9080/`（或你设置的 `OPSFLEET_UI_PORT`），不是省略端口（除非你把 Nginx 配在 80）。
2. **云厂商安全组 / 网络 ACL** 放行 **入站 TCP** 到该 UI 端口（脚本只能改本机 ufw/firewalld，**不能**改云平台规则）。
3. 在服务器上执行：`bash scripts/verify-opsfleet-deployment.sh`（需在仓库根目录，或通过 `OPSFLEET_ROOT` 指向部署路径），查看 systemd 与端口。
4. **PostgreSQL**：`ft-backend/conf/config.yaml` 中账号密码须与实例一致，且库 `opsfleetpilot` 已创建；否则后端进程会退出，页面为 502。可查看：`journalctl -u opsfleet-backend -e`。

## 与 Agent（历史 ft-client）的关系

控制台与 API 以 **本机进程 + Nginx** 方式运行。若使用 **在线 Agent** 流程，受管机需运行 **兼容协议** 的 Agent（原 **ft-client** 配置形态如下，供对照）；**本仓库不再提供** ft-client 源码。

```yaml
server:
  url: "http://<服务器IP>:9080"
```

Agent 请求路径为 **`/api/v1/*`**（不经 `/ft-api` 前缀）；Nginx 已同时反代 **`/api/`** 与 **`/ft-api/`**（浏览器用后者），与现有客户端协议一致。

## VMware Fusion（Mac）与 Ubuntu 静态 IP

Fusion 默认 **NAT（vmnet8）** 一般为 **`172.16.195.0/24`**，宿主机侧 **`172.16.195.1`**，虚拟机网关 **`172.16.195.2`**。若在 Ubuntu 里使用 **`192.168.56.x`** 而 **未** 把 Fusion 子网改成 `192.168.56.0/24`，则 **Mac 与虚拟机无法互通**。

- **推荐（与当前 Fusion 一致）**：在虚拟机内使用 **`scripts/vmware-fusion-vmnet8-netplan.yaml`** 中的示例（如 **`172.16.195.11/24`**，网关 **`172.16.195.2`**），再 `sudo netplan apply`。
- **若必须使用 `192.168.56.11`**：在 **Mac 上**用 `sudo` 执行 **`scripts/vmware-fusion-nat-to-192.168.56.sh`**（先关虚拟机与 Fusion），再按 **`scripts/vmware-fusion-192.168.56-netplan.yaml`** 配置 Ubuntu，网关 **`192.168.56.2`**。

## 发布与文档约定

代码变更后的发布流程、README 维护要求见 **`.cursor/skills/opsfleetpilot-ship/SKILL.md`**。
