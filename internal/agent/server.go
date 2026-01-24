package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

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
	_    sync.Mutex
	convoypb.UnimplementedConvoyServiceServer
}

// NewServer constructs a server with sane defaults.
func NewServer(cfg *Config) *Server {
	maxConcurrent := cfg.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 1
	}

	return &Server{
		cfg:  cfg,
		sema: make(chan struct{}, maxConcurrent),
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

// ExecuteCommand runs a non-interactive command on the host.
func (s *Server) ExecuteCommand(ctx context.Context, req *convoypb.CommandRequest) (*convoypb.CommandResponse, error) {
	if len(req.GetArgs()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "args required")
	}

	if err := s.acquire(ctx); err != nil {
		return nil, err
	}
	defer s.release()

	timeout := durationFromRequest(req.GetTimeoutSeconds(), s.cfg.ExecTimeout)
	cmdCtx := ctx
	var cancel context.CancelFunc
	if timeout > 0 {
		cmdCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(cmdCtx, req.GetArgs()[0], req.GetArgs()[1:]...)
	cmd.Dir = req.GetWorkDir()
	cmd.Env = mergeEnv(req.GetEnv())

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()

	resp := &convoypb.CommandResponse{
		Stdout: stdoutBuf.String(),
		Stderr: stderrBuf.String(),
	}

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			resp.ExitCode = int32(exitErr.ExitCode())
			resp.ErrorMessage = exitErr.Error()
		} else {
			resp.ExitCode = -1
			resp.ErrorMessage = err.Error()
		}

		// Distinguish between context cancellation and execution failure.
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(cmdCtx.Err(), context.DeadlineExceeded) {
			return resp, status.Error(codes.DeadlineExceeded, "command timed out")
		}
		if errors.Is(err, context.Canceled) || errors.Is(cmdCtx.Err(), context.Canceled) {
			return resp, status.Error(codes.Canceled, "command canceled")
		}

		return resp, status.Errorf(codes.Unknown, "command failed: %v", err)
	}

	resp.ExitCode = 0
	return resp, nil
}

// ExecuteShell runs an interactive shell session streamed over gRPC.
func (s *Server) ExecuteShell(stream convoypb.ConvoyService_ExecuteShellServer) error {
	ctx := stream.Context()
	if err := s.acquire(ctx); err != nil {
		return err
	}
	defer s.release()

	firstReq, err := stream.Recv()
	if err != nil {
		return err
	}

	start := firstReq.GetStart()
	if start == nil {
		return status.Error(codes.InvalidArgument, "first message must be start")
	}

	args := start.GetArgs()
	if len(args) == 0 {
		args = []string{s.cfg.ShellPath}
	}

	cmdCtx := ctx
	var cancel context.CancelFunc
	if s.cfg.ExecTimeout > 0 {
		cmdCtx, cancel = context.WithTimeout(ctx, s.cfg.ExecTimeout)
		defer cancel()
	}

	cmd := exec.CommandContext(cmdCtx, args[0], args[1:]...)
	cmd.Env = mergeEnv(start.GetEnv())
	cmd.Dir = start.GetWorkDir()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return status.Errorf(codes.Internal, "stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return status.Errorf(codes.Internal, "stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return status.Errorf(codes.Internal, "stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return status.Errorf(codes.Internal, "start shell: %v", err)
	}

	outputCh := make(chan *convoypb.ShellResponse, 16)
	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	streamPipe := func(r io.Reader, streamType convoypb.ShellOutput_Stream) {
		defer wg.Done()
		buf := make([]byte, 32*1024)
		for {
			n, readErr := r.Read(buf)
			if n > 0 {
				chunk := make([]byte, n)
				copy(chunk, buf[:n])
				resp := &convoypb.ShellResponse{
					Payload: &convoypb.ShellResponse_Output{
						Output: &convoypb.ShellOutput{Stream: streamType, Data: chunk},
					},
				}

				select {
				case outputCh <- resp:
				case <-cmdCtx.Done():
					return
				}
			}

			if readErr != nil {
				if errors.Is(readErr, io.EOF) {
					return
				}
				errCh <- readErr
				return
			}
		}
	}

	go streamPipe(stdout, convoypb.ShellOutput_STDOUT)
	go streamPipe(stderr, convoypb.ShellOutput_STDERR)

	go func() {
		wg.Wait()
		close(outputCh)
		close(errCh)
	}()

	inputErrCh := make(chan error, 1)
	go func() {
		for {
			req, recvErr := stream.Recv()
			if recvErr == io.EOF {
				inputErrCh <- stdin.Close()
				return
			}
			if recvErr != nil {
				inputErrCh <- recvErr
				return
			}
			input := req.GetInput()
			if input == nil {
				continue
			}
			if len(input.GetData()) > 0 {
				if _, writeErr := stdin.Write(input.GetData()); writeErr != nil {
					inputErrCh <- writeErr
					return
				}
			}
			if input.GetEof() {
				inputErrCh <- stdin.Close()
				return
			}
		}
	}()

	for {
		select {
		case resp, ok := <-outputCh:
			if !ok {
				outputCh = nil
				continue
			}
			if resp == nil {
				continue
			}
			if err := stream.Send(resp); err != nil {
				_ = cmd.Process.Kill()
				return err
			}
		case pipeErr, ok := <-errCh:
			if ok && pipeErr != nil {
				_ = cmd.Process.Kill()
				return pipeErr
			}
		case inputErr := <-inputErrCh:
			if inputErr != nil {
				_ = cmd.Process.Kill()
				return inputErr
			}
			inputErrCh = nil
		case <-cmdCtx.Done():
			_ = cmd.Process.Kill()
			return cmdCtx.Err()
		default:
			if outputCh == nil && inputErrCh == nil {
				goto waitExit
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

waitExit:
	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		msg := err.Error()
		exitCode := int32(-1)
		if errors.As(err, &exitErr) {
			exitCode = int32(exitErr.ExitCode())
			msg = exitErr.Error()
		}
		return stream.Send(&convoypb.ShellResponse{
			Payload: &convoypb.ShellResponse_Exit{
				Exit: &convoypb.ShellExit{ExitCode: exitCode, Message: msg},
			},
		})
	}

	return stream.Send(&convoypb.ShellResponse{
		Payload: &convoypb.ShellResponse_Exit{
			Exit: &convoypb.ShellExit{ExitCode: 0, Message: ""},
		},
	})
}

// CheckHealth reports basic readiness.
func (s *Server) CheckHealth(_ context.Context, _ *convoypb.HealthRequest) (*convoypb.HealthResponse, error) {
	log.Printf("health check requested")
	return &convoypb.HealthResponse{
		Status:  convoypb.HealthResponse_STATUS_HEALTHY,
		Message: "ok",
	}, nil
}

func (s *Server) acquire(ctx context.Context) error {
	select {
	case s.sema <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Server) release() {
	select {
	case <-s.sema:
	default:
	}
}

func durationFromRequest(seconds int32, fallback time.Duration) time.Duration {
	if seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return fallback
}

func mergeEnv(overrides map[string]string) []string {
	base := map[string]string{}
	for _, kv := range os.Environ() {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			base[parts[0]] = parts[1]
		}
	}

	for k, v := range overrides {
		if k == "" {
			continue
		}
		base[k] = v
	}

	result := make([]string, 0, len(base))
	for k, v := range base {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}

	return result
}
