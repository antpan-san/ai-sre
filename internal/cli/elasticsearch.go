package cli

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Elasticsearch 部署命令树。安装/更新规格由 OpsFleet 服务端下发，CLI 负责安全
// 落地：默认 single-node + xpack.security 关闭（PoC），并自动处理 ES 部署最常见
// 五大坑：
//  1. vm.max_map_count >= 262144（写 sysctl.d 并立即生效）
//  2. systemd LimitNOFILE/LimitMEMLOCK 通过 drop-in 注入，避免改主单元
//  3. JVM heap 通过 jvm.options.d/heap.options 写入，可重复幂等
//  4. 启动慢：在 enable-start 之后插入 wait-ready 步骤，轮询 _cluster/health
//  5. 卸载残留 data/log，提供 --purge-data；--force 端到端清理
//
// 安装方式：
//   - package：apt/yum 官方仓库
//   - docker：官方镜像 + ulimit
//   - binary：官方 Linux tarball 解压到 install_prefix，ES_PATH_CONF=prefix/config，自管 systemd
func elasticsearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "elasticsearch",
		Short: "Elasticsearch 快诊、维护（diagnose / update / uninstall）",
	}
	cmd.AddCommand(elasticsearchDiagnoseCmd(), elasticsearchUpdateCmd(), elasticsearchUninstallCmd())
	return cmd
}

func elasticsearchUpdateCmd() *cobra.Command {
	var opts serviceUpdateOptions
	cmd := &cobra.Command{
		Use:   "update",
		Short: "从 OpsFleet 拉取最新 Elasticsearch 部署规格并重启生效",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServiceUpdate(cmd, "elasticsearch", opts)
		},
	}
	cmd.Flags().StringVar(&opts.APIURL, "api-url", "", "OpsFleet API base，例如 http://host:9080/ft-api；默认读取本机安装状态")
	cmd.Flags().StringVar(&opts.DeployID, "deploy-id", "", "服务端部署 ID；默认读取本机安装状态")
	cmd.Flags().StringVar(&opts.Token, "token", "", "服务端部署 token；默认读取本机安装状态")
	cmd.Flags().StringVar(&opts.FromURL, "from", "", "完整 spec URL（可替代 api-url/deploy-id/token）")
	return cmd
}

func elasticsearchUninstallCmd() *cobra.Command {
	var opts elasticsearchUninstallOptions
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "卸载 ai-sre 安装的 Elasticsearch（默认仅停服 + 移除 ai-sre 配置）",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runElasticsearchUninstall(cmd, opts)
		},
	}
	cmd.Flags().BoolVar(&opts.PurgePackage, "purge-package", false, "同时卸载 elasticsearch 包或删除二进制安装目录")
	cmd.Flags().BoolVar(&opts.PurgeData, "purge-data", false, "同时清理数据/日志/keystore（默认保留以防误删）")
	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "强制清理本机所有 Elasticsearch 进程/包/容器/配置/数据，绕过 ai-sre 状态校验")
	return cmd
}

