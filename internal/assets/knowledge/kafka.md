# Kafka 极简快诊要点

Kafka 诊断优先级遵循“先集群可用性，再副本健康，再消费堆积”的顺序：

1. P0：offline partition、leader 为 -1/none。此时分区不可读写，应先恢复 broker、磁盘或副本 leader。
2. P1：under replicated partition、ISR 收缩。通常是 broker 磁盘/网络/GC 抖动或副本同步落后，会导致生产延迟和高可用风险。
3. P1/P2：consumer lag 高。要看总 lag、最大 lag 分区和 active member 数；单分区占比很高时优先怀疑热点 key 或分区分配不均。
4. P2：group 有 lag 但无 active member。通常是消费者进程未运行、认证失败、连接失败或部署副本为 0。
5. P2：group 长时间处于 PreparingRebalance / CompletingRebalance。常见原因是 consumer 重启、心跳超时、max.poll.interval.ms 小于业务处理耗时。

最快验证命令：

- `kafka-consumer-groups.sh --bootstrap-server <bs> --describe --group <group>`：查看 total lag、最大 lag 分区、consumer 成员。
- `kafka-consumer-groups.sh --bootstrap-server <bs> --describe --state --group <group>`：查看 group 是否稳定。
- `kafka-topics.sh --bootstrap-server <bs> --describe --topic <topic>`：查看 leader、replicas、isr。
- `kafka-broker-api-versions.sh --bootstrap-server <bs>`：快速验证 broker 连接、认证和协议兼容。

处理建议保持短路径：offline partition 先恢复 leader；URP 先看 broker 磁盘、网络和副本同步；lag 高先扩容 consumer 或定位热点分区；无 active member 先检查消费者服务状态和认证日志；rebalance 先检查实例重启、处理耗时和 `max.poll.interval.ms`。
