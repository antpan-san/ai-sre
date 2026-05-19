package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// SkillEnhancementReview is persisted after each successful server AI call on supported topics.
type SkillEnhancementReview struct {
	Time               time.Time `json:"time"`
	RequestID          string    `json:"request_id,omitempty"`
	Topic              string    `json:"topic"`
	CommandKind        string    `json:"command_kind,omitempty"` // analyze, ask, runbook
	SkillName          string    `json:"skill_name,omitempty"`
	PackKey            string    `json:"pack_key,omitempty"`
	ProblemKey         string    `json:"problem_key,omitempty"`
	Style              string    `json:"style,omitempty"`
	NeedsEnhancement   bool      `json:"needs_enhancement"`
	Priority           string    `json:"priority"` // low, medium, high
	SavingsScore       int       `json:"savings_score"` // 0-100 降低后续 AI 成本潜力
	Recommendations    []string  `json:"recommendations,omitempty"`
	SuggestedActions   []string  `json:"suggested_actions,omitempty"`
	SimilarRecentCount int       `json:"similar_recent_count,omitempty"`
	EnhancementStatus  string    `json:"enhancement_status,omitempty"` // open, refined, dismissed
}

// PostAICallRecord is input for post-AI skill enhancement review.
type PostAICallRecord struct {
	Topic         string
	CommandKind   string
	SkillName     string
	PackKey       string
	ProblemKey    string
	Style         string
	RequestID     string
	Answer        string
	UserContext   map[string]string
	EvidenceKeys  []string
	MatchedSkill  bool
}

var (
	delegateCollectRe = regexp.MustCompile(`(?i)(请执行|请运行|请使用)\s*(top|free|iostat|vmstat|ps\s|redis-cli|kubectl|ai-sre\s+probe|probe\s+\w+)`)
	delegateCollectRe2 = regexp.MustCompile(`(?i)(redis-cli|kubectl\s|手动采集|补采集)`)
)

// RecordPostAICall appends diagnose sample, enhancement review log, and may trigger auto-refine.
func RecordPostAICall(reg *SkillRegistry, in PostAICallRecord) (SkillEnhancementReview, error) {
	review := EvaluateSkillEnhancement(reg, in)
	return review, RecordPostAICallWithReview(reg, in, review)
}

// RecordPostAICallWithReview persists a precomputed review (used when metadata is returned synchronously).
func RecordPostAICallWithReview(reg *SkillRegistry, in PostAICallRecord, review SkillEnhancementReview) error {
	if reg == nil {
		reg = DefaultSkillRegistry()
	}
	sample := DiagnoseSample{
		Time:              time.Now().UTC(),
		Topic:             strings.ToLower(strings.TrimSpace(in.Topic)),
		SkillName:         strings.TrimSpace(in.SkillName),
		Style:             strings.TrimSpace(in.Style),
		UserContext:       stripBulkContextForSample(in.UserContext),
		EvidenceKey:       in.EvidenceKeys,
		AnswerLen:         len(in.Answer),
		AnswerHead:        headRunes(in.Answer, 600),
		AnswerTail:        tailRunes(in.Answer, 400),
		RequestID:         strings.TrimSpace(in.RequestID),
		CommandKind:       strings.TrimSpace(in.CommandKind),
		PackKey:           strings.TrimSpace(in.PackKey),
		ProblemKey:        strings.TrimSpace(in.ProblemKey),
		EnhancementReview: &review,
	}
	return AppendDiagnoseSample(reg, sample)
}

