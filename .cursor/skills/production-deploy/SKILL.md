---
name: production-deploy
description: >-
  Deploy the current ai-sre/OpsFleetPilot repository to production opsfleetpilot.com on
  root@204.44.123.101:10080 while preserving the production Nginx path proxy layout and
  production backend config. Portable for other AI CLI tools that can read this repository.
---

# Production Deploy

Use this skill when the user asks to deploy ai-sre/OpsFleetPilot to production, such as:

- `部署生产环境`
- `上线到 opsfleetpilot.com`
- `将当前版本发布到 204.44.123.101`

This skill is intentionally separate from `release-deploy`: production already has custom Nginx routing on port 80 and must not be overwritten by the lab deployment script.

## Fixed Production Target

| Item | Value |
|------|-------|
| SSH | `ssh -p 10080 root@204.44.123.101` |
| rsync SSH | `ssh -p 10080` |
| Remote root | `/root/sre` |
| Backend service | `opsfleet-backend` |
| Backend port | `19080` from `/root/sre/ft-backend/conf/config.yaml` |
| Web root | `/var/www/opsfleetpilot` |
| Domain | `opsfleetpilot.com` |
| Nginx config to preserve | `/etc/nginx/conf.d/trilium.conf` |
| Public URL | `http://opsfleetpilot.com/` |

Production Nginx currently serves ai-sre at `/`, redirects bare `/` to `/login`, and preserves other services under paths such as `/pentest/` and `/trilium/`.

## Safety Rules

- Do not run `scripts/deploy-opsfleet-remote.sh` against production. It renders the generic Nginx site and can overwrite production path routing.
- Do not overwrite `/root/sre/ft-backend/conf/config.yaml`.
- Do not rewrite `/etc/nginx/conf.d/trilium.conf` unless the user explicitly asks for a production Nginx change.
- Do not delete unknown files on production.
- Always create a timestamped backup before replacing binaries or static assets.
- Preserve existing `/pentest/`, `/trilium/`, `/api/`, `/ft-api/`, `/ws/`, `/uploads/`, and `/health` behavior.
- Exclude local untracked scratch directories such as `examples/` unless the user explicitly asks to publish them.
- If any verification fails after restart, inspect `journalctl -u opsfleet-backend -n 120 --no-pager` and restore from the backup if needed.

## Preflight

Run from repository root:

```bash
pwd
git status --short
git rev-parse --short HEAD
grep -n 'var Version' internal/cli/version.go
```

Check production shape:

```bash
ssh -p 10080 root@204.44.123.101 'hostname; date; test -d /root/sre && echo HAS_SRE || echo NO_SRE'
ssh -p 10080 root@204.44.123.101 'systemctl is-active nginx; systemctl is-active opsfleet-backend; ss -tlnp | grep -E ":(80|19080|8080|8081)" || true'
ssh -p 10080 root@204.44.123.101 'nginx -T 2>/dev/null | grep -nE "server_name|listen|location /|proxy_pass|root " | head -220'
ssh -p 10080 root@204.44.123.101 'test -x /root/sre/bin/ai-sre && /root/sre/bin/ai-sre version || true; curl -fsS http://127.0.0.1/ft-api/api/k8s/deploy/cli/ai-sre/version || true'
```

Expected before deployment may be an older ai-sre version. Do not continue if `opsfleet-backend` or `nginx` is already broken unless the user asked for recovery.

## Backup

Create a full operational backup:

```bash
ssh -p 10080 root@204.44.123.101 'set -euo pipefail
TS=$(date +%Y%m%d%H%M%S)
B=/root/opsfleet-backup-$TS
mkdir -p "$B"
cp -a /etc/nginx/conf.d "$B/nginx-conf.d" 2>/dev/null || true
cp -a /etc/nginx/sites-available "$B/nginx-sites-available" 2>/dev/null || true
cp -a /etc/nginx/sites-enabled "$B/nginx-sites-enabled" 2>/dev/null || true
cp -a /etc/opsfleet "$B/opsfleet-etc" 2>/dev/null || true
cp -a /root/sre/ft-backend/conf/config.yaml "$B/config.yaml" 2>/dev/null || true
cp -a /root/sre/bin "$B/bin" 2>/dev/null || true
cp -a /var/www/opsfleetpilot "$B/web" 2>/dev/null || true
printf "%s\n" "$B"'
```

Record the printed backup directory in the final response.

## Sync Source

Use `rsync` with conservative excludes. This updates source code but preserves production config and local build artifacts:

```bash
rsync -avz --no-owner --no-group -e 'ssh -p 10080' \
  --exclude '.git' \
  --exclude 'ai-sre' \
  --exclude 'bin' \
  --exclude 'dist' \
  --exclude 'ft-front/node_modules' \
  --exclude 'ft-front/dist' \
  --exclude '.DS_Store' \
  --exclude '.env' \
  --exclude '*.zip' \
  --exclude 'examples' \
  --exclude 'ft-backend/conf/config.yaml' \
  ./ root@204.44.123.101:/root/sre/
```

## Build On Production

Build all production artifacts on the server:

```bash
ssh -p 10080 root@204.44.123.101 'cd /root/sre && bash scripts/build-all.sh'
```

Expected outputs:

- `/root/sre/bin/opsfleet-backend`
- `/root/sre/bin/ai-sre`
- optional `/root/sre/bin/ai-sre.arm64`
- `/root/sre/dist/web`

