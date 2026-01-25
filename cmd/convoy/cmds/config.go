package cmds

import (
	"fmt"

	"convoy/internal/app"

	"github.com/spf13/cobra"
)

func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "config",
		Short:        "Show, validate or initialize configuration",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := getApp()
			if err != nil {
				return err
			}

			cfg, err := application.Config()
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Image: %s\n", cfg.Image)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "gRPC Port: %d\n", cfg.GRPCPort)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Docker Host: %s\n", cfg.DockerHost)
			return nil
		},
	}

	cmd.AddCommand(newConfigInitCmd())

	return cmd
}

func newConfigInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create the default configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath := CLIOpts.ConfigPath
			if cfgPath == "" {
				var err error
				cfgPath, err = app.DefaultConfigPath()
				if err != nil {
					return err
				}
			}

			createdPath, err := app.InitializeConfig(cfgPath)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Wrote default config to %s\n", createdPath)
			return nil
		},
	}

	return cmd
}
