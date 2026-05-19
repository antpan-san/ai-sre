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
	cmd.Example = fmt.Sprintf(`  %s check kafka --lag 100000 --topic orders
  %s check k8s --pod pending
  %s check domain opsfleetpilot.com
  %s check domain -d domain=opsfleetpilot.com
  %s check go --pid 1234
  %s check elasticsearch -d base_url=http://127.0.0.1:9200
  %s -o json check domain opsfleetpilot.com
  %s check code OPSFLEET_K8S_E_PAUSE_MISSING`,
		progName, progName, progName, progName, progName, progName, progName, progName)
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
		Long: `topic 取值: kafka | k8s | nginx | redis | mysql | postgresql | elasticsearch | domain | dns

domain / dns：诊断公网或内网域名（DNS、HTTP(S)、TLS）
  · ai-sre check domain <fqdn>  例: check domain opsfleetpilot.com
  · 或 check domain -d domain=<fqdn>  可选 -d scheme=https -d port=443

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
	cmd.Flags().BoolVar(&noFeedback, "no-feedback", false, "禁用诊断后的反馈提示")
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
	if len(args) == 2 && !isDomainTopic(args[0]) {
		return fmt.Errorf("topic %q accepts 1 argument; domain/dns use: check domain <fqdn>", args[0])
	}
	return nil
}

func runCheckTopic(cmd *cobra.Command, args []string) error {
	ctx := buildContextMap()
	topic := args[0]
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
	if hasTopicEvidence(ctx) {
		ctx["diagnosis_style"] = "evidence_root_cause"
	}
	diag, err := runAnalyzeWithOrchestrator(context.Background(), topic, ctx)
	if err != nil {
		return err
	}
	res := &engine.RunResult{
		Answer:       diag.Answer,
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
	maybePromptFeedback(cmd.Context(), topic, diag)
	return nil
}

func checkCodeCmd() *cobra.Command {
	c := analyzeCodeCmd()
	return c
}
