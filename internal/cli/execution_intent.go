package cli

import "strings"

type executionIntent struct {
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

func buildOpsExecutionIntent(args []string) executionIntent {
	if len(args) < 2 || args[0] != "ops" {
		return executionIntent{}
	}
	intent := executionIntent{
		CommandKind: "ops",
		Action:      "delivery_ops",
		PackKey:     "pack.k8s_delivery",
	}
	switch args[1] {
	case "k8s":
		if len(args) < 3 {
			return intent
		}
		intent.Topic = "k8s"
		intent.CapabilityKey = "cap.delivery.k8s"
		intent.PackKey = "pack.k8s_delivery"
		switch args[2] {
		case "recover":
			intent.ProblemKey = "install_recovery"
			intent.ExecutionMode = "server_job_write"
			intent.NodePath = "ops.delivery_implementation.kubernetes.recovery"
			intent.SkillKey = "skill.k8s.delivery.recovery"
		case "install":
			intent.ProblemKey = "install"
			intent.ExecutionMode = "download_or_install"
			intent.NodePath = "ops.delivery_implementation.kubernetes.install"
			intent.SkillKey = "skill.k8s.delivery.install"
		case "uninstall":
			intent.ProblemKey = "uninstall"
			intent.ExecutionMode = "server_job_write"
			intent.NodePath = "ops.delivery_implementation.kubernetes.uninstall"
			intent.SkillKey = "skill.k8s.delivery.uninstall"
		case "preflight":
			intent.ProblemKey = "preflight"
			intent.ExecutionMode = "local_readonly"
			intent.NodePath = "ops.delivery_implementation.kubernetes.preflight"
			intent.SkillKey = "skill.k8s.delivery.preflight"
		}
	case "uninstall":
		if len(args) >= 3 && args[2] == "k8s" {
			intent.Topic = "k8s"
			intent.ProblemKey = "uninstall"
			intent.CapabilityKey = "cap.delivery.k8s"
			intent.PackKey = "pack.k8s_delivery"
			intent.ExecutionMode = "server_job_write"
			intent.NodePath = "ops.delivery_implementation.kubernetes.uninstall"
			intent.SkillKey = "skill.k8s.delivery.uninstall"
		}
	case "service":
		if len(args) < 3 {
			return intent
		}
		intent.CapabilityKey = "cap.delivery.service"
		intent.PackKey = "pack.node_ops"
		switch args[2] {
		case "recover":
			intent.Topic = "service"
			if len(args) >= 4 {
				intent.Topic = normalizeIntentTopic(args[3])
			}
			intent.ProblemKey = "install_recovery"
			intent.ExecutionMode = "server_job_write"
			intent.NodePath = "ops.delivery_implementation.node_ops.service_recovery"
			intent.SkillKey = "skill.service.delivery.recovery"
		case "install":
			intent.Topic = "service"
			intent.ProblemKey = "install"
			intent.ExecutionMode = "download_or_install"
			intent.NodePath = "ops.delivery_implementation.node_ops.service_install"
			intent.SkillKey = "skill.service.delivery.install"
		case "uninstall":
			intent.Topic = "service"
			if len(args) >= 4 {
				intent.Topic = normalizeIntentTopic(args[3])
			}
			intent.ProblemKey = "uninstall"
			intent.ExecutionMode = "server_job_write"
			intent.NodePath = "ops.delivery_implementation.node_ops.service_uninstall"
			intent.SkillKey = "skill.service.delivery.uninstall"
		}
	}
	intent.CandidateNodePath = intent.NodePath
	return intent
}

func buildExecutionIntent(commandKind, topic string, kv map[string]string) executionIntent {
	t := normalizeIntentTopic(topic)
	problem := inferIntentProblem(t, kv)
	intent := executionIntent{
		CommandKind:   strings.TrimSpace(commandKind),
		Action:        intentActionForCommand(commandKind),
		Topic:         t,
		ProblemKey:    problem,
		PackKey:       intentPackKey(t),
		ExecutionMode: intentExecutionMode(t, problem),
	}
	intent.CandidateNodePath, intent.SkillKey, intent.CapabilityKey = intentTreeCoordinates(t, problem)
	intent.NodePath = intent.CandidateNodePath
	return intent
}

func intentActionForCommand(commandKind string) string {
	switch strings.ToLower(strings.TrimSpace(commandKind)) {
	case "ask":
		return "ai_ask"
	case "runbook":
		return "ai_runbook"
	case "check":
		return "ai_diagnose"
	case "diagnose":
		return "runtime_diagnose"
	default:
		return "ai_diagnose"
	}
}

func normalizeIntentTopic(topic string) string {
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
	case "upgrade":
		return "install"
	case "deploy", "install":
		return "errorcode"
	default:
		return strings.ToLower(strings.TrimSpace(topic))
	}
}

