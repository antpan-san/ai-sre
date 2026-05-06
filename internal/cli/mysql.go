package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/cobra"
)

type mysqlDiagnoseOptions struct {
	DSN     string
	Timeout time.Duration
	JSON    bool
}

type mysqlDiagnoseReport struct {
	DSN               string   `json:"dsn"`
	Version           string   `json:"version,omitempty"`
	ReadOnly          bool     `json:"read_only"`
	ThreadsConnected  int64    `json:"threads_connected,omitempty"`
	ThreadsRunning    int64    `json:"threads_running,omitempty"`
	AbortedConnects   int64    `json:"aborted_connects,omitempty"`
	SlowQueries       int64    `json:"slow_queries,omitempty"`
	MaxConnections    int64    `json:"max_connections,omitempty"`
	Findings          []string `json:"findings,omitempty"`
	Errors            []string `json:"errors,omitempty"`
}

func mysqlCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mysql",
		Short: "MySQL 极简快诊",
	}
	cmd.AddCommand(mysqlDiagnoseCmd())
	return cmd
}

func mysqlDiagnoseCmd() *cobra.Command {
	var opts mysqlDiagnoseOptions
	cmd := &cobra.Command{
		Use:   "diagnose <dsn>",
		Short: "只读连接 MySQL 采集关键指标并给出优先排查建议",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.DSN = strings.TrimSpace(args[0])
			if opts.Timeout <= 0 {
				opts.Timeout = 5 * time.Second
			}
			report := runMySQLDiagnose(cmd.Context(), opts)
			if opts.JSON || strings.EqualFold(outputFormat, "json") {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(report)
			}
			fmt.Fprint(cmd.OutOrStdout(), formatMySQLDiagnoseText(report))
			return nil
		},
	}
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 5*time.Second, "MySQL 连接与查询超时")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "输出机器可读 JSON")
	return cmd
}

func runMySQLDiagnose(parent context.Context, opts mysqlDiagnoseOptions) *mysqlDiagnoseReport {
	report := &mysqlDiagnoseReport{DSN: maskMySQLDSN(opts.DSN)}
	db, err := sql.Open("mysql", opts.DSN)
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
	_ = db.QueryRowContext(ctx, "SELECT @@version").Scan(&report.Version)
	var ro int64
	_ = db.QueryRowContext(ctx, "SELECT @@read_only").Scan(&ro)
	report.ReadOnly = ro == 1
	report.ThreadsConnected = mysqlStatusInt(ctx, db, "Threads_connected")
	report.ThreadsRunning = mysqlStatusInt(ctx, db, "Threads_running")
	report.AbortedConnects = mysqlStatusInt(ctx, db, "Aborted_connects")
	report.SlowQueries = mysqlStatusInt(ctx, db, "Slow_queries")
	report.MaxConnections = mysqlVariableInt(ctx, db, "max_connections")

	if report.MaxConnections > 0 && report.ThreadsConnected*100 >= report.MaxConnections*80 {
		report.Findings = append(report.Findings, fmt.Sprintf("连接数接近上限：threads_connected=%d / max_connections=%d", report.ThreadsConnected, report.MaxConnections))
	}
	if report.AbortedConnects > 0 {
		report.Findings = append(report.Findings, fmt.Sprintf("aborted_connects=%d，可能存在认证失败或网络抖动", report.AbortedConnects))
	}
	if report.ThreadsRunning > 100 {
		report.Findings = append(report.Findings, fmt.Sprintf("threads_running=%d 偏高，建议排查慢 SQL 与锁等待", report.ThreadsRunning))
	}
	if len(report.Findings) == 0 {
		report.Findings = append(report.Findings, "未发现明显高优先级 MySQL 异常")
	}
	return report
}

func mysqlStatusInt(ctx context.Context, db *sql.DB, key string) int64 {
	var name string
	var value int64
	if err := db.QueryRowContext(ctx, "SHOW GLOBAL STATUS LIKE ?", key).Scan(&name, &value); err != nil {
		return 0
	}
	return value
}

func mysqlVariableInt(ctx context.Context, db *sql.DB, key string) int64 {
	var name string
	var value int64
	if err := db.QueryRowContext(ctx, "SHOW GLOBAL VARIABLES LIKE ?", key).Scan(&name, &value); err != nil {
		return 0
	}
	return value
}

func maskMySQLDSN(dsn string) string {
	at := strings.Index(dsn, "@")
	if at <= 0 {
		return dsn
	}
	left := dsn[:at]
	if i := strings.Index(left, ":"); i > 0 {
		return left[:i+1] + "******" + dsn[at:]
	}
	return dsn
}

func formatMySQLDiagnoseText(r *mysqlDiagnoseReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "结论：%s\n\n", r.Findings[0])
	for i, f := range r.Findings {
		fmt.Fprintf(&b, "%d. %s\n", i+1, f)
	}
	fmt.Fprintf(&b, "\n观测：version=%s read_only=%t threads_connected=%d threads_running=%d aborted_connects=%d slow_queries=%d max_connections=%d\n",
		r.Version, r.ReadOnly, r.ThreadsConnected, r.ThreadsRunning, r.AbortedConnects, r.SlowQueries, r.MaxConnections)
	if len(r.Errors) > 0 {
		b.WriteString("采集提示：\n")
		for _, e := range r.Errors {
			fmt.Fprintf(&b, "- %s\n", e)
		}
	}
	return b.String()
}
