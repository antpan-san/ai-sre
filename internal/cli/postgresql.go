package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
)

type postgresqlDiagnoseOptions struct {
	DSN     string
	Timeout time.Duration
	JSON    bool
}

type postgresqlDiagnoseReport struct {
	DSN              string   `json:"dsn"`
	Version          string   `json:"version,omitempty"`
	ActiveConns      int64    `json:"active_connections,omitempty"`
	IdleConns        int64    `json:"idle_connections,omitempty"`
	MaxConnections   int64    `json:"max_connections,omitempty"`
	Deadlocks        int64    `json:"deadlocks,omitempty"`
	CacheHitRatioPct float64  `json:"cache_hit_ratio_pct,omitempty"`
	Findings         []string `json:"findings,omitempty"`
	Errors           []string `json:"errors,omitempty"`
}

func postgresqlCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "postgresql",
		Aliases: []string{"postgres", "pg"},
		Short:   "PostgreSQL 极简快诊",
	}
	cmd.AddCommand(postgresqlDiagnoseCmd())
	return cmd
}

func postgresqlDiagnoseCmd() *cobra.Command {
	var opts postgresqlDiagnoseOptions
	cmd := &cobra.Command{
		Use:   "diagnose <dsn>",
		Short: "只读连接 PostgreSQL 采集关键指标并给出优先排查建议",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.DSN = strings.TrimSpace(args[0])
			if opts.Timeout <= 0 {
				opts.Timeout = 5 * time.Second
			}
			report := runPostgreSQLDiagnose(cmd.Context(), opts)
			if opts.JSON || strings.EqualFold(outputFormat, "json") {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(report)
			}
			fmt.Fprint(cmd.OutOrStdout(), formatPostgreSQLDiagnoseText(report))
			return nil
		},
	}
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 5*time.Second, "PostgreSQL 连接与查询超时")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "输出机器可读 JSON")
	return cmd
}

func runPostgreSQLDiagnose(parent context.Context, opts postgresqlDiagnoseOptions) *postgresqlDiagnoseReport {
	report := &postgresqlDiagnoseReport{DSN: maskPostgreSQLDSN(opts.DSN)}
	db, err := sql.Open("postgres", opts.DSN)
	if err != nil {
		report.Errors = append(report.Errors, "连接初始化失败: "+err.Error())
		return report
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(parent, opts.Timeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		report.Errors = append(report.Errors, "连接失败: "+err.Error())
		return report
	}
	_ = db.QueryRowContext(ctx, "SELECT version()").Scan(&report.Version)
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pg_stat_activity WHERE state = 'active'").Scan(&report.ActiveConns)
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pg_stat_activity WHERE state = 'idle'").Scan(&report.IdleConns)
	_ = db.QueryRowContext(ctx, "SELECT setting::bigint FROM pg_settings WHERE name = 'max_connections'").Scan(&report.MaxConnections)
	_ = db.QueryRowContext(ctx, `
SELECT COALESCE(SUM(deadlocks), 0)
FROM pg_stat_database
WHERE datname = current_database()`).Scan(&report.Deadlocks)
	var blksHit, blksRead int64
	_ = db.QueryRowContext(ctx, `
SELECT COALESCE(SUM(blks_hit), 0), COALESCE(SUM(blks_read), 0)
FROM pg_stat_database
WHERE datname = current_database()`).Scan(&blksHit, &blksRead)
	if blksHit+blksRead > 0 {
		report.CacheHitRatioPct = float64(blksHit) * 100 / float64(blksHit+blksRead)
	}

	totalConns := report.ActiveConns + report.IdleConns
	if report.MaxConnections > 0 && totalConns*100 >= report.MaxConnections*80 {
		report.Findings = append(report.Findings, fmt.Sprintf("连接数接近上限：active+idle=%d / max_connections=%d", totalConns, report.MaxConnections))
	}
	if report.Deadlocks > 0 {
		report.Findings = append(report.Findings, fmt.Sprintf("deadlocks=%d，存在死锁历史，建议排查长事务与锁等待", report.Deadlocks))
	}
	if report.CacheHitRatioPct > 0 && report.CacheHitRatioPct < 90 {
		report.Findings = append(report.Findings, fmt.Sprintf("缓存命中率偏低：%.1f%%，关注 shared_buffers / 热点查询", report.CacheHitRatioPct))
	}
	if report.ActiveConns > 50 {
		report.Findings = append(report.Findings, fmt.Sprintf("active_connections=%d 偏高，建议检查慢查询与连接池泄漏", report.ActiveConns))
	}
	if len(report.Findings) == 0 {
		report.Findings = append(report.Findings, "未发现明显高优先级 PostgreSQL 异常")
	}
	return report
}

func maskPostgreSQLDSN(dsn string) string {
	// postgres://user:pass@host/db -> mask password
	if i := strings.Index(dsn, "://"); i >= 0 {
		rest := dsn[i+3:]
		if at := strings.Index(rest, "@"); at > 0 {
			userPart := rest[:at]
			if colon := strings.Index(userPart, ":"); colon > 0 {
				return dsn[:i+3] + userPart[:colon+1] + "******@" + rest[at+1:]
			}
		}
	}
	// key=value DSN
	parts := strings.Fields(dsn)
	for i, p := range parts {
		if strings.HasPrefix(strings.ToLower(p), "password=") {
			parts[i] = "password=******"
		}
	}
	return strings.Join(parts, " ")
}

func formatPostgreSQLDiagnoseText(r *postgresqlDiagnoseReport) string {
	var b strings.Builder
	if len(r.Findings) > 0 {
		fmt.Fprintf(&b, "结论：%s\n\n", r.Findings[0])
	}
	for i, f := range r.Findings {
		fmt.Fprintf(&b, "%d. %s\n", i+1, f)
	}
	fmt.Fprintf(&b, "\n观测：version=%s active=%d idle=%d max_connections=%d deadlocks=%d cache_hit=%.1f%%\n",
		r.Version, r.ActiveConns, r.IdleConns, r.MaxConnections, r.Deadlocks, r.CacheHitRatioPct)
	if len(r.Errors) > 0 {
		b.WriteString("采集提示：\n")
		for _, e := range r.Errors {
			fmt.Fprintf(&b, "- %s\n", e)
		}
	}
	return b.String()
}
