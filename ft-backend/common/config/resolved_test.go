package config

import (
	"os"
	"testing"
	"time"
)

func TestResolvedSkillAutoRefineYamlThenEnv(t *testing.T) {
	GlobalCfg = &Config{
		Skills: SkillsConfig{
			AutoRefine: SkillAutoRefineConfig{
				Enabled:    false,
				MinSamples: 5,
				Cooldown:   "6h",
				Topics:     []string{"k8s"},
				MaxPerDay:  2,
			},
		},
	}
	t.Cleanup(func() { GlobalCfg = nil })

	got := ResolvedSkillAutoRefineConfig()
	if got.Enabled {
		t.Fatal("yaml enabled=false")
	}
	if got.MinSamples != 5 {
		t.Fatalf("min_samples=%d", got.MinSamples)
	}
	if got.Cooldown != 6*time.Hour {
		t.Fatalf("cooldown=%v", got.Cooldown)
	}
	if !got.AllowsTopic("k8s") || got.AllowsTopic("go_runtime") {
		t.Fatal("topics from yaml")
	}

	t.Setenv(EnvSkillAutoRefine, "1")
	t.Setenv(EnvSkillAutoMin, "9")
	t.Setenv(EnvSkillAutoTopics, "go_runtime,mysql")
	got = ResolvedSkillAutoRefineConfig()
	if !got.Enabled || got.MinSamples != 9 {
		t.Fatalf("env override: enabled=%v min=%d", got.Enabled, got.MinSamples)
	}
	if !got.AllowsTopic("go_runtime") || !got.AllowsTopic("mysql") || got.AllowsTopic("k8s") {
		t.Fatal("topics from env")
	}
}

func TestResolvedAIConfigDefaults(t *testing.T) {
	GlobalCfg = &Config{AI: AIConfig{}}
	t.Cleanup(func() { GlobalCfg = nil })
	os.Unsetenv(EnvAIAPIKey)
	os.Unsetenv(EnvAIBaseURL)
	os.Unsetenv(EnvAIModel)
	os.Unsetenv(EnvDeepSeekAPIKey)
	os.Unsetenv(EnvDeepSeekBaseURL)
	os.Unsetenv(EnvDeepSeekModel)
	r := ResolvedAIConfig()
	if r.BaseURL != "https://api.deepseek.com/v1" || r.Model != "deepseek-chat" {
		t.Fatalf("defaults: %+v", r)
	}
}

func TestResolvedAIConfigDeepSeekEnvAlias(t *testing.T) {
	GlobalCfg = &Config{AI: AIConfig{}}
	t.Cleanup(func() {
		GlobalCfg = nil
		os.Unsetenv(EnvAIAPIKey)
		os.Unsetenv(EnvAIBaseURL)
		os.Unsetenv(EnvAIModel)
		os.Unsetenv(EnvDeepSeekAPIKey)
		os.Unsetenv(EnvDeepSeekBaseURL)
		os.Unsetenv(EnvDeepSeekModel)
	})
	t.Setenv(EnvDeepSeekAPIKey, "sk-test")
	t.Setenv(EnvDeepSeekBaseURL, "https://api.deepseek.com")
	t.Setenv(EnvDeepSeekModel, "deepseek-chat")
	r := ResolvedAIConfig()
	if r.APIKey != "sk-test" || r.BaseURL != "https://api.deepseek.com/v1" || r.Model != "deepseek-chat" {
		t.Fatalf("deepseek env: %+v", r)
	}
	t.Setenv(EnvAIAPIKey, "sk-opsfleet")
	r = ResolvedAIConfig()
	if r.APIKey != "sk-opsfleet" {
		t.Fatalf("opsfleet env should win: %+v", r)
	}
}
