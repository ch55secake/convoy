package cmds

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	convoypb "convoy/api"

	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// NewShellCmd creates the shell command for opening interactive shells in containers.
func NewShellCmd() *cobra.Command {
	var (
		envVars []string
		workDir string
		shell   string
		timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "shell [container-id|name] [flags] [-- command args...]",
		Short: "Open an interactive shell",
		Long: `Open an interactive shell session in a container via the gRPC agent.

The shell runs with PTY support for full terminal emulation including
colors, cursor control, and window resize handling.

Examples:
  # Open default shell in container
  convoy shell my-container

  # Run specific command
  convoy shell my-container -- /bin/bash

  # With environment variables and working directory
  convoy shell my-container -w /app -e FOO=bar -- /bin/sh`,
		SilenceUsage: true,
		Args:         cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			containerRef := args[0]

			// Collect command args after "--"
			var shellArgs []string
			if cmd.ArgsLenAtDash() > 0 {
				shellArgs = args[cmd.ArgsLenAtDash():]
			} else if shell != "" {
				shellArgs = []string{shell}
			}

			containers, err := LoadContainers()
			if err != nil {
				return err
			}

			container, err := containers.ResolveWithEndpoint(containerRef)
			if err != nil {
				return err
			}

			env := ParseEnvVars(envVars)

			return runShell(cmd, container.Endpoint, shellArgs, env, workDir, timeout)
		},
	}

	cmd.Flags().StringArrayVarP(&envVars, "env", "e", nil, "Set environment variables (can be repeated)")
	cmd.Flags().StringVarP(&workDir, "workdir", "w", "", "Working directory inside the container")
	cmd.Flags().StringVar(&shell, "shell", "", "Shell to use (default: agent's configured shell)")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Session timeout (0 = no timeout)")

	return cmd
}

// shellSession manages an interactive shell session over gRPC.
type shellSession struct {
	stream     convoypb.ConvoyService_ExecuteShellClient
	stdinFd    int
	isTerminal bool
	cancel     context.CancelFunc
	exitCode   int32
	exitErr    error
	doneCh     chan struct{} // Signals session completion
}

// newShellSession creates a new shell session manager.
func newShellSession(stream convoypb.ConvoyService_ExecuteShellClient, cancel context.CancelFunc, isTerminal bool) *shellSession {
	return &shellSession{
		stream:     stream,
		stdinFd:    int(os.Stdin.Fd()),
		isTerminal: isTerminal,
		cancel:     cancel,
		doneCh:     make(chan struct{}),
	}
}

// readStdin reads from stdin and sends to the gRPC stream.
// This runs until context is cancelled or stdin returns EOF/error.
// Uses non-blocking I/O to allow checking for exit signals.
func (s *shellSession) readStdin(ctx context.Context) {

	// Set stdin to non-blocking mode so we can poll for exit signals
	fd := int(os.Stdin.Fd())
	oldFlags, _ := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
	unix.FcntlInt(uintptr(fd), unix.F_SETFL, oldFlags|unix.O_NONBLOCK)
	defer unix.FcntlInt(uintptr(fd), unix.F_SETFL, oldFlags)

	buf := make([]byte, 32*1024)

	for {
		// Check if we should stop
		select {
		case <-ctx.Done():
			return
		case <-s.doneCh:
			return
		default:
		}

		n, err := os.Stdin.Read(buf)
		if err != nil {
			// Non-blocking: EAGAIN means no data available, retry
			if errors.Is(err, unix.EAGAIN) {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			// Other errors: send EOF if we got one
			if err == io.EOF {
				_ = s.sendEOF()
			}
			return
		}

		if n > 0 {
			// Check for Ctrl+D (EOF) in raw terminal mode
			// In raw mode, Ctrl+D is sent as literal byte 0x04, not EOF
			if bytes.Contains(buf[:n], []byte{0x04}) {
				_ = s.sendEOF()
				return
			}

			if sendErr := s.sendInput(buf[:n]); sendErr != nil {
				return
			}
		}
	}
}

// sendInput sends stdin data to the stream.
func (s *shellSession) sendInput(data []byte) error {
	chunk := make([]byte, len(data))
	copy(chunk, data)
	return s.stream.Send(&convoypb.ShellRequest{
		Payload: &convoypb.ShellRequest_Input{
			Input: &convoypb.ShellInput{Data: chunk},
		},
	})
}

// sendEOF sends end-of-file signal to the stream.
func (s *shellSession) sendEOF() error {
	return s.stream.Send(&convoypb.ShellRequest{
		Payload: &convoypb.ShellRequest_Input{
			Input: &convoypb.ShellInput{Eof: true},
		},
	})
}

// sendResize sends a window resize event to the stream.
func (s *shellSession) sendResize(rows, cols int) error {
	return s.stream.Send(&convoypb.ShellRequest{
		Payload: &convoypb.ShellRequest_Input{
			Input: &convoypb.ShellInput{
				Resize: &convoypb.ShellResize{
					Rows: uint32(rows), //nolint:gosec // terminal size is always positive and small
					Cols: uint32(cols), //nolint:gosec // terminal size is always positive and small
				},
			},
		},
	})
}

// readOutput receives from the gRPC stream and writes to stdout/stderr.
// Returns when stream ends or exit message is received.
func (s *shellSession) readOutput() {

	for {
		resp, err := s.stream.Recv()
		if err != nil {
			if err != io.EOF {
				s.exitErr = fmt.Errorf("receive: %w", err)
			} else {
			}
			return
		}

		switch payload := resp.GetPayload().(type) {
		case *convoypb.ShellResponse_Output:
			s.writeOutput(payload.Output)
		case *convoypb.ShellResponse_Exit:
			s.handleExit(payload.Exit)
			return // Exit immediately after receiving exit message
		}
	}
}

// writeOutput writes shell output to the appropriate stream.
func (s *shellSession) writeOutput(output *convoypb.ShellOutput) {
	if output == nil {
		return
	}
	switch output.GetStream() {
	case convoypb.ShellOutput_STDERR:
		_, _ = os.Stderr.Write(output.GetData())
	default:
		_, _ = os.Stdout.Write(output.GetData())
	}
}

// handleExit processes the exit message from the shell.
func (s *shellSession) handleExit(exit *convoypb.ShellExit) {
	if exit == nil {
		return
	}
	s.exitCode = exit.GetExitCode()
	if s.exitCode != 0 && exit.GetMessage() != "" {
		s.exitErr = fmt.Errorf("shell exited: %s", exit.GetMessage())
	}
}

// handleSignals processes OS signals until context is done.
func (s *shellSession) handleSignals(ctx context.Context, sigCh <-chan os.Signal) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.doneCh:
			return
		case sig := <-sigCh:
			s.processSignal(sig)
		}
	}
}

