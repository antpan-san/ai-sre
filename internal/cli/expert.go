package cli

import "github.com/spf13/cobra"

// expertCmd exposes advanced capabilities for power users.
func expertCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "expert",
		Short: "高级能力：只读采集、技能包、AI 问答与 Runbook",
		Long: `熟练用户入口；普通排障请优先使用 check。

  expert probe   只读采集，不调用 AI
  expert skills  技能包状态、反馈与精炼
  expert ask     AI 问答
  expert runbook Runbook 生成`,
	}
	cmd.AddCommand(expertProbeCmd(), skillsCmd(), askCmd(), runbookCmd())
	return cmd
}

func expertProbeCmd() *cobra.Command {
	cmd := probeCmd()
	cmd.Use = "probe"
	cmd.Short = "只读快采（无 AI）"
	return cmd
}
