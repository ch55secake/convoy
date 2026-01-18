package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitializeConfigCreatesFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	created, err := InitializeConfig(cfgPath)
	if err != nil {
		t.Fatalf("InitializeConfig error: %v", err)
	}

	if created != cfgPath {
		t.Fatalf("expected path %s, got %s", cfgPath, created)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	if string(data) != defaultConfigYAML {
		t.Fatalf("unexpected content: %s", string(data))
	}
}
