package services

import (
	"strings"

	"ft-backend/database"
	"ft-backend/models"
)

const (
	SkillNodeTypeCategory   = "category"
	SkillNodeTypeCapability = "capability"
	SkillNodeTypeSkill      = "skill"

	ExecutionModeLocalReadonly      = "local_readonly"
	ExecutionModeServerAI           = "server_ai"
	ExecutionModeServerPlanReadonly = "server_plan_readonly"
	ExecutionModeServerJobWrite     = "server_job_write"
	ExecutionModeDownloadOrInstall  = "download_or_install"
	ExecutionModeLocalAIFallback    = "local_ai_fallback"
)

// SkillExecutionIntent is the compact coordinate system shared by CLI and server.
// It intentionally contains no YAML, prompt, or entitlement detail.
type SkillExecutionIntent struct {
	CommandKind       string `json:"command_kind,omitempty"`
	Action            string `json:"action,omitempty"`
	Topic             string `json:"topic,omitempty"`
	CandidateNodePath string `json:"candidate_node_path,omitempty"`
	NodePath          string `json:"node_path,omitempty"`
	SkillKey          string `json:"skill_key,omitempty"`
	ProblemKey        string `json:"problem_key,omitempty"`
	CapabilityKey     string `json:"capability_key,omitempty"`
	PackKey           string `json:"pack_key,omitempty"`
	ExecutionMode     string `json:"execution_mode,omitempty"`
}

type SkillTreeNode struct {
	Path          string               `json:"path"`
	ParentPath    string               `json:"parent_path,omitempty"`
	NodeType      string               `json:"node_type"`
	Title         string               `json:"title"`
	Description   string               `json:"description,omitempty"`
	Topic         string               `json:"topic,omitempty"`
	SkillKey      string               `json:"skill_key,omitempty"`
	ProblemKey    string               `json:"problem_key,omitempty"`
	CapabilityKey string               `json:"capability_key,omitempty"`
	PackKey       string               `json:"pack_key,omitempty"`
	FeatureKey    string               `json:"feature_key,omitempty"`
	ExecutionMode string               `json:"execution_mode,omitempty"`
	CLIVisible    bool                 `json:"cli_visible"`
	Status        string               `json:"status,omitempty"` // active | disabled
	SortOrder     int                  `json:"sort_order"`
	AssetStats    *SkillTreeAssetStats `json:"asset_stats,omitempty"`
}

type SkillTreeAssetStats struct {
	Total      int64 `json:"total"`
	Draft      int64 `json:"draft"`
	Review     int64 `json:"review"`
	Approved   int64 `json:"approved"`
	Deprecated int64 `json:"deprecated"`
}

