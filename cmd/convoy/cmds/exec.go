package cmds

import (
	"context"
	"fmt"
	"strings"
	"time"

	convoypb "convoy/api"

	"github.com/spf13/cobra"
)

// NewExecCmd creates the exec command for running commands inside containers.
func NewExecCmd() *cobra.Command {
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

			containers, err := LoadContainers()
			if err != nil {
				return err
			}

			container, err := containers.ResolveWithEndpoint(containerRef)
			if err != nil {
				return err
			}

			env := ParseEnvVars(envVars)

			req := &convoypb.CommandRequest{
				Args:           commandArgs,
				Env:            env,
				WorkDir:        workDir,
				TimeoutSeconds: int32(timeout.Seconds()),
			}

			rpc := NewRPCClientWithTimeout(timeout)
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
