package cmds

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewStopCmd creates the stop command for stopping containers.
func NewStopCmd() *cobra.Command {
	var stopAll bool

	cmd := &cobra.Command{
		Use:          "stop [container-id]",
		Short:        "Stop containers",
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp()
			if err != nil {
				return err
			}

			mgr, err := app.Manager()
			if err != nil {
				return err
			}

			registry := app.Registry()

			containers, err := LoadContainers()
			if err != nil {
				return fmt.Errorf("list containers: %w", err)
			}

			// Sync containers to registry
			for _, container := range containers.List() {
				_ = registry.Register(container)
			}

			var targetIDs []string
			switch {
			case stopAll:
				targetIDs = containers.AllContainerIDs()
				if len(targetIDs) == 0 {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No containers registered")
					return nil
				}
			case len(args) == 0:
				return fmt.Errorf("provide container names or IDs, or use -a")
			default:
				resolved, missing := containers.ResolveContainerIDs(args)
				for _, m := range missing {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Container not found: %s\n", m)
				}
				targetIDs = resolved
			}

			var lastErr error
			for _, containerID := range targetIDs {
				container := containers.Resolve(containerID)
				label := containerID
				if container != nil {
					label = ContainerLabel(container)
				}

				if err := mgr.Stop(containerID); err != nil {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Failed to stop %s: %v\n", label, err)
					lastErr = fmt.Errorf("stop %s: %w", label, err)
					continue
				}

				if removeErr := mgr.Remove(containerID); removeErr != nil {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Failed to remove %s: %v\n", label, removeErr)
					lastErr = fmt.Errorf("remove %s: %w", label, removeErr)
					continue
				}

				registry.Remove(containerID)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Stopped and removed %s\n", label)
			}

			return lastErr
		},
	}

	cmd.Flags().BoolVarP(&stopAll, "all", "a", false, "Stop and remove all managed containers")

	return cmd
}
