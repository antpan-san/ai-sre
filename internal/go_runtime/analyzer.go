package go_runtime

import "fmt"

const (
	severityInfo = "info"
	severityWarn = "warn"
	severityCrit = "critical"
)

func Analyze(r *Report) []Finding {
	if r == nil {
		return nil
	}
	var out []Finding
	rss := maxU64(r.Snapshot.Status.VmRSSBytes, r.Snapshot.SmapsRollup.RSSBytes)
	anon := r.Snapshot.SmapsRollup.AnonymousBytes
	if rss >= 2*GiB {
		out = append(out, Finding{
			Severity: severityCrit,
			Title:    "RSS 已超过 2GiB",
			Evidence: fmt.Sprintf("rss=%s vm_size=%s anonymous=%s", humanBytes(rss), humanBytes(r.Snapshot.Status.VmSizeBytes), humanBytes(anon)),
			Cause:    "进程常驻内存较高，可能是堆增长、mmap/匿名内存增长、缓存未释放或容器 limit 配置过小",
			Verify:   "连续采样对比 Rss/Anonymous/VmData；若业务允许，结合 runtime.MemStats 或后续 eBPF 分配热点确认",
		})
	} else if rss >= GiB {
		out = append(out, Finding{
			Severity: severityWarn,
			Title:    "RSS 较高",
			Evidence: fmt.Sprintf("rss=%s anonymous=%s", humanBytes(rss), humanBytes(anon)),
			Cause:    "Go heap、匿名 mmap 或进程缓存可能处于高水位",
			Verify:   "间隔 1-5 分钟再次采样，观察 RSS 与 Anonymous 是否持续上升",
		})
	}
	if rss > 0 && anon*100/rss >= 70 {
		out = append(out, Finding{
			Severity: severityWarn,
			Title:    "匿名内存占比较高",
			Evidence: fmt.Sprintf("anonymous=%s rss=%s ratio=%d%%", humanBytes(anon), humanBytes(rss), anon*100/rss),
			Cause:    "匿名内存通常对应 Go heap、栈、mmap 或运行时分配，持续增长时优先怀疑内存泄漏或缓存无界",
			Verify:   "对比 smaps_rollup Anonymous、VmData 与 cgroup memory.current 的趋势",
		})
	}
	fdLimit := r.Snapshot.Limits.MaxOpenFilesSoft
	fdOpen := uint64(r.Snapshot.FD.Open)
	if fdLimit > 0 && fdOpen*100/fdLimit >= 80 {
		out = append(out, Finding{
			Severity: severityCrit,
			Title:    "FD 接近上限",
			Evidence: fmt.Sprintf("open_fd=%d soft_limit=%d usage=%d%%", fdOpen, fdLimit, fdOpen*100/fdLimit),
			Cause:    "可能存在连接、文件或 epoll fd 泄漏，达到上限后会出现 accept/open 失败",
			Verify:   "检查 /proc/<pid>/fd 类型分布；连续采样确认 fd 是否单调增长",
		})
	} else if fdOpen >= 1024 {
		out = append(out, Finding{
			Severity: severityWarn,
			Title:    "FD 数量偏高",
			Evidence: fmt.Sprintf("open_fd=%d soft_limit=%d", fdOpen, fdLimit),
			Cause:    "高并发连接或 fd 未关闭都可能造成该现象",
			Verify:   "按 fd 软链接聚合 socket/file/eventfd 类型并复查增长趋势",
		})
	}
	threads := maxInt(r.Snapshot.Status.Threads, r.Snapshot.Stat.NumThreads)
	if threads >= 500 {
		out = append(out, Finding{
			Severity: severityCrit,
			Title:    "线程数异常高",
			Evidence: fmt.Sprintf("threads=%d", threads),
			Cause:    "Go 程序线程暴涨常见于阻塞系统调用、cgo、网络/文件 IO 长时间阻塞或调度压力",
			Verify:   "结合 /proc/<pid>/task 与后续 goroutine/线程趋势采样定位增长来源",
		})
	} else if threads >= 100 {
		out = append(out, Finding{
			Severity: severityWarn,
			Title:    "线程数偏高",
			Evidence: fmt.Sprintf("threads=%d", threads),
			Cause:    "可能存在大量阻塞调用、cgo 或运行时需要更多 M 承载 goroutine",
			Verify:   "连续采样 threads，并结合 CPU throttling 与 IO 等待判断",
		})
	}
	if r.Cgroup.MemoryMaxBytes > 0 && r.Cgroup.MemoryCurrentBytes > 0 {
		pct := r.Cgroup.MemoryCurrentBytes * 100 / r.Cgroup.MemoryMaxBytes
		if pct >= 90 {
			out = append(out, Finding{
				Severity: severityCrit,
				Title:    "cgroup 内存接近 limit",
				Evidence: fmt.Sprintf("memory.current=%s memory.max=%s usage=%d%%", humanBytes(r.Cgroup.MemoryCurrentBytes), humanBytes(r.Cgroup.MemoryMaxBytes), pct),
				Cause:    "容器内存即将触发 OOMKill，可能由 Go heap、匿名内存或 page cache 推高",
				Verify:   "对比 RSS、Anonymous 与 memory.current；查看 Pod 最近 OOMKilled 事件",
			})
		} else if pct >= 75 {
			out = append(out, Finding{
				Severity: severityWarn,
				Title:    "cgroup 内存使用率较高",
				Evidence: fmt.Sprintf("memory.current=%s memory.max=%s usage=%d%%", humanBytes(r.Cgroup.MemoryCurrentBytes), humanBytes(r.Cgroup.MemoryMaxBytes), pct),
				Cause:    "容器内存余量有限，负载上升或 GC 延迟时可能触顶",
				Verify:   "连续采样 memory.current 与 RSS，确认是否持续增长",
			})
		}
	}
	if r.Cgroup.CPUThrottledPeriods > 0 || r.Cgroup.CPUThrottledUsec > 0 {
		out = append(out, Finding{
			Severity: severityInfo,
			Title:    "检测到 CPU throttling",
			Evidence: fmt.Sprintf("nr_throttled=%d throttled_usec=%d", r.Cgroup.CPUThrottledPeriods, r.Cgroup.CPUThrottledUsec),
			Cause:    "容器 CPU limit 可能限制了 Go 调度与 GC 运行窗口",
			Verify:   "结合业务延迟、GOMAXPROCS 与 cgroup CPU 配额继续判断",
		})
	}
	if len(out) == 0 {
		out = append(out, Finding{
			Severity: severityInfo,
			Title:    "未发现明显运行时异常",
			Evidence: fmt.Sprintf("rss=%s fd=%d threads=%d", humanBytes(rss), r.Snapshot.FD.Open, threads),
			Cause:    "单次快照未触发内置阈值",
			Verify:   "若怀疑泄漏，请间隔采样比较 RSS、Anonymous、FD 和 threads 趋势",
		})
	}
	return out
}

func nextSteps(r *Report) []string {
	return []string{
		"间隔 1-5 分钟重复采样，确认 RSS、Anonymous、FD、Threads 是否持续增长",
		"Kubernetes 场景后续可用 namespace/pod/container 解析宿主机 PID 后复用同一采集器",
		"CPU 热点、mallocgc 分配热点与 goroutine 趋势预留给 eBPF/perf 阶段实现",
	}
}
