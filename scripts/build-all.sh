#!/usr/bin/env bash
# 在仓库根目录执行：编译后端二进制 + 构建前端静态资源到 dist/web（均不入库）
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

mkdir -p bin dist/web

export GOFLAGS="${GOFLAGS:-} -buildvcs=false"

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

echo "==> go build ai-sre CLI (linux/arm64) -> bin/ai-sre.arm64（供 ARM 控制机 curl 安装）"
(
  cd "$ROOT"
  if GOOS=linux GOARCH=arm64 GOTOOLCHAIN=auto go build -trimpath -ldflags="-s -w" -o "$ROOT/bin/ai-sre.arm64" .; then
    echo "    ai-sre.arm64: OK"
  else
    rm -f "$ROOT/bin/ai-sre.arm64"
    case "$(uname -m 2>/dev/null)" in
      x86_64|amd64)
        echo "ERROR: linux/arm64 交叉编译失败。AMD64 部署机必须产出 bin/ai-sre.arm64，否则 ARM 控制机会因版本 API 与下载包不一致陷入自动升级死循环。" >&2
        exit 1
        ;;
      *)
        echo "    WARN: linux/arm64 交叉编译失败，本机非 amd64 时可忽略" >&2
        ;;
    esac
  fi
)

if [[ -f "$ROOT/bin/ai-sre" && -f "$ROOT/bin/ai-sre.arm64" ]]; then
  V_NATIVE="$("$ROOT/bin/ai-sre" version 2>/dev/null | awk '{print $NF}' || true)"
  V_ARM="$("$ROOT/bin/ai-sre.arm64" version 2>/dev/null | awk '{print $NF}' || true)"
  if [[ -z "$V_ARM" ]]; then
    V_ARM="$(strings "$ROOT/bin/ai-sre.arm64" 2>/dev/null | grep -oE '0\.5\.[0-9]+' | sort -Vu | tail -1 || true)"
  fi
  if [[ -n "$V_NATIVE" && -n "$V_ARM" && "$V_NATIVE" != "$V_ARM" ]]; then
    echo "ERROR: bin/ai-sre ($V_NATIVE) 与 bin/ai-sre.arm64 ($V_ARM) 版本不一致" >&2
    exit 1
  fi
fi

echo "==> npm build ft-front -> dist/web"
(
  cd ft-front
  # 远端不 rsync node_modules：须按 lockfile 全量重装，否则会缺新依赖
  npm ci
  npm run build
  rm -rf "$ROOT/dist/web"/*
  cp -r dist/* "$ROOT/dist/web/"
)

echo "==> OK"
echo "    Backend: $ROOT/bin/opsfleet-backend"
echo "    K8s mirror-serve: $ROOT/bin/opsfleet-k8s-mirror-serve  (optional; 制品机 install + systemd)"
echo "    Executor: $ROOT/bin/opsfleet-executor  (copy to managed hosts as needed)"
echo "    ai-sre:   $ROOT/bin/ai-sre  (API 公开下载；arm64 见 bin/ai-sre.arm64 与 OPSFLEET_AISRE_BINARY_PATH_ARM64)"
echo "    Static:  $ROOT/dist/web/"
echo "    Run API from directory ft-backend (conf/config.yaml):"
echo "      cd ft-backend && ../bin/opsfleet-backend"
