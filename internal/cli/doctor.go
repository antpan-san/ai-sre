package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/panshuai/ai-sre/internal/config"
	"github.com/panshuai/ai-sre/internal/loader"
	"github.com/panshuai/ai-sre/internal/quota"
)

func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "自检：运行时、配置目录、凭据、配额、技能与知识库加载（不调用 LLM）",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("== %s doctor ==\n", progName)
			fmt.Printf("go_runtime: %s\n", runtime.Version())
			fmt.Printf("cli_version: %s\n", Version)

			cfgDir, err := config.ResolveDir()
			if err != nil {
				return err
			}
			fmt.Printf("config_dir: %s\n", cfgDir)

			_, limits, src, err := config.LoadLLM(configFile, keyFile)
			if err != nil {
				fmt.Printf("credentials: ERROR %v\n", err)
			} else {
				fmt.Printf("credentials: OK (source %s)\n", src)
				if limits != nil {
					fmt.Printf("tier: %q\n", limits.Tier)
					fmt.Printf("max_llm_calls_per_day: %d\n", limits.MaxLLMCallsPerDay)
				}
			}

			cacheDir, err := quota.DefaultCacheDir()
			if err != nil {
				return err
			}
			d, c, err := quota.ReadUsage(cacheDir)
			if err != nil {
				fmt.Printf("quota_read: ERROR %v\n", err)
			} else {
				fmt.Printf("quota_cache: %s\n", cacheDir)
				fmt.Printf("llm_calls_today: %d (date %s)\n", c, d)
			}

			reg, kb, err := loader.LoadSkillsAndKnowledge(effectiveLoaderOptions())
			if err != nil {
				fmt.Printf("skills/knowledge: ERROR %v\n", err)
			} else {
				fmt.Printf("skills_loaded: %d\n", len(reg.Packs))
				fmt.Printf("knowledge_chunks: %d\n", len(kb.Chunks))
			}
			metricsPath := filepath.Join(cfgDir, "evolution_metrics.json")
			if b, err := os.ReadFile(metricsPath); err == nil {
				var m struct {
					UpdatedAt string         `json:"updated_at"`
					Counters  map[string]int `json:"counters"`
				}
				if json.Unmarshal(b, &m) == nil && len(m.Counters) > 0 {
					fmt.Printf("evolution_metrics_updated_at: %s\n", m.UpdatedAt)
					fmt.Printf("evolution_local_hit: %d\n", m.Counters["local_hit"])
					fmt.Printf("evolution_server_fallback: %d\n", m.Counters["server_fallback"])
					fmt.Printf("evolution_generated_skill: %d\n", m.Counters["generated_skill"])
					fmt.Printf("evolution_autopipeline_success: %d\n", m.Counters["autopipeline_success"])
				}
			}

			fmt.Printf("hint: LLM 联通性请用: %s ask \"ping\" 或 SHORT=1 bash scripts/remote-e2e.sh\n", progName)
			return nil
		},
	}
}
