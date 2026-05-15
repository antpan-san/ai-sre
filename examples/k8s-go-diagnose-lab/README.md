# K8s Go 应用 Pod 异常 — ai-sre diagnose 实验室

在 **192.168.56.101** 上逐个注入异常 Pod，运行 `ai-sre diagnose`，验证后删除。

## 前置

- 集群已部署 `memleak-demo:lab`（见 `examples/memleak-demo/deploy-to-k8s-lab.sh`）
- `ai-sre` 已绑定 OpsFleet token（`~/.config/ai-sre/opsfleet_*`）
- worker 节点需能使用 `docker.io/library/memleak-demo:lab`（脚本会 `docker save | ctr import` 同步到 192.168.56.102）

## 运行

```bash
SSH_TARGET=root@192.168.56.101 ./scripts/k8s-go-diagnose-lab-matrix.sh
```

结果目录：`data/k8s-go-diagnose-lab-latest/`（`summary.tsv` + 各 case 的 `*.txt`）。

## 用例（12）

| ID | 异常类型 |
|----|----------|
| 01 | ImagePullBackOff（错误 tag） |
| 02 | CrashLoop（命令/二进制不存在） |
| 03 | OOMKilled（Go 泄漏 + 低 limit） |
| 04 | Pending（CPU 请求过大） |
| 05 | Liveness 探针端口错误 |
| 06 | Readiness 探针端口错误 |
| 07 | 启动命令不存在 |
| 08 | Init 容器失败 |
| 09 | imagePullPolicy Never + 无本地镜像 |
| 10 | Running 对照（可选 proc 采集） |
| 11 | 非法仓库 ErrImagePull |
| 12 | `--deployment` 级 OOM 诊断 |

## 技能包

- 客户端：`internal/assets/skills/go_runtime_golang_pod.yaml`
- 服务端：`ft-backend/skills/builtin/go_runtime_golang_pod.yaml`