func runElasticsearchUninstall(cmd *cobra.Command, opts elasticsearchUninstallOptions) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("卸载 Elasticsearch 需 root 权限，请使用: sudo %s elasticsearch uninstall", progName)
	}
	if opts.Force {
		state, _ := loadServiceDeploymentState("elasticsearch")
		fmt.Fprintln(cmd.OutOrStdout(), "[uninstall] force running: start")
		if err := runBash(elasticsearchForceUninstallScript()); err != nil {
			return fmt.Errorf("elasticsearch force uninstall failed: %w", err)
		}
		_ = removeServiceDeploymentState("elasticsearch")
		if state != nil && state.Service == "elasticsearch" && state.APIURL != "" && state.DeployID != "" && state.Token != "" {
			_ = postServiceFinish(state.APIURL, state.DeployID, state.Token, "uninstalled", "elasticsearch force uninstalled")
		}
		fmt.Fprintln(cmd.OutOrStdout(), "[uninstall] force success: ok")
		return nil
	}
	state, err := loadServiceDeploymentState("elasticsearch")
	if err != nil {
		return fmt.Errorf("未发现 ai-sre 安装的 Elasticsearch 状态，拒绝卸载。只有通过 ai-sre service install 安装并写入本机状态的 Elasticsearch 才允许常规卸载（或使用 --force）: %w", err)
	}
	if state.Service != "elasticsearch" || state.APIURL == "" || state.DeployID == "" || state.Token == "" {
		return fmt.Errorf("本机 Elasticsearch 安装状态不完整或服务类型不匹配，拒绝卸载（如需强制清理请使用 --force）")
	}
	spec := &serviceInstallSpec{Service: "elasticsearch", InstallMethod: "package", Params: map[string]interface{}{}}
	if fetched, fetchErr := fetchServiceSpec(fmt.Sprintf("%s/api/service-deploy/deployments/%s/spec?token=%s", strings.TrimRight(state.APIURL, "/"), url.PathEscape(state.DeployID), url.QueryEscape(state.Token))); fetchErr == nil {
		spec = fetched
	} else {
		fmt.Fprintf(cmd.ErrOrStderr(), "[uninstall] warning: fetch server spec failed, fallback to local protected uninstall: %v\n", fetchErr)
	}
	method := strParam(spec, "install_method", spec.InstallMethod)
	if method == "" {
		method = "package"
	}
	dataDir := strParam(spec, "path_data", "/var/lib/elasticsearch")
	logDir := strParam(spec, "path_logs", "/var/log/elasticsearch")
	installPrefix := strParam(spec, "install_prefix", "/opt/elasticsearch")
	script := elasticsearchUninstallScript(method, opts.PurgePackage, opts.PurgeData, dataDir, logDir, installPrefix)
	fmt.Fprintln(cmd.OutOrStdout(), "[uninstall] running: start")
	if err := runBash(script); err != nil {
		return fmt.Errorf("elasticsearch uninstall failed: %w", err)
	}
	_ = removeServiceDeploymentState("elasticsearch")
	_ = postServiceFinish(state.APIURL, state.DeployID, state.Token, "uninstalled", "elasticsearch uninstalled from ai-sre managed state")
	fmt.Fprintln(cmd.OutOrStdout(), "[uninstall] success: ok")
	return nil
}

// ---------- 安装/更新 step 列表（被 runServiceTemplate / runServiceUpdateTemplate 调度） ----------

func elasticsearchInstallSteps(spec *serviceInstallSpec) []templateStep {
	return []templateStep{
		{"system-tune", elasticsearchSystemTuneScript(spec)},
		{"install", elasticsearchInstallScript(spec)},
		{"write-config", elasticsearchConfigScript(spec)},
		{"enable-start", elasticsearchStartScript(spec)},
		{"wait-ready", elasticsearchWaitReadyScript(spec)},
		{"port-check", servicePortScript(spec)},
		{"service-check", elasticsearchHealthScript(spec)},
	}
}

func elasticsearchUpdateSteps(spec *serviceInstallSpec) []templateStep {
	return []templateStep{
		{"system-tune", elasticsearchSystemTuneScript(spec)},
		{"write-config", elasticsearchConfigScript(spec)},
		{"restart", elasticsearchRestartScript(spec)},
		{"wait-ready", elasticsearchWaitReadyScript(spec)},
		{"port-check", servicePortScript(spec)},
		{"service-check", elasticsearchHealthScript(spec)},
	}
}

// ---------- YAML（package / docker / binary 共用） ----------

func elasticsearchVersionForURL(spec *serviceInstallSpec) string {
	v := strings.TrimSpace(spec.Version)
	if v == "" {
		v = strParam(spec, "version", "8.13.4")
	}
	if strings.EqualFold(v, "latest") {
		return "8.13.4"
	}
	return v
}

