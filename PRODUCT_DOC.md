# OpsFleetPilot 产品需求文档（PRD v2.0）

> **仓库现状（2026）**：**ft-client** Go Agent 源码已从本仓库移除；下文关于 Agent、目录树与命令行的描述保留为**历史与协议参考**。主路径 Kubernetes 部署以 **离线安装包（ansible-agent + inventory + install.sh）** 为准，见根目录 `README.md`。

> 本文档基于对 ft-backend / ft-front /（历史）ft-client 源代码的 Review 重新整理，
> 真实反映当时实现状态，并指出超出原始文档的能力及尚未完成的部分。

---

# 一、产品概述

## 1.1 产品名称

OpsFleetPilot

## 1.2 产品定位

OpsFleetPilot 是一款企业级服务器运维管理与 Kubernetes 集群部署平台，提供从服务器纳管、系统初始化、K8s 自动部署、监控配置、安全审计的一站式运维解决方案。

核心特点：

- B/S + C/S 混合架构（Web 前端 + 服务端 + 客户端 Agent）
- 反向心跳指令下发模型（Client 主动轮询，Server 不主动连接）
- 内网机器可被远程管理（支持 NAT / 内网环境）
- 运维标准化、流程化
- 支持 Master/Worker 集群拓扑自动识别
- Redis 异步解耦 + WebSocket 实时推送
- Ansible 集成 K8s 自动化部署

---

# 二、系统架构

## 2.1 架构模型

```
                ┌──────────────┐
                │   Web 前端    │  Vue3 + Element Plus
                │  (ft-front)  │
                └───────┬──────┘
                        │ HTTP API / WebSocket
                ┌───────▼──────────────┐
                │      Server 端        │  Golang + Gin
                │  (ft-backend)        │  PostgreSQL + Redis
                │  API + Task 调度      │
                └───────┬──────────────┘
                        │
          (HTTPS 心跳轮询 / 任务下发)
                        │
         ┌──────────────┴──────────────┐
         │                             │
   ┌─────▼──────────┐           ┌──────▼──────────┐
   │  Client Master │           │  Client Worker  │
   │  (ft-client)   │──SSH──────│  (Worker 节点)   │
   │  Ansible 控制  │           │  (无 Agent)      │
   └────────────────┘           └─────────────────┘
```

## 2.2 核心通信模型

### 反向心跳机制（已完整实现）

- Client 每 5 秒（默认）向 Server 发送心跳：
  - 机器唯一指纹（hostname + MAC + OS/arch 的 SHA-256）
  - CPU / 内存 / 磁盘 / 网络实时指标
  - 主机身份（Role: master/worker, ClusterID, ClusterName）
  - 受管节点状态（SecondaryHosts，通过 SSH 采集）
- Server 返回：
  - 待执行任务列表（Commands）
  - Agent 升级通知（Upgrade，协议已定义）
- Client 根据返回字段执行操作，结果通过 POST /api/v1/task/report 回传

### Redis 异步解耦（已实现）

```
心跳到达
   ↓
立即刷新在线 Key (TTL=20s)
   ↓
推送 heartbeat 到 Redis Queue
   ↓ (异步)
HeartbeatConsumer 消费：
  - 持久化心跳记录到 PostgreSQL
  - Upsert 机器信息
  - 刷新 Redis 在线状态 + 指标缓存
  - WebSocket 广播机器状态到前端
```

若 Redis 不可用，自动降级为同步处理（优雅降级）。

### WebSocket 实时推送（已实现）

- 前端通过 `/ws/:user_id` 建立 WebSocket 连接
- 每次心跳处理后广播 `machine_heartbeat` 事件
- 事件携带完整机器状态 + Worker 节点状态

---

# 三、功能模块

## 3.1 服务器纳管（已完整实现，超出原始文档）

**已实现功能：**

