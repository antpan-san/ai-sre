package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/panshuai/ai-sre/internal/config"
	"github.com/panshuai/ai-sre/internal/engine"
	"github.com/panshuai/ai-sre/internal/llm"
	"github.com/panshuai/ai-sre/internal/loader"
	"github.com/panshuai/ai-sre/internal/output"
	"github.com/panshuai/ai-sre/internal/quota"
)

var (
	// progName is the argv0-style program name (ai-sre vs opsfleet-executor); set by newRoot.
	progName          string
	configFile        string
	keyFile           string
	verbose           bool
	noRAG             bool
	outputFormat      string
	skillsExtraDir    string
	knowledgeExtraDir string
	lag               string
	topicFlag         string
	pod               string
	namespace         string
	issue             string
	code              string
	upstream          string
	latency           string
	setKV             map[string]string
	noFeedback        bool
	goRuntimeOpts     goRuntimeCLIOptions
)

// Execute runs the Cobra root (entry from main) as ai-sre.
func Execute() {
	ExecuteAs("ai-sre")
}

// ExecuteAs runs the same CLI tree under a different program name (e.g. opsfleet-executor on managed hosts).
func ExecuteAs(programName string) {
	root := newRoot(programName)
	// 当用户调用一个本版本不认识的顶层子命令（例如旧版 ai-sre 0.4.4
	// 收到「ai-sre node tune time-sync」），cobra 会在 PersistentPreRunE 之前
	// 直接报 unknown command。我们在这里做一次 pre-flight：若 OpsFleet
	// 有更新版本则覆盖本机二进制并 re-exec 同一参数，使自升级链路真正能
	// "带新命令过来"。失败或已是最新则按 cobra 原有行为继续报错。
	if programName == "ai-sre" {
		preflightAutoUpgradeIfUnknown(root)
	}
	reporter := newExecutionReporter(programName, os.Args[1:])
	reporter.start()
	if err := root.Execute(); err != nil {
		reporter.finish(err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	reporter.finish(nil)
}

func preflightAutoUpgradeIfUnknown(root *cobra.Command) {
	if os.Getenv("OPSFLEET_NO_AUTO_UPGRADE") == "1" {
		return
	}
	args := os.Args[1:]
	if len(args) == 0 || isHelpInvocation() {
		return
	}
	var first string
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			first = a
			break
		}
	}
	if first == "" {
		return
	}
	switch first {
	case "upgrade", "version", "help", "completion", "doctor":
		return
	}
	if _, _, err := root.Find(args); err == nil {
		return
	}
	base := resolveOpsfleetAPIBase()
	_ = tryAutoUpgradeInPlace(base)
}

