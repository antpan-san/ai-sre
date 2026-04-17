# OpsFleetPilot Client Agent

部署在用户内网环境中的执行代理，负责与 Server 通信、接收指令、管理受控节点。

## 架构

```
          Server (Control Plane)
                  ↑
         HTTPS 心跳 + 指令下发 (每5秒)
                  ↑
          Client (Agent)
                  ↓
        受控机器 (SSH / Ansible)   ← Phase 2
```

**核心模型：反向心跳**
- Client 每5秒向 Server 发送心跳 + 机器状态
- Server 返回待执行命令列表
- Client 执行命令后回传结果
- Server 不主动连接 Client，支持 NAT/内网环境

## 技术栈

- **语言**: Go 1.24+
- **通信**: HTTPS (支持自定义 CA 证书)
- **日志**: log/slog (结构化日志，支持文件输出)
- **配置**: YAML
- **外部依赖**: gopkg.in/yaml.v3 (仅此一个)

## 项目结构

```
ft-client/
├── main.go                              # 入口：参数解析、依赖组装、生命周期管理
├── conf/
│   ├── client.yaml                      # 配置文件 (Server地址/心跳/日志/TLS)
│   └── client-master-example.yaml      # Master 节点配置示例（含受控节点列表）
├── internal/
│   ├── config/
│   │   └── config.go                    # 配置加载 + 校验 + 默认值
│   ├── logger/
│   │   └── logger.go                    # 日志模块 (基于 log/slog，支持文件+控制台)
│   ├── model/
│   │   └── protocol.go                  # 通信协议定义 (心跳/命令/结果/主机信息)
│   ├── transport/
│   │   └── httpclient.go               # HTTPS 通信层 (ServerAPI 接口 + 实现)
│   ├── collector/
│   │   └── collector.go                 # 主机信息采集 (CPU/内存/磁盘/网络/指纹)
│   ├── nodemanager/
│   │   └── nodemanager.go               # 受控节点管理 (健康检查/状态维护)
│   └── heartbeat/
│       └── heartbeat.go                 # 心跳服务 (定时发送 + 命令调度)
├── go.mod
└── go.sum
```

## 模块说明

| 模块 | 职责 | 关键接口/类型 |
|------|------|---------------|
| `config` | 加载 YAML 配置，校验参数 | `Config`, `Load()`, `Default()` |
| `logger` | 结构化日志，支持级别过滤 | `Init()`, `Debug/Info/Warn/Error()` |
| `model` | 定义 Server 通信协议 | `HeartbeatRequest`, `Command`, `CommandResult` |
| `transport` | HTTPS 通信封装，支持 Token 鉴权 | `ServerAPI` 接口, `HTTPClient` 实现 |
| `collector` | 采集本机资源信息，生成机器指纹 | `Collect() HostInfo`, `GenerateFingerprint()` |
| `nodemanager` | 受控节点管理，定期健康检查 | `Manager`, `AddNode()`, `GetNodes()` |
| `heartbeat` | 心跳循环 + 命令调度 + 节点上报 | `Service.Run(ctx)`, `CommandHandler` 接口 |

## 快速开始

### 编译

```bash
go build -o ft-client -ldflags "-X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" .
```

### 运行

```bash
# 使用默认配置
./ft-client

# 指定配置文件
./ft-client -config /path/to/client.yaml

# 命令行覆盖
./ft-client -server https://10.0.0.1:8080 -id my-client-01

# 查看版本
./ft-client -version
```

### 配置文件

编辑 `conf/client.yaml`:

```yaml
server:
  url: "https://your-server:8080"

client:
  id: ""                    # 留空自动生成
  business_module: "default"

heartbeat:
  interval: 5               # 心跳间隔(秒)

log:
  level: "info"             # debug/info/warn/error
  file: "logs/ft-client.log"

tls:
  enabled: false
  ca_cert: ""
  skip_verify: false         # 开发环境可设为 true
```

## 扩展设计

### CommandHandler 接口

心跳模块通过 `CommandHandler` 接口解耦命令执行：

```go
type CommandHandler interface {
    Execute(ctx context.Context, cmd model.Command) *model.CommandResult
}
```

