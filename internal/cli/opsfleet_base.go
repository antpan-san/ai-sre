package cli

import (
	"os"
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
