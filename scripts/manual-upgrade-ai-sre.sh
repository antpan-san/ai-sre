#!/usr/bin/env bash
# Manual ai-sre upgrade when old CLI hits short download timeout (e.g. 0.5.26).
# Usage: OPSFLEET_API_URL=http://opsfleetpilot.com/ft-api bash scripts/manual-upgrade-ai-sre.sh
set -euo pipefail

API_BASE="${OPSFLEET_API_URL:-http://opsfleetpilot.com/ft-api}"
API_BASE="${API_BASE%/}"
ARCH="${OPSFLEET_AISRE_ARCH:-}"
if [[ -z "$ARCH" ]]; then
  case "$(uname -m 2>/dev/null)" in
    aarch64|arm64) ARCH=arm64 ;;
    *) ARCH=amd64 ;;
  esac
fi
DEST="${OPSFLEET_AISRE_INSTALL_PATH:-$(command -v ai-sre 2>/dev/null || echo /usr/local/bin/ai-sre)}"
TMP="${DEST}.downloading.$$"
MAX_TIME="${OPSFLEET_UPGRADE_DOWNLOAD_TIMEOUT:-600}"

echo "==> Download ${API_BASE}/api/k8s/deploy/cli/ai-sre?arch=${ARCH}"
echo "    -> ${TMP} (curl max-time ${MAX_TIME}s)"
curl -fSL --connect-timeout 30 --max-time "$MAX_TIME" \
  -o "$TMP" "${API_BASE}/api/k8s/deploy/cli/ai-sre?arch=${ARCH}"
chmod 0755 "$TMP"
mv -f "$TMP" "$DEST"
echo "==> Installed: $DEST"
"$DEST" version
