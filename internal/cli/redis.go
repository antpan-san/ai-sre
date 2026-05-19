package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func redisCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "redis",
		Short: "Redis 诊断",
	}
	cmd.AddCommand(redisDiagnoseCmd())
	return cmd
}

func probeRedisCmd() *cobra.Command {
	var opts redisProbeOptions
	cmd := &cobra.Command{
		Use:   "redis [addr]",
		Short: "Redis 只读全面快采（INFO/SLOWLOG/LATENCY/CLUSTER 等）",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Address = "127.0.0.1:6379"
			if len(args) >= 1 {
				opts.Address = normalizeCheckTargetValue("redis", strings.TrimSpace(args[0]))
			}
			if opts.Timeout <= 0 {
				opts.Timeout = 8 * time.Second
			}
			report := CollectRedisProbe(opts)
			if report.AuthRequired && strings.TrimSpace(opts.Password) == "" {
				if !isStdinTTY() {
					enc := json.NewEncoder(cmd.OutOrStdout())
					enc.SetIndent("", "  ")
					return enc.Encode(report)
				}
				pw, err := promptRedisPassword(opts.Address)
				if err != nil {
					return err
				}
				opts.Password = pw
				report = CollectRedisProbe(opts)
			}
			if opts.JSON || strings.EqualFold(outputFormat, "json") {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(report)
			}
			fmt.Fprint(cmd.OutOrStdout(), formatRedisProbeText(report))
			return nil
		},
	}
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 8*time.Second, "Redis 连接与命令超时")
	cmd.Flags().StringVar(&opts.Password, "password", "", "Redis AUTH 密码（可选；check 场景优先 TTY 交互输入）")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "输出机器可读 JSON")
	return cmd
}

// redisDiagnoseCmd is a deprecated alias; implementation delegates to probe collector.
func redisDiagnoseCmd() *cobra.Command {
	c := probeRedisCmd()
	c.Use = "diagnose <addr>"
	c.Short = "（已弃用）请改用 probe redis"
	c.Deprecated = "use \"probe redis\" instead"
	c.Args = cobra.ExactArgs(1)
	return c
}
