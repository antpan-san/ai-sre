#!/usr/bin/env bash
# Run on the deployment host after deploy-remote.sh (requires valid ~/.config/ai-sre/api_key).
set -euo pipefail
cd "$(dirname "$0")/.."
echo "==> local vet/build/test"
go vet ./...
go test ./...
go build -o ai-sre .
echo "==> version / doctor / skills"
./ai-sre version
./ai-sre doctor
head -8 < <(./ai-sre skills list)
if [[ "${OPSFLEET_SKIP_REMOTE:-}" != 1 ]]; then
  echo "==> server skills registry"
  if ! head -12 < <(./ai-sre skills server); then
    echo "WARN: ai-sre skills server failed (服务端 /api/ai/skills 不可达？可在 OPSFLEET_API_URL 处确认)" >&2
  fi
fi
echo "==> negative: no creds"
t="$(mktemp -d)"
set +o pipefail
HOME="$t" OPSFLEET_SKIP_REMOTE=1 ./ai-sre ask x 2>&1 | grep -q "credentials not found" || { rm -rf "$t"; echo FAIL; exit 1; }
set -o pipefail
rm -rf "$t"
echo OK
echo "==> LLM smoke (set SHORT=1 to skip)"
if [[ "${SHORT:-}" == 1 ]]; then
  echo "skipped"
  exit 0
fi
timeout 180 ./ai-sre --no-rag ask "用一句话说明什么是 consumer lag"
timeout 180 ./ai-sre analyze kafka --lag 1
timeout 180 ./ai-sre analyze k8s --pod pending
timeout 180 ./ai-sre runbook "Pod Pending 应急"
echo "==> remote-e2e OK"
