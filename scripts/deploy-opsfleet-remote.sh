#!/usr/bin/env bash
# 同步源码 → 远程本机 go/npm 编译 → Nginx + systemd（无 Docker）
# GitHub 仅源码；bin/、dist/ 在 .gitignore 中，由部署机在本地生成。
set -euo pipefail

REMOTE="${OPSFLEET_REMOTE:-root@192.168.56.11}"
# 与 ai-sre 默认部署目录一致（同仓单拷贝）；可用 OPSFLEET_REMOTE_DIR 覆盖
REMOTE_DIR="${OPSFLEET_REMOTE_DIR:-/root/sre}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
UI_PORT="${OPSFLEET_UI_PORT:-9080}"
BACKEND_PORT="${OPSFLEET_BACKEND_PORT:-8080}"
WEB_ROOT="${OPSFLEET_WEB_ROOT:-/var/www/opsfleetpilot}"

echo "==> Ensuring remote dir: $REMOTE:$REMOTE_DIR"
ssh -o BatchMode=yes -o ConnectTimeout=20 "$REMOTE" "mkdir -p '$REMOTE_DIR'"

echo "==> Rsync source (exclude build artifacts)"
rsync -avz \
  --exclude '.git' \
  --exclude 'ai-sre' \
  --exclude 'bin' \
  --exclude 'dist' \
  --exclude 'ft-front/node_modules' \
  --exclude 'ft-front/dist' \
  --exclude '.DS_Store' \
  --exclude '.env' \
  --exclude '*.zip' \
  "$ROOT/" \
  "$REMOTE:$REMOTE_DIR/"

echo "==> Remote: build + nginx + systemd + firewall/selinux checks"
ssh -o BatchMode=yes -o ConnectTimeout=300 "$REMOTE" \
  bash -s "$REMOTE_DIR" "$UI_PORT" "$BACKEND_PORT" "$WEB_ROOT" <<'REMOTE_SCRIPT'
set -euo pipefail
R="$1"
UI="$2"
BP="$3"
WEB_ROOT="$4"

command -v go >/dev/null || { echo "ERROR: go not installed on remote"; exit 1; }
command -v node >/dev/null || { echo "ERROR: node not installed on remote"; exit 1; }
command -v npm >/dev/null || { echo "ERROR: npm not installed on remote"; exit 1; }
command -v nginx >/dev/null || { echo "ERROR: nginx not installed on remote"; exit 1; }

cd "$R"
bash scripts/build-all.sh

if [[ ! -f ft-backend/conf/config.yaml ]]; then
  echo "WARN: ft-backend/conf/config.yaml missing — copy deploy/config.production.example.yaml and edit DB/Redis."
  cp -n deploy/config.production.example.yaml ft-backend/conf/config.yaml || true
fi

# 尽力创建 PostgreSQL 库（需本机已有 postgres 用户与可连上的实例；失败不阻断，由后端日志说明）
if command -v psql >/dev/null && id postgres &>/dev/null; then
  if sudo -u postgres psql -tc "SELECT 1 FROM pg_database WHERE datname = 'opsfleetpilot'" 2>/dev/null | grep -q 1; then
    echo "PostgreSQL database opsfleetpilot already exists."
  else
    echo "Trying: CREATE DATABASE opsfleetpilot (may need manual DBA setup)..."
    sudo -u postgres psql -c "CREATE DATABASE opsfleetpilot;" 2>/dev/null || \
      echo "WARN: could not auto-create DB — create opsfleetpilot manually and match ft-backend/conf/config.yaml"
  fi
else
  echo "WARN: psql or postgres OS user missing — ensure PostgreSQL is installed and DB opsfleetpilot exists."
fi

install -d -m 755 "$WEB_ROOT"
rsync -a --delete "${R}/dist/web/" "${WEB_ROOT}/"
chown -R www-data:www-data "$WEB_ROOT"

sed -e "s|@OPSFLEET_WEB_ROOT@|${WEB_ROOT}|g" \
    -e "s|@OPSFLEET_UI_PORT@|${UI}|g" \
    -e "s|@OPSFLEET_BACKEND_PORT@|${BP}|g" \
  deploy/nginx.opsfleet.conf.template > /etc/nginx/sites-available/opsfleet
ln -sf /etc/nginx/sites-available/opsfleet /etc/nginx/sites-enabled/opsfleet

# 避免 default 站点占用 80 时无影响；确保 Nginx 已运行（reload 在 inactive 时会失败）
systemctl enable nginx 2>/dev/null || true
systemctl start nginx 2>/dev/null || true
nginx -t
systemctl reload nginx 2>/dev/null || systemctl restart nginx

# 防火墙：放行 UI 端口，否则仅本机 curl 通、外网浏览器不通
if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -q 'Status: active'; then
  ufw allow "${UI}/tcp" comment 'opsfleet nginx' 2>/dev/null || ufw allow "${UI}/tcp" || true
  ufw reload 2>/dev/null || true
  echo "ufw: allowed TCP ${UI}"
