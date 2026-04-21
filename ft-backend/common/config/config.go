package config

import (
	"fmt"
	"os"

	"ft-backend/common/logger"

	"gopkg.in/yaml.v3"
)

var (
	GlobalCfg *Config
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
	File     FileConfig     `yaml:"file"`
	Redis    RedisConfig    `yaml:"redis"`
	Opsfleet OpsfleetConfig `yaml:"opsfleet"`
	Log      struct {
		Level string `yaml:"level"`
	} `yaml:"log"`
}

// OpsfleetConfig 控制台扩展（K8s 部署页 curl 安装 ai-sre 等）。
type OpsfleetConfig struct {
	// AiSreBinaryPath 服务器上已构建的 Linux ai-sre 可执行文件绝对路径，用于公开下载（amd64/arm64 暂共用同一文件时请保证与目标机架构一致）。
	AiSreBinaryPath string `yaml:"ai_sre_binary_path"`
}

type ServerConfig struct {
	Host         string `yaml:"host"`
	Port         string `yaml:"port"`
	ReadTimeout  int    `yaml:"read_timeout"`
	WriteTimeout int    `yaml:"write_timeout"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
	TimeZone string `yaml:"timezone"`
}

type JWTConfig struct {
	SecretKey       string `yaml:"secret_key"`
	AccessTokenExp  int    `yaml:"access_token_exp"`
	RefreshTokenExp int    `yaml:"refresh_token_exp"`
}

type FileConfig struct {
	UploadDir      string   `yaml:"upload_dir"`
	MaxFileSize    int64    `yaml:"max_file_size"`
	ChunkSize      int      `yaml:"chunk_size"`
	AllowedFormats []string `yaml:"allowed_formats"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type ClientConfig struct {
	EncryptKey string `yaml:"encrypt_key"`
}

func LoadConfig() (*Config, error) {
	configFile := "conf/config.yaml"

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		defaultConfig := &Config{
			Server: ServerConfig{
				Host:         "0.0.0.0",
				Port:         "8080",
				ReadTimeout:  30,
				WriteTimeout: 30,
			},
			Database: DatabaseConfig{
				Host:     "127.0.0.1",
				Port:     "5432",
				User:     "postgres",
				Password: "postgres",
				DBName:   "opsfleetpilot",
				SSLMode:  "disable",
				TimeZone: "Asia/Shanghai",
			},
			JWT: JWTConfig{
				SecretKey:       "your-secret-key-here",
				AccessTokenExp:  15,
				RefreshTokenExp: 1440,
			},
			File: FileConfig{
				UploadDir:      "uploads",
				MaxFileSize:    1073741824,
				ChunkSize:      1048576,
				AllowedFormats: []string{"jpg", "png", "pdf", "txt", "zip", "rar"},
			},
			Redis: RedisConfig{
				Host:     "localhost",
				Port:     "6379",
				Password: "",
				DB:       0,
			},
			Log: struct {
				Level string `yaml:"level"`
			}{
				Level: "info",
			},
		}

		if err := SaveConfig(defaultConfig, configFile); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}

		return defaultConfig, nil
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults for new PostgreSQL fields if missing
	if config.Database.SSLMode == "" {
		config.Database.SSLMode = "disable"
	}
	if config.Database.TimeZone == "" {
		config.Database.TimeZone = "Asia/Shanghai"
	}

	return &config, nil
}

func EnsureConfigExists(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	logger.Info("Config file does not exist: %s", path)
	return nil
}

func SaveConfig(config *Config, filename string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func GetConfig() *Config {
	return GlobalCfg
}
