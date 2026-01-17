package agent

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	convoypb "convoy/api"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server provides the ConvoyService RPC implementation.
type Server struct {
	cfg  *Config
	sema chan struct{}
	grpc *grpc.Server
	mu   sync.Mutex
	convoypb.UnimplementedConvoyServiceServer
}

// NewServer constructs a server with sane defaults.
func NewServer(cfg *Config) *Server {
	return &Server{
		cfg:  cfg,
		sema: make(chan struct{}, cfg.MaxConcurrent),
	}
}

// Start boots the gRPC server until the context is canceled.
func (s *Server) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.GRPCPort))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	s.grpc = grpc.NewServer()
	convoypb.RegisterConvoyServiceServer(s.grpc, s)

	go func() {
		<-ctx.Done()
		s.grpc.GracefulStop()
	}()

	log.Printf("convoy agent listening on %d", s.cfg.GRPCPort)
	return s.grpc.Serve(lis)
}

// ExecuteCommand is a placeholder implementation.
func (s *Server) ExecuteCommand(ctx context.Context, req *convoypb.CommandRequest) (*convoypb.CommandResponse, error) {
	if len(req.GetArgs()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "args required")
	}

	return &convoypb.CommandResponse{
		Stdout:   fmt.Sprintf("command %v not yet implemented", req.GetArgs()),
		Stderr:   "",
		ExitCode: 0,
	}, nil
}

// ExecuteShell currently streams an unimplemented error.
func (s *Server) ExecuteShell(stream convoypb.ConvoyService_ExecuteShellServer) error {
	return status.Error(codes.Unimplemented, "shell streaming not implemented")
}

// CheckHealth reports basic readiness.
func (s *Server) CheckHealth(ctx context.Context, req *convoypb.HealthRequest) (*convoypb.HealthResponse, error) {
	return &convoypb.HealthResponse{
		Status:  convoypb.HealthResponse_STATUS_HEALTHY,
		Message: "ok",
	}, nil
}
