package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

type nginxDiagnoseOptions struct {
	AccessLog string
	Tail      int
	JSON      bool
}

type nginxPathStat struct {
	Path  string `json:"path"`
	Count int64  `json:"count"`
}

type nginxDiagnoseReport struct {
	AccessLog        string          `json:"access_log"`
	ScannedLines     int64           `json:"scanned_lines"`
	ReqTotal         int64           `json:"req_total"`
	Status2xx        int64           `json:"status_2xx"`
	Status3xx        int64           `json:"status_3xx"`
	Status4xx        int64           `json:"status_4xx"`
	Status5xx        int64           `json:"status_5xx"`
	P95RequestTimeMs int64           `json:"p95_request_time_ms"`
	TopPaths         []nginxPathStat `json:"top_paths,omitempty"`
	Findings         []string        `json:"findings,omitempty"`
	Errors           []string        `json:"errors,omitempty"`
	AIAnswer         string          `json:"ai_answer,omitempty"`
}

func nginxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nginx",
		Short: "Nginx 统计分析与快诊",
	}
	cmd.AddCommand(nginxDiagnoseCmd(), nginxUpdateCmd())
	return cmd
}

func nginxUpdateCmd() *cobra.Command {
	var opts serviceUpdateOptions
	cmd := &cobra.Command{
		Use:   "update",
		Short: "从 OpsFleet 拉取最新 Nginx 部署规格并重启生效",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServiceUpdate(cmd, "nginx", opts)
		},
	}
	cmd.Flags().StringVar(&opts.APIURL, "api-url", "", "OpsFleet API base，例如 http://host:9080/ft-api；默认读取本机安装状态")
	cmd.Flags().StringVar(&opts.DeployID, "deploy-id", "", "服务端部署 ID；默认读取本机安装状态")
	cmd.Flags().StringVar(&opts.Token, "token", "", "服务端部署 token；默认读取本机安装状态")
	cmd.Flags().StringVar(&opts.FromURL, "from", "", "完整 spec URL（可替代 api-url/deploy-id/token）")
	return cmd
}

func nginxDiagnoseCmd() *cobra.Command {
	var opts nginxDiagnoseOptions
	cmd := &cobra.Command{
		Use:   "diagnose",
		Short: "分析 access log 并输出状态码、慢请求、Top 路径",
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Tail <= 0 {
				opts.Tail = 5000
			}
			if strings.TrimSpace(opts.AccessLog) == "" {
				opts.AccessLog = "/var/log/nginx/access.log"
			}
			report := runNginxDiagnose(opts)
			if len(report.Errors) > 0 {
				ctx := map[string]string{
					"issue":       "nginx_log_collect_failed",
					"access_log":  opts.AccessLog,
					"error_count": strconv.Itoa(len(report.Errors)),
				}
				if diag, err := runAnalyzeWithOrchestrator(cmd.Context(), "nginx", ctx); err == nil {
					report.AIAnswer = strings.TrimSpace(diag.Answer)
				}
			}
			if opts.JSON || strings.EqualFold(outputFormat, "json") {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(report)
			}
			fmt.Fprint(cmd.OutOrStdout(), formatNginxDiagnoseText(report))
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.AccessLog, "access-log", "/var/log/nginx/access.log", "nginx access log 路径")
	cmd.Flags().IntVar(&opts.Tail, "tail", 5000, "最多统计末尾多少行")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "输出机器可读 JSON")
	return cmd
}

