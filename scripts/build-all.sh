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

echo "==> go build opsfleet-executor (same skill engine as ai-sre) -> bin/opsfleet-executor"
(
  cd "$ROOT"
  go build -trimpath -ldflags="-s -w" -o "$ROOT/bin/opsfleet-executor" ./cmd/opsfleet-executor
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
echo "    Executor: $ROOT/bin/opsfleet-executor  (copy to managed hosts as needed)"
echo "    Static:  $ROOT/dist/web/"
echo "    Run API from directory ft-backend (conf/config.yaml):"
echo "      cd ft-backend && ../bin/opsfleet-backend"
