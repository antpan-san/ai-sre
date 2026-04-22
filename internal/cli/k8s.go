package cli

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// installRefPrefixV1 须与 ft-backend/handlers/k8s_bundle_invite.go 保持一致。
const installRefPrefixV1 = "ofpk8s1."

type installRefWire struct {
	B string `json:"b"`
	I string `json:"i"`
	T string `json:"t"`
}

// k8sDeployAPIBody mirrors ft-backend/handlers.K8sDeployRequest JSON for POST /api/k8s/deploy/bundle.
type k8sDeployAPIBody struct {
	ClusterName             string   `json:"clusterName"`
	Version                 string   `json:"version"`
	DeployMode              string   `json:"deployMode,omitempty"`
	ImageSource             string   `json:"imageSource,omitempty"`
	ArchVersion             string   `json:"archVersion,omitempty"`
	CustomRegistry          string   `json:"customRegistry,omitempty"`
	RegistryUsername        string   `json:"registryUsername,omitempty"`
	RegistryPassword        string   `json:"registryPassword,omitempty"`
	MasterHosts             []string `json:"masterHosts"`
	WorkerHosts             []string `json:"workerHosts"`
	KubeProxyMode           string   `json:"kubeProxyMode,omitempty"`
	EnableRBAC              bool     `json:"enableRBAC"`
	EnablePodSecurityPolicy bool     `json:"enablePodSecurityPolicy"`
	EnableAudit             bool     `json:"enableAudit"`
	PauseImage              string   `json:"pauseImage,omitempty"`
	NetworkPlugin           string   `json:"networkPlugin,omitempty"`
	PodCIDR                 string   `json:"podCidr,omitempty"`
	ServiceCIDR             string   `json:"serviceCidr,omitempty"`
	DNSServiceIP            string   `json:"dnsServiceIP,omitempty"`
	ClusterDomain           string   `json:"clusterDomain,omitempty"`
	DefaultStorageClass     bool     `json:"defaultStorageClass"`
	StorageProvisioner      string   `json:"storageProvisioner,omitempty"`
	EnableNodeLocalDNS      bool     `json:"enableNodeLocalDNS"`
	EnableMetricsServer     bool     `json:"enableMetricsServer"`
	EnableDashboard         bool     `json:"enableDashboard"`
	EnablePrometheus        bool     `json:"enablePrometheus"`
	EnableIngressNginx      bool     `json:"enableIngressNginx"`
	EnableHelm              bool     `json:"enableHelm"`
	PreDeployCleanup        bool     `json:"preDeployCleanup"`
	DownloadDomain          string   `json:"downloadDomain,omitempty"`
	DownloadProtocol        string   `json:"downloadProtocol,omitempty"`
}

func k8sCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "k8s",
		Short: "Kubernetes 离线包：对接 OpsFleet API 下载、本地安装与卸载清理",
		Long: fmt.Sprintf(`推荐：在控制台「部署确认」生成一键安装引用后，在控制机执行一行命令即可拉包并安装（无需上传 zip）。

未安装 %s 时可用公开引导脚本（需 python3）：
  curl -fsSL 'http://<控制台>/ft-api/api/k8s/deploy/bootstrap.sh' | sudo bash -s -- 'ofpk8s1.xxxxx…'

示例:
  # 一键安装（installRef 由页面复制，整段单引号包裹）
  sudo %s k8s install 'ofpk8s1.xxxxx…'

  # 仍支持本地下载 zip 后安装
  %s k8s download --api-url http://192.168.56.11:9080/ft-api -u admin -p '***' \\
    --cluster lab --version v1.35.4 --master 10.10.120.142 -O ./bundle.zip
  sudo %s k8s install --package ./bundle.zip

  sudo %s k8s install --workdir /opt/opsfleet-k8s

  # 部署失败或需按页面节点全量清理（与 install 使用同一 ofpk8s1 引用；重新拉包并跑 pre_cleanup）
  sudo %s k8s cleanup 'ofpk8s1.…'

  # 一步卸载（优先本机 /var/lib/opsfleet-k8s/last-bundle，无需控制台 id）
  sudo %s uninstall k8s

  sudo %s k8s uninstall --workdir /opt/opsfleet-k8s`, progName, progName, progName, progName, progName, progName, progName, progName),
	}
	cmd.AddCommand(k8sDownloadCmd(), k8sInstallCmd(), k8sCleanupCmd(), k8sUninstallCmd())
	return cmd
}