func elasticsearchElasticsearchYAML(spec *serviceInstallSpec) string {
	httpPort := intParam(spec, "http_port", 9200)
	transportPort := intParam(spec, "transport_port", 9300)
	clusterName := strParam(spec, "cluster_name", "opsfleet-es")
	nodeName := strParam(spec, "node_name", "")
	networkHost := strParam(spec, "network_host", "0.0.0.0")
	dataDir := strParam(spec, "path_data", "/var/lib/elasticsearch")
	logDir := strParam(spec, "path_logs", "/var/log/elasticsearch")
	discovery := strParam(spec, "discovery_type", "single-node")
	seedHosts := strings.TrimSpace(strParam(spec, "seed_hosts", ""))
	initialMasters := strings.TrimSpace(strParam(spec, "initial_master_nodes", ""))
	xpackOn := boolParam(spec, "xpack_security", false)
	memlock := boolParam(spec, "bootstrap_memory_lock", false)

	yml := []string{
		fmt.Sprintf("cluster.name: %s", clusterName),
		fmt.Sprintf("network.host: %s", networkHost),
		fmt.Sprintf("http.port: %d", httpPort),
		fmt.Sprintf("transport.port: %d", transportPort),
		fmt.Sprintf("path.data: %s", dataDir),
		fmt.Sprintf("path.logs: %s", logDir),
	}
	if nodeName != "" {
		yml = append(yml, fmt.Sprintf("node.name: %s", nodeName))
	}
	if memlock {
		yml = append(yml, "bootstrap.memory_lock: true")
	}
	if discovery == "single-node" {
		yml = append(yml, "discovery.type: single-node")
	} else {
		if seedHosts != "" {
			yml = append(yml, fmt.Sprintf("discovery.seed_hosts: [%s]", csvList(seedHosts)))
		}
		if initialMasters != "" {
			yml = append(yml, fmt.Sprintf("cluster.initial_master_nodes: [%s]", csvList(initialMasters)))
		}
	}
	if xpackOn {
		yml = append(yml,
			"xpack.security.enabled: true",
			"xpack.security.http.ssl.enabled: false",
			"xpack.security.transport.ssl.enabled: false",
		)
	} else {
		yml = append(yml,
			"xpack.security.enabled: false",
			"xpack.security.http.ssl.enabled: false",
			"xpack.security.transport.ssl.enabled: false",
		)
	}
	return strings.Join(yml, "\n") + "\n"
}

func elasticsearchHeapOptions(spec *serviceInstallSpec) string {
	heap := strParam(spec, "heap_size", "1g")
	return fmt.Sprintf("-Xms%s\n-Xmx%s\n", heap, heap)
}

// ---------- 脚本生成 ----------

func elasticsearchSystemTuneScript(spec *serviceInstallSpec) string {
	if !boolParam(spec, "vm_max_map_count_setup", true) {
		return `echo "[system-tune] skipped (vm_max_map_count_setup=false)"`
	}
	return `mkdir -p /etc/sysctl.d
cat >/etc/sysctl.d/99-elasticsearch.conf <<'EOF'
# Managed by ai-sre: required for Elasticsearch (mmapfs)
vm.max_map_count=262144
fs.file-max=1048576
EOF
sysctl --system >/dev/null 2>&1 || sysctl -p /etc/sysctl.d/99-elasticsearch.conf >/dev/null 2>&1 || true
sysctl -w vm.max_map_count=262144 >/dev/null 2>&1 || true
sysctl -w fs.file-max=1048576 >/dev/null 2>&1 || true
echo "[system-tune] vm.max_map_count=$(sysctl -n vm.max_map_count) fs.file-max=$(sysctl -n fs.file-max)"`
}

func elasticsearchInstallScript(spec *serviceInstallSpec) string {
	method := strParam(spec, "install_method", spec.InstallMethod)
	if method == "docker" {
		return `command -v docker >/dev/null 2>&1 || { echo "docker is required for elasticsearch docker install" >&2; exit 1; }`
	}
	if method == "binary" {
		return elasticsearchBinaryInstallScript(spec)
	}
	major := elasticsearchMajor(strParam(spec, "version", "8"))
	if major == "" {
		major = "8"
	}
	return fmt.Sprintf(`set -e
if command -v apt-get >/dev/null 2>&1; then
  DEBIAN_FRONTEND=noninteractive apt-get update -y
  DEBIAN_FRONTEND=noninteractive apt-get install -y apt-transport-https ca-certificates curl gnupg
  install -d -m 0755 /usr/share/keyrings
  if [ ! -s /usr/share/keyrings/elasticsearch-keyring.gpg ]; then
    curl -fsSL https://artifacts.elastic.co/GPG-KEY-elasticsearch | gpg --dearmor -o /usr/share/keyrings/elasticsearch-keyring.gpg
    chmod 0644 /usr/share/keyrings/elasticsearch-keyring.gpg
  fi
  echo "deb [signed-by=/usr/share/keyrings/elasticsearch-keyring.gpg] https://artifacts.elastic.co/packages/%[1]s.x/apt stable main" > /etc/apt/sources.list.d/elastic-%[1]s.x.list
  DEBIAN_FRONTEND=noninteractive apt-get update -y
  DEBIAN_FRONTEND=noninteractive apt-get install -y elasticsearch
elif command -v dnf >/dev/null 2>&1 || command -v yum >/dev/null 2>&1; then
  PKG=$(command -v dnf || command -v yum)
  rpm --import https://artifacts.elastic.co/GPG-KEY-elasticsearch || true
  cat >/etc/yum.repos.d/elasticsearch.repo <<EOF
[elasticsearch]
name=Elasticsearch repository for %[1]s.x packages
baseurl=https://artifacts.elastic.co/packages/%[1]s.x/yum
gpgcheck=1
gpgkey=https://artifacts.elastic.co/GPG-KEY-elasticsearch
enabled=0
autorefresh=1
type=rpm-md
EOF
  $PKG install -y --enablerepo=elasticsearch elasticsearch
else
  echo "no supported package manager (apt/dnf/yum) found for elasticsearch install" >&2
  exit 1
fi`, major)
}

