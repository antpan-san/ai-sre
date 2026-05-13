#!/usr/bin/env bash
# 删除 MIRROR_ROOT 下过期的制品大文件，释放国外/小盘中转节点磁盘。
# 保留 manifest.json；与 opsfleet-k8s-mirror-serve 按需落盘配合使用。
set -euo pipefail

ENV_FILE="${ENV_FILE:-/etc/opsfleet/k8s-mirror.env}"
if [[ -f "$ENV_FILE" ]]; then
  # shellcheck source=/dev/null
  source "$ENV_FILE"
fi

MIRROR_ROOT="${MIRROR_ROOT:-/var/lib/opsfleet-k8s-mirror}"
TTL_DAYS="${TTL_DAYS:-14}"
DRY_RUN="${DRY_RUN:-0}"

log() { echo "[k8s-mirror-ttl] $*"; }

if [[ ! -d "$MIRROR_ROOT" ]]; then
  log "MIRROR_ROOT 不存在: $MIRROR_ROOT"
  exit 0
fi

if [[ "$DRY_RUN" == "1" ]]; then
  log "DRY_RUN=1 仅打印将删除项"
  find "$MIRROR_ROOT" -type f \( -name '*.tar.gz' -o -name '*.tgz' -o -name '*.tar' \) \
    ! -path "$MIRROR_ROOT/manifest.json" \
    -mtime "+${TTL_DAYS}" -print || true
  exit 0
fi

n=0
while IFS= read -r -d '' f; do
  log "delete ${f#"$MIRROR_ROOT"/}"
  rm -f "$f"
  n=$((n + 1))
done < <(find "$MIRROR_ROOT" -type f \( -name '*.tar.gz' -o -name '*.tgz' -o -name '*.tar' \) \
  ! -path "$MIRROR_ROOT/manifest.json" \
  -mtime "+${TTL_DAYS}" -print0 2>/dev/null || true)

log "done deleted=$n TTL_DAYS=$TTL_DAYS"
