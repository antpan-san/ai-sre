package cli

import "time"

// LinuxPerfReport is JSON output for probe linux.
type LinuxPerfReport struct {
	Topic          string                 `json:"topic"`
	Sample         linuxPerfSample        `json:"sample"`
	Host           linuxPerfHost          `json:"host"`
	CPU            linuxPerfCPU           `json:"cpu"`
	Load           linuxPerfLoad          `json:"load"`
	Memory         linuxPerfMemory        `json:"memory"`
	Swap           linuxPerfSwap          `json:"swap"`
	Disks          []linuxPerfDisk        `json:"disks"`
	DiskIO         []linuxPerfDiskIO      `json:"disk_io"`
	PSI            map[string]any         `json:"psi"`
	Network        linuxPerfNetwork       `json:"network"`
	Connections    linuxPerfConnections   `json:"connections"`
	System         linuxPerfSystem        `json:"system"`
	ProcessHotspots []linuxPerfProcess    `json:"process_hotspots,omitempty"`
	LeakRisks      []linuxPerfLeakRisk    `json:"leak_risks"`
	KernelSignals  []string               `json:"kernel_signals"`
	Findings       []string               `json:"findings"`
	EvidenceCompleteness map[string]bool `json:"evidence_completeness"`
	Errors         []string               `json:"errors"`
}

type linuxPerfSample struct {
	DurationSeconds float64   `json:"duration_seconds"`
	StartedAt       time.Time `json:"started_at"`
	EndedAt         time.Time `json:"ended_at"`
	TopN            int       `json:"top_n"`
	TargetPID       int       `json:"target_pid,omitempty"`
}

type linuxPerfHost struct {
	Hostname      string  `json:"hostname"`
	Kernel        string  `json:"kernel,omitempty"`
	UptimeSeconds float64 `json:"uptime_seconds,omitempty"`
	CPUCores      int     `json:"cpu_cores"`
}

type linuxPerfCPU struct {
	Cores        int     `json:"cpu_cores"`
	UserPct      float64 `json:"user_pct,omitempty"`
	SystemPct    float64 `json:"system_pct,omitempty"`
	IowaitPct    float64 `json:"iowait_pct,omitempty"`
	StealPct     float64 `json:"steal_pct,omitempty"`
	IrqPct       float64 `json:"irq_pct,omitempty"`
	SoftirqPct   float64 `json:"softirq_pct,omitempty"`
	IdlePct      float64 `json:"idle_pct,omitempty"`
	LoadPerCore1 float64 `json:"load_per_core_1,omitempty"`
}

type linuxPerfLoad struct {
	Load1      float64 `json:"load1"`
	Load5      float64 `json:"load5"`
	Load15     float64 `json:"load15"`
	Running    int     `json:"running,omitempty"`
	TotalTasks int     `json:"total_tasks,omitempty"`
}

type linuxPerfMemory struct {
	MemTotalKB      int64   `json:"mem_total_kb"`
	MemAvailableKB  int64   `json:"mem_available_kb"`
	DirtyKB         int64   `json:"dirty_kb,omitempty"`
	WritebackKB     int64   `json:"writeback_kb,omitempty"`
	SlabKB          int64   `json:"slab_kb,omitempty"`
	SReclaimableKB  int64   `json:"sreclaimable_kb,omitempty"`
	SUnreclaimKB    int64   `json:"sunreclaim_kb,omitempty"`
	UsedPct         float64 `json:"used_pct,omitempty"`
	OOMRisk         string  `json:"oom_risk,omitempty"`
}

type linuxPerfSwap struct {
	SwapTotalKB int64 `json:"swap_total_kb"`
	SwapFreeKB  int64 `json:"swap_free_kb"`
}

type linuxPerfDisk struct {
	Mount      string  `json:"mount"`
	FSType     string  `json:"fs_type,omitempty"`
	TotalBytes int64   `json:"total_bytes"`
	UsedBytes  int64   `json:"used_bytes"`
	UsedPct    float64 `json:"used_pct"`
	InodeUsedPct float64 `json:"inode_used_pct,omitempty"`
	PseudoFS   bool    `json:"pseudo_fs,omitempty"`
}

