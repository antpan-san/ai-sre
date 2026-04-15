#!/usr/bin/env bash
# Run on the deployment host after deploy-remote.sh (requires valid ~/.config/ai-sre/api_key).
set -euo pipefail
cd "$(dirname "$0")/.."
echo "==> local vet/build"
go vet ./...
go build -o ai-sre .
echo "==> version"
./ai-sre version
echo "==> negative: no creds"
t="$(mktemp -d)"
set +o pipefail
HOME="$t" ./ai-sre ask x 2>&1 | grep -q "credentials not found" || { rm -rf "$t"; echo FAIL; exit 1; }
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
