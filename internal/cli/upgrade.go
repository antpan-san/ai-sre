package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

func isHelpInvocation() bool {
	for _, a := range os.Args[1:] {
		if a == "-h" || a == "--help" {
			return true
		}
	}
	return false
}

// shouldSkipPreUpgradeCheck 不拦截 doctor/k8s 等业务命令；自升级/版本/帮助等避免递归与无意义网络请求。
func shouldSkipPreUpgradeCheck(cmd *cobra.Command) bool {
	if cmd == nil {
		return true
	}
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		// 注意: uninstall 不在此列，执行 uninstall k8s 前会与别的子命令一样尝试拉 OpsFleet 上较新的 ai-sre 并 re-exec
		case "upgrade", "version", "help", "completion":
			return true
		}
	}
	return false
}

// opsfleetPersistentPreRun 在**每次**子命令前快速探测 OpsFleet 版本；有更新则下载并 re-exec（Linux/macOS）。
// 关闭：OPSFLEET_NO_AUTO_UPGRADE=1 或 --no-auto-upgrade。
func opsfleetPersistentPreRun(cmd *cobra.Command, _ []string) error {
	if len(os.Args) <= 1 {
		return nil
	}
	if isHelpInvocation() {
		return nil
	}
	if shouldSkipPreUpgradeCheck(cmd) {
		return nil
	}
	if os.Getenv("OPSFLEET_NO_AUTO_UPGRADE") == "1" {
		if os.Getenv("OPSFLEET_UPGRADE_HINT") == "1" || os.Getenv("OPSFLEET_UPGRADE_CHECK") == "1" {
			return runUpgradeHintOnly(resolveOpsfleetAPIBase())
		}
		return nil
	}
	if err := tryAutoUpgradeInPlace(""); err != nil {
		ctx := context.Background()
		_ = recoverInstallDownloadFailure(ctx, "auto_upgrade", err, map[string]string{
			"phase": "persistent_pre_run",
		})
		if upgradeCheckVerbose() {
			_, _ = fmt.Fprintf(os.Stderr, "[%s] 自动检查更新: %v（已转服务端 AI 处置）\n", progName, err)
		}
	}
	return nil
}

func upgradeCheckVerbose() bool {
	return os.Getenv("OPSFLEET_AUTO_UPGRADE_VERBOSE") == "1" || os.Getenv("OPSFLEET_UPGRADE_CHECK_VERBOSE") == "1"
}

// tryAutoUpgradeInPlace 有更新时覆盖正在运行的可执行文件并（Unix）exec 同 argv，使本次命令在**新版本**中重新执行一次。
func tryAutoUpgradeInPlace(preferredBase string) error {
	remote, base, err := fetchRemoteVersionFast(preferredBase)
	if err != nil {
		_ = recoverInstallDownloadFailure(context.Background(), "version_check", err, map[string]string{
			"preferred_base": preferredBase,
		})
		return nil
	}
	if base == "" {
		return nil
	}
	if remote == "" || remote == "unknown" {
		return nil
	}
	if !versionIsOlder(Version, remote) {
		return nil
	}
	if os.Getenv("OPSFLEET_AUTO_UPGRADE_ATTEMPT") == "1" {
		loopErr := fmt.Errorf("自动升级后仍为 %s，OpsFleet 仍声明 %s", Version, remote)
		_ = recoverInstallDownloadFailure(context.Background(), "auto_upgrade", loopErr, map[string]string{
			"remote_version": remote,
			"api_base":       base,
		})
		return nil
	}
	_, _ = fmt.Fprintf(os.Stderr, "[%s] OpsFleet 有更新 %s（当前 %s），正在自动升级…\n", progName, remote, Version)
	ctxDown, cancelDown := context.WithTimeout(context.Background(), upgradeDownloadTimeout())
	defer cancelDown()
	arch := goArchToAiSreArch()
	self, err := os.Executable()
	if err != nil {
		_ = recoverInstallDownloadFailure(context.Background(), "auto_upgrade", err, map[string]string{"phase": "resolve_executable"})
		return nil
	}
	self, err = filepath.EvalSymlinks(self)
	if err != nil {
		_ = recoverInstallDownloadFailure(context.Background(), "auto_upgrade", err, map[string]string{"phase": "eval_symlinks"})
		return nil
	}
	if err := downloadAndReplaceAIsre(ctxDown, base, arch, self); err != nil {
		_ = recoverInstallDownloadFailure(context.Background(), "auto_upgrade", err, map[string]string{
			"remote_version": remote,
			"api_base":       base,
			"arch":           arch,
			"dest_path":      self,
		})
		return nil
	}
	installed, err := readInstalledVersion(self)
	if err != nil {
		_ = recoverInstallDownloadFailure(context.Background(), "auto_upgrade", err, map[string]string{"phase": "read_installed_version"})
		return nil
	}
	if versionIsOlder(installed, remote) {
		mismatch := fmt.Errorf("下载后本地版本仍为 %s，服务端声明 %s", installed, remote)
		_ = recoverInstallDownloadFailure(context.Background(), "auto_upgrade", mismatch, map[string]string{
			"remote_version": remote,
			"installed":      installed,
			"api_base":       base,
		})
		return nil
	}
	if runtime.GOOS == "windows" {
		_, _ = fmt.Fprintf(os.Stderr, "[%s] 已写入新版本。请**再次**运行同一命令以使用新版本（Windows 下无法自动重载进程）。\n", progName)
		os.Exit(0)
	}
	env := appendAttemptUpgradeEnv(os.Environ())
	if err := syscall.Exec(self, os.Args, env); err != nil {
		_ = recoverInstallDownloadFailure(context.Background(), "auto_upgrade", err, map[string]string{"phase": "exec"})
		return nil
	}
	return nil
}

