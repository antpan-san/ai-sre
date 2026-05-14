package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"debug/elf"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"ft-backend/common/config"
	"ft-backend/common/logger"
	"ft-backend/common/response"
	"ft-backend/database"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const cliInstallSessionTTL = 15 * time.Minute
const cliBindingTTL = 90 * 24 * time.Hour

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

func buildAiSreInstallScriptBody(apiBase string) string {
	return fmt.Sprintf(`#!/usr/bin/env bash
# OpsFleet：将 ai-sre 安装/升级到 /usr/local/bin，并保存 API 基址供本机后续每次执行 ai-sre 时联网比对版本并自升级。
set -euo pipefail
API_BASE=%s
ARCH=$(uname -m)
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
echo "已写入: $(command -v ai-sre)；已记录 OpsFleet 基址 $UHOME/.config/ai-sre/opsfleet_api_url（供自升级）"
if ! ai-sre version; then
  echo "已写入 /usr/local/bin/ai-sre，但 version 未成功（多为架构与分发 ELF 不一致）。请让管理员在 OpsFort 配置与本机 $ARCH 一致的二进制。" >&2
fi
`, quoteShellSingleLine(apiBase))
}

// ServeAiSreInstallScript 公开：在控制机执行 curl 管道安装 ai-sre 到 /usr/local/bin（不绑定账号；服务端 AI 仍走匿名 IP 限额）。
func ServeAiSreInstallScript(c *gin.Context) {
	base := publicAPIBaseFromRequest(c)
	body := buildAiSreInstallScriptBody(base)

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=120")
	c.String(http.StatusOK, body)
}

// ServeAiSreInstallScriptForUser 兼容旧命令：仍要求 JWT，但不再把浏览器 JWT 写入磁盘。
// 每次请求都会创建一次性安装会话，并返回专用 CLI token 绑定脚本。
func ServeAiSreInstallScriptForUser(c *gin.Context) {
	session, token, ok := createCLIInstallSessionForContext(c)
	if !ok {
		return
	}
	base := publicAPIBaseFromRequest(c)
	body := buildAiSreInstallSessionScriptBody(base, token, session.ExpiresAt)

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "private, no-store")
	c.String(http.StatusOK, body)
}

func CreateCLIInstallSession(c *gin.Context) {
	session, token, ok := createCLIInstallSessionForContext(c)
	if !ok {
		return
	}
	base := publicAPIBaseFromRequest(c)
	url := base + "/api/cli/install-ai-sre.sh"
	command := fmt.Sprintf("curl -fsSL -H 'X-OpsFleet-Install-Token: %s' '%s' | sudo bash", token, url)
	response.OK(c, gin.H{
		"command":    command,
		"expires_at": session.ExpiresAt,
	})
}

func createCLIInstallSessionForContext(c *gin.Context) (models.CLIInstallSession, string, bool) {
	uidVal, ok := c.Get("userID")
	if !ok {
		response.Unauthorized(c, "缺少登录用户")
		return models.CLIInstallSession{}, "", false
	}
	uid := models.UserIDFromContext(uidVal)
	if uid == uuid.Nil {
		response.Unauthorized(c, "登录用户无效")
		return models.CLIInstallSession{}, "", false
	}
	username, _ := c.Get("username")
	token, err := randomTokenHex(32)
	if err != nil {
		response.ServerError(c, "生成安装会话失败")
		return models.CLIInstallSession{}, "", false
	}
	session := models.CLIInstallSession{
		UserID:    uid,
		Username:  strings.TrimSpace(fmt.Sprint(username)),
		TokenHash: hashSecret(token),
		Status:    models.CLIInstallSessionStatusPending,
		ExpiresAt: time.Now().UTC().Add(cliInstallSessionTTL),
	}
	if err := database.DB.Create(&session).Error; err != nil {
		logger.Error("create cli install session: %v", err)
		response.ServerError(c, "保存安装会话失败")
		return models.CLIInstallSession{}, "", false
	}
	return session, token, true
}

func ServeAiSreInstallScriptForSession(c *gin.Context) {
	token := strings.TrimSpace(c.GetHeader("X-OpsFleet-Install-Token"))
	session, ok := loadPendingCLIInstallSession(c, token)
	if !ok {
		return
	}
	base := publicAPIBaseFromRequest(c)
	body := buildAiSreInstallSessionScriptBody(base, token, session.ExpiresAt)

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "private, no-store")
	c.String(http.StatusOK, body)
}

type cliInstallBindRequest struct {
	InstallToken    string `json:"install_token"`
	FingerprintHash string `json:"fingerprint_hash"`
	Hostname        string `json:"hostname"`
	OS              string `json:"os"`
	Arch            string `json:"arch"`
	InstallUser     string `json:"install_user"`
	Version         string `json:"version"`
}

