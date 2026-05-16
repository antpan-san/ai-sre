---
name: skill-pack-assets
description: >-
  Core skill pack YAML must not be pushed to GitHub. Deploy to lab for testing and
  to production for canonical storage. Code-only git push; use deploy-skill-packs-*.sh.
---

# 技能包核心资产（禁止上 GitHub）

## 策略（用户指定）

| 环境 | 技能包 YAML | GitHub |
|------|-------------|--------|
| **GitHub 仓库** | **禁止** 提交/推送 `*.yaml` 技能包正文 | 仅代码、README 占位、部署脚本 |
| **本地** | 允许维护 `ft-backend/skills/builtin/`、`internal/assets/skills/` | 不 push |
| **实验室 192.168.56.11** | **允许** rsync 部署与联调 | — |
| **生产 204.44.123.101** | **权威存放** + 对外服务注册表 | — |

`git push` **只推代码**；**不得**把「技能包已发布」等同于 push 成功。

## 路径

| 用途 | 本地（gitignore） | 服务端运行时 |
|------|-------------------|--------------|
| Builtin | `ft-backend/skills/builtin/*.yaml` | `/var/lib/opsfleet/ai-skills/builtin/` |
| Generated | — | `.../ai-skills/generated/` |
| 样本/反馈 | — | `.../samples/`、`.../feedback/` |

## 部署命令（代理必须会）

```bash
# 实验室联调
./scripts/deploy-skill-packs-lab.sh

# 生产权威环境
./scripts/deploy-skill-packs-production.sh
```

推送前自检（仓库根）：

```bash
./scripts/check-skill-packs-not-in-git.sh
```

## 与 release 的关系

- 触及 **仅** 技能包 YAML：实验室脚本 + 生产脚本；**不要**把 YAML 放进 `git commit`。
- 触及 **SkillRegistry / API** 代码：走 `release-deploy` → 实验室代码部署 → `git push` → 若同时改了本地 YAML，再跑上述技能包脚本。

## 禁止

- 将 `ft-backend/skills/builtin/*.yaml` 或 `internal/assets/skills/*.yaml` 加入 commit。
- 在回复中宣称「已 push 到 GitHub」即等于技能包已安全发布（生产 `curl .../api/ai/skills` 才算）。
