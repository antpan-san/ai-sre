package cli

import "testing"

func TestCobraFindProbeRedis(t *testing.T) {
	root := newRoot("ai-sre")
	cmd, remaining, err := root.Find([]string{"probe", "redis", "127.0.0.1:6379", "--json"})
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if cmd == nil || cmd.Name() != "redis" {
		t.Fatalf("cmd=%v want redis", cmd)
	}
	_ = remaining
}

func TestCobraFindProbeLinuxOK(t *testing.T) {
	root := newRoot("ai-sre")
	_, _, err := root.Find([]string{"probe", "linux", "-o", "json"})
	if err != nil {
		t.Fatalf("expected find ok: %v", err)
	}
}

func TestArgvHasUnresolvedSubcommandProbeLinux(t *testing.T) {
	root := newRoot("ai-sre")
	if !argvHasUnresolvedSubcommand(root, []string{"probe", "not-a-real-topic", "-o", "json"}) {
		t.Fatal("expected unresolved probe subcommand")
	}
	if argvHasUnresolvedSubcommand(root, []string{"probe", "linux", "-o", "json"}) {
		t.Fatal("probe linux should resolve")
	}
	if argvHasUnresolvedSubcommand(root, []string{"probe", "redis", "127.0.0.1:6379"}) {
		t.Fatal("probe redis should resolve")
	}
}

func TestCobraFindProbeUnknownFails(t *testing.T) {
	root := newRoot("ai-sre")
	cmd, remaining, err := root.Find([]string{"probe", "not-a-real-topic", "-o", "json"})
	if err != nil {
		t.Fatalf("cobra Find err=%v", err)
	}
	if cmd == nil || cmd.Name() != "probe" {
		t.Fatalf("cmd=%v", cmd)
	}
	if len(remaining) == 0 {
		t.Fatal("expected remaining args for unknown probe subcommand (preflight must treat as incomplete)")
	}
}