- Agent 安装后自动注册（首次心跳即完成注册，无需手动添加）
- 三级身份识别体系（优先级：硬件指纹 > client_id > IP），支持 IP 变更后自动跟踪
- 机器在线/离线自动检测（心跳超过 20s 未收到即标记 offline）
- CPU / 内存 / 磁盘 / 内核版本 / OS 版本实时采集与持久化
- 标签（labels）+ 元数据（metadata）分组管理
- Master/Worker 拓扑自动识别与关联（cluster_id, master_machine_id）
- Worker 节点通过 Master SSH 采集状态（无需 Worker 部署 Agent）
- 软删除支持
- 机器 CRUD（手动录入 / 批量删除）

**数据模型（Machine）：**

| 字段 | 说明 |
|------|------|
| id / tenant_id | 主键 / 租户 |
| name / ip | 主机名 / IP（自动更新） |
| client_id | Agent 身份标识 |
| host_fingerprint | 硬件指纹（SHA-256，最稳定身份） |
| node_role | master / worker / standalone |
| cluster_id | 所属集群 UUID（v5，由字符串确定性生成） |
| master_machine_id | Worker 指向 Master 的自引用 FK |
| status | online / offline |
| last_heartbeat_at | 最后心跳时间 |
| cpu_usage / memory_usage / disk_usage | 实时指标（每次心跳更新） |
| os_version / kernel_version | 系统信息 |
| labels / metadata | 标签与元数据（JSONB） |

---

## 3.2 任务系统（核心模块，已完整实现）

**任务类型（TaskType）：**

| 类型 | 命令 | 说明 |
|------|------|------|
| shell | run_shell | Shell 命令执行 |
| file_distribute | distribute_file | 文件分发 |
| k8s_deploy | install_k8s | Kubernetes 部署（Ansible） |
| sys_init | sys_init | 系统初始化 |
| time_sync | time_sync | 时间同步 |
| install_monitor | install_monitor | 安装监控组件 |
| security_harden | security_harden | 安全加固 |
| disk_optimize | disk_optimize | 磁盘分区优化 |
| register_nodes | sync_nodes | 下发受控节点列表到 Client |

**任务状态机：**

```
pending → dispatched → running → success
                              → failed
                              → timeout
         ↘ cancelled（手动取消）
```

**任务流程：**

```
创建任务（Task）
   ↓
按目标机器拆分为子任务（SubTask）
   ↓
写入 Redis 任务队列 (client_id 粒度)
   ↓
Client 心跳时获取 pending SubTask（Redis 快路径 + DB 慢路径兜底）
   ↓
标记 dispatched，Client 异步执行
   ↓
Client POST /api/v1/task/report 回传结果
   ↓
Server 更新 SubTask 状态，聚合父 Task 进度
   ↓
WebSocket 通知前端
```

**任务幂等保证：**
- SubTask dispatched 后不再重复下发
- Redis 快路径与 DB 慢路径共同维护已分发集合
- MaxRetry 字段控制最大重试次数

---

## 3.3 Kubernetes 部署模块（已实现 Ansible 集成）

**已实现：**

- 多版本选择（K8sVersion 表管理，最新版自动标记推荐）
- Master + Worker 节点拆分部署
- 动态生成 Ansible Inventory（INI 格式）
- 动态生成 group_vars（版本、网络插件、CIDR 等）
- 生成 7 步部署脚本（init → resources → etcd → apiserver → controller-manager → scheduler → kubectl）
- 支持网络插件选择：calico / flannel / cilium
- 集群记录持久化（K8sCluster 表）
- 部署进度 + 步骤日志实时查看
- 集群名称重复检查

**部署流程（Ansible 驱动）：**

```
前端提交部署配置
   ↓
Server 生成 Ansible Inventory + group_vars + 部署脚本
   ↓
创建 Task（k8s_deploy）+ SubTask（目标: Master 节点）
   ↓
Client（Master 节点 Agent）收到 install_k8s 命令
   ↓ （Phase 2 实现: 调用 Ansible 执行脚本）
7 步 Ansible Playbook 顺序执行
   ↓
结果回传，进度实时更新
```

**注意：** Client 侧的实际命令执行（Phase 2）尚未实现，当前 StubHandler 会返回"未实现"。

