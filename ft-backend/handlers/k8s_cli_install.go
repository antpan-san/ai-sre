package handlers

import (
	"context"
	"debug/elf"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"ft-backend/common/config"
	"ft-backend/common/response"

	"github.com/gin-gonic/gin"
)

type aiSreInstallBind struct {
	JWT      string
	Username string
}

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

func buildAiSreInstallScriptBody(apiBase string, bind *aiSreInstallBind) string {
	bindHeader := ""
	bindFooter := ""
	if bind != nil && strings.TrimSpace(bind.JWT) != "" {
		bindHeader = fmt.Sprintf("OPSFLEET_BIND_JWT=%s\nOPSFLEET_BIND_USER=%s\n",
			quoteShellSingleLine(bind.JWT), quoteShellSingleLine(strings.TrimSpace(bind.Username)))
		bindFooter = `
if [ -n "${OPSFLEET_BIND_JWT:-}" ]; then
  printf '%s\n' "$OPSFLEET_BIND_JWT" > "$UHOME/.config/ai-sre/opsfleet_token" || true
  chmod 0600 "$UHOME/.config/ai-sre/opsfleet_token" 2>/dev/null || true
  printf '%s\n' "$OPSFLEET_BIND_USER" > "$UHOME/.config/ai-sre/opsfleet_username" || true
  echo "已写入登录令牌：本机 ai-sre 调用 OpsFleet 服务端 AI 时将按当前控制台账号计费与限额；访问令牌过期后请在控制台重新复制安装命令。" >&2
fi
`
	}
	return fmt.Sprintf(`#!/usr/bin/env bash
# OpsFleet：将 ai-sre 安装/升级到 /usr/local/bin，并保存 API 基址供本机后续每次执行 ai-sre 时联网比对版本并自升级。
set -euo pipefail
API_BASE=%s
%sARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) UARCH=amd64 ;;
  aarch64|arm64) UARCH=arm64 ;;
  *) echo "不支持的架构: $ARCH（需 amd64 或 arm64）" >&2; exit 1 ;;
esac
if command -v ai-sre >/dev/null 2>&1; then
  echo "正在从 OpsFleet 拉取并覆盖升级 ai-sre …" >&2
else
  echo "正在从 OpsFleet 拉取并安装最新 ai-sre …" >&2
fi
echo "本机 uname -m=$ARCH → 请求 OpsFleet 分发 arch=$UARCH" >&2
TMP=$(mktemp)
trap 'rm -f "$TMP"' EXIT
DL_URL="$API_BASE/api/k8s/deploy/cli/ai-sre?arch=$UARCH"
DL_OPTS=("--fail" "--location" "--retry" "3" "--retry-delay" "2" "--connect-timeout" "10")
if [ -t 2 ] && [ "${OPSFLEET_NO_PROGRESS:-}" != "1" ]; then
  echo "下载地址: $DL_URL" >&2
  curl "${DL_OPTS[@]}" --progress-bar -o "$TMP" "$DL_URL" || { echo "下载失败。若 HTTP 400：多为 OpsFleet 未配置对应架构的 ai-sre（ARM 机需服务端 OPSFLEET_AISRE_BINARY_PATH_ARM64 或 bin/ai-sre.arm64）。仍请检查 conf 中 opsfleet.ai_sre_binary_path*。" >&2; exit 1; }
else
  curl "${DL_OPTS[@]}" -sS -o "$TMP" "$DL_URL" || { echo "下载失败（同上提示）。" >&2; exit 1; }
fi
SIZE_BYTES=$(stat -c %%s "$TMP" 2>/dev/null || stat -f %%z "$TMP" 2>/dev/null || echo "?")
echo "已下载: ${SIZE_BYTES} bytes" >&2
if command -v file >/dev/null 2>&1; then
  case "$UARCH" in
    amd64)
      if ! file -b "$TMP" 2>/dev/null | grep -qi 'x86-64'; then
        echo "警告: file 显示该下载物可能不是 x86-64 ELF，若执行失败请核对 OpsFleet 分发路径。" >&2
      fi
      ;;
    arm64)
      if ! file -b "$TMP" 2>/dev/null | grep -qE 'aarch64|ARM aarch64|ARM, EABI5'; then
        echo "警告: file 显示该下载物可能不是 aarch64 ELF，若执行失败请核对 OpsFleet 是否配置了 ARM64 分发。" >&2
      fi
      ;;
  esac
fi
install -m 0755 "$TMP" /usr/local/bin/ai-sre
if [ -n "${SUDO_USER:-}" ]; then
  UHOME=$(eval echo "~$SUDO_USER")
else
  UHOME="${HOME:-/root}"
fi
mkdir -p "$UHOME/.config/ai-sre"
printf '%%s\n' "$API_BASE" > "$UHOME/.config/ai-sre/opsfleet_api_url" || true
%s
echo "已写入: $(command -v ai-sre)；已记录 OpsFleet 基址 $UHOME/.config/ai-sre/opsfleet_api_url（供自升级）"
if ! ai-sre version; then
  echo "已写入 /usr/local/bin/ai-sre，但 version 未成功（多为架构与分发 ELF 不一致）。请让管理员在 OpsFort 配置与本机 $ARCH 一致的二进制。" >&2
fi
`, quoteShellSingleLine(apiBase), bindHeader, bindFooter)
}

