package go_runtime

import (
	"debug/buildinfo"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func FindProcessByName(procRoot, query string) (ProcessCandidate, []ProcessCandidate, error) {
	if procRoot == "" {
		procRoot = "/proc"
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return ProcessCandidate{}, nil, os.ErrInvalid
	}
	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return ProcessCandidate{}, nil, err
	}
	var candidates []ProcessCandidate
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(ent.Name())
		if err != nil || pid <= 0 {
			continue
		}
		procDir := filepath.Join(procRoot, ent.Name())
		c := inspectProcessCandidate(procDir, pid)
		if !candidateMatches(c, query) {
			continue
		}
		c.Match = matchKind(c, query)
		candidates = append(candidates, c)
	}
	if len(candidates) == 0 {
		return ProcessCandidate{}, nil, os.ErrNotExist
	}
	sortProcessCandidates(candidates)
	return candidates[0], candidates, nil
}

func inspectProcessCandidate(procDir string, pid int) ProcessCandidate {
	c := ProcessCandidate{PID: pid}
	if status, err := parseStatus(filepath.Join(procDir, "status")); err == nil {
		c.Name = status.Name
		c.RSSBytes = status.VmRSSBytes
	}
	if raw, err := os.ReadFile(filepath.Join(procDir, "cmdline")); err == nil {
		c.Cmdline = normalizeCmdline(string(raw))
	}
	if exe, err := os.Readlink(filepath.Join(procDir, "exe")); err == nil {
		c.Exe = exe
		if _, err := buildinfo.ReadFile(exe); err == nil {
			c.IsGo = true
		}
	}
	if c.Name == "" {
		c.Name = filepath.Base(firstField(c.Cmdline))
	}
	return c
}

func candidateMatches(c ProcessCandidate, query string) bool {
	q := strings.ToLower(query)
	if q == "" {
		return false
	}
	if strings.EqualFold(c.Name, query) || strings.EqualFold(filepath.Base(c.Exe), query) {
		return true
	}
	return strings.Contains(strings.ToLower(c.Cmdline), q) || strings.Contains(strings.ToLower(c.Exe), q)
}

func matchKind(c ProcessCandidate, query string) string {
	switch {
	case strings.EqualFold(c.Name, query):
		return "comm"
	case strings.EqualFold(filepath.Base(c.Exe), query):
		return "exe"
	case strings.Contains(strings.ToLower(c.Cmdline), strings.ToLower(query)):
		return "cmdline"
	default:
		return "exe_contains"
	}
}

func sortProcessCandidates(c []ProcessCandidate) {
	sort.SliceStable(c, func(i, j int) bool {
		if c[i].IsGo != c[j].IsGo {
			return c[i].IsGo
		}
		if exactRank(c[i]) != exactRank(c[j]) {
			return exactRank(c[i]) > exactRank(c[j])
		}
		if c[i].RSSBytes != c[j].RSSBytes {
			return c[i].RSSBytes > c[j].RSSBytes
		}
		return c[i].PID > c[j].PID
	})
}

func exactRank(c ProcessCandidate) int {
	switch c.Match {
	case "comm", "exe":
		return 2
	case "cmdline":
		return 1
	default:
		return 0
	}
}

func normalizeCmdline(s string) string {
	s = strings.ReplaceAll(s, "\x00", " ")
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

func firstField(s string) string {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}
