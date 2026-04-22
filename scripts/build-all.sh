#!/usr/bin/env bash
# 在仓库根目录执行：编译后端二进制 + 构建前端静态资源到 dist/web（均不入库）
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

mkdir -p bin dist/web

echo "==> go build ft-backend -> bin/opsfleet-backend"
(
  cd ft-backend
  go build -trimpath -ldflags="-s -w" -o "$ROOT/bin/opsfleet-backend" .
)

echo "==> go build opsfleet-k8s-mirror-serve (K8s 制品站 miss 时拉公网并落盘，见 deploy/k8s-mirror/)"
(
  cd ft-backend
  go build -trimpath -ldflags="-s -w" -o "$ROOT/bin/opsfleet-k8s-mirror-serve" ./cmd/opsfleet-k8s-mirror-serve
)

echo "==> go build opsfleet-executor (same skill engine as ai-sre) -> bin/opsfleet-executor"
(
  cd "$ROOT"
  go build -trimpath -ldflags="-s -w" -o "$ROOT/bin/opsfleet-executor" ./cmd/opsfleet-executor
)

echo "==> go build ai-sre CLI -> bin/ai-sre (K8s 确认页 curl 安装与 deploy-opsfleet 分发)"
(
  cd "$ROOT"
  go build -trimpath -ldflags="-s -w" -o "$ROOT/bin/ai-sre" .
)

echo "==> npm build ft-front -> dist/web"
(
  cd ft-front
  if [[ ! -d node_modules ]]; then
    npm ci
  fi
  npm run build
  rm -rf "$ROOT/dist/web"/*
  cp -r dist/* "$ROOT/dist/web/"
)

echo "==> OK"
echo "    Backend: $ROOT/bin/opsfleet-backend"
echo "    K8s mirror-serve: $ROOT/bin/opsfleet-k8s-mirror-serve  (optional; 制品机 install + systemd)"
echo "    Executor: $ROOT/bin/opsfleet-executor  (copy to managed hosts as needed)"
echo "    ai-sre:   $ROOT/bin/ai-sre  (API 公开下载，见 OPSFLEET_AISRE_BINARY_PATH)"
echo "    Static:  $ROOT/dist/web/"
echo "    Run API from directory ft-backend (conf/config.yaml):"
echo "      cd ft-backend && ../bin/opsfleet-backend"
