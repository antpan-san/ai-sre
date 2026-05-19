package cli

import "testing"

func TestAllowedCLIAISreDiagnosticCommand_probeAndCheck(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		want bool
	}{
		{name: "expert probe redis", argv: []string{"ai-sre", "expert", "probe", "redis", "127.0.0.1:6379", "--json"}, want: true},
		{name: "expert probe linux", argv: []string{"ai-sre", "expert", "probe", "linux", "--json", "--duration", "3s"}, want: true},
		{name: "check go target", argv: []string{"ai-sre", "check", "go", "pid/1234", "--json"}, want: true},
		{name: "check redis", argv: []string{"ai-sre", "check", "redis", "127.0.0.1:6379"}, want: true},
		{name: "reject analyze", argv: []string{"ai-sre", "analyze", "kafka", "--lag", "1"}, want: false},
		{name: "reject legacy k8s install path", argv: []string{"ai-sre", "k8s", "install", "ref"}, want: false},
		{name: "ops k8s install", argv: []string{"ai-sre", "ops", "k8s", "install", "ref"}, want: true},
		{name: "reject shell", argv: []string{"ai-sre", "expert", "probe", "redis", "127.0.0.1:6379", ";rm"}, want: false},
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
	if err := cmd.Args(cmd, []string{"k8s", "pod/default/api-0"}); err != nil {
		t.Fatalf("check k8s with pod target: %v", err)
	}
	if err := cmd.Args(cmd, []string{"go", "pid/1234"}); err != nil {
		t.Fatalf("check go pid target: %v", err)
	}
	if err := cmd.Args(cmd, []string{"unknown-topic"}); err == nil {
		t.Fatal("unknown topic should fail")
	}
}

func TestPublicCommandsRegistered(t *testing.T) {
	root := newRoot("ai-sre")
	for _, name := range []string{"check", "ops", "expert", "doctor", "upgrade", "version"} {
		if _, _, err := root.Find([]string{name}); err != nil {
			t.Fatalf("missing public command %q: %v", name, err)
		}
	}
	for _, legacy := range []string{"analyze", "diagnose", "probe", "k8s", "ask", "skills", "kafka", "redis"} {
		if _, _, err := root.Find([]string{legacy}); err == nil {
			t.Fatalf("legacy command %q should be removed", legacy)
		}
	}
	probe, _, err := root.Find([]string{"expert", "probe", "kafka"})
	if err != nil || probe == nil {
		t.Fatalf("expert probe kafka missing: %v", err)
	}
}

func TestNormalizeCheckTopicAlias(t *testing.T) {
	cases := map[string]string{
		"postgres": "postgresql",
		"es":       "elasticsearch",
		"dns":      "domain",
		"host":     "linux",
	}
	for in, want := range cases {
		if got := normalizeCheckTopicAlias(in); got != want {
			t.Fatalf("normalizeCheckTopicAlias(%q)=%q want %q", in, got, want)
		}
	}
}
