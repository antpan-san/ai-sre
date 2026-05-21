#!/usr/bin/env python3
import argparse
import json
import os
import re
import subprocess
import sys
from datetime import datetime
from pathlib import Path
import urllib.parse
import urllib.request


SPREADSHEET_TOKEN = "HEyssqGClhz92ztmE1ucn6yNnti"
SHEET_ID = "4397b9"
SHEET_RANGE = f"{SHEET_ID}!A1:J700"
OPS_BASE = "https://ops-api.xiaoluxue.cn"
OPS_ORIGIN = "https://ops.xiaoluxue.cn"
DEFAULT_ENVS = ["test", "test2", "test3", "test4", "test5", "test6"]


def run_json(cmd):
    result = subprocess.run(cmd, text=True, capture_output=True, check=False)
    if result.returncode != 0:
        raise RuntimeError(result.stderr or result.stdout or "command failed")
    return json.loads(result.stdout)


def read_sheet_rows():
    return run_json(
        [
            "lark-cli",
            "sheets",
            "+read",
            "--spreadsheet-token",
            SPREADSHEET_TOKEN,
            "--range",
            SHEET_RANGE,
            "--as",
            "user",
            "--value-render-option",
            "ToString",
        ]
    )


def read_last_row():
    payload = read_sheet_rows()
    rows = payload["data"]["valueRange"]["values"]
    header = rows[0]
    last = None
    for row_index, row in enumerate(rows[1:], start=2):
        if any(cell not in (None, "") for cell in row):
            last = (row_index, row)
    if not last:
        raise RuntimeError("no non-empty row found")
    row_index, row = last
    mapped = {header[i]: row[i] if i < len(row) else None for i in range(len(header))}
    return row_index, mapped


def read_last_n_rows(n):
    if n < 1:
        raise RuntimeError("last-rows must be >= 1")
    payload = read_sheet_rows()
    rows = payload["data"]["valueRange"]["values"]
    header = rows[0]
    nonempty = []
    for row_index, row in enumerate(rows[1:], start=2):
        if any(cell not in (None, "") for cell in row):
            nonempty.append((row_index, row))
    if not nonempty:
        raise RuntimeError("no non-empty row found")
    tail = nonempty[-n:]
    out = []
    for row_index, row in tail:
        mapped = {header[i]: row[i] if i < len(row) else None for i in range(len(header))}
        out.append((row_index, mapped))
    return out


def read_row_by_index(row_index):
    payload = read_sheet_rows()
    rows = payload["data"]["valueRange"]["values"]
    header = rows[0]
    if row_index < 2 or row_index > len(rows):
        raise RuntimeError(f"row index {row_index} out of range (sheet has {len(rows)} rows)")
    row = rows[row_index - 1]
    mapped = {header[i]: row[i] if i < len(row) else None for i in range(len(header))}
    return row_index, mapped


def read_row_by_service_name(service_name):
    payload = read_sheet_rows()
    rows = payload["data"]["valueRange"]["values"]
    header = rows[0]
    matches = []
    for row_index, row in enumerate(rows[1:], start=2):
        name = row[0] if row else ""
        if str(name or "").strip() == service_name:
            mapped = {header[i]: row[i] if i < len(row) else None for i in range(len(header))}
            matches.append((row_index, mapped))
    if not matches:
        raise RuntimeError(f"service name not found in sheet: {service_name}")
    if len(matches) > 1:
        raise RuntimeError(f"multiple rows found for service name {service_name}: {[m[0] for m in matches]}")
    return matches[0]


def patch_row(row, patches):
    out = dict(row)
    for key, value in patches.items():
        if value is None:
            continue
        if out.get(key) in (None, ""):
            out[key] = value
    return out


class OpsClient:
    def __init__(self, token):
        self.token = token

    def request(self, method, path, params=None, data=None, env="test"):
        url = OPS_BASE + path
        if params:
            url += "?" + urllib.parse.urlencode(params)
        headers = {
            "Origin": OPS_ORIGIN,
            "Referer": f"{OPS_ORIGIN}/service-manage/{env}",
            "Accept": "application/json, text/plain, */*",
            "Authorization": f"Bearer {self.token}",
        }
        body = None
        if data is not None:
            body = json.dumps(data, ensure_ascii=False).encode("utf-8")
            headers["Content-Type"] = "application/json"
        req = urllib.request.Request(url, data=body, headers=headers, method=method)
        try:
            with urllib.request.urlopen(req, timeout=30) as resp:
                text = resp.read().decode("utf-8", "replace")
                return json.loads(text) if text else {}
        except urllib.error.HTTPError as exc:
            text = exc.read().decode("utf-8", "replace")
            raise RuntimeError(f"HTTP {exc.code}: {text[:1000]}") from exc


