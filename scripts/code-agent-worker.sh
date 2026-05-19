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
USE_GIT_WORKTREE="${USE_GIT_WORKTREE:-1}"
WORKTREE_ROOT="${WORKTREE_ROOT:-${REPO_ROOT}/.worktrees/auto-iteration}"
STREAM_AGENT_LOGS="${STREAM_AGENT_LOGS:-1}"

STATE_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/ai-sre"
FP_FILE="$STATE_DIR/code-agent-fingerprint"
LOG_FILE="${CODE_AGENT_WORKER_LOG:-$STATE_DIR/code-agent-worker.log}"

ACTIVE_WORKSPACE="$REPO_ROOT"

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
  curl -sS -X "$method" "${OPSFLEET_API_URL}${path}" \
    -H "Authorization: Bearer ${OPSFLEET_CODE_AGENT_TOKEN}" \
    -H "X-OpsFleet-Agent-Fingerprint: ${FINGERPRINT}" \
    -H "Content-Type: application/json" \
    "$@"
}

heartbeat() {
  api POST /api/code-agent/heartbeat -d '{}' | jq -e '.code == 200 and .data.ok == true' >/dev/null
}

report_event() {
  local task_id="$1" message="$2"
  local payload="${3:-{}}"
  api POST "/api/code-agent/tasks/${task_id}/events" \
    -d "$(jq -n --arg m "$message" --argjson p "$payload" '{message:$m,payload:$p}')" >/dev/null
}

report_result() {
  local task_id="$1" success="$2" summary="$3"
  local github_sync="${4:-skipped}"
  local deploy_status="${5:-skipped}"
  local rollback_required="${6:-false}"
  api POST "/api/code-agent/tasks/${task_id}/result" \
    -d "$(jq -n \
      --argjson s "$success" \
      --arg sum "$summary" \
      --arg gs "$github_sync" \
      --arg ds "$deploy_status" \
      --argjson rr "$rollback_required" \
      '{success:$s,summary:$sum,github_sync:$gs,deploy_status:$ds,rollback_required:$rr}')" >/dev/null
}

pull_task() {
  api GET /api/code-agent/tasks/pull | jq -c '.data.task // .task // empty'
}

task_worker_option() {
  local task_json="$1" key="$2"
  jq -r --arg k "$key" '.worker_options[$k] // empty' <<<"$task_json"
}

prepare_worktree() {
  local task_id="$1"
  if [[ "$USE_GIT_WORKTREE" != "1" ]]; then
    ACTIVE_WORKSPACE="$REPO_ROOT"
    return 0
  fi
  require_cmd git
  mkdir -p "$WORKTREE_ROOT"
  local wt="$WORKTREE_ROOT/$task_id"
  local branch="auto-iter/${task_id:0:8}"
  if [[ -d "$wt" ]]; then
    ACTIVE_WORKSPACE="$wt"
    return 0
  fi
  log "Creating git worktree at $wt (branch $branch)"
  git -C "$REPO_ROOT" fetch origin main 2>/dev/null || true
  git -C "$REPO_ROOT" worktree add -B "$branch" "$wt" origin/main 2>/dev/null \
    || git -C "$REPO_ROOT" worktree add -B "$branch" "$wt" main
  ACTIVE_WORKSPACE="$wt"
}

cleanup_worktree() {
  local task_id="$1"
  [[ "$USE_GIT_WORKTREE" == "1" ]] || return 0
  local wt="$WORKTREE_ROOT/$task_id"
  if [[ -d "$wt" ]]; then
    git -C "$REPO_ROOT" worktree remove --force "$wt" 2>/dev/null || rm -rf "$wt"
    git -C "$REPO_ROOT" worktree prune 2>/dev/null || true
  fi
  ACTIVE_WORKSPACE="$REPO_ROOT"
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
    "仓库: 仅改需求相关文件；开发期禁止全量 remote-e2e；发布期 SHORT=1 bash scripts/remote-e2e.sh 通过后再 commit/push。\n"
  ' <<<"$task_json"
}

