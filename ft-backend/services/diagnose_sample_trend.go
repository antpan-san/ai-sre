package services

import (
	"sort"
	"strings"
	"time"

	"ft-backend/database"
	"ft-backend/models"
)

// DiagnoseSampleTrendBucket is one time bucket in the quality trend series.
type DiagnoseSampleTrendBucket struct {
	BucketStart string `json:"bucket_start"`
	Total       int    `json:"total"`
	RuleHit     int    `json:"rule_hit"`
	UsedAI      int    `json:"used_ai"`
	CLICheck    int    `json:"cli_check"`
}

// DiagnoseSampleTrend is a time-series view of diagnose sample quality metrics.
type DiagnoseSampleTrend struct {
	SinceHours  int                         `json:"since_hours"`
	BucketHours int                         `json:"bucket_hours"`
	Buckets     []DiagnoseSampleTrendBucket `json:"buckets"`
}

// TrendDiagnoseSamples aggregates samples into fixed-width UTC buckets.
func TrendDiagnoseSamples(reg *SkillRegistry, since time.Time, sinceHours, bucketHours int) (DiagnoseSampleTrend, error) {
	if bucketHours <= 0 {
		bucketHours = 24
	}
	if bucketHours > 168 {
		bucketHours = 168
	}
	out := DiagnoseSampleTrend{
		SinceHours:  sinceHours,
		BucketHours: bucketHours,
	}
	if diagnoseSamplePGEnabled() {
		if trend, err := trendDiagnoseSamplesPG(since, bucketHours); err == nil {
			out.Buckets = trend
			return out, nil
		}
	}
	if reg == nil {
		reg = DefaultSkillRegistry()
	}
	topics, err := resolveSampleTopics(reg, "")
	if err != nil {
		return out, err
	}
	acc := map[int64]*DiagnoseSampleTrendBucket{}
	for _, tp := range topics {
		rows, err := reg.ReadRecentSamples(tp, 2000)
		if err != nil {
			return out, err
		}
		for _, s := range rows {
			if !since.IsZero() && s.Time.Before(since) {
				continue
			}
			addSampleToTrendBucket(acc, s, bucketHours)
		}
	}
	out.Buckets = sortedTrendBuckets(acc)
	return out, nil
}

func trendDiagnoseSamplesPG(since time.Time, bucketHours int) ([]DiagnoseSampleTrendBucket, error) {
	if !diagnoseSamplePGEnabled() {
		return nil, nil
	}
	q := database.DB.Model(&models.DiagnoseSampleRecord{})
	if !since.IsZero() {
		q = q.Where("sample_time >= ?", since)
	}
	var rows []models.DiagnoseSampleRecord
	if err := q.Order("sample_time ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	acc := map[int64]*DiagnoseSampleTrendBucket{}
	for _, row := range rows {
		s, ok := decodeDiagnoseSamplePayload(row.Payload)
		if !ok {
			s = DiagnoseSample{
				Time:        row.SampleTime,
				SampleSource: row.SampleSource,
				CommandKind: row.CommandKind,
				RuleHit:     row.RuleHit,
				UsedAI:      row.UsedAI,
			}
		}
		if s.Time.IsZero() {
			s.Time = row.SampleTime
		}
		if !s.RuleHit {
			s.RuleHit = row.RuleHit
		}
		if !s.UsedAI {
			s.UsedAI = row.UsedAI
		}
		if strings.TrimSpace(s.SampleSource) == "" {
			s.SampleSource = row.SampleSource
		}
		if strings.TrimSpace(s.CommandKind) == "" {
			s.CommandKind = row.CommandKind
		}
		addSampleToTrendBucket(acc, s, bucketHours)
	}
	return sortedTrendBuckets(acc), nil
}

func addSampleToTrendBucket(acc map[int64]*DiagnoseSampleTrendBucket, s DiagnoseSample, bucketHours int) {
	t := s.Time.UTC()
	if t.IsZero() {
		t = time.Now().UTC()
	}
	start := bucketStartUTC(t, bucketHours)
	key := start.Unix()
	b, ok := acc[key]
	if !ok {
		b = &DiagnoseSampleTrendBucket{BucketStart: start.Format(time.RFC3339)}
		acc[key] = b
	}
	b.Total++
	if s.RuleHit {
		b.RuleHit++
	}
	if s.UsedAI {
		b.UsedAI++
	}
	if s.SampleSource == "cli_check" || strings.EqualFold(s.CommandKind, "check") {
		b.CLICheck++
	}
}

func bucketStartUTC(t time.Time, bucketHours int) time.Time {
	t = t.UTC()
	if bucketHours >= 24 && bucketHours%24 == 0 {
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	}
	epoch := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	hours := int(t.Sub(epoch).Hours())
	bucket := hours / bucketHours * bucketHours
	return epoch.Add(time.Duration(bucket) * time.Hour)
}

func sortedTrendBuckets(acc map[int64]*DiagnoseSampleTrendBucket) []DiagnoseSampleTrendBucket {
	if len(acc) == 0 {
		return nil
	}
	keys := make([]int64, 0, len(acc))
	for k := range acc {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	out := make([]DiagnoseSampleTrendBucket, 0, len(keys))
	for _, k := range keys {
		out = append(out, *acc[k])
	}
	return out
}
