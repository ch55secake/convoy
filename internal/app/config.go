package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	configDirEnvVar = "CONVOY_CONFIG_DIR"
	configDirName   = ".config/convoy"
	configFileName  = "config.yaml"
)

const defaultConfigYAML = `# Configuration for convoy orchestrator
image: convoy:latest
grpc_port: 50051
docker_host: unix:///var/run/docker.sock
docker_network: bridge
agent_grpc_port: 6000
pull_always: false
pull_timeout_sec: 300

# Git credentials mounting - mount host ~/.ssh and ~/.gitconfig into containers
git_credentials:
  enabled: false        # Set to true to enable git credential mounting
  mount_ssh: true       # Mount ~/.ssh as /root/.ssh (read-only)
  mount_gitconfig: true # Mount ~/.gitconfig as /root/.gitconfig (read-only)

# Bash profile mounting - mount a custom bash profile into containers
bash_profile:
  enabled: false        # Set to true to enable bash profile mounting
  host_path: ""         # Path on host (default: ~/.config/convoy/.bash_profile)
`

// InitResult contains the result of a configuration initialization operation.
type InitResult struct {
	Path        string // The path to the configuration file
	Overwritten bool   // Whether an existing config was overwritten
	BackupPath  string // Path to the backup file (empty if no backup created)
}

// GitCredentialsConfig holds configuration for mounting git credentials into containers.
type GitCredentialsConfig struct {
	Enabled        bool `yaml:"enabled"`         // Enable git credential mounting
	MountSSH       bool `yaml:"mount_ssh"`       // Mount ~/.ssh as /root/.ssh
	MountGitconfig bool `yaml:"mount_gitconfig"` // Mount ~/.gitconfig as /root/.gitconfig
}

// BashProfileConfig holds configuration for mounting a bash profile into containers.
type BashProfileConfig struct {
	Enabled  bool   `yaml:"enabled"`   // Enable bash profile mounting
	HostPath string `yaml:"host_path"` // Path on host to the bash profile (default: ~/.config/convoy/.bash_profile)
}

// Config holds application configuration loaded from YAML.
type Config struct {
	Image          string               `yaml:"image"`
	GRPCPort       int                  `yaml:"grpc_port"`
	DockerHost     string               `yaml:"docker_host"`
	DockerNetwork  string               `yaml:"docker_network"`
	AgentGRPCPort  int                  `yaml:"agent_grpc_port"`
	PullAlways     bool                 `yaml:"pull_always"`
	PullTimeoutSec int                  `yaml:"pull_timeout_sec"`
	GitCredentials GitCredentialsConfig `yaml:"git_credentials"`
	BashProfile    BashProfileConfig    `yaml:"bash_profile"`
}

// InitializeConfig creates a default configuration file at the specified path.
// If the file already exists and force is false, it returns an error.
// If force is true, it creates a backup and overwrites the existing config.
// Returns the initialization result or an error if the operation fails.
func InitializeConfig(path string, force bool) (*InitResult, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create config dir %q: %w", dir, err)
	}

	result := &InitResult{
		Path:        path,
		Overwritten: false,
		BackupPath:  "",
	}

	// Check if config already exists
	if _, err := os.Stat(path); err == nil {
		if !force {
			return nil, fmt.Errorf("config already exists at %s (use --force to overwrite)", path)
		}

		// Create backup with timestamp to avoid overwriting existing backups
		timestamp := time.Now().Format("20060102-150405")
		backupPath := fmt.Sprintf("%s.backup.%s", path, timestamp)

		// Read existing config
		existingData, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read existing config for backup %q: %w", path, err)
		}

		// Write backup
		if err := os.WriteFile(backupPath, existingData, 0o644); err != nil {
			return nil, fmt.Errorf("create backup at %q: %w", backupPath, err)
		}

		result.Overwritten = true
		result.BackupPath = backupPath
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("stat config %q: %w", path, err)
	}

	// Write the new configuration
	if err := os.WriteFile(path, []byte(defaultConfigYAML), 0o644); err != nil {
		return nil, fmt.Errorf("write config %q: %w", path, err)
	}

	return result, nil
}

// LoadConfig loads configuration from the provided path. When path is empty the
// default location (~/.config/convoy/config.yaml) is used. The location can be
// overridden with the CONVOY_CONFIG_DIR environment variable.
func LoadConfig(path string) (*Config, error) {
	cfgPath := path
	if cfgPath == "" {
		var err error
		cfgPath, err = DefaultConfigPath()
		if err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", cfgPath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", cfgPath, err)
	}

	applyDefaults(&cfg)
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// DefaultConfigPath returns the absolute path to the config file using the
// default config directory (~/.config/convoy) unless overridden.
func DefaultConfigPath() (string, error) {
	dir, err := defaultConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, configFileName), nil
}

func defaultConfigDir() (string, error) {
	if dir := os.Getenv(configDirEnvVar); dir != "" {
		return dir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}

	return filepath.Join(home, configDirName), nil
}

// Validate ensures the configuration contains the minimum required fields.
func (c *Config) Validate() error {
	var problems []string

	if strings.TrimSpace(c.Image) == "" {
		problems = append(problems, "image is required")
	}

	if c.GRPCPort <= 0 || c.GRPCPort > 65535 {
		problems = append(problems, "grpc_port must be between 1 and 65535")
	}

	if strings.TrimSpace(c.DockerHost) == "" {
		problems = append(problems, "docker_host is required")
	}

	if c.AgentGRPCPort <= 0 || c.AgentGRPCPort > 65535 {
		problems = append(problems, "agent_grpc_port must be between 1 and 65535")
	}

	if c.PullTimeoutSec < 0 {
		problems = append(problems, "pull_timeout_sec cannot be negative")
	}

	if len(problems) > 0 {
		return errors.New("invalid config: " + strings.Join(problems, "; "))
	}

	return nil
}

func applyDefaults(cfg *Config) {
	if cfg.GRPCPort == 0 {
		cfg.GRPCPort = 50051
	}

	if cfg.AgentGRPCPort == 0 {
		cfg.AgentGRPCPort = 6000
	}

	if strings.TrimSpace(cfg.DockerHost) == "" {
		cfg.DockerHost = "unix:///var/run/docker.sock"
	}

	if strings.TrimSpace(cfg.DockerNetwork) == "" {
		cfg.DockerNetwork = "bridge"
	}

	if cfg.PullTimeoutSec == 0 {
		cfg.PullTimeoutSec = 300
	}
}
