#!/usr/bin/env bash
# 在 K8s 实验集群 master 上构建镜像并部署（默认 root@192.168.56.101）。
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
SSH_TARGET="${SSH_TARGET:-root@192.168.56.101}"
REMOTE_DIR="${REMOTE_DIR:-/root/memleak-demo}"
IMAGE="${IMAGE:-memleak-demo:lab}"

echo "==> sync sources to ${SSH_TARGET}:${REMOTE_DIR}"
ssh "$SSH_TARGET" "mkdir -p '$REMOTE_DIR/k8s'"
if [[ ! -f "$ROOT/memleak-demo-linux" ]]; then
  echo "==> cross-compile linux/arm64 (lab nodes are aarch64)"
  (cd "$ROOT" && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags='-s -w' -o memleak-demo-linux .)
fi

rsync -az --delete \
  "$ROOT/memleak-demo-linux" "$ROOT/Dockerfile.scratch" \
  "$SSH_TARGET:$REMOTE_DIR/"
rsync -az "$ROOT/k8s/" "$SSH_TARGET:$REMOTE_DIR/k8s/"

echo "==> scratch image build + import into containerd (k8s.io)"
ssh "$SSH_TARGET" bash -s <<EOF
set -euo pipefail
cd '$REMOTE_DIR'
cp Dockerfile.scratch Dockerfile
docker build -t '$IMAGE' .
docker save '$IMAGE' | ctr -n k8s.io images import -
ctr -n k8s.io images ls | grep memleak-demo || true
kubectl apply -f k8s/deployment.yaml
kubectl -n memleak-demo rollout status deployment/memleak-demo --timeout=120s
kubectl -n memleak-demo get pods -o wide
EOF

echo "==> done. Check: kubectl -n memleak-demo logs -f deploy/memleak-demo"
