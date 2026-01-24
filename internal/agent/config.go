package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	agentConfigDirEnvVar = "CONVOY_CONFIG_DIR"
	agentConfigDirName   = ".config/convoy"
	agentConfigFileName  = "agent.yaml"
)

// Config represents the agent runtime configuration.
type Config struct {
	GRPCPort      int
	ShellPath     string
	MaxConcurrent int
	ExecTimeout   time.Duration
	AgentID       string
	ConfigPath    string
}

type fileConfig struct {
	GRPCPort       int    `yaml:"grpc_port"`
	ShellPath      string `yaml:"shell_path"`
	MaxConcurrent  int    `yaml:"max_concurrent"`
	ExecTimeoutSec int    `yaml:"exec_timeout_sec"`
	AgentID        string `yaml:"agent_id"`
}

const (
	defaultGRPCPort      = 6000
	defaultShellPath     = "/bin/sh"
	defaultMaxConcurrent = 4
	defaultExecTimeout   = 60
)

// LoadConfig loads the agent configuration from disk, applying environment overrides.
func LoadConfig(path string) (*Config, error) {
	configPath := path
	if configPath == "" {
		var err error
		configPath, err = DefaultConfigPath()
		if err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", configPath, err)
	}

	var cfg fileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", configPath, err)
	}

	applyDefaults(&cfg)
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	agentCfg := &Config{
		GRPCPort:      cfg.GRPCPort,
		ShellPath:     cfg.ShellPath,
		MaxConcurrent: cfg.MaxConcurrent,
		ExecTimeout:   time.Duration(cfg.ExecTimeoutSec) * time.Second,
		AgentID:       cfg.AgentID,
		ConfigPath:    configPath,
	}

	if port := getEnvInt("CONVOY_AGENT_GRPC_PORT", 0); port > 0 {
		agentCfg.GRPCPort = port
	}

	if shell := getEnv("CONVOY_AGENT_SHELL", ""); shell != "" {
		agentCfg.ShellPath = shell
	}

	if max := getEnvInt("CONVOY_AGENT_MAX_CONCURRENT", 0); max > 0 {
		agentCfg.MaxConcurrent = max
	}

	if timeout := getEnvDuration("CONVOY_AGENT_EXEC_TIMEOUT", 0); timeout > 0 {
		agentCfg.ExecTimeout = timeout
	}

	if agentID := getEnv("CONVOY_AGENT_ID", ""); agentID != "" {
		agentCfg.AgentID = agentID
	}

	return agentCfg, nil
}

// DefaultConfigPath returns the absolute path to the agent config file.
func DefaultConfigPath() (string, error) {
	dir, err := defaultConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, agentConfigFileName), nil
}

func defaultConfigDir() (string, error) {
	if dir := os.Getenv(agentConfigDirEnvVar); dir != "" {
		return dir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}

	return filepath.Join(home, agentConfigDirName), nil
}

func applyDefaults(cfg *fileConfig) {
	if cfg.GRPCPort == 0 {
		cfg.GRPCPort = defaultGRPCPort
	}

	if strings.TrimSpace(cfg.ShellPath) == "" {
		cfg.ShellPath = defaultShellPath
	}

	if cfg.MaxConcurrent == 0 {
		cfg.MaxConcurrent = defaultMaxConcurrent
	}

	if cfg.ExecTimeoutSec == 0 {
		cfg.ExecTimeoutSec = defaultExecTimeout
	}

	if strings.TrimSpace(cfg.AgentID) == "" {
		cfg.AgentID = defaultAgentID()
	}
}

func validateConfig(cfg fileConfig) error {
	var problems []string

	if cfg.GRPCPort <= 0 || cfg.GRPCPort > 65535 {
		problems = append(problems, "grpc_port must be between 1 and 65535")
	}

	if strings.TrimSpace(cfg.ShellPath) == "" {
		problems = append(problems, "shell_path is required")
	}

	if cfg.MaxConcurrent <= 0 {
		problems = append(problems, "max_concurrent must be greater than 0")
	}

	if cfg.ExecTimeoutSec <= 0 {
		problems = append(problems, "exec_timeout_sec must be greater than 0")
	}

	if strings.TrimSpace(cfg.AgentID) == "" {
		problems = append(problems, "agent_id is required")
	}

	if len(problems) > 0 {
		return errors.New("invalid config: " + strings.Join(problems, "; "))
	}

	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			return parsed
		}
	}
	return fallback
}

func defaultAgentID() string {
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		return hostname
	}
	return "convoy-agent"
}
