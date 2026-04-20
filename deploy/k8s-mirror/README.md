# K8s 内网制品镜像站（部署在 192.168.56.11 或任意主机）

与 `ansible-agent/inventory/group_vars/all.yml` 中 `download_protocol` + `download_domain` 的 URL 路径一致，供 `get_url` / 节点下载使用。

**实验室环境**：默认制品机 IP **`192.168.56.11`** 与 **ai-sre** / **OpsFleet** 远程部署脚本使用**同一台本地虚拟机**（`root` 免密）时，在该机部署本目录即可同时满足 Ansible 拉取与 OpsFleet 控制台展示。**发布与自检**见仓库 **`.cursor/rules/monorepo-release.mdc`**、**`.cursor/skills/k8s-offline-deploy-test/SKILL.md`**、**`.cursor/skills/opsfleetpilot-ship/SKILL.md`**。

## 持久目录（默认）

| 变量 | 默认 |
|------|------|
| `MIRROR_ROOT` | `/var/lib/opsfleet-k8s-mirror` |

Nginx `root` 应指向同一目录，使 `http://<download_domain>/kubernetes/...` 可访问。

## 快速安装（Ubuntu）

```bash
sudo mkdir -p /var/lib/opsfleet-k8s-mirror
sudo cp deploy/k8s-mirror/mirror.env.example /etc/opsfleet/k8s-mirror.env
# 按需编辑版本与架构
sudo cp deploy/k8s-mirror/k8s-mirror-sync.sh /usr/local/bin/k8s-mirror-sync.sh
sudo chmod +x /usr/local/bin/k8s-mirror-sync.sh
sudo cp deploy/k8s-mirror/k8s-mirror-generate-manifest.sh /usr/local/bin/k8s-mirror-generate-manifest.sh
sudo chmod +x /usr/local/bin/k8s-mirror-generate-manifest.sh

# 首次同步 + 生成 manifest.json（供 OpsFleet 页面展示 SHA）
sudo -E bash /usr/local/bin/k8s-mirror-sync.sh
sudo -E bash /usr/local/bin/k8s-mirror-generate-manifest.sh
```

## Nginx 静态站点示例

`deploy/k8s-mirror/nginx-opsfleet-mirror.conf`：将 `root` 设为 `$MIRROR_ROOT`，并暴露根路径下 `manifest.json`。

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
