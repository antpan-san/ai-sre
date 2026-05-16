#!/usr/bin/env bash
# Deploy ai-sre / OpsFleet skill packs to production (opsfleetpilot.com).
# Builtin YAML is embedded in opsfleet-backend via build-all.sh; this script
# rsyncs source, rebuilds on the server, restarts backend, and verifies /api/ai/skills.
#
# Do NOT use deploy-opsfleet-remote.sh (lab). Do NOT use this for 192.168.56.11.
#
# Usage (from repo root):
#   ./scripts/deploy-skill-packs-production.sh
#
# Env:
#   PROD_SSH=root@204.44.123.101
#   PROD_SSH_PORT=10080
#   PROD_REMOTE_DIR=/root/sre

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROD_SSH="${PROD_SSH:-root@204.44.123.101}"
PROD_SSH_PORT="${PROD_SSH_PORT:-10080}"
PROD_REMOTE_DIR="${PROD_REMOTE_DIR:-/root/sre}"
RSYNC_SSH="ssh -p ${PROD_SSH_PORT}"

echo "==> Skill packs -> production (${PROD_SSH}:${PROD_REMOTE_DIR})"

ssh -p "${PROD_SSH_PORT}" "${PROD_SSH}" "test -d ${PROD_REMOTE_DIR} || mkdir -p ${PROD_REMOTE_DIR}"

echo "==> Rsync skill-related source (builtin YAML + backend loader)"
rsync -avz --no-owner --no-group -e "${RSYNC_SSH}" \
  --exclude '.git' \
  --exclude 'bin' \
  --exclude 'dist' \
  "${ROOT}/ft-backend/skills/" "${PROD_SSH}:${PROD_REMOTE_DIR}/ft-backend/skills/"

if [ -d "${ROOT}/internal/assets/skills" ]; then
  rsync -avz --no-owner --no-group -e "${RSYNC_SSH}" \
    "${ROOT}/internal/assets/skills/" "${PROD_SSH}:${PROD_REMOTE_DIR}/internal/assets/skills/"
fi

echo "==> Remote build (embeds builtin into opsfleet-backend) + restart"
ssh -p "${PROD_SSH_PORT}" "${PROD_SSH}" "set -euo pipefail
cd ${PROD_REMOTE_DIR}
bash scripts/build-all.sh
install -d -m 0755 /var/lib/opsfleet/ai-skills/samples /var/lib/opsfleet/ai-skills/feedback /var/lib/opsfleet/ai-skills/generated
systemctl restart opsfleet-backend
sleep 3
systemctl is-active --quiet opsfleet-backend
curl -fsS http://127.0.0.1/health
printf \"\\n\"
curl -fsS http://127.0.0.1/ft-api/api/ai/skills | head -c 2048
printf \"\\n\"
"

echo "==> Done. Builtin skills are in the rebuilt opsfleet-backend; generated packs remain under /var/lib/opsfleet/ai-skills on the server."
