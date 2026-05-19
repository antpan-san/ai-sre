package services

import (
	"strings"

	"ft-backend/models"
)

const (
	FeedbackSourceCLICheck    = "cli_check"
	FeedbackSourceCLISkills   = "cli_skills"
	FeedbackSourceConsoleExec = "console_execution"
)

func inferFeedbackSource(payload map[string]interface{}) string {
	if payload == nil {
		return FeedbackSourceCLICheck
	}
	if src := strings.TrimSpace(strMeta(payload, "source")); src != "" {
		return src
	}
	if strings.TrimSpace(strMeta(payload, "sample_source")) == "cli_check" {
		return FeedbackSourceCLICheck
	}
	if _, ok := payload["helpful"].(bool); ok && strings.TrimSpace(strMeta(payload, "execution_id")) != "" {
		return FeedbackSourceConsoleExec
	}
	return FeedbackSourceCLICheck
}

func populateAutoIterationFeedbackFields(fb *models.AutoIterationFeedback, source, topic, command, summary string, payload map[string]interface{}) {
	enriched := enrichFeedbackPayload(topic, command, summary, payload)
	if strings.TrimSpace(source) == "" {
		source = inferFeedbackSource(enriched)
	}
	fb.RawPayload = models.NewJSONBFromMap(enriched)
	fb.Source = source
	if strings.TrimSpace(fb.Topic) == "" {
		fb.Topic = strings.TrimSpace(topic)
	}
	fb.Command = limitAuditText(firstNonEmpty(strMeta(enriched, "command"), command), 2000)
	fb.Summary = limitAuditText(firstNonEmpty(strMeta(enriched, "summary"), summary), 2000)
	fb.RequestID = strMeta(enriched, "request_id")
	fb.ExecutionID = strMeta(enriched, "execution_id")
	fb.SkillName = firstNonEmpty(strMeta(enriched, "skill_name"), strMeta(enriched, "skill_pack"))
	fb.EvidenceCompleteness = strMeta(enriched, "evidence_completeness")
	fb.RootCauseDigest = strMeta(enriched, "root_cause_digest")
	fb.RecommendationDigest = strMeta(enriched, "recommendation_digest")
	fb.EvidenceDigest = strMeta(enriched, "evidence_digest")
	if v, ok := enriched["helpful"].(bool); ok {
		fb.Helpful = &v
	}
	if v, ok := enriched["used_ai"].(bool); ok {
		fb.UsedAI = &v
	}
	if v, ok := enriched["rule_hit"].(bool); ok {
		fb.RuleHit = &v
	}
}

func feedbackAdminItemFromModel(row models.AutoIterationFeedback) AutoIterationFeedbackAdminItem {
	item := AutoIterationFeedbackAdminItem{
		ID:                   row.ID,
		CreatedAt:            row.CreatedAt,
		Topic:                row.Topic,
		Classification:       row.Classification,
		NeedIteration:        row.NeedIteration,
		UserMessage:          row.UserMessage,
		AutoIterationID:      row.AutoIterationID,
		Source:               row.Source,
		RequestID:            row.RequestID,
		ExecutionID:          row.ExecutionID,
		Command:              row.Command,
		Summary:              row.Summary,
		SkillName:            row.SkillName,
		Helpful:              row.Helpful,
		RuleHit:              row.RuleHit,
		UsedAI:               row.UsedAI,
		EvidenceCompleteness: row.EvidenceCompleteness,
	}
	if item.RequestID == "" || item.ExecutionID == "" || item.Helpful == nil {
		meta := decodeRecordMetadata(row.RawPayload)
		if item.RequestID == "" {
			item.RequestID = strMeta(meta, "request_id")
		}
		if item.ExecutionID == "" {
			item.ExecutionID = strMeta(meta, "execution_id")
		}
		if item.Command == "" {
			item.Command = strMeta(meta, "command")
		}
		if item.Summary == "" {
			item.Summary = strMeta(meta, "summary")
		}
		if item.Helpful == nil {
			if v, ok := meta["helpful"].(bool); ok {
				item.Helpful = &v
			}
		}
		if item.Source == "" {
			item.Source = inferFeedbackSource(meta)
		}
	}
	return item
}
