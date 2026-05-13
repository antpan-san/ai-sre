# K8s 内网制品镜像站（部署在 192.168.56.11 或任意主机）

与 `ansible-agent/inventory/group_vars/all.yml` 中 `download_protocol` + `download_domain` 的 URL 路径一致，供 `get_url` / 节点下载使用。

**实验室环境**：默认制品机 IP **`192.168.56.11`** 与 **ai-sre** / **OpsFleet** 远程部署脚本使用**同一台本地虚拟机**（`root` 免密）时，在该机部署本目录即可同时满足 Ansible 拉取与 OpsFleet 控制台展示。**发布与自检**见仓库 **`.cursor/rules/monorepo-release.mdc`**、**`.cursor/skills/k8s-offline-deploy-test/SKILL.md`**、**`.cursor/skills/opsfleetpilot-ship/SKILL.md`**。

## 磁盘空间

多版本 × 双架构 **kubernetes-server**  tarball 体积大（单套约数百 MB～1GB+），与 **etcd / CNI** 合计常需 **约 5～8GB** 以上空闲；与 PostgreSQL、容器、日志同盘时请先预留空间，避免根分区写满导致数据库与 OpsFleet 异常。

## 持久目录（默认）

| 变量 | 默认 |
|------|------|
| `MIRROR_ROOT` | `/var/lib/opsfleet-k8s-mirror` |

Nginx `root` 应指向同一目录，使 `http://<download_domain>/kubernetes/...` 可访问。

## 从本机仓库一键同步到 192.168.56.11

在**开发机**上（已配置 `ssh root@192.168.56.11`）执行：

```bash
cd /path/to/ai-sre
./scripts/k8s-mirror-sync-remote-11.sh
```

会 rsync 本目录脚本、`/etc/opsfleet/k8s-mirror.env`、版本列表，并在远端拉取**全部**部署页 K8s 版本所需 tar/tgz 与 **etcd / CNI**，最后生成 `manifest.json`。

## 与部署页 K8s 版本对齐

- 可选版本列表在 **数据库种子** `ft-backend/database.initK8sVersions` 与仓库 **`deploy/k8s-mirror/k8s-mirror-versions.txt`** 中一一对应；增删版本时请三处同步（或只改 `k8s-mirror-versions.txt` 与 DB 种子，保持字符串集合一致）。
- `k8s-mirror-sync.sh` 会拉取 `KUBERNETES_VERSIONS`（或版本文件/单版本）中 **每个** `kubernetes-server-linux-{amd64,arm64}.tar.gz` 到 `MIRROR_ROOT/kubernetes/<ver>/<arch>/`，并各拉一套 **etcd**、**CNI**（与 `group_vars` 中 `etcd_version` / `cni_plugins_version` 对齐）。

## 快速安装（Ubuntu）

```bash
sudo mkdir -p /var/lib/opsfleet-k8s-mirror
sudo cp deploy/k8s-mirror/mirror.env.example /etc/opsfleet/k8s-mirror.env
# 多版本 K8s 已在 example 的 KUBERNETES_VERSIONS 中；也可复制版本列表供脚本读取:
sudo cp deploy/k8s-mirror/k8s-mirror-versions.txt /etc/opsfleet/k8s-mirror-versions.txt
# 若只用文件列表，在 k8s-mirror.env 中增加: export KUBERNETES_VERSIONS_FILE=/etc/opsfleet/k8s-mirror-versions.txt

sudo cp deploy/k8s-mirror/k8s-mirror-sync.sh /usr/local/bin/k8s-mirror-sync.sh
sudo chmod +x /usr/local/bin/k8s-mirror-sync.sh
sudo cp deploy/k8s-mirror/k8s-mirror-generate-manifest.sh /usr/local/bin/k8s-mirror-generate-manifest.sh
sudo chmod +x /usr/local/bin/k8s-mirror-generate-manifest.sh

# 从仓库根执行时可直接用未安装的脚本同目录的 k8s-mirror-versions.txt
# 首次同步（会下载多个版本 × 双架构，体积大、耗时长）+ 生成 manifest.json
sudo -E bash /usr/local/bin/k8s-mirror-sync.sh
sudo -E bash /usr/local/bin/k8s-mirror-generate-manifest.sh
```

