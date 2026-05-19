package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"ft-backend/database"
	"ft-backend/models"

	"github.com/google/uuid"
)

const (
	cliSkillSampleDedupWindow       = time.Hour
	cliSkillSampleSimilarThreshold  = 8
	cliSkillSampleReview24hWindow   = 24 * time.Hour
)

type CLISkillSampleInput struct {
	Topic                  string
	Target                 string
	Command                string
	CLIVersion             string
	EvidenceKeys           []string
	EvidenceCompleteness   string
	RuleHit                bool
	UsedAI                 bool
	RequestID              string
	RootCauseDigest        string
	RecommendationDigest   string
	RootCauseSummary       string
	RecommendationSummary  string
	Status                 string
	Severity               string
	DurationMs             int64
	ErrorClassification    string
	ExecutionID            string
	SkillName              string
	PackKey                string
	Style                  string
	UserContext            map[string]string
}

type CLISkillSampleResult struct {
	Recorded                 bool                   `json:"recorded"`
	Deduplicated             bool                   `json:"deduplicated"`
	Classification           string                 `json:"classification"`
	EnhancementReviewCreated bool                   `json:"enhancement_review_created"`
	EnhancementReview        *SkillEnhancementReview `json:"enhancement_review,omitempty"`
	AutoIterationCreated     bool                   `json:"auto_iteration_created"`
	AutoIterationID          string                 `json:"auto_iteration_id,omitempty"`
	SimilarRecentCount       int                    `json:"similar_recent_count"`
}

var cliSkillSampleDedup struct {
	sync.Mutex
	entries map[string]time.Time
}

func init() {
	cliSkillSampleDedup.entries = map[string]time.Time{}
}

