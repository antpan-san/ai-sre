package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/panshuai/ai-sre/internal/loader"
	"github.com/panshuai/ai-sre/internal/output"
)

func skillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "技能包注册表：列出本地/服务端技能，触发服务端精炼，回写反馈",
	}
	cmd.AddCommand(skillsListCmd())
	cmd.AddCommand(skillsServerCmd())
	cmd.AddCommand(skillsRefineCmd())
	cmd.AddCommand(skillsFeedbackCmd())
	return cmd
}

func skillsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "列出本地已加载的技能包（内置 + --skills-dir）",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, _, err := loader.LoadSkillsAndKnowledge(effectiveLoaderOptions())
			if err != nil {
				return err
			}
			if strings.EqualFold(outputFormat, "json") {
				return output.PrintSkillsJSON(os.Stdout, reg)
			}
			fmt.Println("Skill packs:")
			output.PrintSkillsTable(os.Stdout, reg)
			return nil
		},
	}
}

func skillsServerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "server",
		Short: "列出 OpsFleet 服务端已注册的技能包（builtin + generated）",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			list, dataDir, err := callServerSkillsList(ctx)
			if err != nil {
				return err
			}
			if strings.EqualFold(outputFormat, "json") {
				return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
					"skills":   list,
					"data_dir": dataDir,
				})
			}
			fmt.Printf("Server skill registry (data_dir=%s)\n", dataDir)
			if len(list) == 0 {
				fmt.Println("  (no skills returned)")
				return nil
			}
			fmt.Printf("%-32s %-10s %-22s %s\n", "NAME", "SOURCE", "VERSION", "TOPICS")
			for _, s := range list {
				topics := strings.Join(s.Topics, ",")
				fmt.Printf("%-32s %-10s %-22s %s\n", s.Name, s.Source, s.Version, topics)
			}
			return nil
		},
	}
}

func skillsRefineCmd() *cobra.Command {
	var topic, userHint string
	var dryRun bool
	var maxSamples, maxFeedback, timeoutSec int
	cmd := &cobra.Command{
		Use:   "refine",
		Short: "请求服务端基于最近样本/反馈精炼某 topic 的技能包",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(topic) == "" {
				return errors.New("--topic 不能为空")
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Duration(timeoutSec+15)*time.Second)
			defer cancel()
			raw, err := callServerSkillsRefine(ctx, topic, userHint, dryRun, maxSamples, maxFeedback, timeoutSec)
			if err != nil {
				return err
			}
			if strings.EqualFold(outputFormat, "json") {
				_, _ = os.Stdout.Write(raw)
				_, _ = os.Stdout.Write([]byte("\n"))
				return nil
			}
			fmt.Println(string(raw))
			return nil
		},
	}
	cmd.Flags().StringVar(&topic, "topic", "", "目标 topic（如 k8s/kafka/redis/mysql/nginx/elasticsearch）")
	cmd.Flags().StringVar(&userHint, "hint", "", "可选：希望新技能包重点改进的方向")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "（保留）仅返回拟用 prompt，不持久化")
	cmd.Flags().IntVar(&maxSamples, "max-samples", 12, "最多取最近 N 条诊断样本")
	cmd.Flags().IntVar(&maxFeedback, "max-feedback", 8, "最多取最近 N 条客户端反馈")
	cmd.Flags().IntVar(&timeoutSec, "timeout", 120, "服务端 LLM 调用超时（秒）")
	return cmd
}

