#!/usr/bin/env bash
# OpsFleet：可视化下载帮助脚本，供 ansible `command:` / `shell:` 或 install.sh 调用。
# 行为对齐 ai-sre Go 侧 progressReader：TTY 下 curl --progress-bar；非 TTY 退化为 -sS 但仍写一行
# 「[time] 已下载 N / 总 M (avg X/s)」摘要到 stderr，便于在 ansible -v 或日志中追踪。
#
# 用法：
#   download-with-progress.sh <url> <dest> [<sha512_hex>] [<min_bytes>]
#
# 退出码：
#   0   下载成功且（如提供 checksum）校验通过；或目标文件已存在且 checksum 一致
#   2   参数错误
#   3   下载失败（curl 非 0）        -> emit [ERROR-CODE] OPSFLEET_DL_E_NETWORK
#   4   校验失败（sha512 / min_bytes）-> emit [ERROR-CODE] OPSFLEET_DL_E_CHECKSUM
#
# 失败时除中文摘要外，会向 stderr 写一行机器可读的错误码：
#   [ERROR-CODE] <code> url=<url> dest=<dest> detail=<short>
# 供 install.sh / ai-sre analyze code 抓取，匹配 docs/error-codes.yaml。
set -euo pipefail

URL="${1:-}"
DEST="${2:-}"
WANT_SHA="${3:-}"
MIN_BYTES="${4:-0}"

emit_code() {
  local code="$1"; shift
  local detail="${*:-}"
  echo "[ERROR-CODE] $code url=$URL dest=$DEST detail=$detail" >&2
}

if [[ -z "$URL" || -z "$DEST" ]]; then
  echo "用法: download-with-progress.sh <url> <dest> [<sha512_hex>] [<min_bytes>]" >&2
  exit 2
fi

mkdir -p "$(dirname "$DEST")"

sha512_of() {
  local f="$1"
  if command -v sha512sum >/dev/null 2>&1; then
    sha512sum "$f" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 512 "$f" | awk '{print $1}'
  else
    echo "(sha512sum/shasum unavailable)" >&2
    return 1
  fi
}

if [[ -f "$DEST" && -n "$WANT_SHA" ]]; then
  if got="$(sha512_of "$DEST")" && [[ "$got" == "$WANT_SHA" ]]; then
    echo "[$(date '+%H:%M:%S')] skip $DEST (sha512 matches)" >&2
    exit 0
  fi
fi

ts_start=$(date +%s)
echo "[$(date '+%H:%M:%S')] 开始下载 $URL -> $DEST" >&2

CURL_OPTS=("--fail" "--location" "--connect-timeout" "30" "--retry" "3" "--retry-delay" "2")
if [ -t 2 ] && [ "${OPSFLEET_NO_PROGRESS:-}" != "1" ]; then
  if ! curl "${CURL_OPTS[@]}" --progress-bar -o "$DEST" "$URL"; then
    echo "[$(date '+%H:%M:%S')] 下载失败: $URL" >&2
    emit_code "OPSFLEET_DL_E_NETWORK" "curl exited non-zero"
    exit 3
  fi
else
  if ! curl "${CURL_OPTS[@]}" -sS -o "$DEST" "$URL"; then
    echo "[$(date '+%H:%M:%S')] 下载失败: $URL" >&2
    emit_code "OPSFLEET_DL_E_NETWORK" "curl exited non-zero"
    exit 3
  fi
fi
ts_end=$(date +%s)
elapsed=$((ts_end - ts_start)); [[ $elapsed -lt 1 ]] && elapsed=1
size=$(stat -c %s "$DEST" 2>/dev/null || stat -f %z "$DEST" 2>/dev/null || echo "0")
avg=$(( size / elapsed ))
hr() { local n=$1; awk -v n="$n" 'BEGIN{ u[0]="B"; u[1]="KiB"; u[2]="MiB"; u[3]="GiB"; u[4]="TiB"; i=0; while (n>=1024 && i<4) { n/=1024; i++ } if (i==0) printf "%d%s\n", n, u[i]; else printf "%.1f%s\n", n, u[i] }'; }
echo "[$(date '+%H:%M:%S')] 下载完成 $DEST ($(hr "$size"), ${elapsed}s, avg $(hr "$avg")/s)" >&2

if [[ "$MIN_BYTES" -gt 0 && "$size" -lt "$MIN_BYTES" ]]; then
  echo "[$(date '+%H:%M:%S')] 校验失败：文件大小 $size < min_bytes $MIN_BYTES（疑似服务器返回错误页）" >&2
  emit_code "OPSFLEET_DL_E_CHECKSUM" "size=$size < min_bytes=$MIN_BYTES"
  exit 4
fi
if [[ -n "$WANT_SHA" ]]; then
  if got="$(sha512_of "$DEST")"; then
    if [[ "$got" != "$WANT_SHA" ]]; then
      echo "[$(date '+%H:%M:%S')] 校验失败：sha512 mismatch" >&2
      echo "  expected: $WANT_SHA" >&2
      echo "  got     : $got" >&2
      emit_code "OPSFLEET_DL_E_CHECKSUM" "sha512 mismatch"
      exit 4
    fi
    echo "[$(date '+%H:%M:%S')] sha512 校验通过" >&2
  fi
fi
exit 0
