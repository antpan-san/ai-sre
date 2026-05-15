package go_runtime

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func Collect(opts Options) (*Report, error) {
	if opts.PID <= 0 {
		return nil, fmt.Errorf("pid must be positive")
	}
	if opts.ProcRoot == "" {
		opts.ProcRoot = "/proc"
	}
	if opts.CgroupRoot == "" {
		opts.CgroupRoot = "/sys/fs/cgroup"
	}
	if opts.Now.IsZero() {
		opts.Now = now()
	}
	procDir := filepath.Join(opts.ProcRoot, strconv.Itoa(opts.PID))
	snap := ProcSnapshot{}
	var errs []string

	status, err := parseStatus(filepath.Join(procDir, "status"))
	if err != nil {
		errs = append(errs, "status: "+err.Error())
	}
	snap.Status = status
	if smaps, err := parseSmapsRollup(filepath.Join(procDir, "smaps_rollup")); err == nil {
		snap.SmapsRollup = smaps
	} else {
		errs = append(errs, "smaps_rollup: "+err.Error())
	}
	if stat, err := parseStat(filepath.Join(procDir, "stat")); err == nil {
		snap.Stat = stat
	} else {
		errs = append(errs, "stat: "+err.Error())
	}
	if limits, err := parseLimits(filepath.Join(procDir, "limits")); err == nil {
		snap.Limits = limits
	} else {
		errs = append(errs, "limits: "+err.Error())
	}
	if fd, err := countFD(filepath.Join(procDir, "fd")); err == nil {
		snap.FD = fd
	} else {
		errs = append(errs, "fd: "+err.Error())
	}
	if maps, err := parseMaps(filepath.Join(procDir, "maps")); err == nil {
		snap.Maps = maps
	} else {
		errs = append(errs, "maps: "+err.Error())
	}
	if cgroups, err := parseCgroups(filepath.Join(procDir, "cgroup")); err == nil {
		snap.Cgroups = cgroups
	} else {
		errs = append(errs, "cgroup: "+err.Error())
	}

	cg := collectCgroupMetrics(opts.CgroupRoot, snap.Cgroups)
	report := &Report{
		GeneratedAt: opts.Now,
		Target: ProcessIdentity{
			PID:       opts.PID,
			Comm:      firstNonEmpty(snap.Status.Name, snap.Stat.Comm),
			State:     firstNonEmpty(snap.Status.State, snap.Stat.State),
			Namespace: strings.TrimSpace(opts.Namespace),
			Pod:       strings.TrimSpace(opts.Pod),
			Container: strings.TrimSpace(opts.Container),
		},
		Snapshot: snap,
		Cgroup:   cg,
		Errors:   append(errs, cg.Errors...),
	}
	report.Findings = Analyze(report)
	report.Summary = SummarizeReport(report, nil)
	report.Next = nextSteps(report)
	return report, nil
}

func parseStatus(path string) (ProcStatus, error) {
	f, err := os.Open(path)
	if err != nil {
		return ProcStatus{}, err
	}
	defer f.Close()
	out := ProcStatus{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		key, val, ok := splitProcKV(sc.Text())
		if !ok {
			continue
		}
		switch key {
		case "Name":
			out.Name = val
		case "State":
			out.State = val
		case "Threads":
			out.Threads = atoi(val)
		case "VmRSS":
			out.VmRSSBytes = parseKB(val)
		case "VmHWM":
			out.VmHWMBytes = parseKB(val)
		case "VmSize":
			out.VmSizeBytes = parseKB(val)
		case "VmData":
			out.VmDataBytes = parseKB(val)
		case "VmStk":
			out.VmStkBytes = parseKB(val)
		case "VmExe":
			out.VmExeBytes = parseKB(val)
		}
	}
	return out, sc.Err()
}

func parseSmapsRollup(path string) (SmapsRollup, error) {
	f, err := os.Open(path)
	if err != nil {
		return SmapsRollup{}, err
	}
	defer f.Close()
	out := SmapsRollup{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		key, val, ok := splitProcKV(sc.Text())
		if !ok {
			continue
		}
		switch key {
		case "Rss":
			out.RSSBytes = parseKB(val)
		case "Pss":
			out.PSSBytes = parseKB(val)
		case "Anonymous":
			out.AnonymousBytes = parseKB(val)
		case "Private_Clean", "Private_Dirty":
			out.PrivateBytes += parseKB(val)
		case "Shared_Clean", "Shared_Dirty":
			out.SharedBytes += parseKB(val)
		}
	}
	return out, sc.Err()
}

func parseStat(path string) (ProcStat, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return ProcStat{}, err
	}
	s := strings.TrimSpace(string(b))
	l := strings.LastIndex(s, ")")
	r := strings.Index(s, "(")
	if r < 0 || l <= r {
		return ProcStat{}, fmt.Errorf("invalid stat")
	}
	out := ProcStat{Comm: s[r+1 : l]}
	fields := strings.Fields(strings.TrimSpace(s[l+1:]))
	if len(fields) > 19 {
		out.State = fields[0]
		out.NumThreads = atoi(fields[17])
		out.StartTime = atou64(fields[19])
	}
	if len(fields) > 12 {
		out.UtimeTicks = atou64(fields[11])
		out.StimeTicks = atou64(fields[12])
	}
	return out, nil
}

func parseLimits(path string) (ProcLimits, error) {
	f, err := os.Open(path)
	if err != nil {
		return ProcLimits{}, err
	}
	defer f.Close()
	out := ProcLimits{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "Max open files") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 6 {
			out.MaxOpenFilesSoft = parseLimit(fields[len(fields)-3])
			out.MaxOpenFilesHard = parseLimit(fields[len(fields)-2])
		}
	}
	return out, sc.Err()
}

func countFD(path string) (FDSummary, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return FDSummary{}, err
	}
	return FDSummary{Open: len(entries)}, nil
}

func parseMaps(path string) (MapsSummary, error) {
	f, err := os.Open(path)
	if err != nil {
		return MapsSummary{}, err
	}
	defer f.Close()
	out := MapsSummary{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		out.Total++
		if strings.Contains(line, "(deleted)") {
			out.Deleted++
		}
		fields := strings.Fields(line)
		if len(fields) < 6 || strings.HasPrefix(fields[len(fields)-1], "[") {
			out.Anonymous++
		} else {
			out.FileBacked++
		}
	}
	return out, sc.Err()
}

func parseCgroups(path string) ([]CgroupRef, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []CgroupRef
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		parts := strings.SplitN(sc.Text(), ":", 3)
		if len(parts) != 3 {
			continue
		}
		ctrls := []string{}
		if strings.TrimSpace(parts[1]) != "" {
			ctrls = strings.Split(parts[1], ",")
		}
		out = append(out, CgroupRef{Hierarchy: parts[0], Controllers: ctrls, Path: parts[2]})
	}
	return out, sc.Err()
}

func splitProcKV(line string) (string, string, bool) {
	k, v, ok := strings.Cut(line, ":")
	if !ok {
		return "", "", false
	}
	return strings.TrimSpace(k), strings.TrimSpace(v), true
}

func parseKB(s string) uint64 {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return 0
	}
	return atou64(fields[0]) * 1024
}

func parseLimit(s string) uint64 {
	if strings.EqualFold(s, "unlimited") {
		return 0
	}
	return atou64(s)
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

func atou64(s string) uint64 {
	n, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	return n
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