var builtinSkillTreeNodes = []SkillTreeNode{
	{Path: "ops", NodeType: SkillNodeTypeCategory, Title: "运维能力", CLIVisible: false, SortOrder: 1},
	{Path: "ops.delivery_implementation", ParentPath: "ops", NodeType: SkillNodeTypeCategory, Title: "部署实施", CLIVisible: true, SortOrder: 10},
	{Path: "ops.incident_diagnosis", ParentPath: "ops", NodeType: SkillNodeTypeCategory, Title: "问题排查", CLIVisible: true, SortOrder: 20},
	{Path: "ops.incident_diagnosis.kubernetes", ParentPath: "ops.incident_diagnosis", NodeType: SkillNodeTypeCategory, Title: "Kubernetes", Topic: "k8s", PackKey: models.SkillPackK8s, FeatureKey: models.FeatureKeyAIDiagnosis, CLIVisible: true, SortOrder: 30},
	{Path: "ops.incident_diagnosis.linux", ParentPath: "ops.incident_diagnosis", NodeType: SkillNodeTypeCategory, Title: "Linux 系统", CLIVisible: true, SortOrder: 34},
	{Path: "ops.incident_diagnosis.network", ParentPath: "ops.incident_diagnosis", NodeType: SkillNodeTypeCategory, Title: "网络与域名", CLIVisible: true, SortOrder: 35},
	{Path: "ops.incident_diagnosis.middleware", ParentPath: "ops.incident_diagnosis", NodeType: SkillNodeTypeCategory, Title: "中间件", CLIVisible: true, SortOrder: 40},
	{Path: "ops.incident_diagnosis.middleware.kafka", ParentPath: "ops.incident_diagnosis.middleware", NodeType: SkillNodeTypeCapability, Title: "Kafka 诊断", Topic: "kafka", CapabilityKey: "cap.diagnosis.kafka", PackKey: models.SkillPackKafka, FeatureKey: models.FeatureKeyAIDiagnosis, ExecutionMode: ExecutionModeServerAI, CLIVisible: true, SortOrder: 41},
	{Path: "ops.incident_diagnosis.middleware.redis", ParentPath: "ops.incident_diagnosis.middleware", NodeType: SkillNodeTypeCapability, Title: "Redis 诊断", Topic: "redis", CapabilityKey: "cap.diagnosis.redis", PackKey: models.SkillPackRedis, FeatureKey: models.FeatureKeyAIDiagnosis, ExecutionMode: ExecutionModeServerAI, CLIVisible: true, SortOrder: 42},
	{Path: "ops.incident_diagnosis.network.domain", ParentPath: "ops.incident_diagnosis.network", NodeType: SkillNodeTypeCapability, Title: "域名连通性诊断", Topic: "domain", CapabilityKey: "cap.diagnosis.domain", PackKey: models.SkillPackDomain, FeatureKey: models.FeatureKeyAIDiagnosis, ExecutionMode: ExecutionModeServerAI, CLIVisible: true, SortOrder: 36},
	{Path: "ops.incident_diagnosis.middleware.nginx", ParentPath: "ops.incident_diagnosis.middleware", NodeType: SkillNodeTypeCapability, Title: "Nginx / 网关诊断", Topic: "nginx", CapabilityKey: "cap.diagnosis.nginx", PackKey: models.SkillPackNginx, FeatureKey: models.FeatureKeyAIDiagnosis, ExecutionMode: ExecutionModeServerAI, CLIVisible: true, SortOrder: 43},
	{Path: "ops.incident_diagnosis.middleware.mysql", ParentPath: "ops.incident_diagnosis.middleware", NodeType: SkillNodeTypeCapability, Title: "MySQL 诊断", Topic: "mysql", CapabilityKey: "cap.diagnosis.mysql", PackKey: models.SkillPackMySQL, FeatureKey: models.FeatureKeyAIDiagnosis, ExecutionMode: ExecutionModeServerAI, CLIVisible: true, SortOrder: 44},
	{Path: "ops.incident_diagnosis.middleware.postgresql", ParentPath: "ops.incident_diagnosis.middleware", NodeType: SkillNodeTypeCapability, Title: "PostgreSQL 诊断", Topic: "postgresql", CapabilityKey: "cap.diagnosis.postgresql", PackKey: models.SkillPackPostgreSQL, FeatureKey: models.FeatureKeyAIDiagnosis, ExecutionMode: ExecutionModeServerAI, CLIVisible: true, SortOrder: 46},
	{Path: "ops.incident_diagnosis.middleware.elasticsearch", ParentPath: "ops.incident_diagnosis.middleware", NodeType: SkillNodeTypeCapability, Title: "Elasticsearch 诊断", Topic: "elasticsearch", CapabilityKey: "cap.diagnosis.elasticsearch", PackKey: models.SkillPackElasticsearch, FeatureKey: models.FeatureKeyAIDiagnosis, ExecutionMode: ExecutionModeServerAI, CLIVisible: true, SortOrder: 45},
	{Path: "ops.incident_diagnosis.application", ParentPath: "ops.incident_diagnosis", NodeType: SkillNodeTypeCategory, Title: "应用运行时", CLIVisible: true, SortOrder: 50},
	{Path: "ops.knowledge_base", ParentPath: "ops", NodeType: SkillNodeTypeCategory, Title: "经验库", CLIVisible: true, SortOrder: 60},
	{Path: "ops.incident_diagnosis.kubernetes.workload", ParentPath: "ops.incident_diagnosis.kubernetes", NodeType: SkillNodeTypeCapability, Title: "K8s 工作负载诊断", Topic: "k8s", CapabilityKey: "cap.diagnosis.k8s.workload", PackKey: models.SkillPackK8s, FeatureKey: models.FeatureKeyAIDiagnosis, ExecutionMode: ExecutionModeServerPlanReadonly, CLIVisible: true, SortOrder: 100},
	{Path: "ops.incident_diagnosis.kubernetes.workload.pod_pending", ParentPath: "ops.incident_diagnosis.kubernetes.workload", NodeType: SkillNodeTypeSkill, Title: "K8s Pod Pending", Topic: "k8s", SkillKey: "skill.k8s.workload.pod_pending", ProblemKey: "pod_pending", CapabilityKey: "cap.diagnosis.k8s.workload", PackKey: models.SkillPackK8s, FeatureKey: models.FeatureKeyAIDiagnosis, ExecutionMode: ExecutionModeServerPlanReadonly, CLIVisible: true, SortOrder: 101},
	{Path: "ops.incident_diagnosis.kubernetes.workload.crashloop", ParentPath: "ops.incident_diagnosis.kubernetes.workload", NodeType: SkillNodeTypeSkill, Title: "K8s Pod CrashLoopBackOff", Topic: "k8s", SkillKey: "skill.k8s.workload.crashloop", ProblemKey: "crashloop", CapabilityKey: "cap.diagnosis.k8s.workload", PackKey: models.SkillPackK8s, FeatureKey: models.FeatureKeyAIDiagnosis, ExecutionMode: ExecutionModeServerPlanReadonly, CLIVisible: true, SortOrder: 102},
	{Path: "ops.incident_diagnosis.kubernetes.workload.sandbox_changed", ParentPath: "ops.incident_diagnosis.kubernetes.workload", NodeType: SkillNodeTypeSkill, Title: "K8s 集群抖动 / SandboxChanged", Topic: "k8s", SkillKey: "skill.k8s.workload.sandbox_changed", ProblemKey: "sandbox_changed", CapabilityKey: "cap.diagnosis.k8s.workload", PackKey: models.SkillPackK8s, FeatureKey: models.FeatureKeyAIDiagnosis, ExecutionMode: ExecutionModeServerPlanReadonly, CLIVisible: true, SortOrder: 103},
	{Path: "ops.incident_diagnosis.kubernetes.workload.general", ParentPath: "ops.incident_diagnosis.kubernetes.workload", NodeType: SkillNodeTypeSkill, Title: "K8s 通用根因分析", Topic: "k8s", SkillKey: "skill.k8s.workload.general", ProblemKey: "workload_general", CapabilityKey: "cap.diagnosis.k8s.workload", PackKey: models.SkillPackK8s, FeatureKey: models.FeatureKeyAIDiagnosis, ExecutionMode: ExecutionModeServerPlanReadonly, CLIVisible: true, SortOrder: 104},

	{Path: "ops.delivery_implementation.kubernetes", ParentPath: "ops.delivery_implementation", NodeType: SkillNodeTypeCapability, Title: "K8s 交付实施", Topic: "k8s", CapabilityKey: "cap.delivery.k8s", PackKey: models.PackKeyK8sDelivery, FeatureKey: models.FeatureKeyK8sDelivery, ExecutionMode: ExecutionModeDownloadOrInstall, CLIVisible: true, SortOrder: 200},
	{Path: "ops.delivery_implementation.kubernetes.preflight", ParentPath: "ops.delivery_implementation.kubernetes", NodeType: SkillNodeTypeSkill, Title: "K8s 部署前预检", Topic: "k8s", SkillKey: "skill.k8s.delivery.preflight", ProblemKey: "preflight", CapabilityKey: "cap.delivery.k8s", PackKey: models.PackKeyK8sDelivery, FeatureKey: models.FeatureKeyK8sDelivery, ExecutionMode: ExecutionModeLocalReadonly, CLIVisible: true, SortOrder: 201},
	{Path: "ops.delivery_implementation.kubernetes.install", ParentPath: "ops.delivery_implementation.kubernetes", NodeType: SkillNodeTypeSkill, Title: "K8s 离线部署 / 安装", Topic: "k8s", SkillKey: "skill.k8s.delivery.install", ProblemKey: "install", CapabilityKey: "cap.delivery.k8s", PackKey: models.PackKeyK8sDelivery, FeatureKey: models.FeatureKeyK8sDelivery, ExecutionMode: ExecutionModeDownloadOrInstall, CLIVisible: true, SortOrder: 202},
	{Path: "ops.delivery_implementation.kubernetes.recovery", ParentPath: "ops.delivery_implementation.kubernetes", NodeType: SkillNodeTypeSkill, Title: "K8s 安装失败恢复", Topic: "k8s", SkillKey: "skill.k8s.delivery.recovery", ProblemKey: "install_recovery", CapabilityKey: "cap.delivery.k8s", PackKey: models.PackKeyK8sDelivery, FeatureKey: models.FeatureKeyK8sDelivery, ExecutionMode: ExecutionModeServerJobWrite, CLIVisible: true, SortOrder: 203},
	{Path: "ops.delivery_implementation.kubernetes.uninstall", ParentPath: "ops.delivery_implementation.kubernetes", NodeType: SkillNodeTypeSkill, Title: "K8s 集群卸载", Topic: "k8s", SkillKey: "skill.k8s.delivery.uninstall", ProblemKey: "uninstall", CapabilityKey: "cap.delivery.k8s", PackKey: models.PackKeyK8sDelivery, FeatureKey: models.FeatureKeyK8sDelivery, ExecutionMode: ExecutionModeServerJobWrite, CLIVisible: true, SortOrder: 204},

	{Path: "ops.delivery_implementation.node_ops", ParentPath: "ops.delivery_implementation", NodeType: SkillNodeTypeCategory, Title: "节点基础服务", Topic: "service", PackKey: models.PackKeyNodeOps, FeatureKey: models.FeatureKeyNodeOps, CLIVisible: true, SortOrder: 210},
	{Path: "ops.delivery_implementation.node_ops.service_install", ParentPath: "ops.delivery_implementation.node_ops", NodeType: SkillNodeTypeSkill, Title: "基础服务安装", Topic: "service", SkillKey: "skill.service.delivery.install", ProblemKey: "install", CapabilityKey: "cap.delivery.service", PackKey: models.PackKeyNodeOps, FeatureKey: models.FeatureKeyNodeOps, ExecutionMode: ExecutionModeDownloadOrInstall, CLIVisible: true, SortOrder: 211},
	{Path: "ops.delivery_implementation.node_ops.service_uninstall", ParentPath: "ops.delivery_implementation.node_ops", NodeType: SkillNodeTypeSkill, Title: "基础服务卸载", Topic: "service", SkillKey: "skill.service.delivery.uninstall", ProblemKey: "uninstall", CapabilityKey: "cap.delivery.service", PackKey: models.PackKeyNodeOps, FeatureKey: models.FeatureKeyNodeOps, ExecutionMode: ExecutionModeServerJobWrite, CLIVisible: true, SortOrder: 212},
	{Path: "ops.delivery_implementation.node_ops.service_recovery", ParentPath: "ops.delivery_implementation.node_ops", NodeType: SkillNodeTypeSkill, Title: "基础服务安装失败恢复", Topic: "service", SkillKey: "skill.service.delivery.recovery", ProblemKey: "install_recovery", CapabilityKey: "cap.delivery.service", PackKey: models.PackKeyNodeOps, FeatureKey: models.FeatureKeyNodeOps, ExecutionMode: ExecutionModeServerJobWrite, CLIVisible: true, SortOrder: 213},

	{Path: "ops.incident_diagnosis.middleware.kafka.lag", ParentPath: "ops.incident_diagnosis.middleware.kafka", NodeType: SkillNodeTypeSkill, Title: "Kafka 消费堆积 / 集群快诊", Topic: "kafka", SkillKey: "skill.kafka.consumer_lag", ProblemKey: "consumer_lag", CapabilityKey: "cap.diagnosis.kafka", PackKey: models.SkillPackKafka, FeatureKey: models.FeatureKeyAIDiagnosis, ExecutionMode: ExecutionModeServerAI, CLIVisible: true, SortOrder: 300},
	{Path: "ops.incident_diagnosis.middleware.redis.latency", ParentPath: "ops.incident_diagnosis.middleware.redis", NodeType: SkillNodeTypeSkill, Title: "Redis 延迟 / 慢查询快诊", Topic: "redis", SkillKey: "skill.redis.latency", ProblemKey: "latency", CapabilityKey: "cap.diagnosis.redis", PackKey: models.SkillPackRedis, FeatureKey: models.FeatureKeyAIDiagnosis, ExecutionMode: ExecutionModeServerAI, CLIVisible: true, SortOrder: 310},
	{Path: "ops.incident_diagnosis.middleware.nginx.5xx", ParentPath: "ops.incident_diagnosis.middleware.nginx", NodeType: SkillNodeTypeSkill, Title: "Nginx / 网关 5xx 快诊", Topic: "nginx", SkillKey: "skill.nginx.5xx", ProblemKey: "5xx", CapabilityKey: "cap.diagnosis.nginx", PackKey: models.SkillPackNginx, FeatureKey: models.FeatureKeyAIDiagnosis, ExecutionMode: ExecutionModeServerAI, CLIVisible: true, SortOrder: 320},
	{Path: "ops.incident_diagnosis.middleware.mysql.runtime", ParentPath: "ops.incident_diagnosis.middleware.mysql", NodeType: SkillNodeTypeSkill, Title: "MySQL 运行状态快诊", Topic: "mysql", SkillKey: "skill.mysql.runtime", ProblemKey: "runtime", CapabilityKey: "cap.diagnosis.mysql", PackKey: models.SkillPackMySQL, FeatureKey: models.FeatureKeyAIDiagnosis, ExecutionMode: ExecutionModeServerAI, CLIVisible: true, SortOrder: 330},
	{Path: "ops.incident_diagnosis.middleware.postgresql.general", ParentPath: "ops.incident_diagnosis.middleware.postgresql", NodeType: SkillNodeTypeSkill, Title: "PostgreSQL 通用根因分析", Topic: "postgresql", SkillKey: "skill.postgresql.general", ProblemKey: "general", CapabilityKey: "cap.diagnosis.postgresql", PackKey: models.SkillPackPostgreSQL, FeatureKey: models.FeatureKeyAIDiagnosis, ExecutionMode: ExecutionModeServerAI, CLIVisible: true, SortOrder: 335},
	{Path: "ops.incident_diagnosis.middleware.elasticsearch.health", ParentPath: "ops.incident_diagnosis.middleware.elasticsearch", NodeType: SkillNodeTypeSkill, Title: "Elasticsearch 健康快诊", Topic: "elasticsearch", SkillKey: "skill.elasticsearch.health", ProblemKey: "health", CapabilityKey: "cap.diagnosis.elasticsearch", PackKey: models.SkillPackElasticsearch, FeatureKey: models.FeatureKeyAIDiagnosis, ExecutionMode: ExecutionModeServerAI, CLIVisible: true, SortOrder: 340},
	{Path: "ops.incident_diagnosis.network.domain.connectivity", ParentPath: "ops.incident_diagnosis.network.domain", NodeType: SkillNodeTypeSkill, Title: "域名 / DNS / HTTP(S) 诊断", Topic: "domain", SkillKey: "skill.domain.connectivity", ProblemKey: "connectivity", CapabilityKey: "cap.diagnosis.domain", PackKey: models.SkillPackDomain, FeatureKey: models.FeatureKeyAIDiagnosis, ExecutionMode: ExecutionModeServerAI, CLIVisible: true, SortOrder: 345},

	{Path: "ops.incident_diagnosis.linux.performance", ParentPath: "ops.incident_diagnosis.linux", NodeType: SkillNodeTypeCapability, Title: "Linux 系统性能诊断", Topic: "linux", CapabilityKey: "cap.diagnosis.linux.performance", PackKey: models.PackKeyBackupPerformance, FeatureKey: models.FeatureKeyBackupPerformance, ExecutionMode: ExecutionModeServerAI, CLIVisible: true, SortOrder: 350},
	{Path: "ops.incident_diagnosis.linux.performance.general", ParentPath: "ops.incident_diagnosis.linux.performance", NodeType: SkillNodeTypeSkill, Title: "CPU / 内存 / 磁盘 / 进程综合诊断", Topic: "linux", SkillKey: "skill.linux.performance.general", ProblemKey: "performance_general", CapabilityKey: "cap.diagnosis.linux.performance", PackKey: models.PackKeyBackupPerformance, FeatureKey: models.FeatureKeyBackupPerformance, ExecutionMode: ExecutionModeServerAI, CLIVisible: true, SortOrder: 351},
	{Path: "ops.incident_diagnosis.linux.performance.memory_leak", ParentPath: "ops.incident_diagnosis.linux.performance", NodeType: SkillNodeTypeSkill, Title: "Linux 进程内存泄露风险预判", Topic: "linux", SkillKey: "skill.linux.performance.memory_leak", ProblemKey: "memory_leak_risk", CapabilityKey: "cap.diagnosis.linux.performance", PackKey: models.PackKeyBackupPerformance, FeatureKey: models.FeatureKeyBackupPerformance, ExecutionMode: ExecutionModeServerAI, CLIVisible: true, SortOrder: 352},

	{Path: "ops.incident_diagnosis.application.go_runtime", ParentPath: "ops.incident_diagnosis.application", NodeType: SkillNodeTypeCapability, Title: "Go Runtime 智能诊断", Topic: "go_runtime", CapabilityKey: "cap.diagnosis.go_runtime", PackKey: models.PackKeyRuntimeObserve, FeatureKey: models.FeatureKeyRuntimeObserve, ExecutionMode: ExecutionModeServerPlanReadonly, CLIVisible: true, SortOrder: 400},
	{Path: "ops.incident_diagnosis.application.go_runtime.process", ParentPath: "ops.incident_diagnosis.application.go_runtime", NodeType: SkillNodeTypeSkill, Title: "Go 进程运行时诊断", Topic: "go_runtime", SkillKey: "skill.go_runtime.process", ProblemKey: "process_runtime", CapabilityKey: "cap.diagnosis.go_runtime", PackKey: models.PackKeyRuntimeObserve, FeatureKey: models.FeatureKeyRuntimeObserve, ExecutionMode: ExecutionModeLocalAIFallback, CLIVisible: true, SortOrder: 401},
	{Path: "ops.incident_diagnosis.application.go_runtime.k8s_workload", ParentPath: "ops.incident_diagnosis.application.go_runtime", NodeType: SkillNodeTypeSkill, Title: "Go 应用 Pod 运行时诊断", Topic: "go_runtime", SkillKey: "skill.go_runtime.k8s_workload", ProblemKey: "k8s_workload_runtime", CapabilityKey: "cap.diagnosis.go_runtime", PackKey: models.PackKeyRuntimeObserve, FeatureKey: models.FeatureKeyRuntimeObserve, ExecutionMode: ExecutionModeServerPlanReadonly, CLIVisible: true, SortOrder: 402},

	{Path: "ops.knowledge_base.error_codes", ParentPath: "ops.knowledge_base", NodeType: SkillNodeTypeSkill, Title: "OpsFleet 错误码根因库", Topic: "errorcode", SkillKey: "skill.opsfleet.error_codes", ProblemKey: "error_codes", CapabilityKey: "cap.knowledge.error_codes", PackKey: models.SkillPackK8s, FeatureKey: models.FeatureKeyAIDiagnosis, ExecutionMode: ExecutionModeLocalReadonly, CLIVisible: true, SortOrder: 500},
}

