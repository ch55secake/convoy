package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"convoy/internal/orchestrator"
)

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start [container-id]",
		Short: "Start containers",
		Args:  cobra.MinimumNArgs(1),
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
			for _, id := range args {
				containerID := id
				containerName := strings.TrimSpace(id)
				if containerName == "" {
					continue
				}

				if existing, ok := registry.GetByName(containerName); ok {
					containerID = existing.ID
				} else if existing, ok := registry.Get(id); ok {
					containerID = existing.ID
					containerName = strings.TrimSpace(existing.Name)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "No registered container %s; creating new container...\n", id)
					spec := orchestrator.ContainerSpec{
						Name:  containerName,
						Image: cfg.Image,
					}

					container, createErr := mgr.Create(spec)
					if createErr != nil {
						fmt.Fprintf(cmd.OutOrStdout(), "Failed to create container %s: %v\n", id, createErr)
						lastErr = fmt.Errorf("create %s: %w", id, createErr)
						continue
					}

					if regErr := registry.Register(container); regErr != nil {
						fmt.Fprintf(cmd.OutOrStdout(), "Warning: failed to register %s: %v\n", container.ID, regErr)
					}

					containerID = container.ID
					fmt.Fprintf(cmd.OutOrStdout(), "Created container %s (id=%s)\n", containerName, container.ID)
				}

				if err := mgr.Start(containerID); err != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "Failed to start %s: %v\n", containerID, err)
					lastErr = fmt.Errorf("start %s: %w", containerID, err)
					continue
				}

				fmt.Fprintf(cmd.OutOrStdout(), "Started %s\n", containerID)
			}

			return lastErr
		},
	}

	return cmd
}