**可扩展方向（代码已预留接口）：**

- Helm 自动部署
- GPU 节点支持
- 高可用多 Master 部署（HA 模式入口已在前端）

---

## 3.4 系统初始化模块（已完整实现）

所有操作通过任务系统下发，生成 Shell 脚本在目标机器执行：

| 功能 | 接口 | 脚本内容 |
|------|------|---------|
| 内核参数优化 | POST /api/init-tools/system-params | sysctl 批量配置（含输入安全验证） |
| 时间同步 | POST /api/init-tools/time-sync | 安装 chrony，配置 NTP Server，设置时区 |
| 安全加固 | POST /api/init-tools/security-harden | SSH 禁止 root 登录 + 密码策略 + 防火墙 + 禁用高危服务 |
| 磁盘分区优化 | POST /api/init-tools/disk-optimize | 关闭 swap + I/O scheduler 优化 |

---

## 3.5 监控管理（配置存储已实现，部署下发待完善）

**已实现：**

- MonitoringConfig CRUD（支持类型过滤）
- AlertRule CRUD
- 前端支持多种 Exporter 配置页面：
  - Prometheus
  - NodeExporter
  - RedisExporter
  - MongoDBExporter
  - JmxExporter
  - BlackboxExporter

**待完善：**

- 一键安装 Exporter 到目标机器（需通过任务系统下发 `install_monitor` 命令）
- Prometheus 自动 reload/重新配置
- 告警规则实际推送到 AlertManager

---

## 3.6 安全与审计（已完整实现）

**已实现：**

- 操作日志（OperationLog）记录与分页查询（支持按用户、操作类型、资源、状态、时间范围筛选）
- 权限（Permission）CRUD + 批量删除
- 角色-权限关联（RolePermission）管理
- 用户角色字段（User.Role: admin/user）

**注意：** 当前 JWT 中间件仅校验 Token 合法性，RBAC 权限校验未在 API 层强制执行（Permission 表存在但未接入 Gin 中间件）。

---

## 3.7 服务管理

**已实现：**
- 服务（Service）CRUD
- 服务状态变更（start/stop/restart，仅更新 DB 状态）
- Linux Systemd 服务管理接口（当前返回 mock 数据）

**待完善：**
- 服务操作需通过任务系统真实下发 `run_shell` 命令到 Agent 执行
- Linux Systemd 服务查询需通过心跳/任务系统真实采集

---

## 3.8 文件管理（超出原始文档，已实现）

- 文件上传（multipart）
- 文件下载（公开接口，支持 file_id）
- 文件列表 + 详情
- 文件共享（ShareFile）
- 传输历史（Transfer）

---

## 3.9 代理配置（超出原始文档，已实现）

- ProxyConfig CRUD
- 配置应用接口

---

## 3.10 高级功能

| 功能 | 状态 |
|------|------|
| 备份/恢复 | Mock 实现（接口存在，数据为硬编码示例）|
| 性能分析 | 真实实现（PerformanceData 表，统计 CPU/内存/磁盘/网络均值/峰值/谷值）|
| 性能报告生成 | 已实现（按时间范围聚合统计数据并返回）|

---

# 四、技术架构

## 4.1 后端（ft-backend）

| 技术 | 说明 |
|------|------|
| Golang 1.21+ | 主语言 |
| Gin | HTTP 框架 |
| PostgreSQL | 主数据库（GORM，JSONB 字段，UUID 主键，软删除） |
| Redis | 心跳队列 + 任务队列 + 机器在线状态 + 指标缓存 |
| JWT | API 认证（HS256） |
| WebSocket | 机器状态实时推送（gorilla/websocket） |
| UUID v4/v5 | 主键生成 + 集群 ID 确定性生成 |

**目录结构：**

```
ft-backend/
├── common/         # 通用组件（config, logger, redis, response）
├── database/       # PostgreSQL 连接 + 迁移
├── handlers/       # HTTP 处理器（按模块分文件）
├── iotservice/     # 心跳接收 + 异步消费者
├── middleware/     # JWT 认证 + CORS
├── models/         # 数据模型（GORM）
├── routes/         # 路由注册
└── utils/          # JWT/密码/WebSocket 等工具
```