func runNginxDiagnose(opts nginxDiagnoseOptions) *nginxDiagnoseReport {
	report := &nginxDiagnoseReport{AccessLog: opts.AccessLog}
	f, err := os.Open(opts.AccessLog)
	if err != nil {
		report.Errors = append(report.Errors, "打开 access log 失败: "+err.Error())
		report.Findings = append(report.Findings, "日志采集失败，建议检查 nginx 日志路径和权限")
		return report
	}
	defer f.Close()

	lines := make([]string, 0, opts.Tail)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
		if len(lines) > opts.Tail {
			lines = lines[1:]
		}
	}
	if err := sc.Err(); err != nil {
		report.Errors = append(report.Errors, "读取 access log 失败: "+err.Error())
	}
	report.ScannedLines = int64(len(lines))

	pathCount := map[string]int64{}
	var latencies []float64
	for _, line := range lines {
		path, status, reqTime, ok := parseNginxAccessLine(line)
		if !ok {
			continue
		}
		report.ReqTotal++
		pathCount[path]++
		switch {
		case status >= 500:
			report.Status5xx++
		case status >= 400:
			report.Status4xx++
		case status >= 300:
			report.Status3xx++
		case status >= 200:
			report.Status2xx++
		}
		if reqTime >= 0 {
			latencies = append(latencies, reqTime)
		}
	}

	report.TopPaths = topNginxPaths(pathCount, 5)
	report.P95RequestTimeMs = percentileMs(latencies, 95)
	if report.ReqTotal == 0 {
		report.Findings = append(report.Findings, "未解析到有效请求日志，请确认 log_format 是否包含标准请求行")
		return report
	}
	if report.Status5xx > 0 {
		report.Findings = append(report.Findings, fmt.Sprintf("发现 5xx=%d，优先排查 upstream 可用性与超时", report.Status5xx))
	}
	if report.Status4xx > report.ReqTotal/2 {
		report.Findings = append(report.Findings, fmt.Sprintf("4xx 占比偏高（%d/%d），可能有路由、鉴权或客户端错误", report.Status4xx, report.ReqTotal))
	}
	if report.P95RequestTimeMs >= 1000 {
		report.Findings = append(report.Findings, fmt.Sprintf("请求延迟 P95=%dms，存在明显慢请求", report.P95RequestTimeMs))
	}
	if len(report.Findings) == 0 {
		report.Findings = append(report.Findings, "未发现明显高优先级 Nginx 异常")
	}
	return report
}

func parseNginxAccessLine(line string) (path string, status int, reqTime float64, ok bool) {
	reqTime = -1
	first := strings.Index(line, "\"")
	second := strings.Index(line[first+1:], "\"")
	if first < 0 || second < 0 {
		return "", 0, reqTime, false
	}
	second = first + 1 + second
	req := strings.Fields(line[first+1 : second])
	if len(req) >= 2 {
		path = req[1]
	}
	rest := strings.Fields(strings.TrimSpace(line[second+1:]))
	if len(rest) == 0 {
		return path, 0, reqTime, path != ""
	}
	status, _ = strconv.Atoi(rest[0])
	if len(rest) > 0 {
		last := rest[len(rest)-1]
		if v, err := strconv.ParseFloat(last, 64); err == nil {
			reqTime = v
		}
	}
	return path, status, reqTime, path != ""
}

func topNginxPaths(m map[string]int64, n int) []nginxPathStat {
	out := make([]nginxPathStat, 0, len(m))
	for k, v := range m {
		out = append(out, nginxPathStat{Path: k, Count: v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Path < out[j].Path
		}
		return out[i].Count > out[j].Count
	})
	if len(out) > n {
		return out[:n]
	}
	return out
}

func percentileMs(values []float64, p int) int64 {
	if len(values) == 0 {
		return 0
	}
	sort.Float64s(values)
	if p <= 0 {
		return int64(values[0] * 1000)
	}
	if p >= 100 {
		return int64(values[len(values)-1] * 1000)
	}
	idx := int(float64(len(values)-1) * (float64(p) / 100.0))
	return int64(values[idx] * 1000)
}

func formatNginxDiagnoseText(r *nginxDiagnoseReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "结论：%s\n\n", r.Findings[0])
	for i, f := range r.Findings {
		fmt.Fprintf(&b, "%d. %s\n", i+1, f)
	}
	fmt.Fprintf(&b, "\n统计：req=%d 2xx=%d 3xx=%d 4xx=%d 5xx=%d p95=%dms (log=%s, scanned=%d)\n",
		r.ReqTotal, r.Status2xx, r.Status3xx, r.Status4xx, r.Status5xx, r.P95RequestTimeMs, r.AccessLog, r.ScannedLines)
	if len(r.TopPaths) > 0 {
		b.WriteString("Top 路径：\n")
		for _, p := range r.TopPaths {
			fmt.Fprintf(&b, "- %s => %d\n", p.Path, p.Count)
		}
	}
	if len(r.Errors) > 0 {
		b.WriteString("采集提示：\n")
		for _, e := range r.Errors {
			fmt.Fprintf(&b, "- %s\n", e)
		}
	}
	if strings.TrimSpace(r.AIAnswer) != "" {
		b.WriteString("\nAI 补充：\n")
		b.WriteString(strings.TrimSpace(r.AIAnswer))
		b.WriteString("\n")
	}
	return b.String()
}
