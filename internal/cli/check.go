package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/panshuai/ai-sre/internal/engine"
	"github.com/panshuai/ai-sre/internal/output"
	"github.com/spf13/cobra"
)

// checkCmd is the canonical AI-powered incident diagnosis command.
func checkCmd() *cobra.Command {
	cmd := newCheckTopicCommand()
	cmd.Use = "check [topic]"
	cmd.Short = "AI 故障诊断（技能包；未购买时每日免费 5 次）"
	cmd.Example = fmt.Sprintf(`  %s check redis
  %s check redis 192.168.56.11:6379
  %s check kafka
  %s check k8s --pod pending
  %s check domain opsfleetpilot.com
  %s check go --pid 1234
  %s -o json check domain opsfleetpilot.com
  %s check code OPSFLEET_K8S_E_PAUSE_MISSING
  %s check linux`,
		progName, progName, progName, progName, progName, progName, progName, progName, progName)
	cmd.AddCommand(checkCodeCmd(), checkGoCmd())
	return cmd
}

// analyzeCmd is a deprecated alias of check [topic].
func analyzeCmd() *cobra.Command {
	cmd := newCheckTopicCommand()
	cmd.Use = "analyze [topic]"
	cmd.Short = "（已弃用）请改用 check"
	cmd.Deprecated = "use \"check\" instead"
	cmd.Example = fmt.Sprintf(`  %s check kafka --lag 100000`, progName)
	cmd.AddCommand(analyzeCodeCmd())
	return cmd
}

func newCheckTopicCommand() *cobra.Command {
	cmd := &cobra.Command{
		Long: `topic 取值: kafka | k8s | nginx | redis | mysql | postgresql | elasticsearch | domain | dns | linux

linux：本机 Linux 性能诊断（CPU/负载/内存/磁盘/进程/泄露风险）
  · ai-sre check linux
  · 仅采集：ai-sre probe linux [--duration 10s] [--top 10] [--pid <pid>] [--json]

中间件（redis / kafka / mysql / postgresql / elasticsearch）：
  · 最简：ai-sre check redis  （默认连本机常用端口；可用环境变量覆盖，见 AI_SRE_REDIS_ADDR）
  · 指定目标：ai-sre check redis <host:port>  或仅 host（自动补默认端口）
  · 高级场景仍可用 -d / --set，且优先于默认值

domain / dns：DNS、HTTP(S)、TLS 只读采集（纯文本报告）+ 服务端 AI 分析（非 K8s）
  · ai-sre check domain <fqdn>  例: check domain opsfleetpilot.com
  · 仅采集不调用 AI: ai-sre probe domain <fqdn>

k8s 场景 --pod 可填：
  · 问题类型：pending、crashloop、instability（与 --issue 一致）
  · 具体 Pod 名称：如 kube-controller-manager-k8s-master-0（可配合 --namespace）

本机有 kubectl 且可连集群时，会在调用服务端 AI 前自动只读采集；有证据时优先走 evidence_root_cause 模板。`,
		Args: checkTopicArgs,
		RunE: runCheckTopic,
	}
	cmd.Flags().StringVar(&lag, "lag", "", "Kafka consumer lag 等指标")
	cmd.Flags().StringVar(&topicFlag, "topic", "", "Kafka topic 名称")
	cmd.Flags().StringVar(&pod, "pod", "", "K8s: pending/crashloop/instability 或具体 Pod 名（可配 --namespace）")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Kubernetes namespace")
	cmd.Flags().StringVar(&issue, "issue", "", "K8s: pending | crashloop")
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
		return fmt.Errorf("at most 2 arguments: check <topic> [target]")
	}
	if len(args) == 2 {
		if !checkTopicAcceptsOptionalTarget(args[0]) {
			return fmt.Errorf("topic %q 不接受位置参数目标，请用专用 flag 或子命令", args[0])
		}
		if err := validateCheckTargetLiteral(strings.TrimSpace(args[1])); err != nil {
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
	ctx := buildContextMap()
	topic := args[0]
	applyCheckTargetContext(ctx, topic, args)
	mergeDomainIntoContext(ctx, topic, args)
	if isDomainTopic(topic) && strings.TrimSpace(ctx["domain"]) == "" {
		return fmt.Errorf("domain 诊断需要域名：check domain <fqdn> 或 -d domain=<fqdn>")
	}
	if isGoRuntimeAnalyzeTopic(topic) {
		goRuntimeOpts.Namespace = namespace
		goRuntimeOpts.Pod = pod
		return runGoRuntimeAnalyze(cmd.Context(), topic, goRuntimeOpts)
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
	diag, err := runAnalyzeWithOrchestrator(context.Background(), topic, ctx)
	if err != nil {
		return err
	}
	answer := formatCheckAnswerText(topic, diag.Answer)
	if isDomainTopic(topic) && !strings.EqualFold(outputFormat, "json") {
		if probe := strings.TrimSpace(ctx["domain_probe_text"]); probe != "" {
			answer = probe + "\n\n--- AI 分析 ---\n\n" + strings.TrimSpace(answer)
		}
	}
	res := &engine.RunResult{
		Answer:       answer,
		SkillName:    diag.SkillName,
		SkillDisplay: diag.SkillDisplay,
	}
	commandKind := "check"
	if cmd.Name() == "analyze" {
		commandKind = "analyze"
	}
	p := output.BuildPayload(commandKind, topic, "", "", ctx, !noRAG, 0, res)
	if err := output.Print(outputFormat, p); err != nil {
		return err
	}
	finishMeta := map[string]interface{}{
		"topic":        topic,
		"used_ai":      strings.EqualFold(strings.TrimSpace(diag.Source), "server-ai") || strings.TrimSpace(diag.Answer) != "",
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
	return nil
}

func checkCodeCmd() *cobra.Command {
	c := analyzeCodeCmd()
	return c
}