Warnings about Vite chunk size or existing `npm audit` findings do not by themselves fail deployment. Report them as residual risk.

## Activate Build

Replace static files, refresh backend environment, test Nginx config, and restart only the backend service:

```bash
ssh -p 10080 root@204.44.123.101 'set -euo pipefail
rsync -a --delete /root/sre/dist/web/ /var/www/opsfleetpilot/
chown -R www-data:www-data /var/www/opsfleetpilot

install -d -m 755 /etc/opsfleet
ENV_FILE=/etc/opsfleet/backend.env
touch "$ENV_FILE"
chmod 600 "$ENV_FILE"
tmp=$(mktemp)
grep -v "^OPSFLEET_AISRE_BINARY_PATH=" "$ENV_FILE" \
  | grep -v "^OPSFLEET_AISRE_BINARY_PATH_ARM64=" \
  | grep -v "^OPSFLEET_AISRE_BINARY_PATH_AMD64=" \
  | grep -v "^OPSFLEET_AISRE_VERSION=" \
  | grep -v "^OPSFLEET_AI_SKILL_DATA_DIR=" > "$tmp" || true
cat "$tmp" > "$ENV_FILE"
rm -f "$tmp"

echo "OPSFLEET_AISRE_BINARY_PATH=/root/sre/bin/ai-sre" >> "$ENV_FILE"
if [ -f /root/sre/bin/ai-sre.arm64 ]; then
  echo "OPSFLEET_AISRE_BINARY_PATH_ARM64=/root/sre/bin/ai-sre.arm64" >> "$ENV_FILE"
fi
V=$(/root/sre/bin/ai-sre version 2>/dev/null | cut -d" " -f2 | head -1)
if [ -n "${V:-}" ]; then
  echo "OPSFLEET_AISRE_VERSION=$V" >> "$ENV_FILE"
fi
install -d -m 0755 /var/lib/opsfleet/ai-skills/samples /var/lib/opsfleet/ai-skills/feedback /var/lib/opsfleet/ai-skills/generated
echo "OPSFLEET_AI_SKILL_DATA_DIR=/var/lib/opsfleet/ai-skills" >> "$ENV_FILE"

nginx -t
systemctl restart opsfleet-backend
sleep 3
systemctl is-active --quiet opsfleet-backend
curl -fsS http://127.0.0.1/health
printf "\n"
curl -fsS http://127.0.0.1/ft-api/api/k8s/deploy/cli/ai-sre/version
printf "\n"'
```

Do not reload or restart Nginx unless `nginx -t` passes and a Nginx config change was intentionally made.

## Verification

Verify from the production server:

```bash
ssh -p 10080 root@204.44.123.101 'set -e
systemctl is-active opsfleet-backend
systemctl is-active nginx
/root/sre/bin/ai-sre version
curl -fsS http://127.0.0.1/health
printf "\n"
curl -fsS http://127.0.0.1/ft-api/api/k8s/deploy/cli/ai-sre/version
printf "\n"
curl -fsS http://127.0.0.1/login | head -20
printf "\n--- auth-check negative ---\n"
curl -fsS http://127.0.0.1/ft-api/api/cli/go-runtime/auth-check || true
printf "\n--- env ---\n"
grep -E "^OPSFLEET_AISRE_|^OPSFLEET_AI_SKILL" /etc/opsfleet/backend.env'
```

Verify from local network:

```bash
curl -sS -I --connect-timeout 12 http://opsfleetpilot.com/
curl -sS -D - --connect-timeout 12 http://opsfleetpilot.com/ -o /tmp/opsfleet-domain.html
```

Expected:

- `/` returns `302` to `/login`.
- `/login` returns the OpsFleetPilot HTML shell.
- `/health` returns JSON with status `ok`.
- `/ft-api/api/k8s/deploy/cli/ai-sre/version` returns the current `internal/cli/version.go` version.
- unauthenticated `/api/cli/go-runtime/auth-check` or `/ft-api/api/cli/go-runtime/auth-check` returns `401`.

## Rollback

If the backend fails after restart:

```bash
ssh -p 10080 root@204.44.123.101 'journalctl -u opsfleet-backend -n 120 --no-pager'
```

Restore from the backup created at the start:

```bash
ssh -p 10080 root@204.44.123.101 'set -euo pipefail
B=/root/opsfleet-backup-YYYYMMDDHHMMSS
test -d "$B"
if [ -d "$B/bin" ]; then rsync -a --delete "$B/bin/" /root/sre/bin/; fi
if [ -d "$B/web" ]; then rsync -a --delete "$B/web/" /var/www/opsfleetpilot/; chown -R www-data:www-data /var/www/opsfleetpilot; fi
if [ -f "$B/config.yaml" ]; then cp -a "$B/config.yaml" /root/sre/ft-backend/conf/config.yaml; fi
if [ -d "$B/opsfleet-etc" ]; then rsync -a --delete "$B/opsfleet-etc/" /etc/opsfleet/; fi
systemctl restart opsfleet-backend
sleep 3
systemctl is-active --quiet opsfleet-backend
curl -fsS http://127.0.0.1/health'
```

Only restore Nginx directories if the deployment intentionally changed Nginx and the change failed.

## Final Response Checklist

Report:

- Git commit hash deployed.
- ai-sre version deployed.
- Backup directory.
- Whether production `config.yaml` and Nginx routing were preserved.
- Health/version/domain verification results.
- Any warnings, especially npm audit or Redis password warnings.

