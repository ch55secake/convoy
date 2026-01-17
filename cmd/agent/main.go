package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"convoy/internal/agent"
)

func main() {
	cfg, err := agent.LoadConfig("")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	srv := agent.NewServer(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := srv.Start(ctx); err != nil {
		log.Fatalf("agent failed: %v", err)
	}
}
