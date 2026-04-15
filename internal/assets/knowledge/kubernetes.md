# Kubernetes Pod Pending 与调度

Pod 处于 Pending 多数是调度器未将 Pod 绑定到节点。典型原因：资源请求超过可调度容量、节点污点与 Pod 容忍不匹配、亲和性规则过严、PVC 未绑定、镜像拉取策略或密钥问题在部分集群也会表现为长时间 Pending（需结合 Events）。

排查优先查看 `kubectl describe pod` 的 Events 段落，其次查看 ResourceQuota、LimitRange、节点资源分配与 `kubectl get nodes -o wide`。

# Kubernetes CrashLoopBackOff

CrashLoopBackOff 表示容器反复退出。应查看 `kubectl logs` 与 `kubectl logs --previous`，并关注退出码、OOMKilled、探针失败与启动命令错误。

常见类别：应用启动失败、配置错误、依赖不可用、资源过小导致 OOM、探针 initialDelaySeconds 过短。
