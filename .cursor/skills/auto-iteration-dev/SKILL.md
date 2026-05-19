---
name: auto-iteration-dev
description: >-
  Token-efficient development rules for OpsFleet auto-iteration tasks (console-created
  or code-agent-worker). Apply before and during implementation; pair with release-deploy
  only at ship time.
---

# 自动迭代开发规范（省 Token）

**适用**：控制台「自动迭代」提交的需求、本机 `code-agent-worker` 拉取的任务。  
**目标**：在可交付前提下，减少无效探索、重复读文件、冗长回复与超范围改动。

## 1. 需求书写（提交方 / 产品）

控制台需求建议 **≤800 字**，按块填写（不要散文）：

```text
目标: （一句，要改什么）
范围: （模块或路径，如 ft-front/src/views/admin/）
验收: （2～5 条可检查项，动词开头）
不做: （明确排除项，可选）
约束: （兼容性/权限/性能，可选）
```

**禁止**在需求里贴大段日志、全文件代码、重复截图文字；日志用「最后 30 行」或文件路径。

## 2. Agent 执行（开发阶段）

### 2.1 范围

- **只改**与「目标 / 验收」直接相关的文件。
- **禁止**：顺手重构、全文件格式化、重命名无关符号、扩功能、改无关页面。
- 若发现必须扩 scope：**先**在回复中用 ≤3 行说明原因并停止，等人工确认（不要自行扩大）。

### 2.2 探索（读代码）

| 优先 | 做法 |
|------|------|
| 1 | `grep` / 符号搜索定位入口 |
| 2 | **局部** `Read`（带 offset/limit，大文件禁止通读） |
| 3 | 并行批量工具调用（独立搜索一次做完） |

**禁止**：无关键词全仓遍历；同一文件因犹豫反复 `Read`；把大段现有代码复制进回复。

### 2.3 修改与输出

- 以 **最小 diff** 完成任务；匹配仓库既有风格。
- 回复结构建议：**变更摘要（3～6 行）→ 涉及路径列表 → 如何验收**；不要写教程式长文。
- 引用代码用 `path:line`；不要在回复里贴与 diff 重复的整段实现。

### 2.4 测试

- **开发中**：只跑与改动相关的检查（单测路径、`go test ./pkg/...`、目标前端 build）。
- **发布前**：才执行 `release-deploy` 要求的 `SHORT=1 bash scripts/remote-e2e.sh` 等全量冒烟（见发布 skill）。

### 2.5 阻塞时

用固定格式，**≤6 行**：

```text
阻塞: （一句）
已尝试: （最多 3 条）
需要: （人工决策的一项）
```

不要继续盲目尝试消耗 token。

## 3. 发布（仅开发完成且本地验证通过后）

完整清单以 **`.cursor/skills/release-deploy/SKILL.md`** 为准；本 skill **不重复**其长 checklist。  
发布阶段才需要：README（若用户可见变更）、部署脚本、remote-e2e、commit、push（及技能包 YAML 的独立脚本）。

**禁止**：冒烟未通过却写「已完成」；在 summary 里隐瞒未推送/未部署。

## 4. Worker / 控制台约定

| 项 | 值 |
|----|-----|
| 规范版本 | `auto-iteration-dev@v1` |
| 本文件路径 | `.cursor/skills/auto-iteration-dev/SKILL.md` |
| 发布 skill | `.cursor/skills/release-deploy/SKILL.md` |

Worker 注入的短指令已包含 §2 要点；**首次执行本任务时 Read 本文件一次**，无需每次重读 `release-deploy` 全文。
