package go_runtime

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"time"
)

// UserHZ is the assumed jiffies per second for /proc/<pid>/stat utime/stime (Linux default).
const UserHZ = int64(100)

// CollectWatch runs Collect repeatedly. When count is 1, behavior matches a single snapshot
// but TrendFindings are still computed (usually empty).
func CollectWatch(ctx context.Context, opts Options, interval time.Duration, count int) (*WatchReport, error) {
	if count < 1 {
		count = 1
	}
	wr := &WatchReport{
		Target: ProcessIdentity{
			PID:       opts.PID,
			Namespace: opts.Namespace,
			Pod:       opts.Pod,
			Container: opts.Container,
		},
	}
	if count > 1 && interval > 0 {
		wr.IntervalSeconds = interval.Seconds()
	}
	for i := 0; i < count; i++ {
		if i > 0 && interval > 0 {
			select {
			case <-ctx.Done():
				return wr, ctx.Err()
			case <-time.After(interval):
			}
		}
		opts.Now = time.Now()
		rep, err := Collect(opts)
		if err != nil {
			return wr, err
		}
		wr.Samples = append(wr.Samples, rep)
	}
	if len(wr.Samples) == 0 {
		wr.GeneratedAt = time.Now()
		wr.SampleCount = 0
		return wr, nil
	}
	last := wr.Samples[len(wr.Samples)-1]
	wr.Target = last.Target
	wr.GeneratedAt = time.Now()
	wr.SampleCount = len(wr.Samples)
	wr.TrendFindings = AnalyzeTrend(wr.Samples)
	wr.Summary = SummarizeWatchReport(wr)
	return wr, nil
}

func CollectWatchWith(ctx context.Context, interval time.Duration, count int, collect func() (*Report, error)) (*WatchReport, error) {
	if count < 1 {
		count = 1
	}
	wr := &WatchReport{}
	if count > 1 && interval > 0 {
		wr.IntervalSeconds = interval.Seconds()
	}
	for i := 0; i < count; i++ {
		if i > 0 && interval > 0 {
			select {
			case <-ctx.Done():
				return wr, ctx.Err()
			case <-time.After(interval):
			}
		}
		rep, err := collect()
		if err != nil {
			return wr, err
		}
		wr.Samples = append(wr.Samples, rep)
	}
	if len(wr.Samples) == 0 {
		wr.GeneratedAt = time.Now()
		return wr, nil
	}
	wr.GeneratedAt = time.Now()
	wr.SampleCount = len(wr.Samples)
	wr.Target = wr.Samples[len(wr.Samples)-1].Target
	wr.TrendFindings = AnalyzeTrend(wr.Samples)
	wr.Summary = SummarizeWatchReport(wr)
	return wr, nil
}

