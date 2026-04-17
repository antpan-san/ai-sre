// Proxy Configuration Type Definitions

// 全局配置
export interface GlobalConfig {
  worker_processes?: string; // 工作进程数量
  worker_connections?: number; // 每个工作进程的连接数
  error_log?: string; // 错误日志路径
  pid?: string; // PID文件路径
}

// Events配置
export interface EventsConfig {
  worker_connections?: number; // 每个工作进程的连接数
  use?: string; // 使用的事件模型
}

// HTTP配置
export interface HttpConfig {
  include?: string[]; // 包含的配置文件
  default_type?: string; // 默认MIME类型
  log_format?: Record<string, string>; // 日志格式
  access_log?: string; // 访问日志路径
  sendfile?: boolean; // 是否使用sendfile
  tcp_nopush?: boolean; // 是否启用TCP NOPUSH
  tcp_nodelay?: boolean; // 是否启用TCP NODELAY
  keepalive_timeout?: number; // 长连接超时时间
  gzip?: boolean; // 是否启用gzip压缩
  gzip_comp_level?: number; // gzip压缩级别
  gzip_types?: string[]; // gzip压缩的MIME类型
  server_tokens?: boolean; // 是否显示服务器版本
}

// 位置配置
export interface LocationConfig {
  path: string; // 路径
  root?: string; // 根目录
  index?: string[]; // 索引文件
  proxy_pass?: string; // 代理转发地址
  proxy_set_header?: Record<string, string>; // 代理请求头设置
  rewrite?: string[]; // 重写规则
  try_files?: string[]; // 尝试文件
  alias?: string; // 别名
  allow?: string[]; // 允许的IP
  deny?: string[]; // 拒绝的IP
  auth_basic?: string; // 基本认证
  auth_basic_user_file?: string; // 密码文件
  client_max_body_size?: string; // 客户端最大请求体大小
  proxy_read_timeout?: number; // 代理读取超时时间
  proxy_connect_timeout?: number; // 代理连接超时时间
}

// 服务器配置
export interface ServerConfig {
  id?: string;
  listen?: number; // 监听端口
  server_name?: string[]; // 服务器名称
  root?: string; // 根目录
  index?: string[]; // 索引文件
  location?: LocationConfig[]; // 位置配置
  error_page?: Record<string, string>; // 错误页面
  access_log?: string; // 访问日志路径
  ssl_certificate?: string; // SSL证书路径
  ssl_certificate_key?: string; // SSL密钥路径
  ssl_protocols?: string[]; // SSL协议
  ssl_ciphers?: string; // SSL密码套件
  ssl_prefer_server_ciphers?: boolean; // 是否优先使用服务器密码套件
  ssl_session_timeout?: string; // SSL会话超时时间
}

// 上游配置
export interface UpstreamConfig {
  name: string; // 上游名称
  server?: string[]; // 服务器列表
  least_conn?: boolean; // 是否使用最少连接算法
  ip_hash?: boolean; // 是否使用IP哈希算法
  keepalive?: number; // 长连接数
}

// 完整的代理配置
export interface ProxyConfig {
  id?: string;
  name: string; // 配置名称
  description?: string; // 配置描述
  global?: GlobalConfig; // 全局配置
  events?: EventsConfig; // Events配置
  http?: HttpConfig; // HTTP配置
  upstream?: UpstreamConfig[]; // 上游配置
  server?: ServerConfig[]; // 服务器配置
  created_at?: string; // 创建时间
  updated_at?: string; // 更新时间
  status?: 'active' | 'inactive' | 'draft'; // 配置状态
}

// 代理配置列表请求参数
export interface GetProxyConfigListParams {
  page?: number;
  pageSize?: number;
  name?: string;
  status?: 'active' | 'inactive' | 'draft';
}

// 代理配置列表响应
export interface GetProxyConfigListResponse {
  code: number;
  data: {
    list: ProxyConfig[];
    total: number;
  };
  msg: string;
}

// 保存代理配置请求参数
export interface SaveProxyConfigParams {
  id?: string;
  name: string;
  description?: string;
  config: {
    global?: GlobalConfig;
    events?: EventsConfig;
    http?: HttpConfig;
    upstream?: UpstreamConfig[];
    server?: ServerConfig[];
  };
  status?: 'active' | 'inactive' | 'draft';
}

// 保存代理配置响应
export interface SaveProxyConfigResponse {
  code: number;
  data: { id: string };
  msg: string;
}

// 获取代理配置详情请求参数
export interface GetProxyConfigDetailParams {
  id: string;
}

// 获取代理配置详情响应
export interface GetProxyConfigDetailResponse {
  code: number;
  data: ProxyConfig;
  msg: string;
}

// 删除代理配置请求参数
export interface DeleteProxyConfigParams {
  id: string;
}

// 删除代理配置响应
export interface DeleteProxyConfigResponse {
  code: number;
  data: null;
  msg: string;
}

// 应用代理配置请求参数
export interface ApplyProxyConfigParams {
  id: string;
}

// 应用代理配置响应
export interface ApplyProxyConfigResponse {
  code: number;
  data: { success: boolean };
  msg: string;
}