func runUpgradeHintOnly(base string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 900*time.Millisecond)
	defer cancel()
	ver, err := fetchRemoteVersion(ctx, base, upgradeHTTPClient)
	if err != nil || ver == "" || ver == "unknown" {
		return nil
	}
	if !versionIsOlder(Version, ver) {
		return nil
	}
	_, _ = fmt.Fprintf(os.Stderr, "[ai-sre] OpsFleet 提供更新版本 %s（当前 %s），执行 %s upgrade 后重试以覆盖安装\n", ver, Version, progName)
	return nil
}

func upgradeCmd() *cobra.Command {
	var (
		apiURL   string
		yes      bool
		check    bool
		arch     string
		verboseU bool
	)
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "连接 OpsFleet 检测 ai-sre 版本并在需要时覆盖本机二进制",
		Long: "向 GET .../api/k8s/deploy/cli/ai-sre/version 拉取元数据，与当前可执行文件比对；\n" +
			"若服务器版本更新，则下载 GET .../api/k8s/deploy/cli/ai-sre?arch=... 并覆盖正在使用的二进制\n" +
			"（同 curl 安装脚本，通常需 root，目标路径为 which ai-sre，一般为 /usr/local/bin/ai-sre）。\n\n" +
			"默认基址为内建 " + EmbeddedOpsfleetAPIBase + "；仅当需联调其它控制台时设置 OPSFLEET_API_URL。\n" +
			"OPSFLEET_NO_AUTO_UPGRADE=1 可关闭自升级，仅当另设 OPSFLEET_UPGRADE_HINT=1 时提示。",
		RunE: func(cmd *cobra.Command, args []string) error {
			base := strings.TrimSpace(apiURL)
			base = strings.TrimRight(base, "/")
			var remote string
			var err error
			if base != "" {
				vctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				remote, err = fetchRemoteVersion(vctx, base, upgradeHTTPClient)
				cancel()
			} else {
				remote, base, err = fetchRemoteVersionFast("")
			}
			if err != nil {
				_ = recoverInstallDownloadFailure(context.Background(), "version_check", err, map[string]string{"api_url": apiURL})
				return nil
			}
			if base == "" {
				_ = recoverInstallDownloadFailure(context.Background(), "version_check",
					fmt.Errorf("未解析到 OpsFleet API 基址"), nil)
				return nil
			}
			if remote == "" || remote == "unknown" {
				if check {
					os.Exit(2)
				}
				return fmt.Errorf("服务端未返回有效版本，请检查 OpsFleet 是否配置 opsfleet.ai_sre_binary_path 与 OPSFLEET_AISRE_VERSION（可选）")
			}
			if !versionIsOlder(Version, remote) {
				if versionIsOlder(remote, Version) {
					msg := fmt.Sprintf("本地 %s 已高于 OpsFleet 当前分发版 %s，不会从服务端降级", Version, remote)
					if verboseU {
						_, _ = fmt.Fprintf(os.Stdout, "%s（GET .../cli/ai-sre/version 读的是控制台 bin/ai-sre）\n", msg)
					} else {
						fmt.Println(msg)
					}
				} else if verboseU {
					_, _ = fmt.Fprintf(os.Stdout, "当前已是最新：本地 %s，服务端 %s\n", Version, remote)
				} else {
					fmt.Println("已是最新，无需升级（本地", Version+"，OpsFleet", remote+"）")
				}
				if check {
					os.Exit(0)
				}
				return nil
			}
			if check {
				_, _ = fmt.Fprintf(os.Stdout, "有可用更新: %s -> %s\n", Version, remote)
				os.Exit(1)
			}
			if !yes {
				st, _ := os.Stdin.Stat()
				if (st.Mode() & os.ModeCharDevice) == 0 {
					return fmt.Errorf("非交互式环境请使用 -y 确认升级")
				}
				ex, _ := os.Executable()
				_, _ = fmt.Fprintf(os.Stdout, "将从 OpsFleet 下载并覆盖本机二进制（%s -> %s）\n目标: %s\n继续? 输入 y 回车: ", Version, remote, ex)
				_ = os.Stdout.Sync()
				var line string
				_, _ = fmt.Fscanln(os.Stdin, &line)
				if line != "y" && line != "Y" {
					return fmt.Errorf("已取消升级")
				}
			}
			ua := arch
			if ua == "" {
				ua = goArchToAiSreArch()
			}
			self, err := os.Executable()
			if err != nil {
				return err
			}
			self, err = filepath.EvalSymlinks(self)
			if err != nil {
				return err
			}
			dlCtx, dlCancel := context.WithTimeout(context.Background(), upgradeDownloadTimeout())
			defer dlCancel()
			if err := downloadAndReplaceAIsre(dlCtx, base, ua, self); err != nil {
				_ = recoverInstallDownloadFailure(cmd.Context(), "upgrade", err, map[string]string{
					"remote_version": remote,
					"api_base":       base,
					"arch":           ua,
					"dest_path":      self,
				})
				return nil
			}
			_, _ = fmt.Fprintf(os.Stdout, "升级完成。请执行: %s version（当前应显示 %s）\n", progName, remote)
			return nil
		},
	}
	cmd.Flags().StringVar(&apiURL, "api-url", "", "覆盖内建 OpsFort 基址（默认 "+EmbeddedOpsfleetAPIBase+"）")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "不询问，直接覆盖（非 TTY 时必填）")
	cmd.Flags().BoolVar(&check, "check", false, "仅检查：有更新时退出 1，已最新退出 0，错误退出 2")
	cmd.Flags().StringVar(&arch, "arch", "", "目标 arch：amd64|arm64（默认本机 uname 推断，Linux 常用）")
	cmd.Flags().BoolVar(&verboseU, "show-versions", false, "打印详细版本信息")
	return cmd
}