func k8sDownloadCmd() *cobra.Command {
	var (
		apiURL           string
		username         string
		password         string
		token            string
		outPath          string
		requestJSON      string
		cluster          string
		version          string
		masterCSV        string
		workerCSV        string
		deployMode       string
		arch             string
		imageSource      string
		downloadDomain   string
		downloadProtocol string
		preCleanup       bool
		podCIDR          string
		serviceCIDR      string
		dnsServiceIP     string
		clusterDomain    string
		networkPlugin    string
		kubeProxyMode    string
	)
	cmd := &cobra.Command{
		Use:   "download",
		Short: "登录 OpsFleet 并下载 K8s 离线 zip（与页面「生成离线包」相同接口）",
		RunE: func(cmd *cobra.Command, args []string) error {
			base := strings.TrimSpace(apiURL)
			if base == "" {
				base = strings.TrimSpace(os.Getenv("OPSFLEET_API_URL"))
			}
			if base == "" {
				base = resolveOpsfleetAPIBase()
			}
			base = strings.TrimRight(base, "/")

			tok := strings.TrimSpace(token)
			if tok == "" {
				tok = strings.TrimSpace(os.Getenv("OPSFLEET_TOKEN"))
			}
			if tok == "" {
				u := strings.TrimSpace(username)
				if u == "" {
					u = os.Getenv("OPSFLEET_USERNAME")
				}
				p := strings.TrimSpace(password)
				if p == "" {
					p = os.Getenv("OPSFLEET_PASSWORD")
				}
				if u == "" || p == "" {
					return errors.New("请提供 --token，或同时提供 --username/--password（或 OPSFLEET_TOKEN / OPSFLEET_USERNAME+OPSFLEET_PASSWORD）")
				}
				var err error
				tok, err = opsfleetLogin(base, u, p)
				if err != nil {
					return err
				}
			}

			var body []byte
			var err error
			if strings.TrimSpace(requestJSON) != "" {
				body, err = os.ReadFile(requestJSON)
				if err != nil {
					return fmt.Errorf("读取 --request-json: %w", err)
				}
			} else {
				if strings.TrimSpace(cluster) == "" || strings.TrimSpace(version) == "" {
					return errors.New("未使用 --request-json 时，--cluster 与 --version 必填")
				}
				masters := splitCSV(masterCSV)
				if len(masters) == 0 {
					return errors.New("至少指定一个 --master（control plane IP，逗号分隔）")
				}
				req := k8sDeployAPIBody{
					ClusterName:         cluster,
					Version:             version,
					DeployMode:          deployMode,
					ImageSource:         imageSource,
					ArchVersion:         arch,
					MasterHosts:         masters,
					WorkerHosts:         splitCSV(workerCSV),
					KubeProxyMode:       kubeProxyMode,
					EnableRBAC:          true,
					DefaultStorageClass: true,
					PreDeployCleanup:    preCleanup,
					DownloadDomain:      strings.TrimSpace(downloadDomain),
					DownloadProtocol:    strings.TrimSpace(downloadProtocol),
					PodCIDR:             podCIDR,
					ServiceCIDR:         serviceCIDR,
					DNSServiceIP:        dnsServiceIP,
					ClusterDomain:       clusterDomain,
					NetworkPlugin:       networkPlugin,
				}
				if req.DeployMode == "" {
					req.DeployMode = "single"
				}
				if req.ImageSource == "" {
					req.ImageSource = "default"
				}
				if req.ArchVersion == "" {
					req.ArchVersion = "amd64"
				}
				body, err = json.Marshal(req)
				if err != nil {
					return err
				}
			}

			dest := outPath
			if dest == "" {
				dest = "opsfleet-k8s-bundle.zip"
			}
			if err := downloadK8sBundle(base, tok, body, dest); err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "wrote %s\n", dest)
			return nil
		},
	}
	cmd.Flags().StringVar(&apiURL, "api-url", "", "OpsFleet API 基址（含路径前缀，如 http://IP:9080/ft-api）")
	cmd.Flags().StringVarP(&username, "username", "u", "", "登录用户名")
	cmd.Flags().StringVarP(&password, "password", "p", "", "登录密码")
	cmd.Flags().StringVar(&token, "token", "", "跳过登录，直接使用 JWT")
	cmd.Flags().StringVarP(&outPath, "out", "O", "", "保存 zip 的路径（默认 opsfleet-k8s-bundle.zip；勿与全局 -o text|json 混淆）")
	cmd.Flags().StringVar(&requestJSON, "request-json", "", "从文件读取完整 JSON 请求体（与控制台提交体一致；指定后忽略其它表单类 flag）")
	cmd.Flags().StringVar(&cluster, "cluster", "", "集群名称 clusterName")
	cmd.Flags().StringVar(&version, "version", "", "Kubernetes 版本，如 v1.35.4")
	cmd.Flags().StringVar(&masterCSV, "master", "", "control plane 节点 IP，逗号分隔（masterHosts）")
	cmd.Flags().StringVar(&workerCSV, "worker", "", "worker 节点 IP，逗号分隔（可选）")
	cmd.Flags().StringVar(&deployMode, "deploy-mode", "single", "deployMode: single | ha")
	cmd.Flags().StringVar(&arch, "arch", "amd64", "节点 CPU 架构 archVersion: amd64 | arm64")
	cmd.Flags().StringVar(&imageSource, "image-source", "default", "镜像源 imageSource: default | aliyun | tencent | custom")
	cmd.Flags().StringVar(&downloadDomain, "download-domain", "", "覆盖内网制品域名 downloadDomain")
	cmd.Flags().StringVar(&downloadProtocol, "download-protocol", "", "覆盖下载协议前缀，如 http://")
	cmd.Flags().BoolVar(&preCleanup, "pre-cleanup", false, "打包 install.sh 默认先执行 pre_cleanup（preDeployCleanup）")
	cmd.Flags().StringVar(&podCIDR, "pod-cidr", "", "podCidr（默认由服务端写 10.244.0.0/16）")
	cmd.Flags().StringVar(&serviceCIDR, "service-cidr", "", "serviceCidr")
	cmd.Flags().StringVar(&dnsServiceIP, "dns-service-ip", "", "dnsServiceIP")
	cmd.Flags().StringVar(&clusterDomain, "cluster-domain", "", "clusterDomain")
	cmd.Flags().StringVar(&networkPlugin, "network-plugin", "", "networkPlugin（默认 flannel）")
	cmd.Flags().StringVar(&kubeProxyMode, "kube-proxy-mode", "", "kubeProxyMode（默认 iptables）")
	return cmd
}