// bearerTokenFromAuthorization 从 Authorization: Bearer … 中取出原始 token（不做校验，由 JWT 中间件负责）。
func bearerTokenFromAuthorization(c *gin.Context) string {
	h := strings.TrimSpace(c.GetHeader("Authorization"))
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 3)
	if len(parts) < 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// ServeAiSreInstallScript 公开：在控制机执行 curl 管道安装 ai-sre 到 /usr/local/bin（不绑定账号；服务端 AI 仍走匿名 IP 限额）。
func ServeAiSreInstallScript(c *gin.Context) {
	base := publicAPIBaseFromRequest(c)
	body := buildAiSreInstallScriptBody(base, nil)

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=120")
	c.String(http.StatusOK, body)
}

// ServeAiSreInstallScriptForUser 需 JWT：生成与公开脚本相同的安装逻辑，并额外写入当前会话访问令牌与用户名，供本机 ai-sre 关联订阅与 AI 配额。
func ServeAiSreInstallScriptForUser(c *gin.Context) {
	tok := bearerTokenFromAuthorization(c)
	if tok == "" {
		response.Unauthorized(c, "缺少 Authorization Bearer token")
		return
	}
	u, _ := c.Get("username")
	username, _ := u.(string)
	base := publicAPIBaseFromRequest(c)
	body := buildAiSreInstallScriptBody(base, &aiSreInstallBind{JWT: tok, Username: username})

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "private, no-store")
	c.String(http.StatusOK, body)
}

func quoteShellSingleLine(s string) string {
	return `'` + strings.ReplaceAll(s, `'`, `'"'"'`) + `'`
}

// readELFArch 返回 Linux ELF 的 amd64 / arm64，用于校验 ?arch= 与磁盘文件一致。
func readELFArch(path string) (string, error) {
	f, err := elf.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	switch f.Machine {
	case elf.EM_X86_64:
		return "amd64", nil
	case elf.EM_AARCH64:
		return "arm64", nil
	default:
		return "", fmt.Errorf("ELF machine %s not supported for ai-sre distribution", f.Machine.String())
	}
}

func aiSrePathIfFile(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = filepath.Clean(p)
	st, err := os.Stat(p)
	if err != nil || st.IsDir() {
		return ""
	}
	return p
}

// resolveAiSreBinaryPath 与旧逻辑一致：环境变量 OPSFLEET_AISRE_BINARY_PATH 优先于 config。
func resolveAiSreBinaryPath(cfg *config.Config) (path string) {
	if cfg != nil {
		path = strings.TrimSpace(os.Getenv("OPSFLEET_AISRE_BINARY_PATH"))
		if path == "" {
			path = strings.TrimSpace(cfg.Opsfleet.AiSreBinaryPath)
		}
	} else {
		path = strings.TrimSpace(os.Getenv("OPSFLEET_AISRE_BINARY_PATH"))
	}
	return aiSrePathIfFile(path)
}

// resolveAiSreBinaryPathForArch 按客户端请求的 arch 选取分发文件（amd64 / arm64）。
// arm64：优先 *_ARM64 / yaml arm64；否则若 legacy 单文件本身为 aarch64 ELF 则可用。
func resolveAiSreBinaryPathForArch(cfg *config.Config, wantArch string) string {
	wantArch = strings.TrimSpace(strings.ToLower(wantArch))
	if wantArch == "arm64" {
		if p := aiSrePathIfFile(os.Getenv("OPSFLEET_AISRE_BINARY_PATH_ARM64")); p != "" {
			return p
		}
		if cfg != nil {
			if p := aiSrePathIfFile(cfg.Opsfleet.AiSreBinaryPathArm64); p != "" {
				return p
			}
		}
		if lp := resolveAiSreBinaryPath(cfg); lp != "" {
			if got, err := readELFArch(lp); err == nil && got == "arm64" {
				return lp
			}
		}
		return ""
	}
	if wantArch == "amd64" {
		if p := aiSrePathIfFile(os.Getenv("OPSFLEET_AISRE_BINARY_PATH_AMD64")); p != "" {
			return p
		}
		if cfg != nil {
			if p := aiSrePathIfFile(cfg.Opsfleet.AiSreBinaryPathAmd64); p != "" {
				return p
			}
		}
		if lp := resolveAiSreBinaryPath(cfg); lp != "" {
			if got, err := readELFArch(lp); err == nil && got == "amd64" {
				return lp
			}
		}
		return ""
	}
	return ""
}

