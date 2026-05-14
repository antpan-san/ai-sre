package handlers

import (
	"context"
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
)

// hostRuntimeSample 采集运行本后端的操作系统主机上的资源占用（与业务侧 Machine 心跳无关）。
type hostRuntimeSample struct {
	CPU        float64
	Memory     float64
	Disk       float64
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

// collectHostRuntime 阻塞约 interval（CPU 采样）；ctx 建议带超时。
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

	cpuPct, err := cpu.PercentWithContext(ctx, interval, false)
	if err != nil {
		out.ErrCollect = "cpu: " + err.Error()
		return out
	}
	if len(cpuPct) > 0 {
		out.CPU = clampPct(cpuPct[0])
	}
	return out
}
