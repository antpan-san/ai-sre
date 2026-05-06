package cli

import (
	"crypto/sha256"
	"crypto/tls"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/spf13/cobra"
	"github.com/xdg-go/scram"
)

type kafkaDiagnoseOptions struct {
	BootstrapServer string
	Limit           int
	Timeout         time.Duration
	CommandDir      string
	ClientConfig    string
	JSON            bool
	AI              bool
}

type kafkaCommandPaths struct {
	ConsumerGroups string `json:"consumer_groups,omitempty"`
	Topics         string `json:"topics,omitempty"`
	BrokerAPI      string `json:"broker_api,omitempty"`
}

type kafkaSnapshot struct {
	BootstrapServer string              `json:"bootstrap"`
	Groups          []kafkaGroupSummary `json:"groups,omitempty"`
	Topics          []kafkaTopicSummary `json:"topics,omitempty"`
	Commands        kafkaCommandPaths   `json:"commands,omitempty"`
	Errors          []string            `json:"errors,omitempty"`
}

type kafkaGroupSummary struct {
	Name            string                  `json:"name"`
	State           string                  `json:"state,omitempty"`
	TotalLag        int64                   `json:"total_lag"`
	MaxPartitionLag int64                   `json:"max_partition_lag"`
	MaxLagTopic     string                  `json:"max_lag_topic,omitempty"`
	MaxLagPartition int                     `json:"max_lag_partition,omitempty"`
	Partitions      int                     `json:"partitions"`
	ActiveMembers   int                     `json:"active_members"`
	NoActiveMembers bool                    `json:"no_active_members"`
	Rows            []kafkaConsumerGroupRow `json:"rows,omitempty"`
	RawError        string                  `json:"raw_error,omitempty"`
}

type kafkaConsumerGroupRow struct {
	Group         string `json:"group"`
	Topic         string `json:"topic"`
	Partition     int    `json:"partition"`
	CurrentOffset string `json:"current_offset,omitempty"`
	LogEndOffset  string `json:"log_end_offset,omitempty"`
	Lag           int64  `json:"lag"`
	ConsumerID    string `json:"consumer_id,omitempty"`
	Host          string `json:"host,omitempty"`
	ClientID      string `json:"client_id,omitempty"`
}

type kafkaTopicSummary struct {
	Name                      string                `json:"name"`
	PartitionCount            int                   `json:"partition_count"`
	OfflinePartitions         int                   `json:"offline_partitions"`
	UnderReplicatedPartitions int                   `json:"under_replicated_partitions"`
	Partitions                []kafkaTopicPartition `json:"partitions,omitempty"`
}

type kafkaTopicPartition struct {
	Topic     string   `json:"topic"`
	Partition int      `json:"partition"`
	Leader    string   `json:"leader"`
	Replicas  []string `json:"replicas,omitempty"`
	ISR       []string `json:"isr,omitempty"`
}

type kafkaFinding struct {
	Priority int    `json:"priority"`
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Evidence string `json:"evidence"`
	Cause    string `json:"cause"`
	Verify   string `json:"verify"`
	Resource string `json:"resource,omitempty"`
}

type kafkaDiagnoseReport struct {
	BootstrapServer string          `json:"bootstrap"`
	GroupsScanned   int             `json:"groups_scanned"`
	TopicsScanned   int             `json:"topics_scanned"`
	Findings        []kafkaFinding  `json:"findings"`
	RawSummary      kafkaRawSummary `json:"raw_summary"`
	Errors          []string        `json:"errors,omitempty"`
	AIAnswer        string          `json:"ai_answer,omitempty"`
}

type kafkaRawSummary struct {
	ConsumerGroupsListed      int   `json:"consumer_groups_listed"`
	TotalLag                  int64 `json:"total_lag"`
	OfflinePartitions         int   `json:"offline_partitions"`
	UnderReplicatedPartitions int   `json:"under_replicated_partitions"`
}

func kafkaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kafka",
		Short: "Kafka 极简快诊",
	}
	cmd.AddCommand(kafkaDiagnoseCmd())
	return cmd
}

