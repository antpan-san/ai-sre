# Nginx 5xx 与上游问题

HTTP 502 通常表示网关从上游收到无效响应或连接被拒绝；503 多与上游不可用、过载或主动维护有关；504 多为网关到上游读超时或上游处理过慢。

排查应同时看 nginx error log 中的 upstream timed out、connect() failed、no live upstreams 等信息，以及上游服务的连接数、线程池、队列与错误率。

配置层面关注 `proxy_connect_timeout`、`proxy_read_timeout`、`proxy_next_upstream`、keepalive 连接池与 upstream 健康检查。