func k8sInstallCmd() *cobra.Command {
	var (
		pkgPath string
		workdir string
	)
	cmd := &cobra.Command{
		Use:   "install [install-ref]",
		Short: "一键安装：控制台生成的 installRef；或解压 zip / 已解压目录执行 install.sh",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				if pkgPath != "" || workdir != "" {
					return errors.New("已传入安装引用时，不要同时使用 --package 或 --workdir")
				}
				arg := strings.Trim(strings.TrimSpace(args[0]), `"'`)
				if strings.HasPrefix(arg, installRefPrefixV1) {
					return runInstallFromInviteRef(arg)
				}
				return runInstallFromBareInviteID(arg)
			}
			if pkgPath == "" && workdir == "" {
				return errors.New("请传入安装引用：sudo " + progName + ` k8s install 'ofpk8s1.…'（由控制台生成），或使用 --package / --workdir`)
			}
			if pkgPath != "" && workdir != "" {
				return errors.New("--package 与 --workdir 二选一")
			}
			dir := workdir
			if pkgPath != "" {
				var err error
				dir, err = os.MkdirTemp("", "opsfleet-k8s-install-*")
				if err != nil {
					return err
				}
				defer os.RemoveAll(dir)
				if err := unzipFile(pkgPath, dir); err != nil {
					return err
				}
				if verbose {
					fmt.Fprintf(os.Stderr, "[%s] extracted to %s\n", progName, dir)
				}
			}
			return runInstallSh(dir)
		},
	}
	cmd.Flags().StringVar(&pkgPath, "package", "", "离线 zip 路径（将解压到临时目录后执行 install.sh）")
	cmd.Flags().StringVar(&workdir, "workdir", "", "已解压的离线包根目录（含 install.sh、inventory/、ansible-agent/）")
	return cmd
}

