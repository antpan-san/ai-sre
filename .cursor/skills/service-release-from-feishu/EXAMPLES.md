# 飞书 -> 云枢发布：命令示例

所有命令将 `<token>` 替换为用户当次提供的云枢 access token（可带 `Bearer ` 前缀）。不要写入仓库或 runlog。

## 1. 最后一行 -> test 全环境

```bash
OPS_API_BEARER_TOKEN='<token>' \
python3 /Users/panshuai/Documents/work/code/ai-sre/.cursor/skills/service-release-from-feishu/scripts/create_from_last_row.py \
  --envs test,test2,test3,test4,test5,test6

OPS_API_BEARER_TOKEN='<token>' \
python3 /Users/panshuai/Documents/work/code/ai-sre/.cursor/skills/service-release-from-feishu/scripts/create_from_last_row.py \
  --envs test,test2,test3,test4,test5,test6 \
  --execute
```

## 2. 最后 2 行 -> test + kmtest 全环境

```bash
OPS_API_BEARER_TOKEN='<token>' \
python3 /Users/panshuai/Documents/work/code/ai-sre/.cursor/skills/service-release-from-feishu/scripts/create_from_last_row.py \
  --last-rows 2 \
  --envs test,test2,test3,test4,test5,test6,kmtest,kmtest2,kmtest3,kmtest4 \
  --execute
```

## 3. 指定服务名 -> 仅 test

```bash
OPS_API_BEARER_TOKEN='<token>' \
python3 /Users/panshuai/Documents/work/code/ai-sre/.cursor/skills/service-release-from-feishu/scripts/create_from_last_row.py \
  --service-name study-manager-update-course-package \
  --envs test \
  --runtime-code job \
  --execute
```

飞书行若已补全 `P3`、`git` 等，不要再用 CLI 覆盖已有列；脚本也只补空。

## 4. 指定服务名 -> prod（2c4g，副本 1 到 1）

必须先获得用户明确生产确认。示例：

```bash
OPS_API_BEARER_TOKEN='<token>' \
python3 /Users/panshuai/Documents/work/code/ai-sre/.cursor/skills/service-release-from-feishu/scripts/create_from_last_row.py \
  --service-name xxl-class-battle-snapshot \
  --envs prod \
  --resource-code golang-2c4g \
  --hpa-min-replicas 1 \
  --hpa-max-replicas 1 \
  --execute
```

## 5. 飞书列不全时补空（仅空单元格生效）

```bash
OPS_API_BEARER_TOKEN='<token>' \
python3 /Users/panshuai/Documents/work/code/ai-sre/.cursor/skills/service-release-from-feishu/scripts/create_from_last_row.py \
  --service-name astrolabe-app-settings-api \
  --envs test \
  --service-level P1 \
  --runtime-code job \
  --language-framework 'Golang/Kratos' \
  --git-url 'https://gitlab.xiaoluxue.cn/astrolabe/astrolabe-runtime-app-settings-api' \
  --repo-description '学管端后端' \
  --execute
```

## 6. 只读飞书、不创建

```bash
python3 /Users/panshuai/Documents/work/code/ai-sre/.cursor/skills/service-release-from-feishu/scripts/read_last_row.py
```

或按服务名查表：

```bash
python3 <<'PY'
import json, subprocess
name = "xxl-class-battle-snapshot"
cmd = ["lark-cli","sheets","+read","--spreadsheet-token","HEyssqGClhz92ztmE1ucn6yNnti",
       "--range","4397b9!A1:J700","--as","user","--value-render-option","ToString"]
rows = json.loads(subprocess.run(cmd, capture_output=True, text=True, check=True).stdout)["data"]["valueRange"]["values"]
h = rows[0]
for i, r in enumerate(rows[1:], 2):
    if (r[0] or "").strip() == name:
        print(json.dumps({h[j]: r[j] if j < len(r) else None for j in range(len(h))}, ensure_ascii=False, indent=2))
PY
```

## 7. 组授权规则（无需单独命令）

脚本在 `--execute` 时自动：

- 飞书 `前端/后端`/所属组等列 -> 云枢组名
- 必须追加 `测试` 组
- 每组绑定 `服务开发` + `服务测试`

## 8. 成功输出应包含

- 每环境：`workloadId`、`serviceId`、`status`（`created` 或 `skipped_exists`）
- `sync.success`
- `authorization.verified` 中含预期 `groupName` 与 `serviceRoleName`

## 9. Agent 检查清单

- [ ] 已读 `SKILL.md` 和 `EXAMPLES.md`
- [ ] 已读飞书目标行并展示给用户
- [ ] 已说明组授权 = 飞书组 + `测试`
- [ ] 有用户提供的 token 才调用云枢 API
- [ ] 默认先无 `--execute`，或用户已明确确认创建
- [ ] prod 已获用户明确确认
- [ ] 执行后更新 `service-launch-records-runlog.md`
- [ ] 响应中不包含 token
