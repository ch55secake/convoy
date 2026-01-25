package cmds

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// NewListCmd creates the list command for displaying registered containers.
func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List containers",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			containers, err := LoadContainers()
			if err != nil {
				return err
			}

			list := containers.List()
			if len(list) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No containers registered")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintf(w, "ID\tNAME\tIMAGE\tENDPOINT\n")
			for _, c := range list {
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", c.ID, c.Name, c.Image, c.Endpoint)
			}

			return w.Flush()
		},
	}

	return cmd
}
