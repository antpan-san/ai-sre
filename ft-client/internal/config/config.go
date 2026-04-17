// Package config provides configuration loading and validation for the client agent.
package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration structure for the client agent.
type Config struct {
	Server       ServerConfig      `yaml:"server"`
	Client       ClientConfig      `yaml:"client"`
	Cluster      ClusterConfig     `yaml:"cluster"`
	Auth         AuthConfig        `yaml:"auth"`
	Heartbeat    HeartbeatConfig   `yaml:"heartbeat"`
	NodeManager  NodeManagerConfig `yaml:"node_manager"`  // Node management settings
	ManagedNodes []ManagedNode     `yaml:"managed_nodes"` // Worker nodes managed by this master
	Log          LogConfig         `yaml:"log"`
	TLS          TLSConfig         `yaml:"tls"`
}

// NodeManagerConfig holds settings for secondary node management.
type NodeManagerConfig struct {
	Enabled       bool `yaml:"enabled"`        // Enable node management (master role only)
	ProbeInterval int  `yaml:"probe_interval"` // Node health check interval in seconds
	ProbeTimeout  int  `yaml:"probe_timeout"`  // Node health check timeout in seconds
}

// ManagedNode defines a worker node that the master collects metrics from via SSH.
type ManagedNode struct {
	IP          string `yaml:"ip"`
	Hostname    string `yaml:"hostname"`
	SSHPort     int    `yaml:"ssh_port"`
	SSHUser     string `yaml:"ssh_user"`
	AuthType    string `yaml:"auth_type,omitempty"`    // "password" or "key" (default: "key")
	SSHPassword string `yaml:"ssh_password,omitempty"` // SSH password (when auth_type=password)
	SSHKey      string `yaml:"ssh_key,omitempty"`      // SSH private key path (when auth_type=key)
}

// ServerConfig holds the Server connection settings.
type ServerConfig struct {
	URL string `yaml:"url"` // Server URL (http:// or https://)
}

// ClientConfig holds the client identity settings.
type ClientConfig struct {
	ID             string `yaml:"id"`              // Unique client ID (auto-generated if empty)
	Version        string `yaml:"version"`         // Agent version
	BusinessModule string `yaml:"business_module"` // Business module identifier
	Role           string `yaml:"role"`            // Node role: "master" or "worker"
}

// ClusterConfig holds the cluster topology settings.
type ClusterConfig struct {
	ID   string `yaml:"id"`   // Cluster identifier (required)
	Name string `yaml:"name"` // Cluster display name (optional, defaults to ID)
}

// AuthConfig holds the authentication settings for server communication.
type AuthConfig struct {
	Token string `yaml:"token"` // Authentication token for heartbeat API
}

// Role constants for node topology.
const (
	RoleMaster = "master"
	RoleWorker = "worker"
)

// HeartbeatConfig holds the heartbeat timing settings.
type HeartbeatConfig struct {
	Interval      int `yaml:"interval"`       // Heartbeat interval in seconds
	RetryInterval int `yaml:"retry_interval"` // Retry interval after failure in seconds
	MaxFailures   int `yaml:"max_failures"`   // Max consecutive failures before warning
}

// LogConfig holds the logging settings.
type LogConfig struct {
	Level      string `yaml:"level"`       // Log level: debug, info, warn, error
	File       string `yaml:"file"`        // Log file path (empty = stdout only)
	MaxSize    int    `yaml:"max_size"`    // Max file size in MB before rotation
	MaxBackups int    `yaml:"max_backups"` // Number of rotated files to keep
}

// TLSConfig holds the TLS/HTTPS settings.
type TLSConfig struct {
	Enabled    bool   `yaml:"enabled"`     // Enable TLS certificate verification
	CACert     string `yaml:"ca_cert"`     // Path to CA certificate file
	SkipVerify bool   `yaml:"skip_verify"` // Skip TLS verification (dev only)
}

// Load reads and parses a YAML configuration file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file %s: %w", path, err)
	}

	// Remember the config path for later Save() calls
	configPath = path

	cfg.applyDefaults()

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// Default returns a Config with sensible defaults.
func Default() *Config {
	cfg := &Config{}
	cfg.applyDefaults()
	return cfg
}