## 4.2 前端（ft-front）

| 技术 | 说明 |
|------|------|
| Vue 3 | 前端框架（Composition API） |
| TypeScript | 主语言 |
| Element Plus | UI 组件库 |
| Pinia | 状态管理 |
| Axios | HTTP 客户端（含拦截器） |
| Vite | 构建工具 |

**页面清单：**

- 登录页
- Dashboard（资源概览）
- 机器管理（列表/详情）
- 任务中心（Job Center）
- K8s 部署（表单 / 进度 / 集群列表）
- 监控管理（Prometheus / NodeExporter / Redis / MongoDB / JMX / Blackbox）
- 代理配置
- 初始化工具（内核参数 / 时间同步 / 安全加固 / 磁盘优化）
- 服务管理（服务部署 / Linux 服务管理）
- 安全审计（操作日志 / 权限管理）
- 用户管理
- 高级功能（备份恢复 / 性能分析）

## 4.3 客户端（ft-client）

| 技术 | 说明 |
|------|------|
| Golang 1.21+ | 主语言 |
| YAML | 配置文件 |
| crypto/ssh | SSH 连接 + 节点健康探测 |
| /proc / syscall | Linux 真实指标采集 |
| SHA-256 | 硬件指纹生成 |

**目录结构：**

```
ft-client/
├── internal/
│   ├── collector/      # 本机指标采集（CPU/内存/磁盘/网络）
│   ├── config/         # 配置加载 + 持久化
│   ├── heartbeat/      # 心跳服务 + 命令分发（StubHandler）
│   ├── logger/         # 结构化日志
│   ├── model/          # 通信协议定义（与 Server 对齐）
│   ├── nodemanager/    # SSH 节点探测 + 指标采集
│   └── transport/      # HTTP 客户端（TLS 支持）
└── main.go
```

---

# 五、通信协议设计

## 5.1 心跳请求（Client → Server）

```json
{
  "client_id": "client-node01-linux-a1b2c3d4",
  "fingerprint": "sha256-hardware-fingerprint",
  "heartbeat_time": 1700000000000,
  "client_version": "1.0.0",
  "process_id": 12345,
  "status": "normal",
  "local_ip": "192.168.1.10",
  "os_info": "linux amd64",
  "role": "master",
  "cluster_id": "prod-cluster-01",
  "cluster_name": "生产集群",
  "primary_host": {
    "ip": "192.168.1.10",
    "hostname": "node01",
    "os_version": "Ubuntu 22.04",
    "kernel_version": "5.15.0-91-generic",
    "cpu_cores": 8,
    "cpu_usage": 23.5,
    "memory_total": 17179869184,
    "memory_used": 8589934592,
    "memory_usage": 50.0,
    "disk_total": 107374182400,
    "disk_used": 32212254720,
    "disk_usage": 30.0,
    "status": "up"
  },
  "secondary_hosts": [
    { "ip": "192.168.1.11", "hostname": "worker01", "status": "up", ... }
  ]
}
```

## 5.2 心跳响应（Server → Client）

```json
{
  "message": "pong",
  "commands": [
    {
      "task_id": "uuid",
      "sub_task_id": "uuid",
      "command": "run_shell",
      "payload": { "script": "systemctl restart nginx" },
      "timeout": 300
    }
  ],
  "upgrade": null
}
```

## 5.3 命令类型枚举

| command | 说明 | Client 实现状态 |
|---------|------|----------------|
| run_shell | 执行 Shell 脚本 | 待实现（Phase 2）|
| sys_init | 系统初始化 | 待实现 |
| time_sync | 时间同步 | 待实现 |
| security_harden | 安全加固 | 待实现 |
| disk_optimize | 磁盘优化 | 待实现 |
| install_k8s | K8s 安装（Ansible） | 待实现 |
| install_monitor | 安装监控 | 待实现 |
| sync_nodes | 同步受控节点列表 | **已实现** |
| run_playbook | 执行 Ansible Playbook | 待实现 |