def token_from_runtime():
    token = os.environ.get("OPS_API_BEARER_TOKEN")
    if token:
        token = token.strip()
        if token.lower().startswith("bearer "):
            token = token.split(None, 1)[1].strip()
        return token.strip()
    raise RuntimeError("missing OPS_API_BEARER_TOKEN; ask the user to provide a Yunshu access token")


def first_present(row, *keys):
    for key in keys:
        if row.get(key):
            return str(row[key]).strip()
    return ""


def strip_owner(owner_text):
    return [x.strip() for x in re.split(r"@", owner_text or "") if x.strip()]


def split_group_names(group_text):
    names = [x.strip() for x in re.split(r"[,，、;；/\n]+", group_text or "") if x.strip()]
    if "测试" not in names:
        names.append("测试")
    return names


def repo_name_from_url(git_url):
    tail = (git_url or "").rstrip("/").split("/")[-1]
    return re.sub(r"\.git$", "", tail)


def exact_one(items, predicate, label):
    matches = [x for x in items if predicate(x)]
    if len(matches) != 1:
        raise RuntimeError(f"expected one {label}, got {len(matches)}")
    return matches[0]


def resolve_service_groups(row, groups):
    group_text = first_present(row, "服务所属组", "服务所属的组", "所属组", "所属团队", "所属端", "前端/后端")
    if not group_text:
        raise RuntimeError("source row missing service group; read it from the Feishu document instead of inferring")

    resolved = []
    for name in split_group_names(group_text):
        group = exact_one(groups, lambda x, n=name: x.get("name") == n, f"group {name}")
        resolved.append({"id": str(group["id"]), "name": group["name"]})
    return resolved


def resolve_default_service_roles(roles):
    wanted = ["服务开发", "服务测试"]
    return [
        {"id": str(exact_one(roles, lambda x, n=name: x.get("name") == n, f"service role {name}")["id"]), "name": name}
        for name in wanted
    ]


def resolve_template(opts, env, language_code):
    fallbacks = {
        "golang": "gil-test-golang",
        "python": "gil-test-python",
        "typescript": "gil-test-ant-design-pro",
    }
    candidates = []
    if language_code in ("golang", "python"):
        candidates.append(f"gil-{env}-{language_code}")
    elif language_code == "typescript":
        candidates.append(f"gil-{env}-ant-design-pro")

    # Never let prod accidentally fall back to a test template.
    if env != "prod" and fallbacks.get(language_code):
        candidates.append(fallbacks[language_code])

    for name in candidates:
        matches = [t for t in opts.get("serviceTemplates") or [] if t.get("name") == name]
        if len(matches) == 1:
            return matches[0]
    raise RuntimeError(f"no template for env={env} language={language_code}, tried {candidates}")


def default_resource_code(env, language_code):
    return {"golang": "golang-250m512mi", "python": "python-500m1g"}.get(language_code)


def resolve_git_project(client, opts, row):
    git_url = first_present(row, "git地址")
    repo_name = repo_name_from_url(git_url)
    candidates = opts.get("gitProjects") or []
    matches = [g for g in candidates if g.get("name") == repo_name or g.get("code") == repo_name]
    if matches:
        return str(matches[0]["id"]), repo_name, "existing"

    payload = {
        "name": repo_name,
        "code": repo_name,
        "git_url": git_url,
        "description": first_present(row, "仓库简介", "服务用途"),
        "provider_id": 5,
        "gitlab_project_id": None,
        "ci_webhook_id": 24,
    }
    try:
        created = client.request("POST", "/api/v1/git-projects", data=payload)
        gid = created.get("id") or created.get("gitProjectId") or created.get("git_project_id")
        if gid:
            return str(gid), repo_name, "created"
    except Exception:
        pass

    queried = client.request("GET", "/api/v1/git-projects", params={"name": repo_name})
    items = queried.get("items") or queried.get("data") or []
    matches = [g for g in items if g.get("name") == repo_name or g.get("code") == repo_name]
    if not matches:
        raise RuntimeError(f"cannot resolve git project for {repo_name}")
    return str(matches[0]["id"]), repo_name, "queried"


