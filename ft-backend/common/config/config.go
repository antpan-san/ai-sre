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
	AI       AIConfig       `yaml:"ai"`
	Opsfleet OpsfleetConfig `yaml:"opsfleet"`
	K8s      K8sConfig      `yaml:"k8s"`
	Skills         SkillsConfig         `yaml:"skills"`
	AutoIteration  AutoIterationConfig  `yaml:"auto_iteration"`
	Security SecurityConfig `yaml:"security"`
	Billing  BillingConfig  `yaml:"billing"`
	Log      struct {
		Level string `yaml:"level"`
	} `yaml:"log"`
}

// AIConfig 服务端 LLM（DeepSeek 兼容）。密钥建议仅放环境变量 OPSFLEET_AI_API_KEY。
type AIConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
	Model   string `yaml:"model"`
}

// K8sConfig 控制台 K8s 部署与制品代理。
type K8sConfig struct {
	MirrorBaseURL     string `yaml:"mirror_base_url"`
	MirrorManifestURL string `yaml:"mirror_manifest_url"`
	RelayBaseURL      string `yaml:"relay_base_url"`
	AnsibleDir        string `yaml:"ansible_dir"`
}

// SkillsConfig 服务端技能样本与自迭代。
type SkillsConfig struct {
	AutoRefine SkillAutoRefineConfig   `yaml:"auto_refine"`
	Feedback   SkillFeedbackConfig     `yaml:"feedback"`
}

// SkillFeedbackConfig 用户反馈累计阈值（驱动精炼审查）。
type SkillFeedbackConfig struct {
	HelpfulThreshold   int `yaml:"helpful_threshold"`
	UnhelpfulThreshold int `yaml:"unhelpful_threshold"`
	WindowDays         int `yaml:"window_days"`
}

// AutoIterationConfig 平台自动迭代（仅 super_admin 控制台；密钥用环境变量覆盖）。
type AutoIterationConfig struct {
	Enabled                  bool   `yaml:"enabled"`
	MaxConcurrent            int    `yaml:"max_concurrent"`
	HighRiskRequiresApproval bool   `yaml:"high_risk_requires_approval"`
	DingTalkWebhook          string `yaml:"dingtalk_webhook"`
	DingTalkKeyword          string `yaml:"dingtalk_keyword"`
	GitHubRepo               string `yaml:"github_repo"`
	CodeAgentToken           string `yaml:"code_agent_token"`
}

// SkillAutoRefineConfig 样本达阈值后自动 RefineSkill（可被 OPSFLEET_SKILL_AUTO_REFINE_* 覆盖）。
type SkillAutoRefineConfig struct {
	Enabled    bool     `yaml:"enabled"`
	MinSamples int      `yaml:"min_samples"`
	Cooldown   string   `yaml:"cooldown"`
	Topics     []string `yaml:"topics"`
	MaxPerDay  int      `yaml:"max_per_day"`
}

// BillingPackage maps a Stripe Price 到订阅后授予的功能包和兼容 feature_key。
// 若 billing.packages 为空且 stripe_price_id_pro 非空，运行时会退化为备份与性能包（兼容旧配置）。
type BillingPackage struct {
	ID            string   `yaml:"id"`
	DisplayName   string   `yaml:"display_name"`
	StripePriceID string   `yaml:"stripe_price_id"`
	FeatureKeys   []string `yaml:"feature_keys"`
}

// BillingConfig Stripe 对接：密钥留空则关闭收银台；packages 为多档订阅。
type BillingConfig struct {
	StripeSecretKey     string           `yaml:"stripe_secret_key"`
	StripeWebhookSecret string           `yaml:"stripe_webhook_secret"`
	StripePriceIDPro    string           `yaml:"stripe_price_id_pro"` // 兼容旧版单包：授予备份与性能包
	PublicAppBaseURL    string           `yaml:"public_app_base_url"`
	Packages            []BillingPackage `yaml:"packages"`
}

// OpsfleetConfig 控制台扩展（K8s 部署页 curl 安装 ai-sre、技能数据目录等）。
type OpsfleetConfig struct {
	// AiSreBinaryPath 默认/legacy：Linux 可执行文件绝对路径；未单独配置 *_amd64 / *_arm64 时 amd64 与「未带 arch」下载均用此文件。
	AiSreBinaryPath string `yaml:"ai_sre_binary_path"`
	// AiSreBinaryPathAmd64 可选：显式 amd64 分发（GET .../cli/ai-sre?arch=amd64 优先于 ai_sre_binary_path）。
	AiSreBinaryPathAmd64 string `yaml:"ai_sre_binary_path_amd64"`
	// AiSreBinaryPathArm64 可选：显式 arm64 分发（?arch=arm64）；与 amd64 分属不同文件时必配，否则 ARM 机会拿到错误 ELF。
	AiSreBinaryPathArm64 string `yaml:"ai_sre_binary_path_arm64"`
	// AISkillDataDir 样本/反馈/generated 技能包目录（可被 OPSFLEET_AI_SKILL_DATA_DIR 覆盖）。
	AISkillDataDir string `yaml:"ai_skill_data_dir"`
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

type SecurityConfig struct {
	CORSAllowedOrigins []string `yaml:"cors_allowed_origins"`
	// DisablePublicRegistration 为 true 时关闭 POST /api/auth/register（仅管理员可在后台建号）。
	DisablePublicRegistration bool `yaml:"disable_public_registration"`
	// DisableLoginCaptcha 为 true 时关闭登录算术验证码（仅依赖现有限流；内网可设 true）。
	DisableLoginCaptcha bool `yaml:"disable_login_captcha"`
}

// PublicRegistrationAllowed 默认允许公开注册（未配置 disable 时为 true）。
func (s SecurityConfig) PublicRegistrationAllowed() bool {
	return !s.DisablePublicRegistration
}

// LoginCaptchaRequired 默认要求登录验证码（未配置 disable 时为 true）。
func (s SecurityConfig) LoginCaptchaRequired() bool {
	return !s.DisableLoginCaptcha
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
				AccessTokenExp:  1440,
				RefreshTokenExp: 10080,
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
			Security: SecurityConfig{
				CORSAllowedOrigins: []string{
					"http://localhost:5173",
					"http://127.0.0.1:5173",
					"http://127.0.0.1:9080",
					"http://192.168.56.11:9080",
					"http://opsfleetpilot.com",
					"https://opsfleetpilot.com",
					"http://opsfleetpilot.com:9080",
					"https://opsfleetpilot.com:9080",
				},
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
	if len(config.Security.CORSAllowedOrigins) == 0 {
		config.Security.CORSAllowedOrigins = []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
			"http://127.0.0.1:9080",
			"http://192.168.56.11:9080",
			"http://opsfleetpilot.com",
			"https://opsfleetpilot.com",
			"http://opsfleetpilot.com:9080",
			"https://opsfleetpilot.com:9080",
		}
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