## 5.4 任务结果回传（Client → Server）

```json
{
  "task_id": "uuid",
  "sub_task_id": "uuid",
  "client_id": "client-node01-linux-a1b2c3d4",
  "status": "success",
  "output": "执行输出...",
  "exit_code": 0,
  "error": ""
}
```

---

# 六、数据库模型

## 核心表

| 表名 | 说明 |
|------|------|
| users | 用户（软删除，含 role 字段）|
| machines | 托管机器（硬件指纹，拓扑字段，实时指标）|
| tasks | 父任务（含优先级、状态机、计数器）|
| sub_tasks | 子任务（机器级别，含 command/payload/output/retry）|
| task_logs | 任务执行日志 |
| heartbeats | 心跳历史记录 |
| k8s_clusters | K8s 集群记录 |
| k8s_versions | K8s 版本管理 |
| monitoring_configs | 监控配置 |
| alert_rules | 告警规则 |
| operation_logs | 操作审计日志 |
| permissions | 权限定义 |
| role_permissions | 角色-权限关联 |
| services | 服务部署记录 |
| files / shares / transfers | 文件管理 |
| proxy_configs | 代理配置 |
| performance_data | 性能指标历史 |
| tenants | 租户（多租户基础已预留）|

---

# 七、Agent 部署说明

## 7.1 配置文件（conf/client.yaml）

```yaml
server:
  url: "https://your-server:8080"

client:
  id: ""                    # 留空则自动生成
  version: "1.0.0"
  role: "master"            # master 或 worker

cluster:
  id: "prod-cluster-01"
  name: "生产集群"

auth:
  token: ""                 # 可选的认证 Token

heartbeat:
  interval: 5               # 心跳间隔（秒）
  max_failures: 3

node_manager:
  enabled: true             # Master 角色时启用
  probe_interval: 30        # SSH 探测间隔（秒）
  probe_timeout: 5

managed_nodes:
  - ip: "192.168.1.11"
    hostname: "worker01"
    ssh_port: 22
    ssh_user: "root"
    auth_type: "key"        # password 或 key
    ssh_key: "/root/.ssh/id_rsa"
```

## 7.2 启动参数

```bash
./ft-client                                         # 默认配置
./ft-client -config /path/to/config.yaml            # 指定配置文件
./ft-client -server https://x.x.x.x:8080           # 指定 Server 地址
./ft-client -role master -cluster-id prod-cluster   # 指定角色
./ft-client -token <auth-token>                     # 指定认证 Token
./ft-client -version                                # 查看版本
```

---

# 八、当前实现状态总览

## ✅ 已完整实现

| 模块 | 说明 |
|------|------|
| 反向心跳通信模型 | Redis 异步解耦 + 自动降级 |
| 机器自动注册与纳管 | 三级身份识别 + Master/Worker 拓扑 |
| 实时指标采集 | /proc 真实数据 + SSH 远程采集 |
| WebSocket 实时推送 | 心跳 → 广播 → 前端更新 |
| 任务系统（完整状态机）| 创建/下发/执行/回传/聚合 |
| 初始化工具 | 4 类 Shell 脚本下发任务 |
| K8s 部署（Ansible 集成）| 动态生成 Inventory + 7 步脚本 |
| 监控配置管理 | 6 种 Exporter 配置 CRUD |
| 安全审计（权限+日志）| 操作日志 + 权限 CRUD + 角色关联 |
| 用户管理 | CRUD + 角色变更 |
| 文件管理 | 上传/下载/共享 |
| 服务管理（CRUD）| 部署记录管理 |
| 性能分析 | 真实数据统计与报告生成 |
| JWT 认证 | Token 颁发 + 验证 |

## ⚠️ 部分实现 / 待完善

