package cli

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/panshuai/ai-sre/internal/config"
)

// upgradeBaseWarnPrinted avoids repeating cross-env warnings every subcommand in one process.
var upgradeBaseWarnPrinted bool

const (
	opsfleetEnvLab        = "lab"
	opsfleetEnvProduction = "production"
	opsfleetEnvCustom     = "custom"
)

// EmbeddedOpsfleetAPIBase 实验室控制台（未 install 时的开发默认；禁止与生产混探测）。
const EmbeddedOpsfleetAPIBase = "http://192.168.56.11:9080/ft-api"

// EmbeddedOpsfleetAPIBaseProduction 生产控制台（仅 install 脚本或显式配置使用，不参与自动回退链）。
const EmbeddedOpsfleetAPIBaseProduction = "http://opsfleetpilot.com/ft-api"

func normalizeOpsfleetAPIBase(b string) string {
	b = strings.TrimSpace(b)
	if b == "" {
		return ""
	}
	return strings.TrimRight(b, "/")
}

// classifyOpsfleetBase 识别实验 / 生产 / 自定义基址。
func classifyOpsfleetBase(base string) string {
	lower := strings.ToLower(normalizeOpsfleetAPIBase(base))
	if lower == "" {
		return ""
	}
	if strings.Contains(lower, "192.168.56.11") {
		return opsfleetEnvLab
	}
	if strings.Contains(lower, "opsfleetpilot.com") {
		return opsfleetEnvProduction
	}
	return opsfleetEnvCustom
}

func opsfleetEnvLabel(base string) string {
	switch classifyOpsfleetBase(base) {
	case opsfleetEnvLab:
		return "实验 192.168.56.11"
	case opsfleetEnvProduction:
		return "生产 opsfleetpilot.com"
	default:
		if b := normalizeOpsfleetAPIBase(base); b != "" {
			return b
		}
		return "未配置"
	}
}

func opsfleetAPIBasesEquivalent(a, b string) bool {
	a = normalizeOpsfleetAPIBase(a)
	b = normalizeOpsfleetAPIBase(b)
	if a == b {
		return true
	}
	ka, kb := classifyOpsfleetBase(a), classifyOpsfleetBase(b)
	if ka == "" || kb == "" || ka == opsfleetEnvCustom || kb == opsfleetEnvCustom {
		return hostOfOpsfleetBase(a) == hostOfOpsfleetBase(b) && hostOfOpsfleetBase(a) != ""
	}
	return ka == kb
}

func hostOfOpsfleetBase(base string) string {
	u, err := url.Parse(normalizeOpsfleetAPIBase(base))
	if err != nil || u.Host == "" {
		return ""
	}
	return strings.ToLower(u.Host)
}

// resolveOpsfleetAPIBaseStrict 返回唯一绑定的控制台 API；禁止 OPSFLEET_API_URL 与 install 记录跨环境。
func resolveOpsfleetAPIBaseStrict() (string, error) {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("OPSFLEET_SKIP_REMOTE")), "1") {
		return "", nil
	}
	var envURL string
	if v := strings.TrimSpace(os.Getenv("OPSFLEET_API_URL")); v != "" {
		envURL = normalizeOpsfleetAPIBase(v)
	}
	fileURL := normalizeOpsfleetAPIBase(config.LoadOptionalOpsfleetAPIBase())
	if envURL != "" && fileURL != "" && !opsfleetAPIBasesEquivalent(envURL, fileURL) {
		return "", fmt.Errorf(
			"禁止混用实验与生产: OPSFLEET_API_URL=%s（%s）与 install 记录 %s（%s）不一致",
			envURL, opsfleetEnvLabel(envURL), fileURL, opsfleetEnvLabel(fileURL),
		)
	}
	if envURL != "" {
		return envURL, nil
	}
	if fileURL != "" {
		return fileURL, nil
	}
	return EmbeddedOpsfleetAPIBase, nil
}

func resolveOpsfleetAPIBase() string {
	b, _ := resolveOpsfleetAPIBaseStrict()
	return b
}

// resolveOpsfleetAPIBaseForUpgrade 解析用于版本探测/自动升级的 API 基址。
// 业务 API 仍须 resolveOpsfleetAPIBaseStrict；升级探测在环境冲突时优先 install 记录，避免静默跳过。
func resolveOpsfleetAPIBaseForUpgrade() (base string, warn string) {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("OPSFLEET_SKIP_REMOTE")), "1") {
		return "", ""
	}
	b, err := resolveOpsfleetAPIBaseStrict()
	if err == nil && b != "" {
		return b, ""
	}
	if err != nil {
		warn = err.Error()
	}
	fileURL := normalizeOpsfleetAPIBase(config.LoadOptionalOpsfleetAPIBase())
	envURL := normalizeOpsfleetAPIBase(os.Getenv("OPSFLEET_API_URL"))
	switch {
	case fileURL != "":
		return fileURL, warn
	case envURL != "":
		return envURL, warn
	default:
		return EmbeddedOpsfleetAPIBase, warn
	}
}

// resolveOpsfleetAPIBasesForUpgrade 仅返回当前绑定环境的一个基址（不再串联实验+生产）。
func resolveOpsfleetAPIBasesForUpgrade() []string {
	b, _ := resolveOpsfleetAPIBaseForUpgrade()
	if b == "" {
		return nil
	}
	return []string{b}
}
