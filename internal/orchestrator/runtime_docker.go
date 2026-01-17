package orchestrator

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"convoy/internal/app"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

const defaultShell = "/bin/sh"

// DockerRuntime implements Runtime using the Docker Engine API.
type DockerRuntime struct {
	client        *client.Client
	image         string
	agentGRPCPort int
	network       string
	pullAlways    bool
	pullTimeout   time.Duration
}

// NewDockerRuntime constructs a Docker-backed runtime.
func NewDockerRuntime(cfg *app.Config) (*DockerRuntime, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost(cfg.DockerHost),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}

	pullTimeout := time.Duration(cfg.PullTimeoutSec) * time.Second
	if pullTimeout <= 0 {
		pullTimeout = 5 * time.Minute
	}

	return &DockerRuntime{
		client:        cli,
		image:         cfg.Image,
		agentGRPCPort: cfg.AgentGRPCPort,
		network:       cfg.DockerNetwork,
		pullAlways:    cfg.PullAlways,
		pullTimeout:   pullTimeout,
	}, nil
}

// CreateContainer creates a new container for the Convoy agent.
func (d *DockerRuntime) CreateContainer(spec ContainerSpec) (*Container, error) {
	image := strings.TrimSpace(spec.Image)
	if image == "" {
		image = strings.TrimSpace(d.image)
	}
	if image == "" {
		return nil, errors.New("image is required")
	}

	labels := copyStringMap(spec.Labels)
	envVars := mapToEnv(spec.Environment)
	ctx, cancel := context.WithTimeout(context.Background(), d.pullTimeout)
	defer cancel()

	if err := d.ensureImage(ctx, image); err != nil {
		return nil, fmt.Errorf("ensure image %s: %w", image, err)
	}

	portKey := nat.Port(fmt.Sprintf("%d/tcp", d.agentGRPCPort))
	containerConfig := &container.Config{
		Image:        image,
		Labels:       labels,
		Env:          envVars,
		ExposedPorts: nat.PortSet{portKey: struct{}{}},
	}
	if len(spec.Command) > 0 {
		containerConfig.Cmd = spec.Command
	}

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			portKey: {{HostIP: "", HostPort: ""}},
		},
	}

	var networkingConfig *network.NetworkingConfig
	if strings.TrimSpace(d.network) != "" {
		networkingConfig = &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				d.network: {},
			},
		}
	}

	resp, err := d.client.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, nil, "")
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}

	inspect, err := d.client.ContainerInspect(ctx, resp.ID)
	if err != nil {
		return nil, fmt.Errorf("inspect container %s: %w", resp.ID, err)
	}

	createdAt, _ := time.Parse(time.RFC3339Nano, inspect.Created)
	endpoint := deriveEndpoint(inspect, portKey, d.network, d.agentGRPCPort)

	return &Container{
		ID:        resp.ID,
		Image:     image,
		Endpoint:  endpoint,
		Labels:    labels,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}, nil
}

// StartContainer starts the container by ID.
func (d *DockerRuntime) StartContainer(id string) error {
	ctx := context.Background()
	if err := d.client.ContainerStart(ctx, id, container.StartOptions{}); err != nil {
		return fmt.Errorf("start container %s: %w", id, err)
	}
	return nil
}

// StopContainer stops the container by ID.
func (d *DockerRuntime) StopContainer(id string) error {
	ctx := context.Background()
	timeoutSec := 10
	if err := d.client.ContainerStop(ctx, id, container.StopOptions{Timeout: &timeoutSec}); err != nil {
		return fmt.Errorf("stop container %s: %w", id, err)
	}
	return nil
}

// RemoveContainer removes the container and associated resources.
func (d *DockerRuntime) RemoveContainer(id string) error {
	ctx := context.Background()
	opts := container.RemoveOptions{RemoveVolumes: true, Force: true}
	if err := d.client.ContainerRemove(ctx, id, opts); err != nil {
		return fmt.Errorf("remove container %s: %w", id, err)
	}
	return nil
}

// Exec runs a command in the container and returns its combined output.
func (d *DockerRuntime) Exec(id string, cmd []string) (string, error) {
	if len(cmd) == 0 {
		return "", errors.New("command is required")
	}

	ctx := context.Background()
	execConfig := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	resp, err := d.client.ContainerExecCreate(ctx, id, execConfig)
	if err != nil {
		return "", fmt.Errorf("exec create: %w", err)
	}

	attach, err := d.client.ContainerExecAttach(ctx, resp.ID, container.ExecStartOptions{})
	if err != nil {
		return "", fmt.Errorf("exec attach: %w", err)
	}
	defer attach.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdoutBuf, &stderrBuf, attach.Reader); err != nil {
		return "", fmt.Errorf("exec copy: %w", err)
	}

	inspect, err := d.client.ContainerExecInspect(ctx, resp.ID)
	if err != nil {
		return "", fmt.Errorf("exec inspect: %w", err)
	}

	output := stdoutBuf.String() + stderrBuf.String()
	if inspect.ExitCode != 0 {
		return output, fmt.Errorf("exec exit %d", inspect.ExitCode)
	}

	return output, nil
}

