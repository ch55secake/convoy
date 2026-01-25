package cmds

import (
	"errors"

	"github.com/spf13/cobra"
)

// NewShellCmd creates the shell command for opening interactive shells in containers.
func NewShellCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "shell [container-id]",
		Short:        "Open an interactive shell",
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, _ []string) error {
			return errors.New("shell command not implemented")
		},
	}

	return cmd
}
