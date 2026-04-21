package handlers

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"ft-backend/common/config"
	"ft-backend/common/response"

	"github.com/gin-gonic/gin"
)

// publicAPIBaseFromRequest 构造浏览器访问的 API 基址（含 /ft-api），供安装脚本内 curl 使用。
func publicAPIBaseFromRequest(c *gin.Context) string {
	scheme := "http"
	if c.GetHeader("X-Forwarded-Proto") == "https" || c.Request.TLS != nil {
		scheme = "https"
	}
	host := strings.TrimSpace(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = c.Request.Host
	}
	if host == "" {
		host = "127.0.0.1:8080"
	}
	host = strings.TrimRight(host, "/")
	// 反代若只传 hostname（Nginx 曾用 $host 会丢端口），用 X-Forwarded-Port 补上，避免脚本内 curl 默认打 80
	if _, _, err := net.SplitHostPort(host); err != nil {
		if p := strings.TrimSpace(c.GetHeader("X-Forwarded-Port")); p != "" {
			host = net.JoinHostPort(host, p)
		}
	}
	return fmt.Sprintf("%s://%s/ft-api", scheme, host)
}

// ServeAiSreInstallScript 公开：在控制机执行 curl 管道安装 ai-sre 到 /usr/local/bin。
func ServeAiSreInstallScript(c *gin.Context) {
	base := publicAPIBaseFromRequest(c)
	body := fmt.Sprintf(`#!/usr/bin/env bash
# OpsFleet 固定安装方式：将 ai-sre 安装到 /usr/local/bin（需服务器配置 opsfleet.ai_sre_binary_path）
set -euo pipefail
API_BASE=%s
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) UARCH=amd64 ;;
  aarch64|arm64) UARCH=arm64 ;;
  *) echo "不支持的架构: $ARCH（需 amd64 或 arm64）" >&2; exit 1 ;;
esac
TMP=$(mktemp)
trap 'rm -f "$TMP"' EXIT
if ! curl -fsSL "$API_BASE/api/k8s/deploy/cli/ai-sre?arch=$UARCH" -o "$TMP"; then
  echo "下载 ai-sre 失败。请在 OpsFleet 服务器 ft-backend/conf/config.yaml 中配置 opsfleet.ai_sre_binary_path（指向已构建的 Linux 可执行文件）。" >&2
  exit 1
fi
install -m 0755 "$TMP" /usr/local/bin/ai-sre
echo "已安装: $(command -v ai-sre)"
ai-sre version
`, quoteShellSingleLine(base))

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=120")
	c.String(http.StatusOK, body)
}

func quoteShellSingleLine(s string) string {
	return `'` + strings.ReplaceAll(s, `'`, `'"'"'`) + `'`
}

// DownloadAiSreCLI 公开：下载已配置的 ai-sre Linux 二进制（与架构参数校验可选）。
func DownloadAiSreCLI(c *gin.Context) {
	cfg, ok := c.MustGet("config").(*config.Config)
	if !ok || cfg == nil {
		response.ServerError(c, "配置未初始化")
		return
	}
	// 环境变量优先：全栈部署脚本每次写入 /etc/opsfleet/backend.env，保证发布即最新 bin/ai-sre
	path := strings.TrimSpace(os.Getenv("OPSFLEET_AISRE_BINARY_PATH"))
	if path == "" {
		path = strings.TrimSpace(cfg.Opsfleet.AiSreBinaryPath)
	}
	if path == "" {
		response.BadRequest(c, "未配置 ai-sre 分发：请在 conf/config.yaml 设置 opsfleet.ai_sre_binary_path，或环境变量 OPSFLEET_AISRE_BINARY_PATH（指向 Linux 可执行文件）")
		return
	}
	path = filepath.Clean(path)
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		response.NotFound(c, "ai_sre_binary_path 无效或不是文件: "+path)
		return
	}
	arch := strings.TrimSpace(strings.ToLower(c.Query("arch")))
	if arch != "" && arch != "amd64" && arch != "arm64" {
		response.BadRequest(c, "arch 仅支持 amd64 或 arm64")
		return
	}
	// 若配置了路径，当前仅分发该文件；arch 用于将来扩展多文件时校验
	data, err := os.ReadFile(path)
	if err != nil {
		response.ServerError(c, "读取二进制失败")
		return
	}
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", `attachment; filename="ai-sre"`)
	c.Data(http.StatusOK, "application/octet-stream", data)
}