// Shell attaches an interactive shell session.
func (d *DockerRuntime) Shell(id string, stdin io.Reader, stdout, stderr io.Writer) error {
	ctx := context.Background()
	execConfig := container.ExecOptions{
		Cmd:          []string{defaultShell},
		AttachStdin:  stdin != nil,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
	}

	resp, err := d.client.ContainerExecCreate(ctx, id, execConfig)
	if err != nil {
		return fmt.Errorf("shell exec create: %w", err)
	}

	attach, err := d.client.ContainerExecAttach(ctx, resp.ID, container.ExecStartOptions{})
	if err != nil {
		return fmt.Errorf("shell exec attach: %w", err)
	}
	defer attach.Close()

	stdoutWriter := stdout
	if stdoutWriter == nil {
		stdoutWriter = io.Discard
	}
	stderrWriter := stderr
	if stderrWriter == nil {
		stderrWriter = io.Discard
	}

	stdinDone := make(chan error, 1)
	if stdin != nil {
		go func() {
			_, copyErr := io.Copy(attach.Conn, stdin)
			err := attach.CloseWrite()
			if err != nil {
				return
			}
			stdinDone <- copyErr
		}()
	} else {
		err := attach.CloseWrite()
		if err != nil {
			return err
		}
	}

	if _, err := stdcopy.StdCopy(stdoutWriter, stderrWriter, attach.Reader); err != nil {
		return fmt.Errorf("shell copy: %w", err)
	}

	if stdin != nil {
		if copyErr := <-stdinDone; copyErr != nil && !errors.Is(copyErr, io.EOF) {
			return fmt.Errorf("shell stdin: %w", copyErr)
		}
	}

	inspect, err := d.client.ContainerExecInspect(ctx, resp.ID)
	if err != nil {
		return fmt.Errorf("shell exec inspect: %w", err)
	}

	if inspect.ExitCode != 0 {
		return fmt.Errorf("shell exited with %d", inspect.ExitCode)
	}

	return nil
}

// Close cleans up Docker client resources.
func (d *DockerRuntime) Close() error {
	return d.client.Close()
}

func (d *DockerRuntime) ensureImage(ctx context.Context, image string) error {
	if !d.pullAlways {
		if _, _, err := d.client.ImageInspectWithRaw(ctx, image); err == nil {
			return nil
		}
	}

	reader, err := d.client.ImagePull(ctx, image, imagetypes.PullOptions{})
	if err != nil {
		return err
	}
	defer func(reader io.ReadCloser) {
		err := reader.Close()
		if err != nil {
			return
		}
	}(reader)
	_, _ = io.Copy(io.Discard, reader)
	return nil
}

func mapToEnv(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}

	result := make([]string, 0, len(env))
	for k, v := range env {
		if strings.TrimSpace(k) == "" {
			continue
		}
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}

	return result
}

func copyStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}

	out := make(map[string]string, len(input))
	for k, v := range input {
		out[k] = v
	}

	return out
}

func deriveEndpoint(inspect types.ContainerJSON, port nat.Port, preferredNetwork string, agentPort int) string {
	if inspect.NetworkSettings == nil {
		return ""
	}

	if bindings, ok := inspect.NetworkSettings.Ports[port]; ok {
		for _, binding := range bindings {
			if binding.HostPort == "" {
				continue
			}
			host := binding.HostIP
			if host == "" || host == "0.0.0.0" {
				host = "127.0.0.1"
			}
			return net.JoinHostPort(host, binding.HostPort)
		}
	}

	if preferredNetwork != "" {
		if netConf, ok := inspect.NetworkSettings.Networks[preferredNetwork]; ok {
			if ip := strings.TrimSpace(netConf.IPAddress); ip != "" {
				return fmt.Sprintf("%s:%d", ip, agentPort)
			}
		}
	}

	if ip := strings.TrimSpace(inspect.NetworkSettings.IPAddress); ip != "" {
		return fmt.Sprintf("%s:%d", ip, agentPort)
	}

	return ""
}
