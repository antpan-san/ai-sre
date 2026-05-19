package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/panshuai/ai-sre/internal/engine"
	"github.com/panshuai/ai-sre/internal/output"
	"github.com/spf13/cobra"
)

// checkCmd is the canonical incident diagnosis command.
func checkCmd() *cobra.Command {
	cmd := newCheckTopicCommand()
	cmd.Use = "check <topic> [target]"
	cmd.Short = "统一排查入口（自动采集 + 本地规则优先 + 必要时 AI）"
	cmd.Example = fmt.Sprintf(`  %s check redis 127.0.0.1:6379
  %s check linux
  %s check domain opsfleetpilot.com
  %s check k8s pod/default/api-0
  %s check go pid/1234
  %s check kafka 127.0.0.1:9092
  %s check mysql 'user:pass@tcp(127.0.0.1:3306)/'
  %s check postgresql 'postgres://user:pass@127.0.0.1:5432/db?sslmode=disable'
  %s check nginx /var/log/nginx/access.log
  %s check elasticsearch http://127.0.0.1:9200
  %s check code OPSFLEET_K8S_E_PAUSE_MISSING
  %s -o json check redis 127.0.0.1:6379`,
		progName, progName, progName, progName, progName, progName, progName, progName, progName, progName, progName, progName)
	return cmd
}

func newCheckTopicCommand() *cobra.Command {
	cmd := &cobra.Command{
		Long: fmt.Sprintf(`topic: %s

别名: postgres→postgresql, es→elasticsearch, dns/url→domain, system/host→linux

流程: 解析目标 → 自动采集证据 → 本地规则 → 必要时服务端 AI → 技能包增强审查`,
			strings.Join(publicCheckTopicList(), " | ")),
		Args: checkTopicArgs,
		RunE: runCheckTopic,
	}
	cmd.Flags().StringVar(&lag, "lag", "", "Kafka consumer lag 等指标")
	cmd.Flags().StringVar(&topicFlag, "topic", "", "Kafka topic 名称")
	cmd.Flags().StringVar(&pod, "pod", "", "K8s Pod 或问题类型 pending/crashloop/instability")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Kubernetes namespace")
	cmd.Flags().StringVar(&issue, "issue", "", "K8s: pending | crashloop | instability")
	cmd.Flags().StringVar(&code, "code", "", "HTTP 状态码，如 502")
	cmd.Flags().StringVar(&upstream, "upstream", "", "Nginx upstream 名称或服务名")
	cmd.Flags().StringVar(&latency, "latency", "", "延迟描述，如 50ms、p99=20ms")
	cmd.Flags().StringToStringVarP(&setKV, "set", "d", nil, "附加上下文 key=value，可多次使用")
	cmd.Flags().BoolVar(&diagnosticPlanYes, "yes", false, "非 TTY 环境确认执行服务端只读诊断任务单")
	return cmd
}

func checkTopicArgs(cmd *cobra.Command, args []string) error {
	if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
		return err
	}
	if len(args) > 2 {
		return fmt.Errorf("最多 2 个参数: check <topic> [target]")
	}
	canonical := normalizeCheckTopicAlias(args[0])
	if !isRegisteredCheckTopic(canonical) {
		return fmt.Errorf("未知 topic %q，可用: %s", args[0], strings.Join(publicCheckTopicList(), ", "))
	}
	if checkTopicRequiresTarget(canonical) && len(args) < 2 {
		return fmt.Errorf("topic %q 需要 target 参数", canonical)
	}
	if len(args) == 2 {
		if !checkTopicAcceptsOptionalTarget(args[0]) {
			return fmt.Errorf("topic %q 不接受位置参数 target，请用 flag", canonical)
		}
		if err := validateCheckTargetForTopic(canonical, strings.TrimSpace(args[1])); err != nil {
			return err
		}
	}
	return nil
}

func aiSourceLabel(d *diagnoseResponse) string {
	if d == nil {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(d.Source)) {
	case "server-ai":
		return "platform_ai"
	case "local", "local-rule", "local_skill":
		return "local_rule"
	default:
		if strings.TrimSpace(d.Source) != "" {
			return "mixed"
		}
	}
	return "local_rule"
}

func evidenceCompletenessForContext(ctx map[string]string) string {
	if len(ctx) == 0 {
		return "missing"
	}
	if hasTopicEvidence(ctx) {
		return "complete"
	}
	return "partial"
}