// SkillTreeNodes returns nodes from ActiveSkillTree (database active revision, or builtin fallback).
func SkillTreeNodes() []SkillTreeNode {
	return ActiveSkillTree().Nodes
}

// ActiveSkillTreeRev returns the revision id exposed to CLI/admin.
func ActiveSkillTreeRev() string {
	return ActiveSkillTree().TreeRev
}

func SkillTreeNodesWithAssetStats() ([]SkillTreeNode, error) {
	nodes := SkillTreeNodes()
	var rows []models.SkillAsset
	if err := database.DB.Select("id", "status", "topic", "skill_key", "problem_key", "capability_key", "category_path").Find(&rows).Error; err != nil {
		return nodes, err
	}
	applySkillTreeAssetStats(nodes, rows)
	return nodes, nil
}

func applySkillTreeAssetStats(nodes []SkillTreeNode, assets []models.SkillAsset) {
	statsByPath := make(map[string]*SkillTreeAssetStats, len(nodes))
	parentByPath := make(map[string]string, len(nodes))
	for i := range nodes {
		statsByPath[nodes[i].Path] = &SkillTreeAssetStats{}
		nodes[i].AssetStats = statsByPath[nodes[i].Path]
		parentByPath[nodes[i].Path] = nodes[i].ParentPath
	}
	for _, asset := range assets {
		path := resolveSkillAssetTreePath(nodes, asset)
		if path == "" {
			continue
		}
		for path != "" {
			stats, ok := statsByPath[path]
			if !ok {
				break
			}
			incrementSkillTreeAssetStats(stats, asset.Status)
			path = parentByPath[path]
		}
	}
}

