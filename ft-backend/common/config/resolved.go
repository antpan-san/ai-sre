package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// 环境变量名（与 systemd EnvironmentFile / deploy/backend.env.example 对齐）。
const (
	EnvAIAPIKey           = "OPSFLEET_AI_API_KEY"
	EnvAIBaseURL          = "OPSFLEET_AI_BASE_URL"
	EnvAIModel            = "OPSFLEET_AI_MODEL"
	EnvAISkillDataDir     = "OPSFLEET_AI_SKILL_DATA_DIR"
	EnvAISreBinaryPath    = "OPSFLEET_AISRE_BINARY_PATH"
	EnvAISreBinaryAmd64   = "OPSFLEET_AISRE_BINARY_PATH_AMD64"
	EnvAISreBinaryArm64   = "OPSFLEET_AISRE_BINARY_PATH_ARM64"
	EnvAISreVersion       = "OPSFLEET_AISRE_VERSION"
	EnvK8sMirrorBase      = "OPSFLEET_K8S_MIRROR_BASE_URL"
	EnvK8sMirrorManifest  = "OPSFLEET_K8S_MIRROR_MANIFEST_URL"
	EnvK8sRelayBase       = "OPSFLEET_K8S_RELAY_BASE_URL"
	EnvAnsibleDir         = "OPSFLEET_ANSIBLE_DIR"
	EnvSkillAutoRefine    = "OPSFLEET_SKILL_AUTO_REFINE"
	EnvSkillAutoMin       = "OPSFLEET_SKILL_AUTO_REFINE_MIN_SAMPLES"
	EnvSkillAutoCooldown  = "OPSFLEET_SKILL_AUTO_REFINE_COOLDOWN"
	EnvSkillAutoTopics    = "OPSFLEET_SKILL_AUTO_REFINE_TOPICS"
	EnvSkillAutoMaxPerDay = "OPSFLEET_SKILL_AUTO_REFINE_MAX_PER_DAY"
	EnvJWTAccessTokenExp  = "OPSFLEET_JWT_ACCESS_TOKEN_EXP"
	EnvJWTRefreshTokenExp = "OPSFLEET_JWT_REFRESH_TOKEN_EXP"
)

// ResolvedAI LLM 配置：环境变量优先于 conf/config.yaml 的 ai 段。
type ResolvedAI struct {
	APIKey  string
	BaseURL string
	Model   string
}

func yamlAI() AIConfig {
	if GlobalCfg != nil {
		return GlobalCfg.AI
	}
	return AIConfig{}
}

func yamlOpsfleet() OpsfleetConfig {
	if GlobalCfg != nil {
		return GlobalCfg.Opsfleet
	}
	return OpsfleetConfig{}
}

func yamlK8s() K8sConfig {
	if GlobalCfg != nil {
		return GlobalCfg.K8s
	}
	return K8sConfig{}
}

func yamlSkills() SkillsConfig {
	if GlobalCfg != nil {
		return GlobalCfg.Skills
	}
	return SkillsConfig{}
}

// ResolvedAIConfig returns DeepSeek-compatible settings.
func ResolvedAIConfig() ResolvedAI {
	y := yamlAI()
	out := ResolvedAI{
		APIKey:  EnvOrString(EnvAIAPIKey, y.APIKey),
		BaseURL: EnvOrString(EnvAIBaseURL, y.BaseURL),
		Model:   EnvOrString(EnvAIModel, y.Model),
	}
	if out.BaseURL == "" {
		out.BaseURL = "https://api.deepseek.com/v1"
	}
	if out.Model == "" {
		out.Model = "deepseek-chat"
	}
	return out
}

// ResolvedAISkillDataDir returns configured skill data dir (empty → registry uses built-in fallbacks).
func ResolvedAISkillDataDir() string {
	y := yamlOpsfleet()
	return EnvOrString(EnvAISkillDataDir, y.AISkillDataDir)
}

// ResolvedAISreBinaryPath default/legacy ai-sre ELF for download.
func ResolvedAISreBinaryPath() string {
	y := yamlOpsfleet()
	return EnvOrString(EnvAISreBinaryPath, y.AiSreBinaryPath)
}

// ResolvedAISreBinaryPathAmd64 explicit amd64 distribution path.
func ResolvedAISreBinaryPathAmd64() string {
	y := yamlOpsfleet()
	return EnvOrString(EnvAISreBinaryAmd64, y.AiSreBinaryPathAmd64)
}

// ResolvedAISreBinaryPathArm64 explicit arm64 distribution path.
func ResolvedAISreBinaryPathArm64() string {
	y := yamlOpsfleet()
	return EnvOrString(EnvAISreBinaryArm64, y.AiSreBinaryPathArm64)
}

// ResolvedAISreVersion optional version string for API without exec.
func ResolvedAISreVersion() string {
	return EnvOrString(EnvAISreVersion, "")
}

// ResolvedK8sMirrorBaseURL for catalog proxy.
func ResolvedK8sMirrorBaseURL() string {
	y := yamlK8s()
	v := EnvOrString(EnvK8sMirrorBase, y.MirrorBaseURL)
	if v != "" {
		return v
	}
	return "http://192.168.56.11"
}

// ResolvedK8sMirrorManifestURL full manifest URL.
func ResolvedK8sMirrorManifestURL() string {
	if u := EnvOrString(EnvK8sMirrorManifest, yamlK8s().MirrorManifestURL); u != "" {
		return u
	}
	return strings.TrimRight(ResolvedK8sMirrorBaseURL(), "/") + "/manifest.json"
}