// AnalyzeTrend derives findings from a time-ordered series of reports (same PID).
func AnalyzeTrend(samples []*Report) []Finding {
	var clean []*Report
	for _, s := range samples {
		if s != nil {
			clean = append(clean, s)
		}
	}
	if len(clean) < 2 {
		return nil
	}
	var out []Finding

	rss := func(r *Report) uint64 {
		if r == nil {
			return 0
		}
		return maxU64(r.Snapshot.Status.VmRSSBytes, r.Snapshot.SmapsRollup.RSSBytes)
	}
	fd := func(r *Report) int {
		if r == nil {
			return 0
		}
		return r.Snapshot.FD.Open
	}
	if monotonicIncreasing(len(clean), func(i int) uint64 { return rss(clean[i]) }, true) {
		out = append(out, Finding{
			Severity: severityWarn,
			Title:    "RSS 在采样窗口内持续上升",
			Evidence: formatUintSeries("rss_bytes", clean, rss),
			Cause:    "常驻集单调上升常见于堆增长、缓存膨胀或未释放资源；也可能是负载上升",
			Verify:   "延长观测窗口并结合 heap/pprof 或业务指标确认",
		})
	}
	if monotonicIncreasing(len(clean), func(i int) uint64 {
		return clean[i].Snapshot.SmapsRollup.AnonymousBytes
	}, true) {
		out = append(out, Finding{
			Severity: severityWarn,
			Title:    "匿名内存在采样窗口内持续上升",
			Evidence: formatUintSeries("anonymous_bytes", clean, func(r *Report) uint64 {
				return r.Snapshot.SmapsRollup.AnonymousBytes
			}),
			Cause:  "匿名内存持续增长常见于 Go heap、mmap 或缓存增长",
			Verify: "延长观测窗口，并结合业务请求量、GC 指标或后续分配热点确认",
		})
	}
	if monotonicIncreasing(len(clean), func(i int) uint64 { return uint64(fd(clean[i])) }, true) {
		out = append(out, Finding{
			Severity: severityWarn,
			Title:    "打开 FD 数量在采样窗口内持续上升",
			Evidence: formatIntSeries("open_fd", clean, fd),
			Cause:    "FD 单调上升常见于连接或句柄未关闭",
			Verify:   "检查连接池、泄漏的 TLS/HTTP 客户端与文件句柄",
		})
	}
	if monotonicIncreasing(len(clean), func(i int) uint64 {
		return uint64(maxInt(clean[i].Snapshot.Status.Threads, clean[i].Snapshot.Stat.NumThreads))
	}, true) {
		out = append(out, Finding{
			Severity: severityWarn,
			Title:    "线程数在采样窗口内持续上升",
			Evidence: formatIntSeries("threads", clean, func(r *Report) int {
				return maxInt(r.Snapshot.Status.Threads, r.Snapshot.Stat.NumThreads)
			}),
			Cause:  "Go 线程增长常见于阻塞系统调用、cgo 或调度压力",
			Verify: "继续观察 /proc/<pid>/task 与后续 goroutine 趋势",
		})
	}

	first, last := clean[0], clean[len(clean)-1]
	wall := last.GeneratedAt.Sub(first.GeneratedAt).Seconds()
	if wall >= 0.5 {
		dUt := int64(last.Snapshot.Stat.UtimeTicks) - int64(first.Snapshot.Stat.UtimeTicks)
		dSt := int64(last.Snapshot.Stat.StimeTicks) - int64(first.Snapshot.Stat.StimeTicks)
		cpuSec := float64(dUt+dSt) / float64(UserHZ)
		if cpuSec > 0 && wall > 0 {
			frac := cpuSec / wall
			if frac >= 0.85 && last.Cgroup.CPUThrottledUsec == 0 {
				out = append(out, Finding{
					Severity: severityInfo,
					Title:    "采样窗口内 CPU 时间占比偏高",
					Evidence: formatCPUFracEvidence(cpuSec, wall, frac, dUt+dSt),
					Cause:    "进程在用户态/内核态消耗了接近一个 CPU 核的算力；可能是热点循环、批处理或 GC 压力",
					Verify:   "结合 pprof CPU profile 与业务 QPS 判断是否为异常",
				})
			}
		}
	}
	return out
}

func monotonicIncreasing(n int, get func(i int) uint64, strict bool) bool {
	if n < 3 {
		return false
	}
	prev := get(0)
	for i := 1; i < n; i++ {
		v := get(i)
		if strict {
			if v <= prev {
				return false
			}
		} else {
			if v < prev {
				return false
			}
		}
		prev = v
	}
	return true
}

func formatUintSeries(label string, samples []*Report, get func(*Report) uint64) string {
	parts := make([]string, 0, len(samples))
	for _, s := range samples {
		parts = append(parts, humanBytes(get(s)))
	}
	return label + ": " + joinComma(parts)
}

func formatIntSeries(label string, samples []*Report, get func(*Report) int) string {
	parts := make([]string, 0, len(samples))
	for _, s := range samples {
		parts = append(parts, strconv.Itoa(get(s)))
	}
	return label + ": " + joinComma(parts)
}

func joinComma(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	default:
		s := parts[0]
		for i := 1; i < len(parts); i++ {
			s += ", " + parts[i]
		}
		return s
	}
}

func formatCPUFracEvidence(cpuSec, wall, frac float64, deltaTicks int64) string {
	return fmt.Sprintf("cpu_time≈%.2fs wall=%.2fs frac=%.2f Δticks=%d (USER_HZ=%d)", cpuSec, wall, frac, deltaTicks, UserHZ)
}

// WriteWatchJSON writes a watch report as indented JSON.
func WriteWatchJSON(w io.Writer, wrep *WatchReport) error {
	if wrep == nil {
		return nil
	}
	return writeJSONGeneric(w, wrep)
}
