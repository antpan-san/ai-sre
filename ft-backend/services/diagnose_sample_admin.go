package services

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// DiagnoseSampleSummary aggregates skill sample pool metrics for the console.
type DiagnoseSampleSummary struct {
	TotalSamples   int                   `json:"total_samples"`
	CLICheckCount  int                   `json:"cli_check_count"`
	RuleHitCount   int                   `json:"rule_hit_count"`
	UsedAICount    int                   `json:"used_ai_count"`
	SinceHours     int                   `json:"since_hours"`
	ByTopic        map[string]int        `json:"by_topic"`
	TopTopics      []DiagnoseTopicCount  `json:"top_topics"`
	RuleHitRatePct int                   `json:"rule_hit_rate_pct"`
	AICallRatePct  int                   `json:"ai_call_rate_pct"`
	AIAvoidancePct int                   `json:"ai_avoidance_pct"`
}

type DiagnoseTopicCount struct {
	Topic string `json:"topic"`
	Count int    `json:"count"`
}

// ListDiagnoseSamples returns recent samples for one topic or all topics merged by time.
func ListDiagnoseSamples(reg *SkillRegistry, topic string, limit int, since time.Time) ([]DiagnoseSample, error) {
	if reg == nil {
		reg = DefaultSkillRegistry()
	}
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	if diagnoseSamplePGEnabled() {
		perTopic := limit
		if topic == "" {
			perTopic = limit * 2
			if perTopic > 200 {
				perTopic = 200
			}
		}
		rows, err := readDiagnoseSamplesPG(topic, perTopic, since)
		if err == nil && len(rows) > 0 {
			sort.Slice(rows, func(i, j int) bool { return rows[i].Time.After(rows[j].Time) })
			if len(rows) > limit {
				rows = rows[:limit]
			}
			return rows, nil
		}
	}
	topics, err := resolveSampleTopics(reg, topic)
	if err != nil {
		return nil, err
	}
	perTopic := limit
	if topic == "" && len(topics) > 1 {
		perTopic = limit * 2
		if perTopic > 200 {
			perTopic = 200
		}
	}
	var merged []DiagnoseSample
	for _, tp := range topics {
		rows, err := reg.ReadRecentSamples(tp, perTopic)
		if err != nil {
			return nil, err
		}
		for _, s := range rows {
			if !since.IsZero() && s.Time.Before(since) {
				continue
			}
			merged = append(merged, s)
		}
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Time.After(merged[j].Time)
	})
	if len(merged) > limit {
		merged = merged[:limit]
	}
	return merged, nil
}

// SummarizeDiagnoseSamples counts samples since the given time (PostgreSQL preferred).
func SummarizeDiagnoseSamples(reg *SkillRegistry, since time.Time, sinceHours int) (DiagnoseSampleSummary, error) {
	if reg == nil {
		reg = DefaultSkillRegistry()
	}
	if diagnoseSamplePGEnabled() {
		out, err := summarizeDiagnoseSamplesPG(since, sinceHours)
		if err == nil {
			return out, nil
		}
	}
	out := DiagnoseSampleSummary{
		SinceHours: sinceHours,
		ByTopic:    map[string]int{},
	}
	topics, err := resolveSampleTopics(reg, "")
	if err != nil {
		return out, err
	}
	for _, tp := range topics {
		rows, err := reg.ReadRecentSamples(tp, 500)
		if err != nil {
			return out, err
		}
		for _, s := range rows {
			if !since.IsZero() && s.Time.Before(since) {
				continue
			}
			out.TotalSamples++
			out.ByTopic[tp]++
			if s.SampleSource == "cli_check" || strings.EqualFold(s.CommandKind, "check") {
				out.CLICheckCount++
			}
			if s.RuleHit {
				out.RuleHitCount++
			}
			if s.UsedAI {
				out.UsedAICount++
			}
		}
	}
	type pair struct {
		topic string
		count int
	}
	var ranked []pair
	for t, n := range out.ByTopic {
		ranked = append(ranked, pair{t, n})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].count == ranked[j].count {
			return ranked[i].topic < ranked[j].topic
		}
		return ranked[i].count > ranked[j].count
	})
	for i := 0; i < len(ranked) && i < 12; i++ {
		out.TopTopics = append(out.TopTopics, DiagnoseTopicCount{Topic: ranked[i].topic, Count: ranked[i].count})
	}
	applyDiagnoseSampleRates(&out)
	return out, nil
}

func applyDiagnoseSampleRates(out *DiagnoseSampleSummary) {
	if out == nil || out.TotalSamples <= 0 {
		return
	}
	out.RuleHitRatePct = out.RuleHitCount * 100 / out.TotalSamples
	out.AICallRatePct = out.UsedAICount * 100 / out.TotalSamples
	if out.UsedAICount+out.RuleHitCount > 0 {
		out.AIAvoidancePct = out.RuleHitCount * 100 / (out.UsedAICount + out.RuleHitCount)
	}
}

func resolveSampleTopics(reg *SkillRegistry, topic string) ([]string, error) {
	topic = strings.ToLower(strings.TrimSpace(topic))
	if topic != "" {
		return []string{topic}, nil
	}
	seen := map[string]struct{}{}
	var out []string
	if diagnoseSamplePGEnabled() {
		if pgTopics, err := listDiagnoseSampleTopicsPG(); err == nil {
			for _, t := range pgTopics {
				if t == "" {
					continue
				}
				if _, ok := seen[t]; ok {
					continue
				}
				seen[t] = struct{}{}
				out = append(out, t)
			}
		}
	}
	dir := reg.DataDir()
	if dir == "" {
		sort.Strings(out)
		return out, nil
	}
	glob := filepath.Join(dir, "samples", "*.jsonl")
	matches, err := filepath.Glob(glob)
	if err != nil {
		return nil, err
	}
	for _, p := range matches {
		base := strings.TrimSuffix(filepath.Base(p), ".jsonl")
		if base == "" {
			continue
		}
		if _, ok := seen[base]; ok {
			continue
		}
		seen[base] = struct{}{}
		out = append(out, base)
	}
	sort.Strings(out)
	if len(out) == 0 {
		_ = os.MkdirAll(filepath.Join(dir, "samples"), 0o755)
	}
	return out, nil
}
