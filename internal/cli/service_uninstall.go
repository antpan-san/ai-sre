package cli

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type managedServiceUninstallOptions struct {
	PurgePackage bool
	PurgeData    bool
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
		return fmt.Errorf("删除 %s 数据目录为高风险操作，当前版本请人工审批后执行", service)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "[uninstall] %s (%s) start\n", service, method)
	if err := runBash(managedServiceUninstallScript(service, method, opts.PurgePackage)); err != nil {
		if state != nil {
			_ = postServiceFinish(state.APIURL, state.DeployID, state.Token, "uninstall_failed", err.Error())
		}
		return fmt.Errorf("%s uninstall failed: %w", service, err)
	}
	_ = removeServiceDeploymentState(service)
	if state != nil && state.APIURL != "" && state.DeployID != "" && state.Token != "" {
		_ = postServiceFinish(state.APIURL, state.DeployID, state.Token, "uninstalled", service+" uninstalled (service stopped, data retained)")
	}
	fmt.Fprintf(cmd.OutOrStdout(), "[uninstall] %s success (data retained)\n", service)
	return nil
}

func managedServiceUninstallScript(service, method string, purgePackage bool) string {
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
	b.WriteString("exit 0\n")
	return b.String()
}
