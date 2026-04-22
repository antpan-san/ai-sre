package cli

import (
	"os"
	"strings"
)

// EmbeddedOpsfleetAPIBase 内建 OpsFort 控制台 API 基址（须含 /ft-api 前缀，与 Nginx 反代一致）。
// 目标机无需 export、无需写 ~/.config，直接执行 ai-sre（含 uninstall k8s、自升级）即可连到该地址。
// 仅当设置环境变量 OPSFLEET_API_URL 时覆盖（例如联调其它环境）。
const EmbeddedOpsfleetAPIBase = "http://192.168.56.11:9080/ft-api"

func resolveOpsfleetAPIBase() string {
	if v := strings.TrimSpace(os.Getenv("OPSFLEET_API_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return strings.TrimRight(EmbeddedOpsfleetAPIBase, "/")
}
