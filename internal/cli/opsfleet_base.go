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
// 目标机无需 export、无需写 ~/.config，直接执行 ai-sre（含 uninstall k8s、自升级）即可连到该地址。
// 仅当设置环境变量 OPSFLEET_API_URL 时覆盖（例如联调其它环境）。
const EmbeddedOpsfleetAPIBase = "http://192.168.56.11:9080/ft-api"

func resolveOpsfleetAPIBase() string {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("OPSFLEET_SKIP_REMOTE")), "1") {
		return ""
	}
	if v := strings.TrimSpace(os.Getenv("OPSFLEET_API_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	if v := strings.TrimSpace(config.LoadOptionalOpsfleetAPIBase()); v != "" {
		return strings.TrimRight(v, "/")
	}
	return strings.TrimRight(EmbeddedOpsfleetAPIBase, "/")
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