def build_plan(client, row, envs, runtime_override=None, resource_override=None, hpa_min=None, hpa_max=None):
    opts = client.request("GET", "/api/v1/manage/service-workloads/form-options")
    users = client.request("GET", "/api/v1/users", {"page": 1, "page_size": 10000}).get("items") or []
    groups = client.request("GET", "/api/v1/groups", {"page": 1, "page_size": 1000}).get("items") or []
    roles = client.request("GET", "/api/v1/service-roles", {"page": 1, "page_size": 1000}).get("items") or []

    service_name = first_present(row, "student-study-duration", "服务名", "服务名称")
    owner_names = strip_owner(first_present(row, "负责人"))
    if not service_name:
        raise RuntimeError("source row missing service name")
    if not owner_names:
        raise RuntimeError("source row missing owner")
    owner_ids = []
    for name in owner_names:
        user = exact_one(
            users,
            lambda u, n=name: u.get("displayName") == n or u.get("name") == n,
            f"owner {name}",
        )
        owner_ids.append(str(user["id"]))
    service_groups = resolve_service_groups(row, groups)
    service_roles = resolve_default_service_roles(roles)

    language_text = first_present(row, "编程语言/框架")
    language_token, _, framework_token = language_text.partition("/")
    language_token = language_token.strip().lower()
    framework_token = framework_token.strip()
    lang_alias = {"golang": "golang", "go": "golang", "python": "python", "typescript": "typescript"}
    language_code = lang_alias.get(language_token, language_token)
    language = exact_one(
        opts.get("languages") or [],
        lambda x: str(x.get("code", "")).lower() == language_code or str(x.get("name", "")).lower() == language_code,
        f"language {language_code}",
    )
    framework = exact_one(
        opts.get("frameworks") or [],
        lambda x: x.get("name") == framework_token and str(x.get("languageId") or x.get("language_id")) == str(language["id"]),
        f"framework {framework_token}",
    )

    level_code = first_present(row, "服务等级")
    level = exact_one(opts.get("serviceLevels") or [], lambda x: x.get("code") == level_code, f"level {level_code}")
    service_type = exact_one(opts.get("serviceTypes") or [], lambda x: x.get("name") == "后端服务", "backend service type")

    category = first_present(row, "服务分类").lower()
    runtime_code = runtime_override or (category if category in {"api", "consumer", "job", "web"} else "web")
    runtime = exact_one(opts.get("runtimeBehaviors") or [], lambda x: x.get("code") == runtime_code, f"runtime {runtime_code}")
    workload_type = exact_one(opts.get("k8sWorkloadTypes") or [], lambda x: x.get("code") == "Deployment", "Deployment")

    git_project_id, git_project_name, git_status = resolve_git_project(client, opts, row)

    plan = []
    for env in envs:
        template = resolve_template(opts, env, language_code)
        resource_code = resource_override or default_resource_code(env, language_code)
        if not resource_code:
            raise RuntimeError(f"no resource rule for env={env} language={language_code}")
        resource = exact_one(opts.get("resourceProfiles") or [], lambda x: x.get("code") == resource_code, f"resource {resource_code}")
        env_obj = exact_one(opts.get("environments") or [], lambda x, e=env: x.get("code") == e, f"environment {env}")
        cluster = exact_one(
            opts.get("k8sClusters") or [],
            lambda x, eid=str(env_obj["id"]): str(x.get("envId") or x.get("env_id")) == eid,
            f"cluster for {env}",
        )
        namespace = exact_one(
            opts.get("k8sNamespaces") or [],
            lambda x, e=env, cid=str(cluster["id"]): x.get("name") == e and str(x.get("k8sClusterId") or x.get("k8s_cluster_id")) == cid,
            f"namespace {env}",
        )
        image_ns_name = f"gil-{env}-backend"
        image_ns = exact_one(opts.get("imageNamespaces") or [], lambda x, n=image_ns_name: x.get("name") == n, image_ns_name)
        existing = client.request(
            "GET",
            "/api/v1/manage/service-workloads",
            {"page": 1, "page_size": 10, "environment_code": env, "service_name": service_name},
            env=env,
        ).get("items") or []
        exact_existing = [x for x in existing if (x.get("serviceName") or x.get("workloadName")) == service_name]
        payload = {
            "service_name": service_name,
            "k8s_namespace": env,
            "instance_count": 1,
            "environment_id": int(env_obj["id"]),
            "business_id": 1,
            "cloud_provider_id": 1,
            "k8s_cluster_id": int(cluster["id"]),
            "git_project_id": int(git_project_id),
            "git_provider_id": 5,
            "k8s_workload_type_id": int(workload_type["id"]),
            "template_id": int(template["id"]),
            "resource_profile_id": int(resource["id"]),
            "resource_profile_code": resource["code"],
            "image_registry_id": 8,
            "image_namespace_id": int(image_ns["id"]),
            "image_repository_name": service_name,
            "business_container_name": language_code,
            "service_level_id": int(level["id"]),
            "service_type_id": int(service_type["id"]),
            "framework_id": int(framework["id"]),
            "language_id": int(language["id"]),
            "runtime_behavior_id": int(runtime["id"]),
            "hpa_min_replicas": hpa_min if hpa_min is not None else 1,
            "hpa_max_replicas": hpa_max if hpa_max is not None else 1,
            "owner_user_ids": owner_ids,
            "service_description": first_present(row, "服务用途", "仓库简介"),
        }
        plan.append(
            {
                "env": env,
                "skip_existing": bool(exact_existing),
                "existing_service_id": exact_existing[0].get("serviceId") if exact_existing else None,
                "existing_workload_id": exact_existing[0].get("id") if exact_existing else None,
                "summary": {
                    "cluster": cluster["name"],
                    "namespace": namespace["name"],
                    "image_namespace": image_ns["name"],
                    "template": template["name"],
                    "resource": f"{resource['code']} ({resource.get('description')})",
                    "runtime": runtime["code"],
                    "git_project": git_project_name,
                    "git_project_status": git_status,
                    "owners": owner_names,
                    "authorization_groups": [group["name"] for group in service_groups],
                    "authorization_roles": [role["name"] for role in service_roles],
                },
                "payload": payload,
            }
        )
    return {
        "row": row,
        "service_name": service_name,
        "plan": plan,
        "git_project_id": git_project_id,
        "authorization": {
            "source_groups": service_groups,
            "roles": service_roles,
        },
    }


