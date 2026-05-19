package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyCheckTargetContextRedisDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("OPSFLEET_API_URL", "")
	autoBindingWarn = ""
	autoBindingWarnShown = false
	ctx := map[string]string{}
	applyCheckTargetContext(ctx, "redis", []string{"redis"})
	if ctx["addr"] != "127.0.0.1:6379" {
		t.Fatalf("addr=%q", ctx["addr"])
	}
	if ctx["target"] != "127.0.0.1:6379" {
		t.Fatalf("target=%q", ctx["target"])
	}
}

func TestApplyCheckTargetContextRedisSmartDefaultFromInstall(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	cfg := filepath.Join(dir, ".config", "ai-sre")
	if err := os.MkdirAll(cfg, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "opsfleet_api_url"), []byte(EmbeddedOpsfleetAPIBase+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	autoBindingWarn = ""
	autoBindingWarnShown = false
	ctx := map[string]string{}
	applyCheckTargetContext(ctx, "redis", []string{"redis"})
	if ctx["addr"] != "192.168.56.11:6379" {
		t.Fatalf("addr=%q", ctx["addr"])
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
