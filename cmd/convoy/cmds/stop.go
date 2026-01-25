package cmds

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func NewStopCmd() *cobra.Command {
	var stopAll bool

	cmd := &cobra.Command{
		Use:          "stop [container-id]",
		Short:        "Stop containers",
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp()
			if err != nil {
				return err
			}

			mgr, err := app.Manager()
			if err != nil {
				return err
			}

			registry := app.Registry()

			managed, listErr := mgr.List()
			if listErr != nil {
				return fmt.Errorf("list containers: %w", listErr)
			}
			for _, container := range managed {
				_ = registry.Register(container)
			}

			targetIDs := args
			if stopAll {
				containers := registry.List()
				if len(containers) == 0 {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No containers registered")
					return nil
				}

				targetIDs = make([]string, 0, len(containers))
				for _, c := range containers {
					if c == nil {
						continue
					}
					targetIDs = append(targetIDs, c.ID)
				}
			} else if len(targetIDs) == 0 {
				return fmt.Errorf("provide container names or IDs, or use -a")
			}

			resolve := func(input string) (string, string) {
				trimmed := strings.TrimSpace(input)
				if trimmed == "" {
					return "", ""
				}

				if c, ok := registry.GetByName(trimmed); ok {
					return c.ID, c.Name
				}

				if c, ok := registry.Get(trimmed); ok {
					return c.ID, c.Name
				}

				return trimmed, trimmed
			}

			var lastErr error
			for _, target := range targetIDs {
				containerID, containerName := resolve(target)
				if containerID == "" {
					continue
				}

				label := containerName
				if label == "" {
					label = containerID
				}

				if err := mgr.Stop(containerID); err != nil {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Failed to stop %s: %v\n", label, err)
					lastErr = fmt.Errorf("stop %s: %w", label, err)
					continue
				}

				if removeErr := mgr.Remove(containerID); removeErr != nil {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Failed to remove %s: %v\n", label, removeErr)
					lastErr = fmt.Errorf("remove %s: %w", label, removeErr)
					continue
				}

				registry.Remove(containerID)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Stopped and removed %s\n", label)
			}

			return lastErr
		},
	}

	cmd.Flags().BoolVarP(&stopAll, "all", "a", false, "Stop and remove all managed containers")

	return cmd
}