func resolveSkillAssetTreePath(nodes []SkillTreeNode, asset models.SkillAsset) string {
	categoryPath := strings.TrimSpace(asset.CategoryPath)
	if categoryPath != "" {
		for _, n := range nodes {
			if n.Path == categoryPath {
				return n.Path
			}
		}
	}
	skillKey := strings.TrimSpace(asset.SkillKey)
	if skillKey != "" {
		for _, n := range nodes {
			if n.SkillKey == skillKey {
				return n.Path
			}
		}
	}
	topic := normalizeSkillTopic(asset.Topic)
	problemKey := strings.TrimSpace(asset.ProblemKey)
	if topic != "" && problemKey != "" {
		for _, n := range nodes {
			if n.NodeType == SkillNodeTypeSkill && normalizeSkillTopic(n.Topic) == topic && n.ProblemKey == problemKey {
				return n.Path
			}
		}
	}
	capabilityKey := strings.TrimSpace(asset.CapabilityKey)
	if capabilityKey != "" {
		for _, n := range nodes {
			if n.CapabilityKey == capabilityKey && n.NodeType == SkillNodeTypeCapability {
				return n.Path
			}
		}
	}
	if topic != "" {
		for _, n := range nodes {
			if normalizeSkillTopic(n.Topic) == topic && n.NodeType == SkillNodeTypeCapability {
				return n.Path
			}
		}
		for _, n := range nodes {
			if normalizeSkillTopic(n.Topic) == topic {
				return n.Path
			}
		}
	}
	return ""
}