var upgradeHTTPClient = &http.Client{
	Timeout: 1200 * time.Millisecond,
	Transport: &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        4,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: 800 * time.Millisecond,
	},
}

func upgradeDownloadTimeout() time.Duration {
	if v := strings.TrimSpace(os.Getenv("OPSFLEET_UPGRADE_DOWNLOAD_TIMEOUT")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return 10 * time.Minute
}

func upgradeDownloadHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}
}

// fetchRemoteVersionFast 依次探测多个 OpsFleet 基址（单址约 1.2s 超时），返回首个可用版本与基址。
func fetchRemoteVersionFast(preferredBase string) (version string, base string, err error) {
	bases := resolveOpsfleetAPIBasesForUpgrade()
	if preferredBase != "" {
		preferredBase = strings.TrimRight(strings.TrimSpace(preferredBase), "/")
		merged := []string{preferredBase}
		for _, b := range bases {
			if b != preferredBase {
				merged = append(merged, b)
			}
		}
		bases = merged
	}
	if len(bases) == 0 {
		return "", "", fmt.Errorf("未配置 OpsFleet API 基址")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3500*time.Millisecond)
	defer cancel()
	var lastErr error
	for _, b := range bases {
		v, e := fetchRemoteVersion(ctx, b, upgradeHTTPClient)
		if e == nil && v != "" && v != "unknown" {
			return v, b, nil
		}
		lastErr = e
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("所有基址均未返回有效版本")
	}
	return "", "", lastErr
}

// fetchRemoteVersion returns JSON .version from OpsFleet.
func fetchRemoteVersion(ctx context.Context, apiBase string, client *http.Client) (string, error) {
	if client == nil {
		client = http.DefaultClient
	}
	u, err := url.JoinPath(apiBase, "api", "k8s", "deploy", "cli", "ai-sre", "version")
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", progName+"/"+Version)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET version HTTP %d: %s", resp.StatusCode, truncateForErr(body, 512))
	}
	var meta struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(body, &meta); err != nil {
		return "", fmt.Errorf("解析版本 JSON: %w", err)
	}
	return strings.TrimSpace(meta.Version), nil
}

