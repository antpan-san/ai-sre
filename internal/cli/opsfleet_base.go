package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"runtime"
	"strings"

	"github.com/panshuai/ai-sre/internal/config"
)

// EmbeddedOpsfleetAPIBase 内建 OpsFort 控制台 API 基址（须含 /ft-api 前缀，与 Nginx 反代一致）。
// 实验室默认；生产环境 install-ai-sre 会写入 ~/.config/ai-sre/opsfleet_api_url。
const EmbeddedOpsfleetAPIBase = "http://192.168.56.11:9080/ft-api"

// EmbeddedOpsfleetAPIBaseProduction 生产控制台（自升级探测回退，避免仅内嵌实验室 IP 时外网客户端永远连不上）。
const EmbeddedOpsfleetAPIBaseProduction = "http://opsfleetpilot.com/ft-api"

func resolveOpsfleetAPIBase() string {
	bases := resolveOpsfleetAPIBasesForUpgrade()
	if len(bases) == 0 {
		return ""
	}
	return bases[0]
}

// resolveOpsfleetAPIBasesForUpgrade 返回自升级/版本探测用的 API 基址列表（按优先级，去重）。
func resolveOpsfleetAPIBasesForUpgrade() []string {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("OPSFLEET_SKIP_REMOTE")), "1") {
		return nil
	}
	var out []string
	add := func(b string) {
		b = strings.TrimRight(strings.TrimSpace(b), "/")
		if b == "" {
			return
		}
		for _, x := range out {
			if x == b {
				return
			}
		}
		out = append(out, b)
	}
	add(os.Getenv("OPSFLEET_API_URL"))
	add(config.LoadOptionalOpsfleetAPIBase())
	add(EmbeddedOpsfleetAPIBase)
	add(EmbeddedOpsfleetAPIBaseProduction)
	return out
}

// resolveOpsfleetToken 用于访问 OpsFleet 受保护 API（含服务端 AI）；优先环境变量，其次 install 脚本写入的 ~/.config/ai-sre/opsfleet_token。
func resolveOpsfleetToken() string {
	if v := strings.TrimSpace(os.Getenv("OPSFLEET_TOKEN")); v != "" {
		return v
	}
	return strings.TrimSpace(config.LoadOptionalOpsfleetToken())
}

func resolveOpsfleetBindingID() string {
	if v := strings.TrimSpace(os.Getenv("OPSFLEET_BINDING_ID")); v != "" {
		return v
	}
	return strings.TrimSpace(config.LoadOptionalOpsfleetBindingID())
}

func resolveOpsfleetFingerprint() string {
	if v := strings.TrimSpace(os.Getenv("OPSFLEET_CLI_FINGERPRINT")); v != "" {
		return strings.ToLower(v)
	}
	if v := strings.TrimSpace(config.LoadOptionalOpsfleetFingerprint()); v != "" {
		return strings.ToLower(v)
	}
	fp := computeOpsfleetFingerprint()
	if fp == "" {
		return ""
	}
	return fp
}

func computeOpsfleetFingerprint() string {
	machineID := strings.TrimSpace(readFirstExistingFile("/etc/machine-id", "/var/lib/dbus/machine-id"))
	if machineID == "" {
		return ""
	}
	host, _ := os.Hostname()
	osID, osVersion := readOSRelease()
	payload := "machine_id=" + machineID +
		"\nhostname=" + strings.TrimSpace(host) +
		"\nos_id=" + osID +
		"\nos_version=" + osVersion +
		"\narch=" + opsfleetArch()
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func opsfleetArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	default:
		return runtime.GOARCH
	}
}

func readFirstExistingFile(paths ...string) string {
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err == nil {
			if v := strings.TrimSpace(string(b)); v != "" {
				return v
			}
		}
	}
	return ""
}

func readOSRelease() (string, string) {
	b, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "unknown", "unknown"
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[key] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	id := strings.TrimSpace(values["ID"])
	if id == "" {
		id = "unknown"
	}
	version := strings.TrimSpace(values["VERSION_ID"])
	if version == "" {
		version = "unknown"
	}
	return id, version
}
