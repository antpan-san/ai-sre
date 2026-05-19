package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// RefineSkillInput controls a single refine pass.
type RefineSkillInput struct {
	Topic           string
	MaxSamples      int
	MaxFeedback     int
	UserHint        string
	ForceLLMTimeout time.Duration
	DryRun          bool
}

// RefineSkillResult bundles the produced + persisted pack.
type RefineSkillResult struct {
	NewPack       *SkillPack `json:"new_pack"`
	OldPackName   string     `json:"old_pack_name,omitempty"`
	SamplesUsed   int        `json:"samples_used"`
	FeedbackUsed  int        `json:"feedback_used"`
	PersistedPath string     `json:"persisted_path,omitempty"`
	DraftYAML     string     `json:"draft_yaml,omitempty"`
	DryRun        bool       `json:"dry_run,omitempty"`
	Notes         string     `json:"notes,omitempty"`
}

// RefineSkill loads recent samples and feedback for a topic, asks the configured
// LLM to produce a refined pack (YAML in answer body), validates and persists it.
// Returns an error when the LLM is not configured, the call fails, or the
// produced YAML cannot be parsed/validated.
func RefineSkill(ctx context.Context, reg *SkillRegistry, in RefineSkillInput) (*RefineSkillResult, error) {
	if reg == nil {
		return nil, errors.New("skill registry is nil")
	}
	topic := strings.ToLower(strings.TrimSpace(in.Topic))
	if topic == "" {
		return nil, errors.New("topic required")
	}
	maxSamples := in.MaxSamples
	if maxSamples <= 0 {
		maxSamples = 12
	}
	maxFeedback := in.MaxFeedback
	if maxFeedback <= 0 {
		maxFeedback = 8
	}

	cur := reg.Match(topic, nil)
	if cur == nil {
		return nil, fmt.Errorf("no existing skill matches topic %q; add a builtin first", topic)
	}

	samples, _ := reg.ReadRecentSamples(topic, maxSamples)
	feedback, _ := reg.ReadRecentFeedback(topic, maxFeedback)

	prompt := buildRefinePrompt(cur.Pack, samples, feedback, in.UserHint)

	timeout := in.ForceLLMTimeout
	if timeout <= 0 {
		timeout = 90 * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cfg := LoadServerAIConfig()
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("OPSFLEET_AI_API_KEY 未配置：无法调用模型精炼技能包")
	}
	answer, err := DiagnoseWithDeepSeek(callCtx, cfg, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM 精炼失败: %w", err)
	}

	yamlText := extractYAMLBlock(answer)
	if strings.TrimSpace(yamlText) == "" {
		return nil, errors.New("LLM 返回中未发现 YAML 技能包")
	}
	var newPack SkillPack
	if err := yaml.Unmarshal([]byte(yamlText), &newPack); err != nil {
		return nil, fmt.Errorf("解析模型 YAML 失败: %w", err)
	}
	// inherit + normalize
	newPack.Topics = normalizeTopics(newPack.Topics, cur.Pack.Topics)
	if strings.TrimSpace(newPack.Name) == "" {
		newPack.Name = nextRefinedName(cur.Pack.Name)
	}
	if strings.TrimSpace(newPack.DisplayName) == "" {
		newPack.DisplayName = cur.Pack.DisplayName + " (refined)"
	}
	if !ValidateSkillDraft(&newPack) {
		return nil, errors.New("模型产出的技能包未通过最小 schema 校验")
	}

	if in.DryRun {
		return &RefineSkillResult{
			NewPack:      &newPack,
			OldPackName:  cur.Pack.Name,
			SamplesUsed:  len(samples),
			FeedbackUsed: len(feedback),
			DraftYAML:    yamlText,
			DryRun:       true,
			Notes:        "dry run: YAML validated, not written to generated/",
		}, nil
	}

	persisted, err := reg.SaveGenerated(&newPack)
	if err != nil {
		return nil, fmt.Errorf("写入 generated 失败: %w", err)
	}
	return &RefineSkillResult{
		NewPack:       &newPack,
		OldPackName:   cur.Pack.Name,
		SamplesUsed:   len(samples),
		FeedbackUsed:  len(feedback),
		PersistedPath: persisted,
	}, nil
}

