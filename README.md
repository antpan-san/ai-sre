# ai-sre（AI SRE Copilot）

Go 实现的 CLI：**技能包（Skill Pack）+ Prompt 组装 + 可选轻量 RAG + DeepSeek LLM**，对应产品文档中的三条核心能力：

1. **故障诊断** — `analyze`
2. **Runbook 生成** — `runbook`
3. **知识库问答** — `ask`（检索本地 Markdown 片段注入 Prompt）

## 配置

```bash
export DEEPSEEK_API_KEY="你的密钥"   # 必填，勿提交到 Git
# 可选
export DEEPSEEK_BASE_URL="https://api.deepseek.com/v1"
export DEEPSEEK_MODEL="deepseek-chat"
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

密钥仅通过环境变量读取，仓库内不包含任何 API Key。若密钥曾出现在聊天或邮件中，请在 DeepSeek 控制台轮换。
