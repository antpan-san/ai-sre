package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSockstat(t *testing.T) {
	dir := t.TempDir()
	proc := filepath.Join(dir, "proc")
	net := filepath.Join(proc, "net")
	os.MkdirAll(net, 0o755)
	os.WriteFile(filepath.Join(net, "sockstat"), []byte(`sockets: used 200
TCP: inuse 50 orphan 2 tw 120 alloc 80 mem 0
UDP: inuse 10
`), 0o644)
	t.Setenv("OPSFLEET_LINUX_PROC_ROOT", proc)
	got := parseSockstat()
	if got.SocketsUsed != 200 || got.TCPInUse != 50 || got.TCPTimeWait != 120 {
		t.Fatalf("sockstat: %+v", got)
	}
}

func TestParseTCPEstablishedFromSNMP(t *testing.T) {
	dir := t.TempDir()
	proc := filepath.Join(dir, "proc")
	net := filepath.Join(proc, "net")
	os.MkdirAll(net, 0o755)
	os.WriteFile(filepath.Join(net, "snmp"), []byte(`Tcp: RtoAlgorithm RtoMin RtoMax MaxConn CurrEstab
Tcp: 1 3 15 128 42
`), 0o644)
	t.Setenv("OPSFLEET_LINUX_PROC_ROOT", proc)
	if got := parseTCPEstablishedFromSNMP(); got != 42 {
		t.Fatalf("CurrEstab=%d want 42", got)
	}
}

func TestBuildProcessHotspotsDedupes(t *testing.T) {
	procs := []procDelta{
		{linuxPerfProcess: linuxPerfProcess{PID: 1, CPUPercent: 50, RSSBytes: 1e9, RiskScore: 3}, riskScore: 3},
		{linuxPerfProcess: linuxPerfProcess{PID: 2, CPUPercent: 5, RSSBytes: 1e8, FDCount: 600, RiskScore: 2}, riskScore: 2},
	}
	hot := buildProcessHotspots(procs, 10)
	if len(hot) != 2 {
		t.Fatalf("hotspots len=%d", len(hot))
	}
	if hot[0].PID != 1 {
		t.Fatalf("expected pid 1 first, got %d", hot[0].PID)
	}
}