// IngestCLISkillSample persists a CLI check sample, evaluates enhancement, and may queue refine tasks.
func IngestCLISkillSample(userID uuid.UUID, in CLISkillSampleInput) (*CLISkillSampleResult, error) {
	reg := DefaultSkillRegistry()
	topic := strings.ToLower(strings.TrimSpace(in.Topic))
	if topic == "" {
		return nil, fmt.Errorf("topic required")
	}
	classification := classifyCLISkillSample(in)
	dedupKey := cliSkillSampleDedupKey(topic, in.Target, in.RootCauseDigest, classification)
	if cliSkillSampleRecentlySeen(dedupKey) {
		return &CLISkillSampleResult{
			Deduplicated:   true,
			Classification: classification,
		}, nil
	}
	markCLISkillSampleSeen(dedupKey)

	answer := strings.TrimSpace(in.RootCauseSummary)
	if answer == "" {
		answer = strings.TrimSpace(in.RecommendationSummary)
	}
	evidenceKeys := append([]string(nil), in.EvidenceKeys...)
	if len(evidenceKeys) == 0 && in.UserContext != nil {
		for k := range in.UserContext {
			evidenceKeys = append(evidenceKeys, k)
		}
	}
	review := EvaluateSkillEnhancement(reg, PostAICallRecord{
		Topic:        topic,
		CommandKind:  "check",
		SkillName:    strings.TrimSpace(in.SkillName),
		PackKey:      strings.TrimSpace(in.PackKey),
		Style:        strings.TrimSpace(in.Style),
		RequestID:    strings.TrimSpace(in.RequestID),
		Answer:       answer,
		UserContext:  sanitizeCLISampleContext(in.UserContext),
		EvidenceKeys: evidenceKeys,
		MatchedSkill: strings.TrimSpace(in.SkillName) != "" && !strings.Contains(in.SkillName, "_auto"),
	})
	review = adjustCLIEnhancementReview(review, in)
	noteNewTargetKindIfNeeded(reg, topic, in.Target, &review)
	similar := countSimilarCLISamples(reg, topic, in.RootCauseDigest, cliSkillSampleReview24hWindow)
	review.SimilarRecentCount = similar

	sample := DiagnoseSample{
		Time:                 time.Now().UTC(),
		Topic:                topic,
		SkillName:            strings.TrimSpace(in.SkillName),
		Style:                strings.TrimSpace(in.Style),
		UserContext:          sanitizeCLISampleContext(in.UserContext),
		EvidenceKey:          evidenceKeys,
		AnswerHead:           headRunes(answer, 600),
		AnswerTail:           tailRunes(answer, 400),
		AnswerLen:            len(answer),
		RequestID:            strings.TrimSpace(in.RequestID),
		CommandKind:          "check",
		PackKey:              strings.TrimSpace(in.PackKey),
		EnhancementReview:    &review,
		Target:               limitAuditText(in.Target, 200),
		Command:              limitAuditText(in.Command, 500),
		CLIVersion:           strings.TrimSpace(in.CLIVersion),
		UsedAI:               in.UsedAI,
		RuleHit:              in.RuleHit,
		EvidenceCompleteness: strings.TrimSpace(in.EvidenceCompleteness),
		RootCauseDigest:      strings.TrimSpace(in.RootCauseDigest),
		RecommendationDigest: strings.TrimSpace(in.RecommendationDigest),
		SampleStatus:         firstNonEmpty(strings.TrimSpace(in.Status), "success"),
		Severity:             strings.TrimSpace(in.Severity),
		DurationMs:           in.DurationMs,
		ErrorClassification:  strings.TrimSpace(in.ErrorClassification),
		ExecutionID:          strings.TrimSpace(in.ExecutionID),
		SampleSource:         "cli_check",
	}
	if err := AppendDiagnoseSample(reg, sample); err != nil {
		return nil, err
	}

	out := &CLISkillSampleResult{
		Recorded:           true,
		Classification:     classification,
		SimilarRecentCount: similar,
	}
	if review.NeedsEnhancement {
		out.EnhancementReviewCreated = true
		r := review
		out.EnhancementReview = &r
	}

	metaPatch := map[string]interface{}{
		"skill_sample_recorded":       true,
		"skill_sample_classification": classification,
		"skill_sample_similar_count":  similar,
	}
	if rc := strings.TrimSpace(in.RootCauseSummary); rc != "" {
		metaPatch["root_cause"] = limitAuditText(rc, 800)
	}
	if rs := strings.TrimSpace(in.RecommendationSummary); rs != "" {
		metaPatch["recommendation_summary"] = limitAuditText(rs, 400)
	}
	if review.NeedsEnhancement {
		metaPatch["skill_enhancement_review"] = reviewToMap(review)
		metaPatch["enhancement_review_triggered"] = true
	}

	autoID, created, err := maybeCreateSkillRefineAutoIteration(userID, in, review, similar, classification)
	if err != nil {
		return out, err
	}
	if created {
		out.AutoIterationCreated = true
		out.AutoIterationID = autoID
		metaPatch["auto_iteration_id"] = autoID
	}
	if execID := strings.TrimSpace(in.ExecutionID); execID != "" {
		_ = PatchExecutionRecordMetadata(execID, metaPatch)
	}
	return out, nil
}

func classifyCLISkillSample(in CLISkillSampleInput) string {
	if in.RuleHit {
		return "valuable_sample"
	}
	if !in.UsedAI {
		return "valuable_sample"
	}
	ec := strings.ToLower(strings.TrimSpace(in.EvidenceCompleteness))
	if ec == "missing" || ec == "partial" {
		return "diagnosis_insufficient"
	}
	if !in.RuleHit && in.UsedAI {
		return "rule_candidate"
	}
	return "valuable_sample"
}

func adjustCLIEnhancementReview(review SkillEnhancementReview, in CLISkillSampleInput) SkillEnhancementReview {
	if in.RuleHit && !in.UsedAI {
		review.SavingsScore = maxInt(review.SavingsScore, 10)
	}
	if !in.RuleHit && in.UsedAI {
		review.SavingsScore += 20
		review.Recommendations = appendUnique(review.Recommendations, "本地规则未命中但 AI 给出结论，可评估规则化以降本")
		review.SuggestedActions = appendUnique(review.SuggestedActions, "local_rule")
		review.NeedsEnhancement = true
		if review.Priority == "low" {
			review.Priority = "medium"
		}
	}
	ec := strings.ToLower(strings.TrimSpace(in.EvidenceCompleteness))
	if ec == "missing" || ec == "partial" {
		review.Recommendations = appendUnique(review.Recommendations, "证据不完整，需补采集字段")
		review.SuggestedActions = appendUnique(review.SuggestedActions, "add_probe_fields")
		review.NeedsEnhancement = true
		review.SavingsScore += 15
	}
	if review.SavingsScore > 100 {
		review.SavingsScore = 100
	}
	if review.NeedsEnhancement && review.Priority == "" {
		review.Priority = "medium"
	}
	return review
}

