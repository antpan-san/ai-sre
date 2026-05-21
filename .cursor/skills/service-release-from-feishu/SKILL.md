---
name: service-release-from-feishu
description: >-
  Read Feishu/Lark service release rows with lark-cli or feishu-cli and create Yunshu services via ops-api, including Git project binding, webhook sync, and service authorization for Feishu groups plus testing group. Use automatically when the request mentions Feishu service table, Yunshu service creation, test prod kmtest environments, service authorization, service name, row index, last N rows, service release, new service launch, 飞书服务表, 云枢创建, 服务授权, or 新服务上线.
---

# 飞书服务表 -> 云枢服务发布

## Hard Constraints

- Use only `OPS_API_BEARER_TOKEN` that the user provides for this run. Do not retrieve tokens from browser storage, OAuth, refresh flows, local files, or other credentials.
- Never print, persist, or include the token in runlogs, commits, screenshots, or responses.
- Treat Feishu as the business source of truth for service name, owner, group, service level, runtime, framework, Git URL, and description.
- Resolve `所属组` from the Feishu row, then always append `测试` (dedupe). Never infer a group from similar services or naming patterns.
- Default to dry-run. Add `--execute` only after the user explicitly asks to create/execute or confirms the displayed plan.
- Require explicit production confirmation before executing any `prod` run.
- Use the bundled script as the main path; do not click through Yunshu UI as the primary creation path.
- Write/update a runlog after every execute attempt, but do not include secrets.

## Data Source And APIs

- Feishu Wiki: `AXppwWPgpiTjPMk2v5QcufZJn1O`
- Spreadsheet token: `HEyssqGClhz92ztmE1ucn6yNnti`
- Sheet/table: `4397b9`
- Read range: `4397b9!A1:J700`
- Create API: `POST /api/v1/manage/service-workloads/create`
- Authorization API: `POST /api/v1/group-service-roles` for each group x `服务开发`/`服务测试`
- Verified prod anchors: `environment_id=6`, `gil-prod-k8s`, `gil-prod-backend`, `gil-prod-golang`; use `--resource-code golang-2c4g` explicitly when 2c4g is required.

## Workflow

1. Read this `SKILL.md`, then open `EXAMPLES.md` before composing commands.
2. Read the target Feishu row(s) and show the user the service row summary plus planned environments.
3. If no token was provided, ask the user for `OPS_API_BEARER_TOKEN`. Do not proceed to API calls without it.
4. Run the command without `--execute` first unless the user already gave an explicit execute/create confirmation.
5. Confirm the plan includes `authorization_groups = <Feishu groups> + 测试` and roles `服务开发` + `服务测试`.
6. For `prod`, stop unless the user has explicitly confirmed production execution.
7. Execute with `--execute` only after confirmation. Capture serviceId/workloadId/status, webhook sync, and authorization verification.
8. Append a concise runlog entry to `~/skills/global-memory-system/projects/service-launch-records-runlog.md` (or `SERVICE_RELEASE_RUNLOG` when set). Exclude tokens.

## Standard Commands

Preferred script path in this workspace:

```bash
OPS_API_BEARER_TOKEN='<token>' \
python3 /Users/panshuai/Documents/work/code/ai-sre/.cursor/skills/service-release-from-feishu/scripts/create_from_last_row.py \
  --service-name <服务名> \
  --envs test,test2,test3,test4,test5,test6
```

Execute after confirmation:

```bash
OPS_API_BEARER_TOKEN='<token>' \
python3 /Users/panshuai/Documents/work/code/ai-sre/.cursor/skills/service-release-from-feishu/scripts/create_from_last_row.py \
  --service-name <服务名> \
  --envs test,test2,test3,test4,test5,test6 \
  --execute
```

Compatibility path if another agent expects the Cursor skill copy:

```bash
OPS_API_BEARER_TOKEN='<token>' \
python3 ~/.cursor/skills/service-release-from-feishu/scripts/create_from_last_row.py \
  --service-name <服务名> \
  --envs test,test2,test3,test4,test5,test6 \
  --execute
```

Read-only last row:

```bash
python3 /Users/panshuai/Documents/work/code/ai-sre/.cursor/skills/service-release-from-feishu/scripts/read_last_row.py
```

## Script Capabilities

| Capability | Parameter |
| --- | --- |
| Last N non-empty rows | `--last-rows N` |
| Exact service name | `--service-name <name>` |
| Exact sheet row number | `--row-index <1-based row>` |
| Environments | `--envs test,test2,test3,test4,test5,test6,kmtest,kmtest2,kmtest3,kmtest4,prod` |
| Execute instead of dry-run | `--execute` |
| Resource profile | `--resource-code <code>` |
| HPA replicas | `--hpa-min-replicas N --hpa-max-replicas N` |
| Fill empty Feishu service level | `--service-level P1` |
| Fill empty runtime | `--runtime-code api|job|consumer|web` |
| Fill empty framework | `--language-framework 'Golang/Kratos'` |
| Fill empty Git URL | `--git-url <url>` |
| Fill empty repo description | `--repo-description <text>` |

CLI fill parameters only patch empty Feishu cells; existing Feishu values win.

## Mapping Rules

- Service name: first column (`student-study-duration`) or `服务名`/`服务名称`.
- Group source order: `服务所属组` -> `服务所属的组` -> `所属组` -> `所属团队` -> `所属端` -> `前端/后端`; append `测试`.
- Owner: strip `@`, resolve exactly against Yunshu users by `displayName` or `name`; do not substitute the token owner.
- Roles: resolve `服务开发` and `服务测试` dynamically from API.
- Environment templates: resolve by `gil-{env}-{language}` such as `gil-prod-golang` or `gil-kmtest-golang`; avoid reusing `gil-test-golang` for prod. Fallback to test templates only for non-prod when a specific env template is absent.
- Image namespace: match `gil-{env}-backend` exactly, including `kmtest2` etc.; do not confuse `test2` and `kmtest2`.
- IDs: resolve environment, cluster, namespace, template, resource, language, framework, service level, runtime, groups, and roles from API options/lists at runtime. Do not hard-code except documented verified anchors for sanity checks.
- Existing `(environment_code, service_name)` workloads are skipped, but webhook sync and group authorization still run idempotently.

## Prod Rules

- The user must explicitly request `prod`/生产 and explicitly confirm production execution.
- Always dry-run first for `prod` unless the same message includes both token and an unambiguous production execution confirmation.
- Use prod-specific template resolution (`gil-prod-golang`, etc.). Never apply test template IDs or names to prod.
- Specify prod resource intentionally, for example `--resource-code golang-2c4g --hpa-min-replicas 1 --hpa-max-replicas 1`.

## Output And Runlog

Successful execute output should include:

- Per environment: `workloadId`, `serviceId`, and `status` (`created` or `skipped_exists`).
- `sync.success` or equivalent webhook sync result.
- `authorization.verified` with expected `groupName` and `serviceRoleName`.

Runlog path:

`~/skills/global-memory-system/projects/service-launch-records-runlog.md` by default, overrideable with `SERVICE_RELEASE_RUNLOG` for tests or workspace-local runs.

Include timestamp, Feishu row index, service name, environments, serviceId/workloadId, resource code, HPA, authorization groups/roles, Git status, and any error summary. Never include `OPS_API_BEARER_TOKEN`.

## References

- Read `EXAMPLES.md` for command patterns before running.
- Scripts live in `scripts/create_from_last_row.py` and `scripts/read_last_row.py`.
