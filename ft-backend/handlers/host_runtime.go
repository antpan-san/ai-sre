package handlers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	CPU          float64
	CPUCores     int
	Memory       float64
	MemUsed      uint64
	MemTotal     uint64
	Disk         float64
	DiskUsed     uint64
	DiskTotal    uint64
	DiskPath     string
	Load1        float64 // 1 分钟 load average
	DiskIO       float64 // 根盘 IO 忙百分比
	DiskIODevice string
	Hostname     string
	SampledAt    string
	OS           string
	ErrCollect   string `json:"errCollect,omitempty"`
}

var (
	reNVMEPart = regexp.MustCompile(`^(nvme\d+n\d+)p\d+$`)
	reSdPart   = regexp.MustCompile(`^([a-z]+)\d+$`)
)

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

func rootDiskMountDevice(ctx context.Context) (string, error) {
	root := rootPathForDiskUsage()
	parts, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return "", err
	}
	for _, p := range parts {
		if p.Mountpoint == root {
			if p.Device == "" {
				break
			}
			return p.Device, nil
		}
	}
	return "", fmt.Errorf("root mount %q not found", root)
}

func diskNameForIO(device string) string {
	name := filepath.Base(device)
	name = strings.TrimPrefix(name, "/dev/")
	if m := reNVMEPart.FindStringSubmatch(name); len(m) > 1 {
		return m[1]
	}
	if m := reSdPart.FindStringSubmatch(name); len(m) > 1 {
		return m[1]
	}
	return name
}

func collectLoad1(ctx context.Context) (load1 float64, err error) {
	avg, err := load.AvgWithContext(ctx)
	if err != nil {
		return 0, err
	}
	return avg.Load1, nil
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

func collectRootDiskIOBusy(ctx context.Context, interval time.Duration) (busy float64, ioName string, err error) {
	dev, err := rootDiskMountDevice(ctx)
	if err != nil {
		return 0, "", err
	}
	ioName = diskNameForIO(dev)
	c1, err := disk.IOCountersWithContext(ctx, ioName)
	if err != nil {
		return 0, ioName, err
	}
	select {
	case <-ctx.Done():
		return 0, ioName, ctx.Err()
	case <-time.After(interval):
	}
	c2, err := disk.IOCountersWithContext(ctx, ioName)
	if err != nil {
		return 0, ioName, err
	}
	return diskIOBusyFromCounters(c1, c2, interval), ioName, nil
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

// collectHostRuntime 阻塞约 interval（CPU 与根盘 IO 采样）；ctx 建议带超时。
func collectHostRuntime(ctx context.Context, interval time.Duration) hostRuntimeSample {
	out := hostRuntimeSample{
		Hostname:  readHostname(),
		SampledAt: time.Now().UTC().Format(time.RFC3339),
		OS:        runtime.GOOS + "/" + runtime.GOARCH,
		DiskPath:  rootPathForDiskUsage(),
	}
	if interval <= 0 {
		interval = 280 * time.Millisecond
	}
	cores, err := cpu.CountsWithContext(ctx, false)
	if err != nil || cores <= 0 {
		cores = 1
	}
	out.CPUCores = cores

	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		out.ErrCollect = "memory: " + err.Error()
		return out
	}
	out.Memory = clampPct(vm.UsedPercent)
	out.MemUsed = vm.Used
	out.MemTotal = vm.Total

	du, err := disk.UsageWithContext(ctx, out.DiskPath)
	if err != nil {
		out.ErrCollect = "disk: " + err.Error()
		return out
	}
	out.Disk = clampPct(du.UsedPercent)
	out.DiskUsed = du.Used
	out.DiskTotal = du.Total

	load1, err := collectLoad1(ctx)
	if err != nil {
		out.ErrCollect = appendCollectErr(out.ErrCollect, "load: "+err.Error())
	} else {
		out.Load1 = load1
	}

	io1, ioName, ioErr := func() (map[string]disk.IOCountersStat, string, error) {
		dev, err := rootDiskMountDevice(ctx)
		if err != nil {
			return nil, "", err
		}
		name := diskNameForIO(dev)
		c, err := disk.IOCountersWithContext(ctx, name)
		return c, name, err
	}()

	cpuPct, err := cpu.PercentWithContext(ctx, interval, false)
	if err != nil {
		out.ErrCollect = appendCollectErr(out.ErrCollect, "cpu: "+err.Error())
		return out
	}
	if len(cpuPct) > 0 {
		out.CPU = clampPct(cpuPct[0])
	}

	if ioErr == nil {
		io2, err2 := disk.IOCountersWithContext(ctx, ioName)
		if err2 != nil {
			out.ErrCollect = appendCollectErr(out.ErrCollect, "diskIo: "+err2.Error())
		} else {
			out.DiskIO = diskIOBusyFromCounters(io1, io2, interval)
			out.DiskIODevice = ioName
		}
	} else {
		out.ErrCollect = appendCollectErr(out.ErrCollect, "diskIo: "+ioErr.Error())
	}
	return out
}
