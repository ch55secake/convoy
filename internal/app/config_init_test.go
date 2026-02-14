package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitializeConfigCreatesFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	result, err := InitializeConfig(cfgPath, false)
	if err != nil {
		t.Fatalf("InitializeConfig error: %v", err)
	}

	if result.Path != cfgPath {
		t.Fatalf("expected path %s, got %s", cfgPath, result.Path)
	}

	if result.Overwritten {
		t.Fatal("expected Overwritten to be false for new config")
	}

	if result.BackupPath != "" {
		t.Fatal("expected BackupPath to be empty for new config")
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	if string(data) != defaultConfigYAML {
		t.Fatalf("unexpected content: %s", string(data))
	}
}

func TestInitializeConfigOverwriteError(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	// Create initial config
	_, err := InitializeConfig(cfgPath, false)
	if err != nil {
		t.Fatalf("InitializeConfig error: %v", err)
	}

	// Try to create again without force
	_, err = InitializeConfig(cfgPath, false)
	if err == nil {
		t.Fatal("expected error when config exists without force flag")
	}

	expectedErrMsg := "config already exists"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Fatalf("expected error to contain %q, got %q", expectedErrMsg, err.Error())
	}
}

func TestInitializeConfigForceOverwrite(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	// Create initial config
	_, err := InitializeConfig(cfgPath, false)
	if err != nil {
		t.Fatalf("InitializeConfig error: %v", err)
	}

	// Modify the config to verify it gets overwritten
	modifiedContent := "modified content"
	if err := os.WriteFile(cfgPath, []byte(modifiedContent), 0o644); err != nil {
		t.Fatalf("failed to modify config: %v", err)
	}

	// Overwrite with force
	result, err := InitializeConfig(cfgPath, true)
	if err != nil {
		t.Fatalf("InitializeConfig error: %v", err)
	}

	if !result.Overwritten {
		t.Fatal("expected Overwritten to be true")
	}

	if result.BackupPath == "" {
		t.Fatal("expected BackupPath to be set")
	}

	// Verify backup exists and contains modified content
	backupData, err := os.ReadFile(result.BackupPath)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}

	if string(backupData) != modifiedContent {
		t.Fatalf("backup should contain original modified content, got: %s", string(backupData))
	}

	// Verify config now has default content
	configData, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	if string(configData) != defaultConfigYAML {
		t.Fatalf("config should have default content after overwrite, got: %s", string(configData))
	}
}
