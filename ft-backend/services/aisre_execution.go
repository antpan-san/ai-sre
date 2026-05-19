package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ft-backend/database"
	"ft-backend/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ApplyClientExecutionTopLevelScope limits rows to ai-sre client execution sessions (not child AI rows).
func ApplyClientExecutionTopLevelScope(db *gorm.DB) *gorm.DB {
	return db.Where("parent_execution_id IS NULL").Where(`
		COALESCE(metadata->>'record_kind','') IN ('client_execution','go_runtime')
		OR (source = 'cli' AND COALESCE(metadata->>'record_kind','') NOT IN ('ai_call'))
		OR (source = 'ai' AND COALESCE(metadata->>'record_kind','') = 'ai_call')
		OR (COALESCE(metadata->>'record_kind','') = 'cli_install' AND source IN ('cli','k8s','install'))
	`)
}

// ApplyClientExecutionMemberScope restricts to the current user's rows for role user.
func ApplyClientExecutionMemberScope(db *gorm.DB, role, username string) *gorm.DB {
	if models.IsAdminRole(role) {
		return db
	}
	u := strings.TrimSpace(username)
	if u == "" {
		return db.Where("1 = 0")
	}
	return db.Where("(created_by = ? OR trigger_user = ?)", u, u)
}

// ClientExecutionListQuery holds list filters for the client execution hub.
type ClientExecutionListQuery struct {
	TenantID        uuid.UUID
	Role            string
	Username        string
	View            string
	Status          string
	Topic           string
	Target          string
	SkillPack       string
	PackKey         string
	UsedAI          string
	Severity        string
	ClientVersion   string
	Machine         string
	HasAutoIter     bool
	StartDate       string
	EndDate         string
	Page            int
	PageSize        int
}