// EvaluateSkillEnhancement runs deterministic rules (no extra AI call).
func EvaluateSkillEnhancement(reg *SkillRegistry, in PostAICallRecord) SkillEnhancementReview {
	topic := strings.ToLower(strings.TrimSpace(in.Topic))
	review := SkillEnhancementReview{
		Time:              time.Now().UTC(),
		RequestID:         strings.TrimSpace(in.RequestID),
		Topic:             topic,
		CommandKind:       strings.TrimSpace(in.CommandKind),
		SkillName:         strings.TrimSpace(in.SkillName),
		PackKey:           strings.TrimSpace(in.PackKey),
		ProblemKey:        strings.TrimSpace(in.ProblemKey),
		Style:             strings.TrimSpace(in.Style),
		EnhancementStatus: "open",
	}
	if topic == "" {
		topic = "_unknown"
		review.Topic = topic
	}
	var recs, actions []string
	score := 0

	if !in.MatchedSkill || strings.HasPrefix(review.SkillName, topic+"_auto") || review.SkillName == "" {
		recs = append(recs, "未命中稳定 builtin/generated 技能包，同类问题可能重复调用 AI")
		actions = appendUnique(actions, "enhance_skill_yaml")
		score += 25
	}
	if issues := detectEvidenceGaps(topic, in.UserContext, in.EvidenceKeys); len(issues) > 0 {
		recs = append(recs, issues...)
		actions = appendUnique(actions, "add_probe_fields")
		score += 30
	}
	if msg := detectDelegationToUser(in.Answer); msg != "" {
		recs = append(recs, msg)
		actions = appendUnique(actions, "add_probe_fields")
		score += 35
	}
	if msg := detectProbeErrors(in.UserContext); msg != "" {
		recs = append(recs, msg)
		actions = appendUnique(actions, "add_probe_fields")
		score += 20
	}
	if topic == "redis" {
		if extra := evaluateRedisEnhancement(in.UserContext, in.Answer); len(extra) > 0 {
			recs = append(recs, extra...)
			actions = appendUnique(actions, "local_rule")
			actions = appendUnique(actions, "enhance_skill_yaml")
			score += 25
		}
	}
	if reg != nil {
		review.SimilarRecentCount = countSimilarOpenReviews(reg, topic, 7*24*time.Hour)
		if review.SimilarRecentCount >= 5 {
			recs = append(recs, "近 7 日同类 topic 已积累较多待增强样本，建议优先沉淀技能包")
			actions = appendUnique(actions, "auto_refine")
			score += 15
		}
	}
	if strings.TrimSpace(in.CommandKind) == "ask" || strings.TrimSpace(in.CommandKind) == "runbook" {
		recs = append(recs, "问答/Runbook 类调用可评估是否映射到固定 topic 技能与 FAQ 模板")
		actions = appendUnique(actions, "enhance_skill_yaml")
		score += 10
	}

	review.Recommendations = recs
	review.SuggestedActions = actions
	if len(recs) == 0 {
		review.NeedsEnhancement = false
		review.Priority = "low"
		review.SavingsScore = 0
		return review
	}
	review.NeedsEnhancement = true
	if score > 100 {
		score = 100
	}
	review.SavingsScore = score
	switch {
	case score >= 60:
		review.Priority = "high"
	case score >= 30:
		review.Priority = "medium"
	default:
		review.Priority = "low"
	}
	return review
}

func detectEvidenceGaps(topic string, kv map[string]string, keys []string) []string {
	var out []string
	has := func(prefix string) bool {
		for _, k := range keys {
			if strings.HasPrefix(k, prefix) {
				return true
			}
		}
		if kv == nil {
			return false
		}
		for k := range kv {
			if strings.HasPrefix(k, prefix) {
				return true
			}
		}
		return false
	}
	switch topic {
	case "redis":
		if !has("redis_") {
			return []string{"redis 诊断缺少 redis_diagnose_json 证据，probe 采集不完整或未注入"}
		}
	case "linux":
		if !has("linux_") {
			return []string{"linux 诊断缺少 linux_perf_probe_json 证据"}
		}
	case "kafka":
		if !has("kafka_") {
			return []string{"kafka 诊断缺少 kafka_diagnose_json 证据"}
		}
	case "k8s", "kubernetes":
		if !has("kubectl_") && kv != nil && strings.TrimSpace(kv["diagnosis_style"]) == "evidence_root_cause" {
			return []string{"K8s 证据驱动诊断缺少 kubectl 采集输出"}
		}
	case "nginx":
		if !has("nginx_") {
			return []string{"nginx 诊断缺少 nginx 采集证据"}
		}
	case "mysql":
		if !has("mysql_") {
			return []string{"mysql 诊断缺少 mysql_diagnose_json 证据"}
		}
	case "postgresql", "postgres":
		if !has("postgresql_") {
			return []string{"postgresql 诊断缺少 postgresql_diagnose_json 证据"}
		}
	case "elasticsearch", "es":
		if !has("es_") && !has("elasticsearch_") {
			return []string{"elasticsearch 诊断缺少 es 采集证据"}
		}
	}
	return out
}

func detectDelegationToUser(answer string) string {
	a := strings.TrimSpace(answer)
	if a == "" {
		return ""
	}
	if delegateCollectRe.MatchString(a) || delegateCollectRe2.MatchString(a) {
		return "AI 回答仍要求用户手工执行采集命令，应优先扩展 probe/只读采集而非保留该建议"
	}
	return ""
}

