# ai-sre（AI SRE Copilot）

Go 实现的 CLI：**技能包（Skill Pack）+ Prompt 组装 + 可选轻量 RAG + DeepSeek LLM**，对应产品文档中的三条核心能力：

1. **故障诊断** — `analyze`
2. **Runbook 生成** — `runbook`
3. **知识库问答** — `ask`（检索本地 Markdown 片段注入 Prompt）

## 配置（文件，不使用环境变量）

在运行机器上放置密钥，**二选一**即可：

**方式 A — YAML（推荐）** 默认路径：`~/.config/ai-sre/config.yaml`（若设置 `XDG_CONFIG_HOME`，则为 `$XDG_CONFIG_HOME/ai-sre/config.yaml`）

```yaml
api_key: "你的 DeepSeek API Key"
# 可选
base_url: "https://api.deepseek.com/v1"
model: "deepseek-chat"
```

**方式 B — 仅密钥文件** 默认路径：`~/.config/ai-sre/api_key`（纯文本，第一行为密钥；`#` 开头行为注释）

命令行覆盖：

- `--config /path/to/config.yaml` 指定 YAML
- `--key-file /path/to/api_key` 指定仅含密钥的文件

示例：

```bash
mkdir -p ~/.config/ai-sre
chmod 700 ~/.config/ai-sre
printf '%s\n' '你的密钥' > ~/.config/ai-sre/api_key
chmod 600 ~/.config/ai-sre/api_key
```

## 构建与示例

```bash
go build -o ai-sre .
./ai-sre analyze kafka --lag 100000
./ai-sre analyze k8s --pod pending
./ai-sre ask "kafka lag 高怎么办"
./ai-sre runbook "pod频繁重启"
./ai-sre skills list                    # 技能注册表 / 发现
./ai-sre -o json analyze kafka --lag 1  # 结构化 JSON 输出
./ai-sre --no-rag ask "redis 慢查询怎么查"   # 关闭 RAG
```

二进制别名：`ops-ai`（`go build` 后可用 `ln -s ai-sre ops-ai`）。

## 结构化输出（产品文档「结构化输出」）

`analyze` / `ask` / `runbook` 支持 `-o json`：返回 `answer`、`skill`（命中的技能包）、`duration_ms`、`context` 等字段，便于流水线与自动化。

## 自定义技能与知识库（扩展）

与内置 `internal/assets` **合并**加载；同名技能以**后加载的目录为准**（覆盖内置）。

```bash
./ai-sre --skills-dir ./my-skills --knowledge-dir ./my-docs analyze redis --latency 10ms
```

- `--skills-dir`：目录下放多个 `*.yaml`，格式与内置技能相同。
- `--knowledge-dir`：目录下放多个 `*.md`，按段落参与 RAG 检索。

技能 YAML 中可使用占位符 `{{lag}}`、`{{topic}}` 等（与 `--set`/各子命令 flag 注入的 context 键一致）。

## 布局（对齐产品文档）

- `internal/cli` — 命令路由（Cobra）
- `internal/engine` — AI 编排
- `internal/skill` — 技能包加载与匹配
- `internal/prompt` — Prompt 模板
- `internal/rag` — 轻量知识检索
- `internal/output` — 文本 / JSON 输出
- `internal/llm` — DeepSeek（OpenAI 兼容）
- `internal/loader` — 内置资源 + 可选目录合并
- `internal/assets/skills/*.yaml` — 内置技能包
- `internal/assets/knowledge/*.md` — 内置 RAG 片段

## 安全说明

密钥仅从本机文件读取，勿将 `api_key` / `config.yaml` 提交到 Git。建议目录权限 `700`、密钥文件 `600`。

## 发布流程（团队约定）

代码变更后：在仓库根执行 `./scripts/deploy-remote.sh`（同步至 `root@172.16.195.128:/root/sre` 并远程构建冒烟测试），通过后提交并 `git push` 到 `git@github.com:antpan-san/ai-sre.git`。详细步骤见 `.cursor/skills/ai-sre-ship/SKILL.md`（Agent 在修改本仓库代码后应自动遵循）。