func kafkaDiagnoseCmd() *cobra.Command {
	var opts kafkaDiagnoseOptions
	cmd := &cobra.Command{
		Use:   "diagnose <bootstrap-server>",
		Short: "一键快诊 Kafka：自动发现 group/topic 并输出最值得先看的问题",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.BootstrapServer = strings.TrimSpace(args[0])
			if opts.Limit <= 0 {
				opts.Limit = 20
			}
			if opts.Timeout <= 0 {
				opts.Timeout = 10 * time.Second
			}
			report, err := runKafkaDiagnose(cmd.Context(), opts)
			if err != nil {
				return err
			}
			if opts.JSON || strings.EqualFold(outputFormat, "json") {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(report)
			}
			fmt.Print(formatKafkaDiagnoseText(report))
			return nil
		},
	}
	cmd.Flags().IntVar(&opts.Limit, "limit", 20, "最多分析多少个 consumer group")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 10*time.Second, "每个 Kafka CLI 只读命令的超时时间")
	cmd.Flags().StringVar(&opts.CommandDir, "command-dir", "", "Kafka 脚本目录（含 kafka-consumer-groups.sh 等）")
	cmd.Flags().StringVar(&opts.ClientConfig, "config", "", "Kafka client properties 文件（SASL/TLS 等，传给 --command-config）")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "输出机器可读 JSON")
	cmd.Flags().BoolVar(&opts.AI, "ai", false, "基于采集快照调用 LLM 生成补充解释（默认关闭）")
	cmd.Example = fmt.Sprintf(`  %s kafka diagnose 10.0.0.1:9092
  %s kafka diagnose 10.0.0.1:9092 --config ./client.properties
  %s kafka diagnose 10.0.0.1:9092 --json`, progName, progName, progName)
	return cmd
}

func runKafkaDiagnose(ctx context.Context, opts kafkaDiagnoseOptions) (*kafkaDiagnoseReport, error) {
	if nativeSnap, nativeErr := collectKafkaSnapshotByGo(ctx, opts); nativeErr == nil {
		report := buildKafkaReport(nativeSnap)
		if opts.AI {
			if answer, err := explainKafkaReportWithAI(ctx, report); err == nil {
				report.AIAnswer = answer
			} else {
				report.Errors = append(report.Errors, "AI 解释失败: "+err.Error())
			}
		}
		return report, nil
	}

	paths, errs := resolveKafkaCommandPaths(opts.CommandDir)
	if paths.ConsumerGroups == "" && paths.Topics == "" {
		fallbackCtx := map[string]string{
			"bootstrap_servers": opts.BootstrapServer,
			"issue":             "kafka_cli_missing",
			"diagnose_mode":     "fallback_without_cli",
		}
		diag, derr := runAnalyzeWithOrchestrator(ctx, "kafka", fallbackCtx)
		if derr != nil {
			return &kafkaDiagnoseReport{
				BootstrapServer: opts.BootstrapServer,
				Findings: []kafkaFinding{
					{
						Priority: 1, Severity: "P1", Title: "Kafka 采集不可用（原生与回退均失败）",
						Evidence: "native go client failed; kafka cli missing; orchestrator fallback failed",
						Cause:    "网络/认证配置异常，或服务端 AI 诊断接口不可用",
						Verify:   "补充 --config 后重试；或检查服务端 /api/ai/diagnose 可用性",
					},
				},
				Errors: append(errs, "未找到 Kafka CLI，且服务端回退失败："+derr.Error()),
			}, nil
		}
		return &kafkaDiagnoseReport{
			BootstrapServer: opts.BootstrapServer,
			Findings: []kafkaFinding{
				{
					Priority: 2, Severity: "P2", Title: "未安装 Kafka CLI，已切换 AI 回退诊断",
					Evidence: "missing kafka-consumer-groups.sh / kafka-topics.sh",
					Cause:    "当前节点未安装 Kafka 客户端脚本，无法采集实时 group/topic 观测",
					Verify:   "安装 Kafka CLI，或继续使用当前 AI 回退结论",
				},
			},
			Errors:   append(errs, "未找到 Kafka CLI，已回退到通用诊断链路"),
			AIAnswer: strings.TrimSpace(diag.Answer),
		}, nil
	}
	snap := kafkaSnapshot{
		BootstrapServer: opts.BootstrapServer,
		Commands:        paths,
		Errors:          append([]string(nil), errs...),
	}
	if paths.ConsumerGroups != "" {
		groups, err := collectKafkaGroups(ctx, paths.ConsumerGroups, opts)
		if err != nil {
			snap.Errors = append(snap.Errors, shortKafkaError("consumer groups", err))
		}
		snap.Groups = groups
	}
	if paths.Topics != "" {
		topics, err := collectKafkaTopics(ctx, paths.Topics, opts)
		if err != nil {
			snap.Errors = append(snap.Errors, shortKafkaError("topics", err))
		}
		snap.Topics = topics
	}
	if paths.BrokerAPI != "" {
		if _, err := runKafkaCLI(ctx, paths.BrokerAPI, opts.Timeout, kafkaBaseArgs(opts)...); err != nil {
			snap.Errors = append(snap.Errors, shortKafkaError("broker api versions", err))
		}
	}
	report := buildKafkaReport(snap)
	if opts.AI {
		if answer, err := explainKafkaReportWithAI(ctx, report); err == nil {
			report.AIAnswer = answer
		} else {
			report.Errors = append(report.Errors, "AI 解释失败: "+err.Error())
		}
	}
	return report, nil
}

