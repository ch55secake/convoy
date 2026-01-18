package main

import (
	"errors"
	"io"
	"path/filepath"
	"testing"

	"convoy/internal/app"
	"convoy/internal/orchestrator"
)

type runtimeFactoryFunc func(cfg *app.Config) (orchestrator.Runtime, error)

func (f runtimeFactoryFunc) CreateContainer(_ orchestrator.ContainerSpec) (*orchestrator.Container, error) {
	return nil, nil
}

func (f runtimeFactoryFunc) StartContainer(_ string) error             { return nil }
func (f runtimeFactoryFunc) StopContainer(_ string) error              { return nil }
func (f runtimeFactoryFunc) RemoveContainer(_ string) error            { return nil }
func (f runtimeFactoryFunc) Exec(_ string, _ []string) (string, error) { return "", nil }
func (f runtimeFactoryFunc) Shell(_ string, _ io.Reader, _, _ io.Writer) error {
	return nil
}
func (f runtimeFactoryFunc) ListContainers() ([]*orchestrator.Container, error) {
	return nil, nil
}

func TestApplicationConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "missing.yaml")
	application := newApplication(configPath, func(cfg *app.Config) (orchestrator.Runtime, error) {
		return runtimeFactoryFunc(nil), nil
	})

	if _, err := application.Config(); err == nil {
		t.Fatalf("expected error due to missing config file")
	}
}

func TestApplicationManagerErrorsBubblesUp(t *testing.T) {
	errorFactory := func(cfg *app.Config) (orchestrator.Runtime, error) {
		return nil, errors.New("boom")
	}

	application := newApplication("", errorFactory)
	if _, err := application.Manager(); err == nil {
		t.Fatalf("expected error from runtime factory")
	}
}