func incrementSkillTreeAssetStats(stats *SkillTreeAssetStats, status string) {
	if stats == nil {
		return
	}
	stats.Total++
	switch strings.ToLower(strings.TrimSpace(status)) {
	case models.SkillAssetStatusDraft:
		stats.Draft++
	case models.SkillAssetStatusReview:
		stats.Review++
	case models.SkillAssetStatusApproved:
		stats.Approved++
	case models.SkillAssetStatusDeprecated:
		stats.Deprecated++
	}
}

func SkillTreeNodeByPath(path string) (SkillTreeNode, bool) {
	return SkillTreeNodeByPathActive(path)
}

func NormalizeSkillExecutionIntent(topic string, ctx map[string]string, in SkillExecutionIntent) SkillExecutionIntent {
	intent := in
	intent.Topic = normalizeSkillTopic(defaultStringForSkill(intent.Topic, topic))
	if intent.CommandKind == "" {
		intent.CommandKind = "analyze"
	}
	if intent.Action == "" {
		intent.Action = "ai_diagnose"
	}
	if intent.CandidateNodePath != "" {
		if n, ok := SkillTreeNodeByPath(intent.CandidateNodePath); ok {
			return mergeIntentWithNode(intent, n)
		}
	}
	if intent.NodePath != "" {
		if n, ok := SkillTreeNodeByPath(intent.NodePath); ok {
			return mergeIntentWithNode(intent, n)
		}
	}
	if n, ok := inferSkillTreeNode(intent.Topic, ctx); ok {
		return mergeIntentWithNode(intent, n)
	}
	if intent.PackKey == "" {
		intent.PackKey = packKeyForSkillTopic(intent.Topic)
	}
	if intent.ExecutionMode == "" {
		intent.ExecutionMode = ExecutionModeServerAI
	}
	return intent
}

