# Local Baseline

> Updated: 2026-05-21 20:45 Asia/Shanghai

## Local Code

- Workspace: `/Users/panshuai/Documents/work/code/ai-sre`
- Branch: `main`
- HEAD: `3b9d85209279274b70115207587c7e6745a324dd`
- HEAD summary: `3b9d852 feat(front): tighten app workspace routes`
- Cached `origin/main`: `3b9d852`
- Worktree: clean before baseline recording; later changes are test-environment fixes in this session
- CLI version: `0.6.10`
- Frontend package: `ft-front@0.0.0`

## Local Lab

- Host: `root@192.168.56.11`
- App directory: `/root/sre`
- Hostname: `ubuntu`
- CLI version: `0.6.10`
- `bin/ai-sre`: `0.6.10`
- Services: `opsfleet-backend=active`, `nginx=active`
- Lab Git state: branch `codex/cli-fulfillment-auto-iteration`, HEAD `6246dae14bd59420f733fc04af9efd2fbbc7811e`
- Lab worktree: dirty deployment workspace; do not treat it as source of truth.

## Baseline Checks

- `npm --prefix ft-front run build:check`: pass
- `GOCACHE=/private/tmp/ai-sre-gocache go vet ./...`: pass
- `cd ft-backend && GOCACHE=/private/tmp/ai-sre-gocache go vet ./...`: pass
- `GOCACHE=/private/tmp/ai-sre-gocache go test ./...`: pass after sandbox-aware local TCP test guard
- `cd ft-backend && GOCACHE=/private/tmp/ai-sre-gocache go test ./...`: pass after route-test schema refresh

## Notes

- Project source of truth remains `/Users/panshuai/Documents/work/code/ai-sre`.
- `/root/sre` is a lab deployment workspace, not the authoritative code workspace.
- GitHub pull was performed by the user; local `origin/main` and `HEAD` both resolve to `3b9d852`.
- Tests that require local TCP listeners are skipped only when the host sandbox denies local listen with `operation not permitted`; normal local environments still run them.