func detectProbeErrors(kv map[string]string) string {
	if kv == nil {
		return ""
	}
	for k, v := range kv {
		if strings.Contains(k, "_diagnose_json") || strings.Contains(k, "_probe_json") {
			if strings.Contains(strings.ToLower(v), `"errors"`) && strings.Contains(strings.ToLower(v), `"permission"`) {
				return "采集 JSON 含 permission 类错误，技能包应说明缺失证据对判断的影响"
			}
		}
	}
	return ""
}

func evaluateRedisEnhancement(kv map[string]string, answer string) []string {
	raw := ""
	if kv != nil {
		raw = strings.TrimSpace(kv["redis_diagnose_json"])
	}
	if raw == "" {
		return nil
	}
	var probe struct {
		Clients map[string]any `json:"clients"`
		Memory  map[string]any `json:"memory"`
	}
	if json.Unmarshal([]byte(raw), &probe) != nil {
		return nil
	}
	lower := strings.ToLower(answer)
	var out []string
	if strings.Contains(lower, "碎片") && probe.Memory != nil {
		out = append(out, "Redis 技能需约束：低内存场景不得单独将碎片率定性为 rejected_connections 主因")
	}
	if strings.Contains(lower, "redis-cli") {
		out = append(out, "Redis 输出仍提及 redis-cli，应写入 skill extra_guidance 禁止甩锅采集")
	}
	return out
}

func appendEnhancementReviewLog(reg *SkillRegistry, review SkillEnhancementReview) error {
	dir := reg.DataDir()
	if dir == "" {
		return nil
	}
	full := filepath.Join(dir, "enhancement_reviews.jsonl")
	return appendJSONLine(full, review)
}

