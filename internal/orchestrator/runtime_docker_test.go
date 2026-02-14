package orchestrator

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// mockDockerClient implements dockerClient for testing.
type mockDockerClient struct {
	capturedConfig   *container.Config
	capturedHostCfg  *container.HostConfig
	capturedNetCfg   *network.NetworkingConfig
	capturedName     string
	createResponse   container.CreateResponse
	inspectResponse  types.ContainerJSON
	imageInspectErr  error
	containerListRes []types.Container
}

func (m *mockDockerClient) ContainerCreate(_ context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, _ *ocispec.Platform, containerName string) (container.CreateResponse, error) {
	m.capturedConfig = config
	m.capturedHostCfg = hostConfig
	m.capturedNetCfg = networkingConfig
	m.capturedName = containerName
	return m.createResponse, nil
}

func (m *mockDockerClient) ContainerInspect(_ context.Context, _ string) (types.ContainerJSON, error) {
	return m.inspectResponse, nil
}

func (m *mockDockerClient) ContainerStart(_ context.Context, _ string, _ container.StartOptions) error {
	return nil
}

func (m *mockDockerClient) ContainerStop(_ context.Context, _ string, _ container.StopOptions) error {
	return nil
}

func (m *mockDockerClient) ContainerRemove(_ context.Context, _ string, _ container.RemoveOptions) error {
	return nil
}

func (m *mockDockerClient) ContainerList(_ context.Context, _ container.ListOptions) ([]types.Container, error) {
	return m.containerListRes, nil
}

func (m *mockDockerClient) ImageInspectWithRaw(_ context.Context, _ string) (types.ImageInspect, []byte, error) {
	return types.ImageInspect{}, nil, m.imageInspectErr
}

func (m *mockDockerClient) ImagePull(_ context.Context, _ string, _ imagetypes.PullOptions) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func (m *mockDockerClient) Close() error {
	return nil
}

func TestCreateContainer_SetsHostnameToContainerName(t *testing.T) {
	tests := []struct {
		name             string
		containerName    string
		expectedHostname string
	}{
		{"simple name", "my-container", "my-container"},
		{"name with dashes", "convoy-worker-1", "convoy-worker-1"},
		{"name with spaces trimmed", "  trimmed  ", "trimmed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockDockerClient{
				createResponse: container.CreateResponse{ID: "test-id-123"},
				inspectResponse: types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						ID:      "test-id-123",
						Created: time.Now().Format(time.RFC3339Nano),
					},
					Config: &container.Config{
						Labels: map[string]string{},
					},
					NetworkSettings: &types.NetworkSettings{},
				},
			}

			runtime := &DockerRuntime{
				client:        mock,
				image:         "alpine:latest",
				agentGRPCPort: 8080,
				pullTimeout:   5 * time.Minute,
			}

			_, err := runtime.CreateContainer(ContainerSpec{Name: tt.containerName})
			if err != nil {
				t.Fatalf("CreateContainer failed: %v", err)
			}

			if mock.capturedConfig.Hostname != tt.expectedHostname {
				t.Errorf("expected hostname %q, got %q", tt.expectedHostname, mock.capturedConfig.Hostname)
			}

			// Also verify the container name passed to Docker matches
			if mock.capturedName != tt.expectedHostname {
				t.Errorf("expected container name %q, got %q", tt.expectedHostname, mock.capturedName)
			}
		})
	}
}