func resolveKafkaCommandPaths(commandDir string) (kafkaCommandPaths, []string) {
	var errs []string
	resolve := func(name string, required bool) string {
		if commandDir != "" {
			p := filepath.Join(commandDir, name)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p
			}
			if required {
				errs = append(errs, fmt.Sprintf("%s 不存在于 --command-dir=%s", name, commandDir))
			}
			return ""
		}
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
		if home := strings.TrimSpace(os.Getenv("KAFKA_HOME")); home != "" {
			p := filepath.Join(home, "bin", name)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p
			}
		}
		for _, dir := range []string{"/opt/kafka/bin", "/usr/local/kafka/bin", "/usr/share/kafka/bin"} {
			p := filepath.Join(dir, name)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p
			}
		}
		return ""
	}
	paths := kafkaCommandPaths{
		ConsumerGroups: resolve("kafka-consumer-groups.sh", true),
		Topics:         resolve("kafka-topics.sh", true),
		BrokerAPI:      resolve("kafka-broker-api-versions.sh", false),
	}
	return paths, errs
}

func kafkaBaseArgs(opts kafkaDiagnoseOptions) []string {
	args := []string{"--bootstrap-server", opts.BootstrapServer}
	if strings.TrimSpace(opts.ClientConfig) != "" {
		args = append(args, "--command-config", opts.ClientConfig)
	}
	return args
}

func collectKafkaGroups(ctx context.Context, consumerGroupsCmd string, opts kafkaDiagnoseOptions) ([]kafkaGroupSummary, error) {
	listArgs := append(kafkaBaseArgs(opts), "--list")
	out, err := runKafkaCLI(ctx, consumerGroupsCmd, opts.Timeout, listArgs...)
	if err != nil {
		return nil, err
	}
	names := parseKafkaGroupList(out)
	if len(names) > opts.Limit {
		names = names[:opts.Limit]
	}
	groups := make([]kafkaGroupSummary, 0, len(names))
	for _, name := range names {
		descArgs := append(kafkaBaseArgs(opts), "--describe", "--group", name)
		descOut, descErr := runKafkaCLI(ctx, consumerGroupsCmd, opts.Timeout, descArgs...)
		stateArgs := append(kafkaBaseArgs(opts), "--describe", "--state", "--group", name)
		stateOut, _ := runKafkaCLI(ctx, consumerGroupsCmd, opts.Timeout, stateArgs...)
		g := parseKafkaConsumerGroupDescribe(name, descOut)
		g.State = parseKafkaConsumerGroupState(stateOut)
		if descErr != nil {
			g.RawError = shortKafkaError("group "+name, descErr)
		}
		groups = append(groups, g)
	}
	return groups, nil
}

func collectKafkaTopics(ctx context.Context, topicsCmd string, opts kafkaDiagnoseOptions) ([]kafkaTopicSummary, error) {
	args := append(kafkaBaseArgs(opts), "--describe")
	out, err := runKafkaCLI(ctx, topicsCmd, opts.Timeout, args...)
	if err != nil {
		return nil, err
	}
	return parseKafkaTopicsDescribe(out), nil
}

func runKafkaCLI(ctx context.Context, path string, timeout time.Duration, args ...string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, path, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if cctx.Err() == context.DeadlineExceeded {
		return out.String(), fmt.Errorf("命令超时: %s %s", path, strings.Join(args, " "))
	}
	if err != nil {
		msg := strings.TrimSpace(out.String())
		if msg != "" {
			return msg, fmt.Errorf("%w: %s", err, msg)
		}
		return "", err
	}
	return out.String(), nil
}

