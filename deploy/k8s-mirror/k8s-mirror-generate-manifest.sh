#!/usr/bin/env bash
# 扫描 MIRROR_ROOT 下制品，计算 SHA512，写入 $MIRROR_ROOT/manifest.json（供 OpsFleet 页面展示）
set -euo pipefail

ENV_FILE="${ENV_FILE:-/etc/opsfleet/k8s-mirror.env}"
if [[ -f "$ENV_FILE" ]]; then
  # shellcheck source=/dev/null
  source "$ENV_FILE"
fi

MIRROR_ROOT="${MIRROR_ROOT:-/var/lib/opsfleet-k8s-mirror}"
PUBLIC_BASE_URL="${PUBLIC_BASE_URL:-http://192.168.56.11}"

if ! command -v python3 >/dev/null 2>&1; then
  echo "需要 python3 生成 JSON"
  exit 1
fi

MANIFEST_TMP="${MIRROR_ROOT}/.manifest.tmp.json"

export MIRROR_ROOT PUBLIC_BASE_URL
python3 <<'PY'
import json, os, hashlib, time
from pathlib import Path

root = Path(os.environ["MIRROR_ROOT"]).resolve()
base = os.environ.get("PUBLIC_BASE_URL", "http://127.0.0.1").rstrip("/")
files = []
for p in sorted(root.rglob("*")):
    if not p.is_file():
        continue
    n = p.name
    if n in ("manifest.json",) or n.startswith(".manifest"):
        continue
    if n.endswith(".sha512"):
        continue
    suf = n.lower()
    if not (suf.endswith(".tar.gz") or suf.endswith(".tgz")):
        continue
    rel = p.relative_to(root).as_posix()
    h = hashlib.sha512()
    with p.open("rb") as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b""):
            h.update(chunk)
    files.append({
        "relativePath": rel,
        "sizeBytes": p.stat().st_size,
        "sha512": h.hexdigest(),
        "downloadUrl": f"{base}/{rel}",
    })

out = {
    "generatedAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
    "mirrorRoot": str(root),
    "publicBaseUrl": base,
    "files": files,
}
path = root / "manifest.json"
tmp = root / ".manifest.tmp.json"
tmp.write_text(json.dumps(out, ensure_ascii=False, indent=2), encoding="utf-8")
tmp.replace(path)
print(f"Wrote {path} ({len(files)} files)")
PY

echo "=== manifest.json 已更新 ==="
