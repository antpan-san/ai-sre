# Builtin 技能包（核心资产，勿提交 Git）

本目录下的 `*.yaml` 为项目核心资产，**禁止** `git add` / push 到 GitHub（见仓库根 `.gitignore`）。

| 环境 | 部署方式 |
|------|----------|
| 本地开发 | 直接编辑本目录；CLI 自动加载（见 `internal/loader`） |
| 实验室 `192.168.56.11` | `./scripts/deploy-skill-packs-lab.sh` |
| 生产 `204.44.123.101` | `./scripts/deploy-skill-packs-production.sh` |

运行时权威副本（服务端）：`$OPSFLEET_AI_SKILL_DATA_DIR/builtin/`（默认 `/var/lib/opsfleet/ai-skills/builtin/`）。