func parseKafkaGroupList(out string) []string {
	seen := map[string]struct{}{}
	var groups []string
	for _, line := range strings.Split(out, "\n") {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "GROUP") || strings.HasPrefix(s, "Error:") || strings.HasPrefix(s, "WARN") {
			continue
		}
		fields := strings.Fields(s)
		if len(fields) == 0 {
			continue
		}
		name := fields[0]
		if _, ok := seen[name]; !ok {
			seen[name] = struct{}{}
			groups = append(groups, name)
		}
	}
	sort.Strings(groups)
	return groups
}

func parseKafkaConsumerGroupDescribe(groupName, out string) kafkaGroupSummary {
	g := kafkaGroupSummary{Name: groupName, MaxLagPartition: -1}
	memberSet := map[string]struct{}{}
	header := map[string]int{}
	for _, line := range strings.Split(out, "\n") {
		s := strings.TrimSpace(line)
		if s == "" {
			continue
		}
		low := strings.ToLower(s)
		if strings.Contains(low, "has no active members") || strings.Contains(low, "does not have any active members") {
			g.NoActiveMembers = true
			continue
		}
		fields := strings.Fields(s)
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "GROUP" {
			header = mapKafkaHeader(fields)
			continue
		}
		if len(header) == 0 {
			continue
		}
		row, ok := parseKafkaConsumerGroupRow(fields, header)
		if !ok {
			continue
		}
		if row.Group == "" {
			row.Group = groupName
		}
		g.Rows = append(g.Rows, row)
		g.TotalLag += row.Lag
		g.Partitions++
		if row.Lag > g.MaxPartitionLag {
			g.MaxPartitionLag = row.Lag
			g.MaxLagTopic = row.Topic
			g.MaxLagPartition = row.Partition
		}
		if row.ConsumerID != "" && row.ConsumerID != "-" {
			memberSet[row.ConsumerID] = struct{}{}
		}
	}
	g.ActiveMembers = len(memberSet)
	if g.ActiveMembers == 0 && g.TotalLag > 0 {
		g.NoActiveMembers = true
	}
	return g
}

func mapKafkaHeader(fields []string) map[string]int {
	m := map[string]int{}
	for i, f := range fields {
		m[strings.ToUpper(strings.ReplaceAll(f, "-", "_"))] = i
	}
	return m
}

func parseKafkaConsumerGroupRow(fields []string, header map[string]int) (kafkaConsumerGroupRow, bool) {
	get := func(keys ...string) string {
		for _, k := range keys {
			if idx, ok := header[k]; ok && idx >= 0 && idx < len(fields) {
				return fields[idx]
			}
		}
		return ""
	}
	group := get("GROUP")
	topic := get("TOPIC")
	partStr := get("PARTITION")
	lagStr := get("LAG")
	if group == "" || topic == "" || partStr == "" || lagStr == "" {
		return kafkaConsumerGroupRow{}, false
	}
	part, err := strconv.Atoi(partStr)
	if err != nil {
		return kafkaConsumerGroupRow{}, false
	}
	lag := parseKafkaLag(lagStr)
	return kafkaConsumerGroupRow{
		Group:         group,
		Topic:         topic,
		Partition:     part,
		CurrentOffset: get("CURRENT_OFFSET", "CURRENT-OFFSET"),
		LogEndOffset:  get("LOG_END_OFFSET", "LOG-END-OFFSET"),
		Lag:           lag,
		ConsumerID:    get("CONSUMER_ID", "CONSUMER-ID"),
		Host:          get("HOST"),
		ClientID:      get("CLIENT_ID", "CLIENT-ID"),
	}, true
}

func parseKafkaLag(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" {
		return 0
	}
	n, _ := strconv.ParseInt(s, 10, 64)
	if n < 0 {
		return 0
	}
	return n
}

func parseKafkaConsumerGroupState(out string) string {
	header := map[string]int{}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "GROUP" {
			header = mapKafkaHeader(fields)
			continue
		}
		if idx, ok := header["STATE"]; ok && idx < len(fields) {
			return fields[idx]
		}
	}
	return ""
}

