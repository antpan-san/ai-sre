#!/usr/bin/env bash
# Deploy skill pack YAML to the lab OpsFort host (192.168.56.11) for testing.
# Skill packs must NOT be pushed to GitHub; this script rsyncs from your local workspace.
#
# Usage (from repo root):
#   ./scripts/deploy-skill-packs-lab.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LAB_SSH="${LAB_SSH:-root@192.168.56.11}"
LAB_REMOTE_DIR="${LAB_REMOTE_DIR:-/root/sre}"
SKILL_DATA_DIR="${SKILL_DATA_DIR:-/var/lib/opsfleet/ai-skills}"

SRC_BACKEND="${ROOT}/ft-backend/skills/builtin"
SRC_CLI="${ROOT}/internal/assets/skills"

if ! compgen -G "${SRC_BACKEND}/*.yaml" > /dev/null 2>&1; then
  echo "ERROR: no YAML under ${SRC_BACKEND}; maintain skill packs locally (not in git)." >&2
  exit 1
fi

echo "==> Skill packs -> lab ${LAB_SSH} (${SKILL_DATA_DIR}/builtin)"

ssh "${LAB_SSH}" "mkdir -p ${SKILL_DATA_DIR}/builtin ${SKILL_DATA_DIR}/generated ${SKILL_DATA_DIR}/samples ${SKILL_DATA_DIR}/feedback ${LAB_REMOTE_DIR}/ft-backend/skills/builtin"

rsync -avz --no-owner --no-group \
  "${SRC_BACKEND}/" "${LAB_SSH}:${SKILL_DATA_DIR}/builtin/"

rsync -avz --no-owner --no-group \
  "${SRC_BACKEND}/" "${LAB_SSH}:${LAB_REMOTE_DIR}/ft-backend/skills/builtin/"

if compgen -G "${SRC_CLI}/*.yaml" > /dev/null 2>&1; then
  ssh "${LAB_SSH}" "mkdir -p ${LAB_REMOTE_DIR}/internal/assets/skills"
  rsync -avz --no-owner --no-group \
    "${SRC_CLI}/" "${LAB_SSH}:${LAB_REMOTE_DIR}/internal/assets/skills/"
fi

echo "==> Restart opsfleet-backend (reload registry from disk builtin)"
ssh "${LAB_SSH}" "systemctl restart opsfleet-backend && sleep 3 && systemctl is-active --quiet opsfleet-backend"
ssh "${LAB_SSH}" "curl -fsS http://127.0.0.1:9080/ft-api/api/ai/skills | head -c 2048; echo"

echo "==> Lab skill deploy OK"
