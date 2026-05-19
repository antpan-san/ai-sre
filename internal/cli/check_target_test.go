package cli

import "testing"

func TestApplyCheckTargetContextRedisDefault(t *testing.T) {
	ctx := map[string]string{}
	applyCheckTargetContext(ctx, "redis", []string{"redis"})
	if ctx["addr"] != "127.0.0.1:6379" {
		t.Fatalf("addr=%q", ctx["addr"])
	}
	if ctx["target"] != "127.0.0.1:6379" {
		t.Fatalf("target=%q", ctx["target"])
	}
}

func TestApplyCheckTargetContextRedisExplicit(t *testing.T) {
	ctx := map[string]string{}
	applyCheckTargetContext(ctx, "redis", []string{"redis", "192.168.56.11:6379"})
	if ctx["addr"] != "192.168.56.11:6379" {
		t.Fatalf("addr=%q", ctx["addr"])
	}
}

func TestApplyCheckTargetContextRedisHostOnly(t *testing.T) {
	ctx := map[string]string{}
	applyCheckTargetContext(ctx, "redis", []string{"redis", "10.0.0.5"})
	if ctx["addr"] != "10.0.0.5:6379" {
		t.Fatalf("addr=%q", ctx["addr"])
	}
}

func TestApplyCheckTargetContextDoesNotOverrideSet(t *testing.T) {
	ctx := map[string]string{"addr": "1.2.3.4:6379"}
	applyCheckTargetContext(ctx, "redis", []string{"redis", "9.9.9.9:6379"})
	if ctx["addr"] != "1.2.3.4:6379" {
		t.Fatalf("expected -d to win, got %q", ctx["addr"])
	}
}

func TestNormalizeCheckTargetValueElasticsearch(t *testing.T) {
	if got := normalizeCheckTargetValue("elasticsearch", "127.0.0.1:9200"); got != "http://127.0.0.1:9200" {
		t.Fatalf("got %q", got)
	}
}

func TestCheckTopicAcceptsOptionalTarget(t *testing.T) {
	if !checkTopicAcceptsOptionalTarget("redis") {
		t.Fatal("redis should accept target")
	}
	if checkTopicAcceptsOptionalTarget("k8s") {
		t.Fatal("k8s should not use positional target")
	}
}
