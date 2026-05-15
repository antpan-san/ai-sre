package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"ft-backend/common/config"
)

func TestAutoRefineConfigDefaults(t *testing.T) {
	config.GlobalCfg = &config.Config{}
	t.Cleanup(func() { config.GlobalCfg = nil })
	os.Unsetenv(config.EnvSkillAutoRefine)
	os.Unsetenv(config.EnvSkillAutoTopics)
	cfg := loadAutoRefineConfig()
	if cfg.Enabled {
		t.Fatal("expected auto refine disabled by default")
	}
	if !cfg.AllowsTopic("go_runtime") {
		t.Fatal("go_runtime should be in default whitelist")
	}
	if cfg.AllowsTopic("kafka") {
		t.Fatal("kafka should not be in default whitelist")
	}
	if cfg.MinSamples != 8 {
		t.Fatalf("min samples: got %d", cfg.MinSamples)
	}
}

func TestIncrementAutoRefineSampleCounter(t *testing.T) {
	dir := t.TempDir()
	reg, err := NewSkillRegistry(dir)
	if err != nil {
		t.Fatal(err)
	}
	topic := "go_runtime"
	if err := incrementAutoRefineSampleCounter(reg, topic); err != nil {
		t.Fatal(err)
	}
	st, err := readAutoRefineState(reg, topic)
	if err != nil {
		t.Fatal(err)
	}
	if st.SamplesSinceRefine != 1 {
		t.Fatalf("samples_since_refine=%d want 1", st.SamplesSinceRefine)
	}
	path := filepath.Join(dir, "refine_state", topic+".json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("state file: %v", err)
	}
}

func TestMaybeAutoRefineSkipsWhenDisabled(t *testing.T) {
	dir := t.TempDir()
	reg, err := NewSkillRegistry(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(config.EnvSkillAutoRefine, "0")
	for i := 0; i < 20; i++ {
		_ = incrementAutoRefineSampleCounter(reg, "go_runtime")
	}
	MaybeAutoRefine(reg, "go_runtime")
	st, _ := readAutoRefineState(reg, "go_runtime")
	if st.SamplesSinceRefine != 20 {
		t.Fatalf("counter changed unexpectedly: %d", st.SamplesSinceRefine)
	}
	if !st.LastRefineAt.IsZero() {
		t.Fatal("should not have refined")
	}
}

func TestMaybeAutoRefineRespectsCooldown(t *testing.T) {
	dir := t.TempDir()
	reg, err := NewSkillRegistry(dir)
	if err != nil {
		t.Fatal(err)
	}
	topic := "go_runtime"
	t.Setenv(config.EnvSkillAutoRefine, "1")
	t.Setenv(config.EnvSkillAutoMin, "2")
	t.Setenv(config.EnvSkillAutoCooldown, "24h")
	t.Setenv(config.EnvSkillAutoTopics, "go_runtime")
	st := autoRefineState{
		LastRefineAt:       time.Now().UTC(),
		SamplesSinceRefine: 10,
	}
	if err := writeAutoRefineState(reg, topic, st); err != nil {
		t.Fatal(err)
	}
	MaybeAutoRefine(reg, topic)
	st2, _ := readAutoRefineState(reg, topic)
	if st2.SamplesSinceRefine != 10 {
		t.Fatalf("expected no refine under cooldown, samples=%d", st2.SamplesSinceRefine)
	}
}
