package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	convoypb "convoy/api"
	"convoy/internal/orchestrator"

	"github.com/spf13/cobra"
)

type healthTarget struct {
	Label     string
	Endpoint  string
	Container *orchestrator.Container
}

func newHealthCmd() *cobra.Command {
	var checkAll bool
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:           "health [container-id|name]...",
		Short:         "Check container agent health",
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp()
			if err != nil {
				return err
			}

			mgr, err := app.Manager()
			if err != nil {
				return err
			}

			containers, err := mgr.List()
			if err != nil {
				return err
			}

			writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			defer func() {
				_ = writer.Flush()
			}()
			_, _ = fmt.Fprintln(writer, "NAME\tSTATUS")

			if checkAll {
				return runHealthChecks(writer, containers, timeout)
			}

			if len(args) == 0 {
				return errors.New("container id or name is required")
			}

			targets, missing := resolveHealthTargets(args, containers)
			for _, miss := range missing {
				_, _ = fmt.Fprintf(writer, "%s\tunhealthy: container not found\n", miss)
			}

			failed := len(missing) > 0
			if len(targets) == 0 {
				if failed {
					return errors.New("one or more containers unhealthy")
				}
				return errors.New("no matching containers found")
			}

			if err := checkTargets(writer, targets, timeout); err != nil {
				failed = true
			}

			if failed {
				return errors.New("one or more containers unhealthy")
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&checkAll, "all", "a", false, "Check all containers")
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Second, "Timeout for health checks")

	return cmd
}

func runHealthChecks(writer io.Writer, containers []*orchestrator.Container, timeout time.Duration) error {
	if len(containers) == 0 {
		_, _ = fmt.Fprintln(writer, "all\tunhealthy: no containers registered")
		return errors.New("no containers registered")
	}

	targets := make([]healthTarget, 0, len(containers))
	for _, container := range containers {
		if container == nil {
			continue
		}
		label := container.Name
		if label == "" {
			label = container.ID
		}
		targets = append(targets, healthTarget{
			Label:     label,
			Endpoint:  container.Endpoint,
			Container: container,
		})
	}

	if len(targets) == 0 {
		_, _ = fmt.Fprintln(writer, "all\tunhealthy: no containers registered")
		return errors.New("no containers registered")
	}

	if err := checkTargets(writer, targets, timeout); err != nil {
		return errors.New("one or more containers unhealthy")
	}

	return nil
}

func resolveHealthTargets(args []string, containers []*orchestrator.Container) ([]healthTarget, []string) {
	byID := make(map[string]*orchestrator.Container)
	byName := make(map[string]*orchestrator.Container)
	for _, container := range containers {
		if container == nil || container.ID == "" {
			continue
		}
		byID[container.ID] = container
		if container.Name != "" {
			byName[container.Name] = container
		}
	}

	var targets []healthTarget
	var missing []string
	for _, arg := range args {
		if arg == "" {
			continue
		}
		container := byName[arg]
		if container == nil {
			container = byID[arg]
		}
		if container == nil {
			missing = append(missing, arg)
			continue
		}
		label := container.Name
		if label == "" {
			label = container.ID
		}
		targets = append(targets, healthTarget{
			Label:     label,
			Endpoint:  container.Endpoint,
			Container: container,
		})
	}

	return targets, missing
}

func checkTargets(writer io.Writer, targets []healthTarget, timeout time.Duration) error {
	rpc := orchestrator.NewRPC(orchestrator.RPCConfig{DialTimeout: timeout, CallTimeout: timeout})
	defer func() {
		_ = rpc.Close()
	}()

	failed := false
	for _, target := range targets {
		if target.Endpoint == "" {
			_, _ = fmt.Fprintf(writer, "%s\tunhealthy: missing endpoint\n", target.Label)
			failed = true
			continue
		}

		resp, err := rpc.CheckHealth(context.Background(), target.Endpoint, &convoypb.HealthRequest{})
		if err != nil {
			_, _ = fmt.Fprintf(writer, "%s\tunhealthy: %v\n", target.Label, err)
			failed = true
			continue
		}

		if resp.GetStatus() != convoypb.HealthResponse_STATUS_HEALTHY {
			msg := resp.GetMessage()
			if msg == "" {
				msg = resp.GetStatus().String()
			}
			_, _ = fmt.Fprintf(writer, "%s\tunhealthy: %s\n", target.Label, msg)
			failed = true
			continue
		}

		_, _ = fmt.Fprintf(writer, "%s\thealthy\n", target.Label)
	}

	if failed {
		return errors.New("one or more containers unhealthy")
	}
	return nil
}
