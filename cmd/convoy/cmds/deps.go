package cmds

import (
	"convoy/internal/app"
	"convoy/internal/orchestrator"
)

// AppProvider provides access to CLI application dependencies.
type AppProvider interface {
	Config() (*app.Config, error)
	Manager() (*orchestrator.Manager, error)
	Registry() *orchestrator.Registry
	Balancer() (*orchestrator.Balancer, error)
}

// GetAppFunc returns the application provider. Set by the main package.
var GetAppFunc func() (AppProvider, error)

// CLIOpts holds CLI options accessible to commands.
var CLIOpts struct {
	ConfigPath string
}

func getApp() (AppProvider, error) {
	if GetAppFunc == nil {
		panic("GetAppFunc not initialized")
	}
	return GetAppFunc()
}
