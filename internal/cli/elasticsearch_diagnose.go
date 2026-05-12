package cli

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// elasticsearchDiagnoseOptions 与 kafka diagnose / redis diagnose 对齐：子命令 diagnose、单地址参数、--timeout/--json，扩展 --user/--password/--insecure。
type elasticsearchDiagnoseOptions struct {
	BaseURL   string
	Timeout   time.Duration
	JSON      bool
	AI        bool
	User      string
	Password  string
	Insecure  bool
}

type elasticsearchFinding struct {
	Priority int    `json:"priority"`
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Evidence string `json:"evidence,omitempty"`
	Cause    string `json:"cause,omitempty"`
	Verify   string `json:"verify,omitempty"`
}

type elasticsearchClusterHealth struct {
	ClusterName               string  `json:"cluster_name"`
	Status                    string  `json:"status"`
	TimedOut                  bool    `json:"timed_out"`
	NumberOfNodes             int     `json:"number_of_nodes"`
	NumberOfDataNodes         int     `json:"number_of_data_nodes"`
	ActivePrimaryShards       int     `json:"active_primary_shards"`
	ActiveShards              int     `json:"active_shards"`
	RelocatingShards          int     `json:"relocating_shards"`
	InitializingShards        int     `json:"initializing_shards"`
	UnassignedShards          int     `json:"unassigned_shards"`
	DelayedUnassignedShards   int     `json:"delayed_unassigned_shards"`
	NumberOfPendingTasks      int     `json:"number_of_pending_tasks"`
	ActiveShardsPercentAsNum  float64 `json:"active_shards_percent_as_number"`
}

type elasticsearchNodeCatRow map[string]interface{}

type elasticsearchDiagnoseRawSummary struct {
	Mode                    string `json:"mode,omitempty"` // single-node | multi-node
	MaxHeapPercent          int    `json:"max_heap_percent,omitempty"`
	MaxDiskUsedPercent      int    `json:"max_disk_used_percent,omitempty"`
	CatNodesRows            int    `json:"cat_nodes_rows,omitempty"`
	CatNodesSkippedReason   string `json:"cat_nodes_skipped_reason,omitempty"`
}

type elasticsearchDiagnoseReport struct {
	BaseURL    string                         `json:"base_url"`
	Health     *elasticsearchClusterHealth    `json:"cluster_health,omitempty"`
	Nodes      []elasticsearchNodeCatRow      `json:"nodes,omitempty"`
	Findings   []elasticsearchFinding        `json:"findings"`
	RawSummary elasticsearchDiagnoseRawSummary `json:"raw_summary"`
	Errors     []string                       `json:"errors,omitempty"`
	AIAnswer   string                         `json:"ai_answer,omitempty"`
}

func elasticsearchDiagnoseCmd() *cobra.Command {
	var opts elasticsearchDiagnoseOptions
	cmd := &cobra.Command{
		Use:   "diagnose <http-url-or-host:port>",
		Short: "只读快诊 Elasticsearch：集群/单机健康、分片与 JVM/磁盘风险",
		Long: `通过 HTTP 调用 _cluster/health 与 _cat/nodes（只读），输出规则化风险结论；可选 --ai 在本地凭据可用时结合技能包 elasticsearch_health 做补充说明。
地址示例：127.0.0.1:9200、http://127.0.0.1:9200、https://es.internal:9200`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.BaseURL = strings.TrimSpace(args[0])
			if opts.Timeout <= 0 {
				opts.Timeout = 10 * time.Second
			}
			report, err := runElasticsearchDiagnose(cmd.Context(), opts)
			if err != nil {
				return err
			}
			if opts.JSON || strings.EqualFold(outputFormat, "json") {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(report)
			}
			fmt.Fprint(cmd.OutOrStdout(), formatElasticsearchDiagnoseText(report))
			return nil
		},
	}
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 10*time.Second, "HTTP 请求超时")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "输出机器可读 JSON")
	cmd.Flags().BoolVar(&opts.AI, "ai", false, "基于采集快照调用 LLM 生成补充解释（需凭据；默认关闭）")
	cmd.Flags().StringVar(&opts.User, "user", "", "HTTP Basic 用户名（可选）")
	cmd.Flags().StringVar(&opts.Password, "password", "", "HTTP Basic 密码（可选）")
	cmd.Flags().BoolVar(&opts.Insecure, "insecure", false, "HTTPS 时跳过服务端证书校验（等价 curl -k）")
	cmd.Example = fmt.Sprintf(`  %s elasticsearch diagnose 127.0.0.1:9200
  %s elasticsearch diagnose https://es:9200 --insecure --user elastic --password '***'
  %s elasticsearch diagnose 127.0.0.1:9200 --json`, progName, progName, progName)
	return cmd
}

