package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new container",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp()
			if err != nil {
				return err
			}

			cfg, err := app.Config()
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Would create container with image %s (not yet implemented)\n", cfg.Image)
			return nil
		},
	}

	return cmd
}
