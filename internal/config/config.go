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
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
	Model   string `yaml:"model"`
}

// LLM holds resolved settings for the DeepSeek client.
type LLM struct {
	APIKey  string
	BaseURL string
	Model   string
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

// LoadLLM resolves API key and optional base URL / model from files only (no env vars).
// Precedence: explicit configPath > explicit keyFilePath > defaultDir/config.yaml > defaultDir/api_key.
// Returns the resolved path used for credentials (for logging only; never log key contents).
func LoadLLM(configPath, keyFilePath string) (*LLM, string, error) {
	cfgDir, err := ResolveDir()
	if err != nil {
		return nil, "", fmt.Errorf("config dir: %w", err)
	}

	var fc *FileConfig
	var source string

	switch {
	case strings.TrimSpace(configPath) != "":
		p := expandHome(configPath)
		fc, err = loadYAML(p)
		if err != nil {
			return nil, "", err
		}
		source = p
	case strings.TrimSpace(keyFilePath) != "":
		p := expandHome(keyFilePath)
		key, err := loadKeyFile(p)
		if err != nil {
			return nil, "", err
		}
		out, err := finalize(&FileConfig{APIKey: key}, p)
		if err != nil {
			return nil, "", err
		}
		return out, p, nil
	default:
		yamlPath := filepath.Join(cfgDir, "config.yaml")
		if _, err := os.Stat(yamlPath); err == nil {
			fc, err = loadYAML(yamlPath)
			if err != nil {
				return nil, "", err
			}
			source = yamlPath
		} else {
			keyPath := filepath.Join(cfgDir, "api_key")
			if _, err := os.Stat(keyPath); err == nil {
				key, err := loadKeyFile(keyPath)
				if err != nil {
					return nil, "", err
				}
				fc = &FileConfig{APIKey: key}
				source = keyPath
			}
		}
		if fc == nil {
			return nil, "", fmt.Errorf(
				"llm credentials not found: create %q or %q (or use --config / --key-file), see README",
				filepath.Join(cfgDir, "config.yaml"),
				filepath.Join(cfgDir, "api_key"),
			)
		}
	}

	out, err := finalize(fc, source)
	if err != nil {
		return nil, "", err
	}
	return out, source, nil
}

func finalize(fc *FileConfig, source string) (*LLM, error) {
	key := strings.TrimSpace(fc.APIKey)
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
	// first non-empty line only
	// allow comments starting with #
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
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
