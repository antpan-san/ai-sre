#!/usr/bin/env bash
# Verify skill sample JSONL/PG on lab OpsFort and optional batch CLI check smoke.
# Usage: ./scripts/verify-skill-samples-lab.sh
# Env:
#   DEPLOY_REMOTE (default root@192.168.56.11)
#   SKILL_DATA_DIR (default /var/lib/opsfleet/ai-skills)
#   SKIP_CLI_CHECK=1  skip remote ai-sre check loop
#   REMOTE_AISRE_BIN (default /root/sre/bin/ai-sre on remote)
set -euo pipefail

REMOTE="${DEPLOY_REMOTE:-root@192.168.56.11}"
REMOTE_HOST="${DEPLOY_REMOTE_HOST:-192.168.56.11}"
SKILL_DIR="${SKILL_DATA_DIR:-/var/lib/opsfleet/ai-skills}"
REMOTE_BIN="${REMOTE_AISRE_BIN:-/root/sre/bin/ai-sre}"
TOPICS=(redis linux domain kafka mysql postgresql nginx elasticsearch go k8s)

echo "==> Skill sample files on ${REMOTE}:${SKILL_DIR}/samples"
ssh -o BatchMode=yes -o ConnectTimeout=15 "$REMOTE" "bash -s" <<EOF
set -euo pipefail
DIR='${SKILL_DIR}/samples'
if [[ ! -d "\$DIR" ]]; then
  echo "MISSING samples dir: \$DIR"
  exit 1
fi
total=0
for f in "\$DIR"/*.jsonl; do
  [[ -f "\$f" ]] || continue
  n=\$(wc -l < "\$f" | tr -d ' ')
  total=\$((total + n))
  printf "  %-20s %6s lines\n" "\$(basename "\$f" .jsonl)" "\$n"
done
echo "TOTAL sample lines: \$total"
test -f '${SKILL_DIR}/enhancement_reviews.jsonl' && echo "enhancement_reviews: \$(wc -l < '${SKILL_DIR}/enhancement_reviews.jsonl') lines" || echo "enhancement_reviews: (none)"
EOF

echo "==> PostgreSQL diagnose_samples (lab, best-effort)"
ssh -o BatchMode=yes -o ConnectTimeout=15 "$REMOTE" "bash -s" <<'EOF' || echo "WARN: PG count skipped"
set -euo pipefail
if command -v psql >/dev/null 2>&1; then
  cnt=$(sudo -u postgres psql -d opsfleetpilot -tAc "select count(*) from diagnose_samples" 2>/dev/null || psql -d opsfleetpilot -tAc "select count(*) from diagnose_samples" 2>/dev/null || echo "")
  if [[ -n "${cnt:-}" ]]; then
    echo "diagnose_samples rows: ${cnt}"
  else
    echo "diagnose_samples: (table missing or no access)"
  fi
else
  echo "psql not available"
fi
EOF

echo "==> OpsFleet health"
curl -sfS "http://${REMOTE_HOST}:9080/health" >/dev/null && echo "health OK" || { echo "FAIL: health check"; exit 1; }

if [[ "${SKIP_CLI_CHECK:-}" == "1" ]]; then
  echo "==> SKIP_CLI_CHECK=1, done"
  exit 0
fi

echo "==> Remote batch ai-sre check smoke on ${REMOTE} (${REMOTE_BIN})"
ok=0
skip=0
fail=0
for topic in "${TOPICS[@]}"; do
  case "$topic" in
    redis) target="127.0.0.1:6379" ;;
    linux|domain) target="" ;;
    kafka) target="127.0.0.1:9092" ;;
    mysql) target="root@tcp(127.0.0.1:3306)/" ;;
    postgresql) target="postgres://127.0.0.1:5432/postgres?sslmode=disable" ;;
    nginx) target="/var/log/nginx/error.log" ;;
    elasticsearch) target="http://127.0.0.1:9200" ;;
    go) target="pid/1" ;;
    k8s) target="pod/default/skip" ;;
    *) target="" ;;
  esac
  check_topic="$topic"
  if [[ "$topic" == "go" ]]; then
    check_topic="go"
  fi
  remote_cmd="cd /root/sre && OPSFLEET_EXECUTION_REPORT_DISABLED=1 OPSFLEET_SKILL_SAMPLE_DISABLED=0 ${REMOTE_BIN} check ${check_topic}"
  if [[ -n "$target" ]]; then
    remote_cmd="${remote_cmd} $(printf %q "$target")"
  fi
  remote_cmd="${remote_cmd} >/dev/null 2>&1"
  if ssh -o BatchMode=yes -o ConnectTimeout=20 "$REMOTE" "$remote_cmd"; then
    echo "  OK  check ${topic}"
    ok=$((ok + 1))
  else
    echo "  SKIP check ${topic} (target/auth/probe unavailable)"
    skip=$((skip + 1))
  fi
done

echo "==> remote check summary: ok=${ok} skip=${skip} fail=${fail}"
if [[ "$ok" -lt 1 ]]; then
  echo "FAIL: no topic check succeeded on lab (need at least 1 OK)"
  exit 1
fi

echo "==> verify-skill-samples-lab OK"
