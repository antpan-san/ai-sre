package go_runtime

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const (
	KiB = uint64(1024)
	MiB = KiB * 1024
	GiB = MiB * 1024
)

func WriteJSON(w io.Writer, report *Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func WriteText(w io.Writer, r *Report) error {
	if r == nil {
		return nil
	}
	fmt.Fprintf(w, "Go Runtime 诊断报告\n")
	fmt.Fprintf(w, "目标: pid=%d comm=%s state=%s\n", r.Target.PID, dash(r.Target.Comm), dash(r.Target.State))
	if r.Target.Namespace != "" || r.Target.Pod != "" || r.Target.Container != "" {
		fmt.Fprintf(w, "Kubernetes: namespace=%s pod=%s container=%s\n", dash(r.Target.Namespace), dash(r.Target.Pod), dash(r.Target.Container))
	}
	fmt.Fprintf(w, "\n指标快照:\n")
	fmt.Fprintf(w, "- RSS: %s (HWM %s, VmSize %s)\n", humanBytes(maxU64(r.Snapshot.Status.VmRSSBytes, r.Snapshot.SmapsRollup.RSSBytes)), humanBytes(r.Snapshot.Status.VmHWMBytes), humanBytes(r.Snapshot.Status.VmSizeBytes))
	fmt.Fprintf(w, "- Anonymous: %s, Private: %s, Shared: %s\n", humanBytes(r.Snapshot.SmapsRollup.AnonymousBytes), humanBytes(r.Snapshot.SmapsRollup.PrivateBytes), humanBytes(r.Snapshot.SmapsRollup.SharedBytes))
	fmt.Fprintf(w, "- FD: %d / %s, Threads: %d\n", r.Snapshot.FD.Open, limitText(r.Snapshot.Limits.MaxOpenFilesSoft), maxInt(r.Snapshot.Status.Threads, r.Snapshot.Stat.NumThreads))
	fmt.Fprintf(w, "- Maps: total=%d anonymous=%d file_backed=%d deleted=%d\n", r.Snapshot.Maps.Total, r.Snapshot.Maps.Anonymous, r.Snapshot.Maps.FileBacked, r.Snapshot.Maps.Deleted)
	if r.Cgroup.Version != "" {
		fmt.Fprintf(w, "- Cgroup(%s): memory.current=%s memory.max=%s cpu.usage=%dus throttled=%dus\n", r.Cgroup.Version, humanBytes(r.Cgroup.MemoryCurrentBytes), limitText(r.Cgroup.MemoryMaxBytes), r.Cgroup.CPUUsageUsec, r.Cgroup.CPUThrottledUsec)
	}
	fmt.Fprintf(w, "\n发现:\n")
	for i, f := range r.Findings {
		fmt.Fprintf(w, "%d. [%s] %s\n", i+1, strings.ToUpper(f.Severity), f.Title)
		fmt.Fprintf(w, "   证据: %s\n", f.Evidence)
		fmt.Fprintf(w, "   可能原因: %s\n", f.Cause)
		fmt.Fprintf(w, "   验证: %s\n", f.Verify)
	}
	if len(r.Errors) > 0 {
		fmt.Fprintf(w, "\n采集警告:\n")
		for _, e := range r.Errors {
			fmt.Fprintf(w, "- %s\n", e)
		}
	}
	if len(r.Next) > 0 {
		fmt.Fprintf(w, "\n下一步:\n")
		for _, n := range r.Next {
			fmt.Fprintf(w, "- %s\n", n)
		}
	}
	return nil
}

func humanBytes(v uint64) string {
	switch {
	case v >= GiB:
		return fmt.Sprintf("%.1fGiB", float64(v)/float64(GiB))
	case v >= MiB:
		return fmt.Sprintf("%.1fMiB", float64(v)/float64(MiB))
	case v >= KiB:
		return fmt.Sprintf("%.1fKiB", float64(v)/float64(KiB))
	default:
		return fmt.Sprintf("%dB", v)
	}
}

func limitText(v uint64) string {
	if v == 0 {
		return "unlimited"
	}
	return fmt.Sprintf("%d", v)
}

func maxU64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func dash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return strings.TrimSpace(s)
}
