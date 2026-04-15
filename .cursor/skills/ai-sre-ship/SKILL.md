---
name: ai-sre-ship
description: >-
  After any code change in the ai-sre repository, deploys sources to root@172.16.195.128:/root/sre,
  runs remote build and smoke tests, then commits and pushes to git@github.com:antpan-san/ai-sre.git
  when tests pass. Use whenever ai-sre code is edited, fixed, or the user finishes a feature—do not
  wait for an explicit deploy request.
---

# ai-sre 发布与同步（强制工作流）

在 **ai-sre** 项目（`/Users/panshuai/Documents/work/code/ai-sre`）中**每次完成代码修改后**，代理必须按顺序执行本流程；用户已配置 **root@172.16.195.128** 免密 SSH。

## 固定参数

| 项 | 值 |
|----|-----|
| 远程主机 | `root@172.16.195.128` |
| 远程目录 | `/root/sre`（不存在则创建） |
| GitHub 远程 | `git@github.com:antpan-san/ai-sre.git` |
| 本地项目根 | `/Users/panshuai/Documents/work/code/ai-sre` |

## 执行顺序（必须完整）

### 1. 部署到远程

在本地项目根执行（可直接调用脚本）：

```bash
./scripts/deploy-remote.sh
```

脚本行为：确保远程目录存在、`rsync` 同步源码（排除 `.git` 与二进制）、远程 `go mod download`、`go vet ./...`、`go build`、运行 `./ai-sre version`。

若脚本不存在，则等价手动执行：

```bash
ssh root@172.16.195.128 "mkdir -p /root/sre"
rsync -avz \
  --exclude '.git' \
  --exclude 'ai-sre' \
  --exclude '.DS_Store' \
  /Users/panshuai/Documents/work/code/ai-sre/ \
  root@172.16.195.128:/root/sre/
ssh root@172.16.195.128 'cd /root/sre && go mod download && go vet ./... && go build -o ai-sre . && ./ai-sre version'
```

**不要使用 `--delete` rsync**（除非用户明确要求），以免误删服务器上额外文件。

### 2. 远程功能测试（必须通过）

- **必测**：远程构建成功、`./ai-sre version` 正常。
- **可选（若环境中有 `DEEPSEEK_API_KEY`）**：在远程执行一条需 LLM 的命令验证（例如 `export DEEPSEEK_API_KEY=... && ./ai-sre --no-rag ask test`）；若无密钥，则跳过并注明「LLM 集成需在目标机配置 `DEEPSEEK_API_KEY` 后手测」。

若任一步失败：**停止推送 GitHub**，先修复代码并重复 1–2。

### 3. 推送到 GitHub（测试通过后）

在**本地**项目根（含 `.git` 的仓库）：

```bash
cd /Users/panshuai/Documents/work/code/ai-sre
git status
git add -A
git commit -m "feat: <简明英文或中文说明本次变更>"   # 若无变更可跳过 commit
git push -u origin main
```

- 若尚未初始化仓库：`git init`，`git remote add origin git@github.com:antpan-san/ai-sre.git`，首次推送 `git push -u origin main`。
- 默认分支名以仓库为准；若远程为 `main` 而本地为 `master`，按 `git branch -M main` 对齐后再推送。

### 4. 向用户汇报

简短说明：已部署路径、远程测试结果、Git 提交哈希或「已 push」。

## 失败处理

- **SSH/rsync 失败**：检查网络与 `ssh root@172.16.195.128` 连通性，勿反复推送未验证的代码。
- **远程 go build 失败**：在本地复现 `go build` 并修复后再部署。
- **git push 失败**：根据错误处理（权限、冲突、未提交）；不强行覆盖远程。

## 与用户提示的关系

用户不要求每次说「请部署」：只要本轮对话中**改动了 ai-sre 代码**，在完成修改后**自动应用本 skill**，除非用户明确说「仅本地、不要部署」。