def execute_plan(client, plan):
    results = []
    for item in plan["plan"]:
        env = item["env"]
        if item["skip_existing"]:
            results.append(
                {
                    "env": env,
                    "status": "skipped_exists",
                    "serviceId": item["existing_service_id"],
                    "workloadId": item["existing_workload_id"],
                }
            )
            continue
        created = client.request("POST", "/api/v1/manage/service-workloads/create", data=item["payload"], env=env)
        results.append(
            {
                "env": env,
                "status": "created",
                "serviceId": created.get("serviceId"),
                "workloadId": created.get("workloadId"),
                "message": created.get("message"),
            }
        )
    sync = client.request("POST", "/api/v1/git-projects/sync-webhook", data={"git_project_ids": [plan["git_project_id"]]})
    authorization = apply_service_authorization(client, plan, results)
    return {"results": results, "sync": sync, "authorization": authorization}


def create_group_service_role(client, payload):
    try:
        return client.request("POST", "/api/v1/group-service-roles", data=payload)
    except RuntimeError as exc:
        text = str(exc)
        if "HTTP 409" in text and ("already exists" in text.lower() or "已存在" in text):
            return {"idempotent": True}
        raise


def apply_service_authorization(client, plan, results):
    service_ids = sorted({str(item.get("serviceId")) for item in results if item.get("serviceId")})
    if not service_ids:
        raise RuntimeError("cannot authorize groups without a resolved serviceId")

    auth_results = []
    for service_id in service_ids:
        for group in plan["authorization"]["source_groups"]:
            for role in plan["authorization"]["roles"]:
                payload = {
                    "group_id": int(group["id"]),
                    "service_id": int(service_id),
                    "service_role_id": int(role["id"]),
                }
                result = create_group_service_role(client, payload)
                auth_results.append(
                    {
                        "serviceId": service_id,
                        "group": group["name"],
                        "role": role["name"],
                        "result": result,
                    }
                )

    verified = client.request(
        "GET",
        "/api/v1/services-authorization",
        params={"page": 1, "page_size": 10, "service_name": plan["service_name"]},
    )
    return {"created": auth_results, "verified": verified.get("items") or []}


DEFAULT_RUNLOG_PATH = Path.home() / "skills" / "global-memory-system" / "projects" / "service-launch-records-runlog.md"


def resolve_runlog_path():
    return Path(os.environ.get("SERVICE_RELEASE_RUNLOG", DEFAULT_RUNLOG_PATH))