func mergeIntentWithNode(in SkillExecutionIntent, n SkillTreeNode) SkillExecutionIntent {
	out := in
	out.NodePath = n.Path
	if out.CandidateNodePath == "" {
		out.CandidateNodePath = n.Path
	}
	out.Topic = defaultStringForSkill(out.Topic, n.Topic)
	out.SkillKey = defaultStringForSkill(out.SkillKey, n.SkillKey)
	out.ProblemKey = defaultStringForSkill(out.ProblemKey, n.ProblemKey)
	out.CapabilityKey = defaultStringForSkill(out.CapabilityKey, n.CapabilityKey)
	out.PackKey = defaultStringForSkill(out.PackKey, n.PackKey)
	out.ExecutionMode = defaultStringForSkill(out.ExecutionMode, n.ExecutionMode)
	return out
}

func inferSkillTreeNode(topic string, ctx map[string]string) (SkillTreeNode, bool) {
	topic = normalizeSkillTopic(topic)
	problem := inferProblemKey(topic, ctx)
	for _, n := range activeSkillTreeNodes() {
		if n.Status == models.SkillTreeNodeStatusDisabled {
			continue
		}
		if n.NodeType != SkillNodeTypeSkill {
			continue
		}
		if normalizeSkillTopic(n.Topic) == topic && n.ProblemKey == problem {
			return n, true
		}
	}
	for _, n := range activeSkillTreeNodes() {
		if n.Status == models.SkillTreeNodeStatusDisabled {
			continue
		}
		if n.NodeType == SkillNodeTypeSkill && normalizeSkillTopic(n.Topic) == topic {
			return n, true
		}
	}
	return SkillTreeNode{}, false
}