func runInstallSh(dir string) error {
	installSh := filepath.Join(dir, "install.sh")
	if st, err := os.Stat(installSh); err != nil || st.IsDir() {
		return fmt.Errorf("在 %s 未找到 install.sh（请确认目录为离线包根路径）", dir)
	}
	var c *exec.Cmd
	if os.Geteuid() == 0 {
		c = exec.Command("bash", installSh)
	} else {
		c = exec.Command("sudo", "bash", installSh)
	}
	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}

func runInstallFromInviteRef(ref string) error {
	wire, err := decodeInstallRefV1(ref)
	if err != nil {
		return err
	}
	return downloadInviteZipAndRunInstall(wire.B, wire.I, wire.T, strings.TrimSpace(ref))
}

func decodeInstallRefV1(ref string) (installRefWire, error) {
	var z installRefWire
	if !strings.HasPrefix(ref, installRefPrefixV1) {
		return z, fmt.Errorf("安装引用须以 %s 开头（请在控制台点击「生成一键安装命令」）", installRefPrefixV1)
	}
	raw := ref[len(installRefPrefixV1):]
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return z, fmt.Errorf("解码安装引用失败: %w", err)
	}
	if err := json.Unmarshal(b, &z); err != nil {
		return z, fmt.Errorf("解析安装引用失败: %w", err)
	}
	if strings.TrimSpace(z.B) == "" || strings.TrimSpace(z.I) == "" || strings.TrimSpace(z.T) == "" {
		return z, errors.New("安装引用内容不完整")
	}
	return z, nil
}

func runInstallFromBareInviteID(idStr string) error {
	if _, err := uuid.Parse(strings.TrimSpace(idStr)); err != nil {
		return fmt.Errorf("无效的安装引用或资源 ID: %w", err)
	}
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("OPSFLEET_API_URL")), "/")
	if base == "" {
		base = resolveOpsfleetAPIBase()
	}
	tok := strings.TrimSpace(os.Getenv("OPSFLEET_BUNDLE_TOKEN"))
	if tok == "" {
		return errors.New("仅传入资源 UUID 时需设置 OPSFLEET_BUNDLE_TOKEN；请优先使用控制台生成的整段 installRef（ofpk8s1.…）")
	}
	return downloadInviteZipAndRunInstall(base, strings.TrimSpace(idStr), tok, "")
}

// downloadInviteZipFile 将邀请 zip 写入 destPath（完整文件路径）。
func downloadInviteZipFile(apiBase, inviteID, token, destPath string) error {
	base := strings.TrimRight(strings.TrimSpace(apiBase), "/")
	endpoint, err := url.JoinPath(base, "api", "k8s", "deploy", "bundle-invite", inviteID, "zip")
	if err != nil {
		return err
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()
	if verbose {
		fmt.Fprintf(os.Stderr, "[%s] GET %s/api/k8s/deploy/bundle-invite/%s/zip\n", progName, base, inviteID)
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 15 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return fmt.Errorf("拉取离线包 HTTP %d: %s", resp.StatusCode, truncateForErr(b, 2048))
	}
	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "application/json") {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return fmt.Errorf("服务器返回错误: %s", truncateForErr(b, 2048))
	}
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// bundleInstallRoot 在解压目录中定位含 install.sh 的包根（兼容 zip 单层子目录）。
func bundleInstallRoot(extractDir string) (string, error) {
	installSh := filepath.Join(extractDir, "install.sh")
	if st, err := os.Stat(installSh); err == nil && !st.IsDir() {
		return extractDir, nil
	}
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return "", err
	}
	var subdirs []string
	for _, e := range entries {
		if e.IsDir() {
			subdirs = append(subdirs, filepath.Join(extractDir, e.Name()))
		}
	}
	if len(subdirs) == 1 {
		cand := filepath.Join(subdirs[0], "install.sh")
		if st, err := os.Stat(cand); err == nil && !st.IsDir() {
			return subdirs[0], nil
		}
	}
	return "", fmt.Errorf("离线包内未找到 install.sh")
}