def append_runlog(entries, envs, resource_code, hpa_min, hpa_max, error=None):
    try:
        runlog_path = resolve_runlog_path()
        runlog_path.parent.mkdir(parents=True, exist_ok=True)
        now = datetime.now().isoformat(timespec="seconds")
        lines = [f"\n## {now} service-release-from-feishu"]
        for entry in entries:
            plan = entry.get("plan") or {}
            auth = plan.get("authorization") or {}
            groups = [g.get("name") for g in auth.get("source_groups") or []]
            roles = [r.get("name") for r in auth.get("roles") or []]
            lines.append(f"- row_index: {entry.get('row_index')}")
            lines.append(f"- service_name: {plan.get('service_name')}")
            lines.append(f"- envs: {','.join(envs)}")
            lines.append(f"- resource_code: {resource_code or 'default'}")
            lines.append(f"- hpa: {hpa_min if hpa_min is not None else 1}-{hpa_max if hpa_max is not None else 1}")
            lines.append(f"- authorization_groups: {', '.join(groups)}")
            lines.append(f"- authorization_roles: {', '.join(roles)}")
            result = entry.get("result")
            if result:
                compact = json.dumps(result, ensure_ascii=False, separators=(",", ":"))
                lines.append(f"- result: {compact}")
            if error:
                lines.append(f"- error: {error}")
        runlog_path.open("a", encoding="utf-8").write("\n".join(lines) + "\n")
        return str(runlog_path)
    except Exception as exc:
        print(f"WARN: failed to write runlog: {exc}", file=sys.stderr)
        return None


def main():
    parser = argparse.ArgumentParser(description="Create Yunshu services from the last Feishu service row.")
    parser.add_argument("--envs", default=",".join(DEFAULT_ENVS), help="comma-separated env codes")
    parser.add_argument("--execute", action="store_true", help="actually create services; default is plan only")
    parser.add_argument("--runtime-code", help="override runtime behavior code, e.g. job/api/consumer/web")
    parser.add_argument("--resource-code", help="override resource profile code")
    parser.add_argument(
        "--last-rows",
        type=int,
        default=1,
        metavar="N",
        help="process the last N non-empty sheet rows (default: 1)",
    )
    parser.add_argument("--service-name", help="process the row with this exact service name")
    parser.add_argument("--row-index", type=int, help="process a specific 1-based sheet row index")
    parser.add_argument("--service-level", help="override 服务等级 when Feishu cell is empty")
    parser.add_argument("--language-framework", help="override 编程语言/框架 when Feishu cell is empty")
    parser.add_argument("--git-url", help="override git地址 when Feishu cell is empty")
    parser.add_argument("--repo-description", help="override 仓库简介 when Feishu cell is empty")
    parser.add_argument("--hpa-min-replicas", type=int, help="override hpa_min_replicas (default 1)")
    parser.add_argument("--hpa-max-replicas", type=int, help="override hpa_max_replicas (default 1)")
    args = parser.parse_args()

    env_list = [x.strip() for x in args.envs.split(",") if x.strip()]
    if args.service_name:
        rows = [read_row_by_service_name(args.service_name.strip())]
    elif args.row_index:
        rows = [read_row_by_index(args.row_index)]
    else:
        rows = read_last_n_rows(args.last_rows)

    row_patches = {
        "服务等级": args.service_level,
        "编程语言/框架": args.language_framework,
        "git地址": args.git_url,
        "仓库简介": args.repo_description,
        "服务分类": args.runtime_code,
    }
    rows = [(idx, patch_row(row, row_patches)) for idx, row in rows]
    client = OpsClient(token_from_runtime())

    all_plans = []
    for row_index, row in rows:
        plan = build_plan(
            client,
            row,
            env_list,
            args.runtime_code,
            args.resource_code,
            args.hpa_min_replicas,
            args.hpa_max_replicas,
        )
        all_plans.append({"row_index": row_index, "plan": plan})

    print(
        json.dumps(
            [
                {
                    "row_index": item["row_index"],
                    "row": item["plan"]["row"],
                    "service_name": item["plan"]["service_name"],
                    "plan": [{k: v for k, v in p.items() if k != "payload"} for p in item["plan"]["plan"]],
                    "authorization": item["plan"]["authorization"],
                }
                for item in all_plans
            ],
            ensure_ascii=False,
            indent=2,
        )
    )
    if args.execute:
        combined = []
        runlog_entries = []
        for item in all_plans:
            result = execute_plan(client, item["plan"])
            combined_item = {"row_index": item["row_index"], "service_name": item["plan"]["service_name"], "result": result}
            combined.append(combined_item)
            runlog_entries.append({"row_index": item["row_index"], "plan": item["plan"], "result": result})
        print(json.dumps(combined, ensure_ascii=False, indent=2))
        runlog_path = append_runlog(runlog_entries, env_list, args.resource_code, args.hpa_min_replicas, args.hpa_max_replicas)
        if runlog_path:
            print(f"runlog.updated: {runlog_path}")
    else:
        print("DRY_RUN: add --execute after confirmation to create services.")


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        sys.exit(1)
