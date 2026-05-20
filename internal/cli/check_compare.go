package cli

import (
	"context"
	"encoding/json"
	"os"
	"strings"
)

type checkAICompareResult struct {
	Enabled        bool   `json:"enabled"`
	LocalRootCause string `json:"local_root_cause,omitempty"`
	AIRootCause    string `json:"ai_root_cause,omitempty"`
	LocalDigest    string `json:"local_digest,omitempty"`
	AIDigest       string `json:"ai_digest,omitempty"`
	Match          bool   `json:"match"`
	ShadowError    string `json:"shadow_error,omitempty"`
	AISource       string `json:"ai_source,omitempty"`
}

func maybeCompareCheckWithAI(ctx context.Context, topic string, ctxMap map[string]string, local *diagnoseResponse) *checkAICompareResult {
	if local == nil || !checkCompareAI {
		return nil
	}
	if !strings.EqualFold(strings.TrimSpace(local.Source), "local-rule") {
		return nil
	}
	localRoot, _, _ := splitAnswerSections(local)
	if localRoot == "" {
		localRoot = strings.TrimSpace(local.Answer)
	}
	out := &checkAICompareResult{
		Enabled:        true,
		LocalRootCause: truncateBytes(localRoot, 400),
		LocalDigest:    digestText(localRoot),
	}
	aiDiag, err := runAnalyzeWithOrchestrator(ctx, topic, ctxMap)
	if err != nil {
		out.ShadowError = err.Error()
		return out
	}
	aiRoot, _, _ := splitAnswerSections(aiDiag)
	if aiRoot == "" {
		aiRoot = strings.TrimSpace(aiDiag.Answer)
	}
	out.AIRootCause = truncateBytes(aiRoot, 400)
	out.AIDigest = digestText(aiRoot)
	out.AISource = aiSourceLabel(aiDiag)
	out.Match = out.LocalDigest != "" && out.LocalDigest == out.AIDigest
	return out
}

func formatCheckCompareText(cmp *checkAICompareResult) string {
	if cmp == nil || !cmp.Enabled {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n--- 规则 vs AI 对比（shadow）---\n")
	b.WriteString("本地规则: ")
	b.WriteString(strings.TrimSpace(cmp.LocalRootCause))
	b.WriteByte('\n')
	if cmp.ShadowError != "" {
		b.WriteString("AI shadow: （失败）")
		b.WriteString(cmp.ShadowError)
		b.WriteByte('\n')
		return strings.TrimSpace(b.String())
	}
	b.WriteString("AI shadow: ")
	b.WriteString(strings.TrimSpace(cmp.AIRootCause))
	b.WriteByte('\n')
	if cmp.Match {
		b.WriteString("结论一致: 是\n")
	} else {
		b.WriteString("结论一致: 否\n")
	}
	return strings.TrimSpace(b.String())
}

func checkCompareToMap(cmp *checkAICompareResult) map[string]interface{} {
	if cmp == nil {
		return nil
	}
	return map[string]interface{}{
		"enabled":          cmp.Enabled,
		"local_root_cause": cmp.LocalRootCause,
		"ai_root_cause":    cmp.AIRootCause,
		"local_digest":     cmp.LocalDigest,
		"ai_digest":        cmp.AIDigest,
		"match":            cmp.Match,
		"shadow_error":     cmp.ShadowError,
		"ai_source":        cmp.AISource,
	}
}

func printCheckJSONResultWithCompare(topic, target string, diag *diagnoseResponse, ctx map[string]string, usedAI bool, cmp *checkAICompareResult) error {
	payload := buildCheckJSONResult(topic, target, diag, ctx, usedAI)
	if m := checkCompareToMap(cmp); m != nil {
		payload["ai_compare"] = m
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}