func parseKafkaTopicsDescribe(out string) []kafkaTopicSummary {
	topics := map[string]*kafkaTopicSummary{}
	for _, line := range strings.Split(out, "\n") {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(strings.ToLower(s), "warning") {
			continue
		}
		fields := strings.Fields(s)
		var part kafkaTopicPartition
		var hasPartition bool
		var partitionCount int
		var topicName string
		for i := 0; i < len(fields); i++ {
			key := strings.TrimSuffix(fields[i], ":")
			if i+1 >= len(fields) {
				continue
			}
			val := strings.TrimSuffix(fields[i+1], ",")
			switch key {
			case "Topic":
				topicName = val
				part.Topic = val
			case "Partition":
				if n, err := strconv.Atoi(val); err == nil {
					part.Partition = n
					hasPartition = true
				}
			case "Leader":
				part.Leader = val
			case "Replicas":
				part.Replicas = splitKafkaCSV(val)
			case "Isr":
				part.ISR = splitKafkaCSV(val)
			case "PartitionCount":
				partitionCount, _ = strconv.Atoi(val)
			}
		}
		if topicName == "" {
			continue
		}
		ts := topics[topicName]
		if ts == nil {
			ts = &kafkaTopicSummary{Name: topicName}
			topics[topicName] = ts
		}
		if partitionCount > ts.PartitionCount {
			ts.PartitionCount = partitionCount
		}
		if hasPartition {
			if part.Leader == "" {
				part.Leader = "unknown"
			}
			ts.Partitions = append(ts.Partitions, part)
			if part.Partition+1 > ts.PartitionCount {
				ts.PartitionCount = part.Partition + 1
			}
			if isKafkaOfflineLeader(part.Leader) {
				ts.OfflinePartitions++
			}
			if len(part.Replicas) > 0 && len(part.ISR) > 0 && len(part.ISR) < len(part.Replicas) {
				ts.UnderReplicatedPartitions++
			}
		}
	}
	summaries := make([]kafkaTopicSummary, 0, len(topics))
	for _, t := range topics {
		sort.Slice(t.Partitions, func(i, j int) bool { return t.Partitions[i].Partition < t.Partitions[j].Partition })
		summaries = append(summaries, *t)
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].Name < summaries[j].Name })
	return summaries
}

