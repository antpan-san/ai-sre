# Redis 延迟与慢查询

Redis 是单线程执行命令，慢命令、大 key、持久化（fork、AOF fsync）、内存淘汰或网络抖动都会拉高延迟。

优先查看 SLOWLOG、LATENCY DOCTOR、INFO 中的 instantaneous_ops_per_sec、connected_clients、used_memory 与复制积压。

热点 key 可通过采样、客户端缓存或拆分 key 缓解；大 key 需要业务拆分或异步迁移。主从架构下复制延迟与持久化配置密切相关。