func newRoot(programName string) *cobra.Command {
	progName = programName
	var short, long string
	if programName == "opsfleet-executor" {
		short = "OpsFleet 本地执行器 — 与 ai-sre 相同的技能包与执行语义"
		long = fmt.Sprintf(`在需要部署或运维的受管机器上运行；与 ai-sre 共用同一套技能包（YAML）、Prompt、轻量 RAG 与 LLM 编排（需凭据）。
子命令与 flag 与 ai-sre 一致：analyze / ask / runbook / skills / doctor / version / k8s / job。
示例:
  %s analyze kafka --lag 100000
  %s analyze k8s --pod pending
  %s elasticsearch diagnose 127.0.0.1:9200
  %s ask "kafka lag 高怎么办"
  %s runbook "pod频繁重启"
  %s skills list
  %s k8s download --api-url http://host:9080/ft-api -u USER -p PASS --cluster c1 --version v1.35.4 --master 10.0.0.1`, programName, programName, programName, programName, programName, programName, programName)
	} else {
		short = "AI SRE Copilot — 故障诊断、Runbook、知识问答"
		long = fmt.Sprintf(`CLI 工具：技能包（Skill Pack）+ Prompt + 可选轻量 RAG + DeepSeek LLM；并支持通过 OpsFleet API 拉取 K8s 离线包与安装/卸载（见 k8s 子命令）。
示例:
  %s analyze kafka --lag 100000
  %s analyze k8s --pod pending
  %s elasticsearch diagnose 127.0.0.1:9200
  %s ask "kafka lag 高怎么办"
  %s runbook "pod频繁重启"
  %s skills list
  %s k8s download --api-url http://host:9080/ft-api -u USER -p PASS --cluster c1 --version v1.35.4 --master 10.0.0.1`, programName, programName, programName, programName, programName, programName, programName)
	}
	root := &cobra.Command{
		Use:          programName,
		Short:        short,
		Long:         long,
		SilenceUsage: true,
	}
	if programName == "ai-sre" {
		root.Aliases = []string{"ops-ai"}
	}
	root.PersistentFlags().StringVar(&configFile, "config", "", "path to config.yaml (api_key, optional base_url, model); default: ~/.config/ai-sre/config.yaml")
	root.PersistentFlags().StringVar(&keyFile, "key-file", "", "path to file containing API key only (overrides default api_key file if --config not set)")
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose logs")
	root.PersistentFlags().BoolVar(&noRAG, "no-rag", false, "disable knowledge retrieval (RAG)")
	root.PersistentFlags().StringVarP(&outputFormat, "output", "o", "text", "output format: text|json (structured answer for analyze/ask/runbook)")
	root.PersistentFlags().StringVar(&skillsExtraDir, "skills-dir", "", "extra directory of *.yaml skill packs (merged with built-in; same name overrides)")
	root.PersistentFlags().StringVar(&knowledgeExtraDir, "knowledge-dir", "", "extra directory of *.md files for RAG (merged with built-in knowledge)")

	cmds := []*cobra.Command{analyzeCmd(), diagnoseCmd(), askCmd(), runbookCmd(), skillsCmd(), doctorCmd(), versionCmd(), upgradeCmd(), k8sCmd(), serviceCmd(), kafkaCmd(), redisCmd(), mysqlCmd(), nginxCmd(), elasticsearchCmd(), nodeCmd(), jobCmd()}
	if programName == "ai-sre" {
		cmds = append(cmds, uninstallCmd())
	}
	root.AddCommand(cmds...)
	if programName == "ai-sre" {
		root.PersistentPreRunE = opsfleetPersistentPreRun
	}
	return root
}

func analyzeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze [topic]",
		Short: "故障诊断（AI 技能包；未购买时每日免费 5 次）",
		Long: `topic 取值: kafka | k8s | nginx | redis | elasticsearch

k8s 场景 --pod 可填：
  · 问题类型：pending、crashloop、instability（与 --issue 一致）
  · 具体 Pod 名称：如 kube-controller-manager-k8s-master-0（可配合 --namespace；省略时在集群内按 metadata.name 解析命名空间）

当 topic 为 k8s 且本机存在可执行的 kubectl、当前上下文可连集群时，会在调用服务端诊断前自动执行只读 kubectl 采集；若指定了具体 Pod，会额外抓取该 Pod 的 describe、events、logs（含 --previous）并优先送入模型。默认仍会采集节点与 Pending 等全景，并触发两轮服务端推理。`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := buildContextMap()
			topic := args[0]
			if isGoRuntimeAnalyzeTopic(topic) {
				goRuntimeOpts.Namespace = namespace
				goRuntimeOpts.Pod = pod
				return runGoRuntimeAnalyze(cmd.Context(), topic, goRuntimeOpts)
			}
			for k, v := range gatherTopicEvidence(cmd.Context(), topic, ctx) {
				ctx[k] = v
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
			p := output.BuildPayload("analyze", topic, "", "", ctx, !noRAG, 0, res)
			if err := output.Print(outputFormat, p); err != nil {
				return err
			}
			// Optional in-flow feedback: prompts the user "本次结果是否帮你定位了根因 (y/N/note)?"
			// Skipped automatically when stdin/stderr isn't a TTY, when --no-feedback is set,
			// when -o json is requested, or when running with verbose noise.
			maybePromptFeedback(cmd.Context(), topic, diag)
			return nil
		},
	}
	cmd.Flags().StringVar(&lag, "lag", "", "Kafka consumer lag 等指标")
	cmd.Flags().StringVar(&topicFlag, "topic", "", "Kafka topic 名称")
	cmd.Flags().StringVar(&pod, "pod", "", "K8s: 问题类型 pending/crashloop/instability，或具体 Pod 名称（可配 --namespace）")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Kubernetes namespace")
	cmd.Flags().StringVar(&issue, "issue", "", "K8s: pending | crashloop")
	cmd.Flags().StringVar(&code, "code", "", "HTTP 状态码，如 502")
	cmd.Flags().StringVar(&upstream, "upstream", "", "Nginx upstream 名称或服务名")
	cmd.Flags().StringVar(&latency, "latency", "", "延迟描述，如 50ms、p99=20ms")
	cmd.Flags().StringToStringVarP(&setKV, "set", "d", nil, "附加上下文 key=value，可多次使用")
	cmd.Flags().BoolVar(&noFeedback, "no-feedback", false, "禁用诊断后的「是否帮到我」反馈提示（非 TTY、-o json 也会自动跳过）")
	cmd.Example = fmt.Sprintf(`  %s analyze kafka --lag 100000 --topic orders
  %s analyze k8s --pod pending
  %s analyze k8s --pod kube-controller-manager-k8s-master-0 -n kube-system
  %s analyze go-runtime --pid 1234
  %s analyze elasticsearch -d base_url=http://127.0.0.1:9200
  %s -o json analyze kafka --lag 1
  %s analyze code OPSFLEET_K8S_E_PAUSE_MISSING
  %s analyze code OPSFLEET_K8S_E_APISERVER_TIMEOUT --detail "$(journalctl -u kubelet -n 30)"`,
		progName, progName, progName, progName, progName, progName, progName, progName)
	bindGoRuntimeAnalyzeFlags(cmd, &goRuntimeOpts)
	cmd.AddCommand(analyzeCodeCmd())
	return cmd
}

