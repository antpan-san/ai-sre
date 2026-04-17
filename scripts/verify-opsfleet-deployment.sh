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

echo "=== 若浏览器仍无法打开，请检查：云安全组/防火墙是否放行 TCP ${UI_PORT}；访问 URL 是否为 http://<服务器IP>:${UI_PORT}/ ==="
