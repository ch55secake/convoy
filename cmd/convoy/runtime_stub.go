package main

import (
	"fmt"

	"convoy/internal/app"
	"convoy/internal/orchestrator"
)

// dockerRuntimeFactory will connect to Docker in a future implementation.
func dockerRuntimeFactory(cfg *app.Config) (orchestrator.Runtime, error) {
	return nil, fmt.Errorf("docker runtime not yet implemented")
}
