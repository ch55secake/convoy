package main

import (
	"errors"

	"github.com/spf13/cobra"
)

func newShellCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shell [container-id]",
		Short: "Open an interactive shell",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("shell command not implemented")
		},
	}

	return cmd
}