func elasticsearchBinaryInstallScript(spec *serviceInstallSpec) string {
	prefix := strParam(spec, "install_prefix", "/opt/elasticsearch")
	ver := elasticsearchVersionForURL(spec)
	customURL := strings.TrimSpace(strParam(spec, "binary_url", ""))
	dataDir := strParam(spec, "path_data", "/var/lib/elasticsearch")
	logDir := strParam(spec, "path_logs", "/var/log/elasticsearch")

	// shell 内根据 CUSTOM_URL / 架构拼 URL；PREFIX 由 ai-sre 注入已转义路径
	return fmt.Sprintf(`set -euo pipefail
PREFIX=%s
VER=%s
CUSTOM_URL=%s
DATA_DIR=%s
LOG_DIR=%s

if command -v apt-get >/dev/null 2>&1; then
  DEBIAN_FRONTEND=noninteractive apt-get update -y
  DEBIAN_FRONTEND=noninteractive apt-get install -y curl ca-certificates tar gzip
elif command -v dnf >/dev/null 2>&1; then
  dnf install -y curl tar gzip ca-certificates || true
elif command -v yum >/dev/null 2>&1; then
  yum install -y curl tar gzip ca-certificates || true
else
  echo "binary install requires curl+tar (apt/dnf/yum)" >&2
  exit 1
fi

id elasticsearch >/dev/null 2>&1 || useradd --system --no-create-home --shell /bin/false -c "Elasticsearch" elasticsearch

systemctl disable --now elasticsearch 2>/dev/null || true
rm -rf "$PREFIX"
mkdir -p "$(dirname "$PREFIX")"
install -d -m 0755 /var/lib/ai-sre
echo binary >/var/lib/ai-sre/elasticsearch-install-method

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

case "$(uname -m)" in
  x86_64|amd64) TARCH=x86_64 ;;
  aarch64|arm64) TARCH=aarch64 ;;
  *) echo "unsupported arch: $(uname -m)" >&2; exit 1 ;;
esac

if [ -n "$CUSTOM_URL" ]; then
  URL="$CUSTOM_URL"
else
  URL="https://artifacts.elastic.co/downloads/elasticsearch/elasticsearch-${VER}-linux-${TARCH}.tar.gz"
fi

echo "[install] downloading $URL"
curl -fsSL "$URL" | tar xz -C "$TMP"
TOP=$(find "$TMP" -mindepth 1 -maxdepth 1 -type d | head -1)
test -n "$TOP"
mv "$TOP" "$PREFIX"

mkdir -p "$PREFIX/config/jvm.options.d"
mkdir -p "$DATA_DIR" "$LOG_DIR"
chown -R elasticsearch:elasticsearch "$PREFIX" "$DATA_DIR" "$LOG_DIR"

cat >/etc/systemd/system/elasticsearch.service <<UNITEND
[Unit]
Description=Elasticsearch (ai-sre binary tarball)
Documentation=https://www.elastic.co
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
User=elasticsearch
Group=elasticsearch
Environment=ES_PATH_CONF=${PREFIX}/config
ExecStart=${PREFIX}/bin/elasticsearch
Restart=on-failure
RestartSec=10
LimitNOFILE=65535
LimitMEMLOCK=infinity
LimitNPROC=4096
TimeoutStartSec=180

[Install]
WantedBy=multi-user.target
UNITEND

systemctl daemon-reload
echo "[install] binary elasticsearch installed under $PREFIX"`,
		shellQuote(prefix),
		shellQuote(ver),
		shellQuote(customURL),
		shellQuote(dataDir),
		shellQuote(logDir),
	)
}

