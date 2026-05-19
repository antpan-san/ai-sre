package handlers

import (
	"strings"

	"ft-backend/common/logger"
	"ft-backend/services"
)

func recordPostAISuccess(in services.PostAICallRecord) services.SkillEnhancementReview {
	review := services.EvaluateSkillEnhancement(services.DefaultSkillRegistry(), in)
	go func(in services.PostAICallRecord, review services.SkillEnhancementReview) {
		defer func() { _ = recover() }()
		if err := services.RecordPostAICallWithReview(services.DefaultSkillRegistry(), in, review); err != nil {
			logger.Error("RecordPostAICall topic=%s failed: %v", in.Topic, err)
		}
	}(in, review)
	return review
}

func diagnosisStyleFromContext(kv map[string]string) string {
	if kv == nil {
		return ""
	}
	return strings.TrimSpace(kv["diagnosis_style"])
}

func enhancementReviewToMeta(r services.SkillEnhancementReview) map[string]interface{} {
	return map[string]interface{}{
		"needs_enhancement":    r.NeedsEnhancement,
		"priority":             r.Priority,
		"savings_score":        r.SavingsScore,
		"recommendations":      r.Recommendations,
		"suggested_actions":    r.SuggestedActions,
		"similar_recent_count": r.SimilarRecentCount,
	}
}
