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

			var lastErr error
			for _, id := range args {
				if err := mgr.Stop(id); err != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "Failed to stop %s: %v\n", id, err)
					lastErr = fmt.Errorf("stop %s: %w", id, err)
					continue
				}

				fmt.Fprintf(cmd.OutOrStdout(), "Stopped %s\n", id)
			}

			return lastErr
		},
	}

	return cmd
}