// downloadInviteZipAndExtract 下载邀请 zip 到临时目录并解压，返回包根路径与删除整棵临时树的函数。
func downloadInviteZipAndExtract(apiBase, inviteID, token string) (bundleRoot string, cleanup func(), err error) {
	wrapper, err := os.MkdirTemp("", "opsfleet-k8s-invite-*")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() { _ = os.RemoveAll(wrapper) }
	fail := func(e error) (string, func(), error) {
		cleanup()
		return "", nil, e
	}

	zipPath := filepath.Join(wrapper, "bundle.zip")
	if err := downloadInviteZipFile(apiBase, inviteID, token, zipPath); err != nil {
		return fail(err)
	}
	out := filepath.Join(wrapper, "out")
	if err := os.MkdirAll(out, 0755); err != nil {
		return fail(err)
	}
	if err := unzipFile(zipPath, out); err != nil {
		return fail(err)
	}
	_ = os.Remove(zipPath)

	root, err := bundleInstallRoot(out)
	if err != nil {
		return fail(err)
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "[%s] invite bundle root %s\n", progName, root)
	}
	return root, cleanup, nil
}

func downloadInviteZipAndRunInstall(apiBase, inviteID, token, refHint string) error {
	if strings.TrimSpace(refHint) != "" {
		saveK8sInstallRef(refHint)
	}
	root, cleanup, err := downloadInviteZipAndExtract(apiBase, inviteID, token)
	if err != nil {
		return err
	}
	defer cleanup()

	err = runInstallSh(root)
	if err != nil && strings.TrimSpace(refHint) != "" {
		fmt.Fprintf(os.Stderr, "\n[%s] 安装未成功。可在同一台控制机上（须已对 inventory 中各节点 root 免密）任选其一清理：\n  sudo %s uninstall k8s\n  sudo %s k8s cleanup '%s'\n", progName, progName, progName, strings.TrimSpace(refHint))
		fmt.Fprintf(os.Stderr, "  可选：export OPSFLEET_K8S_AUTO_CLEANUP_ON_FAIL=1 后重试安装，失败时将自动对包内节点执行 pre_cleanup。\n")
	}
	if err != nil {
		v := strings.TrimSpace(os.Getenv("OPSFLEET_K8S_AUTO_CLEANUP_ON_FAIL"))
		if v == "1" || strings.EqualFold(v, "true") {
			agent := filepath.Join(root, "ansible-agent")
			inv := filepath.Join(root, "inventory", "hosts.ini")
			if e2 := runCleanupPlaybook(agent, inv); e2 != nil {
				fmt.Fprintf(os.Stderr, "[%s] 自动清理失败: %v\n", progName, e2)
			} else {
				fmt.Fprintf(os.Stderr, "[%s] 已根据包内 inventory 对各节点执行 pre_cleanup。\n", progName)
			}
		}
	}
	return err
}

func runCleanupFromInviteRef(ref string) error {
	wire, err := decodeInstallRefV1(ref)
	if err != nil {
		return err
	}
	root, cleanup, err := downloadInviteZipAndExtract(wire.B, wire.I, wire.T)
	if err != nil {
		return fmt.Errorf("重新拉取离线包失败（引用可能过期或网络异常）: %w", err)
	}
	defer cleanup()
	return runCleanupPlaybook(filepath.Join(root, "ansible-agent"), filepath.Join(root, "inventory", "hosts.ini"))
}

var errNoLocalK8sBundle = errors.New("本机未找到含 ansible-agent 与 inventory/hosts.ini 的离线包根目录")

// localK8sBundleDirCandidates 尝试的目录顺序：上次 install 写入的快照、环境变量、常见解压路径。
func localK8sBundleDirCandidates() []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(p string) {
		p = filepath.Clean(strings.TrimSpace(p))
		if p == "" {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	add(K8sLastBundlePath)
	add(os.Getenv("OPSFLEET_K8S_WORKDIR"))
	add(K8sDefaultUninstallWorkdir)
	return out
}

func isLocalK8sBundleRoot(dir string) bool {
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return false
	}
	pb := filepath.Join(dir, "ansible-agent", "playbooks", "pre_cleanup.yml")
	inv := filepath.Join(dir, "inventory", "hosts.ini")
	_, e1 := os.Stat(pb)
	_, e2 := os.Stat(inv)
	return e1 == nil && e2 == nil
}

// tryRunCleanupFromLocalBundleDirs 在候选目录中查找已解压的离线包并执行 pre_cleanup。
func tryRunCleanupFromLocalBundleDirs(userHint string) error {
	for _, d := range localK8sBundleDirCandidates() {
		if !isLocalK8sBundleRoot(d) {
			continue
		}
		if strings.TrimSpace(userHint) != "" {
			fmt.Fprintln(os.Stderr, userHint)
		}
		fmt.Fprintf(os.Stderr, "[%s] 使用本机离线包目录: %s\n", progName, d)
		return runCleanupPlaybook(filepath.Join(d, "ansible-agent"), filepath.Join(d, "inventory", "hosts.ini"))
	}
	return errNoLocalK8sBundle
}

