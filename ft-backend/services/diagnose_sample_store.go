package services

import (
	"encoding/json"
	"strings"
	"time"

	"ft-backend/database"
	"ft-backend/models"
	"sort"
)

func diagnoseSamplePGEnabled() bool {
	return database.DB != nil
}

func persistDiagnoseSamplePG(s DiagnoseSample) error {
	if !diagnoseSamplePGEnabled() {
		return nil
	}
	topic := strings.ToLower(strings.TrimSpace(s.Topic))
	if topic == "" {
		topic = "_unknown"
	}
	sampleTime := s.Time
	if sampleTime.IsZero() {
		sampleTime = time.Now().UTC()
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return err
	}
	row := models.DiagnoseSampleRecord{
		SampleTime:           sampleTime,
		Topic:                topic,
		SampleSource:         strings.TrimSpace(s.SampleSource),
		CommandKind:          strings.TrimSpace(s.CommandKind),
		SkillName:            strings.TrimSpace(s.SkillName),
		RequestID:            strings.TrimSpace(s.RequestID),
		ExecutionID:          strings.TrimSpace(s.ExecutionID),
		UsedAI:               s.UsedAI,
		RuleHit:              s.RuleHit,
		EvidenceCompleteness: strings.TrimSpace(s.EvidenceCompleteness),
		RootCauseDigest:      strings.TrimSpace(s.RootCauseDigest),
		RecommendationDigest: strings.TrimSpace(s.RecommendationDigest),
		Payload:              models.JSONB(raw),
	}
	return database.DB.Create(&row).Error
}

func readDiagnoseSamplesPG(topic string, limit int, since time.Time) ([]DiagnoseSample, error) {
	if !diagnoseSamplePGEnabled() {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	q := database.DB.Model(&models.DiagnoseSampleRecord{}).Order("sample_time DESC").Limit(limit)
	topic = strings.ToLower(strings.TrimSpace(topic))
	if topic != "" {
		q = q.Where("topic = ?", topic)
	}
	if !since.IsZero() {
		q = q.Where("sample_time >= ?", since)
	}
	var rows []models.DiagnoseSampleRecord
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]DiagnoseSample, 0, len(rows))
	for _, row := range rows {
		s, ok := decodeDiagnoseSamplePayload(row.Payload)
		if !ok {
			s = DiagnoseSample{
				Time:                 row.SampleTime,
				Topic:                row.Topic,
				SampleSource:         row.SampleSource,
				CommandKind:          row.CommandKind,
				SkillName:            row.SkillName,
				RequestID:            row.RequestID,
				ExecutionID:          row.ExecutionID,
				UsedAI:               row.UsedAI,
				RuleHit:              row.RuleHit,
				EvidenceCompleteness: row.EvidenceCompleteness,
				RootCauseDigest:      row.RootCauseDigest,
				RecommendationDigest: row.RecommendationDigest,
			}
		}
		if s.Time.IsZero() {
			s.Time = row.SampleTime
		}
		if strings.TrimSpace(s.Topic) == "" {
			s.Topic = row.Topic
		}
		out = append(out, s)
	}
	return out, nil
}

func decodeDiagnoseSamplePayload(raw models.JSONB) (DiagnoseSample, bool) {
	if len(raw) == 0 {
		return DiagnoseSample{}, false
	}
	var s DiagnoseSample
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return DiagnoseSample{}, false
	}
	return s, true
}

func countSimilarDiagnoseSamplesPG(topic, digest string, since time.Time) int {
	if !diagnoseSamplePGEnabled() || digest == "" {
		return 0
	}
	topic = strings.ToLower(strings.TrimSpace(topic))
	if topic == "" {
		return 0
	}
	var n int64
	q := database.DB.Model(&models.DiagnoseSampleRecord{}).
		Where("topic = ? AND root_cause_digest = ?", topic, digest)
	if !since.IsZero() {
		q = q.Where("sample_time >= ?", since)
	}
	_ = q.Count(&n).Error
	return int(n)
}

func listDiagnoseSampleTopicsPG() ([]string, error) {
	if !diagnoseSamplePGEnabled() {
		return nil, nil
	}
	var topics []string
	err := database.DB.Model(&models.DiagnoseSampleRecord{}).
		Distinct("topic").
		Order("topic ASC").
		Pluck("topic", &topics).Error
	return topics, err
}

func summarizeDiagnoseSamplesPG(since time.Time, sinceHours int) (DiagnoseSampleSummary, error) {
	out := DiagnoseSampleSummary{
		SinceHours: sinceHours,
		ByTopic:    map[string]int{},
	}
	if !diagnoseSamplePGEnabled() {
		return out, nil
	}
	q := database.DB.Model(&models.DiagnoseSampleRecord{})
	if !since.IsZero() {
		q = q.Where("sample_time >= ?", since)
	}
	var rows []models.DiagnoseSampleRecord
	if err := q.Find(&rows).Error; err != nil {
		return out, err
	}
	type pair struct {
		topic string
		count int
	}
	ranked := make([]pair, 0)
	for _, row := range rows {
		out.TotalSamples++
		out.ByTopic[row.Topic]++
		if row.SampleSource == "cli_check" || strings.EqualFold(row.CommandKind, "check") {
			out.CLICheckCount++
		}
		if row.RuleHit {
			out.RuleHitCount++
		}
		if row.UsedAI {
			out.UsedAICount++
		}
	}
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