func BindCLIInstallSession(c *gin.Context) {
	var req cliInstallBindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数: "+err.Error())
		return
	}
	req.InstallToken = strings.TrimSpace(req.InstallToken)
	req.FingerprintHash = strings.ToLower(strings.TrimSpace(req.FingerprintHash))
	if req.InstallToken == "" || !isHexLen(req.FingerprintHash, 64) {
		response.BadRequest(c, "安装 token 或机器指纹无效")
		return
	}

	now := time.Now().UTC()
	var out gin.H
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var session models.CLIInstallSession
		if err := tx.Where("token_hash = ?", hashSecret(req.InstallToken)).First(&session).Error; err != nil {
			return err
		}
		if session.Status != models.CLIInstallSessionStatusPending || session.UsedAt != nil {
			return fmt.Errorf("used")
		}
		if now.After(session.ExpiresAt) {
			_ = tx.Model(&session).Updates(map[string]interface{}{"status": models.CLIInstallSessionStatusExpired}).Error
			return fmt.Errorf("expired")
		}

		cliToken, err := randomTokenHex(32)
		if err != nil {
			return err
		}
		binding := models.CLIBinding{
			UserID:          session.UserID,
			Username:        session.Username,
			TokenHash:       hashSecret(cliToken),
			FingerprintHash: req.FingerprintHash,
			Hostname:        limitRunes(strings.TrimSpace(req.Hostname), 200),
			OS:              limitRunes(strings.TrimSpace(req.OS), 120),
			Arch:            limitRunes(strings.TrimSpace(req.Arch), 40),
			InstallUser:     limitRunes(strings.TrimSpace(req.InstallUser), 80),
			Version:         limitRunes(strings.TrimSpace(req.Version), 40),
			ExpiresAt:       now.Add(cliBindingTTL),
		}
		if err := tx.Create(&binding).Error; err != nil {
			return err
		}
		usedAt := now
		if err := tx.Model(&session).Updates(map[string]interface{}{
			"status":                models.CLIInstallSessionStatusUsed,
			"used_at":               usedAt,
			"used_fingerprint_hash": req.FingerprintHash,
			"cli_binding_id":        binding.ID,
		}).Error; err != nil {
			return err
		}
		reportToken, reportHash, err := newExecutionReportToken()
		if err != nil {
			return err
		}
		rec := models.ExecutionRecord{
			CorrelationID: uuid.NewString(),
			Source:        "install",
			Category:      "install_ai_sre",
			Name:          "安装 ai-sre",
			Command:       "install-ai-sre.sh",
			CommandDigest: digestText("install-ai-sre.sh"),
			Status:        models.ExecutionStatusRunning,
			CreatedBy:     session.Username,
			TriggerUser:   session.Username,
			TargetHost:    binding.Hostname,
			StartedAt:     &now,
			Effects:       models.NewJSONBFromMap(map[string]interface{}{}),
			Metadata: models.NewJSONBFromMap(map[string]interface{}{
				"record_kind":      "cli_install",
				"cli_binding_id":   binding.ID.String(),
				"fingerprint_hash": req.FingerprintHash,
				"os":               binding.OS,
				"arch":             binding.Arch,
				"install_user":     binding.InstallUser,
			}),
			RollbackCapability: models.RollbackCapabilityNone,
			RollbackStatus:     models.RollbackStatusNotStarted,
			RollbackPlan:       models.NewJSONBFromMap(map[string]interface{}{}),
			RollbackAdvice:     "如需移除 CLI，请删除 /usr/local/bin/ai-sre 与 ~/.config/ai-sre/opsfleet_* 文件。",
			ReportTokenHash:    reportHash,
		}
		if err := tx.Create(&rec).Error; err != nil {
			return err
		}
		out = gin.H{
			"cli_token":      cliToken,
			"username":       session.Username,
			"binding_id":     binding.ID.String(),
			"expires_at":     binding.ExpiresAt,
			"record_id":      rec.ID.String(),
			"correlation_id": rec.CorrelationID,
			"report_token":   reportToken,
		}
		return nil
	})
	if err != nil {
		switch {
		case errorsIsRecordNotFound(err):
			response.Unauthorized(c, "安装会话无效")
		case err.Error() == "used":
			response.Conflict(c, "安装会话已使用，请重新登录控制台生成命令")
		case err.Error() == "expired":
			response.Unauthorized(c, "安装会话已过期，请重新生成命令")
		default:
			logger.Error("bind cli install session: %v", err)
			response.ServerError(c, "绑定 ai-sre 失败")
		}
		return
	}
	response.OK(c, out)
}

