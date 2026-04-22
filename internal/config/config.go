package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DefaultBaseURL = "https://api.deepseek.com/v1"
	DefaultModel   = "deepseek-chat"
)

// FileConfig is the optional YAML file at ~/.config/ai-sre/config.yaml (or --config).
type FileConfig struct {
	APIKey            string `yaml:"api_key"`
	BaseURL           string `yaml:"base_url"`
	Model             string `yaml:"model"`
	Tier              string `yaml:"tier"`                  // free | pro | empty (pro/unlimited)
	MaxLLMCallsPerDay int    `yaml:"max_llm_calls_per_day"` // 0 = unlimited
	// OpsfleetAPIBase 可选，与 OPSFLEET_API_URL 同义，如 http://host:9080/ft-api（自升级、默认与 OpsFleet 通信）。
	OpsfleetAPIBase string `yaml:"opsfleet_api_url"`
}

// LLM holds resolved settings for the DeepSeek client.
type LLM struct {
	APIKey  string
	BaseURL string
	Model   string
}

// Limits describes product-tier constraints (变现 / 免费版).
type Limits struct {
	Tier              string
	MaxLLMCallsPerDay int
}

// ResolveDir returns the config directory: $XDG_CONFIG_HOME/ai-sre or ~/.config/ai-sre.
func ResolveDir() (string, error) {
	if d := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); d != "" {
		return filepath.Join(d, "ai-sre"), nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".config", "ai-sre"), nil
}

// LoadLLM resolves API key, optional tier limits, from files only (no env vars).
// Precedence: explicit configPath > explicit keyFilePath > defaultDir/config.yaml > defaultDir/api_key.
// If credentials come from api_key file only, tier/max_llm_calls_per_day may be merged from config.yaml in the same directory.
func LoadLLM(configPath, keyFilePath string) (*LLM, *Limits, string, error) {
	cfgDir, err := ResolveDir()
	if err != nil {
		return nil, nil, "", fmt.Errorf("config dir: %w", err)
	}

	var fc *FileConfig
	var source string

	switch {
	case strings.TrimSpace(configPath) != "":
		p := expandHome(configPath)
		fc, err = loadYAML(p)
		if err != nil {
			return nil, nil, "", err
		}
		source = p
	case strings.TrimSpace(keyFilePath) != "":
		p := expandHome(keyFilePath)
		key, err := loadKeyFile(p)
		if err != nil {
			return nil, nil, "", err
		}
		fc = &FileConfig{APIKey: key}
		source = p
		mergeTierLimitsFromConfigYAML(cfgDir, source, fc)
	default:
		yamlPath := filepath.Join(cfgDir, "config.yaml")
		if _, err := os.Stat(yamlPath); err == nil {
			fc, err = loadYAML(yamlPath)
			if err != nil {
				return nil, nil, "", err
			}
			source = yamlPath
		} else {
			keyPath := filepath.Join(cfgDir, "api_key")
			if _, err := os.Stat(keyPath); err == nil {
				key, err := loadKeyFile(keyPath)
				if err != nil {
					return nil, nil, "", err
				}
				fc = &FileConfig{APIKey: key}
				source = keyPath
				mergeTierLimitsFromConfigYAML(cfgDir, source, fc)
			}
		}
		if fc == nil {
			return nil, nil, "", fmt.Errorf(
				"llm credentials not found: create %q or %q (or use --config / --key-file), see README",
				filepath.Join(cfgDir, "config.yaml"),
				filepath.Join(cfgDir, "api_key"),
			)
		}
	}

	llm, lim, err := finalizeAll(fc, source)
	if err != nil {
		return nil, nil, "", err
	}
	return llm, lim, source, nil
}

func mergeTierLimitsFromConfigYAML(cfgDir, credentialSource string, fc *FileConfig) {
	cfgYaml := filepath.Join(cfgDir, "config.yaml")
	if credentialSource == cfgYaml {
		return
	}
	if _, err := os.Stat(cfgYaml); err != nil {
		return
	}
	b, err := os.ReadFile(cfgYaml)
	if err != nil {
		return
	}
	var extra FileConfig
	if yaml.Unmarshal(b, &extra) != nil {
		return
	}
	if strings.TrimSpace(extra.Tier) != "" {
		fc.Tier = extra.Tier
	}
	if extra.MaxLLMCallsPerDay > 0 {
		fc.MaxLLMCallsPerDay = extra.MaxLLMCallsPerDay
	}
}

func finalizeAll(fc *FileConfig, source string) (*LLM, *Limits, error) {
	llm, err := finalize(fc, source)
	if err != nil {
		return nil, nil, err
	}
	lim := &Limits{
		Tier:              strings.TrimSpace(fc.Tier),
		MaxLLMCallsPerDay: fc.MaxLLMCallsPerDay,
	}
	return llm, lim, nil
}

func finalize(fc *FileConfig, source string) (*LLM, error) {
	key := strings.TrimSpace(fc.APIKey)
	key = strings.Trim(key, "\"'`")
	if key == "" {
		return nil, fmt.Errorf("api_key is empty in %s", source)
	}
	base := strings.TrimSpace(fc.BaseURL)
	if base == "" {
		base = DefaultBaseURL
	}
	model := strings.TrimSpace(fc.Model)
	if model == "" {
		model = DefaultModel
	}
	return &LLM{APIKey: key, BaseURL: base, Model: model}, nil
}

func loadYAML(path string) (*FileConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	var fc FileConfig
	if err := yaml.Unmarshal(b, &fc); err != nil {
		return nil, fmt.Errorf("parse yaml %q: %w", path, err)
	}
	return &fc, nil
}

func loadKeyFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read api key file %q: %w", path, err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return "", errors.New("api key file is empty")
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.Trim(line, "\"'`")
		return line, nil
	}
	return "", errors.New("api key file has no valid line")
}

func expandHome(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}
	if len(path) == 1 || path[1] == '/' {
		h, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(h, path[1:])
	}
	return path
}

// LoadOptionalOpsfleetAPIBase 读取 ~/.config/ai-sre/opsfleet_api_url（install-ai-sre 写入）或 config.yaml 中 opsfleet_api_url。
// 不要求已配置 LLM；用于 ai-sre 自升级与默认指向 OpsFleet API 基址（含 /ft-api）。
func LoadOptionalOpsfleetAPIBase() string {
	cfgDir, err := ResolveDir()
	if err != nil {
		return ""
	}
	one := filepath.Join(cfgDir, "opsfleet_api_url")
	if b, err := os.ReadFile(one); err == nil {
		v := strings.TrimSpace(string(b))
		if v != "" {
			return strings.TrimRight(v, "/")
		}
	}
	p := filepath.Join(cfgDir, "config.yaml")
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	var fc FileConfig
	if yaml.Unmarshal(b, &fc) != nil {
		return ""
	}
	v := strings.TrimSpace(fc.OpsfleetAPIBase)
	if v == "" {
		return ""
	}
	return strings.TrimRight(v, "/")
}