func skillsFeedbackCmd() *cobra.Command {
	var topic, skill, reqID, note string
	var helpful string
	cmd := &cobra.Command{
		Use:   "feedback",
		Short: "向 OpsFleet 提交一次诊断反馈，驱动服务端技能精炼",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(topic) == "" {
				return errors.New("--topic 不能为空")
			}
			var helpfulPtr *bool
			switch strings.ToLower(strings.TrimSpace(helpful)) {
			case "true", "yes", "y", "1":
				v := true
				helpfulPtr = &v
			case "false", "no", "n", "0":
				v := false
				helpfulPtr = &v
			case "":
				// keep nil = unknown
			default:
				return fmt.Errorf("--helpful 只接受 yes/no/(留空)")
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			if err := callServerSkillsFeedback(ctx, topic, skill, reqID, helpfulPtr, note); err != nil {
				return err
			}
			fmt.Println("反馈已上报。可执行 ai-sre skills refine --topic " + topic + " 触发服务端精炼。")
			return nil
		},
	}
	cmd.Flags().StringVar(&topic, "topic", "", "目标 topic")
	cmd.Flags().StringVar(&skill, "skill", "", "对应技能包 name（可选）")
	cmd.Flags().StringVar(&reqID, "request-id", "", "之前那次诊断的 request_id（可选）")
	cmd.Flags().StringVar(&helpful, "helpful", "", "yes/no（是否对定位根因有帮助）")
	cmd.Flags().StringVarP(&note, "message", "m", "", "本次反馈说明（建议描述\"哪里不够准确\"或\"漏看了什么证据\"）")
	return cmd
}

type serverSkillSummary struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Topics      []string `json:"topics"`
	Source      string   `json:"source"`
	Version     string   `json:"version"`
	Path        string   `json:"path,omitempty"`
}

func callServerSkillsList(ctx context.Context) ([]serverSkillSummary, string, error) {
	base := strings.TrimSpace(resolveOpsfleetAPIBase())
	if base == "" {
		return nil, "", errors.New("opsfleet api base is empty")
	}
	endpoint := strings.TrimRight(base, "/") + "/api/ai/skills"
	hreq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(hreq)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("server skills list status=%d: %s", resp.StatusCode, parseOpsfleetErrMsg(raw))
	}
	var env struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, "", err
	}
	if env.Code != 200 {
		return nil, "", fmt.Errorf("api code=%d msg=%s", env.Code, env.Msg)
	}
	var data struct {
		Skills  []serverSkillSummary `json:"skills"`
		DataDir string               `json:"data_dir"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil, "", err
	}
	return data.Skills, data.DataDir, nil
}

func callServerSkillsRefine(ctx context.Context, topic, hint string, dryRun bool, maxSamples, maxFeedback, timeoutSec int) ([]byte, error) {
	base := strings.TrimSpace(resolveOpsfleetAPIBase())
	if base == "" {
		return nil, errors.New("opsfleet api base is empty")
	}
	endpoint := strings.TrimRight(base, "/") + "/api/ai/skills/refine"
	body, err := json.Marshal(map[string]interface{}{
		"topic":        topic,
		"user_hint":    hint,
		"dry_run":      dryRun,
		"max_samples":  maxSamples,
		"max_feedback": maxFeedback,
		"timeout_sec":  timeoutSec,
	})
	if err != nil {
		return nil, err
	}
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	hreq.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: time.Duration(timeoutSec+15) * time.Second}
	resp, err := client.Do(hreq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("server skills refine status=%d: %s", resp.StatusCode, parseOpsfleetErrMsg(raw))
	}
	return raw, nil
}

func callServerSkillsFeedback(ctx context.Context, topic, skill, reqID string, helpful *bool, note string) error {
	base := strings.TrimSpace(resolveOpsfleetAPIBase())
	if base == "" {
		return errors.New("opsfleet api base is empty")
	}
	endpoint := strings.TrimRight(base, "/") + "/api/ai/skills/feedback"
	body, err := json.Marshal(map[string]interface{}{
		"topic":      topic,
		"skill_name": skill,
		"request_id": reqID,
		"helpful":    helpful,
		"note":       note,
	})
	if err != nil {
		return err
	}
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	hreq.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(hreq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("server skills feedback status=%d: %s", resp.StatusCode, parseOpsfleetErrMsg(raw))
	}
	return nil
}