func buildRefinePrompt(cur SkillPack, samples []DiagnoseSample, feedback []SkillFeedback, userHint string) string {
	var b strings.Builder
	b.WriteString("你是一名 SRE 知识工程师。我会给你当前一个技能包（YAML）+ 最近的诊断样本与反馈，请输出一个【更好】的技能包，使下一次同主题的根因定位更快、更准。\n\n")
	b.WriteString("严格要求：\n")
	b.WriteString("1) 只回复一个 ```yaml ... ``` 代码块，内含完整的技能包，**不要**任何额外文字、解释或前言。\n")
	b.WriteString("2) 必须保留 topics 字段并包含原 topics；name 可以保持或升级版本号（如 v1 → v2）。\n")
	b.WriteString("3) analysis_steps 必须 ≥ 4 条，覆盖反馈中常被忽略的检查；output_format 至少包含「根因」「关键证据」「修复要点」三个小节标题。\n")
	b.WriteString("4) extra_guidance 中可加入新的反例（如：避免把历史 previous 日志误判为当前持续故障）。\n")
	b.WriteString("5) 不要写入任何具体客户的主机名 / IP / namespace；只保留通用排查模式。\n\n")
	b.WriteString("=== 当前技能包 YAML ===\n```yaml\n")
	curYaml, _ := yaml.Marshal(cur)
	b.Write(curYaml)
	b.WriteString("```\n\n")
	if h := strings.TrimSpace(userHint); h != "" {
		b.WriteString("=== 用户指令（应在新技能包中体现）===\n")
		b.WriteString(h)
		b.WriteString("\n\n")
	}
	if len(samples) > 0 {
		b.WriteString(fmt.Sprintf("=== 最近 %d 次诊断样本（已脱敏摘要）===\n", len(samples)))
		for i, s := range samples {
			short, _ := json.Marshal(s)
			b.WriteString(fmt.Sprintf("- [%d] %s\n", i+1, string(short)))
		}
		b.WriteString("\n")
	}
	if len(feedback) > 0 {
		b.WriteString(fmt.Sprintf("=== 最近 %d 条客户端反馈 ===\n", len(feedback)))
		for i, f := range feedback {
			short, _ := json.Marshal(f)
			b.WriteString(fmt.Sprintf("- [%d] %s\n", i+1, string(short)))
		}
		b.WriteString("\n")
	}
	b.WriteString("现在请输出新的技能包 YAML。\n")
	return b.String()
}

var yamlFenceRe = regexp.MustCompile("(?s)```\\s*ya?ml\\s*\\n(.*?)```")

func extractYAMLBlock(answer string) string {
	if m := yamlFenceRe.FindStringSubmatch(answer); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	// fallback: if the answer starts with a YAML document marker
	trimmed := strings.TrimSpace(answer)
	if strings.HasPrefix(trimmed, "name:") || strings.HasPrefix(trimmed, "---") {
		return trimmed
	}
	return ""
}

func normalizeTopics(produced []string, fallback []string) []string {
	merged := append([]string{}, fallback...)
	merged = append(merged, produced...)
	seen := map[string]struct{}{}
	out := make([]string, 0, len(merged))
	for _, t := range merged {
		k := strings.ToLower(strings.TrimSpace(t))
		if k == "" {
			continue
		}
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	return out
}

var versionTailRe = regexp.MustCompile(`_v(\d+)$`)

func nextRefinedName(cur string) string {
	cur = strings.TrimSpace(cur)
	if cur == "" {
		return "auto_refined_v1"
	}
	if m := versionTailRe.FindStringSubmatch(cur); len(m) == 2 {
		var n int
		fmt.Sscanf(m[1], "%d", &n)
		return versionTailRe.ReplaceAllString(cur, fmt.Sprintf("_v%d", n+1))
	}
	return cur + "_v2"
}
