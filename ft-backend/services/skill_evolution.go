package services

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type SkillPack struct {
	Name           string   `json:"name" yaml:"name"`
	DisplayName    string   `json:"display_name" yaml:"display_name"`
	Topics         []string `json:"topics" yaml:"topics"`
	MatchKeywords  []string `json:"match_keywords" yaml:"match_keywords"`
	Input          []string `json:"input" yaml:"input"`
	AnalysisSteps  []string `json:"analysis_steps" yaml:"analysis_steps"`
	OutputFormat   []string `json:"output_format" yaml:"output_format"`
	ExtraGuidance  string   `json:"extra_guidance,omitempty" yaml:"extra_guidance,omitempty"`
	PromptTemplate string   `json:"prompt_template,omitempty" yaml:"prompt_template,omitempty"`
}

// BuildSkillDraft builds a minimal valid draft for automatic skill iteration.
func BuildSkillDraft(topic string, kv map[string]string) *SkillPack {
	name := SkillNameForTopic(topic)
	if name == "" {
		return nil
	}
	kws := make([]string, 0, len(kv)+2)
	for k := range kv {
		kws = append(kws, sanitizeSkillToken(k))
	}
	kws = append(kws, topic, "diagnose")
	kws = dedupeNonEmpty(kws)
	if len(kws) > 12 {
		kws = kws[:12]
	}
	return &SkillPack{
		Name:          name,
		DisplayName:   fmt.Sprintf("Auto %s diagnose", strings.ToUpper(topic)),
		Topics:        []string{topic},
		MatchKeywords: kws,
		Input:         dedupeNonEmpty(mapKeys(kv)),
		AnalysisSteps: []string{"确认现象与影响范围", "优先验证最高概率根因", "执行最小风险缓解动作", "补充观测并复盘沉淀"},
		OutputFormat:  []string{"root_cause", "solution", "verification_commands"},
		ExtraGuidance: "输出必须包含可执行命令与回滚建议。",
	}
}

func SkillNameForTopic(topic string) string {
	topic = sanitizeSkillToken(topic)
	if topic == "" {
		return ""
	}
	return topic + "_auto_evolved"
}

func ValidateSkillDraft(p *SkillPack) bool {
	if p == nil {
		return false
	}
	if strings.TrimSpace(p.Name) == "" || len(p.Topics) == 0 || len(p.AnalysisSteps) < 2 || len(p.OutputFormat) == 0 {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-z0-9_\-]+$`, p.Name)
	return matched
}

func sanitizeSkillToken(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "_")
	re := regexp.MustCompile(`[^a-z0-9_\-]+`)
	return strings.Trim(re.ReplaceAllString(s, ""), "_-")
}

func dedupeNonEmpty(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func mapKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
