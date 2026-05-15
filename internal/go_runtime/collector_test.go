package go_runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCollectProcSnapshotAndFindings(t *testing.T) {
	dir := t.TempDir()
	proc := filepath.Join(dir, "proc")
	cgroot := filepath.Join(dir, "cgroup")
	pidDir := filepath.Join(proc, "123")
	mustMkdir(t, filepath.Join(pidDir, "fd"))
	for _, name := range []string{"0", "1", "2", "3"} {
		mustWrite(t, filepath.Join(pidDir, "fd", name), "")
	}
	mustWrite(t, filepath.Join(pidDir, "status"), strings.Join([]string{
		"Name:\tapi",
		"State:\tS (sleeping)",
		"Threads:\t120",
		"VmRSS:\t1048576 kB",
		"VmHWM:\t2097152 kB",
		"VmSize:\t3145728 kB",
		"VmData:\t900000 kB",
	}, "\n"))
	mustWrite(t, filepath.Join(pidDir, "smaps_rollup"), strings.Join([]string{
		"Rss:\t1048576 kB",
		"Pss:\t900000 kB",
		"Anonymous:\t800000 kB",
		"Private_Clean:\t1 kB",
		"Private_Dirty:\t2 kB",
		"Shared_Clean:\t3 kB",
		"Shared_Dirty:\t4 kB",
	}, "\n"))
	mustWrite(t, filepath.Join(pidDir, "stat"), "123 (api) S 1 1 1 0 0 0 0 0 0 0 0 0 0 0 0 0 0 120 0 12345 0")
	mustWrite(t, filepath.Join(pidDir, "limits"), "Limit                     Soft Limit           Hard Limit           Units\nMax open files            4                    1024                 files\n")
	mustWrite(t, filepath.Join(pidDir, "maps"), strings.Join([]string{
		"00400000-00452000 r-xp 00000000 08:01 123 /app/api",
		"7f000000-7f010000 rw-p 00000000 00:00 0",
		"7f020000-7f030000 r--p 00000000 08:01 456 /tmp/lib.so (deleted)",
	}, "\n"))
	mustWrite(t, filepath.Join(pidDir, "cgroup"), "0::/kubepods.slice/pod123/container")
	cgDir := filepath.Join(cgroot, "kubepods.slice/pod123/container")
	mustMkdir(t, cgDir)
	mustWrite(t, filepath.Join(cgDir, "memory.current"), "950")
	mustWrite(t, filepath.Join(cgDir, "memory.max"), "1000")
	mustWrite(t, filepath.Join(cgDir, "cpu.stat"), "usage_usec 100\nnr_throttled 2\nthrottled_usec 30\n")

	report, err := Collect(Options{
		PID:        123,
		ProcRoot:   proc,
		CgroupRoot: cgroot,
		Now:        time.Unix(100, 0),
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Target.Comm != "api" {
		t.Fatalf("comm=%q", report.Target.Comm)
	}
	if report.Snapshot.Status.Threads != 120 {
		t.Fatalf("threads=%d", report.Snapshot.Status.Threads)
	}
	if report.Snapshot.FD.Open != 4 {
		t.Fatalf("fd=%d", report.Snapshot.FD.Open)
	}
	if report.Cgroup.Version != "v2" || report.Cgroup.MemoryCurrentBytes != 950 {
		t.Fatalf("cgroup=%+v", report.Cgroup)
	}
	if len(report.Findings) < 3 {
		t.Fatalf("expected findings, got %+v", report.Findings)
	}
}

func TestWriteJSONAndText(t *testing.T) {
	r := &Report{
		GeneratedAt: time.Unix(1, 0),
		Target:      ProcessIdentity{PID: 1, Comm: "api"},
		Snapshot:    ProcSnapshot{FD: FDSummary{Open: 1}, Status: ProcStatus{Threads: 2, VmRSSBytes: 10}},
		Findings:    []Finding{{Severity: "info", Title: "ok", Evidence: "e", Cause: "c", Verify: "v"}},
	}
	var text strings.Builder
	if err := WriteText(&text, r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text.String(), "Go Runtime") {
		t.Fatalf("text report missing title: %s", text.String())
	}
	var js strings.Builder
	if err := WriteJSON(&js, r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(js.String(), `"pid": 1`) {
		t.Fatalf("json report missing pid: %s", js.String())
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}
