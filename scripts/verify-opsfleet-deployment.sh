#!/usr/bin/env bash
# 在部署机上执行：检查端口、systemd、Nginx 与 /health（排障用）
# 用法：在服务器上 cd 到仓库根目录后执行，或 OPSFLEET_ROOT=/path/to/ai-sre bash scripts/verify-opsfleet-deployment.sh
set -euo pipefail

ROOT="${OPSFLEET_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
UI_PORT="${OPSFLEET_UI_PORT:-9080}"
BACKEND_PORT="${OPSFLEET_BACKEND_PORT:-8080}"
STATIC_ROOT="${OPSFLEET_WEB_ROOT:-/var/www/opsfleetpilot}"

echo "=== OpsFleetPilot 部署自检（ROOT=$ROOT）==="

echo "-- systemd: opsfleet-backend --"
systemctl is-active opsfleet-backend && echo "active OK" || { systemctl status opsfleet-backend --no-pager -l || true; exit 1; }

echo "-- 监听端口 --"
ss -tlnp 2>/dev/null | grep -E ":${UI_PORT}\\b|:${BACKEND_PORT}\\b" || echo "(未看到 ss 输出时请安装 iproute2)"

echo "-- 本机经 Nginx 访问 /health --"
curl -sfS "http://127.0.0.1:${UI_PORT}/health" && echo "" || {
  echo "经 Nginx 失败，尝试直连后端 :${BACKEND_PORT}"
  curl -sfS "http://127.0.0.1:${BACKEND_PORT}/health" && echo "" || true
  exit 1
}

echo "-- 静态目录（Nginx root，须 www-data 可读）--"
test -f "$STATIC_ROOT/index.html" && echo "$STATIC_ROOT/index.html OK" || echo "WARN: 缺少 $STATIC_ROOT/index.html（部署脚本应 rsync 自 dist/web）"

if [[ -r /etc/opsfleet/backend.env ]]; then
  # shellcheck disable=SC1091
  set -a
  source /etc/opsfleet/backend.env
  set +a
fi
MIRROR_BASE="${OPSFLEET_K8S_MIRROR_BASE_URL:-}"
if [[ -n "$MIRROR_BASE" ]]; then
  echo "-- K8s 制品 manifest（OPSFLEET_K8S_MIRROR_BASE_URL=$MIRROR_BASE）--"
  if curl -sfS --connect-timeout 8 "${MIRROR_BASE%/}/manifest.json" -o /tmp/.opsfleet-mirror-check.json 2>/dev/null; then
    echo "manifest.json OK ($(wc -c </tmp/.opsfleet-mirror-check.json) bytes)"
    rm -f /tmp/.opsfleet-mirror-check.json
  else
    echo "WARN: 无法拉取 ${MIRROR_BASE%/}/manifest.json — 若需「K8s 制品镜像」页，请在同机部署 deploy/k8s-mirror（见 deploy/k8s-mirror/README.md）"
  fi
else
  echo "-- K8s 制品 manifest：未配置 OPSFLEET_K8S_MIRROR_BASE_URL（可选，见 /etc/opsfleet/backend.env）--"
fi

AISRE="${OPSFLEET_AISRE_BINARY_PATH:-}"
if [[ -n "$AISRE" && -f "$AISRE" ]]; then
  echo "-- ai-sre 公开下载（二进制 $AISRE）--"
  code=$(curl -sS -o /dev/null -w "%{http_code}" "http://127.0.0.1:${UI_PORT}/ft-api/api/k8s/deploy/cli/ai-sre?arch=amd64" || echo "000")
  if [[ "$code" == "200" ]]; then
    echo "GET /ft-api/api/k8s/deploy/cli/ai-sre HTTP 200 OK"
  else
    echo "WARN: GET .../cli/ai-sre 返回 HTTP $code（预期 200；检查 opsfleet-backend 与 install-ai-sre 路由）"
  fi
else
  echo "-- ai-sre 分发：未设置 OPSFLEET_AISRE_BINARY_PATH 或文件不存在（请执行 deploy-opsfleet-remote.sh 刷新 build-all + backend.env）--"
fi

echo "=== 若浏览器仍无法打开，请检查：云安全组/防火墙是否放行 TCP ${UI_PORT}；访问 URL 是否为 http://<服务器IP>:${UI_PORT}/ ==="
