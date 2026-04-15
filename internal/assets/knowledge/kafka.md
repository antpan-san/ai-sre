# Kafka 消费 Lag 排查要点

Kafka 消费堆积通常表现为 consumer lag 持续增长。首先要区分是全局堆积还是少数分区尖刺：单分区尖刺多见于热点 key 或分区不均衡。

常见根因包括：消费端处理能力不足（线程/分区分配少）、消费逻辑变慢、重平衡频繁、broker 侧磁盘或网络瓶颈、ISR 收缩导致生产/消费抖动。

验证时可对比 messages in per second 与消费速率，检查 consumer group 的 partition 分配是否均衡，并关注 `records-lag-max` 与 `records-lag` 指标。

调优方向包括：增加分区并行度、优化消费代码、调整 `max.poll.interval.ms` 与 `session.timeout.ms`、避免过长 GC 或阻塞；broker 侧关注磁盘 IO、网络带宽与副本同步。