func splitKafkaCSV(s string) []string {
	s = strings.TrimSpace(strings.Trim(s, "[]"))
	if s == "" || s == "-" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func isKafkaOfflineLeader(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "" || s == "-1" || s == "none" || s == "unknown"
}

func buildKafkaReport(s kafkaSnapshot) *kafkaDiagnoseReport {
	findings := diagnoseKafkaSnapshot(s)
	totalLag := int64(0)
	for _, g := range s.Groups {
		totalLag += g.TotalLag
	}
	offline, urp := 0, 0
	for _, t := range s.Topics {
		offline += t.OfflinePartitions
		urp += t.UnderReplicatedPartitions
	}
	return &kafkaDiagnoseReport{
		BootstrapServer: s.BootstrapServer,
		GroupsScanned:   len(s.Groups),
		TopicsScanned:   len(s.Topics),
		Findings:        findings,
		RawSummary: kafkaRawSummary{
			ConsumerGroupsListed:      len(s.Groups),
			TotalLag:                  totalLag,
			OfflinePartitions:         offline,
			UnderReplicatedPartitions: urp,
		},
		Errors: s.Errors,
	}
}

func diagnoseKafkaSnapshot(s kafkaSnapshot) []kafkaFinding {
	var findings []kafkaFinding
	for _, t := range s.Topics {
		if t.OfflinePartitions > 0 {
			findings = append(findings, kafkaFinding{
				Priority: 0, Severity: "P0", Resource: t.Name,
				Title:    fmt.Sprintf("%s 存在 %d 个 offline partition", t.Name, t.OfflinePartitions),
				Evidence: fmt.Sprintf("topic=%s offline_partitions=%d", t.Name, t.OfflinePartitions),
				Cause:    "分区无 leader，生产和消费会直接受影响",
				Verify:   fmt.Sprintf("kafka-topics.sh --bootstrap-server %s --describe --topic %s", s.BootstrapServer, t.Name),
			})
		}
		if t.UnderReplicatedPartitions > 0 {
			findings = append(findings, kafkaFinding{
				Priority: 1, Severity: "P1", Resource: t.Name,
				Title:    fmt.Sprintf("%s 存在 %d 个 under replicated partition", t.Name, t.UnderReplicatedPartitions),
				Evidence: fmt.Sprintf("topic=%s under_replicated_partitions=%d", t.Name, t.UnderReplicatedPartitions),
				Cause:    "副本同步落后或 broker 不健康，可能导致生产/消费抖动",
				Verify:   fmt.Sprintf("kafka-topics.sh --bootstrap-server %s --describe --topic %s", s.BootstrapServer, t.Name),
			})
		}
	}
	for _, g := range s.Groups {
		if g.TotalLag > 0 && g.MaxPartitionLag > 0 {
			share := float64(g.MaxPartitionLag) / float64(g.TotalLag)
			if g.TotalLag >= 10000 || (g.TotalLag >= 1000 && share >= 0.70) {
				findings = append(findings, kafkaFinding{
					Priority: 1, Severity: "P1", Resource: g.Name,
					Title:    fmt.Sprintf("%s 消费堆积 lag=%d", g.Name, g.TotalLag),
					Evidence: fmt.Sprintf("%s-%d lag=%d，占 %.0f%%", g.MaxLagTopic, g.MaxLagPartition, g.MaxPartitionLag, share*100),
					Cause:    "消费处理能力不足、热点 key、分区分配不均或下游依赖变慢",
					Verify:   fmt.Sprintf("kafka-consumer-groups.sh --bootstrap-server %s --describe --group %s", s.BootstrapServer, g.Name),
				})
			}
		}
		if g.NoActiveMembers && g.TotalLag > 0 {
			findings = append(findings, kafkaFinding{
				Priority: 2, Severity: "P2", Resource: g.Name,
				Title:    fmt.Sprintf("%s 有 lag 但无 active member", g.Name),
				Evidence: fmt.Sprintf("group=%s total_lag=%d active_members=0", g.Name, g.TotalLag),
				Cause:    "消费者进程未运行、认证失败、连接失败或部署副本为 0",
				Verify:   fmt.Sprintf("kafka-consumer-groups.sh --bootstrap-server %s --describe --group %s", s.BootstrapServer, g.Name),
			})
		}
		state := strings.ToLower(g.State)
		if state == "preparingrebalance" || state == "completingrebalance" {
			findings = append(findings, kafkaFinding{
				Priority: 2, Severity: "P2", Resource: g.Name,
				Title:    fmt.Sprintf("%s 正在频繁 rebalance", g.Name),
				Evidence: "group_state=" + g.State,
				Cause:    "consumer 心跳超时、实例频繁重启、max.poll.interval.ms 不足或网络抖动",
				Verify:   fmt.Sprintf("kafka-consumer-groups.sh --bootstrap-server %s --describe --state --group %s", s.BootstrapServer, g.Name),
			})
		}
		if g.TotalLag > 0 && g.ActiveMembers > 0 && g.Partitions >= g.ActiveMembers*4 {
			findings = append(findings, kafkaFinding{
				Priority: 3, Severity: "P3", Resource: g.Name,
				Title:    fmt.Sprintf("%s 消费者数量明显少于分区数", g.Name),
				Evidence: fmt.Sprintf("partitions=%d active_members=%d total_lag=%d", g.Partitions, g.ActiveMembers, g.TotalLag),
				Cause:    "并行度可能不足，或分区分配不均导致局部堆积",
				Verify:   fmt.Sprintf("kafka-consumer-groups.sh --bootstrap-server %s --describe --group %s", s.BootstrapServer, g.Name),
			})
		}
	}
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Priority == findings[j].Priority {
			return findings[i].Title < findings[j].Title
		}
		return findings[i].Priority < findings[j].Priority
	})
	if len(findings) > 3 {
		findings = findings[:3]
	}
	return findings
}

