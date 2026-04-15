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
./ai-sre --no-rag ask "redis 慢查询怎么查"   # 关闭 RAG
```

二进制别名：`ops-ai`（`go build` 后可用 `ln -s ai-sre ops-ai`）。

## 布局

- `internal/assets/skills/*.yaml` — 技能包（可扩展）
- `internal/assets/knowledge/*.md` — RAG 知识片段
- `internal/engine` — 编排 skill + prompt + RAG + LLM
- `internal/rag` — 关键词检索（无向量库依赖，可后续换 embedding）

## 安全说明

密钥仅从本机文件读取，勿将 `api_key` / `config.yaml` 提交到 Git。建议目录权限 `700`、密钥文件 `600`。

## 发布流程（团队约定）

代码变更后：在仓库根执行 `./scripts/deploy-remote.sh`（同步至 `root@172.16.195.128:/root/sre` 并远程构建冒烟测试），通过后提交并 `git push` 到 `git@github.com:antpan-san/ai-sre.git`。详细步骤见 `.cursor/skills/ai-sre-ship/SKILL.md`（Agent 在修改本仓库代码后应自动遵循）。