// processSignal handles a single signal.
func (s *shellSession) processSignal(sig os.Signal) {
	switch sig {
	case syscall.SIGWINCH:
		s.handleWindowResize()
	case syscall.SIGINT, syscall.SIGTERM:
		s.handleInterrupt()
	}
}

// handleWindowResize sends a resize event when terminal size changes.
func (s *shellSession) handleWindowResize() {
	if !s.isTerminal {
		return
	}
	width, height, err := term.GetSize(s.stdinFd)
	if err == nil {
		_ = s.sendResize(height, width)
	}
}

// handleInterrupt handles Ctrl+C and termination signals.
func (s *shellSession) handleInterrupt() {
	if s.isTerminal {
		// Forward Ctrl+C as actual byte to PTY
		_ = s.sendInput([]byte{0x03})
	} else {
		s.cancel()
	}
}

// run starts the session and blocks until completion.
func (s *shellSession) run(ctx context.Context) error {

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGWINCH)
	defer signal.Stop(sigCh)

	// Start stdin reader (fire and forget - will exit when doneCh closes)
	go s.readStdin(ctx)

	// Start signal handler (fire and forget - will exit when doneCh closes)
	go s.handleSignals(ctx, sigCh)

	// Read output synchronously - this blocks until shell exits
	s.readOutput()

	// Signal other goroutines to stop
	close(s.doneCh)
	s.cancel()

	// Close the send side of the stream
	_ = s.stream.CloseSend()

	if s.exitCode != 0 {
		return fmt.Errorf("exit code %d", s.exitCode)
	}
	return s.exitErr
}

// runShell executes an interactive shell session over gRPC.
func runShell(cmd *cobra.Command, endpoint string, args []string, env map[string]string, workDir string, timeout time.Duration) error {
	stdinFd := int(os.Stdin.Fd())
	stdoutFd := int(os.Stdout.Fd())
	isTerminal := term.IsTerminal(stdinFd)

	var rows, cols uint32
	if isTerminal {
		width, height, err := term.GetSize(stdinFd)
		if err == nil {
			rows = uint32(height) //nolint:gosec // terminal size is always positive and small
			cols = uint32(width)  //nolint:gosec // terminal size is always positive and small
		}
	}

	// Create RPC client with generous dial timeout
	rpc := NewRPCClient(10*time.Second, 0)
	defer func() {
		_ = rpc.Close()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := rpc.ExecuteShell(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("open shell stream: %w", err)
	}

	// Send start message
	if err := sendShellStart(stream, args, env, workDir, isTerminal, rows, cols, timeout); err != nil {
		return err
	}

	// Set terminal to raw mode if interactive
	var oldState *term.State
	if isTerminal {
		oldState, err = term.MakeRaw(stdinFd)
		if err != nil {
			return fmt.Errorf("set raw mode: %w", err)
		}
		defer func() {
			_ = term.Restore(stdinFd, oldState)
		}()
	}

	// Create and run session
	session := newShellSession(stream, cancel, isTerminal)
	sessionErr := session.run(ctx)

	// Restore terminal and ensure clean output
	if isTerminal && oldState != nil {
		_ = term.Restore(stdinFd, oldState)
		if term.IsTerminal(stdoutFd) {
			_, _ = fmt.Fprintln(cmd.OutOrStdout())
		}
	}

	return sessionErr
}

// sendShellStart sends the initial shell start message.
func sendShellStart(stream convoypb.ConvoyService_ExecuteShellClient, args []string, env map[string]string, workDir string, pty bool, rows, cols uint32, timeout time.Duration) error {
	msg := &convoypb.ShellRequest{
		Payload: &convoypb.ShellRequest_Start{
			Start: &convoypb.ShellStart{
				Args:           args,
				Env:            env,
				WorkDir:        workDir,
				Pty:            pty,
				Rows:           rows,
				Cols:           cols,
				TimeoutSeconds: int32(timeout.Seconds()), //nolint:gosec // timeout in seconds is always reasonable
			},
		},
	}
	if err := stream.Send(msg); err != nil {
		return fmt.Errorf("send start message: %w", err)
	}
	return nil
}
