package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/panshuai/ai-sre/internal/config"
	"github.com/panshuai/ai-sre/internal/loader"
	"github.com/panshuai/ai-sre/internal/skill"
	"gopkg.in/yaml.v3"
)

type diagnoseRequest struct {
	Topic   string            `json:"topic"`
	Context map[string]string `json:"context,omitempty"`
	Command string            `json:"command,omitempty"`
}

type diagnoseResponse struct {
	Source       string                 `json:"source"`
	Answer       string                 `json:"answer"`
	SkillName    string                 `json:"skill_name,omitempty"`
	SkillDisplay string                 `json:"skill_display,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	SkillDraft   *skill.Pack            `json:"skill_draft,omitempty"`
}

type evolutionConfig struct {
	Mode            string `yaml:"mode"`
	TargetBranch    string `yaml:"target_branch"`
	MaxAutoCommits  int    `yaml:"max_auto_commits"`
	PrePushTestCmd  string `yaml:"pre_push_test_cmd"`
	AutoCommitMsg   string `yaml:"auto_commit_msg"`
	FailFastStreak  int    `yaml:"fail_fast_streak"`
	EnableGenerated bool   `yaml:"enable_generated"`
}

type pipelineState struct {
	ConsecutiveFailures int    `json:"consecutive_failures"`
	AutoCommitsToday    int    `json:"auto_commits_today"`
	Date                string `json:"date"`
}

func defaultEvolutionConfig() evolutionConfig {
	return evolutionConfig{
		Mode:            "off",
		TargetBranch:    "main",
		MaxAutoCommits:  1,
		PrePushTestCmd:  "go test ./...",
		AutoCommitMsg:   "chore(skills): auto-evolve generated skill",
		FailFastStreak:  3,
		EnableGenerated: true,
	}
}

func runAnalyzeWithOrchestrator(ctx context.Context, topic string, kv map[string]string) (*diagnoseResponse, error) {
	opts := effectiveLoaderOptions()
	ev := loadEvolutionConfig()
	opts.GeneratedSkillsDir, opts.GeneratedKnowledgeDir = loader.DefaultGeneratedDirs()
	if !ev.EnableGenerated {
		opts.GeneratedSkillsDir = ""
		opts.GeneratedKnowledgeDir = ""
	}
	skills, _, err := loader.LoadSkillsAndKnowledge(opts)
	if err != nil {
		return nil, err
	}
	pack := skills.MatchAnalyze(topic, kv)
	if pack != nil && isSkillCoverageSufficient(pack, kv) {
		eng, err := bootstrap()
		if err != nil {
			if !isCredentialError(err) {
				return nil, err
			}
		} else {
			res, e := eng.Analyze(ctx, topic, kv, !noRAG)
			if e == nil {
				recordDiagnoseMetric("local_hit")
				return &diagnoseResponse{
					Source:       "local",
					Answer:       res.Answer,
					SkillName:    res.SkillName,
					SkillDisplay: res.SkillDisplay,
				}, nil
			}
		}
	}
	recordDiagnoseMetric("server_fallback")
	resp, err := callServerDiagnose(ctx, diagnoseRequest{
		Topic:   topic,
		Context: kv,
		Command: strings.Join(os.Args, " "),
	})
	if err != nil {
		return nil, err
	}
	if resp.SkillDraft != nil && resp.SkillDraft.Name != "" {
		if p, e := writeGeneratedSkill(resp.SkillDraft); e == nil {
			resp.Metadata = ensureMap(resp.Metadata)
			resp.Metadata["generated_skill_path"] = p
			recordDiagnoseMetric("generated_skill")
			if err := maybeAutoPipeline(ev, p); err != nil {
				resp.Metadata["autopipeline_error"] = err.Error()
				recordDiagnoseMetric("autopipeline_error")
			} else if ev.Mode == "full_pipeline" {
				recordDiagnoseMetric("autopipeline_success")
			}
		}
	}
	return resp, nil
}

func ensureMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return map[string]interface{}{}
	}
	return m
}

func isSkillCoverageSufficient(p *skill.Pack, kv map[string]string) bool {
	if p == nil {
		return false
	}
	if len(p.AnalysisSteps) < 2 || len(p.OutputFormat) == 0 {
		return false
	}
	return len(kv) > 0
}

func isCredentialError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "credentials not found") || strings.Contains(msg, "api key")
}

func callServerDiagnose(ctx context.Context, req diagnoseRequest) (*diagnoseResponse, error) {
	base := resolveOpsfleetAPIBase()
	if strings.TrimSpace(base) == "" {
		return nil, errors.New("opsfleet api base is empty")
	}
	endpoint := strings.TrimRight(base, "/") + "/api/ai/diagnose"
	body, _ := json.Marshal(req)
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	hreq.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(hreq)
	if err != nil {
		return nil, fmt.Errorf("call server diagnose: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("server diagnose status=%d", resp.StatusCode)
	}
	var out diagnoseResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.Answer) == "" {
		return nil, errors.New("empty diagnose answer from server")
	}
	return &out, nil
}

func loadEvolutionConfig() evolutionConfig {
	cfg := defaultEvolutionConfig()
	dir, err := config.ResolveDir()
	if err != nil {
		return cfg
	}
	p := filepath.Join(dir, "evolution.yaml")
	b, err := os.ReadFile(p)
	if err != nil {
		return cfg
	}
	_ = yaml.Unmarshal(b, &cfg)
	cfg.Mode = strings.TrimSpace(strings.ToLower(cfg.Mode))
	if cfg.Mode == "" {
		cfg.Mode = "off"
	}
	if cfg.TargetBranch == "" {
		cfg.TargetBranch = "main"
	}
	if cfg.PrePushTestCmd == "" {
		cfg.PrePushTestCmd = "go test ./..."
	}
	if cfg.AutoCommitMsg == "" {
		cfg.AutoCommitMsg = "chore(skills): auto-evolve generated skill"
	}
	if cfg.MaxAutoCommits <= 0 {
		cfg.MaxAutoCommits = 1
	}
	if cfg.FailFastStreak <= 0 {
		cfg.FailFastStreak = 3
	}
	return cfg
}

func generatedSkillDir() (string, error) {
	dir, err := config.ResolveDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "generated-skills"), nil
}

func writeGeneratedSkill(p *skill.Pack) (string, error) {
	root, err := generatedSkillDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		return "", err
	}
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return "", errors.New("empty generated skill name")
	}
	b, err := yaml.Marshal(p)
	if err != nil {
		return "", err
	}
	fp := filepath.Join(root, name+".yaml")
	if err := os.WriteFile(fp, b, 0644); err != nil {
		return "", err
	}
	return fp, nil
}

func maybeAutoPipeline(cfg evolutionConfig, skillPath string) error {
	if cfg.Mode != "full_pipeline" || strings.TrimSpace(skillPath) == "" {
		return nil
	}
	st := loadPipelineState()
	today := time.Now().Format("2006-01-02")
	if st.Date != today {
		st.Date = today
		st.AutoCommitsToday = 0
		st.ConsecutiveFailures = 0
	}
	if st.ConsecutiveFailures >= cfg.FailFastStreak {
		return fmt.Errorf("autopipeline 熔断：连续失败 %d 次", st.ConsecutiveFailures)
	}
	if st.AutoCommitsToday >= cfg.MaxAutoCommits {
		return fmt.Errorf("autopipeline 达到当日提交上限: %d", cfg.MaxAutoCommits)
	}
	if err := runShellCmd(cfg.PrePushTestCmd); err != nil {
		st.ConsecutiveFailures++
		savePipelineState(st)
		return fmt.Errorf("pre-push test failed: %w", err)
	}
	if err := runShellCmd(fmt.Sprintf("git add %q", skillPath)); err != nil {
		st.ConsecutiveFailures++
		savePipelineState(st)
		return err
	}
	if err := runShellCmd(fmt.Sprintf("git commit -m %q", cfg.AutoCommitMsg)); err != nil {
		st.ConsecutiveFailures++
		savePipelineState(st)
		return err
	}
	if err := runShellCmd(fmt.Sprintf("git push origin %s", cfg.TargetBranch)); err != nil {
		st.ConsecutiveFailures++
		savePipelineState(st)
		return err
	}
	st.ConsecutiveFailures = 0
	st.AutoCommitsToday++
	savePipelineState(st)
	return nil
}

func runShellCmd(s string) error {
	cmd := exec.Command("sh", "-c", s)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func recordDiagnoseMetric(event string) {
	dir, err := config.ResolveDir()
	if err != nil {
		return
	}
	_ = os.MkdirAll(dir, 0755)
	p := filepath.Join(dir, "evolution_metrics.json")
	type metrics struct {
		UpdatedAt string         `json:"updated_at"`
		Counters  map[string]int `json:"counters"`
	}
	m := metrics{Counters: map[string]int{}}
	if b, err := os.ReadFile(p); err == nil {
		_ = json.Unmarshal(b, &m)
		if m.Counters == nil {
			m.Counters = map[string]int{}
		}
	}
	m.Counters[event]++
	m.UpdatedAt = time.Now().Format(time.RFC3339)
	if b, err := json.MarshalIndent(m, "", "  "); err == nil {
		_ = os.WriteFile(p, b, 0644)
	}
}

func loadPipelineState() pipelineState {
	dir, err := config.ResolveDir()
	if err != nil {
		return pipelineState{}
	}
	p := filepath.Join(dir, "evolution_pipeline_state.json")
	var st pipelineState
	b, err := os.ReadFile(p)
	if err != nil {
		return st
	}
	_ = json.Unmarshal(b, &st)
	return st
}

func savePipelineState(st pipelineState) {
	dir, err := config.ResolveDir()
	if err != nil {
		return
	}
	_ = os.MkdirAll(dir, 0755)
	p := filepath.Join(dir, "evolution_pipeline_state.json")
	if b, err := json.MarshalIndent(st, "", "  "); err == nil {
		_ = os.WriteFile(p, b, 0644)
	}
}
