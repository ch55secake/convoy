package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "convoy",
	Short:   "Manage multiple containers and tasks",
	Long:    `A CLI tool to orchestrate containers via Docker and RPC.`,
	Aliases: []string{"cvy"},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add subcommands here
}