func elasticsearchNormalizeBase(raw string) (*url.URL, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, errors.New("地址为空")
	}
	if !strings.Contains(s, "://") {
		s = "http://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	if u.Host == "" {
		return nil, fmt.Errorf("无效地址: %q", raw)
	}
	u.Path = ""
	u.RawQuery = ""
	u.Fragment = ""
	return u, nil
}

func elasticsearchBaseString(u *url.URL) string {
	s := u.String()
	return strings.TrimSuffix(s, "/")
}

func newElasticsearchHTTPClient(opts elasticsearchDiagnoseOptions) *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: opts.Insecure, //nolint:gosec // user opt-in for lab / self-signed
		},
	}
	return &http.Client{
		Timeout:   opts.Timeout,
		Transport: tr,
	}
}

func elasticsearchGET(ctx context.Context, client *http.Client, base string, path string, user, pass string) ([]byte, int, error) {
	reqURL := strings.TrimSuffix(base, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, 0, err
	}
	if user != "" || pass != "" {
		req.SetBasicAuth(user, pass)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return body, resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, resp.StatusCode, nil
}

func runElasticsearchDiagnose(ctx context.Context, opts elasticsearchDiagnoseOptions) (*elasticsearchDiagnoseReport, error) {
	u, err := elasticsearchNormalizeBase(opts.BaseURL)
	if err != nil {
		return nil, err
	}
	base := elasticsearchBaseString(u)
	client := newElasticsearchHTTPClient(opts)
	report := &elasticsearchDiagnoseReport{BaseURL: base}

	body, _, err := elasticsearchGET(ctx, client, base, "/_cluster/health", opts.User, opts.Password)
	if err != nil {
		report.Errors = append(report.Errors, "cluster health: "+err.Error())
		report.Findings = append(report.Findings, elasticsearchFinding{
			Priority: 1,
			Severity: "P1",
			Title:    "无法采集 _cluster/health",
			Evidence: err.Error(),
			Cause:    "网络不可达、TLS/证书问题、需 Basic 认证、或路径非 Elasticsearch HTTP 端口",
			Verify:   fmt.Sprintf("curl -sS %s/_cluster/health", shellQuote(base)),
		})
		if opts.AI {
			if a, aerr := explainElasticsearchReportWithAI(ctx, report); aerr == nil {
				report.AIAnswer = a
			} else {
				report.Errors = append(report.Errors, "AI 解释失败: "+aerr.Error())
			}
		}
		return report, nil
	}

	var health elasticsearchClusterHealth
	if err := json.Unmarshal(body, &health); err != nil {
		report.Errors = append(report.Errors, "解析 cluster health JSON: "+err.Error())
		return report, nil
	}
	report.Health = &health

	catBody, status, catErr := elasticsearchGET(ctx, client, base, "/_cat/nodes?format=json", opts.User, opts.Password)
	if catErr != nil {
		report.RawSummary.CatNodesSkippedReason = catErr.Error()
		if status == http.StatusForbidden || status == http.StatusUnauthorized {
			report.Errors = append(report.Errors, "_cat/nodes 被拒绝（权限），仅依据 cluster health 判断")
		} else {
			report.Errors = append(report.Errors, "_cat/nodes: "+catErr.Error())
		}
	} else {
		var rows []elasticsearchNodeCatRow
		if err := json.Unmarshal(catBody, &rows); err != nil {
			report.RawSummary.CatNodesSkippedReason = "json: " + err.Error()
			report.Errors = append(report.Errors, "解析 _cat/nodes JSON: "+err.Error())
		} else {
			report.Nodes = rows
			report.RawSummary.CatNodesRows = len(rows)
			maxHeap, maxDisk := 0, 0
			for _, row := range rows {
				if h := int(esCatGetFloat(row, "heap.percent", "heapPercent")); h > maxHeap {
					maxHeap = h
				}
				if d := int(esCatGetFloat(row, "disk.used_percent", "diskUsedPercent")); d > maxDisk {
					maxDisk = d
				}
			}
			report.RawSummary.MaxHeapPercent = maxHeap
			report.RawSummary.MaxDiskUsedPercent = maxDisk
		}
	}

	if health.NumberOfDataNodes <= 1 {
		report.RawSummary.Mode = "single-node"
	} else {
		report.RawSummary.Mode = "multi-node"
	}

	report.Findings = diagnoseElasticsearchSnapshot(report)
	if opts.AI {
		if answer, err := explainElasticsearchReportWithAI(ctx, report); err == nil {
			report.AIAnswer = answer
		} else {
			report.Errors = append(report.Errors, "AI 解释失败: "+err.Error())
		}
	}
	return report, nil
}

func esCatGetFloat(m map[string]interface{}, keys ...string) float64 {
	for _, k := range keys {
		v, ok := m[k]
		if !ok {
			continue
		}
		switch t := v.(type) {
		case float64:
			return t
		case string:
			f, _ := strconv.ParseFloat(strings.TrimSpace(t), 64)
			return f
		case json.Number:
			f, _ := t.Float64()
			return f
		}
	}
	return 0
}

func diagnoseElasticsearchSnapshot(r *elasticsearchDiagnoseReport) []elasticsearchFinding {
	var out []elasticsearchFinding
	h := r.Health
	if h == nil {
		return out
	}
	base := r.BaseURL
	verify := func(path string) string {
		return fmt.Sprintf("curl -sS %s%s", shellQuote(base), path)
	}

	switch strings.ToLower(strings.TrimSpace(h.Status)) {
	case "red":
		out = append(out, elasticsearchFinding{
			Priority: 0,
			Severity: "P0",
			Title:    "集群状态为 RED：部分主分片不可用",
			Evidence: fmt.Sprintf("status=red unassigned_shards=%d initializing_shards=%d", h.UnassignedShards, h.InitializingShards),
			Cause:    "主分片未分配或数据节点不可用，读写受影响",
			Verify:   verify("/_cluster/allocation/explain?pretty"),
		})
	case "yellow":
		if h.NumberOfDataNodes <= 1 && h.UnassignedShards > 0 {
			out = append(out, elasticsearchFinding{
				Priority: 2,
				Severity: "P2",
				Title:    "集群黄态：单数据节点下副本分片未分配（常见可接受）",
				Evidence: fmt.Sprintf("status=yellow number_of_data_nodes=%d unassigned_shards=%d", h.NumberOfDataNodes, h.UnassignedShards),
				Cause:    "副本无法落在同一 data 节点，属单节点部署预期现象；无跨机冗余",
				Verify:   verify("/_cluster/health?pretty"),
			})
		} else {
			out = append(out, elasticsearchFinding{
				Priority: 1,
				Severity: "P1",
				Title:    "集群黄态：多节点下仍有未分配分片，存在可用性风险",
				Evidence: fmt.Sprintf("status=yellow number_of_data_nodes=%d unassigned_shards=%d", h.NumberOfDataNodes, h.UnassignedShards),
				Cause:    "磁盘/水位、分片限制、集群路由或故障节点导致副本或主分片未就绪",
				Verify:   verify("/_cluster/allocation/explain?pretty"),
			})
		}
	default:
		if strings.EqualFold(h.Status, "green") && h.UnassignedShards == 0 && h.InitializingShards == 0 {
			// ok — may add no-op later
		}
	}

	if h.TimedOut {
		out = append(out, elasticsearchFinding{
			Priority: 1,
			Severity: "P1",
			Title:    "cluster health 请求 timed_out=true",
			Evidence: "master 或服务繁忙，健康接口超时",
			Cause:    "集群负载高、GC、或 master 不稳定",
			Verify:   verify("/_cluster/health?timeout=30s&pretty"),
		})
	}

	if h.RelocatingShards >= 20 {
		out = append(out, elasticsearchFinding{
			Priority: 2,
			Severity: "P2",
			Title:    "大量分片正在搬迁",
			Evidence: fmt.Sprintf("relocating_shards=%d", h.RelocatingShards),
			Cause:    "扩容、节点上下线、磁盘均衡或 routing 变更进行中",
			Verify:   verify("/_cat/shards?h=index,shard,prirep,state,node&s=state:desc&format=json"),
		})
	} else if h.RelocatingShards > 0 {
		out = append(out, elasticsearchFinding{
			Priority: 3,
			Severity: "P3",
			Title:    "存在分片搬迁",
			Evidence: fmt.Sprintf("relocating_shards=%d", h.RelocatingShards),
			Cause:    "集群拓扑或磁盘均衡调整中",
			Verify:   verify("/_cat/recovery?active_only=true&format=json"),
		})
	}

	if h.DelayedUnassignedShards > 0 {
		out = append(out, elasticsearchFinding{
			Priority: 1,
			Severity: "P1",
			Title:    "存在 delayed_unassigned_shards",
			Evidence: fmt.Sprintf("delayed_unassigned_shards=%d", h.DelayedUnassignedShards),
			Cause:    "节点短暂失联或磁盘慢导致分片延迟分配，若持续上升需排查节点与磁盘",
			Verify:   verify("/_cluster/allocation/explain?pretty"),
		})
	}

	if h.NumberOfPendingTasks > 50 {
		out = append(out, elasticsearchFinding{
			Priority: 2,
			Severity: "P2",
			Title:    "Master 队列任务偏多",
			Evidence: fmt.Sprintf("number_of_pending_tasks=%d", h.NumberOfPendingTasks),
			Cause:    "大量元数据变更、快照、或 master 压力大",
			Verify:   verify("/_cat/pending_tasks?v"),
		})
	}

	if r.RawSummary.MaxHeapPercent >= 90 {
		out = append(out, elasticsearchFinding{
			Priority: 0,
			Severity: "P0",
			Title:    "节点堆内存使用率过高（>=90%）",
			Evidence: fmt.Sprintf("max heap.percent≈%d（来自 _cat/nodes）", r.RawSummary.MaxHeapPercent),
			Cause:    "堆偏小或查询/聚合压力导致 OOM 风险",
			Verify:   verify("/_cat/nodes?h=name,heap.percent,heap.current,heap.max&format=json"),
		})
	} else if r.RawSummary.MaxHeapPercent >= 85 {
		out = append(out, elasticsearchFinding{
			Priority: 1,
			Severity: "P1",
			Title:    "节点堆内存使用率偏高（>=85%）",
			Evidence: fmt.Sprintf("max heap.percent≈%d", r.RawSummary.MaxHeapPercent),
			Cause:    "接近 GC 与 CircuitBreaker 压力区",
			Verify:   verify("/_nodes/stats/jvm?filter_path=nodes.*.jvm.mem.*"),
		})
	}

	if r.RawSummary.MaxDiskUsedPercent >= 95 {
		out = append(out, elasticsearchFinding{
			Priority: 0,
			Severity: "P0",
			Title:    "磁盘使用率极高，可能触发 flood stage",
			Evidence: fmt.Sprintf("max disk.used_percent≈%d", r.RawSummary.MaxDiskUsedPercent),
			Cause:    "磁盘写满会导致分片分配拒绝甚至只读索引",
			Verify:   verify("/_cat/allocation?bytes=b&format=json"),
		})
	} else if r.RawSummary.MaxDiskUsedPercent >= 90 {
		out = append(out, elasticsearchFinding{
			Priority: 1,
			Severity: "P1",
			Title:    "磁盘使用率过高（>=90%）",
			Evidence: fmt.Sprintf("max disk.used_percent≈%d", r.RawSummary.MaxDiskUsedPercent),
			Cause:    "接近高水位线，可能影响分片分配与写入",
			Verify:   verify("/_cluster/settings?include_defaults=true&filter_path=**.disk*"),
		})
	} else if r.RawSummary.MaxDiskUsedPercent >= 75 && r.RawSummary.MaxDiskUsedPercent > 0 {
		out = append(out, elasticsearchFinding{
			Priority: 3,
			Severity: "P3",
			Title:    "磁盘使用率进入关注区间（>=75%）",
			Evidence: fmt.Sprintf("max disk.used_percent≈%d", r.RawSummary.MaxDiskUsedPercent),
			Cause:    "建议规划清理或扩容，避免触及 flood watermark",
			Verify:   verify("/_cat/allocation?bytes=b&format=json"),
		})
	}

	if h.ActiveShardsPercentAsNum > 0 && h.ActiveShardsPercentAsNum < 100 && strings.EqualFold(h.Status, "green") {
		out = append(out, elasticsearchFinding{
			Priority: 3,
			Severity: "P3",
			Title:    "active_shards_percent 未满 100%",
			Evidence: fmt.Sprintf("active_shards_percent_as_number=%.1f", h.ActiveShardsPercentAsNum),
			Cause:    "部分索引关闭或 shrink/迁移中，需结合业务确认",
			Verify:   verify("/_cat/indices?health=yellow,red&format=json"),
		})
	}

	if len(out) == 0 {
		out = append(out, elasticsearchFinding{
			Priority: 3,
			Severity: "P3",
			Title:    "规则引擎未见明显高风险项",
			Evidence: fmt.Sprintf("status=%s nodes=%d data_nodes=%d unassigned=%d", h.Status, h.NumberOfNodes, h.NumberOfDataNodes, h.UnassignedShards),
			Cause:    "仍建议结合业务查询延迟与 GC 日志做深度巡检",
			Verify:   verify("/_cluster/health?pretty"),
		})
	}

	// 按 priority 排序后截断
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Priority < out[i].Priority {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	if len(out) > 8 {
		out = out[:8]
	}
	return out
}

func formatElasticsearchDiagnoseText(r *elasticsearchDiagnoseReport) string {
	var b strings.Builder
	h := r.Health
	if h == nil {
		fmt.Fprintf(&b, "结论：未能获取 cluster health，请检查地址与权限。\n\n")
	} else {
		high := 0
		for _, f := range r.Findings {
			if f.Priority <= 1 {
				high++
			}
		}
		mode := r.RawSummary.Mode
		if mode == "" {
			mode = "unknown"
		}
		if len(r.Findings) == 0 {
			fmt.Fprintf(&b, "结论：%s，status=%s，规则引擎无额外项。\n\n", mode, h.Status)
		} else if high > 0 {
			fmt.Fprintf(&b, "结论：%s，status=%s，发现 %d 条高优先级关注项，建议先处理「%s」。\n\n", mode, h.Status, high, r.Findings[0].Title)
		} else {
			fmt.Fprintf(&b, "结论：%s，status=%s，以下为主要提示（无 P0/P1 级规则命中时可继续观察）。\n\n", mode, h.Status)
		}
		fmt.Fprintf(&b, "集群：%s | 节点=%d data=%d | 主分片=%d 未分配=%d 搬迁中=%d\n\n",
			h.ClusterName, h.NumberOfNodes, h.NumberOfDataNodes, h.ActivePrimaryShards, h.UnassignedShards, h.RelocatingShards)
	}

	for i, f := range r.Findings {
		fmt.Fprintf(&b, "%d. [%s] %s\n", i+1, f.Severity, f.Title)
		if f.Evidence != "" {
			fmt.Fprintf(&b, "   证据：%s\n", f.Evidence)
		}
		if f.Cause != "" {
			fmt.Fprintf(&b, "   可能原因：%s\n", f.Cause)
		}
		if f.Verify != "" {
			fmt.Fprintf(&b, "   最快验证：%s\n", f.Verify)
		}
		b.WriteString("\n")
	}

	if r.RawSummary.CatNodesRows > 0 {
		fmt.Fprintf(&b, "节点采样：%d 行；heap 峰值≈%d%%；磁盘峰值≈%d%%\n",
			r.RawSummary.CatNodesRows, r.RawSummary.MaxHeapPercent, r.RawSummary.MaxDiskUsedPercent)
	} else if r.RawSummary.CatNodesSkippedReason != "" {
		fmt.Fprintf(&b, "节点采样：未获取（%s）\n", r.RawSummary.CatNodesSkippedReason)
	}

	if len(r.Errors) > 0 {
		b.WriteString("\n采集提示：\n")
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

func explainElasticsearchReportWithAI(ctx context.Context, report *elasticsearchDiagnoseReport) (string, error) {
	eng, err := bootstrap()
	if err != nil {
		return "", err
	}
	b, _ := json.Marshal(report)
	res, err := eng.Analyze(ctx, "elasticsearch", map[string]string{
		"issue":    "diagnose",
		"snapshot": string(b),
		"base_url": report.BaseURL,
	}, !noRAG)
	if err != nil {
		return "", err
	}
	if res == nil {
		return "", nil
	}
	return res.Answer, nil
}
