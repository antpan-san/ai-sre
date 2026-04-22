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
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// quickUpgradeHintPreRun 在其它子命令执行前，可选做一次远端版本检测（需环境变量，默认不联网）。
func quickUpgradeHintPreRun(cmd *cobra.Command, _ []string) error {
	if cmd == nil {
		return nil
	}
	// 避免与 upgrade 自身、纯查询类命令链式噪音
	switch cmd.Name() {
	case "upgrade", "version", "doctor", "help", "completion":
		return nil
	}
	if os.Getenv("OPSFLEET_UPGRADE_HINT") != "1" && os.Getenv("OPSFLEET_UPGRADE_CHECK") != "1" {
		return nil
	}
	base := strings.TrimSpace(os.Getenv("OPSFLEET_API_URL"))
	if base == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 900*time.Millisecond)
	defer cancel()
	ver, err := fetchRemoteVersion(ctx, strings.TrimRight(base, "/"))
	if err != nil || ver == "" || ver == "unknown" {
		return nil
	}
	if !versionIsOlder(Version, ver) {
		return nil
	}
	_, _ = fmt.Fprintf(os.Stderr, "[ai-sre] OpsFleet 提供更新版本 %s（当前 %s），执行 %s upgrade --api-url %q 可覆盖安装\n", ver, Version, progName, base)
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
		Long: `向 GET .../api/k8s/deploy/cli/ai-sre/version 拉取元数据，与当前可执行文件比对；
若服务器版本更新，则下载 GET .../api/k8s/deploy/cli/ai-sre?arch=... 并覆盖正在使用的二进制
（同 curl 安装脚本，通常需 root，目标路径为 which ai-sre，一般为 /usr/local/bin/ai-sre）。

环境变量: OPSFLEET_API_URL（如 http://host:9080/ft-api）。可选: OPSFLEET_UPGRADE_HINT=1
与其它子命令一起使用时，在「执行前」若设置了 OPSFLEET_API_URL + OPSFLEET_UPGRADE_HINT，会快速提示是否有新版本。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			base := strings.TrimSpace(apiURL)
			if base == "" {
				base = strings.TrimSpace(os.Getenv("OPSFLEET_API_URL"))
			}
			if base == "" {
				return fmt.Errorf("请传 --api-url 或设置环境变量 OPSFLEET_API_URL（例如 %s/ft-api 或 http://IP:9080/ft-api）", "http://host:9080")
			}
			base = strings.TrimRight(base, "/")
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			remote, err := fetchRemoteVersion(ctx, base)
			if err != nil {
				return err
			}
			if remote == "" || remote == "unknown" {
				if check {
					os.Exit(2)
				}
				return fmt.Errorf("服务端未返回有效版本，请检查 OpsFleet 是否配置 opsfleet.ai_sre_binary_path 与 OPSFLEET_AISRE_VERSION（可选）")
			}
			if !versionIsOlder(Version, remote) {
				if verboseU {
					_, _ = fmt.Fprintf(os.Stdout, "当前已是最新或较新：本地 %s，服务端 %s\n", Version, remote)
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
			if err := downloadAndReplaceAIsre(ctx, base, ua); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(os.Stdout, "升级完成。请执行: %s version（当前应显示 %s）\n", progName, remote)
			return nil
		},
	}
	cmd.Flags().StringVar(&apiURL, "api-url", "", "OpsFleet API 基址（同 k8s download；也可 OPSFLEET_API_URL）")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "不询问，直接覆盖（非 TTY 时必填）")
	cmd.Flags().BoolVar(&check, "check", false, "仅检查：有更新时退出 1，已最新退出 0，错误退出 2")
	cmd.Flags().StringVar(&arch, "arch", "", "目标 arch：amd64|arm64（默认本机 uname 推断，Linux 常用）")
	cmd.Flags().BoolVar(&verboseU, "show-versions", false, "打印详细版本信息")
	return cmd
}

// fetchRemoteVersion returns JSON .version from OpsFleet.
func fetchRemoteVersion(ctx context.Context, apiBase string) (string, error) {
	u, err := url.JoinPath(apiBase, "api", "k8s", "deploy", "cli", "ai-sre", "version")
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
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

func downloadAndReplaceAIsre(ctx context.Context, apiBase, arch string) error {
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
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("下载失败 HTTP %d: %s", resp.StatusCode, truncateForErr(b, 1024))
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	self, err = filepath.EvalSymlinks(self)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(self), ".ai-sre-upgrading-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return err
	}
	// 原子覆盖正在运行的可执行文件（Linux 上允许；下次 exec 用新内容）
	if err := os.Rename(tmpPath, self); err != nil {
		// 若同目录重命名因跨设备失败，可尝试 cp
		_ = os.Remove(tmpPath)
		return fmt.Errorf("无法覆盖 %s: %w（请用 root 或写权限）", self, err)
	}
	return nil
}