Phase 2 可实现：
- `ShellHandler` — 执行 Shell 脚本
- `AnsibleHandler` — 调用 ansible-playbook
- `K8sHandler` — Kubernetes 部署流程

### ServerAPI 接口

通信层通过 `ServerAPI` 接口解耦，便于测试和替换：

```go
type ServerAPI interface {
    SendHeartbeat(ctx context.Context, req *model.HeartbeatRequest) (*model.HeartbeatResponse, error)
    ReportResult(ctx context.Context, result *model.CommandResult) error
}
```

## 核心特性

### 1. 主控-受控拓扑支持

Client 支持 `master` 和 `worker` 两种角色：

- **Worker**: 仅上报本机状态
- **Master**: 上报本机状态 + 管理的所有 worker 节点状态

配置示例：

```yaml
client:
  role: "master"  # 或 "worker"

cluster:
  id: "prod-cluster-01"
  name: "生产集群"

managed_nodes:  # master 角色必配
  - ip: "192.168.1.101"
    hostname: "worker-1"
    ssh_port: 22
    ssh_user: "root"
    ssh_key: "/root/.ssh/id_rsa"
```

### 2. 受控节点健康检查与信息采集

Master 节点会定期探测所有 worker 节点的健康状态，并通过 SSH 采集详细系统信息：

**探测方式**：
- **优先**：SSH 密钥认证连接 + 远程命令执行
- **降级**：SSH 失败时自动降级为 TCP 连接测试
- **探测间隔**: 可配置（默认 30 秒）
- **超时时间**: 可配置（默认 5 秒）

**采集信息**：
- **系统版本**：简化的 OS 版本（如 "Ubuntu 22.04", "CentOS 7"）
- **CPU 核数**：逻辑 CPU 数量
- **内存**：总量和使用量（以 GB 为单位）
- **磁盘**：总量和使用量（以 GB 为单位）
- **CPU 使用率**：实时使用百分比

配置示例：

```yaml
node_manager:
  enabled: true
  probe_interval: 30  # 探测间隔（秒）
  probe_timeout: 10   # 超时时间（秒，SSH 采集需要更长时间）

managed_nodes:
  - ip: "192.168.1.101"
    hostname: "worker-1"
    ssh_port: 22
    ssh_user: "root"
    ssh_key: "/root/.ssh/id_rsa"  # SSH 私钥路径
```

### 3. 机器指纹（Fingerprint）

自动生成稳定的机器指纹，用于 Server 端幂等入库：

- **生成方式**: SHA-256(hostname + MAC 地址 + OS/arch)
- **特性**: 同一机器重启后指纹不变，确保唯一性
- **用途**: Server 端根据指纹去重，避免重复注册

### 4. Token 鉴权

支持 Bearer Token 认证，保障心跳接口安全：

```yaml
auth:
  token: "your-secret-token"
```

HTTP 请求自动携带：`Authorization: Bearer <token>`

### 5. 详细日志输出

每次心跳上报时打印详细信息：

```
[INFO] sending heartbeat to server
  client_id=client-xxx
  role=master
  cluster_id=prod-cluster-01
  primary_host_ip=192.168.1.100
  secondary_hosts_count=3
  ...

[INFO] secondary host status index=1
  ip=192.168.1.101
  hostname=worker-1
  status=up
  network_delay_ms=5
  ...
```

## 开发路线

- [x] **Phase 1**: 基础架构 + 心跳 + 日志 + 通信接口
- [x] **Phase 1.5**: 主控-受控拓扑 + 节点管理 + 指纹生成 + Token 鉴权
- [x] **Phase 2**: SSH 远程系统信息采集（OS版本/CPU/内存/磁盘）
- [ ] **Phase 3**: Shell 命令执行 + 任务下发
- [ ] **Phase 4**: Ansible playbook 集成
- [ ] **Phase 5**: 状态采集增强 (gopsutil) + 历史数据
- [ ] **Phase 6**: 自动升级 + 本地缓存

## 通信协议

### 心跳请求 → Server

```
POST /api/v1/heartbeats
```

### Server 返回命令

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

### 结果回传

```
POST /api/v1/task/report
```

```json
{
  "task_id": "uuid",
  "sub_task_id": "uuid",
  "client_id": "client-xxx",
  "status": "success",
  "output": "...",
  "exit_code": 0
}
```
