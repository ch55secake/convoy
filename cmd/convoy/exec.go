package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	convoypb "convoy/api"
	"convoy/internal/orchestrator"

	"github.com/spf13/cobra"
)

func newExecCmd() *cobra.Command {
	var (
		envVars []string
		workDir string
		timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:          "exec [container-id|name] [command] [args...]",
		Short:        "Execute command in container",
		Long:         "Execute a non-interactive command inside a container via the gRPC agent.",
		Args:         cobra.MinimumNArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			containerRef := args[0]
			commandArgs := []string{"sh", "-c", strings.Join(args[1:], " ")}

			app, err := getApp()
			if err != nil {
				return err
			}

			mgr, err := app.Manager()
			if err != nil {
				return err
			}

			containers, err := mgr.List()
			if err != nil {
				return err
			}

			container := resolveContainer(containerRef, containers)
			if container == nil {
				return fmt.Errorf("container not found: %s", containerRef)
			}

			if container.Endpoint == "" {
				return fmt.Errorf("container %s has no gRPC endpoint", containerRef)
			}

			env := parseEnvVars(envVars)

			req := &convoypb.CommandRequest{
				Args:           commandArgs,
				Env:            env,
				WorkDir:        workDir,
				TimeoutSeconds: int32(timeout.Seconds()),
			}

			rpc := orchestrator.NewRPC(orchestrator.RPCConfig{
				DialTimeout: timeout,
				CallTimeout: timeout,
			})
			defer func() {
				_ = rpc.Close()
			}()

			resp, err := rpc.ExecuteCommand(context.Background(), container.Endpoint, req)
			if err != nil {
				return fmt.Errorf("execute command: %w", err)
			}

			if stdout := resp.GetStdout(); stdout != "" {
				_, _ = fmt.Fprint(cmd.OutOrStdout(), stdout)
			}
			if stderr := resp.GetStderr(); stderr != "" {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), stderr)
			}

			if resp.GetErrorMessage() != "" {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "error: %s\n", resp.GetErrorMessage())
			}

			return nil
		},
	}

	cmd.Flags().StringArrayVarP(&envVars, "env", "e", nil, "Set environment variables (can be repeated)")
	cmd.Flags().StringVarP(&workDir, "workdir", "w", "", "Working directory inside the container")
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "Timeout for command execution")

	return cmd
}

// parseEnvVars converts ["KEY=value", ...] to map[string]string.
func parseEnvVars(envVars []string) map[string]string {
	env := make(map[string]string)
	for _, e := range envVars {
		if idx := strings.Index(e, "="); idx > 0 {
			env[e[:idx]] = e[idx+1:]
		}
	}
	return env
}