func isGoRuntimeAnalyzeTopic(topic string) bool {
	switch strings.ToLower(strings.TrimSpace(topic)) {
	case "go-runtime", "pod-go":
		return true
	default:
		return false
	}
}

func askCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ask [question]",
		Short: "知识库问答（AI 技能包；未购买时每日免费 5 次）",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			q := strings.Join(args, " ")
			t0 := time.Now()
			var serverErr error
			if strings.TrimSpace(resolveOpsfleetAPIBase()) != "" {
				ans, err := callServerAsk(cmd.Context(), q, noRAG)
				if err == nil && strings.TrimSpace(ans) != "" {
					ms := time.Since(t0).Milliseconds()
					p := output.BuildPayload("ask", "", q, "", nil, !noRAG, ms, serverAIResult(ans))
					return output.Print(outputFormat, p)
				}
				if err != nil {
					serverErr = err
				} else {
					serverErr = errors.New("服务端返回空回答")
				}
			}
			eng, err := bootstrap()
			if err != nil {
				if serverErr != nil {
					return fmt.Errorf("服务端 ask 失败: %v; 本机无 LLM 凭据: %w", serverErr, err)
				}
				return err
			}
			res, err := eng.Ask(context.Background(), q, !noRAG)
			if err != nil {
				if serverErr != nil {
					return fmt.Errorf("服务端 ask 失败: %v; 本机 ask 失败: %w", serverErr, err)
				}
				return err
			}
			ms := time.Since(t0).Milliseconds()
			p := output.BuildPayload("ask", "", q, "", nil, !noRAG, ms, res)
			return output.Print(outputFormat, p)
		},
	}
}

func runbookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runbook [scenario]",
		Short: "生成 Runbook 文档（AI 技能包；未购买时每日免费 5 次）",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scenario := strings.Join(args, " ")
			ctx := map[string]string{}
			if len(setKV) > 0 {
				for k, v := range setKV {
					ctx[k] = v
				}
			}
			t0 := time.Now()
			var serverErr error
			if strings.TrimSpace(resolveOpsfleetAPIBase()) != "" {
				ans, err := callServerRunbook(cmd.Context(), scenario, ctx)
				if err == nil && strings.TrimSpace(ans) != "" {
					ms := time.Since(t0).Milliseconds()
					p := output.BuildPayload("runbook", "", "", scenario, ctx, !noRAG, ms, serverAIResult(ans))
					return output.Print(outputFormat, p)
				}
				if err != nil {
					serverErr = err
				} else {
					serverErr = errors.New("服务端返回空回答")
				}
			}
			eng, err := bootstrap()
			if err != nil {
				if serverErr != nil {
					return fmt.Errorf("服务端 runbook 失败: %v; 本机无 LLM 凭据: %w", serverErr, err)
				}
				return err
			}
			res, err := eng.Runbook(context.Background(), scenario, ctx, !noRAG)
			if err != nil {
				if serverErr != nil {
					return fmt.Errorf("服务端 runbook 失败: %v; 本机 runbook 失败: %w", serverErr, err)
				}
				return err
			}
			ms := time.Since(t0).Milliseconds()
			p := output.BuildPayload("runbook", "", "", scenario, ctx, !noRAG, ms, res)
			return output.Print(outputFormat, p)
		},
	}
	cmd.Flags().StringToStringVarP(&setKV, "set", "d", nil, "附加上下文 key=value")
	return cmd
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(progName, Version)
		},
	}
}

