package main

import (
	"errors"

	"github.com/spf13/cobra"
)

func newExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "exec [container-id] [command]",
		Short:        "Execute command in container",
		SilenceUsage: true,
		Args:         cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("exec command not implemented")
		},
	}

	return cmd
}
