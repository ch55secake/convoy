package cmds

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"convoy/internal/orchestrator"
)

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

			var lastErr error
			for _, arg := range args {
				containerID := arg
				containerName := strings.TrimSpace(arg)
				if containerName == "" {
					continue
				}

				_, err := mgr.List()
				if err != nil {
					return err
				}
				if existing, ok := registry.GetByName(containerName); ok {
					containerID = existing.ID
				} else if existing, ok := registry.Get(arg); ok {
					containerID = existing.ID
					containerName = strings.TrimSpace(existing.Name)
				} else {
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
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created container %s (id=%s)\n", containerName, container.ID)
				}

				if err := mgr.Start(containerID); err != nil {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Failed to start %s: %v\n", containerID, err)
					lastErr = fmt.Errorf("start %s: %w", containerID, err)
					continue
				}

				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Started %s\n", containerID)
			}

			return lastErr
		},
	}

	return cmd
}
