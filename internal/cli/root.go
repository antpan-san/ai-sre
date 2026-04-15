package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/panshuai/ai-sre/internal/assets"
	"github.com/panshuai/ai-sre/internal/engine"
	"github.com/panshuai/ai-sre/internal/llm"
)

var (
	verbose   bool
	noRAG     bool
	lag       string
	topicFlag string
	pod       string
	namespace string
	issue     string
	code      string
	upstream  string
	latency   string
	setKV     map[string]string
)

// Execute runs the Cobra root (entry from main).
func Execute() {
	if err := newRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRoot() *cobra.Command {
	root := &cobra.Command{
		Use:     "ai-sre",
		Aliases: []string{"ops-ai"},
		Short:   "AI SRE Copilot — 故障诊断、Runbook、知识问答",
		Long: `CLI 工具：技能包（Skill Pack）+ Prompt + 可选轻量 RAG + DeepSeek LLM。
示例:
  ai-sre analyze kafka --lag 100000
  ai-sre analyze k8s --pod pending
  ai-sre ask "kafka lag 高怎么办"
  ai-sre runbook "pod频繁重启"`,
		SilenceUsage: true,
	}
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose logs")
	root.PersistentFlags().BoolVar(&noRAG, "no-rag", false, "disable knowledge retrieval (RAG)")

	root.AddCommand(analyzeCmd(), askCmd(), runbookCmd(), versionCmd())
	return root
}

func analyzeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze [topic]",
		Short: "故障诊断（核心能力）",
		Long: `topic 取值: kafka | k8s | nginx | redis

k8s 场景可用 --pod 区分: pending（调度/Pending）或 crashloop（CrashLoopBackOff）。
也可用 --issue pending|crashloop。`,
		Example: `  ai-sre analyze kafka --lag 100000 --topic orders
  ai-sre analyze k8s --pod pending
  ai-sre analyze k8s --pod crashloop --namespace prod
  ai-sre analyze nginx --code 502
  ai-sre analyze redis --latency 50ms`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := buildContextMap()
			eng, err := bootstrap()
			if err != nil {
				return err
			}
			topic := args[0]
			out, err := eng.Analyze(context.Background(), topic, ctx, !noRAG)
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&lag, "lag", "", "Kafka consumer lag 等指标")
	cmd.Flags().StringVar(&topicFlag, "topic", "", "Kafka topic 名称")
	cmd.Flags().StringVar(&pod, "pod", "", "K8s: pod 名或问题类型（如 pending / crashloop）")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Kubernetes namespace")
	cmd.Flags().StringVar(&issue, "issue", "", "K8s: pending | crashloop")
	cmd.Flags().StringVar(&code, "code", "", "HTTP 状态码，如 502")
	cmd.Flags().StringVar(&upstream, "upstream", "", "Nginx upstream 名称或服务名")
	cmd.Flags().StringVar(&latency, "latency", "", "延迟描述，如 50ms、p99=20ms")
	cmd.Flags().StringToStringVarP(&setKV, "set", "d", nil, "附加上下文 key=value，可多次使用")
	return cmd
}

func askCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ask [question]",
		Short: "知识库问答（结合轻量 RAG）",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, err := bootstrap()
			if err != nil {
				return err
			}
			q := strings.Join(args, " ")
			out, err := eng.Ask(context.Background(), q, !noRAG)
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil
		},
	}
}

func runbookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runbook [scenario]",
		Short: "生成 Runbook 文档",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, err := bootstrap()
			if err != nil {
				return err
			}
			scenario := strings.Join(args, " ")
			ctx := map[string]string{}
			if len(setKV) > 0 {
				for k, v := range setKV {
					ctx[k] = v
				}
			}
			out, err := eng.Runbook(context.Background(), scenario, ctx, !noRAG)
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil
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
			fmt.Println("ai-sre 0.1.0")
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
	client, err := llm.NewFromEnv()
	if err != nil {
		return nil, err
	}
	skills, err := engine.LoadSkillsFromFS(assets.FS)
	if err != nil {
		return nil, fmt.Errorf("load skills: %w", err)
	}
	kb, err := engine.LoadKnowledgeFromFS(assets.FS)
	if err != nil {
		return nil, fmt.Errorf("load knowledge: %w", err)
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "[ai-sre] loaded %d skill(s), %d knowledge chunk(s)\n", len(skills.Packs), len(kb.Chunks))
	}
	return &engine.Engine{Skills: skills, RAG: kb, LLM: client}, nil
}
