#!/usr/bin/env bash
# Safe git push: block skill-pack YAML and common secret paths from reaching GitHub.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "==> skill pack gate"
bash "$ROOT/scripts/check-skill-packs-not-in-git.sh"

deny_paths=(
  'ft-backend/skills/builtin/*.yaml'
  'internal/assets/skills/*.yaml'
  '.env'
  '**/.env'
  '**/backend.env'
  '**/code-agent-worker.env'
  '**/*api_key*'
  '**/*credentials*'
  '**/*.pem'
  '**/*.key'
  '**/secrets/**'
)

echo "==> unstage forbidden paths"
while IFS= read -r f; do
  [[ -n "$f" ]] || continue
  for pat in "${deny_paths[@]}"; do
    if [[ "$f" == $pat ]] || [[ "$f" == */backend.env ]] || [[ "$f" == */code-agent-worker.env ]]; then
      echo "  unstage: $f"
      git reset -q HEAD -- "$f" 2>/dev/null || true
      break
    fi
  done
done < <(git diff --cached --name-only 2>/dev/null || true)

echo "==> scan staged diff for secret-like content"
if git diff --cached | grep -Ei '(api[_-]?key|password\s*=|secret\s*=|BEGIN (RSA |OPENSSH )?PRIVATE|dingtalk.*webhook|OPSFLEET_CODE_AGENT_TOKEN\s*=)' >/dev/null; then
  echo "ERROR: staged diff may contain secrets; remove before push" >&2
  exit 1
fi

echo "==> push"
git push "$@"
