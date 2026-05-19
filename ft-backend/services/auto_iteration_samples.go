package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ft-backend/database"
	"ft-backend/models"

	"github.com/google/uuid"
)

// AutoIterationSampleContext links an auto-iteration row to trigger/similar samples.
type AutoIterationSampleContext struct {
	Topic              string           `json:"topic,omitempty"`
	ExecutionID        string           `json:"execution_id,omitempty"`
	RootCauseDigest    string           `json:"root_cause_digest,omitempty"`
	SimilarRecentCount int              `json:"similar_recent_count,omitempty"`
	Classification     string           `json:"sample_classification,omitempty"`
	TriggerSample      *DiagnoseSample  `json:"trigger_sample,omitempty"`
	SimilarSamples     []DiagnoseSample `json:"similar_samples,omitempty"`
}

// GetAutoIterationSampleContext returns trigger and similar diagnose samples for an iteration.
func GetAutoIterationSampleContext(iterationID uuid.UUID) (*AutoIterationSampleContext, error) {
	if database.DB == nil {
		return &AutoIterationSampleContext{}, nil
	}
	var row models.AutoIteration
	if err := database.DB.Where("id = ?", iterationID).First(&row).Error; err != nil {
		return nil, err
	}
	meta := decodeRecordMetadata(row.Metadata)
	out := &AutoIterationSampleContext{
		Topic:              firstNonEmpty(row.Topic, strMeta(meta, "topic")),
		ExecutionID:        strMeta(meta, "execution_id"),
		RootCauseDigest:    strMeta(meta, "root_cause_digest"),
		SimilarRecentCount: intFromMeta(meta, "similar_recent_count"),
		Classification:     strMeta(meta, "sample_classification"),
	}
	topic := strings.ToLower(strings.TrimSpace(out.Topic))
	reg := DefaultSkillRegistry()
	if execID := strings.TrimSpace(out.ExecutionID); execID != "" {
		if s, ok := findDiagnoseSampleByExecutionID(reg, topic, execID); ok {
			out.TriggerSample = &s
		}
	}
	if digest := strings.TrimSpace(out.RootCauseDigest); digest != "" && topic != "" {
		out.SimilarSamples = listSimilarDiagnoseSamples(reg, topic, digest, 12)
	} else if topic != "" {
		rows, _ := reg.ReadRecentSamples(topic, 8)
		out.SimilarSamples = rows
	}
	return out, nil
}

func findDiagnoseSampleByExecutionID(reg *SkillRegistry, topic, execID string) (DiagnoseSample, bool) {
	topic = strings.ToLower(strings.TrimSpace(topic))
	rows, err := readDiagnoseSamplesPG(topic, 80, time.Time{})
	if err == nil && len(rows) > 0 {
		for _, s := range rows {
			if strings.TrimSpace(s.ExecutionID) == execID {
				return s, true
			}
		}
	}
	if reg == nil {
		reg = DefaultSkillRegistry()
	}
	topics := []string{topic}
	if topic == "" {
		topics, _ = resolveSampleTopics(reg, "")
	}
	for _, tp := range topics {
		rows, err := reg.ReadRecentSamples(tp, 80)
		if err != nil {
			continue
		}
		for _, s := range rows {
			if strings.TrimSpace(s.ExecutionID) == execID {
				return s, true
			}
		}
	}
	return DiagnoseSample{}, false
}

func listSimilarDiagnoseSamples(reg *SkillRegistry, topic, digest string, limit int) []DiagnoseSample {
	if reg == nil {
		reg = DefaultSkillRegistry()
	}
	rows, err := reg.ReadRecentSamples(topic, 200)
	if err != nil {
		return nil
	}
	cutoff := time.Now().UTC().Add(-7 * 24 * time.Hour)
	var out []DiagnoseSample
	for _, s := range rows {
		if s.Time.Before(cutoff) {
			continue
		}
		if strings.TrimSpace(s.RootCauseDigest) != digest {
			continue
		}
		out = append(out, s)
		if len(out) >= limit {
			break
		}
	}
	return out
}

// BackfillDiagnoseSamplesResult summarizes JSONL → PostgreSQL backfill.
type BackfillDiagnoseSamplesResult struct {
	TopicsScanned int `json:"topics_scanned"`
	LinesRead     int `json:"lines_read"`
	Inserted      int `json:"inserted"`
	Skipped       int `json:"skipped"`
	Errors        int `json:"errors"`
}

