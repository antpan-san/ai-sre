# 功能分级与多功能包订阅

## 策略

ai-sre 采用“功能入口可见，执行前强校验”的订阅模型。普通用户可以看到高级能力入口、配置表单和说明，但真实客户机动作、报告生成、离线包下载、CLI 收费命令和 AI 技能调用必须由后端校验权益。

`super_admin` 永久豁免订阅限制，但仍受功能执行总开关影响，便于紧急下线。

## 功能包

| 功能包 | 权益键 | 主要能力 |
|---|---|---|
| K8s 交付包 | `pack.k8s_delivery` | K8s 在线部署、离线包、installRef、bootstrap、集群清理、制品分发 |
| 节点运维包 | `pack.node_ops` | 系统初始化、时间同步、安全加固、磁盘优化、Shell、文件分发、Linux 服务 |
| 监控包 | `pack.monitoring` | Prometheus 与各类 exporter 安装、配置、下发 |
| 备份与性能包 | `pack.backup_performance` | 备份恢复、性能分析、真实报告生成 |
| AI 技能包 | `skillpack.k8s` 等 | 对应领域 AI 诊断、问答、Runbook |

兼容旧功能键：`feature.k8s_ops`、`feature.service_ops`、`feature.infra_ops`、`feature.advanced` 会映射到对应功能包。

## API

- `GET /api/billing/capabilities`：Web/CLI 共用能力清单。
- `GET /api/billing/packages`：返回可订阅功能包。
- `POST /api/billing/checkout-session`：支持 `pack_key` 或兼容 `package_id`。
- `GET/PUT /api/admin/billing/features`：`super_admin` 管理展示、执行、计费、Stripe Price。
- `POST /api/admin/users/:id/entitlement`：`super_admin` 手动授予 `pack_key` 或兼容功能键。

Paywall 统一返回 HTTP `403`，并包含 `biz=PAYWALL_<pack_key>`、`feature_key`、`pack_key`、`reason` 和 `checkout_available`。

## 执行边界

- Web：菜单和页面显示订阅标签；执行按钮触发后端校验。
- CLI：help 标注收费能力；AI 请求携带 `OPSFLEET_TOKEN` 时按账号识别。
- K8s installRef：生成、bundle 下载、bootstrap 安装均实时校验 `pack.k8s_delivery`。
- Agent：任务创建时写入功能包快照，心跳下发前再次校验订阅状态。

## AI 免费额度

未购买对应 `skillpack.*` 时，每账号或请求来源每天免费 5 次，按 `Asia/Shanghai` 自然日重置。只有服务端 AI 成功返回后才扣次数；参数错误、鉴权失败、Paywall、模型失败不扣次数。
