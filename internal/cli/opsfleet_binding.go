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
	base, err := resolveOpsfleetAPIBaseStrict()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "[%s] %v\n", progName, err)
		return
	}
	if base == "" {
		return
	}
	_ = tryAutoUpgradeInPlace(base)
	if upgradeCheckVerbose() {
		_, _ = fmt.Fprintf(os.Stderr, "[%s] 版本检查 %s\n", progName, opsfleetEnvLabel(base))
		reportUpgradeCheckResult(base)
	}
}

func validateOpsfleetCredentials() error {
	if _, err := resolveOpsfleetAPIBaseStrict(); err != nil {
		return err
	}
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
	hint := "在本机执行 install-ai-sre.sh 绑定（api_key 不能代替 opsfleet_token）"
	if hasLocalAPIKeyOnly() {
		hint = "本机仅有 api_key；请在当前环境重新 install-ai-sre 绑定"
	}
	return fmt.Errorf("缺少 CLI token（%s）: %s", opsfleetEnvLabel(apiBase), hint)
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
	if opsfleetAPIErrorAlreadyFormatted(msg) {
		return err
	}
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "cli token 无效") || strings.Contains(lower, "token 无效") {
		base := strings.TrimSpace(resolveOpsfleetAPIBase())
		return fmt.Errorf("%s（%s%s）: 请在本机重新 install-ai-sre 绑定",
			msg, opsfleetEnvLabel(base), endpointSuffix(endpoint))
	}
	if strings.Contains(lower, "cli token 与当前机器不匹配") || strings.Contains(lower, "机器不匹配") {
		return fmt.Errorf("%s: 指纹不匹配，请在本机重新 install-ai-sre", msg)
	}
	return err
}

func opsfleetAPIErrorAlreadyFormatted(msg string) bool {
	return strings.Contains(msg, "install-ai-sre 绑定") || strings.Contains(msg, "install-ai-sre.sh")
}

func endpointSuffix(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ""
	}
	return " " + endpoint
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
