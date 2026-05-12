---
name: error-code-development-gate
description: >-
  门控清单：在开始或合并任何「平台部署/离线 K8s/安装脚本/ansible/制品镜像/API 报错」功能时，
  必须补齐结构化错误码、emit 语义与 ai-sre analyze code / 控制台同源目录，便于使用者按码直达根因而非排查清单。
---

# OpsFleet / ai-sre 错误码「开发门控」

在 **ai-sre 同仓** 内，凡是可能让 **install.sh、bootstrap.sh、ansible、镜像同步、控制台 K8s 流程** 出现**可预测的失败**，代理与贡献者在 **开始设计与结尾 PR** 时都须过本门禁。

## 何时必须打开本 Skill

符合**任一**即视为触发（与 `.cursor/rules/monorepo-release.mdc` 中 K8s 离线路径部分重叠）：

- 改动 `ansible-agent/`（roles、playbooks、inventory、脚本）
- 改动 `ft-backend/handlers/k8s_bundle.go`、`k8s_ansible.go`、`k8s_mirror_*`、离线包生成
- 改动 `deploy/k8s-mirror/`、动态 `install-ai-sre.sh` / `bootstrap.sh` 中与安装失败相关的输出
- 新增或调整 **可被用户看见的失败形态**（超时、checksum、镜像拉取、端口未监听等）

不要求：纯 UI 样式、与本安装链无关的业务 API。

## 门控清单（Definition of Done）

1. **`[ERROR-CODE]` 可机读落地**  
   - 在安装链关键失败点输出一行：`[ERROR-CODE] <CODE> ...`（已有 pattern：`download-with-progress.sh`、`wait_apiserver.yml`、`install.sh` 的 `run()` 等）。  
   - **禁止**只写中文长文而无稳定 CODE；CODE 必须 `OPSFLEET_` 前缀、全大写、下划线分段。

2. **目录有「根因卡」**  
   - 在 **`ft-backend/skills/builtin/error_codes.yaml`** 增加或更新 `error_codes[]` 条目：  
     `code`、`summary`、`root_cause`、`typical_evidence[]`、`recovery_one_liner`、`platform_followup`、`related_codes[]`。  
   - **根因卡**回答的是「为什么」+「一行恢复」+「代码落点」，**不是**步骤清单。

3. **后端与 CLI 同源**  
   - 公开 API：`GET /ft-api/api/ai/error-codes`、`POST /ft-api/api/ai/error-codes/analyze`（实现于 `ft-backend/handlers/error_codes.go`）。  
   - CLI：`ai-sre analyze code <CODE> [--detail …]`、`--list`（实现于 `internal/cli/analyze_code.go`）。

4. **注册表可查**  
   - `services.SkillRegistry` 载入 builtin YAML 后技能数递增；`/api/ai/skills` 可看到 `opsfleet_error_codes_v1`。  
   - 若为 **同一现象的新子类**，优先 **扩展 `related_codes`** 或在 `recovery_one_liner` 中区分场景，避免重复 CODE 碎片化。

5. **文档与用户可见一句话**  
   - 若影响运维路径：在 **`README.md`**「错误码」小节补：**遇到失败 → 搜 `[ERROR-CODE]` → `ai-sre analyze code <CODE>`**。  
   - 若控制台需要独立页：再在 `ft-front` 增加路由（与本 skill 可分期；门控上不阻塞纯后端可先合）。

6. **测试**  
   - `cd ft-backend && go test ./services/... ./handlers/...`（builtin YAML 可被解析）。  
   - 如能连实验室：冒烟 `curl`/CLI 校验新 CODE 的分析响应非空。

## 新增错误码命名约定（供 AI 与人的共同语言）

| 前缀 / 分段 | 含义 |
|-------------|------|
| `OPSFLEET_K8S_E_*` | Kubernetes 离线/控制平面/组件安装 |
| `OPSFLEET_DL_E_*` | 下载/校验（checksum、磁盘、mirror） |
| `OPSFLEET_NET_E_*` | （预留）网络、DNS、超时（若与下载重叠则归 `DL`） |
| `OPSFLEET_EXEC_E_*` | （预留）执行机/控制机环境（见「部署机 kubectl」相关） |

## 与发布流程的关系

本 skill **不替代** `release-deploy`、`k8s-offline-deploy-test`；在命中 K8s 离线条件时 **两者一起做**：先满足本门禁，再跑离线部署测试与远端冒烟。

## 交互提问（代理人若不确定）

在实现前若仍不清楚，**必须**向用户确认其中一项或多选：

1. 失败是否在 **单机 / 全域**、**可否重试自愈**？
2. 用户可见的是 **ANSI 日志**、**HTTP JSON**、**仅 Web Toast**？
3. 同一 CODE 是否要 **多语言**（当前 builtin 卡以中文为主，API 不改变 code 本身）。
4. 是否需要 **租户/工单** 维度关联 CODE（若要，再在 `analyze` payload 中加字段，不落本门禁范围）。
