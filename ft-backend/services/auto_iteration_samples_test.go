package services

import "testing"

func TestClassifyTargetKind(t *testing.T) {
	if got := classifyTargetKind("redis", "127.0.0.1:6379"); got != "redis_single" {
		t.Fatalf("redis single=%q", got)
	}
	if got := classifyTargetKind("redis", "a:6379,b:6379"); got != "redis_cluster" {
		t.Fatalf("redis cluster=%q", got)
	}
	if got := classifyTargetKind("k8s", "pod/default/app-1"); got != "k8s_pod" {
		t.Fatalf("k8s pod=%q", got)
	}
}
