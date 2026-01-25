package cmds

import (
	"errors"

	"github.com/spf13/cobra"
)

// NewRemoveCmd creates the remove command for removing containers.
func NewRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "remove [container-id]",
		Short:        "Remove containers",
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return errors.New("remove command not implemented")
		},
	}

	return cmd
}
