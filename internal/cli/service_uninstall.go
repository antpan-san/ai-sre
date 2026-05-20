package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type managedServiceUninstallOptions struct {
	PurgePackage bool
	PurgeData    bool
	PurgeToken   string
	Force        bool
}

func runManagedServiceUninstall(cmd *cobra.Command, service string, opts managedServiceUninstallOptions) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("卸载 %s 需 root 权限", service)
	}
	service = strings.TrimSpace(strings.ToLower(service))
	state, err := loadServiceDeploymentState(service)
	if err != nil {
		if !opts.Force {
			return fmt.Errorf("拒绝卸载 %s：无 ai-sre 管理状态证明（%w）", service, err)
		}
	}
	spec := &serviceInstallSpec{Service: service, InstallMethod: "package", Params: map[string]interface{}{}}
	if state != nil && state.APIURL != "" && state.DeployID != "" && state.Token != "" {
		if fetched, fetchErr := fetchServiceSpec(fmt.Sprintf("%s/api/service-deploy/deployments/%s/spec?token=%s",
			strings.TrimRight(state.APIURL, "/"), url.PathEscape(state.DeployID), url.QueryEscape(state.Token))); fetchErr == nil {
			spec = fetched
		}
		_ = postServiceFinish(state.APIURL, state.DeployID, state.Token, "uninstalling", service+" uninstall started")
	}
	method := strParam(spec, "install_method", spec.InstallMethod)
	if method == "" {
		method = "package"
	}
	if opts.PurgeData {
		if err := requirePurgeDataApproval(service, state, opts.PurgeToken); err != nil {
			return err
		}
	}
	fmt.Fprintf(cmd.OutOrStdout(), "[uninstall] %s (%s) start\n", service, method)
	if err := runBash(managedServiceUninstallScript(service, method, opts.PurgePackage, opts.PurgeData)); err != nil {
		if state != nil {
			_ = postServiceFinish(state.APIURL, state.DeployID, state.Token, "uninstall_failed", err.Error())
		}
		return fmt.Errorf("%s uninstall failed: %w", service, err)
	}
	_ = removeServiceDeploymentState(service)
	if state != nil && state.APIURL != "" && state.DeployID != "" && state.Token != "" {
		msg := service + " uninstalled (service stopped, data retained)"
		if opts.PurgeData {
			msg = service + " uninstalled with data purge"
		}
		_ = postServiceFinish(state.APIURL, state.DeployID, state.Token, "uninstalled", msg)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "[uninstall] %s success (%s)\n", service, uninstallOutcomeLabel(opts.PurgeData))
	return nil
}

func uninstallOutcomeLabel(purgeData bool) string {
	if purgeData {
		return "data purged"
	}
	return "data retained"
}

func requirePurgeDataApproval(service string, state *serviceDeploymentState, purgeToken string) error {
	purgeToken = strings.TrimSpace(purgeToken)
	if purgeToken == "" {
		return fmt.Errorf("删除 %s 数据目录为高风险操作，须在控制台审批后使用 --purge-data --purge-token", service)
	}
	if state == nil || state.APIURL == "" || state.DeployID == "" || state.Token == "" {
		return fmt.Errorf("删除数据须存在 ai-sre 平台部署状态以校验审批 token")
	}
	base := strings.TrimRight(state.APIURL, "/")
	verifyURL := fmt.Sprintf("%s/api/service-deploy/deployments/%s/purge-token/verify?token=%s&purge_token=%s",
		base, url.PathEscape(state.DeployID), url.QueryEscape(state.Token), url.QueryEscape(purgeToken))
	resp, err := http.Get(verifyURL)
	if err != nil {
		return fmt.Errorf("校验 purge token 失败: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("purge token 无效或已过期: HTTP %d", resp.StatusCode)
	}
	var env struct {
		Data struct {
			OK bool `json:"ok"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &env) == nil && env.Data.OK {
		return nil
	}
	return fmt.Errorf("purge token 无效或已过期")
}

func managedServiceUninstallScript(service, method string, purgePackage, purgeData bool) string {
	service = strings.TrimSpace(strings.ToLower(service))
	if method == "docker" {
		name := service
		if service == "postgresql" {
			name = "postgres"
		}
		return fmt.Sprintf(`set -euo pipefail
docker rm -f %s 2>/dev/null || true
exit 0`, name)
	}
	var units []string
	switch service {
	case "redis":
		units = []string{"redis-server", "redis"}
	case "mysql":
		units = []string{"mysql", "mysqld"}
	case "postgresql":
		units = []string{"postgresql"}
	case "kafka":
		units = []string{"kafka"}
	case "haproxy":
		units = []string{"haproxy"}
	default:
		units = []string{service}
	}
	var b strings.Builder
	b.WriteString("set -euo pipefail\n")
	for _, u := range units {
		fmt.Fprintf(&b, "systemctl disable --now %s 2>/dev/null || true\n", u)
	}
	if purgePackage {
		b.WriteString(`if command -v apt-get >/dev/null 2>&1; then
  apt-get remove -y --purge `)
		switch service {
		case "redis":
			b.WriteString("redis-server redis-tools")
		case "mysql":
			b.WriteString("mysql-server mysql-client")
		case "postgresql":
			b.WriteString("postgresql")
		case "kafka":
			b.WriteString("kafka")
		case "haproxy":
			b.WriteString("haproxy")
		default:
			b.WriteString(service)
		}
		b.WriteString(` 2>/dev/null || true
fi
`)
	}
	if purgeData {
		switch service {
		case "redis":
			b.WriteString("rm -rf /var/lib/redis /var/lib/redis-server 2>/dev/null || true\n")
		case "mysql":
			b.WriteString("rm -rf /var/lib/mysql 2>/dev/null || true\n")
		case "postgresql":
			b.WriteString("rm -rf /var/lib/postgresql 2>/dev/null || true\n")
		case "kafka":
			b.WriteString("rm -rf /var/lib/kafka /opt/kafka/logs 2>/dev/null || true\n")
		case "haproxy":
			b.WriteString("rm -rf /var/lib/haproxy 2>/dev/null || true\n")
		}
	}
	b.WriteString("exit 0\n")
	return b.String()
}