func elasticsearchConfigScript(spec *serviceInstallSpec) string {
	method := strParam(spec, "install_method", spec.InstallMethod)
	body := elasticsearchElasticsearchYAML(spec)
	heapBody := elasticsearchHeapOptions(spec)
	dataDir := strParam(spec, "path_data", "/var/lib/elasticsearch")
	logDir := strParam(spec, "path_logs", "/var/log/elasticsearch")

	if method == "docker" {
		return fmt.Sprintf(`mkdir -p /etc/elasticsearch/conf.d %s %s
cat >/etc/elasticsearch/elasticsearch.yml <<'EOF'
%sEOF
cat >/etc/elasticsearch/jvm.options.d/heap.options 2>/dev/null <<'EOF' || true
%sEOF
chmod 0644 /etc/elasticsearch/elasticsearch.yml
echo "[write-config] /etc/elasticsearch/elasticsearch.yml OK"`, dataDir, logDir, body, heapBody)
	}

	if method == "binary" {
		prefix := strParam(spec, "install_prefix", "/opt/elasticsearch")
		return fmt.Sprintf(`mkdir -p %s/config/jvm.options.d %s %s
cat >%s/config/elasticsearch.yml <<'EOF'
%sEOF
cat >%s/config/jvm.options.d/heap.options <<'EOF'
%sEOF
chown -R elasticsearch:elasticsearch %s %s %s 2>/dev/null || true
chmod 0640 %s/config/elasticsearch.yml %s/config/jvm.options.d/heap.options 2>/dev/null || true
systemctl daemon-reload
echo "[write-config] binary config under %s/config"`,
			shellQuote(prefix), shellQuote(dataDir), shellQuote(logDir),
			shellQuote(prefix), body,
			shellQuote(prefix), heapBody,
			shellQuote(prefix), shellQuote(dataDir), shellQuote(logDir),
			shellQuote(prefix), shellQuote(prefix),
			shellQuote(prefix),
		)
	}

	dropIn := `[Service]
LimitNOFILE=65535
LimitMEMLOCK=infinity
LimitNPROC=4096
TimeoutStartSec=180
`
	return fmt.Sprintf(`mkdir -p /etc/elasticsearch /etc/elasticsearch/jvm.options.d /etc/systemd/system/elasticsearch.service.d %s %s
cat >/etc/elasticsearch/elasticsearch.yml <<'EOF'
%sEOF
cat >/etc/elasticsearch/jvm.options.d/heap.options <<'EOF'
%sEOF
cat >/etc/systemd/system/elasticsearch.service.d/limits.conf <<'EOF'
%sEOF
chown -R elasticsearch:elasticsearch /etc/elasticsearch %s %s 2>/dev/null || true
chmod 0660 /etc/elasticsearch/elasticsearch.yml /etc/elasticsearch/jvm.options.d/heap.options 2>/dev/null || true
systemctl daemon-reload
echo "[write-config] /etc/elasticsearch updated"`, dataDir, logDir, body, heapBody, dropIn, dataDir, logDir)
}

func elasticsearchStartScript(spec *serviceInstallSpec) string {
	method := strParam(spec, "install_method", spec.InstallMethod)
	if method == "docker" {
		httpPort := intParam(spec, "http_port", 9200)
		transportPort := intParam(spec, "transport_port", 9300)
		dataDir := strParam(spec, "path_data", "/var/lib/elasticsearch")
		logDir := strParam(spec, "path_logs", "/var/log/elasticsearch")
		heap := strParam(spec, "heap_size", "1g")
		clusterName := strParam(spec, "cluster_name", "opsfleet-es")
		discovery := strParam(spec, "discovery_type", "single-node")
		image := elasticsearchDockerImage(spec)
		envs := []string{
			fmt.Sprintf(`-e cluster.name=%s`, shellQuote(clusterName)),
			fmt.Sprintf(`-e ES_JAVA_OPTS=%s`, shellQuote(fmt.Sprintf("-Xms%s -Xmx%s", heap, heap))),
		}
		if discovery == "single-node" {
			envs = append(envs, "-e discovery.type=single-node")
		}
		if !boolParam(spec, "xpack_security", false) {
			envs = append(envs, "-e xpack.security.enabled=false")
		}
		return fmt.Sprintf(`mkdir -p %s %s
docker rm -f elasticsearch 2>/dev/null || true
docker run -d --name elasticsearch --restart=always \
  -p %d:9200 -p %d:9300 \
  --ulimit memlock=-1:-1 \
  --ulimit nofile=65535:65535 \
  %s \
  -v %s:/usr/share/elasticsearch/data \
  -v %s:/usr/share/elasticsearch/logs \
  %s
echo "[enable-start] container started: elasticsearch"`, dataDir, logDir, httpPort, transportPort, strings.Join(envs, " \\\n  "), dataDir, logDir, image)
	}
	return `systemctl daemon-reload
systemctl enable elasticsearch
systemctl restart elasticsearch
echo "[enable-start] systemd elasticsearch started"`
}

