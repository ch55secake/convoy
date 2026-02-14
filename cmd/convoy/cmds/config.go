package cmds

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"convoy/internal/app"

	"github.com/spf13/cobra"
)

// NewConfigCmd creates the config command for displaying and managing configuration.
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "config",
		Short:        "Show, validate or initialize configuration",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
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

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Git Credentials Enabled: %v\n", cfg.GitCredentials.Enabled)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Mount SSH: %v\n", cfg.GitCredentials.MountSSH)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Mount Git Config: %v\n", cfg.GitCredentials.MountGitconfig)
			return nil
		},
	}

	cmd.AddCommand(newConfigInitCmd())

	return cmd
}

func newConfigInitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create the default configuration file",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfgPath := CLIOpts.ConfigPath
			if cfgPath == "" {
				var err error
				cfgPath, err = app.DefaultConfigPath()
				if err != nil {
					return err
				}
			}

			// Check if config exists for interactive prompt
			if _, err := os.Stat(cfgPath); err == nil && !force {
				// Config exists and no force flag - prompt user
				reader := bufio.NewReader(cmd.InOrStdin())
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Config already exists at %s. Overwrite? (y/N): ", cfgPath)

				input, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("read user input: %w", err)
				}

				input = strings.TrimSpace(strings.ToLower(input))
				if input != "y" && input != "yes" {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Aborted. No changes made.")
					return nil
				}

				// User confirmed - set force to true
				force = true
			}

			result, err := app.InitializeConfig(cfgPath, force)
			if err != nil {
				return err
			}

			if result.Overwritten {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Overwrote config at %s (backup saved to %s)\n", result.Path, result.BackupPath)
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Wrote default config to %s\n", result.Path)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing configuration file without prompting")

	return cmd
}