// runUninstallK8s 实现 `ai-sre uninstall k8s`：
// 1) 优先本机 last-bundle/常见路径（不依赖平台 id、不联网）；
// 2) 若无副本且未 --force，再尝试 ofpk8s1 拉 zip（旧版兼容）；
// 3) --force 仅本机，失败则报错。
func runUninstallK8s(refOverride, workdir string, forceLocal bool) error {
	workdir = strings.TrimSpace(workdir)
	if workdir != "" {
		root, err := filepath.Abs(workdir)
		if err != nil {
			return err
		}
		if !isLocalK8sBundleRoot(root) {
			return fmt.Errorf("路径不是有效离线包根目录（需含 ansible-agent/playbooks/pre_cleanup.yml 与 inventory/hosts.ini）: %s", root)
		}
		fmt.Fprintf(os.Stderr, "[%s] 使用 --workdir: %s\n", progName, root)
		return runCleanupPlaybook(filepath.Join(root, "ansible-agent"), filepath.Join(root, "inventory", "hosts.ini"))
	}
	ref := strings.TrimSpace(refOverride)
	if ref == "" {
		ref = loadK8sInstallRef()
	}

	// 默认：先用语义 install.sh 同步的快照，无需控制台、无需联网
	if err := tryRunCleanupFromLocalBundleDirs(""); err == nil {
		return nil
	}
	if forceLocal {
		return fmt.Errorf(
			"未找到本机离线包副本。请使用带「同步至 %s」的新版 install.sh 执行过一次安装，或: sudo %s uninstall k8s --workdir <解压根>\n%w",
			K8sLastBundlePath, progName, errNoLocalK8sBundle,
		)
	}
	if ref != "" {
		if !strings.HasPrefix(ref, installRefPrefixV1) {
			return fmt.Errorf("安装引用须以 %s 开头，或改用 --workdir / --force", installRefPrefixV1)
		}
		if err := runCleanupFromInviteRef(ref); err != nil {
			return fmt.Errorf(
				"拉取邀请包失败且本机无 %s 等有效副本: %w\n请用新版离线包执行 install.sh 生成快照，或: sudo %s uninstall k8s --workdir <解压根目录>",
				K8sLastBundlePath, err, progName,
			)
		}
		return nil
	}
	return fmt.Errorf(
		"本机无可用离线包（已查 %s、$OPSFLEET_K8S_WORKDIR、%s），且无 ofpk8s1 可拉取。\n"+
			"请用当前平台生成的离线包执行 install.sh（会写入 %s），或: sudo %s uninstall k8s --workdir <根>  或: sudo %s uninstall k8s --ref 'ofpk8s1…'",
		K8sLastBundlePath, K8sDefaultUninstallWorkdir, K8sLastBundlePath, progName, progName,
	)
}

func runCleanupPlaybook(agentRoot, inventoryPath string) error {
	pb := filepath.Join(agentRoot, "playbooks", "pre_cleanup.yml")
	for _, p := range []string{inventoryPath, agentRoot, pb} {
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("清理路径无效 %s: %w", p, err)
		}
	}
	run := func(name string, arg ...string) *exec.Cmd {
		c := exec.Command(name, arg...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Stdin = os.Stdin
		return c
	}
	if os.Geteuid() == 0 {
		c := run("ansible-playbook", "-i", inventoryPath, pb)
		c.Dir = agentRoot
		c.Env = append(os.Environ(), "ANSIBLE_HOST_KEY_CHECKING=False")
		return c.Run()
	}
	c := run("sudo", "-E", "ansible-playbook", "-i", inventoryPath, pb)
	c.Dir = agentRoot
	c.Env = append(os.Environ(), "ANSIBLE_HOST_KEY_CHECKING=False")
	return c.Run()
}

