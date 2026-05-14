---
name: security-first-development
description: Use for every ai-sre/OpsFleetPilot code change that touches backend APIs, frontend routing/request code, auth, files, task execution, agents, deployment scripts, Nginx, databases, Redis, or any externally reachable surface. Enforces secure-by-default development, threat review, and release checks; portable to other AI CLI tools as a standalone SKILL.md.
---

# Security-First Development

Use this skill before editing and before finalizing any change that can affect runtime security.

## Goal

Prevent easy compromise of the OpsFleetPilot site, API, managed servers, deployment hosts, and user data. Treat this project as an operations control plane: a low-friction bug can become server takeover.

## Minimum Workflow

1. **Classify the touched surface**
   - Public unauthenticated endpoint
   - JWT-protected endpoint
   - Agent/client-token endpoint
   - File upload/download/static serving
   - Shell/task/service/Kubernetes execution
   - Frontend route/request/storage/rendering
   - Deployment/Nginx/systemd/database/Redis configuration

2. **Name the attacker**
   - Anonymous internet user
   - Authenticated low-privilege user
   - Stolen token holder
   - Malicious/compromised managed machine
   - User controlling form fields, file names, URLs, command text, or YAML

3. **Apply the control checklist below**

4. **Add or update tests/checks**
   - At minimum, cover auth/unauth path, invalid input, authorization boundary, and dangerous payload rejection.

5. **Remove redundant description**
   - UI text, docs, code comments, labels, alerts, and final notes must only say what changes behavior or prevents mistakes.
   - Do not add marketing copy, duplicated label explanations, or prose that repeats adjacent UI text.
   - Code comments are for non-obvious constraints, security boundaries, migrations, and operational gotchas only.

6. **Run completion checks**
   - Always run `git diff --check`.
   - If backend or Go CLI changed, run `go test ./...`; when the repository has multiple Go modules, run it in each touched module. Also run `go build ./...` for backend/API changes.
   - If frontend changed, run `npm run build`.
   - If visual layout changed, inspect with a browser or screenshot workflow.
   - If a check is blocked by sandbox, missing services, or environment permissions, rerun with the appropriate approved/escalated path when safe; otherwise report the exact blocker.

7. **Final response must include a Security Notes section**
   - State changed attack surface, controls added, tests run, and residual risk.

## Backend API Rules

- Every new route must be explicitly classified as `public`, `protected`, or `agent-token`.
- Default to JWT-protected. Public routes require a written reason in code review notes.
- Public routes must have at least one of:
  - unguessable high-entropy token checked with constant-time comparison,
  - strict read-only semantics,
  - explicit rate limiting / abuse boundary,
  - no sensitive data or server-side side effects.
- Do not leak internal errors, stack traces, SQL errors, filesystem paths, tokens, secrets, or command output to anonymous callers.
- Do not use user-controlled strings in SQL except through GORM parameters or placeholders.
- Validate and bound all pagination: clamp `pageSize`, reject negative `page`, avoid unbounded `Find`.
- Validate IDs with UUID parsing before use.
- Enforce tenant/user ownership in queries, not just on the frontend.
- Enforce role authorization server-side for admin-only actions; frontend role checks are not security.
- For dangerous protected actions, require admin role and consider a second confirmation token or explicit dry-run/preview path.

## Auth, Token, Session

- Passwords must use bcrypt/argon2; never store plaintext or reversible passwords.
- JWT secret must come from production config/secret storage and must not be a default.
- JWT validation must not reveal parse details to clients.
- Do not log Authorization headers, JWTs, deploy tokens, passwords, private keys, or generated install refs.
- Prefer short-lived tokens for browser auth.
- Long-lived deploy/invite/client tokens must be random, high entropy, stored hashed when possible, and compared constant-time.
- If adding refresh-token behavior, store revocation state or avoid issuing refresh tokens.
- Login should have brute-force protection or rate limiting before exposing broadly.

## CORS and Browser Security

- Never ship `Access-Control-Allow-Origin: *` with credentials. Prefer an allowlist from config.
- Add security headers at Nginx or backend:
  - `X-Content-Type-Options: nosniff`
  - `X-Frame-Options: DENY` or equivalent CSP `frame-ancestors 'none'`
  - `Referrer-Policy: no-referrer` or `strict-origin-when-cross-origin`
  - restrictive `Content-Security-Policy` when feasible
