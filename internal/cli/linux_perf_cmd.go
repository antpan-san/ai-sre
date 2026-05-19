package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func probeLinuxCmd() *cobra.Command {
	var opts LinuxPerfOptions
	cmd := &cobra.Command{
		Use:   "linux",
		Short: "Linux 主机性能只读快采（CPU/负载/内存/磁盘/进程）",
		Long:  "只读采集当前 Linux 主机性能证据，不调用 AI。默认采样窗口 10s。",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			normalizeLinuxPerfOptions(&opts)
			if err := ValidateLinuxPerfOptions(opts); err != nil {
				return err
			}
			report := CollectLinuxPerf(opts)
			if opts.JSON || strings.EqualFold(outputFormat, "json") {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(report)
			}
			fmt.Fprint(cmd.OutOrStdout(), formatLinuxProbeText(report))
			return nil
		},
	}
	cmd.Flags().DurationVar(&opts.Duration, "duration", linuxPerfDefaultDuration, "采样窗口 (3s-60s)")
	cmd.Flags().IntVar(&opts.TopN, "top", linuxPerfDefaultTop, "资源排序 Top N (5-30)")
	cmd.Flags().IntVar(&opts.PID, "pid", 0, "可选：针对单个进程做更深分析")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "输出机器可读 JSON")
	return cmd
}

func finishLinuxCheckEvidence(topic string, ctx map[string]string) error {
	if normalizeCheckTopic(topic) != "linux" {
		return nil
	}
	if strings.TrimSpace(ctx["linux_perf_probe_json"]) != "" {
		return nil
	}
	body, _, err := collectLinuxProbeJSON(nil, ctx)
	if err != nil {
		return err
	}
	if body != "" {
		ctx["linux_perf_probe_json"] = body
	}
	return nil
}

func gatherLinuxEvidence(ctx context.Context, flags map[string]string, out map[string]string) {
	body, _, err := collectLinuxProbeJSON(ctx, flags)
	if err != nil {
		return
	}
	if body != "" {
		out["linux_perf_probe_json"] = body
	}
}

// parseLinuxPIDFlag validates --pid for probe linux.
func parseLinuxPIDFlag(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("--pid 须为数字")
	}
	if n < 0 {
		return 0, fmt.Errorf("--pid 须为非负整数")
	}
	return n, nil
}

func isLinuxPerformanceTopic(topic string) bool {
	return normalizeCheckTopic(topic) == "linux"
}