type linuxPerfDiskIO struct {
	Device         string  `json:"device"`
	ReadBytesPerSec  float64 `json:"read_bytes_per_sec"`
	WriteBytesPerSec float64 `json:"write_bytes_per_sec"`
	IOTimePct      float64 `json:"io_time_pct,omitempty"`
	WeightedIOTime float64 `json:"weighted_io_time,omitempty"`
}

type linuxPerfNetwork struct {
	TotalRxBps   float64              `json:"total_rx_bps,omitempty"`
	TotalTxBps   float64              `json:"total_tx_bps,omitempty"`
	TotalRxErr   uint64               `json:"total_rx_errors,omitempty"`
	TotalTxErr   uint64               `json:"total_tx_errors,omitempty"`
	TotalRxDrop  uint64               `json:"total_rx_dropped,omitempty"`
	Interfaces   []linuxPerfNetIface  `json:"interfaces,omitempty"`
}

type linuxPerfNetIface struct {
	Name    string  `json:"name"`
	RxBps   float64 `json:"rx_bps,omitempty"`
	TxBps   float64 `json:"tx_bps,omitempty"`
	RxErr   uint64  `json:"rx_errors,omitempty"`
	TxErr   uint64  `json:"tx_errors,omitempty"`
}

type linuxPerfConnections struct {
	SocketsUsed    int `json:"sockets_used,omitempty"`
	TCPInUse       int `json:"tcp_inuse,omitempty"`
	TCPEstablished int `json:"tcp_established,omitempty"`
	TCPTimeWait    int `json:"tcp_time_wait,omitempty"`
	TCPCloseWait   int `json:"tcp_close_wait,omitempty"`
	TCPSynRecv     int `json:"tcp_syn_recv,omitempty"`
	TCPOrphan      int `json:"tcp_orphan,omitempty"`
	TCPAlloc       int `json:"tcp_alloc,omitempty"`
	UDPInUse       int `json:"udp_inuse,omitempty"`
}

type linuxPerfSystem struct {
	ProcessCount int   `json:"process_count,omitempty"`
	ThreadCount  int   `json:"thread_count,omitempty"`
	OpenFiles    int64 `json:"open_files,omitempty"`
	MaxOpenFiles int64 `json:"max_open_files,omitempty"`
}

type linuxPerfProcess struct {
	PID         int     `json:"pid"`
	PPID        int     `json:"ppid,omitempty"`
	User        string  `json:"user,omitempty"`
	Comm        string  `json:"comm,omitempty"`
	Cmdline     string  `json:"cmdline,omitempty"`
	State       string  `json:"state,omitempty"`
	Threads     int     `json:"threads,omitempty"`
	FDCount     int     `json:"fd_count,omitempty"`
	CPUPercent  float64 `json:"cpu_percent,omitempty"`
	RSSBytes    int64   `json:"rss_bytes,omitempty"`
	VMSBytes    int64   `json:"vms_bytes,omitempty"`
	ReadBps     float64 `json:"read_bps,omitempty"`
	WriteBps    float64 `json:"write_bps,omitempty"`
	OOMScore    int     `json:"oom_score,omitempty"`
	UptimeSec   float64 `json:"uptime_sec,omitempty"`
	Cgroup      string  `json:"cgroup,omitempty"`
	RiskScore   int     `json:"risk_score,omitempty"`
	RiskReason  string  `json:"risk_reason,omitempty"`
}

type linuxPerfLeakRisk struct {
	PID        int      `json:"pid"`
	Comm       string   `json:"comm,omitempty"`
	Signals    []string `json:"signals"`
	Severity   string   `json:"severity"`
}

// LinuxPerfOptions configures probe/check linux collection.
type LinuxPerfOptions struct {
	Duration time.Duration
	TopN     int
	PID      int
	JSON     bool
}