## 按需拉取（边下边存，推荐与全量预同步二选一或并存）

当部署页/Inventory 引用了**尚未**由 `k8s-mirror-sync.sh` 预置的路径（或临时新增版本/架构）时，控制机 `get_url` 会请求 `http://<download_domain>/kubernetes/...` 等。若磁盘上**不存在**该文件，可依赖 **`opsfleet-k8s-mirror-serve`**：

- 监听 **`127.0.0.1:8090`**（`LISTEN` 可改，勿对公网裸曝）。
- 与 **`k8s-mirror-sync.sh` 相同**的上游 URL（`K8S_UPSTREAM`=`https://dl.k8s.io` 等，见 `mirror.env.example`），将 tarball **写入同一 `MIRROR_ROOT`** 目录布局，**持久保存**；下次请求直接由 Nginx `try_files` 读盘，不再访问公网。
- **K8s** 制品在下载后会尝试拉取对应 **`.sha512` 并强校验**（与全量预拉一致口径）；**etcd / CNI** 仅做 HTTP 200 与落盘。

**制品机需能访问公网**（至少 `dl.k8s.io`、`github.com` release），或自行把上游改为可达的国内镜像，并在 `k8s-mirror.env` 中设置 `K8S_UPSTREAM` / `ETCD_UPSTREAM` / `CNI_UPSTREAM`。

安装（在仓库经 `build-all.sh` 或仅编译该二进制后）：

```bash
sudo cp /path/to/ai-sre/bin/opsfleet-k8s-mirror-serve /usr/local/bin/
sudo cp deploy/k8s-mirror/opsfleet-k8s-mirror-serve.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now opsfleet-k8s-mirror-serve
curl -sS 127.0.0.1:8090/health
```

Nginx 使用本目录 **`nginx-opsfleet-mirror.conf`**：先 `try_files` 读 `$MIRROR_ROOT`，未命中再 **`proxy_pass` 到 `127.0.0.1:8090`** 触发上述落盘。更新配置后 `sudo nginx -t && sudo systemctl reload nginx`。

新增文件落盘后，**控制台 manifest** 与部署页若需展示新条目，可再执行 `k8s-mirror-generate-manifest.sh` 或等定时任务，使 `manifest.json` 与磁盘一致。

## Nginx 静态站点示例

`deploy/k8s-mirror/nginx-opsfleet-mirror.conf`：将 `root` 设为 `$MIRROR_ROOT`；`/` 在「未命中磁盘」时反代到 **opsfleet-k8s-mirror-serve**；根路径下 `manifest.json` 仍由全量/定时脚本预生成时直出，未生成时返回 404（不属于按需逻辑）。

## systemd 定时同步

```bash
sudo cp deploy/k8s-mirror/k8s-mirror-sync.{service,timer} /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now k8s-mirror-sync.timer
```

## OpsFleet 后端

配置环境变量（任选其一）：

- `OPSFLEET_K8S_MIRROR_MANIFEST_URL`：完整 manifest URL（如 `http://192.168.56.11/manifest.json`）
- 或 `OPSFLEET_K8S_MIRROR_BASE_URL`：默认 `http://192.168.56.11`，实际请求 `{BASE}/manifest.json`

前端菜单：**服务与交付 → K8s 制品镜像**。

## 国外机作中转（跨境 / 小盘）

典型拓扑：**国外 VPS** 能访问 `dl.k8s.io` / `github.com`，**国内控制机**仅访问国外机 HTTP(S)。将本目录栈部署在国外机，控制台「内网制品地址」填 **国外机公网域名或 IP**（即 `download_domain` 指向该主机）。