elif command -v firewall-cmd >/dev/null 2>&1 && firewall-cmd --state 2>/dev/null | grep -q running; then
  firewall-cmd --permanent --add-port="${UI}/tcp" 2>/dev/null || true
  firewall-cmd --reload 2>/dev/null || true
  echo "firewalld: opened TCP ${UI}"
else
  echo "No active ufw/firewalld detected — if browser cannot connect, open TCP ${UI} in cloud security group / iptables."
fi

# SELinux：允许 Nginx 反代到本机后端
if command -v getenforce >/dev/null 2>&1 && [ "$(getenforce 2>/dev/null)" = "Enforcing" ]; then
  setsebool -P httpd_can_network_connect 1 2>/dev/null && echo "SELinux: httpd_can_network_connect=1" || true
fi

sed "s|/OPSFLEET_ROOT|${R}|g" deploy/opsfleet-backend.service.example > /etc/systemd/system/opsfleet-backend.service

# 首次部署时写入 K8s 制品镜像地址（与 ansible download_domain 对齐；可事后编辑 /etc/opsfleet/backend.env）
install -d -m 755 /etc/opsfleet
if [[ ! -f /etc/opsfleet/backend.env ]]; then
  cat > /etc/opsfleet/backend.env <<'ENV'
# OpsFleet 后端环境变量（systemd EnvironmentFile）
# K8s 制品页 /api/k8s/mirror/catalog 会代理拉取该 URL 的 manifest.json
# 同机仅跑 Nginx 制品站时可改为 http://127.0.0.1
OPSFLEET_K8S_MIRROR_BASE_URL=http://192.168.56.11
ENV
  chmod 600 /etc/opsfleet/backend.env
  echo "Created /etc/opsfleet/backend.env (edit OPSFLEET_K8S_MIRROR_* if needed)"
fi

# 每次全栈部署：指向刚构建的 ai-sre（供 GET /api/k8s/deploy/cli/ai-sre；优先于 config.yaml）
# 同时写入版本号，供 GET .../cli/ai-sre/version 免 exec（可选，失败则后端仍会 probe）
ENV_FILE=/etc/opsfleet/backend.env
tmp_be=$(mktemp)
grep -v '^OPSFLEET_AISRE_BINARY_PATH=' "$ENV_FILE" | grep -v '^OPSFLEET_AISRE_VERSION=' > "$tmp_be" && cat "$tmp_be" > "$ENV_FILE"
rm -f "$tmp_be"
echo "OPSFLEET_AISRE_BINARY_PATH=${R}/bin/ai-sre" >> "$ENV_FILE"
if [[ -x "${R}/bin/ai-sre" ]]; then
  V="$("${R}/bin/ai-sre" version 2>/dev/null | head -1 | awk '{print $2}')"
  if [[ -n "${V:-}" ]]; then
    echo "OPSFLEET_AISRE_VERSION=${V}" >> "$ENV_FILE"
    echo "opsfleet: OPSFLEET_AISRE_VERSION=${V}"
  fi
fi
chmod 600 "$ENV_FILE"
echo "opsfleet: OPSFLEET_AISRE_BINARY_PATH=${R}/bin/ai-sre"

systemctl daemon-reload
systemctl enable opsfleet-backend
systemctl restart opsfleet-backend

sleep 3
if ! systemctl is-active --quiet opsfleet-backend; then
  echo "=== ERROR: opsfleet-backend is not active ==="
  journalctl -u opsfleet-backend -n 120 --no-pager
  echo "Fix DB credentials in ft-backend/conf/config.yaml (and ensure PostgreSQL accepts connections), then: systemctl restart opsfleet-backend"
  exit 1
fi

if ! curl -sfS "http://127.0.0.1:${UI}/health" >/dev/null; then
  echo "=== ERROR: health check via Nginx failed (port ${UI}) ==="
  echo "--- curl Nginx ---"
  curl -vS "http://127.0.0.1:${UI}/health" 2>&1 | tail -n 40 || true
  echo "--- curl backend direct ---"
  curl -sfS "http://127.0.0.1:${BP}/health" && echo " (backend OK, check nginx config)" || echo "backend also failed"
  journalctl -u nginx -n 40 --no-pager || true
  exit 1
fi

if ! curl -sfS "http://127.0.0.1:${UI}/" | head -c 256 | grep -qiE 'doctype|html'; then
  echo "=== ERROR: GET / did not return HTML (check Nginx root & permissions on ${WEB_ROOT}) ==="
  exit 1
fi

echo ""
echo "Remote health OK: http://127.0.0.1:${UI}/health (nginx -> backend :${BP})"
echo "Listening:"
ss -tlnp 2>/dev/null | grep -E ":${UI}|:${BP}" || true
REMOTE_SCRIPT

echo ""
echo "==> Done."
echo "    浏览器访问: http://<服务器公网或内网IP>:${UI_PORT}/"
echo "    若仍无法打开：云主机请在安全组放行 TCP ${UI_PORT}；服务器上可执行: bash $REMOTE_DIR/scripts/verify-opsfleet-deployment.sh"
