package main

import (
	"errors"
	"io"
	"path/filepath"
	"testing"

	"convoy/internal/app"
	"convoy/internal/orchestrator"
)

type fakeRuntime struct{}

type runtimeFactoryFunc func(cfg *app.Config) (orchestrator.Runtime, error)

func (f runtimeFactoryFunc) CreateContainer(spec orchestrator.ContainerSpec) (*orchestrator.Container, error) {
	return nil, nil
}

func (f runtimeFactoryFunc) StartContainer(id string) error               { return nil }
func (f runtimeFactoryFunc) StopContainer(id string) error                { return nil }
func (f runtimeFactoryFunc) RemoveContainer(id string) error              { return nil }
func (f runtimeFactoryFunc) Exec(id string, cmd []string) (string, error) { return "", nil }
func (f runtimeFactoryFunc) Shell(id string, stdin io.Reader, stdout, stderr io.Writer) error {
	return nil
}
func (f runtimeFactoryFunc) ListContainers() ([]*orchestrator.Container, error) {
	return nil, nil
}

func TestApplicationConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "missing.yaml")
	app := newApplication(configPath, func(cfg *app.Config) (orchestrator.Runtime, error) {
		return runtimeFactoryFunc(nil), nil
	})

	if _, err := app.Config(); err == nil {
		t.Fatalf("expected error due to missing config file")
	}
}

func TestApplicationManagerErrorsBubblesUp(t *testing.T) {
	errorFactory := func(cfg *app.Config) (orchestrator.Runtime, error) {
		return nil, errors.New("boom")
	}

	app := newApplication("", errorFactory)
	if _, err := app.Manager(); err == nil {
		t.Fatalf("expected error from runtime factory")
	}
}
