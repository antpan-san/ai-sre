package cli

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
)

func uninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "卸载本机已安装的环境（当前仅 Kubernetes 离线装集群）",
		Long: `优先使用本机保存的「上次部署」离线包副本 ` + K8sLastBundlePath + `（由新版 install.sh / 引导脚本在预检后自动同步），
执行与安装时相同的 inventory 清单做全节点 pre_cleanup；不依赖控制台资源 id、不要求邀请链接仍有效。

若本机尚无该目录（旧版安装），可显式指定解压根 --workdir，或临时使用 ofpk8s1 拉 zip（--ref / 环境变量）。

另: 安装引用仍会写入 ` + k8sInstallRefSystemPath() + `，仅作兼容旧流程。`,
	}
	cmd.AddCommand(uninstallK8sCmd())
	return cmd
}

func uninstallK8sCmd() *cobra.Command {
	var refOverride, workdir string
	var force bool
	cmd := &cobra.Command{
		Use:   "k8s",
		Short: "卸载 ofpk8s1 编排的集群（执行前与其它子命令相同，可自连 OpsFleet 升级 ai-sre）",
		RunE: func(cmd *cobra.Command, args []string) error {
			if os.Geteuid() != 0 {
				return errors.New("卸载集群需 root 权限，请使用: sudo " + progName + " uninstall k8s")
			}
			return runUninstallK8s(refOverride, workdir, force)
		},
	}
	cmd.Flags().StringVar(&refOverride, "ref", "", "显式指定 ofpk8s1… 安装引用（覆盖状态文件与 OPSFLEET_K8S_INSTALL_REF）")
	cmd.Flags().StringVar(&workdir, "workdir", "", "已解压的离线包根目录（含 ansible-agent 与 inventory；不拉取邀请 zip）")
	cmd.Flags().BoolVar(&force, "force", false, "仅使用本机副本（"+K8sLastBundlePath+" 等），不尝试 ofpk8s1 拉包")
	return cmd
}
