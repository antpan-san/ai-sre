package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func opsUninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "卸载 ai-sre 平台安装并有本机状态证明的环境",
		Long: `默认保留数据目录与日志；删除数据须显式 --purge-data（高风险）。

示例:
  sudo ai-sre ops uninstall k8s
  sudo ai-sre ops uninstall nginx
  sudo ai-sre ops uninstall elasticsearch`,
	}
	cmd.AddCommand(
		opsUninstallK8sCmd(),
		opsUninstallTopicCmd("nginx", "Nginx"),
		opsUninstallTopicCmd("elasticsearch", "Elasticsearch"),
	)
	return cmd
}

func opsUninstallK8sCmd() *cobra.Command {
	var refOverride, workdir string
	var force, dryRun, yes bool
	cmd := &cobra.Command{
		Use:   "k8s",
		Short: "卸载 ofpk8s1 编排的 K8s 集群（优先 last-bundle）",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireOpsRoot("K8s 卸载"); err != nil {
				return err
			}
			if dryRun {
				fmt.Println("将执行: runUninstallK8s（pre_cleanup 全节点）")
				fmt.Printf("  last-bundle: %s\n", K8sLastBundlePath)
				fmt.Printf("  install-ref: %s\n", loadK8sInstallRef())
				return nil
			}
			if err := requireOpsMutationConfirm(yes, "K8s 卸载"); err != nil {
				return err
			}
			return runUninstallK8s(refOverride, workdir, force)
		},
	}
	cmd.Flags().StringVar(&refOverride, "ref", "", "显式 ofpk8s1 安装引用")
	cmd.Flags().StringVar(&workdir, "workdir", "", "离线包解压根目录")
	cmd.Flags().BoolVar(&force, "force", false, "仅使用本机 last-bundle，不尝试拉取邀请 zip")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "只展示将执行的操作")
	cmd.Flags().BoolVar(&yes, "yes", false, "非 TTY 确认")
	return cmd
}

func opsUninstallTopicCmd(topic, label string) *cobra.Command {
	var dryRun, yes, force bool
	var purgePackage, purgeData bool
	cmd := &cobra.Command{
		Use:   topic,
		Short: "卸载 " + label,
		RunE: func(c *cobra.Command, args []string) error {
			if err := requireOpsRoot(label + " 卸载"); err != nil {
				return err
			}
			if dryRun {
				st, err := loadServiceDeploymentState(topic)
				if err != nil {
					return fmt.Errorf("dry-run: 无 ai-sre 管理状态 (%v)", err)
				}
				fmt.Printf("将卸载 %s（deploy_id=%s）\n", topic, st.DeployID)
				return nil
			}
			if err := requireOpsMutationConfirm(yes, label+" 卸载"); err != nil {
				return err
			}
			switch topic {
			case "nginx":
				return runNginxUninstall(c, nginxUninstallOptions{PurgePackage: purgePackage, Force: force})
			case "elasticsearch":
				return runElasticsearchUninstall(c, elasticsearchUninstallOptions{PurgePackage: purgePackage, PurgeData: purgeData, Force: force})
			default:
				return fmt.Errorf("unsupported topic %q", topic)
			}
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "只检查本机状态证明")
	cmd.Flags().BoolVar(&yes, "yes", false, "非 TTY 确认")
	cmd.Flags().BoolVar(&force, "force", false, "跳过服务端 spec 校验（仍须本机状态文件）")
	cmd.Flags().BoolVar(&purgePackage, "purge-package", false, "卸载系统包（高风险）")
	cmd.Flags().BoolVar(&purgeData, "purge-data", false, "删除数据目录（高风险，仅 elasticsearch）")
	return cmd
}

func serviceUninstallCmd() *cobra.Command {
	var dryRun, yes, force bool
	var purgePackage, purgeData bool
	var purgeToken string
	cmd := &cobra.Command{
		Use:   "uninstall [service]",
		Short: "卸载基础服务（须存在 ai-sre 平台安装状态证明）",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			svc := strings.TrimSpace(strings.ToLower(args[0]))
			if err := requireOpsRoot("服务卸载"); err != nil {
				return err
			}
			if purgeData && !yes {
				return fmt.Errorf("--purge-data 须与 --yes 一并使用")
			}
			if dryRun {
				if _, err := loadServiceDeploymentState(svc); err != nil {
					return fmt.Errorf("dry-run: 无 ai-sre 管理状态 (%w)", err)
				}
				fmt.Printf("将卸载服务 %s（停止服务，%s）\n", svc, map[bool]string{true: "删除数据", false: "保留数据"}[purgeData])
				return nil
			}
			if err := requireOpsMutationConfirm(yes, "服务卸载"); err != nil {
				return err
			}
			opts := managedServiceUninstallOptions{
				PurgePackage: purgePackage,
				PurgeData:    purgeData,
				PurgeToken:   purgeToken,
				Force:        force,
			}
			switch svc {
			case "nginx":
				return runNginxUninstall(c, nginxUninstallOptions{PurgePackage: purgePackage, Force: force})
			case "elasticsearch":
				return runElasticsearchUninstall(c, elasticsearchUninstallOptions{PurgePackage: purgePackage, PurgeData: purgeData, Force: force})
			case "redis", "mysql", "postgresql", "kafka", "haproxy":
				return runManagedServiceUninstall(c, svc, opts)
			default:
				return fmt.Errorf("未知服务 %q", svc)
			}
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "只检查状态证明")
	cmd.Flags().BoolVar(&yes, "yes", false, "非 TTY 确认")
	cmd.Flags().BoolVar(&force, "force", false, "强制本地卸载")
	cmd.Flags().BoolVar(&purgePackage, "purge-package", false, "卸载系统包（高风险）")
	cmd.Flags().BoolVar(&purgeData, "purge-data", false, "删除数据目录（须控制台 --purge-token）")
	cmd.Flags().StringVar(&purgeToken, "purge-token", "", "控制台审批的数据清理 token")
	return cmd
}
