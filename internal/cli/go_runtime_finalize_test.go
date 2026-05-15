package cli

import "testing"

func TestParseAIDiagnosis(t *testing.T) {
	rc, ev := parseAIDiagnosis("根因: 采集器镜像拉取失败\n证据: ImagePullBackOff busybox:1.36")
	if rc == "" || ev == "" {
		t.Fatalf("parse failed: %q %q", rc, ev)
	}
}
