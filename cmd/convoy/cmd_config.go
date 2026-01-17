package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show or validate configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp()
			if err != nil {
				return err
			}

			cfg, err := app.Config()
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Image: %s\n", cfg.Image)
			fmt.Fprintf(cmd.OutOrStdout(), "gRPC Port: %d\n", cfg.GRPCPort)
			fmt.Fprintf(cmd.OutOrStdout(), "Docker Host: %s\n", cfg.DockerHost)
			return nil
		},
	}

	return cmd
}
