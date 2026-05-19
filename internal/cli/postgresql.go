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
	DSN                 string   `json:"dsn"`
	Version             string   `json:"version,omitempty"`
	InRecovery          bool     `json:"in_recovery"`
	MaxConnections      int64    `json:"max_connections,omitempty"`
	ActiveConnections   int64    `json:"active_connections,omitempty"`
	IdleInTransaction   int64    `json:"idle_in_transaction,omitempty"`
	TotalConnections    int64    `json:"total_connections,omitempty"`
	Deadlocks           int64    `json:"deadlocks,omitempty"`
	Findings            []string `json:"findings,omitempty"`
	Errors              []string `json:"errors,omitempty"`
}

func postgresqlCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "postgresql",
		Short: "PostgreSQL 极简快诊",
	}
	cmd.AddCommand(postgresqlDiagnoseCmd())
	return cmd
}

func postgresqlDiagnoseCmd() *cobra.Command {
	var opts postgresqlDiagnoseOptions
	cmd := &cobra.Command{
		Use:        "diagnose <dsn>",
		Short:      "（已弃用）请改用 probe postgresql",
		Deprecated: "use \"probe postgresql\" instead",
		Args:       cobra.ExactArgs(1),
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
	var inRecovery bool
	_ = db.QueryRowContext(ctx, "SELECT pg_is_in_recovery()").Scan(&inRecovery)
	report.InRecovery = inRecovery
	report.MaxConnections = pgSettingInt(ctx, db, "max_connections")
	report.TotalConnections = pgScalarInt(ctx, db, "SELECT count(*)::bigint FROM pg_stat_activity")
	report.ActiveConnections = pgScalarInt(ctx, db, "SELECT count(*)::bigint FROM pg_stat_activity WHERE state = 'active'")
	report.IdleInTransaction = pgScalarInt(ctx, db, "SELECT count(*)::bigint FROM pg_stat_activity WHERE state = 'idle in transaction'")
	report.Deadlocks = pgScalarInt(ctx, db, "SELECT COALESCE(sum(deadlocks),0)::bigint FROM pg_stat_database")

	if report.MaxConnections > 0 && report.TotalConnections*100 >= report.MaxConnections*80 {
		report.Findings = append(report.Findings, fmt.Sprintf("连接数接近上限：total=%d / max_connections=%d", report.TotalConnections, report.MaxConnections))
	}
	if report.IdleInTransaction > 0 {
		report.Findings = append(report.Findings, fmt.Sprintf("idle in transaction=%d，可能存在长事务未提交", report.IdleInTransaction))
	}
	if report.Deadlocks > 0 {
		report.Findings = append(report.Findings, fmt.Sprintf("deadlocks=%d，建议排查并发更新与锁等待", report.Deadlocks))
	}
	if report.InRecovery {
		report.Findings = append(report.Findings, "实例处于恢复/只读副本模式（pg_is_in_recovery=true）")
	}
	if report.ActiveConnections > 100 {
		report.Findings = append(report.Findings, fmt.Sprintf("active 连接=%d 偏高，建议排查慢查询与锁", report.ActiveConnections))
	}
	if len(report.Findings) == 0 {
		report.Findings = append(report.Findings, "未发现明显高优先级 PostgreSQL 异常")
	}
	return report
}

func pgSettingInt(ctx context.Context, db *sql.DB, name string) int64 {
	var value int64
	if err := db.QueryRowContext(ctx, "SELECT setting::bigint FROM pg_settings WHERE name = $1", name).Scan(&value); err != nil {
		return 0
	}
	return value
}

func pgScalarInt(ctx context.Context, db *sql.DB, query string) int64 {
	var value int64
	if err := db.QueryRowContext(ctx, query).Scan(&value); err != nil {
		return 0
	}
	return value
}

func maskPostgreSQLDSN(dsn string) string {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		if at := strings.Index(dsn, "@"); at > 0 {
			schemeEnd := strings.Index(dsn, "://")
			if schemeEnd >= 0 && at > schemeEnd+3 {
				prefix := dsn[:schemeEnd+3]
				rest := dsn[at:]
				if colon := strings.Index(dsn[schemeEnd+3:at], ":"); colon >= 0 {
					return prefix + "******" + rest
				}
			}
		}
		return dsn
	}
	if i := strings.Index(strings.ToLower(dsn), "password="); i >= 0 {
		end := strings.IndexAny(dsn[i:], " \t")
		if end < 0 {
			return dsn[:i] + "password=******"
		}
		return dsn[:i] + "password=******" + dsn[i+end:]
	}
	return dsn
}

func formatPostgreSQLDiagnoseText(r *postgresqlDiagnoseReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "结论：%s\n\n", r.Findings[0])
	for i, f := range r.Findings {
		fmt.Fprintf(&b, "%d. %s\n", i+1, f)
	}
	fmt.Fprintf(&b, "\n观测：version=%s in_recovery=%t total=%d active=%d idle_in_tx=%d deadlocks=%d max_connections=%d\n",
		r.Version, r.InRecovery, r.TotalConnections, r.ActiveConnections, r.IdleInTransaction, r.Deadlocks, r.MaxConnections)
	if len(r.Errors) > 0 {
		b.WriteString("采集提示：\n")
		for _, e := range r.Errors {
			fmt.Fprintf(&b, "- %s\n", e)
		}
	}
	return b.String()
}
