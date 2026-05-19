package cli

import "testing"

func TestAllowedCLIAISreDiagnosticCommand_probeAndCheck(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		want bool
	}{
		{name: "probe redis", argv: []string{"ai-sre", "probe", "redis", "127.0.0.1:6379", "--json"}, want: true},
		{name: "probe nginx", argv: []string{"ai-sre", "probe", "nginx", "--json", "--access-log", "/var/log/nginx/access.log"}, want: true},
		{name: "check go", argv: []string{"ai-sre", "check", "go", "--json", "--pod", "prod/api-0"}, want: true},
		{name: "legacy kafka diagnose", argv: []string{"ai-sre", "kafka", "diagnose", "b1:9092", "--json"}, want: true},
		{name: "reject analyze", argv: []string{"ai-sre", "analyze", "kafka", "--lag", "1"}, want: false},
		{name: "reject shell", argv: []string{"ai-sre", "probe", "redis", "127.0.0.1:6379", ";rm"}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := allowedCLIAISreDiagnosticCommand(tc.argv); got != tc.want {
				t.Fatalf("allowedCLIAISreDiagnosticCommand(%v)=%v want %v", tc.argv, got, tc.want)
			}
		})
	}
}

func TestCheckTopicOptionalTargetArgs(t *testing.T) {
	root := newRoot("ai-sre")
	cmd, _, err := root.Find([]string{"check"})
	if err != nil {
		t.Fatalf("find check: %v", err)
	}
	if err := cmd.Args(cmd, []string{"domain", "opsfleetpilot.com"}); err != nil {
		t.Fatalf("check domain args: %v", err)
	}
	if err := cmd.Args(cmd, []string{"redis", "127.0.0.1:6379"}); err != nil {
		t.Fatalf("check redis with target: %v", err)
	}
	if err := cmd.Args(cmd, []string{"k8s", "extra"}); err == nil {
		t.Fatal("check k8s with positional target should fail")
	}
}

func TestCheckAndAnalyzeCommandsRegistered(t *testing.T) {
	root := newRoot("ai-sre")
	check, _, err := root.Find([]string{"check"})
	if err != nil || check == nil {
		t.Fatalf("check command missing: %v", err)
	}
	probe, _, err := root.Find([]string{"probe", "kafka"})
	if err != nil || probe == nil {
		t.Fatalf("probe kafka missing: %v", err)
	}
	analyze, _, err := root.Find([]string{"analyze"})
	if err != nil || analyze == nil {
		t.Fatalf("analyze alias missing: %v", err)
	}
	if analyze.Deprecated == "" {
		t.Fatal("analyze should be deprecated")
	}
}