func elasticsearchRestartScript(spec *serviceInstallSpec) string {
	if strParam(spec, "install_method", spec.InstallMethod) == "docker" {
		return elasticsearchStartScript(spec)
	}
	return `systemctl daemon-reload
systemctl restart elasticsearch
echo "[restart] systemd elasticsearch restarted"`
}

func elasticsearchWaitReadyScript(spec *serviceInstallSpec) string {
	httpPort := intParam(spec, "http_port", 9200)
	scheme, authArg := elasticsearchHTTPAuth(spec)
	return fmt.Sprintf(`URL=%s://127.0.0.1:%d/_cluster/health
echo "[wait-ready] polling $URL up to 90s ..."
for i in $(seq 1 45); do
  if curl -fsS %s "$URL" >/dev/null 2>&1; then
    echo "[wait-ready] elasticsearch is responding"
    exit 0
  fi
  sleep 2
done
echo "[wait-ready] elasticsearch did not become ready within 90s" >&2
echo "[wait-ready] last health probe output:" >&2
curl -s %s "$URL" >&2 || true
exit 1`, scheme, httpPort, authArg, authArg)
}

func elasticsearchHealthScript(spec *serviceInstallSpec) string {
	httpPort := intParam(spec, "http_port", 9200)
	scheme, authArg := elasticsearchHTTPAuth(spec)
	return fmt.Sprintf(`HEALTH=$(curl -fsS %s %s://127.0.0.1:%d/_cluster/health)
echo "[service-check] $HEALTH"
echo "$HEALTH" | grep -E '"status":"(green|yellow)"' >/dev/null
echo "[service-check] cluster status acceptable"`, authArg, scheme, httpPort)
}

func elasticsearchHTTPAuth(spec *serviceInstallSpec) (scheme string, curlArgs string) {
	scheme = "http"
	if boolParam(spec, "xpack_security", false) {
		scheme = "https"
		user := strParam(spec, "xpack_user", "elastic")
		pass := strParam(spec, "xpack_password", "")
		if pass != "" {
			return "https", fmt.Sprintf("-k -u %s:%s", shellQuote(user), shellQuote(pass))
		}
		return "https", "-k"
	}
	return scheme, ""
}

func elasticsearchDockerImage(spec *serviceInstallSpec) string {
	v := strings.TrimSpace(spec.Version)
	if v == "" {
		v = strParam(spec, "version", "8.13.4")
	}
	if strings.EqualFold(v, "latest") {
		v = "8.13.4"
	}
	if v == "" {
		v = "8.13.4"
	}
	return "docker.elastic.co/elasticsearch/elasticsearch:" + v
}

func elasticsearchMajor(version string) string {
	v := strings.TrimSpace(version)
	if v == "" {
		return "8"
	}
	if strings.EqualFold(v, "latest") {
		return "8"
	}
	if i := strings.Index(v, "."); i > 0 {
		return v[:i]
	}
	return v
}

func csvList(raw string) string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == ' ' || r == '\t'
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, fmt.Sprintf("%q", p))
	}
	return strings.Join(out, ", ")
}

