#!/usr/bin/env bash
# Local OpsFleet code-agent worker: pull auto-iteration tasks and run Cursor Agent + release verify.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT_DEFAULT="$(cd "$SCRIPT_DIR/.." && pwd)"
ENV_FILE="${CODE_AGENT_WORKER_ENV:-$SCRIPT_DIR/code-agent-worker.env}"

if [[ -f "$ENV_FILE" ]]; then
  # shellcheck source=/dev/null
  source "$ENV_FILE"
fi

OPSFLEET_API_URL="${OPSFLEET_API_URL:-http://192.168.56.11:9080/ft-api}"
REPO_ROOT="${REPO_ROOT:-$REPO_ROOT_DEFAULT}"
AGENT_BIN="${AGENT_BIN:-agent}"
POLL_INTERVAL="${POLL_INTERVAL:-30}"
RUN_POST_VERIFY="${RUN_POST_VERIFY:-1}"
DRY_RUN_AGENT="${DRY_RUN_AGENT:-0}"

STATE_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/ai-sre"
FP_FILE="$STATE_DIR/code-agent-fingerprint"
LOG_FILE="${CODE_AGENT_WORKER_LOG:-$STATE_DIR/code-agent-worker.log}"

log() {
  echo "$(date '+%Y-%m-%d %H:%M:%S') $*" | tee -a "$LOG_FILE"
}

die() {
  log "ERROR: $*"
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing command: $1"
}

