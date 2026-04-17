# ft-backend

OpsFleetPilot 后端服务 — 企业级服务器运维管理与 Kubernetes 集群部署平台。

基于 **Go + Gin + GORM + PostgreSQL** 构建，采用反向心跳 + 指令下发模型，支持内网 NAT 环境下的远程服务器管理。

## 技术栈

- **语言**: Go 1.24+
- **Web 框架**: Gin 1.11
- **ORM**: GORM 1.31
- **数据库**: PostgreSQL 14+
- **配置管理**: YAML
- **认证**: JWT
- **实时通信**: WebSocket

## 核心架构

```
┌──────────────┐
│   Web 前端    │  (Vue3 + Element Plus)
└───────┬──────┘
        │
┌───────▼──────┐
│   Server端    │  (Gin API + 任务调度)
│ API + Task调度 │
└───────┬──────┘
        │
   (HTTPS 心跳/指令下发)
        │
  ┌─────▼─────┐
  │  Client    │  (Go 二进制 Agent)
  └───────────┘
```

### 反向心跳模型
- Client 定时向 Server 发送心跳 + 机器状态
- Server 返回待执行的任务/命令列表
- Client 执行后回传结果
- Server 不主动连接 Client，支持 NAT/内网环境

## 功能特性

### 核心功能
- **服务器纳管**: Agent 自动注册、机器信息采集、标签分组管理
- **任务系统**: Shell 执行、文件分发、系统初始化、任务状态机
- **K8s 部署**: 多版本选择、多节点部署、部署进度跟踪
- **监控管理**: Prometheus/Node Exporter 等配置管理、告警规则
- **安全审计**: RBAC 权限管理、操作日志记录、敏感操作审计

### 高级功能
- **初始化工具**: 系统参数优化、时间同步、安全加固、磁盘优化
- **作业中心**: 远程 Shell 命令批量执行
- **实时通信**: WebSocket 状态推送
- **多租户就绪**: 所有核心表支持 tenant_id

## 快速开始

### 环境要求
- Go >= 1.24.0
- PostgreSQL >= 14.0

### 安装依赖
```bash
go mod download
```

### 配置数据库
修改 `conf/config.yaml`:

```yaml
database:
  host: localhost
  port: "5432"
  user: postgres
  password: postgres
  dbname: opsfleetpilot
  sslmode: disable
  timezone: Asia/Shanghai
```

### 初始化数据库
```bash
# 使用完整迁移脚本（包含分区表、触发器、种子数据）
psql -d opsfleetpilot -f database/migration_pg.sql
```

### 编译和运行
```bash
# 编译
go build -o ft-backend

# 运行
./ft-backend
```

服务将在 `http://localhost:8080` 启动。

## 项目结构

```
ft-backend/
├── main.go                    # 入口文件
├── conf/
│   └── config.yaml            # 主配置文件 (PostgreSQL)
├── common/
│   ├── config/
│   │   └── config.go          # 配置加载
│   └── logger/
│       └── logger.go          # 日志
├── database/
│   ├── database.go            # PostgreSQL 连接、迁移、种子数据
│   ├── migration_pg.sql       # 完整 PostgreSQL Schema
│   └── init_k8s_versions.sql  # K8s 版本种子数据
├── handlers/
│   ├── auth.go                # 认证 (登录/注册/刷新)
│   ├── user.go                # 用户管理
│   ├── machine.go             # 机器管理
│   ├── task.go                # 任务系统 (创建/分发/状态机/日志)
│   ├── job.go                 # 作业中心 (Shell 批量执行)
│   ├── init_tools.go          # 初始化工具 (系统优化/时间同步/安全加固)
│   ├── monitoring.go          # 监控配置管理 + 告警规则
│   ├── k8s_deploy.go          # K8s 部署 (提交/进度/日志)
│   ├── file.go                # 文件管理
│   ├── dashboard.go           # 仪表盘
│   ├── security_audit.go      # 安全审计
│   ├── advanced.go            # 高级功能 (备份/性能)
│   ├── health.go              # 健康检查
│   └── websocket.go           # WebSocket
├── iotservice/
│   └── heartbeat.go           # 反向心跳 (接收状态 + 下发命令)
├── middleware/
│   ├── auth.go                # JWT 认证中间件
│   └── cors.go                # CORS 中间件
├── models/
│   ├── base.go                # 基础模型 (UUID, JSONB, 软删除)
│   ├── task.go                # 任务/子任务/任务日志/命令协议
│   ├── role.go                # 角色
│   ├── monitoring_config.go   # 监控配置/告警规则
│   ├── machine.go             # 机器
│   ├── user.go                # 用户
│   ├── heartbeat.go           # 心跳 (分区表)
│   ├── k8s_cluster.go         # K8s 集群
│   ├── k8s_version.go         # K8s 版本
│   └── ...                    # 其他模型
├── routes/
│   └── router.go              # 路由注册
└── utils/
    ├── jwt.go                 # JWT 工具
    ├── password.go            # 密码工具
    ├── websocket_manager.go   # WebSocket 管理器
    └── monitoring.go          # 机器状态监控器
```

## API 文档

### 公开接口
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /health | 健康检查 |
| POST | /api/auth/login | 用户登录 |
| POST | /api/v1/heartbeats | Client 心跳 (返回待执行命令) |
| POST | /api/v1/task/report | Client 任务结果回传 |

### 任务系统
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/task | 创建任务 |
| GET | /api/task | 任务列表 |
| GET | /api/task/:id | 任务详情 + 子任务 |
| POST | /api/task/:id/cancel | 取消任务 |
| GET | /api/task/:id/logs | 任务执行日志 |

### 作业中心
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/job/machines | 获取在线机器列表 |
| POST | /api/job/execute | 执行 Shell 命令 |
| GET | /api/job/result/:jobId | 获取执行结果 |

### K8s 部署
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/k8s/deploy/versions | K8s 版本列表 |
| POST | /api/k8s/deploy/submit | 提交部署配置 |
| GET | /api/k8s/deploy/progress | 部署进度 |
| GET | /api/k8s/deploy/logs | 部署日志 |

### 初始化工具
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/init-tools/system-params | 获取系统参数配置 |
| POST | /api/init-tools/system-params | 应用系统参数优化 |
| POST | /api/init-tools/time-sync | 时间同步 |
| POST | /api/init-tools/security-harden | 安全加固 |
| POST | /api/init-tools/disk-optimize | 磁盘优化 |

### 监控管理
| 方法 | 路径 | 说明 |
|------|------|------|
| GET/POST | /api/monitoring/configs | 监控配置 CRUD |
| GET/POST | /api/monitoring/alert-rules | 告警规则 CRUD |

### 命令协议

心跳响应中返回的命令格式:
```json
{
  "message": "pong",
  "commands": [
    {
      "task_id": "uuid",
      "sub_task_id": "uuid",
      "command": "run_shell",
      "payload": { "script": "systemctl restart docker" },
      "timeout": 300
    }
  ]
}
```

Client 执行后回传:
```json
{
  "task_id": "uuid",
  "sub_task_id": "uuid",
  "client_id": "client-001",
  "status": "success",
  "output": "...",
  "exit_code": 0
}
```

## 许可证

MIT License