func k8sCleanupCmd() *cobra.Command {
	var pkgPath, workdir string
	cmd := &cobra.Command{
		Use:   "cleanup [install-ref]",
		Short: "按安装引用或离线包目录，对页面配置的全部节点执行 pre_cleanup",
		Long: `与 ansible-agent playbooks/pre_cleanup.yml 相同：对 inventory 中 k8s_cluster（全部 master/worker）停止 systemd 单元并删除 /etc/kubernetes、/var/lib/etcd 等。

传入控制台生成的 ofpk8s1… 时，会重新下载与「一键安装」相同的 zip（须在有效期内），无需保留解压目录。

示例:
  sudo ai-sre k8s cleanup 'ofpk8s1.…'
  sudo ai-sre k8s cleanup --workdir /path/to/解压目录
  sudo ai-sre k8s cleanup --package ./bundle.zip`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				if pkgPath != "" || workdir != "" {
					return errors.New("已传入安装引用时不要同时使用 --package 或 --workdir")
				}
				ref := strings.Trim(strings.TrimSpace(args[0]), `"'`)
				return runCleanupFromInviteRef(ref)
			}
			if pkgPath != "" && workdir != "" {
				return errors.New("--package 与 --workdir 二选一")
			}
			if pkgPath == "" && workdir == "" {
				return errors.New("请传入 ofpk8s1… 安装引用，或使用 --workdir / --package")
			}
			var root string
			if workdir != "" {
				var err error
				root, err = filepath.Abs(strings.TrimSpace(workdir))
				if err != nil {
					return err
				}
			} else {
				td, err := os.MkdirTemp("", "opsfleet-k8s-cleanup-*")
				if err != nil {
					return err
				}
				defer os.RemoveAll(td)
				if err := unzipFile(pkgPath, td); err != nil {
					return err
				}
				root, err = bundleInstallRoot(td)
				if err != nil {
					return err
				}
			}
			return runCleanupPlaybook(filepath.Join(root, "ansible-agent"), filepath.Join(root, "inventory", "hosts.ini"))
		},
	}
	cmd.Flags().StringVar(&pkgPath, "package", "", "离线 zip 路径")
	cmd.Flags().StringVar(&workdir, "workdir", "", "已解压根目录（含 install.sh、ansible-agent）")
	return cmd
}

func k8sUninstallCmd() *cobra.Command {
	var workdir string
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "在已解压的离线包目录上执行 Ansible pre_cleanup（清理 K8s/etcd 残留）",
		Long: `等同 k8s cleanup --workdir：停止 systemd 单元并删除 /var/lib/etcd、/etc/kubernetes 等。
推荐新流程使用 k8s cleanup 'ofpk8s1…'（无需保留解压目录）。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(workdir) == "" {
				return errors.New("请指定 --workdir <离线包解压根目录>")
			}
			root, err := filepath.Abs(workdir)
			if err != nil {
				return err
			}
			return runCleanupPlaybook(filepath.Join(root, "ansible-agent"), filepath.Join(root, "inventory", "hosts.ini"))
		},
	}
	cmd.Flags().StringVar(&workdir, "workdir", "", "离线包解压根目录")
	return cmd
}

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func opsfleetLogin(base, username, password string) (string, error) {
	payload, _ := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	req, err := http.NewRequest(http.MethodPost, base+"/api/auth/login", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login HTTP %d: %s", resp.StatusCode, truncateForErr(body, 512))
	}
	var wrap struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
		Msg     string `json:"msg"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		return "", fmt.Errorf("login parse: %w", err)
	}
	if wrap.Data.Token == "" {
		return "", fmt.Errorf("login: empty token (%s)", firstNonEmpty(wrap.Msg, wrap.Message, string(body)))
	}
	return wrap.Data.Token, nil
}

func downloadK8sBundle(base, token string, jsonBody []byte, outPath string) error {
	req, err := http.NewRequest(http.MethodPost, base+"/api/k8s/deploy/bundle", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{Timeout: 15 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return fmt.Errorf("bundle HTTP %d: %s", resp.StatusCode, truncateForErr(b, 2048))
	}
	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "application/json") {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return fmt.Errorf("unexpected JSON (check token/role): %s", truncateForErr(b, 2048))
	}
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}
	return nil
}

func unzipFile(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}
	for _, f := range r.File {
		path := filepath.Join(dest, f.Name)
		rel, err := filepath.Rel(filepath.Clean(dest), filepath.Clean(path))
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("illegal zip path: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, f.Mode()); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			out.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func truncateForErr(b []byte, max int) string {
	s := strings.TrimSpace(string(b))
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}
