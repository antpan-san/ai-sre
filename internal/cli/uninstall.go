package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func uninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "卸载本机已安装的环境（当前仅 Kubernetes 离线装集群）",
		Long: `自适配备份：在曾执行过 k8s install 或引导脚本 的机器上，会把安装引用写入
` + k8sInstallRefSystemPath() + `（root）或 ~/.config/ai-sre/` + k8sInstallRefFile + `。
执行 uninstall k8s 时自动读取并调用与 k8s cleanup 相同的全节点 pre_cleanup。

也可在任意时刻设置: export OPSFLEET_K8S_INSTALL_REF='ofpk8s1.…'`,
	}
	cmd.AddCommand(uninstallK8sCmd())
	return cmd
}

func uninstallK8sCmd() *cobra.Command {
	var refOverride string
	cmd := &cobra.Command{
		Use:   "k8s",
		Short: "卸载此前通过 ofpk8s1 装上的集群（全节点 pre_cleanup，无需再抄命令）",
		RunE: func(cmd *cobra.Command, args []string) error {
			ref := strings.TrimSpace(refOverride)
			if ref == "" {
				ref = loadK8sInstallRef()
			}
			if ref == "" {
				return fmt.Errorf(
					"未找到本机记录的安装引用。\n"+
						"请先在同一台控制机执行过: sudo %s k8s install 'ofpk8s1.…' 或 curl 引导脚本（会写入 %s），\n"+
						"或显式: sudo %s k8s cleanup 'ofpk8s1.…' / export OPSFLEET_K8S_INSTALL_REF='ofpk8s1…' / sudo %s uninstall k8s --ref 'ofpk8s1…'",
					progName, k8sInstallRefSystemPath(), progName, progName,
				)
			}
			if !strings.HasPrefix(ref, installRefPrefixV1) {
				return fmt.Errorf("安装引用须以 %s 开头", installRefPrefixV1)
			}
			if os.Geteuid() != 0 {
				return errors.New("卸载集群需 root 权限，请使用: sudo " + progName + " uninstall k8s")
			}
			return runCleanupFromInviteRef(ref)
		},
	}
	cmd.Flags().StringVar(&refOverride, "ref", "", "显式指定 ofpk8s1… 安装引用（覆盖状态文件与环境变量）")
	return cmd
}