func maybeCreateSkillRefineAutoIteration(userID uuid.UUID, in CLISkillSampleInput, review SkillEnhancementReview, similar int, classification string) (string, bool, error) {
	settings, err := GetAutoIterationSettings()
	if err != nil || settings == nil || !settings.Enabled {
		return "", false, nil
	}
	source := models.AutoIterationSourceSkillRefine
	titlePrefix := "技能精炼"
	if classification == "rule_candidate" {
		source = models.AutoIterationSourceRuleCandidate
		titlePrefix = "规则候选"
	} else if !in.RuleHit && in.UsedAI && review.SavingsScore >= 50 {
		source = models.AutoIterationSourceAICostReduce
		titlePrefix = "AI 成本优化"
	} else if classification == "diagnosis_insufficient" {
		source = models.AutoIterationSourceDiagnosisGap
		titlePrefix = "诊断不足"
	}
	shouldCreate := similar >= cliSkillSampleSimilarThreshold && review.NeedsEnhancement
	if !shouldCreate && classification == "diagnosis_insufficient" && review.Priority == "high" {
		shouldCreate = true
	}
	if !shouldCreate {
		return "", false, nil
	}
	topic := strings.TrimSpace(in.Topic)
	pattern := limitAuditText(firstNonEmpty(in.RootCauseDigest, in.Target, topic), 40)
	title := fmt.Sprintf("%s: %s %s", titlePrefix, topic, pattern)
	cmd := strings.TrimSpace(in.Command)
	if cmd == "" {
		cmd = "ai-sre check " + topic
		if t := strings.TrimSpace(in.Target); t != "" {
			cmd += " " + t
		}
	}
	desc := fmt.Sprintf("复现: %s\n期望: 本地规则或完整证据覆盖\n当前: %s\n样本数: %d\n相似摘要: %s",
		cmd, limitAuditText(in.RootCauseSummary, 400), similar, pattern)
	status := models.AutoIterationStatusPending
	requiresApproval := settings.HighRiskRequiresApproval && review.Priority == "high"
	if requiresApproval {
		status = models.AutoIterationStatusAwaitingApproval
	}
	if !settings.AutoDispatchEnabled {
		status = models.AutoIterationStatusDraft
	}
	row := models.AutoIteration{
		Title:                      limitAuditText(title, 200),
		Description:                limitAuditText(desc, 2000),
		Command:                    limitAuditText(cmd, 2000),
		Status:                     status,
		Source:                     source,
		RiskLevel:                  models.AutoIterationRiskLow,
		RequiresSuperAdminApproval: requiresApproval,
		Topic:                      topic,
		Summary:                    limitAuditText(in.RootCauseSummary, 500),
		CreatedByUserID:            &userID,
		CreatedBy:                  "cli_sample",
		Metadata: MergeAgentTaskMetadata(models.NewJSONBFromMap(map[string]interface{}{
			"sample_classification": classification,
			"similar_recent_count":  similar,
			"execution_id":          strings.TrimSpace(in.ExecutionID),
			"root_cause_digest":     strings.TrimSpace(in.RootCauseDigest),
		})),
	}
	if err := databaseCreateAutoIteration(&row, "cli_sample", classification); err != nil {
		return "", false, err
	}
	if settings.DingTalkNotifyEnabled {
		notifyAutoIterationDingTalkKind(DingTalkKindTaskCreated, row, "", classification)
	}
	return row.ID.String(), status == models.AutoIterationStatusPending || status == models.AutoIterationStatusAwaitingApproval, nil
}

