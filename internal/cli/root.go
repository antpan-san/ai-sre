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
	diagnosticPlanYes bool
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
	args := os.Args[1:]
	if programName == "ai-sre" {
		if res := ValidateParamContract(root, args); res != nil && !res.OK {
			reporter.finish(errParamContract)
			emitParamContractError(res)
			os.Exit(2)
		}
	}
	if err := root.Execute(); err != nil {
		if programName == "ai-sre" {
			if res := ClassifyCobraError(root, args, err); res != nil && !res.OK {
				reporter.finish(errParamContract)
				emitParamContractError(res)
				os.Exit(2)
			}
		}
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
	_ = tryAutoUpgradeInPlace("")
}

func newRoot(programName string) *cobra.Command {
	progName = programName
	var short, long string
	if programName == "opsfleet-executor" {
		short = "OpsFleet 本地执行器 — 与 ai-sre 相同的技能包与执行语义"
		long = fmt.Sprintf(`在需要部署或运维的受管机器上运行；与 ai-sre 共用同一套技能包（YAML）、Prompt、轻量 RAG 与 LLM 编排（需凭据）。
子命令：check（AI 诊断）/ probe（只读快采）/ ask / runbook / skills / doctor / k8s / job。
示例:
  %s check kafka --lag 100000
  %s check k8s --pod pending
  %s probe elasticsearch 127.0.0.1:9200
  %s ask "kafka lag 高怎么办"
  %s runbook "pod频繁重启"
  %s skills list
  %s k8s download --api-url http://host:9080/ft-api -u USER -p PASS --cluster c1 --version v1.35.4 --master 10.0.0.1`, programName, programName, programName, programName, programName, programName, programName)
	} else {
		short = "AI SRE Copilot — 故障诊断、Runbook、知识问答"
		long = fmt.Sprintf(`CLI 工具：技能包 + Prompt + 可选 RAG + LLM；支持 OpsFleet K8s 离线包与安装（见 k8s）。

命令分层（尽量少记）：
  check [topic]   AI 故障诊断（kafka/k8s/redis/…）
  probe <topic>   只读快采，不调 LLM
  check go        Go 运行时诊断（原 diagnose）
  doctor          CLI 环境自检（凭据/技能，不调 LLM）

示例:
  %s check kafka --lag 100000
  %s check k8s --pod pending
  %s probe redis 127.0.0.1:6379
  %s check go --pod default/api-0
  %s ask "kafka lag 高怎么办"
  %s doctor`, programName, programName, programName, programName, programName, programName)
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
	root.PersistentFlags().StringVarP(&outputFormat, "output", "o", "text", "output format: text|json (structured answer for check/ask/runbook)")
	root.PersistentFlags().StringVar(&skillsExtraDir, "skills-dir", "", "extra directory of *.yaml skill packs (merged with built-in; same name overrides)")
	root.PersistentFlags().StringVar(&knowledgeExtraDir, "knowledge-dir", "", "extra directory of *.md files for RAG (merged with built-in knowledge)")
	var noAutoUpgrade bool
	root.PersistentFlags().BoolVar(&noAutoUpgrade, "no-auto-upgrade", false, "跳过每次命令前的 OpsFleet 版本快检与自动升级（等同 OPSFLEET_NO_AUTO_UPGRADE=1）")

	cmds := []*cobra.Command{
		checkCmd(), probeCmd(), askCmd(), runbookCmd(), skillsCmd(), doctorCmd(),
		analyzeCmd(), diagnoseCmd(), // deprecated aliases
		versionCmd(), upgradeCmd(), k8sCmd(), serviceCmd(),
		kafkaCmd(), redisCmd(), mysqlCmd(), postgresqlCmd(), nginxCmd(), elasticsearchCmd(),
		nodeCmd(), jobCmd(),
	}
	if programName == "ai-sre" {
		cmds = append(cmds, uninstallCmd())
	}
	root.AddCommand(cmds...)
	if programName == "ai-sre" {
		root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
			if noAutoUpgrade {
				_ = os.Setenv("OPSFLEET_NO_AUTO_UPGRADE", "1")
			}
			return opsfleetPersistentPreRun(cmd, args)
		}
	}
	return root
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
	genSkillsDir, genKnowledgeDir := "", ""
	if localGeneratedSkillsEnabled() {
		genSkillsDir, genKnowledgeDir = loader.DefaultGeneratedDirs()
	}
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