// versionIsOlder 判断 a 是否严格小于 b（x.y.z 数字段，忽略 v 前缀和预发布）。
func versionIsOlder(a, b string) bool {
	pa := versionParts(normalizeVersionString(a))
	pb := versionParts(normalizeVersionString(b))
	if len(pb) == 0 {
		return false
	}
	if len(pa) == 0 {
		return true
	}
	for i := 0; i < maxLen(len(pa), len(pb)); i++ {
		var na, nb int
		if i < len(pa) {
			na = pa[i]
		}
		if i < len(pb) {
			nb = pb[i]
		}
		if na < nb {
			return true
		}
		if na > nb {
			return false
		}
	}
	return false
}

func maxLen(i, j int) int {
	if i > j {
		return i
	}
	return j
}

func normalizeVersionString(s string) string {
	s = strings.TrimSpace(strings.TrimPrefix(s, "v"))
	return s
}

func versionParts(s string) []int {
	if s == "" {
		return nil
	}
	var out []int
	for _, p := range strings.Split(s, ".") {
		p = strings.TrimSpace(p)
		// 截断 1.2-beta -> 1,2
		for i := 0; i < len(p); i++ {
			if p[i] < '0' || p[i] > '9' {
				p = p[:i]
				break
			}
		}
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func goArchToAiSreArch() string {
	g := os.Getenv("GOARCH")
	if g == "arm64" {
		return "arm64"
	}
	if g == "386" {
		return "amd64" // 近似，极少用于分发
	}
	if g == "arm" {
		return "arm64"
	}
	// 常见 Linux amd64
	if g == "amd64" {
		return "amd64"
	}
	// 运行 env GOARCH
	out, _ := exec.Command("uname", "-m").Output()
	m := strings.TrimSpace(string(out))
	switch m {
	case "x86_64", "amd64":
		return "amd64"
	case "aarch64", "arm64":
		return "arm64"
	default:
		return "amd64"
	}
}

func appendAttemptUpgradeEnv(env []string) []string {
	const key = "OPSFLEET_AUTO_UPGRADE_ATTEMPT="
	out := make([]string, 0, len(env)+1)
	for _, e := range env {
		if strings.HasPrefix(e, key) {
			continue
		}
		out = append(out, e)
	}
	return append(out, key+"1")
}

func readInstalledVersion(bin string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "version").Output()
	if err != nil {
		return "", fmt.Errorf("读取升级后版本: %w", err)
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) < 2 {
		return "", fmt.Errorf("解析升级后版本输出: %q", strings.TrimSpace(string(out)))
	}
	return fields[1], nil
}

func downloadAndReplaceAIsre(ctx context.Context, apiBase, arch, destPath string) error {
	uStr, err := url.JoinPath(apiBase, "api", "k8s", "deploy", "cli", "ai-sre")
	if err != nil {
		return err
	}
	full, err := url.Parse(uStr)
	if err != nil {
		return err
	}
	q := full.Query()
	q.Set("arch", arch)
	full.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, full.String(), nil)
	if err != nil {
		return err
	}
	resp, err := upgradeDownloadHTTPClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("下载失败 HTTP %d: %s", resp.StatusCode, truncateForErr(b, 1024))
	}
	expected := resp.ContentLength
	destPath = strings.TrimSpace(destPath)
	if destPath == "" {
		return fmt.Errorf("目标路径为空")
	}
	tmp, err := os.CreateTemp(filepath.Dir(destPath), ".ai-sre-upgrading-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	pr := newProgressReader(resp.Body, resp.ContentLength, fmt.Sprintf("下载 ai-sre 二进制 (arch=%s)", arch))
	n, err := io.Copy(tmp, pr)
	_ = pr.Close()
	if err != nil {
		tmp.Close()
		return err
	}
	if expected > 0 && n < expected {
		tmp.Close()
		return fmt.Errorf("下载不完整: %d/%d 字节（网络过慢或超时，可增大 OPSFLEET_UPGRADE_DOWNLOAD_TIMEOUT 或重试）", n, expected)
	}
	_ = pr.Close()
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return err
	}
	// 原子覆盖正在运行的可执行文件（Linux 上允许；下次 exec 用新内容）
	if err := os.Rename(tmpPath, destPath); err != nil {
		// 若同目录重命名因跨设备失败，可尝试 cp
		_ = os.Remove(tmpPath)
		return fmt.Errorf("无法覆盖 %s: %w（请用 root 或写权限）", destPath, err)
	}
	return nil
}