// ListEnhancementReviews returns recent reviews, optionally only open/high priority.
func ListEnhancementReviews(reg *SkillRegistry, limit int, openOnly bool) ([]SkillEnhancementReview, error) {
	if reg == nil {
		reg = DefaultSkillRegistry()
	}
	dir := reg.DataDir()
	if dir == "" {
		return nil, nil
	}
	full := filepath.Join(dir, "enhancement_reviews.jsonl")
	lines, err := readRecentJSONLines(full, limit*5)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	latest := map[string]SkillEnhancementReview{}
	for _, ln := range lines {
		var r SkillEnhancementReview
		if json.Unmarshal([]byte(ln), &r) != nil {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(r.Topic)) + "|" + strings.TrimSpace(r.RequestID)
		if key == "|" {
			key = r.Time.Format(time.RFC3339Nano)
		}
		if prev, ok := latest[key]; !ok || r.Time.After(prev.Time) {
			latest[key] = r
		}
	}
	out := make([]SkillEnhancementReview, 0, limit)
	for _, r := range latest {
		if openOnly && (!r.NeedsEnhancement || r.EnhancementStatus == "refined" || r.EnhancementStatus == "dismissed") {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.After(out[j].Time) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// UpdateEnhancementReviewStatus records a refined/dismissed decision for a review.
func UpdateEnhancementReviewStatus(reg *SkillRegistry, requestID, topic, status, note string) error {
	if reg == nil {
		reg = DefaultSkillRegistry()
	}
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "refined" && status != "dismissed" {
		return fmt.Errorf("status must be refined or dismissed")
	}
	review := SkillEnhancementReview{
		Time:              time.Now().UTC(),
		RequestID:         strings.TrimSpace(requestID),
		Topic:             strings.ToLower(strings.TrimSpace(topic)),
		CommandKind:       "admin_review",
		NeedsEnhancement:  false,
		Priority:          "low",
		EnhancementStatus: status,
		Recommendations:   []string{limitAuditText(note, 300)},
	}
	return appendEnhancementReviewLog(reg, review)
}

// EnhancementReviewSummary aggregates pending enhancement work for console.
type EnhancementReviewSummary struct {
	OpenCount        int                        `json:"open_count"`
	HighPriority     int                        `json:"high_priority"`
	MediumPriority   int                        `json:"medium_priority"`
	TotalSavingsScore int                       `json:"total_savings_score"`
	ByTopic          map[string]int             `json:"by_topic"`
	TopTopics        []EnhancementTopicScore    `json:"top_topics"`
	Recent           []SkillEnhancementReview   `json:"recent,omitempty"`
}

type EnhancementTopicScore struct {
	Topic        string `json:"topic"`
	OpenCount    int    `json:"open_count"`
	SavingsScore int    `json:"savings_score"`
}

func SummarizeEnhancementReviews(reg *SkillRegistry, recentLimit int) (EnhancementReviewSummary, error) {
	reviews, err := ListEnhancementReviews(reg, 500, true)
	if err != nil {
		return EnhancementReviewSummary{}, err
	}
	sum := EnhancementReviewSummary{ByTopic: map[string]int{}}
	for _, r := range reviews {
		if r.EnhancementStatus == "refined" || r.EnhancementStatus == "dismissed" {
			continue
		}
		if !r.NeedsEnhancement {
			continue
		}
		sum.OpenCount++
		sum.TotalSavingsScore += r.SavingsScore
		sum.ByTopic[r.Topic]++
		switch r.Priority {
		case "high":
			sum.HighPriority++
		case "medium":
			sum.MediumPriority++
		}
	}
	for topic, n := range sum.ByTopic {
		sum.TopTopics = append(sum.TopTopics, EnhancementTopicScore{Topic: topic, OpenCount: n})
	}
	// sort top topics by count
	for i := 0; i < len(sum.TopTopics); i++ {
		for j := i + 1; j < len(sum.TopTopics); j++ {
			if sum.TopTopics[j].OpenCount > sum.TopTopics[i].OpenCount {
				sum.TopTopics[i], sum.TopTopics[j] = sum.TopTopics[j], sum.TopTopics[i]
			}
		}
	}
	if len(sum.TopTopics) > 10 {
		sum.TopTopics = sum.TopTopics[:10]
	}
	if recentLimit > 0 {
		sum.Recent, _ = ListEnhancementReviews(reg, recentLimit, true)
	}
	return sum, nil
}

func countSimilarOpenReviews(reg *SkillRegistry, topic string, within time.Duration) int {
	reviews, err := ListEnhancementReviews(reg, 200, true)
	if err != nil {
		return 0
	}
	cutoff := time.Now().UTC().Add(-within)
	n := 0
	for _, r := range reviews {
		if r.Topic != topic || r.Time.Before(cutoff) {
			continue
		}
		n++
	}
	return n
}

func stripBulkContextForSample(kv map[string]string) map[string]string {
	if kv == nil {
		return nil
	}
	out := make(map[string]string, len(kv))
	for k, v := range kv {
		if strings.HasPrefix(k, "kubectl_") || strings.HasPrefix(k, "host_") {
			continue
		}
		if strings.HasSuffix(k, "_diagnose_json") || strings.HasSuffix(k, "_probe_json") {
			continue
		}
		if k == "prior_answer_round1" || k == "go_runtime_watch_json" {
			continue
		}
		if len(v) > 256 {
			v = v[:256] + "...(truncated)"
		}
		out[k] = v
	}
	return out
}

func headRunes(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func tailRunes(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

func appendUnique(dst []string, v string) []string {
	for _, x := range dst {
		if x == v {
			return dst
		}
	}
	return append(dst, v)
}

// CollectEvidenceKeysFromContext lists evidence keys for samples.
func CollectEvidenceKeysFromContext(kv map[string]string) []string {
	if kv == nil {
		return nil
	}
	var out []string
	for k := range kv {
		switch {
		case strings.HasPrefix(k, "kubectl_"), strings.HasPrefix(k, "host_"),
			strings.HasPrefix(k, "redis_"), strings.HasPrefix(k, "kafka_"),
			strings.HasPrefix(k, "mysql_"), strings.HasPrefix(k, "postgresql_"),
			strings.HasPrefix(k, "nginx_"), strings.HasPrefix(k, "es_"),
			strings.HasPrefix(k, "elasticsearch_"), strings.HasPrefix(k, "domain_"),
			strings.HasPrefix(k, "linux_"):
			out = append(out, k)
		}
	}
	return out
}

// InferTopicFromText maps free-form text to a diagnostic topic for ask/runbook samples.
func InferTopicFromText(text string) string {
	s := strings.ToLower(text)
	switch {
	case strings.Contains(s, "kafka"):
		return "kafka"
	case strings.Contains(s, "redis"):
		return "redis"
	case strings.Contains(s, "nginx"):
		return "nginx"
	case strings.Contains(s, "mysql"):
		return "mysql"
	case strings.Contains(s, "postgres"):
		return "postgresql"
	case strings.Contains(s, "elastic"):
		return "elasticsearch"
	case strings.Contains(s, "kubernetes") || strings.Contains(s, "k8s") || strings.Contains(s, "pod "):
		return "k8s"
	case strings.Contains(s, "linux") || strings.Contains(s, "iowait") || strings.Contains(s, "oom"):
		return "linux"
	default:
		return "_general"
	}
}
