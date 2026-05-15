package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// EnvOrString returns the first non-empty value among env (if set) then yamlVal.
func EnvOrString(envKey, yamlVal string) string {
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		return v
	}
	return strings.TrimSpace(yamlVal)
}

// EnvOrBool: when envKey is set, parses 1/true/yes/on vs 0/false/no/off; else yamlVal.
func EnvOrBool(envKey string, yamlVal bool) bool {
	v := strings.TrimSpace(os.Getenv(envKey))
	if v == "" {
		return yamlVal
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return yamlVal
	}
}

// EnvOrInt: when envKey is set and parseable, returns it; else yamlVal (yamlVal<=0 uses def).
func EnvOrInt(envKey string, yamlVal, def int) int {
	v := strings.TrimSpace(os.Getenv(envKey))
	if v == "" {
		if yamlVal > 0 {
			return yamlVal
		}
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		if yamlVal > 0 {
			return yamlVal
		}
		return def
	}
	return n
}

// EnvOrDuration parses env or yaml duration string (e.g. 12h); def if both empty/invalid.
func EnvOrDuration(envKey, yamlDuration string, def time.Duration) time.Duration {
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	if d, err := time.ParseDuration(strings.TrimSpace(yamlDuration)); err == nil {
		return d
	}
	return def
}

// EnvOrStringList: env comma-separated overrides non-empty yaml slice.
func EnvOrStringList(envKey string, yamlList []string) []string {
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		out := make([]string, 0, 4)
		for _, p := range strings.Split(v, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return yamlList
}
