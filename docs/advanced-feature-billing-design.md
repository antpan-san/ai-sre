# 高级功能与订阅计费（功能分级 × 多功能包）

## 目标

控制台能力按 **`feature_*` 功能键分级**计费；Stripe 可多档 Price（多功能包）。OpsFleetPilot 仅存订阅行与 entitlement，网关层、`RequireEntitlementOrSuperAdmin` 做校验。

### 内置功能键

| Key | API / 业务能力 |
|-----|----------------|
| `feature.k8s_ops` | Kubernetes 部署（submit/terminate、mirror catalog、clusters、bundle、bundle-invite、relay）、及部署进度与日志读取 |
| `feature.service_ops` | 控制台服务部署、Linux 服务、service-deploy CRUD |
| `feature.infra_ops` | 代理配置、监控告警、初始化工具 |
| `feature.advanced` | 备份与恢复、性能分析、报告 |

未列出的管理能力（仪表盘、机器、作业中心、审计、执行记录列表等）在 **admin** 仍可用（仅受角色限制），不参与上述四门 paywall。


## 角色与权益（摘要）

| 角色 | `super_admin` | `admin` / `user`（计费启用时） |
|------|----------------|--------------------------------|
| 管理端路由 | super_admin：全豁免 | `admin` / `user`：均可进入 **`/admin/...`** 浏览与操作（以 JWT + 计费权益 + `RequireAdmin()` 危险行为为准）；`/app` 为兼容入口 |
| 四门 paywall | 永久绕过 | **非 super_admin** 均须具备对应 `entitlement`，或该功能计费未开启 |

`super_admin` 与 `billing_exempt`、`GET /billing/me` 中的 `feature_access`／`billing_exempt` 一致。

Stripe 成功后 Webhook **按当前订阅的第一个 Price** 映射到 `billing.packages[*].stripe_price_id`，为用户写入 `source=stripe` 的一组 entitlement；切换到其他档位会 **删除多余的 stripe 权益**（仅 Stripe 写入的，`manual` 不动）。


## Stripe 多功能包配置

配置 `ft-backend/conf/config.yaml` 的 **`billing.packages`**：

```yaml
billing:
  stripe_secret_key: sk_live_...
  stripe_webhook_secret: whsec_...
  public_app_base_url: "https://console.example.com"
  packages:
    - id: k8s_only
      display_name: "K8s 交付"
      stripe_price_id: price_xxx
      feature_keys:
        - feature.k8s_ops

    - id: ops_bundle
      display_name: "交付全套（无高级）"
      stripe_price_id: price_yyy
      feature_keys:
        - feature.k8s_ops
        - feature.service_ops
        - feature.infra_ops

    - id: full
      display_name: "全域（含高级）"
      stripe_price_id: price_zzz
      feature_keys:
        - feature.k8s_ops
        - feature.service_ops
        - feature.infra_ops
        - feature.advanced
```

- **兼容性**：若 `packages` **为空**，且配置了 **`stripe_price_id_pro`**，则运行时等价单包：`pro_legacy`，仅授予 `feature.advanced`（与最早的单包 Stripe 集成一致）。
- Checkout：`POST /api/billing/checkout-session` Body `{"package_id":"full"}`；缺省选中 **解析后的第一个档位**。


## Stripe 同步流程（摘要）

1. 用户对某 paywall：`403` / `biz=PAYWALL_<feature_key>`。
2. 前端可先 `GET /api/billing/packages` 列出档位，`POST /api/billing/checkout-session` 拉起 Checkout。
3. Webhook：`checkout.session.completed` / `customer.subscription.*` → 写入 `subscriptions` + 对齐 `source=stripe` 的 entitlement 集合。


## API 纪要

| 路由 | 说明 |
|------|------|
| `GET /api/billing/me` | 订阅、原始 entitlements、`feature_access`/`can_use_*`、`feature_flags` |
| `GET /api/billing/packages` | 档位列表（id、名称、所含 feature_keys） |
| `POST /api/billing/checkout-session` | `package_id` 可选 |
| `GET|PUT /api/admin/billing/features` | super_admin：`FeatureBillingSetting` 行 |
| `POST /api/admin/users/:id/entitlement` | super_admin：`source=manual` 单行授权 |

`feature_*`/`package_id`/Price 映射均需 **服务端可信配置**（禁止客户端直接写任意 entitlement）。


## 降级与失败

- `FeatureBillingSetting.billing_enabled=false`：该键 **不交 paywall**，历史兼容。
- Stripe 密钥或 `packages`/兼容价缺失：Checkout `503`，仍可用手动授权。
- Webhook **签名校验失败不得写库**。
- Stripe 档位无法解析：清空该用户 **全部** `source=stripe` entitlement，日志告警。
