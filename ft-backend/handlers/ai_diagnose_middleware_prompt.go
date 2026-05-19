package handlers

import (
	"fmt"
	"strings"

	"ft-backend/services"
)

func buildMiddlewareEvidencePromptWithSkill(topic string, kv map[string]string, matched *services.RegisteredSkill) string {
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
	b.WriteString("你是资深中间件 SRE。下方 JSON/文本来自 ai-sre probe 自动只读采集，是唯一事实来源。\n\n")
	b.WriteString("硬性要求：\n")
	b.WriteString("1) 根因必须可由采集 JSON 中的字段直接支撑；禁止夸大因果（例如：used_memory 很低时不得把 mem_fragmentation_ratio 单独说成 rejected_connections 主因）。\n")
	b.WriteString("2) rejected_connections 很少（个位数）时，应描述为「少量拒绝」并优先排查监听/backlog/瞬时连接，不得建议「立即重启」作为首选。\n")
	b.WriteString("3) 禁止要求用户执行 redis-cli、probe、shell 补采集。\n")
	b.WriteString("4) 输出必须是**纯文本**（非 Markdown），且仅包含以下三个小节，标题逐字使用中文方括号：\n")
	b.WriteString("【根因与触发条件】\n")
	b.WriteString("【关键指标证据】\n")
	b.WriteString("【缓解与根治建议】\n")
	b.WriteString("5) 证据条目请用 - 开头列举，引用 JSON 字段名与取值，不要代码块。\n")
	b.WriteString("6) 信息不足时只在【根因与触发条件】说明缺失项，禁止编造。\n")
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
		b.WriteString("（无采集 JSON）\n")
		return b.String()
	}
	b.WriteString("=== 只读采集（原文） ===\n")
	for _, k := range sortedEvidenceKeysForPrompt(evidence) {
		body := evidence[k]
		if len(body) > 14000 {
			body = body[:14000] + "\n... (truncated)"
		}
		b.WriteString(fmt.Sprintf("\n--- %s ---\n%s\n", k, body))
	}
	return b.String()
}

func writeSkillSectionPlain(b *strings.Builder, matched *services.RegisteredSkill) {
	if matched == nil {
		return
	}
	pack := matched.Pack
	b.WriteString("【适用技能包】 ")
	b.WriteString(pack.Name)
	if pack.DisplayName != "" {
		b.WriteString(" — ")
		b.WriteString(pack.DisplayName)
	}
	b.WriteString("\n")
	if strings.TrimSpace(pack.ExtraGuidance) != "" {
		b.WriteString("约束：")
		b.WriteString(pack.ExtraGuidance)
		b.WriteString("\n")
	}
}
