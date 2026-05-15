package go_runtime

import "time"

type Options struct {
	PID        int
	Namespace  string
	Pod        string
	Container  string
	ProcRoot   string
	CgroupRoot string
	Now        time.Time
}

type ProcessIdentity struct {
	PID       int    `json:"pid"`
	Comm      string `json:"comm,omitempty"`
	State     string `json:"state,omitempty"`
	Exe       string `json:"exe,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Pod       string `json:"pod,omitempty"`
	Container string `json:"container,omitempty"`
}

type ProcSnapshot struct {
	Status      ProcStatus  `json:"status"`
	SmapsRollup SmapsRollup `json:"smaps_rollup"`
	Stat        ProcStat    `json:"stat"`
	Limits      ProcLimits  `json:"limits"`
	FD          FDSummary   `json:"fd"`
	Maps        MapsSummary `json:"maps"`
	Cgroups     []CgroupRef `json:"cgroups,omitempty"`
}

type ProcStatus struct {
	Name        string `json:"name,omitempty"`
	State       string `json:"state,omitempty"`
	Threads     int    `json:"threads,omitempty"`
	VmRSSBytes  uint64 `json:"vm_rss_bytes,omitempty"`
	VmHWMBytes  uint64 `json:"vm_hwm_bytes,omitempty"`
	VmSizeBytes uint64 `json:"vm_size_bytes,omitempty"`
	VmDataBytes uint64 `json:"vm_data_bytes,omitempty"`
	VmStkBytes  uint64 `json:"vm_stk_bytes,omitempty"`
	VmExeBytes  uint64 `json:"vm_exe_bytes,omitempty"`
}

type SmapsRollup struct {
	RSSBytes       uint64 `json:"rss_bytes,omitempty"`
	PSSBytes       uint64 `json:"pss_bytes,omitempty"`
	AnonymousBytes uint64 `json:"anonymous_bytes,omitempty"`
	PrivateBytes   uint64 `json:"private_bytes,omitempty"`
	SharedBytes    uint64 `json:"shared_bytes,omitempty"`
}

type ProcStat struct {
	Comm       string `json:"comm,omitempty"`
	State      string `json:"state,omitempty"`
	NumThreads int    `json:"num_threads,omitempty"`
	StartTime  uint64 `json:"start_time,omitempty"`
	// UtimeTicks and StimeTicks are jiffies from /proc/<pid>/stat (fields 14–15).
	UtimeTicks uint64 `json:"utime_ticks,omitempty"`
	StimeTicks uint64 `json:"stime_ticks,omitempty"`
}

type ProcLimits struct {
	MaxOpenFilesSoft uint64 `json:"max_open_files_soft,omitempty"`
	MaxOpenFilesHard uint64 `json:"max_open_files_hard,omitempty"`
}

type FDSummary struct {
	Open int `json:"open"`
}

type MapsSummary struct {
	Total      int `json:"total"`
	Anonymous  int `json:"anonymous"`
	FileBacked int `json:"file_backed"`
	Deleted    int `json:"deleted"`
}

type CgroupRef struct {
	Hierarchy   string   `json:"hierarchy,omitempty"`
	Controllers []string `json:"controllers,omitempty"`
	Path        string   `json:"path"`
}

type CgroupMetrics struct {
	Version             string   `json:"version,omitempty"`
	Path                string   `json:"path,omitempty"`
	MemoryCurrentBytes  uint64   `json:"memory_current_bytes,omitempty"`
	MemoryMaxBytes      uint64   `json:"memory_max_bytes,omitempty"`
	MemoryHighBytes     uint64   `json:"memory_high_bytes,omitempty"`
	CPUUsageUsec        uint64   `json:"cpu_usage_usec,omitempty"`
	CPUUserUsec         uint64   `json:"cpu_user_usec,omitempty"`
	CPUSystemUsec       uint64   `json:"cpu_system_usec,omitempty"`
	CPUThrottledPeriods uint64   `json:"cpu_throttled_periods,omitempty"`
	CPUThrottledUsec    uint64   `json:"cpu_throttled_usec,omitempty"`
	Errors              []string `json:"errors,omitempty"`
}

type Finding struct {
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Evidence string `json:"evidence"`
	Cause    string `json:"cause"`
	Verify   string `json:"verify"`
}

type Report struct {
	GeneratedAt time.Time       `json:"generated_at"`
	Target      ProcessIdentity `json:"target"`
	Snapshot    ProcSnapshot    `json:"snapshot"`
	Cgroup      CgroupMetrics   `json:"cgroup"`
	Findings    []Finding       `json:"findings"`
	Errors      []string        `json:"errors,omitempty"`
	Next        []string        `json:"next,omitempty"`
}

// WatchReport aggregates multiple Collect snapshots for trend-style analysis.
type WatchReport struct {
	GeneratedAt     time.Time         `json:"generated_at"`
	Target          ProcessIdentity   `json:"target"`
	IntervalSeconds float64           `json:"interval_seconds,omitempty"`
	SampleCount     int               `json:"sample_count"`
	Samples         []*Report         `json:"samples"`
	TrendFindings   []Finding         `json:"trend_findings"`
	Errors          []string          `json:"errors,omitempty"`
}
