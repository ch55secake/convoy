package main

import (
	"sync"

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

	rootCmd.AddCommand(
		newConfigCmd(),
		newCreateCmd(),
		newListCmd(),
		newStartCmd(),
		newStopCmd(),
		newRemoveCmd(),
		newExecCmd(),
		newShellCmd(),
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

func setApplication(app *Application) {
	appInstance = app
	appInitErr = nil
}