func elasticsearchUninstallScript(method string, purgePackage, purgeData bool, dataDir, logDir, installPrefix string) string {
	if dataDir == "" {
		dataDir = "/var/lib/elasticsearch"
	}
	if logDir == "" {
		logDir = "/var/log/elasticsearch"
	}
	if installPrefix == "" {
		installPrefix = "/opt/elasticsearch"
	}
	dataPurge := ""
	if purgeData {
		dataPurge = fmt.Sprintf(`rm -rf %s %s
`, shellQuote(dataDir), shellQuote(logDir))
	}
	if method == "docker" {
		return fmt.Sprintf(`if command -v docker >/dev/null 2>&1; then
  docker rm -f elasticsearch 2>/dev/null || true
fi
rm -f /etc/elasticsearch/elasticsearch.yml /etc/elasticsearch/jvm.options.d/heap.options
%sexit 0`, dataPurge)
	}

	if method == "binary" {
		rmPrefix := ""
		if purgePackage {
			rmPrefix = fmt.Sprintf(`rm -rf %s
`, shellQuote(installPrefix))
		}
		return fmt.Sprintf(`if command -v systemctl >/dev/null 2>&1; then
  systemctl disable --now elasticsearch 2>/dev/null || true
fi
rm -f /etc/systemd/system/elasticsearch.service
rm -f /var/lib/ai-sre/elasticsearch-install-method
%s%srm -f /etc/sysctl.d/99-elasticsearch.conf
sysctl --system >/dev/null 2>&1 || true
if command -v systemctl >/dev/null 2>&1; then
  systemctl daemon-reload 2>/dev/null || true
fi
exit 0`, rmPrefix, dataPurge)
	}

	pkgPurge := ""
	if purgePackage {
		pkgPurge = `if command -v apt-get >/dev/null 2>&1; then
  DEBIAN_FRONTEND=noninteractive apt-get purge -y elasticsearch || true
  DEBIAN_FRONTEND=noninteractive apt-get autoremove -y || true
elif command -v dnf >/dev/null 2>&1; then
  dnf remove -y elasticsearch || true
elif command -v yum >/dev/null 2>&1; then
  yum remove -y elasticsearch || true
fi
`
	}
	return fmt.Sprintf(`if command -v systemctl >/dev/null 2>&1; then
  systemctl disable --now elasticsearch 2>/dev/null || true
fi
rm -f /etc/systemd/system/elasticsearch.service.d/limits.conf
rm -f /etc/elasticsearch/jvm.options.d/heap.options
rm -f /etc/sysctl.d/99-elasticsearch.conf
sysctl --system >/dev/null 2>&1 || true
%s%sif command -v systemctl >/dev/null 2>&1; then
  systemctl daemon-reload 2>/dev/null || true
fi
exit 0`, pkgPurge, dataPurge)
}

func elasticsearchForceUninstallScript() string {
	return `set +e
if command -v systemctl >/dev/null 2>&1; then
  systemctl disable --now elasticsearch 2>/dev/null || true
fi
if command -v docker >/dev/null 2>&1; then
  docker ps -aq --filter name='^/elasticsearch$' | xargs -r docker rm -f
  docker images --format '{{.Repository}}:{{.Tag}} {{.ID}}' | awk '$1 ~ /elasticsearch/ {print $2}' | xargs -r docker rmi -f
fi
pkill -f 'org.elasticsearch.bootstrap.Elasticsearch' 2>/dev/null || true
pkill -x elasticsearch 2>/dev/null || true
if command -v apt-get >/dev/null 2>&1; then
  DEBIAN_FRONTEND=noninteractive apt-get purge -y elasticsearch 'elasticsearch-*' 2>/dev/null || true
  DEBIAN_FRONTEND=noninteractive apt-get autoremove -y 2>/dev/null || true
elif command -v dnf >/dev/null 2>&1; then
  dnf remove -y elasticsearch 'elasticsearch-*' 2>/dev/null || true
elif command -v yum >/dev/null 2>&1; then
  yum remove -y elasticsearch 'elasticsearch-*' 2>/dev/null || true
fi
rm -rf /etc/elasticsearch /var/lib/elasticsearch /var/log/elasticsearch /var/cache/elasticsearch
rm -rf /usr/share/elasticsearch /opt/elasticsearch
rm -f /etc/systemd/system/elasticsearch.service /etc/systemd/system/elasticsearch.service.d/limits.conf
rm -f /etc/sysctl.d/99-elasticsearch.conf
rm -f /usr/share/keyrings/elasticsearch-keyring.gpg
rm -f /etc/apt/sources.list.d/elastic-*.list
rm -f /etc/yum.repos.d/elasticsearch.repo
rm -f /var/lib/ai-sre/elasticsearch-install-method
sysctl --system >/dev/null 2>&1 || true
if command -v systemctl >/dev/null 2>&1; then
  systemctl daemon-reload 2>/dev/null || true
fi
if command -v elasticsearch >/dev/null 2>&1; then
  echo "elasticsearch command still exists after cleanup: $(command -v elasticsearch)" >&2
  exit 1
fi
exit 0`
}