func buildContextMap() map[string]string {
	ctx := map[string]string{}
	if len(setKV) > 0 {
		for k, v := range setKV {
			ctx[k] = v
		}
	}
	put := func(k, v string) {
		if strings.TrimSpace(v) != "" {
			ctx[k] = v
		}
	}
	put("lag", lag)
	put("topic", topicFlag)
	put("pod", pod)
	put("namespace", namespace)
	put("issue", issue)
	put("status_code", code)
	put("upstream", upstream)
	put("latency_p99", latency)
	return ctx
}

func bootstrap() (*engine.Engine, error) {
	llmCfg, limits, credSrc, err := config.LoadLLM(configFile, keyFile)
	if err != nil {
		return nil, err
	}
	cacheDir, err := quota.DefaultCacheDir()
	if err != nil {
		return nil, err
	}
	if limits != nil && limits.MaxLLMCallsPerDay > 0 {
		if err := quota.TakeDaily(cacheDir, limits.MaxLLMCallsPerDay); err != nil {
			return nil, err
		}
	}
	client, err := llm.NewFromConfig(llmCfg)
	if err != nil {
		return nil, err
	}
	sDir := skillsExtraDir
	kDir := knowledgeExtraDir
	if limits != nil && strings.EqualFold(limits.Tier, "free") {
		if sDir != "" && verbose {
			fmt.Fprintf(os.Stderr, "[%s] tier=free: ignoring --skills-dir\n", progName)
		}
		if kDir != "" && verbose {
			fmt.Fprintf(os.Stderr, "[%s] tier=free: ignoring --knowledge-dir\n", progName)
		}
		sDir, kDir = "", ""
	}
	genSkillsDir, genKnowledgeDir := loader.DefaultGeneratedDirs()
	skills, kb, err := loader.LoadSkillsAndKnowledge(loader.Options{
		SkillsExtraDir:        sDir,
		KnowledgeExtraDir:     kDir,
		GeneratedSkillsDir:    genSkillsDir,
		GeneratedKnowledgeDir: genKnowledgeDir,
	})
	if err != nil {
		return nil, err
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "[%s] llm credentials file: %s\n", progName, credSrc)
		if limits != nil && limits.Tier != "" {
			fmt.Fprintf(os.Stderr, "[%s] tier=%s max_llm_calls_per_day=%d\n", progName, limits.Tier, limits.MaxLLMCallsPerDay)
		}
		fmt.Fprintf(os.Stderr, "[%s] loaded %d skill(s), %d knowledge chunk(s)\n", progName, len(skills.Packs), len(kb.Chunks))
	}
	return &engine.Engine{Skills: skills, RAG: kb, LLM: client}, nil
}

// effectiveLoaderOptions applies tier=free (ignore custom skill/knowledge dirs). If credentials cannot be loaded, flags are used as-is.
func effectiveLoaderOptions() loader.Options {
	_, limits, _, err := config.LoadLLM(configFile, keyFile)
	if err != nil {
		return loader.Options{SkillsExtraDir: skillsExtraDir, KnowledgeExtraDir: knowledgeExtraDir}
	}
	if limits != nil && strings.EqualFold(limits.Tier, "free") {
		return loader.Options{}
	}
	return loader.Options{SkillsExtraDir: skillsExtraDir, KnowledgeExtraDir: knowledgeExtraDir}
}
