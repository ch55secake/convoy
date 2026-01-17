package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_DefaultLocation(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	configYAML := "image: test\ngrpc_port: 1234\ndocker_host: unix:///tmp/docker.sock\nagent_grpc_port: 7000"
	if err := os.WriteFile(cfgPath, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv(configDirEnvVar, tmpDir)

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	if cfg.Image != "test" {
		t.Fatalf("expected image test, got %s", cfg.Image)
	}

	if cfg.GRPCPort != 1234 {
		t.Fatalf("expected grpc_port 1234, got %d", cfg.GRPCPort)
	}

	if cfg.DockerHost != "unix:///tmp/docker.sock" {
		t.Fatalf("unexpected docker host: %s", cfg.DockerHost)
	}

	if cfg.AgentGRPCPort != 7000 {
		t.Fatalf("unexpected agent grpc port: %d", cfg.AgentGRPCPort)
	}
}

func TestConfigValidate(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}

	cfg.Image = "alpine"
	cfg.GRPCPort = 4999
	cfg.DockerHost = "unix:///var/run/docker.sock"
	cfg.AgentGRPCPort = 6000

	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
