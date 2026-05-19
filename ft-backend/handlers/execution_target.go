package handlers

import "strings"

// executionTargetFromAIContext extracts a user-visible diagnosis target from CLI context.
func executionTargetFromAIContext(kv map[string]string) string {
	if kv == nil {
		return ""
	}
	for _, k := range []string{"addr", "target", "bootstrap", "bootstrap_server", "dsn", "url", "base_url", "domain"} {
		if v := strings.TrimSpace(kv[k]); v != "" {
			return v
		}
	}
	host := strings.TrimSpace(kv["host"])
	if host == "" {
		return ""
	}
	port := strings.TrimSpace(kv["port"])
	if port != "" {
		return host + ":" + port
	}
	return host
}
