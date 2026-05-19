package cli

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/panshuai/ai-sre/internal/config"
)

// upgradeBaseWarnPrinted avoids repeating cross-env warnings every subcommand in one process.
var upgradeBaseWarnPrinted bool

// autoBindingWarn is set when OPSFLEET_API_URL conflicts with install record and is auto-ignored.
var (
	autoBindingWarn      string
	autoBindingWarnShown bool
)

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
		autoBindingWarn = fmt.Sprintf(
			"已自动采用 install 记录 %s（%s），忽略 OPSFLEET_API_URL=%s（%s）",
			fileURL, opsfleetEnvLabel(fileURL), envURL, opsfleetEnvLabel(envURL),
		)
		emitAutoBindingWarning()
		return fileURL, nil
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

func emitAutoBindingWarning() {
	if strings.TrimSpace(autoBindingWarn) == "" || autoBindingWarnShown {
		return
	}
	autoBindingWarnShown = true
	_, _ = fmt.Fprintf(os.Stderr, "[%s] %s\n", progName, autoBindingWarn)
}

// collectOpsfleetAPIBaseCandidates returns deduplicated bases to probe for version/upgrade (reachability order).
func collectOpsfleetAPIBaseCandidates() []string {
	var out []string
	add := func(b string) {
		b = normalizeOpsfleetAPIBase(b)
		if b == "" {
			return
		}
		for _, existing := range out {
			if opsfleetAPIBasesEquivalent(existing, b) {
				return
			}
		}
		out = append(out, b)
	}
	add(config.LoadOptionalOpsfleetAPIBase())
	if v := strings.TrimSpace(os.Getenv("OPSFLEET_API_URL")); v != "" {
		add(v)
	}
	add(EmbeddedOpsfleetAPIBase)
	add(EmbeddedOpsfleetAPIBaseProduction)
	return out
}

// opsfleetConsoleHost returns non-loopback host from user-bound OpsFleet API (install file or env only).
// Embedded lab/production defaults are not used for check target inference (avoids surprising remote targets).
func opsfleetConsoleHost() string {
	b := normalizeOpsfleetAPIBase(config.LoadOptionalOpsfleetAPIBase())
	if b == "" {
		b = normalizeOpsfleetAPIBase(os.Getenv("OPSFLEET_API_URL"))
	}
	if b == "" {
		return ""
	}
	u, err := url.Parse(b)
	if err != nil || u.Host == "" {
		return ""
	}
	host := u.Hostname()
	switch strings.ToLower(host) {
	case "", "127.0.0.1", "localhost", "::1":
		return ""
	}
	if net.ParseIP(host) != nil {
		return host
	}
	// hostname: use as-is for DNS names
	return host
}

// resolveOpsfleetAPIBaseForUpgrade returns the first candidate base (reachability resolved in fetchRemoteVersionFast).
func resolveOpsfleetAPIBaseForUpgrade() (base string, warn string) {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("OPSFLEET_SKIP_REMOTE")), "1") {
		return "", ""
	}
	candidates := collectOpsfleetAPIBaseCandidates()
	if len(candidates) == 0 {
		return EmbeddedOpsfleetAPIBase, ""
	}
	if strings.TrimSpace(autoBindingWarn) != "" {
		warn = autoBindingWarn
	}
	return candidates[0], warn
}

// resolveOpsfleetAPIBasesForUpgrade 仅返回当前绑定环境的一个基址（不再串联实验+生产）。
func resolveOpsfleetAPIBasesForUpgrade() []string {
	b, _ := resolveOpsfleetAPIBaseForUpgrade()
	if b == "" {
		return nil
	}
	return []string{b}
}