func firstAiSrePathWithProbe(cfg *config.Config) string {
	var candidates []string
	add := func(p string) {
		p = aiSrePathIfFile(p)
		if p == "" {
			return
		}
		for _, x := range candidates {
			if x == p {
				return
			}
		}
		candidates = append(candidates, p)
	}
	add(os.Getenv("OPSFLEET_AISRE_BINARY_PATH_AMD64"))
	add(os.Getenv("OPSFLEET_AISRE_BINARY_PATH_ARM64"))
	add(os.Getenv("OPSFLEET_AISRE_BINARY_PATH"))
	if cfg != nil {
		o := cfg.Opsfleet
		add(o.AiSreBinaryPathAmd64)
		add(o.AiSreBinaryPathArm64)
		add(o.AiSreBinaryPath)
	}
	for _, p := range candidates {
		if probeAiSreVersion(p) != "" {
			return p
		}
	}
	for _, p := range candidates {
		if p != "" {
			return p
		}
	}
	return ""
}

// probeAiSreVersion 优先环境变量，其次执行二进制 `version` 子命令解析第二列。
func probeAiSreVersion(bin string) string {
	if v := strings.TrimSpace(os.Getenv("OPSFLEET_AISRE_VERSION")); v != "" {
		return v
	}
	if bin == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "version").Output()
	if err != nil {
		return ""
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) >= 2 {
		return fields[1]
	}
	return ""
}

// GetAiSreCLIVersion 公开：返回当前分发的 ai-sre 版本号（供客户端升级前比对），轻量无大响应体。
func GetAiSreCLIVersion(c *gin.Context) {
	cfg, ok := c.MustGet("config").(*config.Config)
	if !ok || cfg == nil {
		response.ServerError(c, "配置未初始化")
		return
	}
	p := firstAiSrePathWithProbe(cfg)
	ver := probeAiSreVersion(p)
	if ver == "" {
		ver = "unknown"
	}
	if p == "" {
		c.JSON(http.StatusOK, gin.H{
			"name":    "ai-sre",
			"version": ver,
			"ok":      false,
			"message": "未配置 ai-sre 分发路径，无法执行版本探测；请在 conf 或环境变量中设置 opsfleet.ai_sre_binary_path / OPSFLEET_AISRE_BINARY_PATH，可选分架构 OPSFLEET_AISRE_BINARY_PATH_AMD64 / _ARM64 与 OPSFLEET_AISRE_VERSION。",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"name":    "ai-sre",
		"version": ver,
		"ok":      true,
	})
}

// DownloadAiSreCLI 公开：下载已配置的 ai-sre Linux 二进制；?arch= 与 ELF 必须一致。
func DownloadAiSreCLI(c *gin.Context) {
	cfg, ok := c.MustGet("config").(*config.Config)
	if !ok || cfg == nil {
		response.ServerError(c, "配置未初始化")
		return
	}
	want := strings.TrimSpace(strings.ToLower(c.Query("arch")))
	if want != "" && want != "amd64" && want != "arm64" {
		response.BadRequest(c, "arch 仅支持 amd64 或 arm64")
		return
	}
	var path string
	var enforceELFMatch bool
	if want == "" {
		path = resolveAiSreBinaryPath(cfg)
		enforceELFMatch = false
	} else {
		path = resolveAiSreBinaryPathForArch(cfg, want)
		enforceELFMatch = true
	}
	if path == "" {
		switch want {
		case "arm64":
			response.BadRequest(c, "未配置 ARM64 的 ai-sre：请在服务端设置 OPSFLEET_AISRE_BINARY_PATH_ARM64 或 opsfleet.ai_sre_binary_path_arm64（或由全栈构建生成 bin/ai-sre.arm64）。仅 amd64 分发无法在 aarch64 控制机上运行。")
		case "amd64":
			response.BadRequest(c, "未配置 AMD64 的 ai-sre：请在服务端设置 OPSFLEET_AISRE_BINARY_PATH_AMD64 或 opsfleet.ai_sre_binary_path_amd64。若 OpsFort 本机为 aarch64 且仅构建 arm64，须单独提供 x86_64 二进制或交叉构建后配置上述路径。")
		default:
			response.BadRequest(c, "未配置 ai-sre 分发：请在 conf/config.yaml 设置 opsfleet.ai_sre_binary_path，或环境变量 OPSFLEET_AISRE_BINARY_PATH（指向 Linux 可执行文件）")
		}
		return
	}
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		response.NotFound(c, "ai_sre_binary_path 无效或不是文件: "+path)
		return
	}
	if enforceELFMatch {
		got, err := readELFArch(path)
		if err != nil {
			response.BadRequest(c, "分发文件不是有效 Linux ELF: "+path+" ("+err.Error()+")")
			return
		}
		if got != want {
			response.BadRequest(c, fmt.Sprintf("分发与 arch 不一致: 请求 %s，但 %s 的 ELF 为 %s。请为对应架构单独配置 OPSFLEET_AISRE_BINARY_PATH_* 或上传匹配的二进制。", want, path, got))
			return
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		response.ServerError(c, "读取二进制失败")
		return
	}
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", `attachment; filename="ai-sre"`)
	c.Data(http.StatusOK, "application/octet-stream", data)
}
