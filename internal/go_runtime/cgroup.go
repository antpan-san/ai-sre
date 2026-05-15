package go_runtime

import (
	"os"
	"path/filepath"
	"strings"
)

func collectCgroupMetrics(root string, refs []CgroupRef) CgroupMetrics {
	if root == "" {
		root = "/sys/fs/cgroup"
	}
	if len(refs) == 0 {
		return CgroupMetrics{}
	}
	if cg := collectCgroupV2(root, refs); cg.Version != "" {
		return cg
	}
	return collectCgroupV1(root, refs)
}

func collectCgroupV2(root string, refs []CgroupRef) CgroupMetrics {
	for _, ref := range refs {
		if len(ref.Controllers) != 0 {
			continue
		}
		dir := safeCgroupPath(root, ref.Path)
		if dir == "" {
			continue
		}
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		out := CgroupMetrics{Version: "v2", Path: ref.Path}
		out.MemoryCurrentBytes = readUintFile(filepath.Join(dir, "memory.current"))
		out.MemoryMaxBytes = readMaxFile(filepath.Join(dir, "memory.max"))
		out.MemoryHighBytes = readMaxFile(filepath.Join(dir, "memory.high"))
		readCPUStat(filepath.Join(dir, "cpu.stat"), &out)
		return out
	}
	return CgroupMetrics{}
}

func collectCgroupV1(root string, refs []CgroupRef) CgroupMetrics {
	out := CgroupMetrics{Version: "v1"}
	for _, ref := range refs {
		if hasController(ref, "memory") {
			dir := safeCgroupPath(filepath.Join(root, "memory"), ref.Path)
			out.Path = ref.Path
			out.MemoryCurrentBytes = readUintFile(filepath.Join(dir, "memory.usage_in_bytes"))
			out.MemoryMaxBytes = readMaxFile(filepath.Join(dir, "memory.limit_in_bytes"))
		}
		if hasController(ref, "cpu") || hasController(ref, "cpuacct") {
			dir := safeCgroupPath(filepath.Join(root, "cpu,cpuacct"), ref.Path)
			if _, err := os.Stat(dir); err != nil {
				dir = safeCgroupPath(filepath.Join(root, "cpuacct"), ref.Path)
			}
			if out.Path == "" {
				out.Path = ref.Path
			}
			out.CPUUsageUsec = readUintFile(filepath.Join(dir, "cpuacct.usage")) / 1000
		}
	}
	if out.MemoryCurrentBytes == 0 && out.CPUUsageUsec == 0 {
		return CgroupMetrics{}
	}
	return out
}

func hasController(ref CgroupRef, name string) bool {
	for _, c := range ref.Controllers {
		if c == name {
			return true
		}
	}
	return false
}

func safeCgroupPath(root, cgroupPath string) string {
	root = filepath.Clean(root)
	p := filepath.Clean(filepath.Join(root, strings.TrimPrefix(cgroupPath, "/")))
	if p != root && !strings.HasPrefix(p, root+string(os.PathSeparator)) {
		return ""
	}
	return p
}

func readUintFile(path string) uint64 {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	return atou64(strings.TrimSpace(string(b)))
}

func readMaxFile(path string) uint64 {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	s := strings.TrimSpace(string(b))
	if s == "" || s == "max" {
		return 0
	}
	return atou64(s)
}

func readCPUStat(path string, out *CgroupMetrics) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		switch fields[0] {
		case "usage_usec":
			out.CPUUsageUsec = atou64(fields[1])
		case "user_usec":
			out.CPUUserUsec = atou64(fields[1])
		case "system_usec":
			out.CPUSystemUsec = atou64(fields[1])
		case "nr_throttled":
			out.CPUThrottledPeriods = atou64(fields[1])
		case "throttled_usec":
			out.CPUThrottledUsec = atou64(fields[1])
		}
	}
}
