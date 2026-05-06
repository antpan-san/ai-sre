package cli

import (
	"bytes"
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

type serviceInstallOptions struct {
	APIURL   string
	DeployID string
	Token    string
	FromURL  string
}

type serviceUpdateOptions = serviceInstallOptions

type serviceInstallSpec struct {
	ID            string                 `json:"id"`
	Service       string                 `json:"service"`
	Profile       string                 `json:"profile"`
	InstallMethod string                 `json:"install_method"`
	Version       string                 `json:"version"`
	Params        map[string]interface{} `json:"params"`
}

type serviceDeploymentState struct {
	Service   string    `json:"service"`
	APIURL    string    `json:"api_url"`
	DeployID  string    `json:"deploy_id"`
	Token     string    `json:"token"`
	UpdatedAt time.Time `json:"updated_at"`
}

type nginxUninstallOptions struct {
	PurgePackage bool
}

func serviceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "基础服务安装（从 OpsFleet 服务端拉取参数）",
	}
	cmd.AddCommand(serviceInstallCmd())
	return cmd
}

func serviceInstallCmd() *cobra.Command {
	var opts serviceInstallOptions
	cmd := &cobra.Command{
		Use:   "install",
		Short: "按服务端部署规格安装基础服务",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServiceInstall(cmd, opts)
		},
	}
	cmd.Flags().StringVar(&opts.APIURL, "api-url", "", "OpsFleet API base，例如 http://host:9080/ft-api")
	cmd.Flags().StringVar(&opts.DeployID, "deploy-id", "", "服务端部署 ID")
	cmd.Flags().StringVar(&opts.Token, "token", "", "服务端部署 token")
	cmd.Flags().StringVar(&opts.FromURL, "from", "", "完整 spec URL（可替代 api-url/deploy-id/token）")
	return cmd
}

func runServiceInstall(cmd *cobra.Command, opts serviceInstallOptions) error {
	specURL, apiURL, deployID, token, err := resolveServiceSpecURL(opts)
	if err != nil {
		return err
	}
	spec, err := fetchServiceSpec(specURL)
	if err != nil {
		_ = postServiceFinish(apiURL, deployID, token, "failed", err.Error())
		return err
	}
	if spec.ID == "" {
		spec.ID = deployID
	}
	report := func(step, status, msg string) {
		fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s: %s\n", step, status, msg)
		_ = postServiceEvent(apiURL, deployID, token, step, status, msg)
	}
	if err := runServiceTemplate(spec, report); err != nil {
		_ = postServiceFinish(apiURL, deployID, token, "failed", err.Error())
		return err
	}
	if err := saveServiceDeploymentState(spec.Service, apiURL, deployID, token); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "[state] warning: save local deployment state failed: %v\n", err)
	}
	return postServiceFinish(apiURL, deployID, token, "success", "service install completed")
}

func runServiceUpdate(cmd *cobra.Command, service string, opts serviceUpdateOptions) error {
	opts = fillServiceUpdateOptionsFromState(service, opts)
	specURL, apiURL, deployID, token, err := resolveServiceSpecURL(serviceInstallOptions(opts))
	if err != nil {
		return fmt.Errorf("%w; 如首次安装已成功，请使用 sudo 执行，或显式传入 --api-url/--deploy-id/--token", err)
	}
	spec, err := fetchServiceSpec(specURL)
	if err != nil {
		_ = postServiceFinish(apiURL, deployID, token, "failed", err.Error())
		return err
	}
	if spec.ID == "" {
		spec.ID = deployID
	}
	if spec.Service != service {
		return fmt.Errorf("deployment service mismatch: want %s, got %s", service, spec.Service)
	}
	report := func(step, status, msg string) {
		fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s: %s\n", step, status, msg)
		_ = postServiceEvent(apiURL, deployID, token, step, status, msg)
	}
	if err := runServiceUpdateTemplate(spec, report); err != nil {
		_ = postServiceFinish(apiURL, deployID, token, "failed", err.Error())
		return err
	}
	if err := saveServiceDeploymentState(service, apiURL, deployID, token); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "[state] warning: save local deployment state failed: %v\n", err)
	}
	return postServiceFinish(apiURL, deployID, token, "success", "service update completed")
}

