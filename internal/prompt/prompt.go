package prompt

import (
	"fmt"
	"strings"

	"github.com/panshuai/ai-sre/internal/skill"
)

// BuildAnalyze builds the main diagnosis prompt from a skill pack + user context.
func BuildAnalyze(sp *skill.Pack, topic string, context map[string]string, ragContext string) string {
	var b strings.Builder
	b.WriteString(`你是一位有10年以上经验的SRE工程师，擅长结构化故障诊断与可执行排障步骤。
请严格用中文回答（专有名词可保留英文），输出要有工程落地性，避免空泛建议。

`)

	if sp != nil {
		b.WriteString("【技能包】\n")
		b.WriteString(fmt.Sprintf("- 名称: %s\n", sp.DisplayName))
		if len(sp.AnalysisSteps) > 0 {
			b.WriteString("- 分析路径:\n")
			for _, s := range sp.AnalysisSteps {
				b.WriteString(fmt.Sprintf("  • %s\n", s))
			}
		}
		if len(sp.OutputFormat) > 0 {
			b.WriteString("- 输出需覆盖的要点: ")
			b.WriteString(strings.Join(sp.OutputFormat, "、"))
			b.WriteString("\n")
		}
		if sp.ExtraGuidance != "" {
			b.WriteString("- 补充要求:\n")
			b.WriteString(sp.ExtraGuidance)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("【问题域 / Topic】\n")
	b.WriteString(topic)
	b.WriteString("\n\n【观测与输入数据】\n")
	if len(context) == 0 {
		b.WriteString("（用户未提供额外键值，请基于常见生产场景给出假设与验证步骤）\n")
	} else {
		for k, v := range context {
			b.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
		}
	}

	if strings.TrimSpace(ragContext) != "" {
		b.WriteString("\n【知识库摘录（供参考，需自行交叉验证）】\n")
		b.WriteString(ragContext)
		b.WriteString("\n")
	}

	b.WriteString(`
【请输出以下结构化结果】
1. 问题判断（当前最可能处于什么状态）
2. 可能原因（按概率从高到低，说明为何）
3. 验证步骤（命令或控制台操作，逐步可执行）
4. 解决方案与缓解（具体参数/配置/容量/变更建议）
5. 需要进一步采集的信息（若信息不足）

若涉及线上变更，请提示灰度、回滚与变更窗口。`)

	if sp != nil && sp.PromptTemplate != "" {
		b.WriteString("\n\n【技能包附加模板】\n")
		b.WriteString(sp.PromptTemplate)
	}

	return b.String()
}

// BuildAsk is for `ask` — Q&A with optional RAG and optional skill hint.
func BuildAsk(question string, ragContext string, sp *skill.Pack) string {
	var b strings.Builder
	b.WriteString(`你是资深SRE。用中文简洁回答运维问题，先给结论再给可操作步骤。
`)
	if sp != nil {
		b.WriteString("\n【匹配技能包】")
		b.WriteString(sp.DisplayName)
		b.WriteString("\n")
		if len(sp.AnalysisSteps) > 0 {
			b.WriteString("建议排查路径：\n")
			for _, s := range sp.AnalysisSteps {
				b.WriteString("- ")
				b.WriteString(s)
				b.WriteString("\n")
			}
		}
	}
	b.WriteString("\n【用户问题】\n")
	b.WriteString(question)
	if strings.TrimSpace(ragContext) != "" {
		b.WriteString(`

【参考资料】
`)
		b.WriteString(ragContext)
	}
	b.WriteString(`

请给出：要点摘要、排查/处理步骤、注意事项。`)
	return b.String()
}

// BuildRunbook generates a runbook-style document.
func BuildRunbook(topic string, sp *skill.Pack, context map[string]string, ragContext string) string {
	var b strings.Builder
	b.WriteString(`你是SRE文档工程师。请生成一份可交付的 Runbook（中文），读者为值班工程师。

`)
	if sp != nil {
		b.WriteString(fmt.Sprintf("【关联技能】%s（%s）\n\n", sp.DisplayName, sp.Name))
	}
	b.WriteString("【场景概述】\n")
	b.WriteString(topic)
	b.WriteString("\n\n【上下文】\n")
	if len(context) == 0 {
		b.WriteString("（未提供）\n")
	} else {
		for k, v := range context {
			b.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
		}
	}
	if strings.TrimSpace(ragContext) != "" {
		b.WriteString("\n【知识库摘录】\n")
		b.WriteString(ragContext)
	}
	b.WriteString(`

Runbook 结构请包含：
1. 现象与告警特征
2. 影响面评估
3. 快速检查清单（命令级）
4. 根因分类与确认方法
5. 处理步骤（含回滚）
6. 事后复盘与预防`)

	return b.String()
}