func inferIntentProblem(topic string, kv map[string]string) string {
	switch normalizeIntentTopic(topic) {
	case "k8s":
		issue := strings.ToLower(strings.TrimSpace(kv["issue"]))
		pod := strings.ToLower(strings.TrimSpace(kv["pod"]))
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
		if hasIntentKV(kv, "pod", "deployment", "statefulset", "daemonset", "replicaset", "job", "cronjob", "service", "ingress", "pvc") {
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
		if hasIntentKV(kv, "problem", "problem_key") {
			pk := strings.ToLower(strings.TrimSpace(kv["problem"]))
			if pk == "" {
				pk = strings.ToLower(strings.TrimSpace(kv["problem_key"]))
			}
			if pk == "memory_leak_risk" {
				return "memory_leak_risk"
			}
		}
		return "performance_general"
	case "install":
		return "download_failure"
	case "errorcode":
		return "error_codes"
	default:
		return "general"
	}
}

func intentTreeCoordinates(topic, problem string) (nodePath, skillKey, capabilityKey string) {
	switch normalizeIntentTopic(topic) {
	case "k8s":
		capabilityKey = "cap.diagnosis.k8s.workload"
		switch problem {
		case "pod_pending":
			return "ops.incident_diagnosis.kubernetes.workload.pod_pending", "skill.k8s.workload.pod_pending", capabilityKey
		case "crashloop":
			return "ops.incident_diagnosis.kubernetes.workload.crashloop", "skill.k8s.workload.crashloop", capabilityKey
		case "sandbox_changed":
			return "ops.incident_diagnosis.kubernetes.workload.sandbox_changed", "skill.k8s.workload.sandbox_changed", capabilityKey
		case "preflight":
			return "ops.delivery_implementation.kubernetes.preflight", "skill.k8s.delivery.preflight", "cap.delivery.k8s"
		case "install":
			return "ops.delivery_implementation.kubernetes.install", "skill.k8s.delivery.install", "cap.delivery.k8s"
		case "install_recovery":
			return "ops.delivery_implementation.kubernetes.recovery", "skill.k8s.delivery.recovery", "cap.delivery.k8s"
		case "uninstall":
			return "ops.delivery_implementation.kubernetes.uninstall", "skill.k8s.delivery.uninstall", "cap.delivery.k8s"
		default:
			return "ops.incident_diagnosis.kubernetes.workload.general", "skill.k8s.workload.general", capabilityKey
		}
	case "go_runtime":
		capabilityKey = "cap.diagnosis.go_runtime"
		if problem == "k8s_workload_runtime" {
			return "ops.incident_diagnosis.application.go_runtime.k8s_workload", "skill.go_runtime.k8s_workload", capabilityKey
		}
		return "ops.incident_diagnosis.application.go_runtime.process", "skill.go_runtime.process", capabilityKey
	case "kafka":
		return "ops.incident_diagnosis.middleware.kafka.lag", "skill.kafka.consumer_lag", "cap.diagnosis.kafka"
	case "redis":
		return "ops.incident_diagnosis.middleware.redis.latency", "skill.redis.latency", "cap.diagnosis.redis"
	case "nginx":
		return "ops.incident_diagnosis.middleware.nginx.5xx", "skill.nginx.5xx", "cap.diagnosis.nginx"
	case "mysql":
		return "ops.incident_diagnosis.middleware.mysql.runtime", "skill.mysql.runtime", "cap.diagnosis.mysql"
	case "postgresql", "postgres":
		return "ops.incident_diagnosis.middleware.postgresql.general", "skill.postgresql.general", "cap.diagnosis.postgresql"
	case "elasticsearch":
		return "ops.incident_diagnosis.middleware.elasticsearch.health", "skill.elasticsearch.health", "cap.diagnosis.elasticsearch"
	case "domain":
		return "ops.incident_diagnosis.network.domain.connectivity", "skill.domain.connectivity", "cap.diagnosis.domain"
	case "linux":
		capabilityKey = "cap.diagnosis.linux.performance"
		if problem == "memory_leak_risk" {
			return "ops.incident_diagnosis.linux.performance.memory_leak", "skill.linux.performance.memory_leak", capabilityKey
		}
		return "ops.incident_diagnosis.linux.performance.general", "skill.linux.performance.general", capabilityKey
	case "install":
		return "ops.delivery_implementation.cli.install", "skill.cli.install_recovery", "cap.delivery.cli"
	case "errorcode":
		return "ops.knowledge_base.error_codes", "skill.opsfleet.error_codes", "cap.knowledge.error_codes"
	default:
		return "", "", ""
	}
}

func intentPackKey(topic string) string {
	switch normalizeIntentTopic(topic) {
	case "k8s", "errorcode":
		return "skillpack.k8s"
	case "kafka":
		return "skillpack.kafka"
	case "redis":
		return "skillpack.redis"
	case "nginx":
		return "skillpack.nginx"
	case "mysql":
		return "skillpack.mysql"
	case "postgresql", "postgres":
		return "skillpack.postgresql"
	case "elasticsearch":
		return "skillpack.elasticsearch"
	case "domain":
		return "skillpack.domain"
	case "linux":
		return "pack.backup_performance"
	case "install":
		return "skillpack.cli"
	case "go_runtime":
		return "pack.runtime_observe"
	default:
		return "skillpack.k8s"
	}
}

func intentExecutionMode(topic, problem string) string {
	switch normalizeIntentTopic(topic) {
	case "k8s":
		if problem == "preflight" {
			return "local_readonly"
		}
		return "server_plan_readonly"
	case "go_runtime":
		if problem == "process_runtime" {
			return "local_ai_fallback"
		}
		return "server_plan_readonly"
	case "kafka", "redis", "nginx", "mysql", "postgresql", "postgres", "elasticsearch", "domain", "linux", "install":
		return "server_ai"
	case "errorcode":
		return "local_readonly"
	default:
		return "server_ai"
	}
}

func hasIntentKV(kv map[string]string, keys ...string) bool {
	for _, k := range keys {
		if strings.TrimSpace(kv[k]) != "" {
			return true
		}
	}
	return false
}
