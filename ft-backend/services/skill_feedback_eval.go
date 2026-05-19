package services

import (
	"fmt"
	"strings"
	"time"

	"ft-backend/common/config"
)

type SkillFeedbackEvalResult struct {
	HelpfulCount     int    `json:"helpful_count"`
	UnhelpfulCount   int    `json:"unhelpful_count"`
	ReviewTriggered  bool   `json:"review_triggered"`
	ReviewPriority   string `json:"review_priority,omitempty"`
	Classification   string `json:"classification,omitempty"`
	Recommendations  []string `json:"recommendations,omitempty"`
}

// ProcessSkillFeedback evaluates cumulative feedback and may append an enhancement review.
func ProcessSkillFeedback(reg *SkillRegistry, fb SkillFeedback) (*SkillFeedbackEvalResult, error) {
	if reg == nil {
		reg = DefaultSkillRegistry()
	}
	cfg := config.ResolvedSkillFeedbackConfig()
	window := time.Duration(cfg.WindowDays) * 24 * time.Hour
	topic := strings.ToLower(strings.TrimSpace(fb.Topic))
	helpful, unhelpful := countTopicFeedback(reg, topic, window)
	out := &SkillFeedbackEvalResult{
		HelpfulCount:   helpful,
		UnhelpfulCount: unhelpful,
	}
	if fb.Helpful == nil {
		return out, nil
	}
	var review SkillEnhancementReview
	trigger := false
	if !*fb.Helpful && unhelpful >= cfg.UnhelpfulThreshold {
		trigger = true
		out.Classification = "product_gap"
		review = SkillEnhancementReview{
			Time:             time.Now().UTC(),
			RequestID:        strings.TrimSpace(fb.RequestID),
			Topic:            topic,
			CommandKind:      "feedback",
			SkillName:        strings.TrimSpace(fb.SkillName),
			NeedsEnhancement: true,
			Priority:         "high",
			SavingsScore:     40,
			EnhancementStatus: "open",
			Recommendations: []string{
				fmt.Sprintf("近 %d 天「无用」反馈达 %d 次，需审查技能包与采集字段", cfg.WindowDays, unhelpful),
			},
			SuggestedActions: []string{"enhance_skill_yaml", "add_probe_fields"},
		}
		if note := strings.TrimSpace(fb.Note); note != "" {
			review.Recommendations = append(review.Recommendations, limitAuditText(note, 200))
		}
	} else if *fb.Helpful && helpful >= cfg.HelpfulThreshold {
		trigger = true
		out.Classification = "rule_candidate"
		review = SkillEnhancementReview{
			Time:             time.Now().UTC(),
			RequestID:        strings.TrimSpace(fb.RequestID),
			Topic:            topic,
			CommandKind:      "feedback",
			SkillName:        strings.TrimSpace(fb.SkillName),
			NeedsEnhancement: true,
			Priority:         "medium",
			SavingsScore:     35,
			EnhancementStatus: "open",
			Recommendations: []string{
				fmt.Sprintf("近 %d 天「有用」反馈达 %d 次，可评估沉淀为本地规则", cfg.WindowDays, helpful),
			},
			SuggestedActions: []string{"local_rule", "enhance_skill_yaml"},
		}
	}
	if trigger {
		out.ReviewTriggered = true
		out.ReviewPriority = review.Priority
		out.Recommendations = review.Recommendations
		_ = appendEnhancementReviewLog(reg, review)
	}
	return out, nil
}

func countTopicFeedback(reg *SkillRegistry, topic string, window time.Duration) (helpful, unhelpful int) {
	rows, err := reg.ReadRecentFeedback(topic, 200)
	if err != nil || len(rows) == 0 {
		return 0, 0
	}
	cutoff := time.Now().UTC().Add(-window)
	for _, f := range rows {
		if !f.Time.IsZero() && f.Time.Before(cutoff) {
			continue
		}
		if f.Helpful == nil {
			continue
		}
		if *f.Helpful {
			helpful++
		} else {
			unhelpful++
		}
	}
	return helpful, unhelpful
}