func runNginxUninstall(cmd *cobra.Command, opts nginxUninstallOptions) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("卸载 Nginx 需 root 权限，请使用: sudo %s nginx uninstall", progName)
	}
	state, err := loadServiceDeploymentState("nginx")
	if err != nil {
		return fmt.Errorf("未发现 ai-sre 安装的 Nginx 状态，拒绝卸载。只有通过 ai-sre service install 安装并写入本机状态的 Nginx 才允许卸载: %w", err)
	}
	if state.Service != "nginx" || state.APIURL == "" || state.DeployID == "" || state.Token == "" {
		return fmt.Errorf("本机 Nginx 安装状态不完整或服务类型不匹配，拒绝卸载")
	}
	spec := &serviceInstallSpec{Service: "nginx", InstallMethod: "package", Params: map[string]interface{}{}}
	if fetched, fetchErr := fetchServiceSpec(fmt.Sprintf("%s/api/service-deploy/deployments/%s/spec?token=%s", strings.TrimRight(state.APIURL, "/"), url.PathEscape(state.DeployID), url.QueryEscape(state.Token))); fetchErr == nil {
		spec = fetched
	} else {
		fmt.Fprintf(cmd.ErrOrStderr(), "[uninstall] warning: fetch server spec failed, fallback to local protected uninstall: %v\n", fetchErr)
	}
	method := strParam(spec, "install_method", spec.InstallMethod)
	if method == "" {
		method = "package"
	}
	script := nginxUninstallScript(method, opts.PurgePackage)
	fmt.Fprintln(cmd.OutOrStdout(), "[uninstall] running: start")
	if err := runBash(script); err != nil {
		return fmt.Errorf("nginx uninstall failed: %w", err)
	}
	if err := removeServiceDeploymentState("nginx"); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "[state] warning: remove local deployment state failed: %v\n", err)
	}
	_ = postServiceFinish(state.APIURL, state.DeployID, state.Token, "uninstalled", "nginx uninstalled from ai-sre managed state")
	fmt.Fprintln(cmd.OutOrStdout(), "[uninstall] success: ok")
	return nil
}

func resolveServiceSpecURL(opts serviceInstallOptions) (specURL, apiURL, deployID, token string, err error) {
	if strings.TrimSpace(opts.FromURL) != "" {
		u, e := url.Parse(opts.FromURL)
		if e != nil {
			err = e
			return
		}
		token = u.Query().Get("token")
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		for i := range parts {
			if parts[i] == "deployments" && i+1 < len(parts) {
				deployID = parts[i+1]
				break
			}
		}
		path := u.Path
		if idx := strings.Index(path, "/api/service-deploy/"); idx >= 0 {
			u.Path = strings.TrimRight(path[:idx], "/")
		}
		u.RawQuery = ""
		apiURL = strings.TrimRight(u.String(), "/")
		specURL = opts.FromURL
		return
	}
	apiURL = strings.TrimRight(opts.APIURL, "/")
	deployID = strings.TrimSpace(opts.DeployID)
	token = strings.TrimSpace(opts.Token)
	if apiURL == "" || deployID == "" || token == "" {
		err = fmt.Errorf("requires --api-url, --deploy-id and --token (or --from)")
		return
	}
	specURL = fmt.Sprintf("%s/api/service-deploy/deployments/%s/spec?token=%s", apiURL, url.PathEscape(deployID), url.QueryEscape(token))
	return
}

