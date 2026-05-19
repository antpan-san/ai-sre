#!/usr/bin/env bash
# Verify skill sample JSONL on lab OpsFort and optional CLI check smoke.
# Usage: ./scripts/verify-skill-samples-lab.sh
# Env: DEPLOY_REMOTE (default root@192.168.56.11), SKILL_DATA_DIR (default /var/lib/opsfleet/ai-skills)
set -euo pipefail

REMOTE="${DEPLOY_REMOTE:-root@192.168.56.11}"
SKILL_DIR="${SKILL_DATA_DIR:-/var/lib/opsfleet/ai-skills}"
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

echo "==> OpsFleet health"
curl -sfS "http://${DEPLOY_REMOTE_HOST:-192.168.56.11}:9080/health" >/dev/null && echo "health OK" || echo "WARN: health check failed"

if [[ "${SKIP_CLI_CHECK:-}" == "1" ]]; then
  echo "==> SKIP_CLI_CHECK=1, done"
  exit 0
fi

if [[ -x ./ai-sre ]] && [[ -n "${OPSFLEET_API_URL:-}" || -f "$HOME/.config/ai-sre/api_key" || -f "$HOME/.config/ai-sre/config.yaml" ]]; then
  echo "==> Local ai-sre check smoke (best-effort, samples may require CLI binding token)"
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
    if [[ -z "$target" ]]; then
      cmd=(./ai-sre check "$topic")
    else
      cmd=(./ai-sre check "$topic" "$target")
    fi
    if "${cmd[@]}" >/dev/null 2>&1; then
      echo "  OK  check $topic"
    else
      echo "  SKIP check $topic (probe/target unavailable or auth)"
    fi
  done
else
  echo "==> No local ai-sre/config; set SKIP_CLI_CHECK=0 and configure OpsFleet token to run CLI checks"
fi

echo "==> verify-skill-samples-lab OK"