- Browser tokens in `localStorage` are XSS-sensitive. If touching auth/frontend rendering, actively look for XSS sinks.
- Avoid `dangerouslyUseHTMLString`, `v-html`, `innerHTML`, `eval`, `new Function`. If unavoidable, sanitize with a proven sanitizer and document why.
- For links with `target="_blank"`, add `rel="noopener noreferrer"`.

## File and Static Serving

- Uploads must enforce:
  - max body size,
  - allowed extension and MIME/content sniffing where meaningful,
  - generated server-side filename,
  - storage under a fixed upload root,
  - path traversal prevention with `filepath.Clean` + root prefix check.
- Public download routes must check visibility/share token/ownership. A bare file UUID is not enough for private files.
- `Content-Disposition` filenames must be safely quoted/escaped.
- Never serve arbitrary paths from user input.
- Do not expose `/uploads` publicly unless every file under it is intended to be public.

## Command, Task, Agent, and Kubernetes Execution

- Treat any command execution feature as critical severity.
- Never pass untrusted input to `sh -c`, `bash -c`, `sudo`, SSH, Ansible, kubectl, helm, docker, tar, zip, curl, or rsync without a reviewable allowlist.
- Prefer structured command builders with fixed argv arrays over shell strings.
- If raw shell is a product feature, restrict it to admin, log it, require explicit target selection, bound timeout/output size, and display a destructive-action warning.
- Validate machine IDs and ownership/tenant before dispatch.
- Do not let a compromised agent pull arbitrary tasks unless it proves identity with a secret not derived from public IP/name.
- Client/agent report endpoints must authenticate the reporting machine and verify task ownership.
- Generated install scripts must quote all variables, use `set -euo pipefail`, and avoid printing secrets.
- Kubernetes/deployment YAML generated from UI input must validate IPs, CIDRs, hostnames, versions, registry names, and URLs.

## AI/LLM Features

- Public LLM endpoints are abuse-prone. Require auth or a dedicated server-side token unless intentionally public.
- Never send secrets, private keys, JWTs, database passwords, or full config files to LLM providers.
- Treat model output as untrusted text. Do not execute, save as trusted YAML, or apply generated skill changes without validation.
- For skill evolution/refine endpoints, validate topic/file names against a strict allowlist and prevent path traversal.

## Deployment and Infrastructure

- Production Nginx must route only intended paths; preserve existing services with explicit path prefixes.
- Do not bind backend admin APIs directly to the public internet; expose through Nginx.
- Do not reuse ports without checking `ss -tlnp`.
- Production config files must be `0600` when they contain secrets.
- Redis should bind to localhost or require auth. Do not expose Redis publicly.
- PostgreSQL should use a least-privilege app user, not superuser, whenever possible.
- Back up Nginx/systemd/config files before editing production.

## ai-sre Current High-Risk Areas To Recheck

When touching these files, perform a security review even for small changes:

- `ft-backend/routes/router.go`: public vs protected route placement.
- `ft-backend/middleware/cors.go`: wildcard CORS and credentials.
- `ft-backend/handlers/auth.go`: login, token creation, brute-force behavior.
- `ft-backend/handlers/file.go`: public downloads, upload validation, `/uploads` serving.
- `ft-backend/handlers/job.go`: raw shell dispatch to managed machines.
- `ft-backend/handlers/k8s_*.go`: generated install scripts, bundle tokens, inventory/YAML.
- `ft-backend/handlers/service_deployments.go`: deploy tokens, bootstrap scripts, event reporting.
- `ft-backend/handlers/ai_*.go`: unauthenticated LLM/skill endpoints.
- `ft-front/src/utils/request.ts`: token handling and 401 behavior.
- `ft-front/src/router/index.ts`: frontend guards are UX only, not authorization.
- `ft-front/src/views/job/JobCenter.vue`: HTML rendering and command execution warnings.
- `deploy/`, `scripts/`: production Nginx/systemd/SSH deployment behavior.

## Security Review Output Template

Use this format in final answers and PR notes:

```markdown
Security Notes:
- Attack surface changed:
- Auth/authorization:
- Input validation:
- Secrets/logging:
- File/command/agent risk:
- Tests/checks run:
- Residual risk:
```

## Blockers

Do not mark a change complete if any of these are true:

- New public endpoint without documented reason and abuse boundary.
- Admin/server operation reachable by non-admin users.
- Untrusted input reaches shell/system commands without allowlist or clear containment.
- File path is derived from user input without root confinement.
- Secret or token can appear in logs, response bodies, URLs, or generated scripts unnecessarily.
- Frontend adds an XSS sink without sanitization.
- Production deployment modifies Nginx/systemd without backup and verification.
