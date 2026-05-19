package handlers

import (
	"context"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
)

// hostRuntimeSample 采集运行本后端的操作系统主机上的资源占用（与业务侧 Machine 心跳无关）。
type hostRuntimeSample struct {
	CPU        float64
	Memory     float64
	Disk       float64
	Load       float64 // 1 分钟 load average 相对 CPU 核数的百分比（0–100）
	Load1      float64 // 原始 1 分钟 load average，供 tooltip
	DiskIO     float64 // 磁盘 IO 忙百分比（两次采样 IoTime/读写时间增量）
	Hostname   string
	SampledAt  string
	OS         string
	ErrCollect string `json:"errCollect,omitempty"`
}

func readHostname() string {
	h, err := os.Hostname()
	if err != nil || h == "" {
		return "unknown"
	}
	return h
}

func rootPathForDiskUsage() string {
	if runtime.GOOS == "windows" {
		return `C:\`
	}
	return "/"
}

func collectLoadPct(ctx context.Context) (pct float64, load1 float64, err error) {
	avg, err := load.AvgWithContext(ctx)
	if err != nil {
		return 0, 0, err
	}
	n, err := cpu.CountsWithContext(ctx, false)
	if err != nil || n <= 0 {
		n = 1
	}
	load1 = avg.Load1
	pct = clampPct((load1 / float64(n)) * 100)
	return pct, load1, nil
}

func ioBusyDelta(s1, s2 disk.IOCountersStat) uint64 {
	if s2.IoTime > s1.IoTime {
		return s2.IoTime - s1.IoTime
	}
	var d uint64
	if s2.ReadTime >= s1.ReadTime {
		d += s2.ReadTime - s1.ReadTime
	}
	if s2.WriteTime >= s1.WriteTime {
		d += s2.WriteTime - s1.WriteTime
	}
	return d
}

func diskIOBusyFromCounters(c1, c2 map[string]disk.IOCountersStat, interval time.Duration) float64 {
	if interval <= 0 {
		interval = time.Millisecond
	}
	ms := float64(interval.Milliseconds())
	if ms <= 0 {
		return 0
	}
	var delta uint64
	for name, s2 := range c2 {
		s1, ok := c1[name]
		if !ok {
			continue
		}
		delta += ioBusyDelta(s1, s2)
	}
	return clampPct(float64(delta) / ms * 100)
}

func appendCollectErr(existing, part string) string {
	part = strings.TrimSpace(part)
	if part == "" {
		return existing
	}
	if existing == "" {
		return part
	}
	return existing + "; " + part
}

// collectHostRuntime 阻塞约 interval（CPU 与磁盘 IO 采样）；ctx 建议带超时。
func collectHostRuntime(ctx context.Context, interval time.Duration) hostRuntimeSample {
	out := hostRuntimeSample{
		Hostname:  readHostname(),
		SampledAt: time.Now().UTC().Format(time.RFC3339),
		OS:        runtime.GOOS + "/" + runtime.GOARCH,
	}
	if interval <= 0 {
		interval = 280 * time.Millisecond
	}
	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		out.ErrCollect = "memory: " + err.Error()
		return out
	}
	out.Memory = clampPct(vm.UsedPercent)

	du, err := disk.UsageWithContext(ctx, rootPathForDiskUsage())
	if err != nil {
		out.ErrCollect = "disk: " + err.Error()
		return out
	}
	out.Disk = clampPct(du.UsedPercent)

	loadPct, load1, err := collectLoadPct(ctx)
	if err != nil {
		out.ErrCollect = appendCollectErr(out.ErrCollect, "load: "+err.Error())
	} else {
		out.Load = loadPct
		out.Load1 = load1
	}

	io1, ioErr := disk.IOCountersWithContext(ctx)

	cpuPct, err := cpu.PercentWithContext(ctx, interval, false)
	if err != nil {
		out.ErrCollect = appendCollectErr(out.ErrCollect, "cpu: "+err.Error())
		return out
	}
	if len(cpuPct) > 0 {
		out.CPU = clampPct(cpuPct[0])
	}

	if ioErr == nil {
		io2, err2 := disk.IOCountersWithContext(ctx)
		if err2 != nil {
			out.ErrCollect = appendCollectErr(out.ErrCollect, "diskIo: "+err2.Error())
		} else {
			out.DiskIO = diskIOBusyFromCounters(io1, io2, interval)
		}
	} else {
		out.ErrCollect = appendCollectErr(out.ErrCollect, "diskIo: "+ioErr.Error())
	}
	return out
}
