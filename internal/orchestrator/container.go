package orchestrator

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// Container represents a managed container instance.
type Container struct {
	ID        string
	Image     string
	Endpoint  string
	Labels    map[string]string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ContainerSpec describes how a new container should be created.
type ContainerSpec struct {
	Image       string
	Labels      map[string]string
	Environment map[string]string
	Command     []string
}

// Runtime defines the behavior required from a container runtime implementation.
type Runtime interface {
	CreateContainer(spec ContainerSpec) (*Container, error)
	StartContainer(id string) error
	StopContainer(id string) error
	RemoveContainer(id string) error
	Exec(id string, cmd []string) (string, error)
	Shell(id string, stdin io.Reader, stdout, stderr io.Writer) error
}

// Manager coordinates container operations through the Runtime interface.
type Manager struct {
	runtime Runtime
}

// NewManager constructs a Manager backed by the provided runtime.
func NewManager(runtime Runtime) (*Manager, error) {
	if runtime == nil {
		return nil, errors.New("runtime is required")
	}

	return &Manager{runtime: runtime}, nil
}

// Create provisions a new container and returns its metadata.
func (m *Manager) Create(spec ContainerSpec) (*Container, error) {
	if err := validateSpec(spec); err != nil {
		return nil, err
	}

	container, err := m.runtime.CreateContainer(spec)
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}

	return container, nil
}

// Start ensures the container is running.
func (m *Manager) Start(id string) error {
	if id == "" {
		return errors.New("container id is required")
	}

	return m.runtime.StartContainer(id)
}

// Stop stops the running container.
func (m *Manager) Stop(id string) error {
	if id == "" {
		return errors.New("container id is required")
	}

	return m.runtime.StopContainer(id)
}

// Remove deletes the container resources.
func (m *Manager) Remove(id string) error {
	if id == "" {
		return errors.New("container id is required")
	}

	return m.runtime.RemoveContainer(id)
}

// Exec executes a command inside the container and returns its combined output.
func (m *Manager) Exec(id string, cmd []string) (string, error) {
	if id == "" {
		return "", errors.New("container id is required")
	}

	if len(cmd) == 0 {
		return "", errors.New("command is required")
	}

	return m.runtime.Exec(id, cmd)
}

// Shell attaches an interactive shell session to the container.
func (m *Manager) Shell(id string, stdin io.Reader, stdout, stderr io.Writer) error {
	if id == "" {
		return errors.New("container id is required")
	}

	return m.runtime.Shell(id, stdin, stdout, stderr)
}

func validateSpec(spec ContainerSpec) error {
	if strings.TrimSpace(spec.Image) == "" {
		return errors.New("image is required")
	}

	return nil
}
