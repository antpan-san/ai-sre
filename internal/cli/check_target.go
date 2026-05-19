package cli

import (
	"fmt"
	"os"
	"strings"
)

// checkTargetSpec defines how check <topic> resolves a connection target without -d flags.
type checkTargetSpec struct {
	Default    string
	EnvKeys    []string
	PrimaryKey string // canonical context key for server diagnostic plan
	AliasKeys  []string // extra keys for local probe / legacy -d usage
}

var checkTargetSpecs = map[string]checkTargetSpec{
	"redis": {
		Default:    "127.0.0.1:6379",
		EnvKeys:    []string{"AI_SRE_REDIS_ADDR", "REDIS_ADDR"},
		PrimaryKey: "addr",
		AliasKeys:  []string{"target"},
	},
	"kafka": {
		Default:    "127.0.0.1:9092",
		EnvKeys:    []string{"AI_SRE_KAFKA_BOOTSTRAP", "KAFKA_BOOTSTRAP_SERVERS", "KAFKA_BOOTSTRAP"},
		PrimaryKey: "bootstrap",
		AliasKeys:  []string{"bootstrap_server", "addr"},
	},
	"mysql": {
		Default:    "root@tcp(127.0.0.1:3306)/",
		EnvKeys:    []string{"AI_SRE_MYSQL_DSN", "MYSQL_DSN"},
		PrimaryKey: "dsn",
	},
	"postgresql": {
		Default:    "postgres://127.0.0.1:5432/postgres?sslmode=disable",
		EnvKeys:    []string{"AI_SRE_POSTGRES_DSN", "POSTGRES_DSN", "DATABASE_URL"},
		PrimaryKey: "dsn",
	},
	"elasticsearch": {
		Default:    "http://127.0.0.1:9200",
		EnvKeys:    []string{"AI_SRE_ELASTICSEARCH_URL", "ELASTICSEARCH_URL"},
		PrimaryKey: "url",
		AliasKeys:  []string{"base_url", "addr"},
	},
	"es": {
		Default:    "http://127.0.0.1:9200",
		EnvKeys:    []string{"AI_SRE_ELASTICSEARCH_URL", "ELASTICSEARCH_URL"},
		PrimaryKey: "url",
		AliasKeys:  []string{"base_url", "addr"},
	},
}

func normalizeCheckTopic(topic string) string {
	t := strings.ToLower(strings.TrimSpace(topic))
	if t == "postgres" {
		return "postgresql"
	}
	if t == "es" {
		return "elasticsearch"
	}
	return t
}

func checkTopicAcceptsOptionalTarget(topic string) bool {
	_, ok := checkTargetSpecs[normalizeCheckTopic(topic)]
	if ok {
		return true
	}
	return isDomainTopic(topic)
}

func applyCheckTargetContext(ctx map[string]string, topic string, args []string) {
	if ctx == nil {
		return
	}
	explicit := ""
	if len(args) >= 2 {
		explicit = strings.TrimSpace(args[1])
	}
	t := normalizeCheckTopic(topic)
	spec, ok := checkTargetSpecs[t]
	if !ok {
		return
	}
	target := explicit
	if target == "" {
		// -d / --set values in ctx take precedence over env and smart defaults.
		if v := strings.TrimSpace(ctx[spec.PrimaryKey]); v != "" {
			target = v
		}
		for _, k := range spec.AliasKeys {
			if v := strings.TrimSpace(ctx[k]); v != "" {
				target = v
				break
			}
		}
	}
	if target == "" {
		target = resolveCheckTargetFromEnv(spec.EnvKeys)
	}
	if target == "" {
		target = smartDefaultCheckTarget(t)
	}
	if target == "" {
		target = spec.Default
	}
	if target == "" {
		return
	}
	target = normalizeCheckTargetValue(t, target)
	setCheckContextKey(ctx, spec.PrimaryKey, target)
	for _, k := range spec.AliasKeys {
		setCheckContextKey(ctx, k, target)
	}
}

func setCheckContextKey(ctx map[string]string, key, value string) {
	if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
		return
	}
	if strings.TrimSpace(ctx[key]) != "" {
		return
	}
	ctx[key] = value
}

// smartDefaultCheckTarget prefers local middleware on the current host, then OpsFleet install host.
func smartDefaultCheckTarget(topic string) string {
	if local := localhostMiddlewareTarget(topic); local != "" {
		return local
	}
	return opsfleetHostMiddlewareTarget(topic)
}

func localhostMiddlewareTarget(topic string) string {
	switch topic {
	case "redis":
		return "127.0.0.1:6379"
	case "kafka":
		return "127.0.0.1:9092"
	case "mysql":
		return "root@tcp(127.0.0.1:3306)/"
	case "postgresql":
		return "postgres://127.0.0.1:5432/postgres?sslmode=disable"
	case "elasticsearch":
		return "http://127.0.0.1:9200"
	default:
		return ""
	}
}

func opsfleetHostMiddlewareTarget(topic string) string {
	host := opsfleetConsoleHost()
	if host == "" {
		return ""
	}
	switch topic {
	case "redis":
		return host + ":6379"
	case "kafka":
		return host + ":9092"
	case "mysql":
		return fmt.Sprintf("root@tcp(%s:3306)/", host)
	case "postgresql":
		return fmt.Sprintf("postgres://%s:5432/postgres?sslmode=disable", host)
	case "elasticsearch":
		return "http://" + host + ":9200"
	default:
		return ""
	}
}

func resolveCheckTargetFromEnv(keys []string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func normalizeCheckTargetValue(topic, target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	switch topic {
	case "redis", "kafka":
		if !strings.Contains(target, ":") {
			port := "6379"
			if topic == "kafka" {
				port = "9092"
			}
			return target + ":" + port
		}
	case "elasticsearch":
		if !strings.Contains(target, "://") && strings.Contains(target, ":") {
			return "http://" + target
		}
		if !strings.Contains(target, "://") && !strings.Contains(target, ":") {
			return "http://" + target + ":9200"
		}
	}
	return target
}

func validateCheckTargetLiteral(target string) error {
	if target == "" {
		return nil
	}
	if !diagnosticAISreValueRe.MatchString(target) {
		return fmt.Errorf("非法目标 %q：仅允许字母数字与 _./:-", target)
	}
	return nil
}
