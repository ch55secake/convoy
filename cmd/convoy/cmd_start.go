package main

import (
	"fmt"

	"github.com/spf13/cobra"
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

			mgr, err := app.Manager()
			if err != nil {
				return err
			}

			var lastErr error
			for _, id := range args {
				if err := mgr.Start(id); err != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "Failed to start %s: %v\n", id, err)
					lastErr = fmt.Errorf("start %s: %w", id, err)
					continue
				}

				fmt.Fprintf(cmd.OutOrStdout(), "Started %s\n", id)
			}

			return lastErr
		},
	}

	return cmd
}
