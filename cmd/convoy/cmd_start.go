package main

import (
	"fmt"

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
				if err := mgr.Start(id); err == nil {
					fmt.Fprintf(cmd.OutOrStdout(), "Started %s\n", id)
					continue
				}

				fmt.Fprintf(cmd.OutOrStdout(), "Starting %s failed, creating container...\n", id)

				spec := orchestrator.ContainerSpec{
					Image: cfg.Image,
					Labels: map[string]string{
						"convoy.cli.name": id,
					},
				}
				container, createErr := mgr.Create(spec)
				if createErr != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "Failed to create container for %s: %v\n", id, createErr)
					lastErr = fmt.Errorf("create %s: %w", id, createErr)
					continue
				}

				if regErr := registry.Register(container); regErr != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "Warning: failed to register %s: %v\n", container.ID, regErr)
				}

				if err := mgr.Start(container.ID); err != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "Failed to start new container %s: %v\n", container.ID, err)
					lastErr = fmt.Errorf("start new %s: %w", id, err)
					continue
				}

				fmt.Fprintf(cmd.OutOrStdout(), "Created and started %s (id=%s)\n", id, container.ID)
			}

			return lastErr
		},
	}

	return cmd
}