func formatKafkaDiagnoseText(r *kafkaDiagnoseReport) string {
	var b strings.Builder
	high := 0
	for _, f := range r.Findings {
		if f.Priority <= 1 {
			high++
		}
	}
	if len(r.Findings) == 0 {
		fmt.Fprintf(&b, "结论：未发现明显高优先级 Kafka 问题。\n\n")
	} else {
		first := r.Findings[0]
		if high > 0 {
			fmt.Fprintf(&b, "结论：发现 %d 个高优先级问题，最可能先处理 %s。\n\n", high, first.Title)
		} else {
			fmt.Fprintf(&b, "结论：发现 %d 个待关注问题，建议先看 %s。\n\n", len(r.Findings), first.Title)
		}
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
	fmt.Fprintf(&b, "集群副本状态：offline partition=%d；under replicated partition=%d\n",
		r.RawSummary.OfflinePartitions, r.RawSummary.UnderReplicatedPartitions)
	fmt.Fprintf(&b, "扫描范围：consumer group=%d；topic=%d；total lag=%d\n",
		r.GroupsScanned, r.TopicsScanned, r.RawSummary.TotalLag)
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

func explainKafkaReportWithAI(ctx context.Context, report *kafkaDiagnoseReport) (string, error) {
	eng, err := bootstrap()
	if err != nil {
		return "", err
	}
	b, _ := json.Marshal(report)
	res, err := eng.Analyze(ctx, "kafka", map[string]string{
		"issue":    "diagnose",
		"snapshot": string(b),
	}, !noRAG)
	if err != nil {
		return "", err
	}
	if res == nil {
		return "", nil
	}
	return res.Answer, nil
}

func shortKafkaError(scope string, err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	msg = strings.ReplaceAll(msg, "\n", " ")
	if len(msg) > 240 {
		msg = msg[:240] + "..."
	}
	return scope + ": " + msg
}

type kafkaSCRAMClient struct {
	hashGenerator scram.HashGeneratorFcn
	client        *scram.Client
	conversation  *scram.ClientConversation
}

func (x *kafkaSCRAMClient) Begin(userName, password, authzID string) error {
	client, err := x.hashGenerator.NewClient(userName, password, authzID)
	if err != nil {
		return err
	}
	x.client = client
	x.conversation = client.NewConversation()
	return nil
}

func (x *kafkaSCRAMClient) Step(challenge string) (string, error) {
	return x.conversation.Step(challenge)
}

func (x *kafkaSCRAMClient) Done() bool {
	return x.conversation.Done()
}

func collectKafkaSnapshotByGo(ctx context.Context, opts kafkaDiagnoseOptions) (kafkaSnapshot, error) {
	props, _ := loadKafkaClientProperties(opts.ClientConfig)
	cfg, err := buildSaramaConfig(opts, props)
	if err != nil {
		return kafkaSnapshot{}, err
	}
	brokers := splitKafkaCSV(opts.BootstrapServer)
	if len(brokers) == 0 {
		return kafkaSnapshot{}, errors.New("bootstrap-server 为空")
	}
	client, err := sarama.NewClient(brokers, cfg)
	if err != nil {
		return kafkaSnapshot{}, err
	}
	defer client.Close()
	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		return kafkaSnapshot{}, err
	}
	defer admin.Close()

	snap := kafkaSnapshot{BootstrapServer: opts.BootstrapServer}
	topics, terr := collectKafkaTopicsByGo(client)
	if terr != nil {
		snap.Errors = append(snap.Errors, shortKafkaError("native topics", terr))
	} else {
		snap.Topics = topics
	}
	groups, gerr := collectKafkaGroupsByGo(ctx, client, admin, opts.Limit)
	if gerr != nil {
		snap.Errors = append(snap.Errors, shortKafkaError("native groups", gerr))
	} else {
		snap.Groups = groups
	}
	if len(snap.Topics) == 0 && len(snap.Groups) == 0 {
		return kafkaSnapshot{}, errors.New(strings.Join(snap.Errors, "; "))
	}
	return snap, nil
}

func collectKafkaTopicsByGo(client sarama.Client) ([]kafkaTopicSummary, error) {
	names, err := client.Topics()
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	out := make([]kafkaTopicSummary, 0, len(names))
	for _, topic := range names {
		partitions, err := client.Partitions(topic)
		if err != nil {
			continue
		}
		ts := kafkaTopicSummary{Name: topic, PartitionCount: len(partitions)}
		for _, p := range partitions {
			leader, _ := client.Leader(topic, p)
			replicas, _ := client.Replicas(topic, p)
			isr, _ := client.InSyncReplicas(topic, p)
			part := kafkaTopicPartition{
				Topic:     topic,
				Partition: int(p),
				Leader:    fmt.Sprintf("%d", leader.ID()),
				Replicas:  int32ToStringSlice(replicas),
				ISR:       int32ToStringSlice(isr),
			}
			if leader == nil {
				part.Leader = "unknown"
			}
			if isKafkaOfflineLeader(part.Leader) {
				ts.OfflinePartitions++
			}
			if len(part.Replicas) > 0 && len(part.ISR) < len(part.Replicas) {
				ts.UnderReplicatedPartitions++
			}
			ts.Partitions = append(ts.Partitions, part)
		}
		sort.Slice(ts.Partitions, func(i, j int) bool { return ts.Partitions[i].Partition < ts.Partitions[j].Partition })
		out = append(out, ts)
	}
	return out, nil
}

func collectKafkaGroupsByGo(ctx context.Context, client sarama.Client, admin sarama.ClusterAdmin, limit int) ([]kafkaGroupSummary, error) {
	groupMap, err := admin.ListConsumerGroups()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(groupMap))
	for name := range groupMap {
		names = append(names, name)
	}
	sort.Strings(names)
	if limit > 0 && len(names) > limit {
		names = names[:limit]
	}

	out := make([]kafkaGroupSummary, 0, len(names))
	for _, name := range names {
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		default:
		}
		descList, _ := admin.DescribeConsumerGroups([]string{name})
		g := kafkaGroupSummary{Name: name, MaxLagPartition: -1}
		if len(descList) > 0 {
			g.State = descList[0].State
			g.ActiveMembers = len(descList[0].Members)
		}
		offsets, err := admin.ListConsumerGroupOffsets(name, nil)
		if err != nil {
			g.RawError = shortKafkaError("group "+name, err)
			out = append(out, g)
			continue
		}
		for topic, pmap := range offsets.Blocks {
			for part, block := range pmap {
				latest, lerr := client.GetOffset(topic, part, sarama.OffsetNewest)
				if lerr != nil {
					continue
				}
				current := block.Offset
				if current < 0 {
					current = 0
				}
				lag := latest - current
				if lag < 0 {
					lag = 0
				}
				row := kafkaConsumerGroupRow{
					Group:         name,
					Topic:         topic,
					Partition:     int(part),
					CurrentOffset: fmt.Sprintf("%d", current),
					LogEndOffset:  fmt.Sprintf("%d", latest),
					Lag:           lag,
				}
				g.Rows = append(g.Rows, row)
				g.Partitions++
				g.TotalLag += lag
				if lag > g.MaxPartitionLag {
					g.MaxPartitionLag = lag
					g.MaxLagTopic = topic
					g.MaxLagPartition = int(part)
				}
			}
		}
		if g.ActiveMembers == 0 && g.TotalLag > 0 {
			g.NoActiveMembers = true
		}
		out = append(out, g)
	}
	return out, nil
}

