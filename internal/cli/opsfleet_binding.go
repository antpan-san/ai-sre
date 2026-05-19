package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/panshuai/ai-sre/internal/config"
)

var upgradeCheckedThisProcess bool

// ensureUpgradeBeforeOpsfleetAPI runs the version check once per process before OpsFleet API calls.
// PersistentPreRun also upgrades, but some paths (or older installs) may skip it; this guarantees a visible first step.
func ensureUpgradeBeforeOpsfleetAPI() {
	if upgradeCheckedThisProcess {
		return
	}
	upgradeCheckedThisProcess = true
	if os.Getenv("OPSFLEET_NO_AUTO_UPGRADE") == "1" {
		return
	}
	base := strings.TrimSpace(resolveOpsfleetAPIBase())
	if base == "" {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "[%s] 检查 OpsFleet 版本更新（%s）…\n", progName, base)
	_ = tryAutoUpgradeInPlace("")
}

func validateOpsfleetCredentials() error {
	base := strings.TrimSpace(resolveOpsfleetAPIBase())
	if base == "" {
		return nil
	}
	tok := strings.TrimSpace(resolveOpsfleetToken())
	if tok != "" {
		if fp := strings.TrimSpace(resolveOpsfleetFingerprint()); fp == "" {
			return fmt.Errorf("已配置 OpsFleet CLI token，但无法生成本机指纹（需要可读 /etc/machine-id）；请在该机器上执行控制台 install-ai-sre 安装脚本")
		}
		return nil
	}
	return formatMissingOpsfleetTokenError(base)
}

func formatMissingOpsfleetTokenError(apiBase string) error {
	var b strings.Builder
	b.WriteString("未配置 OpsFleet CLI token，无法调用服务端能力（cli sync / check 等）\n")
	b.WriteString("  · 须在本机执行控制台 install-ai-sre.sh，会写入 ~/.config/ai-sre/opsfleet_token 与机器指纹\n")
	b.WriteString("  · config.yaml 中的 api_key 是 LLM（DeepSeek）凭据，不能代替 opsfleet_token\n")
	b.WriteString("  · 当前 API 基址: " + apiBase + "\n")
	if hasLocalAPIKeyOnly() {
		b.WriteString("  · 检测到本机仅有 api_key，无 opsfleet_token — 若只在实验环境配过 key，请在**当前执行 ai-sre 的机器**上重新安装绑定\n")
	}
	if env := strings.TrimSpace(os.Getenv("OPSFLEET_TOKEN")); env == "" {
		b.WriteString("  · 或设置环境变量 OPSFLEET_TOKEN=<控制台安装令牌>\n")
	}
	return fmt.Errorf("%s", strings.TrimSpace(b.String()))
}

func hasLocalAPIKeyOnly() bool {
	if strings.TrimSpace(resolveOpsfleetToken()) != "" {
		return false
	}
	cfgDir, err := config.ResolveDir()
	if err != nil {
		return false
	}
	if _, err := os.Stat(cfgDir + "/opsfleet_token"); err == nil {
		return false
	}
	if _, err := os.Stat(cfgDir + "/api_key"); err == nil {
		return true
	}
	yamlPath := cfgDir + "/config.yaml"
	if b, err := os.ReadFile(yamlPath); err == nil {
		if strings.Contains(string(b), "api_key:") && !strings.Contains(string(b), "opsfleet_token:") {
			return true
		}
	}
	return false
}

func formatOpsfleetAPIError(err error, endpoint string) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "cli token 无效") || strings.Contains(lower, "token 无效") {
		base := strings.TrimSpace(resolveOpsfleetAPIBase())
		var b strings.Builder
		b.WriteString(msg)
		b.WriteString("\n可能原因:\n")
		b.WriteString("  1) 本机 opsfleet_token 与当前控制台不一致（令牌须在**本机** install-ai-sre 生成）\n")
		b.WriteString("  2) OPSFLEET_API_URL / opsfleet_api_url 指向的环境与安装令牌时不一致\n")
		if base != "" {
			b.WriteString("  当前 API: " + base + "\n")
		}
		if endpoint != "" {
			b.WriteString("  请求: " + endpoint + "\n")
		}
		b.WriteString("建议: 在本机执行 ai-sre doctor --opsfleet 或重新 curl install-ai-sre.sh")
		return fmt.Errorf("%s", strings.TrimSpace(b.String()))
	}
	if strings.Contains(lower, "cli token 与当前机器不匹配") || strings.Contains(lower, "机器不匹配") {
		return fmt.Errorf("%s\n本机指纹与绑定时不一致；请在该机器上重新执行 install-ai-sre.sh", msg)
	}
	return err
}

// probeOpsfleetCLISync performs a lightweight GET /api/cli/sync for doctor --opsfleet.
func probeOpsfleetCLISync(ctx context.Context) error {
	ensureUpgradeBeforeOpsfleetAPI()
	if err := validateOpsfleetCredentials(); err != nil {
		return err
	}
	_, err := callCLISync(ctx)
	return formatOpsfleetAPIError(err, "/api/cli/sync")
}
