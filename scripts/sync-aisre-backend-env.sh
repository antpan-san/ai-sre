#!/usr/bin/env bash
# Sync /etc/opsfleet/backend.env ai-sre paths + OPSFLEET_AISRE_VERSION from built binaries.
# Fails deploy when amd64/arm64 versions diverge (prevents client auto-upgrade loops).
set -euo pipefail

REPO_ROOT="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
ENV_FILE="${OPSFLEET_ENV_FILE:-/etc/opsfleet/backend.env}"
REQUIRE_ARM64="${OPSFLEET_REQUIRE_ARM64_AISRE:-1}"

aisre_ver() {
  local b="$1"
  [[ -f "$b" ]] || return 0
  local v=""
  if [[ -x "$b" ]]; then
    v="$("$b" version 2>/dev/null | awk '{print $NF}' || true)"
  fi
  if [[ -n "$v" ]]; then
    echo "$v"
    return 0
  fi
  # amd64 主机无法 exec arm64 ELF；从二进制字符串读取 ai-sre 发行线版本（0.5.x）
  strings "$b" 2>/dev/null | grep -oE '0\.5\.[0-9]+' | sort -Vu | tail -1
}

BIN_AMD64="${REPO_ROOT}/bin/ai-sre"
BIN_ARM64="${REPO_ROOT}/bin/ai-sre.arm64"
V_AMD64="$(aisre_ver "$BIN_AMD64")"
V_ARM64="$(aisre_ver "$BIN_ARM64")"

if [[ "$REQUIRE_ARM64" == "1" && "$(uname -m 2>/dev/null)" =~ ^(x86_64|amd64)$ ]]; then
  if [[ ! -f "$BIN_ARM64" ]]; then
    echo "ERROR: missing $BIN_ARM64 — ARM 控制机 (k8s-master-0 等) 需要 arm64 包。" >&2
    echo "       请在仓库根执行: bash scripts/build-all.sh（须 linux/arm64 交叉编译成功）" >&2
    exit 1
  fi
  if [[ -z "$V_ARM64" ]]; then
    echo "ERROR: cannot read version from $BIN_ARM64" >&2
    exit 1
  fi
fi

if [[ -n "$V_AMD64" && -n "$V_ARM64" && "$V_AMD64" != "$V_ARM64" ]]; then
  echo "ERROR: ai-sre version mismatch: amd64=$V_AMD64 ($BIN_AMD64) arm64=$V_ARM64 ($BIN_ARM64)" >&2
  echo "       重新执行 bash scripts/build-all.sh 并确保两架构版本一致后再部署。" >&2
  exit 1
fi

V_PUBLISH="${V_ARM64:-$V_AMD64}"
if [[ -z "$V_PUBLISH" ]]; then
  echo "ERROR: no ai-sre binary version under ${REPO_ROOT}/bin" >&2
  exit 1
fi

install -d -m 755 "$(dirname "$ENV_FILE")"
touch "$ENV_FILE"
chmod 600 "$ENV_FILE"
tmp=$(mktemp)
grep -v '^OPSFLEET_AISRE_BINARY_PATH=' "$ENV_FILE" \
  | grep -v '^OPSFLEET_AISRE_BINARY_PATH_ARM64=' \
  | grep -v '^OPSFLEET_AISRE_BINARY_PATH_AMD64=' \
  | grep -v '^OPSFLEET_AISRE_VERSION=' >"$tmp" || true
cat "$tmp" >"$ENV_FILE"
rm -f "$tmp"

echo "OPSFLEET_AISRE_BINARY_PATH=${BIN_AMD64}" >>"$ENV_FILE"
if [[ -f "$BIN_ARM64" ]]; then
  echo "OPSFLEET_AISRE_BINARY_PATH_ARM64=${BIN_ARM64}" >>"$ENV_FILE"
fi
echo "OPSFLEET_AISRE_VERSION=${V_PUBLISH}" >>"$ENV_FILE"

echo "sync-aisre-backend-env: OPSFLEET_AISRE_VERSION=${V_PUBLISH} (amd64=${V_AMD64:-—} arm64=${V_ARM64:-—})"
