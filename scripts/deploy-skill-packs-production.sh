#!/usr/bin/env bash
# Deploy skill pack YAML to production (opsfleetpilot.com). Canonical store:
#   /var/lib/opsfleet/ai-skills/builtin/
# Do NOT commit skill YAML to GitHub — rsync from local workspace only.
#
# Usage (from repo root):
#   ./scripts/deploy-skill-packs-production.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROD_SSH="${PROD_SSH:-root@204.44.123.101}"
PROD_SSH_PORT="${PROD_SSH_PORT:-10080}"
PROD_REMOTE_DIR="${PROD_REMOTE_DIR:-/root/sre}"
SKILL_DATA_DIR="${SKILL_DATA_DIR:-/var/lib/opsfleet/ai-skills}"
RSYNC_SSH="ssh -p ${PROD_SSH_PORT}"

SRC_BACKEND="${ROOT}/ft-backend/skills/builtin"
SRC_CLI="${ROOT}/internal/assets/skills"

if ! compgen -G "${SRC_BACKEND}/*.yaml" > /dev/null 2>&1; then
  echo "ERROR: no YAML under ${SRC_BACKEND}; maintain skill packs locally (not in git)." >&2
  exit 1
fi

echo "==> Skill packs -> production (${PROD_SSH}:${SKILL_DATA_DIR}/builtin)"

ssh -p "${PROD_SSH_PORT}" "${PROD_SSH}" "mkdir -p ${SKILL_DATA_DIR}/builtin ${SKILL_DATA_DIR}/generated ${SKILL_DATA_DIR}/samples ${SKILL_DATA_DIR}/feedback ${PROD_REMOTE_DIR}/ft-backend/skills/builtin"

rsync -avz --no-owner --no-group -e "${RSYNC_SSH}" \
  "${SRC_BACKEND}/" "${PROD_SSH}:${SKILL_DATA_DIR}/builtin/"

rsync -avz --no-owner --no-group -e "${RSYNC_SSH}" \
  "${SRC_BACKEND}/" "${PROD_SSH}:${PROD_REMOTE_DIR}/ft-backend/skills/builtin/"

if compgen -G "${SRC_CLI}/*.yaml" > /dev/null 2>&1; then
  ssh -p "${PROD_SSH_PORT}" "${PROD_SSH}" "mkdir -p ${PROD_REMOTE_DIR}/internal/assets/skills"
  rsync -avz --no-owner --no-group -e "${RSYNC_SSH}" \
    "${SRC_CLI}/" "${PROD_SSH}:${PROD_REMOTE_DIR}/internal/assets/skills/"
fi

echo "==> Restart opsfleet-backend"
ssh -p "${PROD_SSH_PORT}" "${PROD_SSH}" "set -euo pipefail
grep -q '^OPSFLEET_AI_SKILL_DATA_DIR=' /etc/opsfleet/backend.env 2>/dev/null || echo 'OPSFLEET_AI_SKILL_DATA_DIR=${SKILL_DATA_DIR}' >> /etc/opsfleet/backend.env
systemctl restart opsfleet-backend
sleep 3
systemctl is-active --quiet opsfleet-backend
curl -fsS http://127.0.0.1/health
printf \"\\n\"
curl -fsS http://127.0.0.1/ft-api/api/ai/skills | head -c 2048
printf \"\\n\"
"

echo "==> Production skill deploy OK (builtin on disk; generated unchanged unless you rsync it separately)"
