package main

import (
	"fmt"
	"sync"

	"convoy/internal/app"
	"convoy/internal/orchestrator"
	"convoy/pkg/loadbalancer"
)

// Application wires together config, registry, manager, and balancer for CLI commands.
type Application struct {
	cfgPath string

	configMu sync.Mutex
	config   *app.Config

	registryOnce sync.Once
	registry     *orchestrator.Registry

	managerOnce sync.Once
	manager     *orchestrator.Manager

	balancerOnce sync.Once
	balancer     *orchestrator.Balancer

	runtimeFactory RuntimeFactory
}

// RuntimeFactory defines how to create a Runtime for orchestrator.Manager.
type RuntimeFactory func(cfg *app.Config) (orchestrator.Runtime, error)

func newApplication(cfgPath string, factory RuntimeFactory) *Application {
	if factory == nil {
		factory = noopRuntimeFactory
	}
	return &Application{cfgPath: cfgPath, runtimeFactory: factory}
}

func (a *Application) Config() (*app.Config, error) {
	a.configMu.Lock()
	defer a.configMu.Unlock()

	if a.config != nil {
		return a.config, nil
	}

	cfg, err := app.LoadConfig(a.cfgPath)
	if err != nil {
		return nil, err
	}

	a.config = cfg
	return a.config, nil
}

func (a *Application) Registry() *orchestrator.Registry {
	a.registryOnce.Do(func() {
		a.registry = orchestrator.NewRegistry()
	})

	return a.registry
}

func (a *Application) Manager() (*orchestrator.Manager, error) {
	var err error
	a.managerOnce.Do(func() {
		cfg, cfgErr := a.Config()
		if cfgErr != nil {
			err = cfgErr
			return
		}

		runtime, runtimeErr := a.runtimeFactory(cfg)
		if runtimeErr != nil {
			err = runtimeErr
			return
		}

		var mgrErr error
		a.manager, mgrErr = orchestrator.NewManager(runtime)
		if mgrErr != nil {
			err = mgrErr
		}
	})

	if err != nil {
		return nil, err
	}

	return a.manager, nil
}

func (a *Application) Balancer() (*orchestrator.Balancer, error) {
	var err error
	a.balancerOnce.Do(func() {
		lb := loadbalancer.NewRoundRobin()
		var balancerErr error
		a.balancer, balancerErr = orchestrator.NewBalancer(lb)
		if balancerErr != nil {
			err = balancerErr
		}
	})

	if err != nil {
		return nil, err
	}

	return a.balancer, nil
}

func noopRuntimeFactory(cfg *app.Config) (orchestrator.Runtime, error) {
	return nil, fmt.Errorf("runtime factory not implemented")
}
