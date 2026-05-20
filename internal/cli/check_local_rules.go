package cli

import (
	"encoding/json"
	"fmt"
	"strings"
)

// tryLocalCheckRules evaluates probe evidence locally before calling server AI.
func tryLocalCheckRules(topic string, ctx map[string]string) (*diagnoseResponse, bool) {
	t := normalizeCheckTopicAlias(topic)
	switch t {
	case "redis":
		return tryLocalRedisRules(ctx)
	default:
		return nil, false
	}
}

func tryLocalRedisRules(ctx map[string]string) (*diagnoseResponse, bool) {
	raw := strings.TrimSpace(ctx["redis_diagnose_json"])
	if raw == "" {
		raw = strings.TrimSpace(ctx["redis_probe_json"])
	}
	if raw == "" {
		return nil, false
	}
	var report RedisProbeReport
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		return nil, false
	}
	if report.AuthRequired {
		return &diagnoseResponse{
			Source: "local-rule",
			Answer: formatCheckStructuredText(checkStructuredResult{
				RootCause:       "Redis 需要 AUTH，但未提供密码",
				Evidence:        []string{"PING 返回 NOAUTH", "目标: " + report.Address},
				Impact:          "客户端无法读写 Redis",
				Recommendations: []string{"在 TTY 下重试以交互输入密码，或通过环境变量 AI_SRE_REDIS_PASSWORD 注入（仅本次会话）"},
				UsedAI:          false,
				EvidenceLevel:   "partial",
			}),
			SkillName: "redis-auth-required",
		}, true
	}
	for _, f := range report.Findings {
		lower := strings.ToLower(f)
		if strings.Contains(lower, "maxclients") || strings.Contains(lower, "rejected_connections") {
			return &diagnoseResponse{
				Source: "local-rule",
				Answer: formatCheckStructuredText(checkStructuredResult{
					RootCause:       "Redis 连接数已达上限或存在大量拒绝连接",
					Evidence:        append([]string{f}, report.Findings...),
					Impact:          "新连接被拒绝，应用可能出现超时",
					Recommendations: []string{"检查 maxclients 与 connected_clients", "排查连接泄漏或短连接风暴", "必要时调高 maxclients 并重启"},
					UsedAI:          false,
					EvidenceLevel:   evidenceCompletenessForContext(ctx),
				}),
				SkillName: "redis-maxclients",
			}, true
		}
	}
	if diag, ok := tryLocalRedisHealthyRule(&report, ctx); ok {
		return diag, true
	}
	return nil, false
}

func redisProbeHasActionableFinding(report *RedisProbeReport) bool {
	if report == nil {
		return true
	}
	if len(report.Errors) > 0 {
		return true
	}
	for _, f := range report.Findings {
		lower := strings.ToLower(f)
		if strings.Contains(lower, "rejected_connections") ||
			strings.Contains(lower, "evicted_keys") ||
			strings.Contains(lower, "偏高") {
			return true
		}
	}
	if atoi64OrZero(fmt.Sprint(report.Clients["rejected_connections"])) > 0 {
		return true
	}
	if atoi64OrZero(fmt.Sprint(report.Memory["evicted_keys"])) > 0 {
		return true
	}
	if atoi64OrZero(fmt.Sprint(report.Stats["total_error_replies"])) > 0 {
		return true
	}
	return false
}

func tryLocalRedisHealthyRule(report *RedisProbeReport, ctx map[string]string) (*diagnoseResponse, bool) {
	if report == nil || redisProbeHasActionableFinding(report) {
		return nil, false
	}
	usedMem := atoi64OrZero(fmt.Sprint(report.Memory["used_memory"]))
	if usedMem <= 0 {
		return nil, false
	}
	frag := atofOrZero(fmt.Sprint(report.Memory["mem_fragmentation_ratio"]))
	allocFrag := atofOrZero(fmt.Sprint(report.Memory["allocator_frag_ratio"]))
	usedHuman := strings.TrimSpace(fmt.Sprint(report.Memory["used_memory_human"]))
	if usedHuman == "" {
		usedHuman = fmt.Sprintf("%d 字节", usedMem)
	}
	maxMem := strings.TrimSpace(fmt.Sprint(report.Memory["maxmemory"]))
	ops := strings.TrimSpace(fmt.Sprint(report.Stats["instantaneous_ops_per_sec"]))
	uptime := strings.TrimSpace(fmt.Sprint(report.Stats["uptime_in_seconds"]))
	cmds := strings.TrimSpace(fmt.Sprint(report.Stats["total_commands_processed"]))
	connected := strings.TrimSpace(fmt.Sprint(report.Clients["connected_clients"]))
	rejected := strings.TrimSpace(fmt.Sprint(report.Clients["rejected_connections"]))
	totalConn := strings.TrimSpace(fmt.Sprint(report.Clients["total_connections_received"]))

	root := "当前实例处于空闲/健康状态，无故障触发条件。"
	if frag > 5 && usedMem < 50*1024*1024 {
		root += fmt.Sprintf(" mem_fragmentation_ratio=%.2f 在 used_memory=%s 的极低占用下属于 jemalloc 正常分配粒度现象，并非内存压力或连接拒绝根因。", frag, usedHuman)
		if allocFrag > 0 && allocFrag < frag {
			root += fmt.Sprintf(" allocator_frag_ratio=%.2f 表明碎片主要来自分配器内部开销。", allocFrag)
		}
	}

	evidence := []string{
		fmt.Sprintf("used_memory=%s", usedHuman),
		fmt.Sprintf("mem_fragmentation_ratio=%.2f", frag),
		fmt.Sprintf("rejected_connections=%s", rejected),
		fmt.Sprintf("connected_clients=%s", connected),
		fmt.Sprintf("total_connections_received=%s", totalConn),
		fmt.Sprintf("instantaneous_ops_per_sec=%s", ops),
		fmt.Sprintf("total_commands_processed=%s", cmds),
		fmt.Sprintf("uptime_in_seconds=%s", uptime),
	}
	if maxMem != "" && maxMem != "0" {
		evidence = append(evidence, "maxmemory="+maxMem)
	}
	if len(report.Findings) > 0 {
		evidence = append(evidence, report.Findings[0])
	}

	recs := []string{
		"无需任何操作；当前实例健康。",
		"禁止在低占用高碎片场景启用 activedefrag 或执行 MEMORY PURGE，碎片率会随数据量增加自然改善。",
	}
	if rejected != "0" {
		recs = append(recs, "若后续出现连接拒绝，优先检查监听 backlog（ss -tlnp Recv-Q）与瞬时连接峰值，而非直接重启。")
	}

	return &diagnoseResponse{
		Source: "local-rule",
		Answer: formatCheckStructuredText(checkStructuredResult{
			RootCause:       root,
			Evidence:        evidence,
			Impact:          "当前无业务影响；指标正常。",
			Recommendations: recs,
			UsedAI:          false,
			EvidenceLevel:   evidenceCompletenessForContext(ctx),
		}),
		SkillName: "redis-healthy-idle",
	}, true
}
