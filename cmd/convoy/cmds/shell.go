package cmds

import (
	"errors"

	"github.com/spf13/cobra"
)

func NewShellCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "shell [container-id]",
		Short:        "Open an interactive shell",
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("shell command not implemented")
		},
	}

	return cmd
}
