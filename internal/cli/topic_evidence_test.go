package cli

import (
	"testing"
)

func TestHasTopicEvidence(t *testing.T) {
	if hasTopicEvidence(map[string]string{"pod": "pending"}) {
		t.Fatalf("plain user flags should not count as evidence")
	}
	if !hasTopicEvidence(map[string]string{"kafka_diagnose_json": "{}"}) {
		t.Fatalf("kafka_* should count")
	}
	if !hasTopicEvidence(map[string]string{"redis_diagnose_json": "{}"}) {
		t.Fatalf("redis_* should count")
	}
	if !hasTopicEvidence(map[string]string{"mysql_diagnose_json": "{}"}) {
		t.Fatalf("mysql_* should count")
	}
	if !hasTopicEvidence(map[string]string{"nginx_diagnose_json": "{}"}) {
		t.Fatalf("nginx_* should count")
	}
	if !hasTopicEvidence(map[string]string{"es_diagnose_json": "{}"}) {
		t.Fatalf("es_* should count")
	}
	if !hasTopicEvidence(map[string]string{"kubectl_nodes": "..."}) {
		t.Fatalf("kubectl_* should count")
	}
	if !hasTopicEvidence(map[string]string{"host_uptime": "10d"}) {
		t.Fatalf("host_* should count")
	}
}

func TestGatherTopicEvidenceNoFlagsNoEvidence(t *testing.T) {
	// kafka/redis/mysql/nginx/es helpers all bail out early when the required
	// flag is missing -- they must not invoke the binary in that case.
	cases := []string{"kafka", "redis", "mysql", "nginx", "elasticsearch", "es"}
	for _, topic := range cases {
		out := gatherTopicEvidence(nil, topic, map[string]string{})
		if len(out) != 0 {
			t.Errorf("topic=%s: expected empty evidence when no flags supplied, got %v", topic, out)
		}
	}
	// Unknown topics return empty map, never nil panic.
	if out := gatherTopicEvidence(nil, "unknown", nil); len(out) != 0 {
		t.Errorf("unknown topic should be empty, got %v", out)
	}
}