run_agent() {
  local task_id="$1" prompt="$2"
  local agent_args=(--print --trust --workspace "$ACTIVE_WORKSPACE")
  if [[ -n "${AGENT_MODEL:-}" ]]; then
    agent_args+=(--model "$AGENT_MODEL")
  fi
  log "Starting Cursor Agent in $ACTIVE_WORKSPACE"
  if [[ "$DRY_RUN_AGENT" == "1" ]]; then
    log "DRY_RUN_AGENT=1, skipping agent"
    return 0
  fi
  if [[ "$STREAM_AGENT_LOGS" == "1" ]]; then
    # Stream stdout/stderr lines to server events for SSE consumers.
    set +e
    # shellcheck disable=SC2090
    "$AGENT_BIN" "${agent_args[@]}" "$prompt" 2>&1 | while IFS= read -r line; do
      echo "$line" >>"$LOG_FILE"
      if [[ ${#line} -gt 4000 ]]; then
        line="${line:0:4000}…"
      fi
      report_event "$task_id" "$line" '{"stream":"agent"}' || true
    done
    local rc=${PIPESTATUS[0]}
    set -e
    return "$rc"
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
  (cd "$ACTIVE_WORKSPACE" && SHORT=1 bash scripts/remote-e2e.sh)
}

run_github_push() {
  local task_json="$1"
  local enabled task_id script
  enabled=$(task_worker_option "$task_json" "github_sync_enabled")
  task_id=$(jq -r '.id' <<<"$task_json")
  if [[ "$enabled" == "false" ]]; then
    return 2
  fi
  if [[ "$enabled" != "true" && "${RUN_GITHUB_PUSH:-1}" != "1" ]]; then
    return 2
  fi
  script="$ACTIVE_WORKSPACE/scripts/github-push-safe.sh"
  if [[ ! -f "$script" ]]; then
    script="$REPO_ROOT/scripts/github-push-safe.sh"
  fi
  if [[ ! -f "$script" ]]; then
    return 2
  fi
  log "GitHub push via github-push-safe.sh"
  report_event "$task_id" "开始 GitHub 同步" '{"phase":"github"}'
  set +e
  (cd "$ACTIVE_WORKSPACE" && bash "$script") >>"$LOG_FILE" 2>&1
  local rc=$?
  set -e
  return "$rc"
}

process_task() {
  local task_json="$1"
  local task_id title github_sync=skipped deploy_status=skipped
  task_id=$(jq -r '.id' <<<"$task_json")
  title=$(jq -r '.title' <<<"$task_json")
  log "Processing task $task_id: $title"

  prepare_worktree "$task_id"
  trap 'cleanup_worktree "'"$task_id"'"' RETURN

  report_event "$task_id" "Worker 开始执行（本机 Cursor Agent）" "{\"phase\":\"agent\",\"workspace\":\"$ACTIVE_WORKSPACE\"}"

  local prompt agent_rc=0 verify_rc=0 push_rc=0 summary
  prompt=$(build_prompt "$task_json")

  set +e
  run_agent "$task_id" "$prompt"
  agent_rc=$?
  set -e

  if [[ $agent_rc -ne 0 ]]; then
    summary="Cursor Agent 退出码 ${agent_rc}，详见 ${LOG_FILE}"
    report_event "$task_id" "$summary" "{\"agent_exit\":$agent_rc}"
    report_result "$task_id" false "$summary" skipped skipped false
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
    report_result "$task_id" false "$summary" skipped failed true
    log "Task $task_id failed (verify)"
    return
  fi
  deploy_status=ok

  set +e
  run_github_push "$task_json"
  push_rc=$?
  set -e
  case "$push_rc" in
    0) github_sync=ok ;;
    2) github_sync=skipped ;;
    *) github_sync=failed ;;
  esac
  if [[ "$github_sync" == "failed" ]]; then
    report_event "$task_id" "GitHub push 失败，可重试 sync-github" "{\"github_sync\":\"failed\"}"
  fi

  summary="本机 Agent 与 remote-e2e 均通过"
  if [[ "$github_sync" == "ok" ]]; then
    summary+="；GitHub 已同步"
  elif [[ "$github_sync" == "failed" ]]; then
    summary+="；GitHub 同步失败（可重试）"
  fi
  report_event "$task_id" "$summary" '{"phase":"done"}'
  report_result "$task_id" true "$summary" "$github_sync" "$deploy_status" false
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
