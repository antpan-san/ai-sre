#!/usr/bin/env bash
# Deploy ai-sre sources to remote SRE host and run build smoke tests.
set -euo pipefail

REMOTE_USER_HOST="${DEPLOY_REMOTE:-root@192.168.56.11}"
REMOTE_DIR="${DEPLOY_REMOTE_DIR:-/root/sre}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "==> Ensuring remote directory exists: $REMOTE_USER_HOST:$REMOTE_DIR"
ssh -o BatchMode=yes -o ConnectTimeout=15 "$REMOTE_USER_HOST" "mkdir -p '$REMOTE_DIR'"

echo "==> Rsync project -> $REMOTE_USER_HOST:$REMOTE_DIR"
rsync -avz \
  --exclude '.git' \
  --exclude 'ai-sre' \
  --exclude '.DS_Store' \
  --exclude '.env' \
  --exclude 'bin' \
  --exclude 'dist' \
  --exclude 'ft-front/node_modules' \
  --exclude 'ft-front/dist' \
  "$PROJECT_ROOT/" \
  "$REMOTE_USER_HOST:$REMOTE_DIR/"

echo "==> Remote build + smoke test"
ssh -o BatchMode=yes -o ConnectTimeout=30 "$REMOTE_USER_HOST" \
  "bash -lc 'set -euo pipefail; cd \"$REMOTE_DIR\"; export GOTOOLCHAIN=auto; command -v go >/dev/null || { echo \"Remote: go not found. Install e.g. apt-get install -y golang-go\" >&2; exit 1; }; go mod download; go vet ./...; go build -o ai-sre .; ./ai-sre version; echo Remote deploy OK.'"

echo "==> Done."
