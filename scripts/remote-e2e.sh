#!/usr/bin/env bash
# Run on the deployment host after deploy-remote.sh (requires valid ~/.config/ai-sre/api_key).
set -euo pipefail
cd "$(dirname "$0")/.."
export GOFLAGS="${GOFLAGS:--buildvcs=false}"
echo "==> local vet/build/test"
_pkgs=($(go list ./...))
go vet "${_pkgs[@]}"
go test "${_pkgs[@]}"
go build -o ai-sre .
echo "==> version / doctor / skills"
./ai-sre version
./ai-sre doctor
./ai-sre expert skills list | awk 'NR<=8 {print}'
if [[ "${OPSFLEET_SKIP_REMOTE:-}" != 1 ]]; then
  echo "==> server skills registry"
  if ! ./ai-sre expert skills server | awk 'NR<=12 {print}'; then
    echo "WARN: ai-sre expert skills server failed (服务端 /api/ai/skills 不可达？可在 OPSFLEET_API_URL 处确认)" >&2
  fi
fi
echo "==> negative: no creds"
t="$(mktemp -d)"
set +o pipefail
HOME="$t" OPSFLEET_SKIP_REMOTE=1 ./ai-sre expert ask x 2>&1 | grep -q "credentials not found" || { rm -rf "$t"; echo FAIL; exit 1; }
set -o pipefail
rm -rf "$t"
echo OK
if [[ "${SHORT:-}" != 1 && "${OPSFLEET_SKIP_REMOTE:-}" != 1 && "${SKIP_SKILL_SAMPLES_VERIFY:-}" != 1 ]]; then
  echo "==> skill samples lab verify"
  SKIP_CLI_CHECK="${SKIP_SKILL_SAMPLES_CLI:-1}" ./scripts/verify-skill-samples-lab.sh
fi
echo "==> LLM smoke (set SHORT=1 to skip)"
if [[ "${SHORT:-}" == 1 ]]; then
  echo "skipped"
  exit 0
fi
timeout 180 ./ai-sre --no-rag expert ask "用一句话说明什么是 consumer lag"
timeout 180 ./ai-sre check kafka --lag 1
timeout 180 ./ai-sre check k8s --pod pending
timeout 180 ./ai-sre expert runbook "Pod Pending 应急"
echo "==> remote-e2e OK"
