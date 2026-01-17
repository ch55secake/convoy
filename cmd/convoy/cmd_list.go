package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List containers",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp()
			if err != nil {
				return err
			}

			registry := app.Registry()
			containers := registry.List()
			if len(containers) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No containers registered")
				return nil
			}

			for _, c := range containers {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", c.ID, c.Image, c.Endpoint)
			}

			return nil
		},
	}

	return cmd
}