func loadPendingCLIInstallSession(c *gin.Context, token string) (models.CLIInstallSession, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		response.Unauthorized(c, "缺少安装 token")
		return models.CLIInstallSession{}, false
	}
	var session models.CLIInstallSession
	if err := database.DB.Where("token_hash = ?", hashSecret(token)).First(&session).Error; err != nil {
		response.Unauthorized(c, "安装会话无效")
		return models.CLIInstallSession{}, false
	}
	if session.Status != models.CLIInstallSessionStatusPending || session.UsedAt != nil {
		response.Conflict(c, "安装会话已使用，请重新生成命令")
		return models.CLIInstallSession{}, false
	}
	if time.Now().UTC().After(session.ExpiresAt) {
		_ = database.DB.Model(&session).Update("status", models.CLIInstallSessionStatusExpired).Error
		response.Unauthorized(c, "安装会话已过期，请重新生成命令")
		return models.CLIInstallSession{}, false
	}
	return session, true
}

func buildAiSreInstallSessionScriptBody(apiBase, installToken string, expiresAt time.Time) string {
	body := `#!/usr/bin/env bash
set -euo pipefail
API_BASE=__API_BASE__
INSTALL_TOKEN=__INSTALL_TOKEN__
INSTALL_EXPIRES_AT=__INSTALL_EXPIRES_AT__
REPORT_RECORD_ID=""
REPORT_TOKEN=""
REPORT_CORRELATION_ID=""

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || { echo "缺少依赖: $1" >&2; exit 1; }
}

sha256_text() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 | awk '{print $1}'
  else
    echo "缺少 sha256sum 或 shasum" >&2
    exit 1
  fi
}

json_string() {
  python3 -c 'import json,sys; print(json.dumps(sys.argv[1]))' "$1"
}

finish_report() {
  code=$?
  trap - EXIT
  if [ -n "${REPORT_TOKEN:-}" ] && [ -n "${REPORT_CORRELATION_ID:-}" ]; then
    status="success"
    if [ "$code" -ne 0 ]; then status="failed"; fi
    version="$(ai-sre version 2>/dev/null | head -n 1 || true)"
    payload="$(REPORT_RECORD_ID="$REPORT_RECORD_ID" REPORT_CORRELATION_ID="$REPORT_CORRELATION_ID" REPORT_TOKEN="$REPORT_TOKEN" STATUS="$status" CODE="$code" VERSION="$version" python3 - <<'PY'
import json, os
print(json.dumps({
  "record_id": os.environ.get("REPORT_RECORD_ID", ""),
  "correlation_id": os.environ.get("REPORT_CORRELATION_ID", ""),
  "token": os.environ.get("REPORT_TOKEN", ""),
  "status": os.environ.get("STATUS", "failed"),
  "exit_code": int(os.environ.get("CODE", "1")),
  "stdout_summary": os.environ.get("VERSION", ""),
  "metadata": {"record_kind": "cli_install", "version": os.environ.get("VERSION", "")}
}))
PY
)"
    curl -fsS -m 3 -X POST "$API_BASE/api/execution-records/report/finish" -H 'Content-Type: application/json' --data "$payload" >/dev/null 2>&1 || true
  fi
  exit "$code"
}

need_cmd curl
need_cmd python3

ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) UARCH=amd64 ;;
  aarch64|arm64) UARCH=arm64 ;;
  *) echo "不支持的架构: $ARCH（需 amd64 或 arm64）" >&2; exit 1 ;;
esac

HOSTNAME_VALUE="$(hostname 2>/dev/null || echo unknown)"
MACHINE_ID=""
if [ -r /etc/machine-id ]; then MACHINE_ID="$(cat /etc/machine-id 2>/dev/null || true)"; fi
if [ -z "$MACHINE_ID" ] && [ -r /var/lib/dbus/machine-id ]; then MACHINE_ID="$(cat /var/lib/dbus/machine-id 2>/dev/null || true)"; fi
if [ -z "$MACHINE_ID" ]; then echo "缺少机器身份文件 /etc/machine-id" >&2; exit 1; fi
OS_ID="unknown"
OS_VERSION="unknown"
if [ -r /etc/os-release ]; then
  . /etc/os-release
  OS_ID="${ID:-unknown}"
  OS_VERSION="${VERSION_ID:-unknown}"
fi
INSTALL_USER="${SUDO_USER:-${USER:-unknown}}"
FP_PAYLOAD="$(printf 'machine_id=%s\nhostname=%s\nos_id=%s\nos_version=%s\narch=%s' "$MACHINE_ID" "$HOSTNAME_VALUE" "$OS_ID" "$OS_VERSION" "$UARCH")"
FINGERPRINT_HASH="$(printf '%s' "$FP_PAYLOAD" | sha256_text)"

BIND_PAYLOAD="$(INSTALL_TOKEN="$INSTALL_TOKEN" FINGERPRINT_HASH="$FINGERPRINT_HASH" HOSTNAME_VALUE="$HOSTNAME_VALUE" OS_NAME="$OS_ID $OS_VERSION" UARCH="$UARCH" INSTALL_USER="$INSTALL_USER" python3 - <<'PY'
import json, os
print(json.dumps({
  "install_token": os.environ["INSTALL_TOKEN"],
  "fingerprint_hash": os.environ["FINGERPRINT_HASH"],
  "hostname": os.environ.get("HOSTNAME_VALUE", ""),
  "os": os.environ.get("OS_NAME", ""),
  "arch": os.environ.get("UARCH", ""),
  "install_user": os.environ.get("INSTALL_USER", ""),
  "version": ""
}))
PY
)"
BIND_RESP="$(curl -fsS -X POST "$API_BASE/api/cli/install-bind" -H 'Content-Type: application/json' --data "$BIND_PAYLOAD")" || {
  echo "绑定 ai-sre 安装会话失败；该命令可能已使用、过期，或被复制到其它机器。" >&2
  exit 1
}
eval "$(BIND_RESP="$BIND_RESP" python3 - <<'PY'
import json, os, shlex, sys
env = json.loads(os.environ["BIND_RESP"])
if env.get("code") != 200:
    print("echo " + shlex.quote(env.get("msg", "绑定失败")) + " >&2")
    print("exit 1")
    sys.exit(0)
d = env.get("data") or {}
for key, src in {
    "CLI_TOKEN": "cli_token",
    "OPSFLEET_USERNAME": "username",
    "CLI_BINDING_ID": "binding_id",
    "REPORT_RECORD_ID": "record_id",
    "REPORT_CORRELATION_ID": "correlation_id",
    "REPORT_TOKEN": "report_token",
}.items():
    print(f"{key}={shlex.quote(str(d.get(src, '')))}")
PY
)"
trap finish_report EXIT

if command -v ai-sre >/dev/null 2>&1; then
  echo "正在从 OpsFleet 拉取并覆盖升级 ai-sre …" >&2
else
  echo "正在从 OpsFleet 拉取并安装最新 ai-sre …" >&2
fi
echo "本机 uname -m=$ARCH → 请求 OpsFleet 分发 arch=$UARCH" >&2
TMP=$(mktemp)
trap 'rm -f "$TMP"; finish_report' EXIT
DL_URL="$API_BASE/api/k8s/deploy/cli/ai-sre?arch=$UARCH"
DL_OPTS=("--fail" "--location" "--retry" "3" "--retry-delay" "2" "--connect-timeout" "10")
if [ -t 2 ] && [ "${OPSFLEET_NO_PROGRESS:-}" != "1" ]; then
  echo "下载地址: $DL_URL" >&2
  curl "${DL_OPTS[@]}" --progress-bar -o "$TMP" "$DL_URL"
else
  curl "${DL_OPTS[@]}" -sS -o "$TMP" "$DL_URL"
fi
SIZE_BYTES=$(stat -c %s "$TMP" 2>/dev/null || stat -f %z "$TMP" 2>/dev/null || echo "?")
echo "已下载: ${SIZE_BYTES} bytes" >&2
install -m 0755 "$TMP" /usr/local/bin/ai-sre
if [ -n "${SUDO_USER:-}" ]; then
  UHOME=$(eval echo "~$SUDO_USER")
else
  UHOME="${HOME:-/root}"
fi
mkdir -p "$UHOME/.config/ai-sre"
printf '%s\n' "$API_BASE" > "$UHOME/.config/ai-sre/opsfleet_api_url" || true
printf '%s\n' "$CLI_TOKEN" > "$UHOME/.config/ai-sre/opsfleet_token" || true
chmod 0600 "$UHOME/.config/ai-sre/opsfleet_token" 2>/dev/null || true
printf '%s\n' "$OPSFLEET_USERNAME" > "$UHOME/.config/ai-sre/opsfleet_username" || true
printf '%s\n' "$CLI_BINDING_ID" > "$UHOME/.config/ai-sre/opsfleet_binding_id" || true
printf '%s\n' "$FINGERPRINT_HASH" > "$UHOME/.config/ai-sre/opsfleet_fingerprint" || true
echo "已写入: $(command -v ai-sre)；已绑定当前机器与账号 $OPSFLEET_USERNAME。" >&2
ai-sre version
`
	body = strings.ReplaceAll(body, "__API_BASE__", quoteShellSingleLine(apiBase))
	body = strings.ReplaceAll(body, "__INSTALL_TOKEN__", quoteShellSingleLine(installToken))
	body = strings.ReplaceAll(body, "__INSTALL_EXPIRES_AT__", quoteShellSingleLine(expiresAt.Format(time.RFC3339)))
	return body
}

func randomTokenHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashSecret(s string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(s)))
	return hex.EncodeToString(sum[:])
}

func isHexLen(s string, n int) bool {
	if len(s) != n {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

func limitRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

func errorsIsRecordNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
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
