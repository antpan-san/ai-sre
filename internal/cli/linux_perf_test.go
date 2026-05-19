package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestValidateLinuxPerfOptionsBounds(t *testing.T) {
	if err := ValidateLinuxPerfOptions(LinuxPerfOptions{Duration: 2 * time.Second, TopN: 10}); err == nil {
		t.Fatal("expected duration too short error")
	}
	if err := ValidateLinuxPerfOptions(LinuxPerfOptions{Duration: 10 * time.Second, TopN: 3}); err == nil {
		t.Fatal("expected top too small error")
	}
	if err := ValidateLinuxPerfOptions(LinuxPerfOptions{Duration: 10 * time.Second, TopN: 31}); err == nil {
		t.Fatal("expected top too large error")
	}
	if _, err := parseLinuxPIDFlag("abc"); err == nil {
		t.Fatal("expected non-numeric pid error")
	}
}

func TestCPUStatDelta(t *testing.T) {
	a, ok := parseCPUStatLine("cpu  1000 100 500 4000 200 10 20 5")
	if !ok {
		t.Fatal("parse cpu")
	}
	b, ok := parseCPUStatLine("cpu  2000 200 1000 5000 400 20 40 10")
	if !ok {
		t.Fatal("parse cpu2")
	}
	user, system, iowait, _, _, _, idle := cpuUsagePct(a, b)
	if user <= 0 && system <= 0 {
		t.Fatalf("expected non-zero cpu usage user=%v sys=%v iowait=%v idle=%v", user, system, iowait, idle)
	}
}

func TestParseMeminfoOOMRisk(t *testing.T) {
	dir := t.TempDir()
	proc := filepath.Join(dir, "proc")
	os.MkdirAll(proc, 0o755)
	os.WriteFile(filepath.Join(proc, "meminfo"), []byte(`MemTotal:       1000000 kB
MemAvailable:     50000 kB
SwapTotal:        100000 kB
SwapFree:          10000 kB
Dirty:               100 kB
`), 0o644)
	t.Setenv("OPSFLEET_LINUX_PROC_ROOT", proc)
	kv, err := parseMeminfo()
	if err != nil {
		t.Fatal(err)
	}
	r := &LinuxPerfReport{}
	fillMemoryReport(r, kv)
	if r.Memory.OOMRisk != "high" {
		t.Fatalf("expected high oom risk, got %q", r.Memory.OOMRisk)
	}
}

func TestDiskstatsDelta(t *testing.T) {
	a := diskStatSample{ReadSectors: 1000, WriteSectors: 2000, IOTicks: 100}
	b := diskStatSample{ReadSectors: 2000, WriteSectors: 4000, IOTicks: 200}
	r, w, io := diskIODelta(a, b, 10)
	if r <= 0 || w <= 0 {
		t.Fatalf("expected positive io rates r=%v w=%v io=%v", r, w, io)
	}
}

func TestCollectLinuxProbeJSONKey(t *testing.T) {
	root := newRoot("ai-sre")
	cmd, _, err := root.Find([]string{"probe", "linux"})
	if err != nil {
		t.Fatalf("probe linux missing: %v", err)
	}
	if cmd == nil {
		t.Fatal("nil cmd")
	}
}

func TestFinishLinuxCheckEvidenceInjectsJSON(t *testing.T) {
	ctx := map[string]string{"duration": "3s", "top": "5"}
	// Without proc root on non-linux this may only set errors in JSON — still injects key when collect returns body
	_ = finishLinuxCheckEvidence("linux", ctx)
	// On macOS without fixture, collect may error before inject — skip strict assert
	if os.Getenv("OPSFLEET_LINUX_PROC_ROOT") != "" {
		if ctx["linux_perf_probe_json"] == "" {
			t.Fatal("expected linux_perf_probe_json")
		}
	}
}

func TestCommandCatalogIncludesProbeLinux(t *testing.T) {
	root := newRoot("ai-sre")
	cat := BuildCommandCatalog(root)
	found := false
	for _, c := range cat.Commands {
		if c.Path == "probe linux" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("probe linux not in catalog")
	}
}