func countSimilarCLISamples(reg *SkillRegistry, topic, digest string, window time.Duration) int {
	if digest == "" {
		return 0
	}
	cutoff := time.Now().UTC().Add(-window)
	if n := countSimilarDiagnoseSamplesPG(topic, digest, cutoff); n > 0 {
		return n
	}
	if reg == nil {
		return 0
	}
	samples, err := reg.ReadRecentSamples(topic, 200)
	if err != nil {
		return 0
	}
	n := 0
	for _, s := range samples {
		if s.Time.Before(cutoff) {
			continue
		}
		if strings.TrimSpace(s.RootCauseDigest) == digest {
			n++
		}
	}
	return n
}

func cliSkillSampleDedupKey(topic, target, digest, classification string) string {
	raw := strings.Join([]string{topic, target, digest, classification}, "|")
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:16])
}

func cliSkillSampleRecentlySeen(key string) bool {
	cliSkillSampleDedup.Lock()
	defer cliSkillSampleDedup.Unlock()
	exp, ok := cliSkillSampleDedup.entries[key]
	if !ok {
		return false
	}
	if time.Since(exp) > cliSkillSampleDedupWindow {
		delete(cliSkillSampleDedup.entries, key)
		return false
	}
	return true
}

func markCLISkillSampleSeen(key string) {
	cliSkillSampleDedup.Lock()
	defer cliSkillSampleDedup.Unlock()
	cliSkillSampleDedup.entries[key] = time.Now()
	if len(cliSkillSampleDedup.entries) > 5000 {
		cutoff := time.Now().Add(-cliSkillSampleDedupWindow)
		for k, t := range cliSkillSampleDedup.entries {
			if t.Before(cutoff) {
				delete(cliSkillSampleDedup.entries, k)
			}
		}
	}
}

func sanitizeCLISampleContext(kv map[string]string) map[string]string {
	if kv == nil {
		return nil
	}
	out := stripBulkContextForSample(kv)
	denyKeys := []string{"password", "token", "secret", "authorization", "cookie", "webhook", "private_key", "api_key"}
	clean := make(map[string]string, len(out))
	for k, v := range out {
		kl := strings.ToLower(k)
		skip := false
		for _, d := range denyKeys {
			if strings.Contains(kl, d) {
				skip = true
				break
			}
		}
		if skip {
			clean[k] = "<redacted>"
			continue
		}
		if strings.Contains(strings.ToLower(v), "password=") || strings.Contains(strings.ToLower(v), "://") && strings.Contains(v, "@") {
			clean[k] = redactDSNLike(v)
			continue
		}
		clean[k] = v
	}
	return clean
}

func redactDSNLike(v string) string {
	if i := strings.Index(v, "://"); i >= 0 {
		rest := v[i+3:]
		if at := strings.Index(rest, "@"); at > 0 {
			return v[:i+3] + "<redacted>@" + rest[at+1:]
		}
	}
	if i := strings.Index(v, "@tcp("); i > 0 {
		return "<redacted>" + v[i:]
	}
	return "<redacted>"
}

func reviewToMap(r SkillEnhancementReview) map[string]interface{} {
	return map[string]interface{}{
		"needs_enhancement":      r.NeedsEnhancement,
		"priority":               r.Priority,
		"savings_score":          r.SavingsScore,
		"recommendations":        r.Recommendations,
		"suggested_actions":      r.SuggestedActions,
		"similar_recent_count":   r.SimilarRecentCount,
		"enhancement_status":     r.EnhancementStatus,
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// PatchExecutionRecordMetadata merges patch into execution_records.metadata.
func PatchExecutionRecordMetadata(execID string, patch map[string]interface{}) error {
	if database.DB == nil || len(patch) == 0 {
		return nil
	}
	id, err := uuid.Parse(strings.TrimSpace(execID))
	if err != nil {
		return nil
	}
	var rec models.ExecutionRecord
	if err := database.DB.Where("id = ?", id).First(&rec).Error; err != nil {
		return err
	}
	meta := decodeRecordMetadata(rec.Metadata)
	for k, v := range patch {
		meta[k] = v
	}
	return database.DB.Model(&rec).Update("metadata", models.NewJSONBFromMap(meta)).Error
}
