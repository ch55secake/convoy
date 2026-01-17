package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	configDirEnvVar = "CONVOY_CONFIG_DIR"
	configDirName   = ".config/convoy"
	configFileName  = "config.yaml"
)

// Config holds application configuration loaded from YAML.
type Config struct {
	Image      string `yaml:"image"`
	GRPCPort   int    `yaml:"grpc_port"`
	DockerHost string `yaml:"docker_host"`
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

	if len(problems) > 0 {
		return errors.New("invalid config: " + strings.Join(problems, "; "))
	}

	return nil
}

func applyDefaults(cfg *Config) {
	if cfg.GRPCPort == 0 {
		cfg.GRPCPort = 50051
	}

	if strings.TrimSpace(cfg.DockerHost) == "" {
		cfg.DockerHost = "unix:///var/run/docker.sock"
	}
}
