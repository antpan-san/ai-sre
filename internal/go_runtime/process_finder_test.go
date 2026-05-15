package go_runtime

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestParsePodTarget(t *testing.T) {
	cases := []struct {
		in        string
		namespace string
		pod       string
		container string
	}{
		{"api-0", "", "api-0", ""},
		{"prod/api-0", "prod", "api-0", ""},
		{"prod/api-0/app", "prod", "api-0", "app"},
	}
	for _, tc := range cases {
		ref, err := ParsePodTarget(tc.in)
		if err != nil {
			t.Fatalf("%s: %v", tc.in, err)
		}
		if ref.Namespace != tc.namespace || ref.Pod != tc.pod || ref.Container != tc.container {
			t.Fatalf("%s: got %+v", tc.in, ref)
		}
	}
}

func TestFindProcessByNamePrefersGoAndRSS(t *testing.T) {
	root := filepath.Join(t.TempDir(), "proc")
	writeProcessFixture(t, root, 100, "api", "/usr/bin/api", "api --serve", 100*MiB)
	writeProcessFixture(t, root, 200, "api", filepath.Join(os.Args[0]), "api --serve", 80*MiB)
	writeProcessFixture(t, root, 300, "api", filepath.Join(os.Args[0]), "api --serve", 200*MiB)
	selected, candidates, err := FindProcessByName(root, "api")
	if err != nil {
		t.Fatal(err)
	}
	if selected.PID != 300 {
		t.Fatalf("selected pid=%d candidates=%+v", selected.PID, candidates)
	}
	if !selected.IsGo {
		t.Fatalf("expected Go candidate: %+v", selected)
	}
}

func writeProcessFixture(t *testing.T, root string, pid int, name, exe, cmdline string, rss uint64) {
	t.Helper()
	dir := filepath.Join(root, intString(pid))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	status := "Name:\t" + name + "\nVmRSS:\t" + intString(int(rss/1024)) + " kB\n"
	if err := os.WriteFile(filepath.Join(dir, "status"), []byte(status), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cmdline"), []byte(cmdline), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(exe, filepath.Join(dir, "exe")); err != nil {
		t.Fatal(err)
	}
}

func intString(v int) string {
	return strconv.Itoa(v)
}
