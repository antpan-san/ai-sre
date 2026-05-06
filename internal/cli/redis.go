package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type redisDiagnoseOptions struct {
	Address  string
	Password string
	Timeout  time.Duration
	JSON     bool
}

type redisDiagnoseReport struct {
	Address            string   `json:"address"`
	Role               string   `json:"role,omitempty"`
	UsedMemoryHuman    string   `json:"used_memory_human,omitempty"`
	ConnectedClients   int      `json:"connected_clients,omitempty"`
	RejectedConnections int64   `json:"rejected_connections,omitempty"`
	EvictedKeys        int64    `json:"evicted_keys,omitempty"`
	ExpiredKeys        int64    `json:"expired_keys,omitempty"`
	InstantOpsPerSec   int64    `json:"instant_ops_per_sec,omitempty"`
	Findings           []string `json:"findings,omitempty"`
	Errors             []string `json:"errors,omitempty"`
}

func redisCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "redis",
		Short: "Redis 极简快诊",
	}
	cmd.AddCommand(redisDiagnoseCmd())
	return cmd
}

func redisDiagnoseCmd() *cobra.Command {
	var opts redisDiagnoseOptions
	cmd := &cobra.Command{
		Use:   "diagnose <addr>",
		Short: "只读采集 Redis INFO，输出最值得优先处理的问题",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Address = strings.TrimSpace(args[0])
			if opts.Timeout <= 0 {
				opts.Timeout = 5 * time.Second
			}
			report := runRedisDiagnose(opts)
			if opts.JSON || strings.EqualFold(outputFormat, "json") {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(report)
			}
			fmt.Fprint(cmd.OutOrStdout(), formatRedisDiagnoseText(report))
			return nil
		},
	}
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 5*time.Second, "Redis 连接与命令超时")
	cmd.Flags().StringVar(&opts.Password, "password", "", "Redis AUTH 密码（可选）")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "输出机器可读 JSON")
	return cmd
}

func runRedisDiagnose(opts redisDiagnoseOptions) *redisDiagnoseReport {
	report := &redisDiagnoseReport{Address: opts.Address}
	conn, err := net.DialTimeout("tcp", opts.Address, opts.Timeout)
	if err != nil {
		report.Errors = append(report.Errors, "连接失败: "+err.Error())
		return report
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(opts.Timeout))
	r := bufio.NewReader(conn)

	if opts.Password != "" {
		if err := redisWriteCommand(conn, "AUTH", opts.Password); err != nil {
			report.Errors = append(report.Errors, "AUTH 写入失败: "+err.Error())
			return report
		}
		line, _ := r.ReadString('\n')
		if !strings.HasPrefix(line, "+OK") {
			report.Errors = append(report.Errors, "AUTH 失败: "+strings.TrimSpace(line))
			return report
		}
	}

	if err := redisWriteCommand(conn, "INFO"); err != nil {
		report.Errors = append(report.Errors, "INFO 写入失败: "+err.Error())
		return report
	}
	info, err := redisReadBulkString(r)
	if err != nil {
		report.Errors = append(report.Errors, "INFO 读取失败: "+err.Error())
		return report
	}
	kv := parseRedisInfo(info)
	report.Role = kv["role"]
	report.UsedMemoryHuman = kv["used_memory_human"]
	report.ConnectedClients = atoiOrZero(kv["connected_clients"])
	report.RejectedConnections = atoi64OrZero(kv["rejected_connections"])
	report.EvictedKeys = atoi64OrZero(kv["evicted_keys"])
	report.ExpiredKeys = atoi64OrZero(kv["expired_keys"])
	report.InstantOpsPerSec = atoi64OrZero(kv["instantaneous_ops_per_sec"])

	if report.RejectedConnections > 0 {
		report.Findings = append(report.Findings, fmt.Sprintf("发现 rejected_connections=%d，存在连接被拒绝", report.RejectedConnections))
	}
	if report.EvictedKeys > 0 {
		report.Findings = append(report.Findings, fmt.Sprintf("发现 evicted_keys=%d，可能触发内存淘汰", report.EvictedKeys))
	}
	if report.ConnectedClients > 10000 {
		report.Findings = append(report.Findings, fmt.Sprintf("connected_clients=%d 偏高，建议核查连接池和慢请求", report.ConnectedClients))
	}
	if len(report.Findings) == 0 {
		report.Findings = append(report.Findings, "未发现明显高优先级 Redis 异常")
	}
	return report
}

func redisWriteCommand(conn net.Conn, args ...string) error {
	var b strings.Builder
	b.WriteString("*")
	b.WriteString(strconv.Itoa(len(args)))
	b.WriteString("\r\n")
	for _, arg := range args {
		b.WriteString("$")
		b.WriteString(strconv.Itoa(len(arg)))
		b.WriteString("\r\n")
		b.WriteString(arg)
		b.WriteString("\r\n")
	}
	_, err := conn.Write([]byte(b.String()))
	return err
}

func redisReadBulkString(r *bufio.Reader) (string, error) {
	head, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(head, "$") {
		return "", fmt.Errorf("unexpected reply: %s", strings.TrimSpace(head))
	}
	n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(head, "$")))
	if err != nil || n < 0 {
		return "", fmt.Errorf("bad bulk size: %s", strings.TrimSpace(head))
	}
	buf := make([]byte, n+2)
	if _, err := r.Read(buf); err != nil {
		return "", err
	}
	return string(buf[:n]), nil
}

func parseRedisInfo(info string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(info, "\n") {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		kv := strings.SplitN(s, ":", 2)
		if len(kv) == 2 {
			out[kv[0]] = kv[1]
		}
	}
	return out
}

func formatRedisDiagnoseText(r *redisDiagnoseReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "结论：%s\n\n", r.Findings[0])
	for i, f := range r.Findings {
		fmt.Fprintf(&b, "%d. %s\n", i+1, f)
	}
	fmt.Fprintf(&b, "\n观测：role=%s connected_clients=%d ops=%d used_memory=%s evicted=%d rejected=%d expired=%d\n",
		r.Role, r.ConnectedClients, r.InstantOpsPerSec, r.UsedMemoryHuman, r.EvictedKeys, r.RejectedConnections, r.ExpiredKeys)
	if len(r.Errors) > 0 {
		b.WriteString("采集提示：\n")
		for _, e := range r.Errors {
			fmt.Fprintf(&b, "- %s\n", e)
		}
	}
	return b.String()
}

func atoiOrZero(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

func atoi64OrZero(s string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}
