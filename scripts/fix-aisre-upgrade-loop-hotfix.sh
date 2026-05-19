#!/usr/bin/env bash
# One-shot heal: rebuild ai-sre* if needed, sync backend.env, restart backend.
# Run ON the OpsFleet host (lab or production). Prefer: deploy-opsfleet-remote.sh (includes this logic).
set -euo pipefail

ROOT="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
cd "$ROOT"

echo "==> build-all (ensures bin/ai-sre + bin/ai-sre.arm64 same version)"
bash scripts/build-all.sh

echo "==> sync backend.env"
bash scripts/sync-aisre-backend-env.sh "$ROOT"

echo "==> restart opsfleet-backend"
systemctl restart opsfleet-backend
sleep 2

API="${OPSFLEET_API_HEALTH:-http://127.0.0.1/ft-api/api/k8s/deploy/cli/ai-sre/version}"
echo "==> API: $API"
curl -fsS "$API" || curl -fsS "http://127.0.0.1:9080/ft-api/api/k8s/deploy/cli/ai-sre/version"
echo
echo "Done. ARM 控制机可再执行: ai-sre upgrade -y"
