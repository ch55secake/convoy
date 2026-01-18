package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop [container-id]",
		Short: "Stop containers",
		Args:  cobra.MinimumNArgs(1),
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

			var lastErr error
			for _, id := range args {
				containerID := id
				if existing, ok := registry.Get(id); ok {
					containerID = existing.ID
				}

				if err := mgr.Stop(containerID); err != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "Failed to stop %s: %v\n", containerID, err)
					lastErr = fmt.Errorf("stop %s: %w", containerID, err)
					continue
				}

				if removeErr := mgr.Remove(containerID); removeErr != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "Failed to remove %s: %v\n", containerID, removeErr)
					lastErr = fmt.Errorf("remove %s: %w", containerID, removeErr)
					continue
				}

				registry.Remove(containerID)
				fmt.Fprintf(cmd.OutOrStdout(), "Stopped and removed %s\n", containerID)
			}

			return lastErr
		},
	}

	return cmd
}
