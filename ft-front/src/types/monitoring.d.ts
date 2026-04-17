// Monitoring Type Definitions

// 监控配置基础信息
export interface MonitoringConfig {
  id: string;
  name: string;
  description?: string;
  enabled: boolean;
  createTime: string;
  updateTime: string;
}

// Prometheus 主配置
export interface PrometheusConfig extends MonitoringConfig {
  type: 'prometheus';
  global: {
    scrapeInterval: string;
    evaluationInterval: string;
    scrapeTimeout: string;
  };
  scrapeConfigs: ScrapeConfig[];
  alerting: {
    alertmanagers: AlertmanagerConfig[];
  };
  ruleFiles: string[];
}

// 抓取配置
export interface ScrapeConfig {
  jobName: string;
  scrapeInterval?: string;
  scrapeTimeout?: string;
  metricsPath?: string;
  scheme?: 'http' | 'https';
  staticConfigs?: StaticConfig[];
  fileSDConfigs?: FileSDConfig[];
  relabelConfigs?: RelabelConfig[];
}

// 静态配置
export interface StaticConfig {
  targets: string[];
  labels?: Record<string, string>;
}

// 文件服务发现配置
export interface FileSDConfig {
  files: string[];
  refreshInterval?: string;
}

// 重新标记配置
export interface RelabelConfig {
  sourceLabels?: string[];
  separator?: string;
  regex?: string;
  targetLabel?: string;
  replacement?: string;
  action?: 'replace' | 'keep' | 'drop' | 'hashmod' | 'labelmap' | 'labeldrop' | 'labelkeep';
}

// Alertmanager 配置
export interface AlertmanagerConfig {
  staticConfigs: StaticConfig[];
  scheme?: 'http' | 'https';
  timeout?: string;
}

// Node Exporter 配置
export interface NodeExporterConfig extends MonitoringConfig {
  type: 'node-exporter';
  port: number;
  path: string;
  collectors: string[];
  noCollectors: string[];
}

// JMX Exporter 配置
export interface JmxExporterConfig extends MonitoringConfig {
  type: 'jmx-exporter';
  port: number;
  configFile: string;
  jvmArgs?: string;
}

// Redis Exporter 配置
export interface RedisExporterConfig extends MonitoringConfig {
  type: 'redis-exporter';
  port: number;
  redisAddr: string;
  redisPassword?: string;
  redisUser?: string;
}

// MongoDB Exporter 配置
export interface MongoExporterConfig extends MonitoringConfig {
  type: 'mongodb-exporter';
  port: number;
  mongodbUri: string;
  collectDatabase?: boolean;
  collectCollection?: boolean;
}

// Blackbox Exporter 配置
export interface BlackboxExporterConfig extends MonitoringConfig {
  type: 'blackbox-exporter';
  port: number;
  configFile: string;
}

// 告警规则
export interface AlertRule {
  id: string;
  name: string;
  expr: string;
  for: string;
  labels: Record<string, string>;
  annotations: Record<string, string>;
  enabled: boolean;
}

// 告警规则组
export interface AlertRuleGroup {
  name: string;
  rules: AlertRule[];
}

// 监控配置类型联合
export type ExporterConfig = 
  | PrometheusConfig
  | NodeExporterConfig
  | JmxExporterConfig
  | RedisExporterConfig
  | MongoExporterConfig
  | BlackboxExporterConfig;

// 获取监控配置列表响应
export interface GetMonitoringConfigListResponse {
  code: number;
  data: {
    list: ExporterConfig[];
    total: number;
  };
  msg: string;
}

// 获取单个监控配置响应
export interface GetMonitoringConfigResponse {
  code: number;
  data: ExporterConfig;
  msg: string;
}

// 创建/更新监控配置请求
export type CreateMonitoringConfigRequest = Omit<ExporterConfig, 'id' | 'createTime' | 'updateTime'>;

// 创建/更新监控配置响应
export interface CreateMonitoringConfigResponse {
  code: number;
  data: ExporterConfig;
  msg: string;
}

// 删除监控配置响应
export interface DeleteMonitoringConfigResponse {
  code: number;
  data: null;
  msg: string;
}

// 获取告警规则列表响应
export interface GetAlertRulesResponse {
  code: number;
  data: AlertRuleGroup[];
  msg: string;
}

// 创建/更新告警规则响应
export interface SaveAlertRuleResponse {
  code: number;
  data: AlertRule;
  msg: string;
}

// 删除告警规则响应
export interface DeleteAlertRuleResponse {
  code: number;
  data: null;
  msg: string;
}