// applyDefaults fills zero-value fields with default values.
func (c *Config) applyDefaults() {
	// Server defaults
	if c.Server.URL == "" {
		c.Server.URL = "https://localhost:8080"
	}

	// Client defaults
	if c.Client.Version == "" {
		c.Client.Version = "1.0.0"
	}
	if c.Client.BusinessModule == "" {
		c.Client.BusinessModule = "default"
	}
	if c.Client.Role == "" {
		// Default to master: each ft-client instance is the control node for its
		// cluster, managing worker nodes via SSH.  Users can override this with
		// client.role = worker/standalone in the config file.
		c.Client.Role = RoleMaster
	}

	// Cluster defaults — provide a stable default so topology resolution works
	// even when the user hasn't explicitly configured a cluster ID.
	if c.Cluster.ID == "" {
		c.Cluster.ID = "default-cluster"
	}
	if c.Cluster.Name == "" {
		c.Cluster.Name = c.Cluster.ID // Use cluster ID as display name if not set
	}

	// Heartbeat defaults
	if c.Heartbeat.Interval <= 0 {
		c.Heartbeat.Interval = 5
	}
	if c.Heartbeat.RetryInterval <= 0 {
		c.Heartbeat.RetryInterval = 10
	}
	if c.Heartbeat.MaxFailures <= 0 {
		c.Heartbeat.MaxFailures = 3
	}

	// ManagedNodes defaults
	for i := range c.ManagedNodes {
		if c.ManagedNodes[i].SSHPort <= 0 {
			c.ManagedNodes[i].SSHPort = 22
		}
		if c.ManagedNodes[i].SSHUser == "" {
			c.ManagedNodes[i].SSHUser = "root"
		}
	}

	// NodeManager defaults
	if c.NodeManager.ProbeInterval <= 0 {
		c.NodeManager.ProbeInterval = 30 // 30 seconds between probes
	}
	if c.NodeManager.ProbeTimeout <= 0 {
		c.NodeManager.ProbeTimeout = 15 // 15 seconds per probe (SSH collect + high-latency nodes)
	}
	// Auto-enable node manager if this is a master with managed nodes
	if c.Client.Role == RoleMaster && len(c.ManagedNodes) > 0 {
		c.NodeManager.Enabled = true
	}

	// Log defaults
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if c.Log.MaxSize <= 0 {
		c.Log.MaxSize = 100
	}
	if c.Log.MaxBackups <= 0 {
		c.Log.MaxBackups = 3
	}
}

// validate checks that the configuration is valid.
func (c *Config) validate() error {
	if c.Server.URL == "" {
		return fmt.Errorf("server.url is required")
	}
	if !strings.HasPrefix(c.Server.URL, "http://") && !strings.HasPrefix(c.Server.URL, "https://") {
		return fmt.Errorf("server.url must start with http:// or https://")
	}

	// Validate role — allow "standalone" as a valid role for single-node operation
	role := strings.ToLower(c.Client.Role)
	if role != RoleMaster && role != RoleWorker && role != "standalone" {
		return fmt.Errorf("client.role must be one of: master, worker, standalone")
	}
	c.Client.Role = role

	// cluster.id is optional: if empty, the machine registers as standalone.
	// It can be assigned later via the web UI or a sync_nodes command from the server.

	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[strings.ToLower(c.Log.Level)] {
		return fmt.Errorf("log.level must be one of: debug, info, warn, error")
	}

	return nil
}

// configPath stores the path to the config file for persistence (set during Load or SaveTo).
var configPath string

// Save writes the current configuration back to the config file.
// Returns an error if no config file path has been established yet; use SaveTo in that case.
func Save(cfg *Config) error {
	if configPath == "" {
		return fmt.Errorf("config path not set (config was not loaded from file)")
	}
	return SaveTo(cfg, configPath)
}

// SaveTo writes the configuration to an explicit file path and remembers it for future Save calls.
func SaveTo(cfg *Config, path string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	configPath = path
	return nil
}

// GenerateClientID creates a unique client identifier based on hostname and random bytes.
func GenerateClientID() string {
	hostname, _ := os.Hostname()
	hostname = strings.ReplaceAll(hostname, " ", "-")
	hostname = strings.ToLower(hostname)

	b := make([]byte, 4)
	_, _ = rand.Read(b)
	suffix := hex.EncodeToString(b)

	return fmt.Sprintf("client-%s-%s-%s", hostname, runtime.GOOS, suffix)
}
