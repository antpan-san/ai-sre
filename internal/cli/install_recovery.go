package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

// recoverInstallDownloadFailure 在客户端安装/下载失败后**必须**走 OpsFleet 服务端 AI，禁止仅打印本地 error 后结束。
// 成功返回 true：已输出服务端方案，调用方应继续或正常退出（勿再把 cause 原样抛给用户）。
func recoverInstallDownloadFailure(ctx context.Context, operation string, cause error, extra map[string]string) bool {
	if cause == nil {
		return false
	}
	if os.Getenv("OPSFLEET_SKIP_INSTALL_AI_RECOVERY") == "1" {
		printInstallManualFallback(operation, cause, extra)
		return true
	}
	base := strings.TrimSpace(resolveOpsfleetAPIBase())
	kv := map[string]string{
		"operation":     strings.TrimSpace(operation),
		"error":         cause.Error(),
		"local_version": Version,
		"arch":          goArchToAiSreArch(),
		"api_base":      base,
	}
	for k, v := range extra {
		if strings.TrimSpace(v) != "" {
			kv[k] = v
		}
	}
	if ex, err := os.Executable(); err == nil {
		kv["dest_path"] = ex
	}

	var answer string
	var source string
	if base != "" {
		rctx, cancel := context.WithTimeout(ctx, 90*time.Second)
		resp, err := callServerDiagnose(rctx, diagnoseRequest{
			Topic:     "install",
			Context:   kv,
			Command:   strings.Join(os.Args, " "),
			RequestID: uuid.NewString(),
			Client:    opsfleetAIClient(),
			Intent:    buildExecutionIntent("check", "install", kv),
		})
		cancel()
		if err == nil && resp != nil && strings.TrimSpace(resp.Answer) != "" {
			answer = strings.TrimSpace(resp.Answer)
			source = "diagnose"
		}
	}
	if answer == "" && base != "" {
		q := fmt.Sprintf(
			"ai-sre 客户端%s失败：%s。上下文：arch=%s local=%s api=%s。请给出可执行的 curl 安装命令与排查步骤。",
			kv["operation"], kv["error"], kv["arch"], kv["local_version"], kv["api_base"],
		)
		rctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		ans, err := callServerAsk(rctx, q, true)
		cancel()
		if err == nil && strings.TrimSpace(ans) != "" {
			answer = strings.TrimSpace(ans)
			source = "ask"
		}
	}
	if answer != "" {
		_, _ = fmt.Fprintf(os.Stderr, "[%s] 本机%s未成功（%v）；已由 OpsFleet 服务端 AI（%s）给出处置方案：\n\n",
			progName, operationLabel(operation), cause, source)
		fmt.Println(answer)
		_, _ = fmt.Fprintln(os.Stderr, "\n（升级/安装未自动完成时，请按上述步骤操作后执行 ai-sre version 核对）")
		return true
	}
	printInstallManualFallback(operation, cause, extra)
	return true
}

func operationLabel(op string) string {
	switch strings.ToLower(strings.TrimSpace(op)) {
	case "auto_upgrade", "upgrade":
		return "升级"
	case "version_check":
		return "版本检查"
	default:
		return "安装/下载"
	}
}

func printInstallManualFallback(operation string, cause error, extra map[string]string) {
	base := strings.TrimSpace(resolveOpsfleetAPIBase())
	if base == "" {
		base = EmbeddedOpsfleetAPIBaseProduction
	}
	arch := goArchToAiSreArch()
	if v := extra["arch"]; strings.TrimSpace(v) != "" {
		arch = strings.TrimSpace(v)
	}
	dest := "/usr/local/bin/ai-sre"
	if v := extra["dest_path"]; strings.TrimSpace(v) != "" {
		dest = strings.TrimSpace(v)
	}
	_, _ = fmt.Fprintf(os.Stderr, "[%s] 本机%s失败（%v）；服务端 AI 暂不可用，改用手工命令：\n\n", progName, operationLabel(operation), cause)
	fmt.Printf(`export OPSFLEET_API_URL=%s
curl -fSL --connect-timeout 30 --max-time 600 -o %s.new \
  "%s/api/k8s/deploy/cli/ai-sre?arch=%s"
chmod 755 %s.new && mv -f %s.new %s
%s version
`, base, dest, base, arch, dest, dest, dest, progName)
}