| 模块 | 现状 | 缺失部分 |
|------|------|---------|
| Client 命令执行 | StubHandler（仅记录日志）| 所有命令的实际执行逻辑（Phase 2）|
| Linux 服务管理 | 返回硬编码 mock 数据 | 通过任务系统真实采集和下发 |
| 服务 start/stop/restart | 仅更新 DB 状态 | 通过任务系统下发 run_shell 到 Agent |
| Agent 自动升级 | 协议已定义 | Client 侧接收升级通知后的自更新逻辑 |
| RBAC 权限中间件 | 权限表已建 | 在 Gin 路由层按 API 强制权限校验 |
| 监控一键安装 | 配置存储已实现 | 通过 install_monitor 任务下发到 Agent |

## ❌ Mock 实现

| 模块 | 说明 |
|------|------|
| 备份/恢复 | 接口存在但返回硬编码数据，无真实文件备份逻辑 |

---

# 九、后续开发优先级建议

## Phase 2（最优先）：打通 Client 命令执行

```
实现 ShellHandler，替代 StubHandler：
  1. 接收 run_shell 命令，执行 bash 脚本，返回 stdout/stderr/exit_code
  2. 接收 sys_init / time_sync / security_harden / disk_optimize，
     均转化为 run_shell 执行（payload 中已含完整脚本）
  3. 接收 install_k8s，调用本地 Ansible 执行部署脚本
  4. 支持超时控制（Timeout 字段）
  5. 支持执行结果流式回传（可选）
```

## Phase 3：生产加固

1. RBAC 中间件：在 Gin 路由层按 Permission.Code 检查用户权限
2. 操作日志中间件：自动记录 POST/PUT/DELETE 操作到 operation_logs
3. 真实备份/恢复：文件系统 tar + PostgreSQL pg_dump 备份
4. Linux 服务管理：通过 run_shell 任务下发 systemctl 命令
5. 监控一键安装：通过 install_monitor 任务下发安装脚本

## Phase 4：进阶能力

1. Agent 自动升级：Client 收到 Upgrade 通知后下载新版本并替换
2. 告警推送：AlertRule → AlertManager 实际触发
3. 多租户完善：利用已有 tenant_id 字段实现数据隔离
4. 分布式任务调度：多 Server 节点时的任务去重与协调

---

# 十、扩展方向（超出当前版本）

- 灰度发布支持
- 私有镜像仓库集成
- GPU 节点 K8s 部署
- Helm Chart 自动化部署
- 多 Server 集群（分布式调度）
- Webhook 告警通知

---

# 附录：API 路由速查

## 公开接口（无 JWT）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/auth/login | 登录获取 Token |
| POST | /api/auth/logout | 登出 |
| GET  | /api/files/download/:file_id | 文件下载 |
| POST | /api/v1/heartbeats | Agent 心跳上报 |
| POST | /api/v1/task/report | Agent 任务结果回传 |
| GET  | /api/v1/tasks/running | 查询运行中任务 |
| GET  | /health | 健康检查 |
| GET  | /ws/:user_id | WebSocket 连接 |

## 认证接口（需 JWT）

| 分组 | 路径前缀 | 主要操作 |
|------|---------|---------|
| 用户 | /api/user | CRUD + 角色变更 |
| 机器 | /api/machine | CRUD + 状态更新 + Worker 注册 |
| 任务 | /api/task | CRUD + 取消 + 日志 |
| Job Center | /api/job | 批量执行 + 结果查询 |
| 服务 | /api/service | 部署/操作/Linux 服务管理 |
| K8s | /api/k8s | 版本/机器/集群/部署/进度/日志 |
| 代理 | /api/proxy/config | CRUD + 应用 |
| 初始化工具 | /api/init-tools | 内核参数/时间同步/安全加固/磁盘优化 |
| 监控 | /api/monitoring | 配置 CRUD + 告警规则 CRUD |
| 安全审计 | /api/security-audit | 操作日志 + 权限 + 角色权限 |
| 文件 | /api/files | 上传/列表/详情/删除/共享 |
| 传输 | /api/transfers | 传输历史 |
| 高级 | /api/advanced | 备份/恢复/性能分析 |
| Dashboard | /api/dashboard/data | 概览数据 |
