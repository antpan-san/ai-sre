#!/usr/bin/env bash
# 将仓库内 deploy/k8s-mirror 同步到实验室制品机 192.168.56.11，并执行多版本制品拉取 + manifest 生成。
# 制品机须能访问 dl.k8s.io 与 GitHub（代拉公网）；日志在远端 /tmp/k8s-mirror-sync.log
set -euo pipefail
REMOTE="${K8S_MIRROR_REMOTE:-root@192.168.56.11}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MIRROR_DIR="$ROOT/deploy/k8s-mirror"

echo "==> rsync k8s-mirror -> $REMOTE:/root/sre/deploy/k8s-mirror"
ssh -o BatchMode=yes -o ConnectTimeout=20 "$REMOTE" "mkdir -p /root/sre/deploy"
rsync -avz \
  "$MIRROR_DIR/" "$REMOTE:/root/sre/deploy/k8s-mirror/"

echo "==> install /etc/opsfleet/k8s-mirror.env + k8s-mirror-versions.txt"
scp -o BatchMode=yes -o ConnectTimeout=20 \
  "$MIRROR_DIR/mirror.env.example" \
  "$REMOTE:/etc/opsfleet/k8s-mirror.env"
scp -o BatchMode=yes -o ConnectTimeout=20 \
  "$MIRROR_DIR/k8s-mirror-versions.txt" \
  "$REMOTE:/etc/opsfleet/k8s-mirror-versions.txt"

echo "==> remote: sync (多版本，耗时长)"
ssh -o BatchMode=yes -o ConnectTimeout=30 \
  -o ServerAliveInterval=30 \
  "$REMOTE" \
  "ENV_FILE=/etc/opsfleet/k8s-mirror.env bash /root/sre/deploy/k8s-mirror/k8s-mirror-sync.sh 2>&1 | tee /tmp/k8s-mirror-sync.log"

echo "==> remote: generate manifest"
ssh -o BatchMode=yes "$REMOTE" \
  "ENV_FILE=/etc/opsfleet/k8s-mirror.env bash /root/sre/deploy/k8s-mirror/k8s-mirror-generate-manifest.sh"

echo "==> OK. 查看远端: tail -50 /tmp/k8s-mirror-sync.log"
echo "    manifest: http://192.168.56.11/manifest.json（Nginx root 须为 /var/lib/opsfleet-k8s-mirror）"
