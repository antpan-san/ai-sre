package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

var errRedisAuthRequired = errors.New("redis_auth_required")

// RedisProbeReport is the structured output of ai-sre probe redis --json.
type RedisProbeReport struct {
	Address              string            `json:"address"`
	AuthRequired         bool              `json:"auth_required"`
	RedisVersion         string            `json:"redis_version,omitempty"`
	Role                 string            `json:"role,omitempty"`
	Mode                 string            `json:"mode,omitempty"`
	Memory               map[string]any    `json:"memory,omitempty"`
	Clients              map[string]any    `json:"clients,omitempty"`
	Stats                map[string]any    `json:"stats,omitempty"`
	Persistence          map[string]any    `json:"persistence,omitempty"`
	Replication          map[string]any    `json:"replication,omitempty"`
	Cluster              map[string]any    `json:"cluster,omitempty"`
	Slowlog              map[string]any    `json:"slowlog,omitempty"`
	Latency              map[string]any    `json:"latency,omitempty"`
	Commandstats         string            `json:"commandstats,omitempty"`
	Errorstats           string            `json:"errorstats,omitempty"`
	Config               map[string]any    `json:"config,omitempty"`
	Findings             []string          `json:"findings,omitempty"`
	EvidenceCompleteness map[string]bool   `json:"evidence_completeness,omitempty"`
	Errors               []string          `json:"errors,omitempty"`
}

type redisProbeOptions struct {
	Address  string
	Password string
	Timeout  time.Duration
	JSON     bool
}

type redisClient struct {
	conn net.Conn
	r    *bufio.Reader
	dead time.Time
}

func CollectRedisProbe(opts redisProbeOptions) *RedisProbeReport {
	report := &RedisProbeReport{
		Address:              strings.TrimSpace(opts.Address),
		EvidenceCompleteness: map[string]bool{},
	}
	if report.Address == "" {
		report.Errors = append(report.Errors, "地址为空")
		return report
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 8 * time.Second
	}
	c, err := dialRedisClient(opts.Address, opts.Timeout)
	if err != nil {
		report.Errors = append(report.Errors, "连接失败: "+err.Error())
		return report
	}
	defer c.close()

	if err := c.doSimple("PING"); err != nil {
		if isRedisNOAUTH(err) && opts.Password == "" {
			report.AuthRequired = true
			report.Errors = append(report.Errors, "需要 Redis AUTH")
			return report
		}
		report.Errors = append(report.Errors, "PING 失败: "+err.Error())
		return report
	}
	report.EvidenceCompleteness["ping"] = true

	if opts.Password != "" {
		if err := c.auth(opts.Password); err != nil {
			report.Errors = append(report.Errors, "AUTH 失败: "+err.Error())
			return report
		}
	}

	infoAll, err := c.doBulk("INFO", "all")
	if err != nil {
		if isRedisNOAUTH(err) {
			report.AuthRequired = true
			if opts.Password == "" {
				report.Errors = append(report.Errors, "需要 Redis AUTH")
				return report
			}
		}
		report.Errors = append(report.Errors, "INFO all 失败: "+err.Error())
		return report
	}
	report.EvidenceCompleteness["info_all"] = true
	kv := parseRedisInfo(infoAll)
	fillRedisProbeFromInfo(report, kv)

	report.Commandstats, _ = c.doBulkOptional(report, "commandstats", "INFO", "commandstats")
	report.Errorstats, _ = c.doBulkOptional(report, "errorstats", "INFO", "errorstats")

	report.Slowlog = map[string]any{}
	if n, err := c.doBulk("SLOWLOG", "LEN"); err == nil {
		report.Slowlog["len"] = strings.TrimSpace(n)
		report.EvidenceCompleteness["slowlog"] = true
		if entries, err := c.doBulk("SLOWLOG", "GET", "20"); err == nil {
			report.Slowlog["entries"] = entries
		}
	} else {
		recordRedisPermission(report, "slowlog", err)
	}

	report.Latency = map[string]any{}
	if doc, err := c.doBulk("LATENCY", "DOCTOR"); err == nil {
		report.Latency["doctor"] = doc
		report.EvidenceCompleteness["latency"] = true
	} else {
		recordRedisPermission(report, "latency_doctor", err)
	}
	if lat, err := c.doBulk("LATENCY", "LATEST"); err == nil {
		report.Latency["latest"] = lat
	} else {
		recordRedisPermission(report, "latency_latest", err)
	}

	report.Cluster = map[string]any{}
	if kv["cluster_enabled"] == "0" || kv["cluster_enabled"] == "" {
		report.Cluster["cluster_enabled"] = "0"
		report.EvidenceCompleteness["cluster"] = true
	} else {
		if ci, err := c.doBulk("CLUSTER", "INFO"); err == nil {
			report.Cluster["info"] = ci
			report.EvidenceCompleteness["cluster"] = true
		} else {
			recordRedisPermission(report, "cluster_info", err)
		}
		if cn, err := c.doBulk("CLUSTER", "NODES"); err == nil {
			report.Cluster["nodes_summary"] = summarizeClusterNodes(cn)
		}
	}

	report.Config = map[string]any{}
	for _, key := range []string{"maxmemory", "maxmemory-policy", "appendonly", "save"} {
		if val, err := c.doBulk("CONFIG", "GET", key); err == nil {
			report.Config[key] = val
			report.EvidenceCompleteness["config"] = true
		} else {
			recordRedisPermission(report, "config_"+key, err)
		}
	}

	redisBuildFindings(report)
	return report
}