func runCheckTopic(cmd *cobra.Command, args []string) error {
	rawTopic := args[0]
	topic := normalizeCheckTopicAlias(rawTopic)
	target := checkTargetDisplay(topic, args, nil)

	if topic == "code" {
		codeVal := strings.ToUpper(strings.TrimSpace(args[1]))
		return runAnalyzeErrorCode(cmd.Context(), codeVal, "")
	}

	var goOpts goRuntimeCLIOptions
	goOpts.Namespace = namespace
	goOpts.Pod = pod

	ctx := buildContextMap()
	if err := applyUnifiedCheckTarget(ctx, topic, args, &goOpts); err != nil {
		return err
	}
	target = checkTargetDisplay(topic, args, ctx)

	if topic == "go" {
		return runSmartGoRuntimeDiagnose(cmd.Context(), goOpts)
	}

	if isDomainTopic(topic) && strings.TrimSpace(ctx["domain"]) == "" {
		return fmt.Errorf("domain 排查需要域名: check domain <fqdn>")
	}

	for k, v := range gatherTopicEvidence(cmd.Context(), topic, ctx) {
		ctx[k] = v
	}
	if err := finishRedisCheckEvidence(topic, ctx); err != nil {
		return err
	}
	if err := finishLinuxCheckEvidence(topic, ctx); err != nil {
		return err
	}
	if shouldRequestServerDiagnosticPlan(topic, ctx) {
		obs, ran, err := maybeRunServerDiagnosticPlan(cmd.Context(), topic, ctx, diagnosticPlanYes)
		if err != nil {
			return err
		}
		if ran {
			for k, v := range obs {
				ctx[k] = v
			}
		}
	}
	if isDomainTopic(topic) {
		ctx["diagnosis_style"] = "domain_connectivity"
	} else if isLinuxPerformanceTopic(topic) && hasTopicEvidence(ctx) {
		ctx["diagnosis_style"] = "linux_performance_evidence"
	} else if isMiddlewareEvidenceTopic(topic) && hasTopicEvidence(ctx) {
		ctx["diagnosis_style"] = "middleware_evidence"
	} else if hasTopicEvidence(ctx) {
		ctx["diagnosis_style"] = "evidence_root_cause"
	}
	stripSensitiveCheckContext(ctx)

	var diag *diagnoseResponse
	var err error
	usedAI := false

	if local, ok := tryLocalCheckRules(topic, ctx); ok {
		diag = local
	} else {
		diag, err = runAnalyzeWithOrchestrator(context.Background(), topic, ctx)
		if err != nil {
			return err
		}
		usedAI = strings.EqualFold(strings.TrimSpace(diag.Source), "server-ai")
	}

	answer := formatCheckAnswerText(topic, diag.Answer)
	if !usedAI && strings.EqualFold(strings.TrimSpace(diag.Source), "local-rule") {
		answer = diag.Answer
	} else if !strings.Contains(answer, "【根因结论】") {
		answer = formatCheckStructuredText(checkStructuredResult{
			RootCause:       strings.TrimSpace(diag.Answer),
			EvidenceLevel:   evidenceCompletenessForContext(ctx),
			UsedAI:          usedAI,
			Recommendations: []string{},
		})
	}

	if isDomainTopic(topic) && !strings.EqualFold(outputFormat, "json") && usedAI {
		if probe := strings.TrimSpace(ctx["domain_probe_text"]); probe != "" {
			answer = probe + "\n\n--- AI 分析 ---\n\n" + strings.TrimSpace(formatCheckAnswerText(topic, diag.Answer))
		}
	}

	if strings.EqualFold(outputFormat, "json") {
		if err := printCheckJSONResult(topic, target, diag, ctx, usedAI); err != nil {
			return err
		}
	} else {
		fmt.Println(answer)
	}

	finishMeta := map[string]interface{}{
		"topic":        topic,
		"target":       target,
		"used_ai":      usedAI,
		"rule_hit":     strings.EqualFold(strings.TrimSpace(diag.Source), "local-rule"),
		"ai_source":    aiSourceLabel(diag),
		"skill_name":   diag.SkillName,
		"skill_pack":   diag.SkillName,
		"summary":      truncateBytes(strings.TrimSpace(diag.Answer), 400),
	}
	if diag.Metadata != nil {
		if r, ok := diag.Metadata["skill_enhancement_review"].(map[string]interface{}); ok {
			finishMeta["skill_enhancement_review"] = r
		}
	}
	finishMeta["evidence_completeness"] = evidenceCompletenessForContext(ctx)
	MergeExecutionFinishMeta(finishMeta)

	_ = output.BuildPayload("check", topic, "", "", ctx, !noRAG, 0, &engine.RunResult{
		Answer:       answer,
		SkillName:    diag.SkillName,
		SkillDisplay: diag.SkillDisplay,
	})
	return nil
}
