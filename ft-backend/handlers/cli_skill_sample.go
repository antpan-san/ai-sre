package handlers

import (
	"ft-backend/common/logger"
	"ft-backend/common/response"
	"ft-backend/services"

	"github.com/gin-gonic/gin"
)

type cliSkillSampleRequest struct {
	Topic                  string            `json:"topic"`
	Target                 string            `json:"target"`
	Command                string            `json:"command"`
	CLIVersion             string            `json:"cli_version"`
	EvidenceKeys           []string          `json:"evidence_keys"`
	EvidenceCompleteness   string            `json:"evidence_completeness"`
	RuleHit                bool              `json:"rule_hit"`
	UsedAI                 bool              `json:"used_ai"`
	RequestID              string            `json:"request_id"`
	RootCauseDigest        string            `json:"root_cause_digest"`
	RecommendationDigest   string            `json:"recommendation_digest"`
	RootCauseSummary       string            `json:"root_cause_summary"`
	RecommendationSummary  string            `json:"recommendation_summary"`
	Status                 string            `json:"status"`
	Severity               string            `json:"severity"`
	DurationMs             int64             `json:"duration_ms"`
	ErrorClassification    string            `json:"error_classification"`
	ExecutionID            string            `json:"execution_id"`
	SkillName              string            `json:"skill_name"`
	PackKey                string            `json:"pack_key"`
	Style                  string            `json:"style"`
	Context                map[string]string `json:"context"`
}

// PostCLISkillSample ingests a CLI check skill sample for accumulation and refinement.
func PostCLISkillSample(c *gin.Context) {
	ident, ok := resolveCLIBearerIdentity(c)
	if !ok {
		return
	}
	var req cliSkillSampleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数")
		return
	}
	out, err := services.IngestCLISkillSample(ident.UserID, services.CLISkillSampleInput{
		Topic:                 req.Topic,
		Target:                req.Target,
		Command:               req.Command,
		CLIVersion:            req.CLIVersion,
		EvidenceKeys:          req.EvidenceKeys,
		EvidenceCompleteness:  req.EvidenceCompleteness,
		RuleHit:               req.RuleHit,
		UsedAI:                req.UsedAI,
		RequestID:             req.RequestID,
		RootCauseDigest:       req.RootCauseDigest,
		RecommendationDigest:  req.RecommendationDigest,
		RootCauseSummary:      req.RootCauseSummary,
		RecommendationSummary: req.RecommendationSummary,
		Status:                req.Status,
		Severity:              req.Severity,
		DurationMs:            req.DurationMs,
		ErrorClassification:   req.ErrorClassification,
		ExecutionID:           req.ExecutionID,
		SkillName:             req.SkillName,
		PackKey:               req.PackKey,
		Style:                 req.Style,
		UserContext:           req.Context,
	})
	if err != nil {
		logger.Error("PostCLISkillSample: %v", err)
		response.ServerError(c, "样本入库失败")
		return
	}
	response.OK(c, out)
}
