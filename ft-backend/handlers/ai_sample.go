package handlers

import (
	"sort"
	"strings"
)

// stripBulkEvidenceForSample keeps only small user-supplied flags out of bulk evidence.
func stripBulkEvidenceForSample(kv map[string]string) map[string]string {
	if kv == nil {
		return nil
	}
	out := make(map[string]string, len(kv))
	for k, v := range kv {
		if strings.HasPrefix(k, "kubectl_") || strings.HasPrefix(k, "host_") {
			continue
		}
		if k == "prior_answer_round1" || k == "go_runtime_watch_json" {
			continue
		}
		if len(v) > 256 {
			v = v[:256] + "...(truncated)"
		}
		out[k] = v
	}
	return out
}

func evidenceKeyList(kv map[string]string) []string {
	if kv == nil {
		return nil
	}
	out := make([]string, 0, 8)
	for k := range kv {
		if strings.HasPrefix(k, "kubectl_") || strings.HasPrefix(k, "host_") {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func headSample(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func tailSample(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
