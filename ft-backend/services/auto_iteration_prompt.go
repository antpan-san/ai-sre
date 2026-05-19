package services

import (
	"strings"

	"ft-backend/models"
)

const (
	// AutoIterationDevSkillPath is the Cursor skill agents must follow for token-efficient dev.
	AutoIterationDevSkillPath = ".cursor/skills/auto-iteration-dev/SKILL.md"
	AutoIterationDevSpecVer  = "auto-iteration-dev@v1"
)

// AutoIterationAgentPromptCompact is inlined into every code-agent task (keep short).
const AutoIterationAgentPromptCompact = `【省 Token 开发规范 — 必须遵守】
规范: ` + AutoIterationDevSkillPath + `（本任务首次用 Read 读一次即可；发布阶段再读 release-deploy）
硬规则:
1) 仅改需求相关文件，禁止顺手重构/扩 scope
2) 先 grep 定位再局部 Read，禁止无目标全仓扫、禁止重复读同一大文件
3) 回复简洁：摘要+路径+验收；勿贴大段已有代码
4) 开发期只做最小相关测试；全量 remote-e2e 仅在发布阶段
5) 阻塞时 ≤6 行说明，勿继续盲目尝试
发布: 仅本地验证通过后按 .cursor/skills/release-deploy/SKILL.md 执行（禁止未测通宣称完成）`

// FormatAutoIterationUserRequirement wraps console/CLI user text for storage and agent consumption.
func FormatAutoIterationUserRequirement(title, userBody, topic string) (description, command string) {
	body := strings.TrimSpace(userBody)
	title = strings.TrimSpace(title)
	topic = strings.TrimSpace(topic)
	if body == "" {
		return "", ""
	}
	var b strings.Builder
	b.WriteString("## 需求\n")
	if title != "" {
		b.WriteString("标题: ")
		b.WriteString(title)
		b.WriteString("\n")
	}
	if topic != "" {
		b.WriteString("Topic: ")
		b.WriteString(topic)
		b.WriteString("\n\n")
	}
	b.WriteString(body)
	description = limitAuditText(b.String(), 2000)
	command = limitAuditText(AutoIterationAgentPromptCompact+"\n\n"+description, 2000)
	return description, command
}

// AgentTaskDevSpecMeta returns metadata attached to auto-iteration rows for workers.
func AgentTaskDevSpecMeta() map[string]interface{} {
	return map[string]interface{}{
		"dev_spec":      AutoIterationDevSpecVer,
		"dev_skill":     AutoIterationDevSkillPath,
		"release_skill": ".cursor/skills/release-deploy/SKILL.md",
	}
}

// MergeAgentTaskMetadata ensures dev spec keys exist on task metadata.
func MergeAgentTaskMetadata(existing models.JSONB) models.JSONB {
	out := jsonbToMap(existing)
	for k, v := range AgentTaskDevSpecMeta() {
		out[k] = v
	}
	return models.NewJSONBFromMap(out)
}
