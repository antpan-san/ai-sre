package go_runtime

import (
	"strings"
)

// DiagnosisConclusion is the user-facing root cause conclusion uploaded to OpsFleet.
type DiagnosisConclusion struct {
	RootCause string `json:"root_cause"`
	Evidence  string `json:"evidence"`
	Source    string `json:"source,omitempty"` // local | ai
}

// ApplyDiagnosis sets Summary and Diagnosis from a concluded root cause.
func ApplyDiagnosis(wr *WatchReport, level, rootCause, evidence, source string) {
	if wr == nil {
		return
	}
	rootCause = strings.TrimSpace(rootCause)
	evidence = strings.TrimSpace(evidence)
	if rootCause == "" {
		return
	}
	if level == "" {
		level = "WARN"
	}
	wr.Diagnosis = DiagnosisConclusion{
		RootCause: rootCause,
		Evidence:  evidence,
		Source:    source,
	}
	wr.Summary = ReportSummary{
		Level:    level,
		Title:    rootCause,
		Evidence: evidence,
	}
}

// DeriveLocalDiagnosis tries to conclude root cause from proc/trend/K8s findings without AI.
func DeriveLocalDiagnosis(wr *WatchReport) (ReportSummary, bool) {
	if wr == nil {
		return ReportSummary{}, false
	}
	if len(wr.Samples) > 0 {
		s := SummarizeWatchReport(wr)
		if s.Level != "OK" && !strings.Contains(s.Title, "未发现明显") {
			return s, true
		}
	}
	top := topFinding(wr.TrendFindings)
	if top == nil {
		return ReportSummary{}, false
	}
	if strings.Contains(top.Title, "已基于 Kubernetes 状态完成部分诊断") {
		return ReportSummary{}, false
	}
	if strings.Contains(top.Title, "采集器镜像拉取失败") {
		rc := "诊断采集器镜像无法拉取，未能采集宿主机进程指标"
		ev := strings.TrimSpace(top.Evidence)
		if ev == "" {
			ev = probeSnippet(wr.ProbeBundle, "kubectl_collector_describe", 1200)
		}
		return ReportSummary{Level: "CRITICAL", Title: rc, Evidence: ev}, true
	}
	if strings.Contains(top.Title, "诊断采集器未能启动") && collectorImagePullInBundle(wr.ProbeBundle) {
		rc := "诊断采集器镜像无法拉取，未能采集宿主机进程指标"
		ev := probeSnippet(wr.ProbeBundle, "kubectl_collector_describe", 1200)
		return ReportSummary{Level: "CRITICAL", Title: rc, Evidence: ev}, true
	}
	cause := strings.TrimSpace(top.Cause)
	if cause == "" {
		cause = strings.TrimSpace(top.Title)
	}
	ev := strings.TrimSpace(top.Evidence)
	if top.Severity == severityCrit {
		if ev == "" {
			return ReportSummary{}, false
		}
		return ReportSummary{Level: "CRITICAL", Title: cause, Evidence: ev}, true
	}
	if top.Severity == severityWarn && ev != "" && !isGenericWarn(cause) {
		return ReportSummary{Level: "WARN", Title: cause, Evidence: ev}, true
	}
	return ReportSummary{}, false
}

func isGenericWarn(cause string) bool {
	cause = strings.TrimSpace(cause)
	if cause == "" {
		return true
	}
	for _, p := range []string{"可能", "建议", "常见为", "需", "或"} {
		if strings.Contains(cause, p) && len(cause) < 40 {
			return true
		}
	}
	return false
}

func probeSnippet(bundle map[string]string, key string, max int) string {
	if bundle == nil {
		return ""
	}
	return truncateProbe(strings.TrimSpace(bundle[key]), max)
}

// StripFindingHints removes operator instructions from exported findings.
func StripFindingHints(wr *WatchReport) {
	if wr == nil {
		return
	}
	for i := range wr.TrendFindings {
		wr.TrendFindings[i].Verify = ""
	}
	for _, s := range wr.Samples {
		if s == nil {
			continue
		}
		for i := range s.Findings {
			s.Findings[i].Verify = ""
		}
	}
}
