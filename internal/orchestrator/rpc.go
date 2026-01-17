package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	convoypb "convoy/api"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// RPCConfig configures the RPC client behavior.
type RPCConfig struct {
	DialTimeout time.Duration
	CallTimeout time.Duration
}

// RPC handles gRPC communication with containers.
type RPC struct {
	cfg      RPCConfig
	mu       sync.Mutex
	conns    map[string]*grpc.ClientConn
	dialOpts []grpc.DialOption
}

// NewRPC creates a new RPC helper with sensible defaults.
func NewRPC(cfg RPCConfig) *RPC {
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 5 * time.Second
	}
	if cfg.CallTimeout <= 0 {
		cfg.CallTimeout = 30 * time.Second
	}

	return &RPC{
		cfg:   cfg,
		conns: make(map[string]*grpc.ClientConn),
		dialOpts: []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		},
	}
}

// Close shuts down all open connections.
func (r *RPC) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var firstErr error
	for endpoint, conn := range r.conns {
		if err := conn.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		delete(r.conns, endpoint)
	}

	return firstErr
}

// ExecuteCommand calls ExecuteCommand on the target endpoint.
func (r *RPC) ExecuteCommand(ctx context.Context, endpoint string, req *convoypb.CommandRequest) (*convoypb.CommandResponse, error) {
	client, err := r.client(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, r.cfg.CallTimeout)
	defer cancel()

	return client.ExecuteCommand(ctx, req)
}

// ExecuteShell opens a bidirectional shell stream.
func (r *RPC) ExecuteShell(ctx context.Context, endpoint string) (convoypb.ConvoyService_ExecuteShellClient, error) {
	client, err := r.client(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, r.cfg.CallTimeout)
	stream, err := client.ExecuteShell(ctx)
	if err != nil {
		cancel()
		return nil, err
	}

	// The caller is responsible for canceling via stream.Context().Done when finished.
	return stream, nil
}

// CheckHealth queries the agent health endpoint.
func (r *RPC) CheckHealth(ctx context.Context, endpoint string, req *convoypb.HealthRequest) (*convoypb.HealthResponse, error) {
	client, err := r.client(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, r.cfg.CallTimeout)
	defer cancel()

	return client.CheckHealth(ctx, req)
}

func (r *RPC) client(ctx context.Context, endpoint string) (convoypb.ConvoyServiceClient, error) {
	if endpoint == "" {
		return nil, errors.New("endpoint is required")
	}

	conn, err := r.connection(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	return convoypb.NewConvoyServiceClient(conn), nil
}

func (r *RPC) connection(ctx context.Context, endpoint string) (*grpc.ClientConn, error) {
	r.mu.Lock()
	if conn, ok := r.conns[endpoint]; ok {
		r.mu.Unlock()
		return conn, nil
	}
	r.mu.Unlock()

	dialCtx, cancel := context.WithTimeout(ctx, r.cfg.DialTimeout)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, endpoint, r.dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", endpoint, err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Another goroutine might have created the connection while we dialed.
	if existing, ok := r.conns[endpoint]; ok {
		conn.Close()
		return existing, nil
	}

	r.conns[endpoint] = conn
	return conn, nil
}
