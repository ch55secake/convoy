package main

import (
	"sync"

	"convoy/cmd/convoy/cmds"

	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:     "convoy",
		Short:   "Manage multiple containers and tasks",
		Long:    `A CLI tool to orchestrate containers via Docker and RPC.`,
		Aliases: []string{"cvy"},
	}

	cliOpts struct {
		configPath string
	}

	runtimeFactory RuntimeFactory = dockerRuntimeFactory

	appOnce     sync.Once
	appInstance *Application
	appInitErr  error
)

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cliOpts.configPath, "config", "", "Path to config file (defaults to ~/.config/convoy/config.yaml)")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if shouldSkipAppInit(cmd) {
			return nil
		}

		return initializeApplication()
	}

	// Wire up the cmds package with the application provider
	cmds.GetAppFunc = func() (cmds.AppProvider, error) {
		// Sync CLI options to cmds package
		cmds.CLIOpts.ConfigPath = cliOpts.configPath
		return getApp()
	}

	rootCmd.AddCommand(
		cmds.NewConfigCmd(),
		cmds.NewListCmd(),
		cmds.NewHealthCmd(),
		cmds.NewStartCmd(),
		cmds.NewStopCmd(),
		cmds.NewRemoveCmd(),
		cmds.NewExecCmd(),
		cmds.NewShellCmd(),
		cmds.NewCopyCmd(),
	)
}

func shouldSkipAppInit(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}

	if cmd.Name() == "init" && cmd.HasParent() && cmd.Parent().Name() == "config" {
		return true
	}

	return false
}

func initializeApplication() error {
	if appInstance != nil {
		return nil
	}

	appOnce.Do(func() {
		appInstance = newApplication(cliOpts.configPath, runtimeFactory)
		_, appInitErr = appInstance.Config()
	})

	return appInitErr
}

func getApp() (*Application, error) {
	if err := initializeApplication(); err != nil {
		return nil, err
	}

	return appInstance, nil
}
