#!/usr/bin/env bash
# Fail if skill pack YAML is staged or tracked for commit (core assets must not go to GitHub).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

patterns=(
  'ft-backend/skills/builtin/*.yaml'
  'internal/assets/skills/*.yaml'
)

fail=0
for pat in "${patterns[@]}"; do
  # Allow deletions (stop tracking); block adding or modifying YAML content.
  if git diff --cached --diff-filter=AM --name-only -- "$pat" 2>/dev/null | grep -q '\.yaml$'; then
    echo "ERROR: staged skill pack add/modify (forbidden on GitHub):" >&2
    git diff --cached --diff-filter=AM --name-only -- "$pat" >&2
    fail=1
  fi
done

tracked=$(git ls-files 'ft-backend/skills/builtin/*.yaml' 'internal/assets/skills/*.yaml' 2>/dev/null || true)
if [ -n "$tracked" ]; then
  echo "ERROR: skill pack YAML still tracked by git (run: git rm --cached <files>):" >&2
  echo "$tracked" >&2
  fail=1
fi

if [ "$fail" -ne 0 ]; then
  echo "See .cursor/skills/skill-pack-assets/SKILL.md" >&2
  exit 1
fi

echo "OK: no skill pack YAML staged or tracked"