func fetchServiceSpec(specURL string) (*serviceInstallSpec, error) {
	resp, err := http.Get(specURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("fetch spec status=%d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var envelope struct {
		Code int                `json:"code"`
		Data serviceInstallSpec `json:"data"`
		Msg  string             `json:"msg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	if envelope.Code != 200 {
		return nil, fmt.Errorf("fetch spec failed: %s", envelope.Msg)
	}
	return &envelope.Data, nil
}

func postServiceEvent(apiURL, deployID, token, step, status, message string) error {
	return postServiceJSON(apiURL, deployID, token, "events", map[string]string{
		"step": step, "status": status, "message": message,
	})
}

func postServiceFinish(apiURL, deployID, token, status, message string) error {
	return postServiceJSON(apiURL, deployID, token, "finish", map[string]string{
		"status": status, "message": message,
	})
}

func postServiceJSON(apiURL, deployID, token, action string, payload map[string]string) error {
	if apiURL == "" || deployID == "" || token == "" {
		return nil
	}
	b, _ := json.Marshal(payload)
	endpoint := fmt.Sprintf("%s/api/service-deploy/deployments/%s/%s?token=%s", strings.TrimRight(apiURL, "/"), url.PathEscape(deployID), action, url.QueryEscape(token))
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("post %s status=%d", action, resp.StatusCode)
	}
	return nil
}

func runServiceTemplate(spec *serviceInstallSpec, report func(string, string, string)) error {
	steps := []struct {
		name string
		body string
	}{
		{"install", serviceInstallScript(spec)},
		{"write-config", serviceConfigScript(spec)},
		{"enable-start", serviceStartScript(spec)},
		{"status-check", serviceStatusScript(spec)},
		{"port-check", servicePortScript(spec)},
		{"service-check", serviceHealthScript(spec)},
	}
	for _, s := range steps {
		if strings.TrimSpace(s.body) == "" {
			continue
		}
		report(s.name, "running", "start")
		if err := runBash(s.body); err != nil {
			report(s.name, "failed", err.Error())
			return fmt.Errorf("%s failed: %w", s.name, err)
		}
		report(s.name, "success", "ok")
	}
	return nil
}

func runServiceUpdateTemplate(spec *serviceInstallSpec, report func(string, string, string)) error {
	steps := []struct {
		name string
		body string
	}{
		{"write-config", serviceConfigScript(spec)},
		{"restart", serviceRestartScript(spec)},
		{"status-check", serviceStatusScript(spec)},
		{"port-check", servicePortScript(spec)},
		{"service-check", serviceHealthScript(spec)},
	}
	for _, s := range steps {
		if strings.TrimSpace(s.body) == "" {
			continue
		}
		report(s.name, "running", "start")
		if err := runBash(s.body); err != nil {
			report(s.name, "failed", err.Error())
			return fmt.Errorf("%s failed: %w", s.name, err)
		}
		report(s.name, "success", "ok")
	}
	return nil
}

func runBash(script string) error {
	cmd := exec.Command("bash", "-e", "-u", "-o", "pipefail", "-c", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func serviceInstallScript(spec *serviceInstallSpec) string {
	method := strParam(spec, "install_method", spec.InstallMethod)
	if method == "docker" || spec.Service == "kafka" {
		return `(command -v docker >/dev/null 2>&1) || { echo "docker is required for this install method" >&2; exit 1; }`
	}
	switch spec.Service {
	case "nginx":
		return pkgInstallShell("nginx")
	case "haproxy":
		return pkgInstallShell("haproxy")
	case "redis":
		return pkgInstallShell("redis-server")
	case "mysql":
		return pkgInstallShell("mysql-server")
	case "postgresql":
		return pkgInstallShell("postgresql postgresql-contrib")
	default:
		return `echo "unsupported service" >&2; exit 1`
	}
}

func pkgInstallShell(pkgs string) string {
	return fmt.Sprintf(`if command -v apt-get >/dev/null 2>&1; then
  apt-get update -y
  DEBIAN_FRONTEND=noninteractive apt-get install -y %s
elif command -v dnf >/dev/null 2>&1; then
  dnf install -y %s
else
  yum install -y %s
fi`, pkgs, pkgs, pkgs)
}

func serviceConfigScript(spec *serviceInstallSpec) string {
	switch spec.Service {
	case "nginx":
		return renderNginxServiceScript(spec)
	case "haproxy":
		return renderHAProxyServiceScript(spec)
	case "redis":
		return renderRedisServiceScript(spec)
	case "kafka":
		return renderKafkaDockerScript(spec)
	case "mysql":
		return renderMySQLServiceScript(spec)
	case "postgresql":
		return renderPostgresServiceScript(spec)
	default:
		return ""
	}
}

func serviceStartScript(spec *serviceInstallSpec) string {
	if strParam(spec, "install_method", spec.InstallMethod) == "docker" || spec.Service == "kafka" {
		return ""
	}
	unit := spec.Service
	if unit == "redis" {
		return `systemctl enable redis-server || systemctl enable redis
systemctl restart redis-server || systemctl restart redis`
	}
	if unit == "mysql" {
		return `systemctl enable mysql || systemctl enable mysqld
systemctl restart mysql || systemctl restart mysqld`
	}
	if unit == "postgresql" {
		return `systemctl enable postgresql
systemctl restart postgresql`
	}
	return fmt.Sprintf("systemctl enable %s\nsystemctl restart %s", unit, unit)
}

func serviceRestartScript(spec *serviceInstallSpec) string {
	if strParam(spec, "install_method", spec.InstallMethod) == "docker" || spec.Service == "kafka" {
		return fmt.Sprintf("docker restart %s", shellQuote(spec.Service))
	}
	unit := spec.Service
	if unit == "redis" {
		return "systemctl restart redis-server || systemctl restart redis"
	}
	if unit == "mysql" {
		return "systemctl restart mysql || systemctl restart mysqld"
	}
	return fmt.Sprintf("systemctl restart %s", unit)
}

func serviceStatusScript(spec *serviceInstallSpec) string {
	if strParam(spec, "install_method", spec.InstallMethod) == "docker" || spec.Service == "kafka" {
		return fmt.Sprintf("docker ps --filter name=%s --filter status=running | grep %s", shellQuote(spec.Service), shellQuote(spec.Service))
	}
	unit := spec.Service
	if unit == "redis" {
		return "systemctl is-active redis-server || systemctl is-active redis"
	}
	if unit == "mysql" {
		return "systemctl is-active mysql || systemctl is-active mysqld"
	}
	return fmt.Sprintf("systemctl is-active %s", unit)
}

func servicePortScript(spec *serviceInstallSpec) string {
	port := intParam(spec, portKey(spec.Service), defaultPort(spec.Service))
	return fmt.Sprintf(`PORT=%d
if command -v ss >/dev/null 2>&1; then
  ss -lntpn | awk '{print $4}' | grep -E '(^|:)'$PORT'$' >/dev/null
elif command -v netstat >/dev/null 2>&1; then
  netstat -lntp 2>/dev/null | awk '{print $4}' | grep -E '(^|:)'$PORT'$' >/dev/null
else
  echo "neither ss nor netstat is available for port check" >&2
  exit 1
fi`, port)
}

func serviceHealthScript(spec *serviceInstallSpec) string {
	switch spec.Service {
	case "nginx":
		return "nginx -t || docker exec nginx nginx -t"
	case "haproxy":
		return "haproxy -c -f /etc/haproxy/haproxy.cfg || docker exec haproxy haproxy -c -f /usr/local/etc/haproxy/haproxy.cfg"
	case "redis":
		pass := strParam(spec, "requirepass", "")
		if pass != "" {
			return fmt.Sprintf("redis-cli -p %d -a %s PING", intParam(spec, "port", 6379), shellQuote(pass))
		}
		return fmt.Sprintf("redis-cli -p %d PING", intParam(spec, "port", 6379))
	case "mysql":
		return fmt.Sprintf("mysqladmin ping -uroot -p%s -h127.0.0.1 -P%d", shellQuote(strParam(spec, "root_password", "changeme")), intParam(spec, "port", 3306))
	case "postgresql":
		return fmt.Sprintf("pg_isready -h 127.0.0.1 -p %d", intParam(spec, "port", 5432))
	case "kafka":
		return "docker exec kafka bash -lc 'kafka-broker-api-versions.sh --bootstrap-server 127.0.0.1:9092 >/dev/null'"
	default:
		return ""
	}
}

func nginxUninstallScript(method string, purgePackage bool) string {
	if method == "docker" {
		return `if command -v docker >/dev/null 2>&1; then
  docker rm -f nginx 2>/dev/null || true
fi
rm -f /etc/nginx/nginx.conf.ai-sre`
	}
	if purgePackage {
		return `rm -f /etc/nginx/conf.d/ai-sre-service.conf
if command -v systemctl >/dev/null 2>&1; then
  systemctl disable --now nginx 2>/dev/null || true
fi
if command -v apt-get >/dev/null 2>&1; then
  DEBIAN_FRONTEND=noninteractive apt-get purge -y nginx nginx-common || true
  DEBIAN_FRONTEND=noninteractive apt-get autoremove -y || true
elif command -v dnf >/dev/null 2>&1; then
  dnf remove -y nginx || true
elif command -v yum >/dev/null 2>&1; then
  yum remove -y nginx || true
else
  echo "no supported package manager found; ai-sre config removed but nginx package not purged" >&2
fi`
	}
	return `rm -f /etc/nginx/conf.d/ai-sre-service.conf
if command -v nginx >/dev/null 2>&1; then
  if nginx -t; then
    if command -v systemctl >/dev/null 2>&1 && systemctl list-unit-files nginx.service >/dev/null 2>&1; then
      systemctl reload nginx 2>/dev/null || systemctl restart nginx 2>/dev/null || true
    fi
  else
    echo "nginx config test failed after removing ai-sre config; please inspect /etc/nginx" >&2
    exit 1
  fi
fi`
}

func portKey(service string) string {
	if service == "haproxy" {
		return "frontend_port"
	}
	if service == "nginx" {
		return "http_port"
	}
	return "port"
}

func defaultPort(service string) int {
	switch service {
	case "nginx", "haproxy":
		return 80
	case "redis":
		return 6379
	case "mysql":
		return 3306
	case "postgresql":
		return 5432
	case "kafka":
		return 9092
	default:
		return 0
	}
}

func renderNginxServiceScript(spec *serviceInstallSpec) string {
	port := intParam(spec, "http_port", 80)
	root := strParam(spec, "docroot", "/var/www/html")
	mainConf := fmt.Sprintf(`worker_processes auto;
events { worker_connections %d; }
http {
  include mime.types;
  default_type application/octet-stream;
  sendfile on;
  keepalive_timeout 65;
  server {
    listen %d;
    server_name %s;
    root %s;
    index index.html index.htm;
    location / { try_files $uri $uri/ =404; }
  }
}`, intParam(spec, "worker_connections", 1024), port, strParam(spec, "server_name", "_"), root)
	siteConf := fmt.Sprintf(`server {
  listen %d;
  server_name %s;
  root %s;
  index index.html index.htm;
  location / { try_files $uri $uri/ =404; }
}`, port, strParam(spec, "server_name", "_"), root)
	if strParam(spec, "install_method", spec.InstallMethod) == "docker" {
		return fmt.Sprintf(`mkdir -p /etc/nginx %s
cat >/etc/nginx/nginx.conf <<'EOF'
%s
EOF
docker rm -f nginx 2>/dev/null || true
docker run -d --name nginx --restart=always -p %d:%d -v /etc/nginx/nginx.conf:/etc/nginx/nginx.conf:ro -v %s:%s:ro %s`, root, mainConf, port, port, root, root, dockerImage(spec, "nginx", "stable"))
	}
	return fmt.Sprintf(`mkdir -p %s /etc/nginx/conf.d
cat >/etc/nginx/conf.d/ai-sre-service.conf <<'EOF'
%s
EOF
nginx -t`, root, siteConf)
}

func renderHAProxyServiceScript(spec *serviceInstallSpec) string {
	port := intParam(spec, "frontend_port", 80)
	backends := strings.Split(strParam(spec, "backends", "127.0.0.1:8080"), "\n")
	var servers []string
	for i, b := range backends {
		b = strings.TrimSpace(b)
		if b != "" {
			servers = append(servers, fmt.Sprintf("  server srv%d %s check", i+1, b))
		}
	}
	conf := fmt.Sprintf(`global
  log /dev/log local0
  maxconn %d
defaults
  log global
  mode %s
  timeout connect %s
  timeout client %s
  timeout server %s
frontend web
  bind *:%d
  default_backend app
backend app
  balance %s
%s`, intParam(spec, "maxconn", 4096), strParam(spec, "mode", "http"), strParam(spec, "timeout_connect", "5s"), strParam(spec, "timeout_client", "30s"), strParam(spec, "timeout_server", "30s"), port, strParam(spec, "algorithm", "roundrobin"), strings.Join(servers, "\n"))
	if strParam(spec, "install_method", spec.InstallMethod) == "docker" {
		return fmt.Sprintf(`mkdir -p /etc/haproxy
cat >/etc/haproxy/haproxy.cfg <<'EOF'
%s
EOF
docker rm -f haproxy 2>/dev/null || true
docker run -d --name haproxy --restart=always -p %d:%d -v /etc/haproxy/haproxy.cfg:/usr/local/etc/haproxy/haproxy.cfg:ro %s`, conf, port, port, dockerImage(spec, "haproxy", "lts"))
	}
	return fmt.Sprintf(`cat >/etc/haproxy/haproxy.cfg <<'EOF'
%s
EOF
haproxy -c -f /etc/haproxy/haproxy.cfg`, conf)
}

func renderRedisServiceScript(spec *serviceInstallSpec) string {
	port := intParam(spec, "port", 6379)
	dir := strParam(spec, "dir", "/var/lib/redis")
	conf := fmt.Sprintf(`bind %s
protected-mode %s
port %d
dir %s
databases %d
maxmemory %s
maxmemory-policy %s
appendonly %s
%s`, strParam(spec, "bind", "0.0.0.0"), yesNo(boolParam(spec, "protected_mode", true)), port, dir, intParam(spec, "databases", 16), strParam(spec, "maxmemory", "512mb"), strParam(spec, "maxmemory_policy", "allkeys-lru"), yesNo(boolParam(spec, "appendonly", false)), requirepassLine(spec))
	if strParam(spec, "install_method", spec.InstallMethod) == "docker" {
		return fmt.Sprintf(`mkdir -p /etc/redis %s
cat >/etc/redis/redis.conf <<'EOF'
%s
EOF
docker rm -f redis 2>/dev/null || true
docker run -d --name redis --restart=always -p %d:%d -v /etc/redis/redis.conf:/usr/local/etc/redis/redis.conf:ro -v %s:%s %s redis-server /usr/local/etc/redis/redis.conf`, dir, conf, port, port, dir, dir, dockerImage(spec, "redis", "7"))
	}
	return fmt.Sprintf(`mkdir -p %s
cat >/etc/redis/redis.conf <<'EOF'
%s
EOF`, dir, conf)
}

func renderKafkaDockerScript(spec *serviceInstallSpec) string {
	port := intParam(spec, "port", 9092)
	return fmt.Sprintf(`docker rm -f kafka 2>/dev/null || true
docker run -d --name kafka --restart=always -p %d:9092 \
  -e KAFKA_BROKER_ID=%d \
  -e KAFKA_CFG_ZOOKEEPER_CONNECT=%s \
  -e KAFKA_CFG_LISTENERS=PLAINTEXT://:9092 \
  -e KAFKA_CFG_ADVERTISED_LISTENERS=PLAINTEXT://127.0.0.1:%d \
  -e KAFKA_CFG_NUM_PARTITIONS=%d \
  -e KAFKA_CFG_DEFAULT_REPLICATION_FACTOR=%d \
  -e KAFKA_CFG_LOG_RETENTION_HOURS=%d \
  -e ALLOW_PLAINTEXT_LISTENER=yes \
  -v kafka-data:/bitnami/kafka %s`, port, intParam(spec, "broker_id", 1), shellQuote(strParam(spec, "zookeeper", "localhost:2181")), port, intParam(spec, "num_partitions", 3), intParam(spec, "default_replication_factor", 1), intParam(spec, "log_retention_hours", 168), dockerImage(spec, "bitnami/kafka", "3.6"))
}

func renderMySQLServiceScript(spec *serviceInstallSpec) string {
	port := intParam(spec, "port", 3306)
	datadir := strParam(spec, "datadir", "/var/lib/mysql")
	cnf := fmt.Sprintf(`[mysqld]
port=%d
bind-address=%s
character-set-server=%s
collation-server=%s
max_connections=%d
innodb_buffer_pool_size=%s
%s`, port, strParam(spec, "bind_address", "0.0.0.0"), strParam(spec, "charset", "utf8mb4"), strParam(spec, "collation", "utf8mb4_0900_ai_ci"), intParam(spec, "max_connections", 500), strParam(spec, "innodb_buffer_pool_size", "512M"), skipNameResolveLine(spec))
	if strParam(spec, "install_method", spec.InstallMethod) == "docker" {
		return fmt.Sprintf(`mkdir -p /etc/mysql/conf.d %s
cat >/etc/mysql/conf.d/99-ai-sre.cnf <<'EOF'
%s
EOF
docker rm -f mysql 2>/dev/null || true
docker run -d --name mysql --restart=always -p %d:3306 -e MYSQL_ROOT_PASSWORD=%s -v %s:/var/lib/mysql -v /etc/mysql/conf.d/99-ai-sre.cnf:/etc/mysql/conf.d/99-ai-sre.cnf:ro %s`, datadir, cnf, port, shellQuote(strParam(spec, "root_password", "changeme")), datadir, dockerImage(spec, "mysql", "8.0"))
	}
	return fmt.Sprintf(`mkdir -p /etc/mysql/mysql.conf.d
cat >/etc/mysql/mysql.conf.d/99-ai-sre.cnf <<'EOF'
%s
EOF`, cnf)
}

func renderPostgresServiceScript(spec *serviceInstallSpec) string {
	port := intParam(spec, "port", 5432)
	datadir := strParam(spec, "datadir", "/var/lib/postgresql/data")
	conf := fmt.Sprintf(`listen_addresses = '%s'
port = %d
max_connections = %d
shared_buffers = '%s'
work_mem = '%s'
wal_level = %s
log_min_duration_statement = %d`, strParam(spec, "listen_addresses", "*"), port, intParam(spec, "max_connections", 200), strParam(spec, "shared_buffers", "512MB"), strParam(spec, "work_mem", "8MB"), strParam(spec, "wal_level", "replica"), intParam(spec, "log_min_duration_statement", 1000))
	if strParam(spec, "install_method", spec.InstallMethod) == "docker" {
		return fmt.Sprintf(`mkdir -p /etc/postgresql %s
cat >/etc/postgresql/postgresql.conf <<'EOF'
%s
EOF
docker rm -f postgres 2>/dev/null || true
docker run -d --name postgres --restart=always -p %d:5432 -e POSTGRES_PASSWORD=%s -e PGDATA=%s -v %s:%s -v /etc/postgresql/postgresql.conf:/etc/postgresql/postgresql.conf:ro %s -c config_file=/etc/postgresql/postgresql.conf`, datadir, conf, port, shellQuote(strParam(spec, "password", "changeme")), datadir, datadir, datadir, dockerImage(spec, "postgres", "16"))
	}
	return fmt.Sprintf(`PG_CONF="$(sudo -u postgres psql -tAc 'show config_file' 2>/dev/null || true)"
if [ -n "$PG_CONF" ]; then
  cat >>"$PG_CONF" <<'EOF'
%s
EOF
fi`, conf)
}

func strParam(spec *serviceInstallSpec, key, def string) string {
	if spec.Params == nil {
		return def
	}
	if v, ok := spec.Params[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return def
}

func intParam(spec *serviceInstallSpec, key string, def int) int {
	v := strParam(spec, key, "")
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return def
	}
	return i
}

func boolParam(spec *serviceInstallSpec, key string, def bool) bool {
	if spec.Params == nil {
		return def
	}
	v, ok := spec.Params[key]
	if !ok || v == nil {
		return def
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true" || t == "yes" || t == "1"
	default:
		return fmt.Sprint(t) == "true"
	}
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func requirepassLine(spec *serviceInstallSpec) string {
	if p := strParam(spec, "requirepass", ""); p != "" {
		return "requirepass " + p
	}
	return ""
}

func skipNameResolveLine(spec *serviceInstallSpec) string {
	if boolParam(spec, "skip_name_resolve", true) {
		return "skip-name-resolve"
	}
	return ""
}

func dockerImage(spec *serviceInstallSpec, image, fallback string) string {
	v := strings.TrimSpace(spec.Version)
	if v == "" {
		v = strParam(spec, "version", fallback)
	}
	if v == "" {
		v = fallback
	}
	return image + ":" + v
}

func fillServiceUpdateOptionsFromState(service string, opts serviceUpdateOptions) serviceUpdateOptions {
	if strings.TrimSpace(opts.FromURL) != "" {
		return opts
	}
	st, err := loadServiceDeploymentState(service)
	if err != nil {
		return opts
	}
	if strings.TrimSpace(opts.APIURL) == "" {
		opts.APIURL = st.APIURL
	}
	if strings.TrimSpace(opts.DeployID) == "" {
		opts.DeployID = st.DeployID
	}
	if strings.TrimSpace(opts.Token) == "" {
		opts.Token = st.Token
	}
	return opts
}

func saveServiceDeploymentState(service, apiURL, deployID, token string) error {
	service = strings.TrimSpace(strings.ToLower(service))
	if service == "" || apiURL == "" || deployID == "" || token == "" {
		return nil
	}
	path, err := serviceDeploymentStatePath(service)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	state := serviceDeploymentState{
		Service:   service,
		APIURL:    strings.TrimRight(apiURL, "/"),
		DeployID:  deployID,
		Token:     token,
		UpdatedAt: time.Now(),
	}
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}

func loadServiceDeploymentState(service string) (*serviceDeploymentState, error) {
	path, err := serviceDeploymentStatePath(service)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var state serviceDeploymentState
	if err := json.Unmarshal(b, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func removeServiceDeploymentState(service string) error {
	path, err := serviceDeploymentStatePath(service)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func serviceDeploymentStatePath(service string) (string, error) {
	service = strings.TrimSpace(strings.ToLower(service))
	if service == "" {
		return "", fmt.Errorf("service is required")
	}
	if os.Geteuid() == 0 {
		return filepath.Join("/etc/ai-sre/service-deployments", service+".json"), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ai-sre", "service-deployments", service+".json"), nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
