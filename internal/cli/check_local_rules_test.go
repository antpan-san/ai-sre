package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTryLocalRedisHealthyIdleRule(t *testing.T) {
	report := RedisProbeReport{
		Address: "127.0.0.1:6379",
		Memory: map[string]any{
			"used_memory":             "889960",
			"used_memory_human":       "869.10K",
			"maxmemory":               "536870912",
			"mem_fragmentation_ratio": "16.03",
			"allocator_frag_ratio":    "2.17",
			"evicted_keys":            "0",
		},
		Clients: map[string]any{
			"connected_clients":          "1",
			"rejected_connections":       "0",
			"total_connections_received": "5",
		},
		Stats: map[string]any{
			"instantaneous_ops_per_sec": "0",
			"total_commands_processed":  "35",
			"total_error_replies":       "0",
			"uptime_in_seconds":         "876",
		},
		Findings: []string{"未发现明显高优先级 Redis 异常"},
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	ctx := map[string]string{"redis_diagnose_json": string(raw)}
	diag, ok := tryLocalRedisRules(ctx)
	if !ok || diag == nil {
		t.Fatal("expected local rule hit for healthy idle redis")
	}
	if !strings.EqualFold(diag.Source, "local-rule") {
		t.Fatalf("source=%q", diag.Source)
	}
	if diag.SkillName != "redis-healthy-idle" {
		t.Fatalf("skill=%q", diag.SkillName)
	}
	if !strings.Contains(diag.Answer, "【是否调用 AI】") || strings.Contains(diag.Answer, "【是否调用 AI】\n是") {
		t.Fatalf("expected UsedAI=false in answer, got %q", diag.Answer)
	}
	if !strings.Contains(diag.Answer, "mem_fragmentation_ratio=16.03") {
		t.Fatalf("expected fragmentation evidence, got %q", diag.Answer)
	}
}

func TestTryLocalRedisHealthyRuleSkipsWhenRejected(t *testing.T) {
	report := RedisProbeReport{
		Memory: map[string]any{
			"used_memory":             "889960",
			"mem_fragmentation_ratio": "16.03",
			"evicted_keys":            "0",
		},
		Clients: map[string]any{
			"rejected_connections": "3",
		},
		Stats: map[string]any{
			"total_error_replies": "0",
		},
		Findings: []string{"rejected_connections=3"},
	}
	raw, _ := json.Marshal(report)
	ctx := map[string]string{"redis_diagnose_json": string(raw)}
	_, ok := tryLocalRedisRules(ctx)
	if !ok {
		t.Fatal("expected maxclients/rejected local rule")
	}
}

func TestTryLocalRedisHealthyRuleSkipsWithoutEvidence(t *testing.T) {
	ctx := map[string]string{"redis_diagnose_json": `{"findings":["未发现明显高优先级 Redis 异常"]}`}
	if _, ok := tryLocalRedisRules(ctx); ok {
		t.Fatal("expected no rule without used_memory")
	}
}