// ClientExecutionListItem is a row in the client execution hub.
type ClientExecutionListItem struct {
	ID                   string                 `json:"id"`
	Time                 time.Time              `json:"time"`
	Command              string                 `json:"command"`
	NormalizedCommand    string                 `json:"normalized_command,omitempty"`
	Target               string                 `json:"target,omitempty"`
	Topic                string                 `json:"topic,omitempty"`
	SkillPack            string                 `json:"skill_pack,omitempty"`
	PackKey              string                 `json:"pack_key,omitempty"`
	Status               string                 `json:"status"`
	Severity             string                 `json:"severity,omitempty"`
	Summary              string                 `json:"summary,omitempty"`
	RootCause            string                 `json:"root_cause,omitempty"`
	EvidenceCompleteness string                 `json:"evidence_completeness,omitempty"`
	AISource             string                 `json:"ai_source,omitempty"`
	UsedAI               bool                   `json:"used_ai"`
	RuleHit              bool                   `json:"rule_hit"`
	EnhancementNeeds     bool                   `json:"enhancement_needs,omitempty"`
	EnhancementPriority  string                 `json:"enhancement_priority,omitempty"`
	User                 string                 `json:"user,omitempty"`
	Machine              string                 `json:"machine,omitempty"`
	ClientVersion        string                 `json:"client_version,omitempty"`
	DurationMs           int64                  `json:"duration_ms,omitempty"`
	LegacyKind           string                 `json:"legacy_kind,omitempty"`
	HasAutoIteration          bool                   `json:"has_auto_iteration"`
	SkillSampleRecorded       bool                   `json:"skill_sample_recorded,omitempty"`
	SkillSampleClassification string                 `json:"skill_sample_classification,omitempty"`
	EnhancementReviewTriggered bool                  `json:"enhancement_review_triggered,omitempty"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
}

// ClientExecutionStats holds 24h dashboard counters for the hub.
type ClientExecutionStats struct {
	Total24h           int64 `json:"total_24h"`
	Success24h         int64 `json:"success_24h"`
	Failed24h          int64 `json:"failed_24h"`
	AICalls24h         int64 `json:"ai_calls_24h"`
	AutoIteration24h   int64 `json:"auto_iteration_24h"`
	IncompleteEvidence int64 `json:"incomplete_evidence_24h"`
	SkillSamples24h    int64 `json:"skill_samples_24h"`
	RuleHit24h         int64 `json:"rule_hit_24h"`
	EnhancementOpen24h int64 `json:"enhancement_open_24h"`
}

// ClientExecutionDetail aggregates a session and related child records.
type ClientExecutionDetail struct {
	Record         models.ExecutionRecord   `json:"record"`
	LegacyKind     string                   `json:"legacy_kind,omitempty"`
	Children       []models.ExecutionRecord `json:"children,omitempty"`
	Events         []models.ExecutionEvent  `json:"events,omitempty"`
	Timeline       []ClientExecutionPhase   `json:"timeline,omitempty"`
	RuntimeReport  *RuntimeReportSummary    `json:"runtime_report,omitempty"`
	Enhancement    map[string]interface{}   `json:"enhancement_review,omitempty"`
	AutoIterationID string                  `json:"auto_iteration_id,omitempty"`
	SkillSampleRecorded       bool            `json:"skill_sample_recorded,omitempty"`
	SkillSampleClassification string          `json:"skill_sample_classification,omitempty"`
	EnhancementReviewTriggered bool           `json:"enhancement_review_triggered,omitempty"`
}

type ClientExecutionPhase struct {
	Phase   string    `json:"phase"`
	Message string    `json:"message"`
	Time    time.Time `json:"time,omitempty"`
	Level   string    `json:"level,omitempty"`
}

type RuntimeReportSummary struct {
	SessionID     string `json:"session_id"`
	TargetDisplay string `json:"target_display,omitempty"`
	RootCause     string `json:"root_cause,omitempty"`
	SampleCount   int    `json:"sample_count,omitempty"`
	DiagnosisSource string `json:"diagnosis_source,omitempty"`
}

func ListClientExecutions(q ClientExecutionListQuery) ([]ClientExecutionListItem, int64, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 20
	}
	db := database.DB.Model(&models.ExecutionRecord{}).Where("tenant_id = ?", q.TenantID)
	db = ApplyClientExecutionTopLevelScope(db)
	db = ApplyClientExecutionMemberScope(db, q.Role, q.Username)
	db = applyClientExecutionListFilters(db, q)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []models.ExecutionRecord
	offset := (q.Page - 1) * q.PageSize
	if err := db.Order("created_at DESC").Offset(offset).Limit(q.PageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]ClientExecutionListItem, 0, len(rows))
	for i := range rows {
		item, err := buildClientExecutionListItem(&rows[i])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, nil
}

func GetClientExecutionStats(tenantID uuid.UUID, role, username string, since time.Time) (ClientExecutionStats, error) {
	base := database.DB.Model(&models.ExecutionRecord{}).Where("tenant_id = ?", tenantID).Where("created_at >= ?", since)
	base = ApplyClientExecutionTopLevelScope(base)
	base = ApplyClientExecutionMemberScope(base, role, username)

	var st ClientExecutionStats
	base.Count(&st.Total24h)
	base.Where("status = ?", models.ExecutionStatusSuccess).Count(&st.Success24h)
	base.Where("status IN ?", []string{models.ExecutionStatusFailed, models.ExecutionStatusCancelled}).Count(&st.Failed24h)

	childAI := database.DB.Model(&models.ExecutionRecord{}).
		Where("tenant_id = ?", tenantID).
		Where("created_at >= ?", since).
		Where("COALESCE(metadata->>'record_kind','') = ?", "ai_call")
	childAI = ApplyClientExecutionMemberScope(childAI, role, username)
	childAI.Count(&st.AICalls24h)

	incomplete := database.DB.Model(&models.ExecutionRecord{}).Where("tenant_id = ?", tenantID).Where("created_at >= ?", since)
	incomplete = ApplyClientExecutionTopLevelScope(incomplete)
	incomplete = ApplyClientExecutionMemberScope(incomplete, role, username)
	incomplete.Where("COALESCE(metadata->>'evidence_completeness','') IN ('partial','missing','incomplete')").Count(&st.IncompleteEvidence)

	autoIter := database.DB.Model(&models.ExecutionRecord{}).Where("tenant_id = ?", tenantID).Where("created_at >= ?", since)
	autoIter = ApplyClientExecutionTopLevelScope(autoIter)
	autoIter = ApplyClientExecutionMemberScope(autoIter, role, username)
	autoIter.Where("COALESCE(metadata->>'auto_iteration_id','') <> ''").Count(&st.AutoIteration24h)

	samples := database.DB.Model(&models.ExecutionRecord{}).Where("tenant_id = ?", tenantID).Where("created_at >= ?", since)
	samples = ApplyClientExecutionTopLevelScope(samples)
	samples = ApplyClientExecutionMemberScope(samples, role, username)
	samples.Where("COALESCE(metadata->>'skill_sample_recorded','') IN ('true','1')").Count(&st.SkillSamples24h)

	ruleHit := database.DB.Model(&models.ExecutionRecord{}).Where("tenant_id = ?", tenantID).Where("created_at >= ?", since)
	ruleHit = ApplyClientExecutionTopLevelScope(ruleHit)
	ruleHit = ApplyClientExecutionMemberScope(ruleHit, role, username)
	ruleHit.Where("COALESCE(metadata->>'rule_hit','') IN ('true','1')").Count(&st.RuleHit24h)

	enhOpen := database.DB.Model(&models.ExecutionRecord{}).Where("tenant_id = ?", tenantID).Where("created_at >= ?", since)
	enhOpen = ApplyClientExecutionTopLevelScope(enhOpen)
	enhOpen = ApplyClientExecutionMemberScope(enhOpen, role, username)
	enhOpen.Where("COALESCE(metadata->>'enhancement_review_triggered','') IN ('true','1') OR COALESCE(metadata->'skill_enhancement_review'->>'needs_enhancement','') IN ('true','1')").Count(&st.EnhancementOpen24h)

	return st, nil
}

func GetClientExecutionDetail(id uuid.UUID, role, username string) (*ClientExecutionDetail, error) {
	var rec models.ExecutionRecord
	if err := database.DB.Where("id = ?", id).First(&rec).Error; err != nil {
		return nil, err
	}
	if !clientExecutionVisible(rec, role, username) {
		return nil, gorm.ErrRecordNotFound
	}
	detail := &ClientExecutionDetail{Record: rec, LegacyKind: legacyKindForRecord(rec)}

	var children []models.ExecutionRecord
	database.DB.Where("parent_execution_id = ?", rec.ID).Order("created_at ASC").Find(&children)
	if len(children) == 0 && rec.CorrelationID != "" {
		database.DB.Where("correlation_id = ? AND id <> ?", rec.CorrelationID, rec.ID).
			Where("COALESCE(metadata->>'record_kind','') = ?", "ai_call").
			Order("created_at ASC").Find(&children)
	}
	detail.Children = children

	var events []models.ExecutionEvent
	database.DB.Where("execution_id = ?", rec.ID).Order("created_at ASC").Limit(200).Find(&events)
	detail.Events = events
	detail.Timeline = buildClientExecutionTimeline(rec, events, children)

	meta := decodeRecordMetadata(rec.Metadata)
	if sid, _ := meta["runtime_watch_session_id"].(string); strings.TrimSpace(sid) != "" {
		detail.RuntimeReport = loadRuntimeReportSummary(strings.TrimSpace(sid))
	}
	if v, _ := meta["skill_enhancement_review"].(map[string]interface{}); len(v) > 0 {
		detail.Enhancement = v
	}
	if aid, _ := meta["auto_iteration_id"].(string); strings.TrimSpace(aid) != "" {
		detail.AutoIterationID = strings.TrimSpace(aid)
	}
	detail.SkillSampleRecorded = boolMeta(meta, "skill_sample_recorded")
	detail.SkillSampleClassification = strMeta(meta, "skill_sample_classification")
	detail.EnhancementReviewTriggered = boolMeta(meta, "enhancement_review_triggered")
	if detail.EnhancementReviewTriggered == false && detail.Enhancement != nil {
		detail.EnhancementReviewTriggered = boolMeta(detail.Enhancement, "needs_enhancement")
	}
	return detail, nil
}

func applyClientExecutionListFilters(db *gorm.DB, q ClientExecutionListQuery) *gorm.DB {
	switch strings.TrimSpace(q.View) {
	case "check":
		db = db.Where("category IN ? OR category LIKE ?", []string{"check", "analyze"}, "check%")
	case "probe":
		db = db.Where("category = ? OR category LIKE ?", "probe", "probe%")
	case "deploy":
		db = db.Where("category LIKE ? OR category LIKE ? OR category LIKE ? OR COALESCE(metadata->>'record_kind','') = ?",
			"k8s_%", "node_%", "install%", "cli_install")
	case "failed":
		db = db.Where("status IN ?", []string{models.ExecutionStatusFailed, models.ExecutionStatusCancelled})
	case "auto_iteration":
		db = db.Where("COALESCE(metadata->>'auto_iteration_id','') <> ''")
	}
	if s := strings.TrimSpace(q.Status); s != "" {
		db = db.Where("status = ?", s)
	}
	if t := strings.TrimSpace(q.Topic); t != "" {
		db = db.Where("COALESCE(metadata->>'topic','') = ? OR category = ?", t, t)
	}
	if t := strings.TrimSpace(q.Target); t != "" {
		like := "%" + t + "%"
		db = db.Where("target_host ILIKE ? OR resource_name ILIKE ?", like, like)
	}
	if sp := strings.TrimSpace(q.SkillPack); sp != "" {
		db = db.Where("COALESCE(metadata->>'skill_pack','') = ?", sp)
	}
	if pk := strings.TrimSpace(q.PackKey); pk != "" {
		db = db.Where("COALESCE(metadata->>'pack_key','') = ?", pk)
	}
	if v := strings.TrimSpace(q.ClientVersion); v != "" {
		db = db.Where("COALESCE(metadata->>'version','') = ?", v)
	}
	if m := strings.TrimSpace(q.Machine); m != "" {
		like := "%" + m + "%"
		db = db.Where("target_host ILIKE ? OR COALESCE(metadata->>'hostname','') ILIKE ?", like, like)
	}
	if q.HasAutoIter {
		db = db.Where("COALESCE(metadata->>'auto_iteration_id','') <> ''")
	}
	if strings.TrimSpace(q.StartDate) != "" {
		db = db.Where("created_at >= ?", q.StartDate)
	}
	if strings.TrimSpace(q.EndDate) != "" {
		db = db.Where("created_at <= ?", q.EndDate)
	}
	return db
}

func buildClientExecutionListItem(rec *models.ExecutionRecord) (ClientExecutionListItem, error) {
	meta := decodeRecordMetadata(rec.Metadata)
	item := ClientExecutionListItem{
		ID:         rec.ID.String(),
		Time:       rec.CreatedAt,
		Command:    rec.Command,
		Status:     rec.Status,
		DurationMs: rec.DurationMs,
		LegacyKind: legacyKindForRecord(*rec),
		Metadata:   meta,
	}
	if v, _ := meta["normalized_command"].(string); v != "" {
		item.NormalizedCommand = v
	}
	if v, _ := meta["topic"].(string); v != "" {
		item.Topic = v
	} else {
		item.Topic = inferTopicFromCategory(rec.Category)
	}
	item.Target = firstNonEmpty(rec.TargetHost, rec.ResourceName, strMeta(meta, "diagnosis_target"))
	item.SkillPack = firstNonEmpty(strMeta(meta, "skill_pack"), strMeta(meta, "skill_name"))
	item.PackKey = strMeta(meta, "pack_key")
	item.Severity = strMeta(meta, "severity")
	item.RootCause = firstNonEmpty(strMeta(meta, "root_cause"), strMeta(meta, "root_cause_summary"))
	item.Summary = firstNonEmpty(rec.StdoutSummary, strMeta(meta, "summary"))
	item.EvidenceCompleteness = strMeta(meta, "evidence_completeness")
	item.ClientVersion = strMeta(meta, "version")
	item.Machine = firstNonEmpty(rec.TargetHost, strMeta(meta, "hostname"))
	item.User = firstNonEmpty(rec.TriggerUser, rec.CreatedBy)
	item.UsedAI = boolMeta(meta, "used_ai") || hasAIChild(rec)
	item.RuleHit = boolMeta(meta, "rule_hit")
	item.AISource = strMeta(meta, "ai_source")
	if er, ok := meta["skill_enhancement_review"].(map[string]interface{}); ok && len(er) > 0 {
		item.EnhancementNeeds = boolMeta(er, "needs_enhancement")
		item.EnhancementPriority = strMeta(er, "priority")
	}
	if item.AISource == "" && item.UsedAI {
		item.AISource = "platform_ai"
	}
	item.HasAutoIteration = strings.TrimSpace(strMeta(meta, "auto_iteration_id")) != ""
	item.SkillSampleRecorded = boolMeta(meta, "skill_sample_recorded")
	item.SkillSampleClassification = strMeta(meta, "skill_sample_classification")
	item.EnhancementReviewTriggered = boolMeta(meta, "enhancement_review_triggered") || item.EnhancementNeeds

	if database.DB != nil && rec.ID != uuid.Nil {
		var childCount int64
		database.DB.Model(&models.ExecutionRecord{}).
			Where("parent_execution_id = ? OR (correlation_id = ? AND COALESCE(metadata->>'record_kind','') = ? AND id <> ?)",
				rec.ID, rec.CorrelationID, "ai_call", rec.ID).
			Count(&childCount)
		if childCount > 0 {
			item.UsedAI = true
			if item.AISource == "" {
				item.AISource = "platform_ai"
			}
		}
	}
	return item, nil
}

func hasAIChild(rec *models.ExecutionRecord) bool {
	if database.DB == nil || rec.ID == uuid.Nil {
		return false
	}
	var n int64
	database.DB.Model(&models.ExecutionRecord{}).
		Where("parent_execution_id = ?", rec.ID).
		Where("COALESCE(metadata->>'record_kind','') = ?", "ai_call").
		Count(&n)
	return n > 0
}

func legacyKindForRecord(rec models.ExecutionRecord) string {
	meta := decodeRecordMetadata(rec.Metadata)
	kind := strMeta(meta, "record_kind")
	if kind == "client_execution" {
		return ""
	}
	if rec.Source == "ai" && kind == "ai_call" {
		return "legacy_ai_diagnose"
	}
	if kind == "go_runtime" {
		return "legacy_go_runtime"
	}
	if rec.Source == "cli" && kind == "" {
		return "legacy_cli"
	}
	return ""
}

func clientExecutionVisible(rec models.ExecutionRecord, role, username string) bool {
	if models.IsAdminRole(role) {
		return true
	}
	u := strings.TrimSpace(username)
	if u == "" {
		return false
	}
	return rec.CreatedBy == u || rec.TriggerUser == u
}

func buildClientExecutionTimeline(rec models.ExecutionRecord, events []models.ExecutionEvent, children []models.ExecutionRecord) []ClientExecutionPhase {
	out := []ClientExecutionPhase{{Phase: "cli_start", Message: "CLI 执行开始", Time: rec.CreatedAt, Level: "info"}}
	for _, e := range events {
		out = append(out, ClientExecutionPhase{Phase: e.Phase, Message: e.Message, Time: e.CreatedAt, Level: e.Level})
	}
	for _, c := range children {
		meta := decodeRecordMetadata(c.Metadata)
		msg := "AI 分析: " + c.Category
		if c.Status != models.ExecutionStatusSuccess {
			msg += " (" + c.Status + ")"
		}
		out = append(out, ClientExecutionPhase{Phase: "ai_analysis", Message: msg, Time: c.CreatedAt, Level: "info"})
		_ = meta
	}
	if rec.FinishedAt != nil {
		out = append(out, ClientExecutionPhase{Phase: "finish", Message: "执行结束: " + rec.Status, Time: *rec.FinishedAt, Level: "info"})
	}
	return out
}

func loadRuntimeReportSummary(sessionID string) *RuntimeReportSummary {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil
	}
	var sess models.RuntimeWatchSession
	if database.DB.Where("id = ?", sid).First(&sess).Error != nil {
		return nil
	}
	return &RuntimeReportSummary{
		SessionID:       sessionID,
		TargetDisplay:   sess.TargetDisplay,
		RootCause:       sess.RootCause,
		SampleCount:     sess.SampleCount,
		DiagnosisSource: sess.DiagnosisSource,
	}
}

func decodeRecordMetadata(raw models.JSONB) map[string]interface{} {
	var out map[string]interface{}
	if len(raw) == 0 {
		return map[string]interface{}{}
	}
	_ = json.Unmarshal(raw, &out)
	if out == nil {
		return map[string]interface{}{}
	}
	return out
}

func strMeta(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		return strings.TrimSpace(toString(v))
	}
	return ""
}

func boolMeta(m map[string]interface{}, key string) bool {
	v, ok := m[key]
	if !ok {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true" || t == "1" || t == "yes"
	}
	return false
}

func toString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		s := string(b)
		if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
			return s[1 : len(s)-1]
		}
		return s
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func inferTopicFromCategory(category string) string {
	c := strings.ToLower(strings.TrimSpace(category))
	switch {
	case c == "check", c == "analyze", strings.HasPrefix(c, "check"):
		return ""
	case c == "probe":
		return ""
	default:
		return c
	}
}
