package handlers

import (
	"fmt"
	"strings"

	"ft-backend/services"
)

func buildLinuxPerformanceEvidencePromptWithSkill(topic string, kv map[string]string, matched *services.RegisteredSkill) string {
	var b strings.Builder
	user := map[string]string{}
	evidence := map[string]string{}
	if kv != nil {
		for k, v := range kv {
			switch {
			case k == "diagnosis_style", k == "prior_answer_round1":
				continue
			case isCollectedEvidenceKey(k):
				evidence[k] = v
			default:
				user[k] = v
			}
		}
	}
	b.WriteString("你是资深 Linux 系统 SRE。下方 JSON 来自 ai-sre probe linux 自动只读采集，是唯一事实来源。\n\n")
	b.WriteString("硬性要求：\n")
	b.WriteString("1) 根因必须可由 linux_perf_probe_json 字段直接支撑；禁止编造 PID、设备、挂载点或内核日志。\n")
	b.WriteString("2) 禁止要求用户执行 top、free、iostat、ps、vmstat、probe 或 shell 补采集。\n")
	b.WriteString("3) 内存泄露只能作风险预警，禁止单次采样断言必然泄露。\n")
	b.WriteString("4) 证据不足时说明 ai-sre 无权限读取的项及对判断的影响。\n")
	b.WriteString("5) 输出为纯文本（非 Markdown），固定小节标题逐字使用：\n")
	b.WriteString("【根因判断】\n")
	b.WriteString("【关键证据】\n")
	b.WriteString("【系统与网络】\n")
	b.WriteString("【进程与资源风险】\n")
	b.WriteString("【修复建议】\n")
	b.WriteString("【后续观测】\n")
	b.WriteString("6) 证据条目用 - 开头，引用 JSON 字段（cpu/load/memory/network/connections/system/process_hotspots/leak_risks），禁止逐条罗列排序榜或 Top N 列表。\n")
	b.WriteString("7) 综合分析 network、connections、system、process_hotspots：覆盖网络吞吐/错误、TCP 连接态、文件描述符、线程与内存压力，但只写结论性要点。\n")
	if matched != nil {
		b.WriteString("\n")
		writeSkillSectionPlain(&b, matched)
	}
	b.WriteString("\ntopic=" + topic + "\n\n")
	if len(user) > 0 {
		b.WriteString("用户上下文：\n")
		for _, k := range sortedStringKeys(user) {
			b.WriteString(fmt.Sprintf("- %s=%s\n", k, user[k]))
		}
		b.WriteString("\n")
	}
	if len(evidence) == 0 {
		b.WriteString("（无 linux_perf_probe_json）\n")
		return b.String()
	}
	b.WriteString("=== 只读采集（原文） ===\n")
	for _, k := range sortedEvidenceKeysForPrompt(evidence) {
		body := evidence[k]
		if len(body) > 16000 {
			body = body[:16000] + "\n... (truncated)"
		}
		b.WriteString(fmt.Sprintf("\n--- %s ---\n%s\n", k, body))
	}
	return b.String()
}

func isLinuxPerformanceTopic(topic string) bool {
	return strings.ToLower(strings.TrimSpace(topic)) == "linux"
}
