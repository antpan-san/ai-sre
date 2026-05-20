package cli

import "github.com/spf13/cobra"

// opsCmd groups deployment, install, change, and job commands.
func opsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ops",
		Short: "部署、安装、变更与作业",
		Long: `运维变更统一入口：K8s 离线安装、服务部署、节点调优、批量作业等。

示例:
  sudo ` + progName + ` ops k8s install 'ofpk8s1.…'
  sudo ` + progName + ` ops k8s cleanup 'ofpk8s1.…'
  sudo ` + progName + ` ops k8s recover
  sudo ` + progName + ` ops uninstall k8s
  sudo ` + progName + ` ops service install --api-url … --deploy-id … --token …
  sudo ` + progName + ` ops node tune time-sync
  ` + progName + ` ops job run --machines … -c 'uptime'`,
	}
	cmd.AddCommand(
		k8sCmd(),
		serviceCmd(),
		opsUninstallCmd(),
		opsNginxCmd(),
		opsElasticsearchCmd(),
		nodeCmd(),
		jobCmd(),
	)
	return cmd
}

func opsNginxCmd() *cobra.Command {
	cmd := nginxCmd()
	cmd.Use = "nginx"
	cmd.Short = "Nginx 生命周期（update / uninstall）"
	for _, sub := range cmd.Commands() {
		if sub.Name() == "diagnose" {
			sub.Hidden = true
		}
	}
	return cmd
}

func opsElasticsearchCmd() *cobra.Command {
	cmd := elasticsearchCmd()
	cmd.Use = "elasticsearch"
	cmd.Short = "Elasticsearch 生命周期（update / uninstall）"
	for _, sub := range cmd.Commands() {
		if sub.Name() == "diagnose" {
			sub.Hidden = true
		}
	}
	return cmd
}
