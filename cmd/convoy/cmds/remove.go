package cmds

import (
	"errors"

	"github.com/spf13/cobra"
)

func NewRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "remove [container-id]",
		Short:        "Remove containers",
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("remove command not implemented")
		},
	}

	return cmd
}