// ResolvedK8sRelayBaseURL relay download base (falls back to mirror base).
func ResolvedK8sRelayBaseURL() string {
	y := yamlK8s()
	if v := EnvOrString(EnvK8sRelayBase, y.RelayBaseURL); v != "" {
		return v
	}
	return ResolvedK8sMirrorBaseURL()
}

// ResolvedAnsibleDir optional override for bundle generation.
func ResolvedAnsibleDir() string {
	return EnvOrString(EnvAnsibleDir, yamlK8s().AnsibleDir)
}

// ResolvedSkillAutoRefine merges yaml skills.auto_refine with OPSFLEET_SKILL_AUTO_REFINE_* env.
type ResolvedSkillAutoRefine struct {
	Enabled    bool
	MinSamples int
	Cooldown   time.Duration
	Topics     map[string]struct{}
	MaxPerDay  int
}

func (c ResolvedSkillAutoRefine) AllowsTopic(topic string) bool {
	topic = strings.ToLower(strings.TrimSpace(topic))
	if topic == "" {
		return false
	}
	_, ok := c.Topics[topic]
	return ok
}

// ResolvedSkillAutoRefineConfig returns effective auto-refine policy.
func ResolvedSkillAutoRefineConfig() ResolvedSkillAutoRefine {
	y := yamlSkills().AutoRefine
	enabled := EnvOrBool(EnvSkillAutoRefine, y.Enabled)
	minSamples := EnvOrInt(EnvSkillAutoMin, y.MinSamples, 8)
	cooldown := EnvOrDuration(EnvSkillAutoCooldown, y.Cooldown, 12*time.Hour)
	maxPerDay := EnvOrInt(EnvSkillAutoMaxPerDay, y.MaxPerDay, 3)
	topics := EnvOrStringList(EnvSkillAutoTopics, y.Topics)
	if len(topics) == 0 {
		topics = []string{"go_runtime", "k8s"}
	}
	set := make(map[string]struct{}, len(topics))
	for _, t := range topics {
		t = strings.ToLower(strings.TrimSpace(t))
		if t != "" {
			set[t] = struct{}{}
		}
	}
	if minSamples < 1 {
		minSamples = 1
	}
	if maxPerDay < 1 {
		maxPerDay = 1
	}
	return ResolvedSkillAutoRefine{
		Enabled:    enabled,
		MinSamples: minSamples,
		Cooldown:   cooldown,
		Topics:     set,
		MaxPerDay:  maxPerDay,
	}
}

// ResolvedAutoIteration merges yaml auto_iteration with OPSFLEET_AUTO_ITERATION_* env.
type ResolvedAutoIteration struct {
	Enabled                  bool
	MaxConcurrent            int
	HighRiskRequiresApproval bool
	DingTalkWebhook          string
	GitHubRepo               string
	CodeAgentToken           string
}

func yamlAutoIteration() AutoIterationConfig {
	if GlobalCfg != nil {
		return GlobalCfg.AutoIteration
	}
	return AutoIterationConfig{}
}

// ResolvedAutoIterationConfig returns effective auto-iteration policy (secrets from env only in production).
func ResolvedAutoIterationConfig() ResolvedAutoIteration {
	y := yamlAutoIteration()
	enabled := y.Enabled
	if v := strings.TrimSpace(os.Getenv("OPSFLEET_AUTO_ITERATION_ENABLED")); v != "" {
		enabled = v == "1" || strings.EqualFold(v, "true")
	}
	maxConcurrent := y.MaxConcurrent
	if maxConcurrent < 1 {
		maxConcurrent = 2
	}
	if v := strings.TrimSpace(os.Getenv("OPSFLEET_AUTO_ITERATION_MAX_CONCURRENT")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxConcurrent = n
		}
	}
	highRisk := y.HighRiskRequiresApproval
	if v := strings.TrimSpace(os.Getenv("OPSFLEET_AUTO_ITERATION_HIGH_RISK_REQUIRES_APPROVAL")); v != "" {
		highRisk = v == "1" || strings.EqualFold(v, "true")
	}
	webhook := strings.TrimSpace(os.Getenv("OPSFLEET_AUTO_ITERATION_DINGTALK_WEBHOOK"))
	if webhook == "" {
		webhook = strings.TrimSpace(y.DingTalkWebhook)
	}
	repo := strings.TrimSpace(os.Getenv("OPSFLEET_AUTO_ITERATION_GITHUB_REPO"))
	if repo == "" {
		repo = strings.TrimSpace(y.GitHubRepo)
	}
	agentToken := strings.TrimSpace(os.Getenv("OPSFLEET_CODE_AGENT_TOKEN"))
	if agentToken == "" {
		agentToken = strings.TrimSpace(y.CodeAgentToken)
	}
	return ResolvedAutoIteration{
		Enabled:                  enabled,
		MaxConcurrent:            maxConcurrent,
		HighRiskRequiresApproval: highRisk,
		DingTalkWebhook:          webhook,
		GitHubRepo:               repo,
		CodeAgentToken:           agentToken,
	}
}

// ApplyEnvOverrides patches in-memory config after yaml load (production secrets / JWT TTL).
func ApplyEnvOverrides(cfg *Config) {
	if cfg == nil {
		return
	}
	if v := strings.TrimSpace(os.Getenv(EnvJWTAccessTokenExp)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.JWT.AccessTokenExp = n
		}
	}
	if v := strings.TrimSpace(os.Getenv(EnvJWTRefreshTokenExp)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.JWT.RefreshTokenExp = n
		}
	}
}