func inferProblemKey(topic string, ctx map[string]string) string {
	switch normalizeSkillTopic(topic) {
	case "k8s":
		issue := strings.ToLower(strings.TrimSpace(valueFromSkillContext(ctx, "issue")))
		pod := strings.ToLower(strings.TrimSpace(valueFromSkillContext(ctx, "pod")))
		switch {
		case strings.Contains(issue, "pending") || strings.Contains(pod, "pending"):
			return "pod_pending"
		case strings.Contains(issue, "crash") || strings.Contains(pod, "crash"):
			return "crashloop"
		case strings.Contains(issue, "instability") || strings.Contains(issue, "sandbox") || strings.Contains(pod, "calico") || strings.Contains(pod, "coredns"):
			return "sandbox_changed"
		default:
			return "workload_general"
		}
	case "go_runtime":
		if hasAnySkillContext(ctx, "pod", "deployment", "statefulset", "daemonset", "replicaset", "job", "cronjob", "service", "ingress", "pvc") {
			return "k8s_workload_runtime"
		}
		return "process_runtime"
	case "kafka":
		return "consumer_lag"
	case "redis":
		return "latency"
	case "nginx":
		return "5xx"
	case "mysql":
		return "runtime"
	case "postgresql", "postgres":
		return "general"
	case "elasticsearch":
		return "health"
	case "domain":
		return "connectivity"
	case "linux":
		if pk := strings.ToLower(strings.TrimSpace(valueFromSkillContext(ctx, "problem"))); pk == "memory_leak_risk" {
			return "memory_leak_risk"
		}
		return "performance_general"
	case "errorcode":
		return "error_codes"
	default:
		return "general"
	}
}

