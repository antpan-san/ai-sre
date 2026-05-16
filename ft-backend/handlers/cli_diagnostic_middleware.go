package handlers

import (
	"fmt"
	"strings"
)

func buildReadonlyDiagnosticPlan(topic string, kv map[string]string) ([]diagnosticPlanStep, error) {
	switch strings.ToLower(strings.TrimSpace(topic)) {
	case "k8s", "kubernetes":
		return buildK8sReadonlyDiagnosticPlan(kv), nil
	case "go_runtime", "go-runtime":
		return buildGoRuntimeReadonlyDiagnosticPlan(kv), nil
	case "redis":
		return buildRedisReadonlyDiagnosticPlan(kv)
	case "kafka":
		return buildKafkaReadonlyDiagnosticPlan(kv)
	case "nginx":
		return buildNginxReadonlyDiagnosticPlan(kv)
	case "mysql":
		return buildMySQLReadonlyDiagnosticPlan(kv)
	case "elasticsearch", "es":
		return buildElasticsearchReadonlyDiagnosticPlan(kv)
	default:
		return nil, fmt.Errorf("当前仅支持 k8s / go_runtime / redis / kafka / nginx / mysql / elasticsearch 只读诊断任务单")
	}
}

func buildRedisReadonlyDiagnosticPlan(kv map[string]string) ([]diagnosticPlanStep, error) {
	addr := diagnosticTargetFromMap(kv, "addr", "host", "port")
	if addr == "" {
		return nil, fmt.Errorf("redis 诊断需要 context.addr（或 host+port）")
	}
	argv := []string{"ai-sre", "redis", "diagnose", addr, "--json"}
	if pass := strings.TrimSpace(kv["password"]); pass != "" && diagnosticSafeLiteral(pass) {
		argv = append(argv, "--password", pass)
	}
	return []diagnosticPlanStep{
		{ID: "redis_diagnose_json", Title: "Redis 只读快诊（JSON）", Argv: argv, TimeoutSeconds: 30, EvidenceKey: "redis_diagnose_json"},
	}, nil
}

func buildKafkaReadonlyDiagnosticPlan(kv map[string]string) ([]diagnosticPlanStep, error) {
	bs := diagnosticTargetFromMap(kv, "bootstrap", "bootstrap_server", "addr")
	if bs == "" {
		return nil, fmt.Errorf("kafka 诊断需要 context.bootstrap（bootstrap-server）")
	}
	argv := []string{"ai-sre", "kafka", "diagnose", bs, "--json"}
	if dir := strings.TrimSpace(kv["command_dir"]); dir != "" && diagnosticSafePath(dir) {
		argv = append(argv, "--command-dir", dir)
	}
	if cfg := strings.TrimSpace(kv["config"]); cfg != "" && diagnosticSafePath(cfg) {
		argv = append(argv, "--config", cfg)
	}
	return []diagnosticPlanStep{
		{ID: "kafka_diagnose_json", Title: "Kafka 只读快诊（JSON）", Argv: argv, TimeoutSeconds: 90, EvidenceKey: "kafka_diagnose_json"},
	}, nil
}

func buildNginxReadonlyDiagnosticPlan(kv map[string]string) ([]diagnosticPlanStep, error) {
	argv := []string{"ai-sre", "nginx", "diagnose", "--json"}
	if log := strings.TrimSpace(kv["access_log"]); log != "" && diagnosticSafePath(log) {
		argv = append(argv, "--access-log", log)
	}
	return []diagnosticPlanStep{
		{ID: "nginx_diagnose_json", Title: "Nginx access log 只读快诊（JSON）", Argv: argv, TimeoutSeconds: 45, EvidenceKey: "nginx_diagnose_json"},
	}, nil
}

func buildMySQLReadonlyDiagnosticPlan(kv map[string]string) ([]diagnosticPlanStep, error) {
	dsn := strings.TrimSpace(kv["dsn"])
	if dsn == "" || !diagnosticSafeLiteral(dsn) {
		return nil, fmt.Errorf("mysql 诊断需要 context.dsn")
	}
	return []diagnosticPlanStep{
		{
			ID: "mysql_diagnose_json", Title: "MySQL 只读快诊（JSON）",
			Argv:           []string{"ai-sre", "mysql", "diagnose", dsn, "--json"},
			TimeoutSeconds: 45, EvidenceKey: "mysql_diagnose_json",
		},
	}, nil
}

func buildElasticsearchReadonlyDiagnosticPlan(kv map[string]string) ([]diagnosticPlanStep, error) {
	url := diagnosticTargetFromMap(kv, "url", "host", "addr")
	if url == "" {
		return nil, fmt.Errorf("elasticsearch 诊断需要 context.url（或 host:port）")
	}
	argv := []string{"ai-sre", "elasticsearch", "diagnose", url, "--json"}
	if strings.EqualFold(strings.TrimSpace(kv["insecure"]), "true") {
		argv = append(argv, "--insecure")
	}
	return []diagnosticPlanStep{
		{ID: "elasticsearch_diagnose_json", Title: "Elasticsearch 只读快诊（JSON）", Argv: argv, TimeoutSeconds: 45, EvidenceKey: "elasticsearch_diagnose_json"},
	}, nil
}

func diagnosticTargetFromMap(kv map[string]string, keys ...string) string {
	if kv == nil {
		return ""
	}
	for _, k := range keys {
		if v := strings.TrimSpace(kv[k]); v != "" && diagnosticSafeLiteral(v) {
			return v
		}
	}
	host := strings.TrimSpace(kv["host"])
	port := strings.TrimSpace(kv["port"])
	if host != "" && port != "" && diagnosticSafeLiteral(host) && diagnosticSafeLiteral(port) {
		return host + ":" + port
	}
	return ""
}

func diagnosticSafeLiteral(s string) bool {
	return aisreDiagnosticValueRe.MatchString(s)
}

func diagnosticSafePath(s string) bool {
	if s == "" || strings.ContainsAny(s, ";&|`$<>") {
		return false
	}
	return len(s) <= 512
}
