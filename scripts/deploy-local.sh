#!/usr/bin/env bash
# Build and smoke-test ai-sre on the local machine (agent workspace host).
# Used by release-deploy before remote deploy / git push.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"

echo "==> local: go mod download"
go mod download

echo "==> local: go vet ./..."
go vet ./...

echo "==> local: go build -o ai-sre ."
go build -buildvcs=false -o ai-sre .

echo "==> local: ./ai-sre version"
./ai-sre version

EXPECTED="$(grep -E '^\s*var Version\s*=' internal/cli/version.go | sed -E 's/.*"([^"]+)".*/\1/')"
ACTUAL="$(./ai-sre version 2>/dev/null | awk '{print $NF}' | tr -d '\r')"
if [[ -n "$EXPECTED" && -n "$ACTUAL" && "$EXPECTED" != "$ACTUAL" ]]; then
  echo "ERROR: binary version ($ACTUAL) != internal/cli/version.go ($EXPECTED)" >&2
  exit 1
fi

if [[ "${DEPLOY_LOCAL_OPSFLEET:-0}" == "1" ]]; then
  echo "==> local: vet-opsfleet + build-all (DEPLOY_LOCAL_OPSFLEET=1)"
  make vet-opsfleet
  bash scripts/build-all.sh
  test -x bin/ai-sre && bin/ai-sre version
fi

echo "==> local deploy OK"
