package cli

import (
	"encoding/json"
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
	return nil, false
}