func normalizeSkillTopic(topic string) string {
	switch strings.ToLower(strings.TrimSpace(topic)) {
	case "kubernetes":
		return "k8s"
	case "go-runtime", "pod-go":
		return "go_runtime"
	case "es":
		return "elasticsearch"
	case "postgres":
		return "postgresql"
	case "dns":
		return "domain"
	case "deploy", "install":
		return "errorcode"
	default:
		return strings.ToLower(strings.TrimSpace(topic))
	}
}

func packKeyForSkillTopic(topic string) string {
	switch normalizeSkillTopic(topic) {
	case "k8s", "errorcode":
		return models.SkillPackK8s
	case "kafka":
		return models.SkillPackKafka
	case "redis":
		return models.SkillPackRedis
	case "nginx":
		return models.SkillPackNginx
	case "mysql":
		return models.SkillPackMySQL
	case "postgresql", "postgres":
		return models.SkillPackPostgreSQL
	case "elasticsearch":
		return models.SkillPackElasticsearch
	case "domain":
		return models.SkillPackDomain
	case "linux":
		return models.PackKeyBackupPerformance
	case "go_runtime":
		return models.PackKeyRuntimeObserve
	default:
		return models.SkillPackK8s
	}
}

func valueFromSkillContext(ctx map[string]string, key string) string {
	if ctx == nil {
		return ""
	}
	return strings.TrimSpace(ctx[key])
}

func hasAnySkillContext(ctx map[string]string, keys ...string) bool {
	for _, k := range keys {
		if valueFromSkillContext(ctx, k) != "" {
			return true
		}
	}
	return false
}

func defaultStringForSkill(v, fallback string) string {
	if strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(fallback)
}
