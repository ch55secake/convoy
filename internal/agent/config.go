package agent

import (
	"os"
	"strconv"
	"time"

	"convoy/internal/app"
)

// Config represents the agent runtime configuration.
type Config struct {
	AppConfig     *app.Config
	GRPCPort      int
	ShellPath     string
	MaxConcurrent int
	ExecTimeout   time.Duration
	AgentID       string
	ConfigPath    string
}

// LoadConfig loads the agent configuration from disk, applying environment overrides.
func LoadConfig(path string) (*Config, error) {
	appCfg, err := app.LoadConfig(path)
	if err != nil {
		return nil, err
	}

	agentCfg := &Config{
		AppConfig:     appCfg,
		GRPCPort:      appCfg.GRPCPort,
		ShellPath:     getEnv("CONVOY_AGENT_SHELL", "/bin/sh"),
		MaxConcurrent: getEnvInt("CONVOY_AGENT_MAX_CONCURRENT", 4),
		ExecTimeout:   getEnvDuration("CONVOY_AGENT_EXEC_TIMEOUT", time.Minute),
		AgentID:       getEnv("CONVOY_AGENT_ID", defaultAgentID()),
		ConfigPath:    path,
	}

	if port := getEnvInt("CONVOY_AGENT_GRPC_PORT", 0); port > 0 {
		agentCfg.GRPCPort = port
	}

	if agentCfg.MaxConcurrent <= 0 {
		agentCfg.MaxConcurrent = 1
	}

	if agentCfg.ExecTimeout <= 0 {
		agentCfg.ExecTimeout = time.Minute
	}

	return agentCfg, nil
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
