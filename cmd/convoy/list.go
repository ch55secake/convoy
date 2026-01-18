package main

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List containers",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp()
			if err != nil {
				return err
			}

			mgr, err := app.Manager()
			if err != nil {
				return err
			}

			containers, err := mgr.List()
			if err != nil {
				return err
			}

			if len(containers) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No containers registered")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintf(w, "ID\tNAME\tIMAGE\tENDPOINT\n")
			for _, c := range containers {
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", c.ID, c.Name, c.Image, c.Endpoint)
			}

			return w.Flush()
		},
	}

	return cmd
}