func collectRedisProbeJSON(ctx context.Context, flags map[string]string) (string, *RedisProbeReport, error) {
	addr := redisTargetFromFlags(flags)
	if addr == "" {
		return "", nil, fmt.Errorf("redis 地址未配置")
	}
	pw := strings.TrimSpace(flags["password"])
	opts := redisProbeOptions{Address: addr, Password: pw, Timeout: 20 * time.Second}
	report := CollectRedisProbe(opts)
	if report.AuthRequired && pw == "" {
		if !isStdinTTY() {
			b, _ := json.Marshal(report)
			return string(b), report, errRedisAuthRequired
		}
		entered, err := promptRedisPassword(addr)
		if err != nil {
			return "", report, err
		}
		opts.Password = entered
		report = CollectRedisProbe(opts)
	}
	b, err := json.Marshal(report)
	if err != nil {
		return "", report, err
	}
	return string(b), report, nil
}

func redisTargetFromFlags(flags map[string]string) string {
	if flags == nil {
		return ""
	}
	tgt := strings.TrimSpace(flags["addr"])
	if tgt == "" {
		tgt = strings.TrimSpace(flags["target"])
	}
	if tgt == "" {
		host := strings.TrimSpace(flags["host"])
		if host == "" {
			return ""
		}
		port := strings.TrimSpace(flags["port"])
		if port == "" {
			port = "6379"
		}
		tgt = fmt.Sprintf("%s:%s", host, port)
	}
	return normalizeCheckTargetValue("redis", tgt)
}

func promptRedisPassword(addr string) (string, error) {
	return promptSecret(fmt.Sprintf("Redis %s 密码", addr))
}

func dialRedisClient(addr string, timeout time.Duration) (*redisClient, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}
	return &redisClient{conn: conn, r: bufio.NewReader(conn), dead: time.Now().Add(timeout)}, nil
}

func (c *redisClient) close() { _ = c.conn.Close() }

func (c *redisClient) refreshDeadline() { _ = c.conn.SetDeadline(c.dead) }

func (c *redisClient) auth(password string) error {
	if err := redisWriteCommand(c.conn, "AUTH", password); err != nil {
		return err
	}
	c.refreshDeadline()
	line, err := c.r.ReadString('\n')
	if err != nil {
		return err
	}
	if !strings.HasPrefix(line, "+OK") {
		return fmt.Errorf("%s", strings.TrimSpace(line))
	}
	return nil
}

func (c *redisClient) doSimple(args ...string) error {
	if err := redisWriteCommand(c.conn, args...); err != nil {
		return err
	}
	c.refreshDeadline()
	line, err := c.r.ReadString('\n')
	if err != nil {
		return err
	}
	if strings.HasPrefix(line, "-") {
		return fmt.Errorf("%s", strings.TrimSpace(line))
	}
	return nil
}

func (c *redisClient) doBulk(args ...string) (string, error) {
	if err := redisWriteCommand(c.conn, args...); err != nil {
		return "", err
	}
	c.refreshDeadline()
	return redisReadBulkString(c.r)
}

func (c *redisClient) doBulkOptional(report *RedisProbeReport, section string, args ...string) (string, error) {
	s, err := c.doBulk(args...)
	if err != nil {
		recordRedisPermission(report, section, err)
		return "", err
	}
	report.EvidenceCompleteness[section] = true
	return s, nil
}

func isRedisNOAUTH(err error) bool {
	return err != nil && strings.Contains(strings.ToUpper(err.Error()), "NOAUTH")
}

func recordRedisPermission(report *RedisProbeReport, section string, err error) {
	if report.Config == nil {
		report.Config = map[string]any{}
	}
	report.Config[section+"_error"] = permissionLabel(err)
}