// BackfillDiagnoseSamplesFromJSONL imports historical JSONL samples into PostgreSQL.
func BackfillDiagnoseSamplesFromJSONL(reg *SkillRegistry) (BackfillDiagnoseSamplesResult, error) {
	out := BackfillDiagnoseSamplesResult{}
	if !diagnoseSamplePGEnabled() {
		return out, nil
	}
	if reg == nil {
		reg = DefaultSkillRegistry()
	}
	dir := reg.DataDir()
	if dir == "" {
		return out, nil
	}
	matches, err := filepath.Glob(filepath.Join(dir, "samples", "*.jsonl"))
	if err != nil {
		return out, err
	}
	sort.Strings(matches)
	for _, path := range matches {
		out.TopicsScanned++
		lines, err := readAllJSONLines(path)
		if err != nil {
			out.Errors++
			continue
		}
		for _, ln := range lines {
			out.LinesRead++
			var s DiagnoseSample
			if json.Unmarshal([]byte(ln), &s) != nil {
				out.Errors++
				continue
			}
			if s.Time.IsZero() {
				s.Time = time.Now().UTC()
			}
			if existsDiagnoseSamplePG(s) {
				out.Skipped++
				continue
			}
			if err := persistDiagnoseSamplePG(s); err != nil {
				out.Errors++
				continue
			}
			out.Inserted++
		}
	}
	return out, nil
}

func existsDiagnoseSamplePG(s DiagnoseSample) bool {
	topic := strings.ToLower(strings.TrimSpace(s.Topic))
	if topic == "" {
		return false
	}
	q := database.DB.Model(&models.DiagnoseSampleRecord{}).Where("topic = ?", topic)
	if execID := strings.TrimSpace(s.ExecutionID); execID != "" {
		q = q.Where("execution_id = ?", execID)
	} else if digest := strings.TrimSpace(s.RootCauseDigest); digest != "" {
		q = q.Where("root_cause_digest = ? AND sample_time = ?", digest, s.Time.UTC())
	} else {
		q = q.Where("sample_time = ?", s.Time.UTC())
	}
	var n int64
	_ = q.Count(&n).Error
	return n > 0
}

func readAllJSONLines(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, ln := range strings.Split(string(raw), "\n") {
		ln = strings.TrimSpace(ln)
		if ln != "" {
			out = append(out, ln)
		}
	}
	return out, nil
}

func intFromMeta(meta map[string]interface{}, key string) int {
	if meta == nil {
		return 0
	}
	switch v := meta[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	default:
		return 0
	}
}

func classifyTargetKind(topic, target string) string {
	topic = strings.ToLower(strings.TrimSpace(topic))
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	switch topic {
	case "redis":
		if strings.Contains(target, ",") {
			return "redis_cluster"
		}
		return "redis_single"
	case "k8s", "kubernetes":
		lower := strings.ToLower(target)
		if strings.HasPrefix(lower, "pod/") || strings.Contains(lower, "/") {
			return "k8s_pod"
		}
		if strings.HasPrefix(lower, "deploy/") || strings.Contains(lower, "deployment") {
			return "k8s_deployment"
		}
		return "k8s_other"
	case "kafka":
		if strings.Contains(target, ",") {
			return "kafka_multi_broker"
		}
		return "kafka_single_broker"
	default:
		return topic + "_target"
	}
}

func noteNewTargetKindIfNeeded(reg *SkillRegistry, topic, target string, review *SkillEnhancementReview) {
	kind := classifyTargetKind(topic, target)
	if kind == "" || review == nil {
		return
	}
	if seenTargetKindBefore(reg, topic, kind) {
		return
	}
	review.NeedsEnhancement = true
	review.Recommendations = appendUnique(review.Recommendations, "出现新的 target 类型 "+kind+"，建议补充采集与规则")
	review.SuggestedActions = appendUnique(review.SuggestedActions, "add_target_profile")
	if review.Priority == "low" {
		review.Priority = "medium"
	}
	review.SavingsScore += 10
}

func seenTargetKindBefore(reg *SkillRegistry, topic, kind string) bool {
	if reg == nil {
		reg = DefaultSkillRegistry()
	}
	rows, err := reg.ReadRecentSamples(topic, 300)
	if err != nil {
		return true
	}
	for _, s := range rows {
		if classifyTargetKind(topic, s.Target) == kind {
			return true
		}
	}
	return false
}
