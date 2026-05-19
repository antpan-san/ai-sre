package cli

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// gatherTopicEvidence dispatches per-topic read-only evidence collection.
//
// It is a thin coordinator: K8s reuses the existing kubectl-based gather, while
// non-K8s topics opportunistically invoke the matching `ai-sre probe <topic> …
// --json` subprocess to capture concrete metrics, mirroring how an SRE would
// gather quick observability before asking the AI. All errors are swallowed --
// the goal is best-effort enrichment of the diagnose payload.
//
// The returned map should be merged into the diagnose request's `context`. When
// any evidence is captured (`evidence_root_cause` style), the orchestrator will
// switch the server prompt to the evidence-driven template that's already used
// for K8s and now for any topic.
func gatherTopicEvidence(ctx context.Context, topic string, flags map[string]string) (collected map[string]string) {
	collected = map[string]string{}
	t := strings.ToLower(strings.TrimSpace(topic))
	switch t {
	case "k8s", "kubernetes":
		for k, v := range gatherK8sDiagnoseEvidence(ctx, flags) {
			collected[k] = v
		}
	case "kafka":
		gatherKafkaEvidence(ctx, flags, collected)
	case "redis":
		gatherRedisEvidence(ctx, flags, collected)
	case "mysql":
		gatherMySQLEvidence(ctx, flags, collected)
	case "postgresql", "postgres":
		gatherPostgreSQLEvidence(ctx, flags, collected)
	case "nginx":
		gatherNginxEvidence(ctx, flags, collected)
	case "elasticsearch", "es":
		gatherElasticsearchEvidence(ctx, flags, collected)
	case "domain", "dns":
		gatherDomainEvidence(ctx, flags, collected)
	case "linux":
		gatherLinuxEvidence(ctx, flags, collected)
	}
	return collected
}

// hasTopicEvidence reports whether any non-flag evidence key was added.
func hasTopicEvidence(kv map[string]string) bool {
	if hasKubectlEvidence(kv) {
		return true
	}
	for k := range kv {
		if strings.HasPrefix(k, "host_") || strings.HasPrefix(k, "kafka_") ||
			strings.HasPrefix(k, "redis_") || 			strings.HasPrefix(k, "mysql_") ||
			strings.HasPrefix(k, "postgresql_") ||
			strings.HasPrefix(k, "nginx_") || strings.HasPrefix(k, "es_") ||
			strings.HasPrefix(k, "domain_") || strings.HasPrefix(k, "linux_") {
			return true
		}
	}
	return false
}

const maxBytesPerTopicEvidence = 60_000

// runSelfSubcommand executes the in-process binary with the given args and
// returns combined stdout+stderr (capped). Useful to reuse existing
// kafka/redis/mysql/nginx/es diagnose subcommands without duplicating logic.
func runSelfSubcommand(ctx context.Context, timeout time.Duration, args ...string) string {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	self := selfExecutablePath()
	if self == "" {
		return ""
	}
	cmd := exec.CommandContext(cctx, self, args...)
	out, _ := cmd.CombinedOutput()
	return truncateBytes(string(out), maxBytesPerTopicEvidence)
}

func selfExecutablePath() string {
	if v := strings.TrimSpace(progName); v != "" {
		if abs, err := exec.LookPath(v); err == nil {
			return abs
		}
	}
	if abs, err := exec.LookPath("ai-sre"); err == nil {
		return abs
	}
	return ""
}

func gatherKafkaEvidence(ctx context.Context, flags map[string]string, out map[string]string) {
	boot := strings.TrimSpace(flags["bootstrap"])
	if boot == "" {
		boot = strings.TrimSpace(flags["bootstrap_server"])
	}
	if boot == "" {
		boot = strings.TrimSpace(flags["addr"])
	}
	if boot == "" {
		boot = strings.TrimSpace(flags["kafka_bootstrap"])
	}
	if boot == "" {
		return
	}
	args := []string{"probe", "kafka", boot, "--json"}
	if t := strings.TrimSpace(flags["topic"]); t != "" {
		args = append(args, "--topic", t)
	}
	if g := strings.TrimSpace(flags["group"]); g != "" {
		args = append(args, "--group", g)
	}
	body := runSelfSubcommand(ctx, 20*time.Second, args...)
	if body != "" {
		out["kafka_diagnose_json"] = body
	}
}

func gatherRedisEvidence(ctx context.Context, flags map[string]string, out map[string]string) {
	body, _, err := collectRedisProbeJSON(ctx, flags)
	if err == errRedisAuthRequired && body != "" {
		out["redis_diagnose_json"] = body
		out["redis_auth_required"] = "true"
		return
	}
	if body != "" {
		out["redis_diagnose_json"] = body
	}
}

func gatherMySQLEvidence(ctx context.Context, flags map[string]string, out map[string]string) {
	dsn := strings.TrimSpace(flags["dsn"])
	if dsn == "" {
		return
	}
	args := []string{"probe", "mysql", dsn, "--json"}
	body := runSelfSubcommand(ctx, 25*time.Second, args...)
	if body != "" {
		out["mysql_diagnose_json"] = body
	}
}

func gatherPostgreSQLEvidence(ctx context.Context, flags map[string]string, out map[string]string) {
	dsn := strings.TrimSpace(flags["dsn"])
	if dsn == "" {
		return
	}
	args := []string{"probe", "postgresql", dsn, "--json"}
	body := runSelfSubcommand(ctx, 25*time.Second, args...)
	if body != "" {
		out["postgresql_diagnose_json"] = body
	}
}

func gatherNginxEvidence(ctx context.Context, flags map[string]string, out map[string]string) {
	logFile := strings.TrimSpace(flags["access_log"])
	if logFile == "" {
		return
	}
	args := []string{"probe", "nginx", "--access-log", logFile, "--json"}
	if u := strings.TrimSpace(flags["upstream"]); u != "" {
		args = append(args, "--upstream", u)
	}
	body := runSelfSubcommand(ctx, 25*time.Second, args...)
	if body != "" {
		out["nginx_diagnose_json"] = body
	}
}

func gatherElasticsearchEvidence(ctx context.Context, flags map[string]string, out map[string]string) {
	url := strings.TrimSpace(flags["url"])
	if url == "" {
		url = strings.TrimSpace(flags["base_url"])
	}
	if url == "" {
		url = strings.TrimSpace(flags["addr"])
	}
	if url == "" {
		return
	}
	args := []string{"probe", "elasticsearch", url, "--json"}
	body := runSelfSubcommand(ctx, 25*time.Second, args...)
	if body != "" {
		out["es_diagnose_json"] = body
	}
}