func int32ToStringSlice(v []int32) []string {
	if len(v) == 0 {
		return nil
	}
	out := make([]string, 0, len(v))
	for _, n := range v {
		out = append(out, strconv.Itoa(int(n)))
	}
	return out
}

func loadKafkaClientProperties(path string) (map[string]string, error) {
	props := map[string]string{}
	if strings.TrimSpace(path) == "" {
		return props, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(string(b), "\n") {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		idx := strings.Index(s, "=")
		if idx <= 0 {
			continue
		}
		k := strings.TrimSpace(s[:idx])
		v := strings.TrimSpace(s[idx+1:])
		props[k] = v
	}
	return props, nil
}

func buildSaramaConfig(opts kafkaDiagnoseOptions, props map[string]string) (*sarama.Config, error) {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.MaxVersion
	cfg.Net.DialTimeout = opts.Timeout
	cfg.Net.ReadTimeout = opts.Timeout
	cfg.Net.WriteTimeout = opts.Timeout
	cfg.Metadata.Timeout = opts.Timeout
	cfg.Admin.Timeout = opts.Timeout

	sec := strings.ToUpper(strings.TrimSpace(props["security.protocol"]))
	if strings.Contains(sec, "SSL") {
		cfg.Net.TLS.Enable = true
		cfg.Net.TLS.Config = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	if strings.Contains(sec, "SASL") || strings.TrimSpace(props["sasl.mechanism"]) != "" {
		cfg.Net.SASL.Enable = true
		cfg.Net.SASL.User = strings.TrimSpace(props["sasl.username"])
		cfg.Net.SASL.Password = strings.TrimSpace(props["sasl.password"])
		mech := strings.ToUpper(strings.TrimSpace(props["sasl.mechanism"]))
		switch mech {
		case "PLAIN", "":
			cfg.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		case "SCRAM-SHA-256":
			cfg.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
			cfg.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
				return &kafkaSCRAMClient{hashGenerator: sha256.New}
			}
		default:
			return nil, fmt.Errorf("暂不支持的 sasl.mechanism: %s", mech)
		}
	}
	return cfg, nil
}