- **按需落盘（省磁盘）**：部署 **Nginx + `opsfleet-k8s-mirror-serve`** 后，**不必**先跑全量 `k8s-mirror-sync.sh`；国内第一次 `GET` 缺失路径时由 serve **回源公网并落盘**。
- **TTL 清理**：小盘可启用 **`k8s-mirror-ttl-cleanup.timer`**（见下），按 `mtime` 删除超过 `TTL_DAYS` 天的 `*.tar.gz` / `*.tgz` / `*.tar`（**保留** `manifest.json`）。清理后下次访问会再次回源，延迟增加属预期。
- **TLS**：公网访问建议 **HTTPS + 正式证书**；`PUBLIC_BASE_URL` 与控制台填写的协议一致。

### TTL 定时清理（systemd）

```bash
sudo cp deploy/k8s-mirror/k8s-mirror-ttl-cleanup.sh /usr/local/bin/
sudo chmod +x /usr/local/bin/k8s-mirror-ttl-cleanup.sh
sudo cp deploy/k8s-mirror/k8s-mirror-ttl-cleanup.{service,timer} /etc/systemd/system/
# 可选：在 /etc/opsfleet/k8s-mirror.env 增加 export TTL_DAYS=7
sudo systemctl daemon-reload
sudo systemctl enable --now k8s-mirror-ttl-cleanup.timer
# 干跑：sudo TTL_DAYS=14 DRY_RUN=1 bash /usr/local/bin/k8s-mirror-ttl-cleanup.sh
```

### 二级镜像站（国内再 rsync 一级）

若国内访问国外仍不稳定：在国内机房另设一台 **二级** 仅 `rsync` 国外 `MIRROR_ROOT`（或定时拉取），控制台 `download_domain` 指向二级；国外机作一级缓存。可参考仓库 **`scripts/k8s-mirror-sync-remote-11.sh`** 思路扩展。

### OpsFleet 控制台 API（中转预检 / warm）

后端（需 JWT）：

- `GET /ft-api/api/k8s/deploy/relay/preflight`：只读探测 **relay `/health`**、**manifest.json**；可选 query `primary_probe_url` 对公网 tarball 做 **Range 0-0** 探测（**不下载整包**）。
- `POST /ft-api/api/k8s/deploy/relay/warm`：body `{"paths":["/kubernetes/v1.35.4/amd64/kubernetes-server-linux-amd64.tar.gz"]}`，对 **relay 基址** 发起 GET 触发按需落盘（**应在用户已决定执行安装后**由脚本调用，避免仅生成命令就占带宽）。

环境变量（**OpsFleet 后端** `/etc/opsfleet/backend.env`）：

- `OPSFLEET_K8S_MIRROR_BASE_URL` / `OPSFLEET_K8S_MIRROR_MANIFEST_URL`：与制品目录页一致。
- `OPSFLEET_K8S_RELAY_BASE_URL`（可选）：与 mirror 不同主机时单独指定 relay 根 URL（无则回退 mirror base）。

### 离线包内「国内优先公网、失败再走中转」

选择 **阿里云** 镜像源打 zip 时，包内包含 **`resource_sources.json`** + **`resource_routing_relay_overlay.yml`**。执行 **`install.sh`** 时在控制机对 **dl.k8s.io** 做短探测，失败则 **追加 overlay** 将 `image_source` 切为 **default** 并走 `download_domain` 布局（与 ansible `group_vars` 一致）。跳过：`OPSFLEET_SKIP_CLIENT_ROUTE=1`。该逻辑**不调用 AI**。

### OCI / 任意容器镜像（后续阶段）

HTTP 制品站不服务 **OCI registry 协议**。若需 Calico 等镜像经国外中转，需在国外部署 **Registry pull-through**（或 Harbor Proxy Cache）并在节点 **containerd `registry.mirrors`** 中配置；与本 HTTP mirror 为**两条线**，见产品路线图（控制台统一镜像源字段待扩展）。
