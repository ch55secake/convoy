package cmds

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"convoy/internal/orchestrator"
)

// NewStartCmd creates the start command for starting containers.
func NewStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "start [container-id]",
		Short:        "Start containers",
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp()
			if err != nil {
				return err
			}

			cfg, err := app.Config()
			if err != nil {
				return err
			}

			mgr, err := app.Manager()
			if err != nil {
				return err
			}

			registry := app.Registry()

			// Load existing containers for resolution
			containers, err := LoadContainers()
			if err != nil {
				return err
			}

			var lastErr error
			for _, arg := range args {
				containerName := strings.TrimSpace(arg)
				if containerName == "" {
					continue
				}

				// Try to resolve existing container
				var containerID string
				var displayLabel string
				if existing := containers.Resolve(containerName); existing != nil {
					containerID = existing.ID
					displayLabel = ContainerLabel(existing)
				} else {
					// Create new container
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No registered container: %s\nCreating new container...\n", arg)
					spec := orchestrator.ContainerSpec{
						Name:  containerName,
						Image: cfg.Image,
					}

					container, createErr := mgr.Create(spec)
					if createErr != nil {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Failed to create container %s: %v\n", arg, createErr)
						lastErr = fmt.Errorf("create %s: %w", arg, createErr)
						continue
					}

					if regErr := registry.Register(container); regErr != nil {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Warning: failed to register %s: %v\n", container.ID, regErr)
					}

					containerID = container.ID
					displayLabel = containerName
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created container %s (id=%s)\n", containerName, container.ID)
				}

				if err := mgr.Start(containerID); err != nil {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Failed to start %s: %v\n", displayLabel, err)
					lastErr = fmt.Errorf("start %s: %w", displayLabel, err)
					continue
				}

				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Started %s\n", displayLabel)
			}

			return lastErr
		},
	}

	return cmd
}