func permissionLabel(err error) string {
	if err == nil {
		return ""
	}
	up := strings.ToUpper(err.Error())
	if strings.Contains(up, "NOPERM") || strings.Contains(up, "NOAUTH") || strings.Contains(up, "ERR") {
		return "permission_denied"
	}
	return err.Error()
}

func fillRedisProbeFromInfo(report *RedisProbeReport, kv map[string]string) {
	report.RedisVersion = kv["redis_version"]
	report.Role = kv["role"]
	report.Mode = kv["redis_mode"]
	report.Memory = map[string]any{
		"used_memory":              kv["used_memory"],
		"used_memory_human":        kv["used_memory_human"],
		"used_memory_peak":         kv["used_memory_peak"],
		"maxmemory":                kv["maxmemory"],
		"maxmemory_policy":         kv["maxmemory_policy"],
		"evicted_keys":             kv["evicted_keys"],
		"mem_fragmentation_ratio":  kv["mem_fragmentation_ratio"],
		"allocator_frag_ratio":     kv["allocator_frag_ratio"],
	}
	report.Clients = map[string]any{
		"connected_clients":          kv["connected_clients"],
		"blocked_clients":            kv["blocked_clients"],
		"tracking_clients":           kv["tracking_clients"],
		"maxclients":                 kv["maxclients"],
		"rejected_connections":       kv["rejected_connections"],
		"total_connections_received": kv["total_connections_received"],
	}
	report.Stats = map[string]any{
		"instantaneous_ops_per_sec": kv["instantaneous_ops_per_sec"],
		"total_commands_processed":  kv["total_commands_processed"],
		"total_error_replies":       kv["total_error_replies"],
		"tcp_port":                  kv["tcp_port"],
		"uptime_in_seconds":         kv["uptime_in_seconds"],
	}
	report.Persistence = map[string]any{
		"rdb_last_bgsave_status":    kv["rdb_last_bgsave_status"],
		"rdb_bgsave_in_progress":    kv["rdb_bgsave_in_progress"],
		"rdb_last_bgsave_time_sec":  kv["rdb_last_bgsave_time_sec"],
		"aof_enabled":               kv["aof_enabled"],
		"aof_last_bgrewrite_status": kv["aof_last_bgrewrite_status"],
		"aof_rewrite_in_progress":   kv["aof_rewrite_in_progress"],
		"latest_fork_usec":          kv["latest_fork_usec"],
	}
	report.Replication = map[string]any{
		"role":                        kv["role"],
		"connected_slaves":            kv["connected_slaves"],
		"master_link_status":          kv["master_link_status"],
		"master_last_io_seconds_ago":  kv["master_last_io_seconds_ago"],
		"repl_backlog_active":         kv["repl_backlog_active"],
		"repl_backlog_size":           kv["repl_backlog_size"],
	}
}

func summarizeClusterNodes(raw string) string {
	lines := strings.Split(raw, "\n")
	if len(lines) > 12 {
		return strings.Join(lines[:12], "\n") + fmt.Sprintf("\n... (%d lines total)", len(lines))
	}
	return raw
}

func redisBuildFindings(report *RedisProbeReport) {
	if report == nil {
		return
	}
	if n := atoi64OrZero(fmt.Sprint(report.Clients["rejected_connections"])); n > 0 {
		report.Findings = append(report.Findings, fmt.Sprintf("rejected_connections=%d", n))
	}
	if n := atoi64OrZero(fmt.Sprint(report.Memory["evicted_keys"])); n > 0 {
		report.Findings = append(report.Findings, fmt.Sprintf("evicted_keys=%d", n))
	}
	if n := atoiOrZero(fmt.Sprint(report.Clients["connected_clients"])); n > 10000 {
		report.Findings = append(report.Findings, fmt.Sprintf("connected_clients=%d 偏高", n))
	}
	if len(report.Findings) == 0 && len(report.Errors) == 0 {
		report.Findings = append(report.Findings, "未发现明显高优先级 Redis 异常")
	}
}

func formatRedisProbeText(report *RedisProbeReport) string {
	if report == nil {
		return ""
	}
	var b strings.Builder
	if len(report.Findings) > 0 {
		fmt.Fprintf(&b, "结论：%s\n\n", report.Findings[0])
		for i, f := range report.Findings {
			fmt.Fprintf(&b, "%d. %s\n", i+1, f)
		}
	}
	fmt.Fprintf(&b, "\nRedis %s role=%s version=%s mode=%s\n",
		report.Address, report.Role, report.RedisVersion, report.Mode)
	if len(report.Errors) > 0 {
		b.WriteString("\n采集提示：\n")
		for _, e := range report.Errors {
			fmt.Fprintf(&b, "- %s\n", e)
		}
	}
	return b.String()
}
