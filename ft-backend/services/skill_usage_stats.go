package services

import (
	"fmt"
	"strings"
	"time"

	"ft-backend/database"
	"ft-backend/models"
)

// SkillUsageStats is an admin dashboard snapshot.
type SkillUsageStats struct {
	DiagnosticPlans []SkillUsageRow `json:"diagnostic_plans"`
	AIExecutions    []SkillUsageRow `json:"ai_executions"`
	SkillAssets     []SkillUsageRow `json:"skill_assets"`
	Reviews         []SkillUsageRow `json:"reviews"`
}

// SkillUsageRow is one grouped metric line.
type SkillUsageRow struct {
	Label  string `json:"label"`
	Count  int64  `json:"count"`
	Extra  string `json:"extra,omitempty"`
	Status string `json:"status,omitempty"`
}

// GetSkillUsageStats aggregates tables since the given time.
func GetSkillUsageStats(since time.Time) (*SkillUsageStats, error) {
	out := &SkillUsageStats{}
	type planRow struct {
		Topic      string
		SkillKey   string
		ProblemKey string
		Status     string
		Cnt        int64
	}
	var plans []planRow
	if err := database.DB.Model(&models.DiagnosticPlan{}).
		Select("topic, skill_key, problem_key, status, COUNT(*) AS cnt").
		Where("created_at >= ?", since).
		Group("topic, skill_key, problem_key, status").
		Order("cnt DESC").
		Scan(&plans).Error; err != nil {
		return nil, err
	}
	for _, p := range plans {
		out.DiagnosticPlans = append(out.DiagnosticPlans, SkillUsageRow{
			Label:  strings.Join(filterNonEmpty([]string{p.Topic, p.SkillKey, p.ProblemKey}), "/"),
			Status: p.Status,
			Count:  p.Cnt,
		})
	}

	type assetRow struct {
		Topic  string
		Status string
		Cnt    int64
	}
	var assets []assetRow
	if err := database.DB.Model(&models.SkillAsset{}).
		Select("topic, status, COUNT(*) AS cnt").
		Where("created_at >= ?", since).
		Group("topic, status").
		Order("cnt DESC").
		Scan(&assets).Error; err != nil {
		return nil, err
	}
	for _, a := range assets {
		out.SkillAssets = append(out.SkillAssets, SkillUsageRow{
			Label:  a.Topic,
			Status: a.Status,
			Count:  a.Cnt,
		})
	}

	type reviewRow struct {
		Action string
		Cnt    int64
	}
	var reviews []reviewRow
	if err := database.DB.Model(&models.SkillAssetReview{}).
		Select("action, COUNT(*) AS cnt").
		Where("created_at >= ?", since).
		Group("action").
		Order("cnt DESC").
		Scan(&reviews).Error; err != nil {
		return nil, err
	}
	for _, r := range reviews {
		out.Reviews = append(out.Reviews, SkillUsageRow{
			Label: r.Action,
			Count: r.Cnt,
		})
	}

	type execRow struct {
		Category string
		Status   string
		Cnt      int64
	}
	var execs []execRow
	if err := database.DB.Model(&models.ExecutionRecord{}).
		Select("category, status, COUNT(*) AS cnt").
		Where("source = ? AND created_at >= ?", "ai", since).
		Where("category IN ?", []string{"analyze", "diagnostic_plan", "diagnostic_plan_observations", "ask", "runbook"}).
		Group("category, status").
		Order("cnt DESC").
		Scan(&execs).Error; err != nil {
		return nil, err
	}
	for _, e := range execs {
		out.AIExecutions = append(out.AIExecutions, SkillUsageRow{
			Label:  e.Category,
			Status: e.Status,
			Count:  e.Cnt,
		})
	}
	return out, nil
}

// SkillUsageCSV renders diagnostic plan counts as CSV.
func SkillUsageCSV(since time.Time) (string, error) {
	summary, err := GetSkillUsageStats(since)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("section,label,status,count\n")
	for _, row := range summary.DiagnosticPlans {
		b.WriteString(csvLine("diagnostic_plans", row.Label, row.Status, row.Count))
	}
	for _, row := range summary.AIExecutions {
		b.WriteString(csvLine("ai_executions", row.Label, row.Status, row.Count))
	}
	for _, row := range summary.SkillAssets {
		b.WriteString(csvLine("skill_assets", row.Label, row.Status, row.Count))
	}
	for _, row := range summary.Reviews {
		b.WriteString(csvLine("reviews", row.Label, row.Status, row.Count))
	}
	return b.String(), nil
}

func csvLine(section, label, status string, count int64) string {
	return fmt.Sprintf("%s,%s,%s,%d\n", csvEscape(section), csvEscape(label), csvEscape(status), count)
}

func csvEscape(s string) string {
	s = strings.ReplaceAll(s, "\"", "\"\"")
	if strings.ContainsAny(s, ",\"\n") {
		return "\"" + s + "\""
	}
	return s
}

func filterNonEmpty(parts []string) []string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
