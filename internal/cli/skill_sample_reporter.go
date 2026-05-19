package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type skillSampleReportInput struct {
	Topic                string
	Target               string
	Command              string
	EvidenceKeys         []string
	EvidenceCompleteness string
	RuleHit              bool
	UsedAI               bool
	RequestID            string
	RootCauseSummary     string
	RecommendationSummary string
	SkillName            string
	PackKey              string
	Style                string
	Context              map[string]string
	DurationMs           int64
	Status               string
}

type skillSampleReportResult struct {
	Recorded             bool   `json:"recorded"`
	Deduplicated         bool   `json:"deduplicated"`
	Classification       string `json:"classification"`
	AutoIterationCreated bool   `json:"auto_iteration_created"`
	AutoIterationID      string `json:"auto_iteration_id,omitempty"`
}

var skillSampleDedup struct {
	sync.Mutex
	entries map[string]time.Time
}

func init() {
	skillSampleDedup.entries = map[string]time.Time{}
}

func reportCheckSkillSample(ctx context.Context, in skillSampleReportInput) *skillSampleReportResult {
	if os.Getenv("OPSFLEET_SKILL_SAMPLE_DISABLED") == "1" {
		return nil
	}
	base := strings.TrimRight(strings.TrimSpace(resolveOpsfleetAPIBase()), "/")
	if base == "" || strings.TrimSpace(resolveOpsfleetToken()) == "" {
		return nil
	}
	rootDigest := digestText(in.RootCauseSummary)
	recDigest := digestText(in.RecommendationSummary)
	classification := classifyCheckSample(in)
	dedupKey := digestText(strings.Join([]string{in.Topic, in.Target, rootDigest, classification}, "|"))
	if skillSampleRecentlyReported(dedupKey) {
		return &skillSampleReportResult{Deduplicated: true, Classification: classification}
	}
	markSkillSampleReported(dedupKey)

	ctxKeys := evidenceKeysFromContext(in.Context, in.EvidenceKeys)
	body, err := json.Marshal(map[string]interface{}{
		"topic":                   strings.TrimSpace(in.Topic),
		"target":                  strings.TrimSpace(in.Target),
		"command":                 strings.TrimSpace(in.Command),
		"cli_version":             Version,
		"evidence_keys":           ctxKeys,
		"evidence_completeness":   strings.TrimSpace(in.EvidenceCompleteness),
		"rule_hit":                in.RuleHit,
		"used_ai":                 in.UsedAI,
		"request_id":              strings.TrimSpace(in.RequestID),
		"root_cause_digest":       rootDigest,
		"recommendation_digest":   recDigest,
		"root_cause_summary":      truncateBytes(strings.TrimSpace(in.RootCauseSummary), 800),
		"recommendation_summary":  truncateBytes(strings.TrimSpace(in.RecommendationSummary), 400),
		"status":                  firstNonEmptyStr(in.Status, "success"),
		"duration_ms":             in.DurationMs,
		"execution_id":            ActiveExecutionRecordID(),
		"skill_name":              strings.TrimSpace(in.SkillName),
		"pack_key":                strings.TrimSpace(in.PackKey),
		"style":                   strings.TrimSpace(in.Style),
		"context":                 scrubSampleContext(in.Context),
	})
	if err != nil {
		return nil
	}
	endpoint := base + "/api/cli/skill-samples"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil
	}
	req.Header.Set("Content-Type", "application/json")
	attachOpsfleetAuth(req)
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return nil
	}
	var env struct {
		Code int                     `json:"code"`
		Data skillSampleReportResult `json:"data"`
	}
	if json.Unmarshal(raw, &env) != nil || env.Code != 200 {
		return nil
	}
	out := env.Data
	if out.Recorded {
		meta := map[string]interface{}{
			"skill_sample_recorded":     true,
			"skill_sample_classification": out.Classification,
		}
		if out.AutoIterationID != "" {
			meta["auto_iteration_id"] = out.AutoIterationID
		}
		MergeExecutionFinishMeta(meta)
	}
	maybeTriggerCheckFeedbackAnalyze(ctx, in, classification)
	return &out
}

func maybeTriggerCheckFeedbackAnalyze(ctx context.Context, in skillSampleReportInput, classification string) {
	switch classification {
	case "diagnosis_insufficient", "bug", "product_gap", "ai_failure":
	default:
		return
	}
	if strings.TrimSpace(resolveOpsfleetToken()) == "" {
		return
	}
	cmd := strings.TrimSpace(in.Command)
	if cmd == "" {
		cmd = fmt.Sprintf("ai-sre check %s", strings.TrimSpace(in.Topic))
	}
	summary := truncateBytes(strings.TrimSpace(in.RootCauseSummary), 400)
	extra := map[string]interface{}{
		"classification":         classification,
		"rule_hit":               in.RuleHit,
		"used_ai":                in.UsedAI,
		"evidence_completeness":  in.EvidenceCompleteness,
		"execution_id":           ActiveExecutionRecordID(),
	}
	_, _ = callCLIFeedbackAnalyze(ctx, in.Topic, cmd, summary, extra)
}

func classifyCheckSample(in skillSampleReportInput) string {
	if in.RuleHit {
		return "valuable_sample"
	}
	ec := strings.ToLower(strings.TrimSpace(in.EvidenceCompleteness))
	if ec == "missing" || ec == "partial" {
		return "diagnosis_insufficient"
	}
	if in.UsedAI {
		return "rule_candidate"
	}
	return "valuable_sample"
}

func digestText(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:12])
}

func evidenceKeysFromContext(ctx map[string]string, extra []string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(k string) {
		k = strings.TrimSpace(k)
		if k == "" {
			return
		}
		if _, ok := seen[k]; ok {
			return
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	for _, k := range extra {
		add(k)
	}
	for k := range ctx {
		add(k)
	}
	return out
}

func scrubSampleContext(ctx map[string]string) map[string]string {
	if ctx == nil {
		return nil
	}
	out := make(map[string]string, len(ctx))
	deny := []string{"password", "token", "secret", "authorization", "cookie", "webhook", "private_key", "api_key", "config", "client_config"}
	for k, v := range ctx {
		kl := strings.ToLower(k)
		skip := false
		for _, d := range deny {
			if strings.Contains(kl, d) {
				skip = true
				break
			}
		}
		if skip {
			out[k] = "<redacted>"
			continue
		}
		if strings.HasSuffix(k, "_diagnose_json") || strings.HasSuffix(k, "_probe_json") {
			out[k] = "<evidence_omitted>"
			continue
		}
		if len(v) > 200 {
			v = v[:200] + "..."
		}
		out[k] = v
	}
	return out
}

func skillSampleRecentlyReported(key string) bool {
	if key == "" {
		return false
	}
	skillSampleDedup.Lock()
	defer skillSampleDedup.Unlock()
	exp, ok := skillSampleDedup.entries[key]
	if !ok {
		return false
	}
	if time.Since(exp) > time.Hour {
		delete(skillSampleDedup.entries, key)
		return false
	}
	return true
}

func markSkillSampleReported(key string) {
	skillSampleDedup.Lock()
	defer skillSampleDedup.Unlock()
	skillSampleDedup.entries[key] = time.Now()
}

func firstNonEmptyStr(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return strings.TrimSpace(a)
	}
	return strings.TrimSpace(b)
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func finishCheckSkillSampleAsync(in skillSampleReportInput) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = reportCheckSkillSample(ctx, in)
	}()
}