ensure_fingerprint() {
  mkdir -p "$STATE_DIR"
  if [[ -f "$FP_FILE" ]]; then
    FINGERPRINT=$(tr -d '[:space:]' <"$FP_FILE")
    [[ ${#FINGERPRINT} -eq 64 ]] && return
  fi
  FINGERPRINT=$(python3 - <<'PY'
import hashlib, os, platform, uuid
parts = [platform.node(), platform.machine(), str(uuid.getnode())]
mid = "/etc/machine-id"
if os.path.isfile(mid):
    parts.append(open(mid).read().strip())
print(hashlib.sha256("|".join(parts).encode()).hexdigest())
PY
)
  echo "$FINGERPRINT" >"$FP_FILE"
  chmod 600 "$FP_FILE"
}

api() {
  local method="$1" path="$2"
  shift 2
  local raw
  if ! raw=$(curl -sS -X "$method" "${OPSFLEET_API_URL}${path}" \
    -H "Authorization: Bearer ${OPSFLEET_CODE_AGENT_TOKEN}" \
    -H "X-OpsFleet-Agent-Fingerprint: ${FINGERPRINT}" \
    -H "Content-Type: application/json" \
    "$@"); then
    log "API request failed: ${method} ${path}"
    return 1
  fi
  if ! jq -e . >/dev/null 2>&1 <<<"$raw"; then
    local sample
    sample=$(printf '%s' "$raw" | head -c 240 | tr '\n' ' ')
    log "API returned non-JSON: ${method} ${path}: ${sample}"
    return 1
  fi
  printf '%s\n' "$raw"
}

heartbeat() {
  local raw
  raw=$(api POST /api/code-agent/heartbeat -d '{}') || return 1
  jq -e '.code == 200 and .data.ok == true' >/dev/null <<<"$raw"
}

report_event() {
  local task_id="$1" message="$2"
  local payload="${3:-{}}"
  api POST "/api/code-agent/tasks/${task_id}/events" \
    -d "$(jq -n --arg m "$message" --argjson p "$payload" '{message:$m,payload:$p}')" >/dev/null
}

report_result() {
  local task_id="$1" success="$2" summary="$3"
  api POST "/api/code-agent/tasks/${task_id}/result" \
    -d "$(jq -n --argjson s "$success" --arg sum "$summary" '{success:$s,summary:$sum}')" >/dev/null
}

pull_task() {
  local raw
  raw=$(api GET /api/code-agent/tasks/pull) || return 1
  jq -c '.data.task // .task // empty' <<<"$raw"
}

build_prompt() {
  local task_json="$1"
  jq -r '
    . as $t |
    "【自动迭代任务 \($t.id)】\n" +
    "标题: \($t.title)\n" +
    (if ($t.topic // "") != "" then "Topic: \($t.topic)\n" else "" end) +
    (if ($t.summary // "") != "" then "上下文摘要: \($t.summary)\n" else "" end) +
    (if ($t.command // "") != "" then "\n\($t.command)\n" else "" end) +
    "\n---\n" +
    "规范 skill（本任务首次 Read 一次）: \($t.dev_skill // ".cursor/skills/auto-iteration-dev/SKILL.md")\n" +
    "发布 skill（仅开发完成且本地验证通过后 Read 并执行）: \($t.release_skill // ".cursor/skills/release-deploy/SKILL.md")\n" +
    "仓库: 仅改需求相关文件；开发期禁止全量 remote-e2e（无 SHORT=1）。\n" +
    "推送 GitHub: 仅使用 bash scripts/github-push-safe.sh（会先拦截技能包 YAML 与密钥路径）。\n" +
    "技能包: ft-backend/skills/builtin/*.yaml 禁止进 git；改 YAML 后走 deploy-skill-packs-production.sh。\n"
  ' <<<"$task_json"
}

run_agent() {
  local prompt="$1"
  local agent_args=(--print --trust --workspace "$REPO_ROOT")
  if [[ -n "${AGENT_MODEL:-}" ]]; then
    agent_args+=(--model "$AGENT_MODEL")
  fi
  log "Starting Cursor Agent in $REPO_ROOT"
  if [[ "$DRY_RUN_AGENT" == "1" ]]; then
    log "DRY_RUN_AGENT=1, skipping agent"
    return 0
  fi
  # shellcheck disable=SC2090
  "$AGENT_BIN" "${agent_args[@]}" "$prompt"
}

verify_failure_summary() {
  local rc="$1"
  local hint
  hint=$(grep -E '^(==>|FAIL|error:|found packages|security-hardening)' "$LOG_FILE" 2>/dev/null | tail -3 | tr '\n' ' ' | sed 's/  */ /g;s/ $//')
  if [[ -n "$hint" ]]; then
    echo "冒烟验证失败(exit=${rc})：${hint} 详见 ${LOG_FILE}"
  else
    echo "冒烟验证失败(exit=${rc})，详见 ${LOG_FILE}"
  fi
}

run_post_verify() {
  [[ "$RUN_POST_VERIFY" == "1" ]] || return 0
  log "Running post-verify: SHORT=1 remote-e2e"
  (cd "$REPO_ROOT" && SHORT=1 bash scripts/remote-e2e.sh)
  log "Running post-verify: check-skill-packs-not-in-git"
  (cd "$REPO_ROOT" && bash scripts/check-skill-packs-not-in-git.sh)
}

process_task() {
  local task_json="$1"
  local task_id title
  task_id=$(jq -r '.id' <<<"$task_json")
  title=$(jq -r '.title' <<<"$task_json")
  log "Processing task $task_id: $title"

  report_event "$task_id" "Worker 开始执行（本机 Cursor Agent）" '{"phase":"agent"}'

  local prompt agent_rc=0 verify_rc=0 summary
  prompt=$(build_prompt "$task_json")

  set +e
  run_agent "$prompt" >>"$LOG_FILE" 2>&1
  agent_rc=$?
  set -e

  if [[ $agent_rc -ne 0 ]]; then
    summary="Cursor Agent 退出码 ${agent_rc}，详见 ${LOG_FILE}"
    report_event "$task_id" "$summary" "{\"agent_exit\":$agent_rc}"
    report_result "$task_id" false "$summary"
    log "Task $task_id failed (agent)"
    return
  fi

  report_event "$task_id" "Agent 完成，开始冒烟验证" '{"phase":"verify"}'

  set +e
  run_post_verify >>"$LOG_FILE" 2>&1
  verify_rc=$?
  set -e

  if [[ $verify_rc -ne 0 ]]; then
    summary=$(verify_failure_summary "$verify_rc")
    report_event "$task_id" "$summary" "{\"verify_exit\":$verify_rc}"
    report_result "$task_id" false "$summary"
    log "Task $task_id failed (verify)"
    return
  fi

  summary="本机 Agent 与 remote-e2e 均通过"
  report_event "$task_id" "$summary" '{"phase":"done"}'
  report_result "$task_id" true "$summary"
  log "Task $task_id completed"
}

main_loop() {
  log "Code-agent worker started (API=$OPSFLEET_API_URL, repo=$REPO_ROOT)"
  while true; do
    if ! heartbeat; then
      log "Heartbeat failed — check OPSFLEET_API_URL and OPSFLEET_CODE_AGENT_TOKEN"
      sleep "$POLL_INTERVAL"
      continue
    fi
    task_json=$(pull_task || true)
    if [[ -n "$task_json" && "$task_json" != "null" ]]; then
      process_task "$task_json" || log "process_task error (continuing)"
    fi
    sleep "$POLL_INTERVAL"
  done
}

# --- main ---
require_cmd curl
require_cmd jq
require_cmd python3
[[ -n "${OPSFLEET_CODE_AGENT_TOKEN:-}" ]] || die "set OPSFLEET_CODE_AGENT_TOKEN in $ENV_FILE"
[[ -d "$REPO_ROOT" ]] || die "REPO_ROOT not found: $REPO_ROOT"
command -v "$AGENT_BIN" >/dev/null 2>&1 || die "AGENT_BIN not found: $AGENT_BIN (run: agent login)"

mkdir -p "$STATE_DIR"
ensure_fingerprint

case "${1:-run}" in
  run) main_loop ;;
  once)
    heartbeat || die "heartbeat failed"
    task_json=$(pull_task || true)
    [[ -n "$task_json" && "$task_json" != "null" ]] || { log "No task"; exit 0; }
    process_task "$task_json"
    ;;
  heartbeat)
    ensure_fingerprint
    heartbeat && log "heartbeat ok"
    ;;
  pull)
    ensure_fingerprint
    pull_task | jq .
    ;;
  *)
    echo "Usage: $0 {run|once|heartbeat|pull}" >&2
    exit 1
    ;;
esac
