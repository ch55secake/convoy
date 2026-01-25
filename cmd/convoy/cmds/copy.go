package cmds

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	convoypb "convoy/api"
	"convoy/internal/orchestrator"

	"github.com/spf13/cobra"
)

// copyEndpoint represents a source or destination for copy operations.
type copyEndpoint struct {
	isContainer bool
	container   string
	path        string
}

func NewCopyCmd() *cobra.Command {
	var (
		timeout   time.Duration
		overwrite bool
	)

	cmd := &cobra.Command{
		Use:   "copy <source> <destination> [destination...]",
		Short: "Copy files/folders to or from containers",
		Long: `Copy files or folders between host and containers.

				Paths can be specified as:
				  - Local path: /path/to/file or ./relative/path
				  - Container path: container-name:/path/in/container
				
				Examples:
				  # Copy from host to single container
				  convoy copy ./myfile.txt mycontainer:/tmp/myfile.txt
				
				  # Copy from host to multiple containers
				  convoy copy ./config.yaml c1:/etc/app/config.yaml c2:/etc/app/config.yaml
				
				  # Copy from container to host
				  convoy copy mycontainer:/var/log/app.log ./app.log
				
				  # Copy directory from host to container
				  convoy copy ./mydir mycontainer:/opt/mydir
				
				  # Copy between containers (uses host as relay)
				  convoy copy c1:/data/file.txt c2:/backup/file.txt`,
		Args:         cobra.MinimumNArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			source, err := parseEndpoint(args[0])
			if err != nil {
				return fmt.Errorf("invalid source: %w", err)
			}

			var destinations []copyEndpoint
			for _, arg := range args[1:] {
				dest, err := parseEndpoint(arg)
				if err != nil {
					return fmt.Errorf("invalid destination %q: %w", arg, err)
				}
				destinations = append(destinations, dest)
			}

			hasContainer := source.isContainer
			for _, d := range destinations {
				if d.isContainer {
					hasContainer = true
					break
				}
			}
			if !hasContainer {
				return fmt.Errorf("at least one endpoint must be a container")
			}

			hostDestCount := 0
			for _, d := range destinations {
				if !d.isContainer {
					hostDestCount++
				}
			}
			if hostDestCount > 1 {
				return fmt.Errorf("only one host destination allowed per invocation")
			}

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

			rpc := orchestrator.NewRPC(orchestrator.RPCConfig{
				DialTimeout: timeout,
				CallTimeout: 0,
			})
			defer func() {
				_ = rpc.Close()
			}()

			ctx := context.Background()

			switch {
			case !source.isContainer:
				return copyHostToContainers(ctx, cmd, rpc, containers, source, destinations, overwrite)
			case len(destinations) == 1 && !destinations[0].isContainer:
				return copyContainerToHost(ctx, cmd, rpc, containers, source, destinations[0], overwrite)
			default:
				return copyContainerToContainers(ctx, cmd, rpc, containers, source, destinations, overwrite)
			}
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "Timeout for copy operations")
	cmd.Flags().BoolVar(&overwrite, "overwrite", true, "Overwrite existing files")

	return cmd
}

// parseEndpoint parses a string like "container:/path" or "/local/path".
func parseEndpoint(s string) (copyEndpoint, error) {
	if s == "" {
		return copyEndpoint{}, fmt.Errorf("empty endpoint")
	}

	if !strings.HasPrefix(s, "/") && !strings.HasPrefix(s, ".") && !strings.HasPrefix(s, "~") {
		if idx := strings.Index(s, ":"); idx > 0 {
			container := s[:idx]
			path := s[idx+1:]
			if path == "" {
				path = "/"
			}
			return copyEndpoint{
				isContainer: true,
				container:   container,
				path:        path,
			}, nil
		}
	}

	return copyEndpoint{
		isContainer: false,
		path:        s,
	}, nil
}

// copyHostToContainers copies from local filesystem to one or more containers.
func copyHostToContainers(ctx context.Context, cmd *cobra.Command, rpc *orchestrator.RPC, containers []*orchestrator.Container, source copyEndpoint, destinations []copyEndpoint, overwrite bool) error {
	srcPath := source.path

	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("source not found: %w", err)
	}

	var failed bool
	for _, dest := range destinations {
		if !dest.isContainer {
			continue
		}

		container := resolveContainer(dest.container, containers)
		if container == nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "container not found: %s\n", dest.container)
			failed = true
			continue
		}

		if container.Endpoint == "" {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "container %s has no gRPC endpoint\n", dest.container)
			failed = true
			continue
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Copying %s to %s:%s\n", srcPath, dest.container, dest.path)

		if err := pushToContainer(ctx, rpc, container.Endpoint, srcPath, srcInfo, dest.path, overwrite); err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "failed to copy to %s: %v\n", dest.container, err)
			failed = true
			continue
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Successfully copied to %s\n", dest.container)
	}

	if failed {
		return fmt.Errorf("one or more copy operations failed")
	}
	return nil
}

// copyContainerToHost copies from a container to local filesystem.
func copyContainerToHost(ctx context.Context, cmd *cobra.Command, rpc *orchestrator.RPC, containers []*orchestrator.Container, source copyEndpoint, dest copyEndpoint, overwrite bool) error {
	container := resolveContainer(source.container, containers)
	if container == nil {
		return fmt.Errorf("container not found: %s", source.container)
	}

	if container.Endpoint == "" {
		return fmt.Errorf("container %s has no gRPC endpoint", source.container)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Copying %s:%s to %s\n", source.container, source.path, dest.path)

	if err := pullFromContainer(ctx, rpc, container.Endpoint, source.path, dest.path, overwrite); err != nil {
		return fmt.Errorf("failed to copy from %s: %w", source.container, err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Successfully copied from %s\n", source.container)
	return nil
}

// copyContainerToContainers copies from one container to other containers via host relay.
func copyContainerToContainers(ctx context.Context, cmd *cobra.Command, rpc *orchestrator.RPC, containers []*orchestrator.Container, source copyEndpoint, destinations []copyEndpoint, overwrite bool) error {
	srcContainer := resolveContainer(source.container, containers)
	if srcContainer == nil {
		return fmt.Errorf("source container not found: %s", source.container)
	}

	if srcContainer.Endpoint == "" {
		return fmt.Errorf("source container %s has no gRPC endpoint", source.container)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Pulling %s:%s for relay...\n", source.container, source.path)

	tarData, err := pullTarFromContainer(ctx, rpc, srcContainer.Endpoint, source.path)
	if err != nil {
		return fmt.Errorf("failed to pull from source container: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Pulled %d bytes from %s\n", len(tarData), source.container)

	var failed bool
	for _, dest := range destinations {
		if !dest.isContainer {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Extracting to local path %s\n", dest.path)
			if err := extractTarToLocal(tarData, dest.path, overwrite); err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "failed to extract to %s: %v\n", dest.path, err)
				failed = true
			}
			continue
		}

		destContainer := resolveContainer(dest.container, containers)
		if destContainer == nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "destination container not found: %s\n", dest.container)
			failed = true
			continue
		}

		if destContainer.Endpoint == "" {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "destination container %s has no gRPC endpoint\n", dest.container)
			failed = true
			continue
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Pushing to %s:%s\n", dest.container, dest.path)

		if err := pushTarToContainer(ctx, rpc, destContainer.Endpoint, tarData, dest.path, overwrite); err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "failed to push to %s: %v\n", dest.container, err)
			failed = true
			continue
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Successfully copied to %s\n", dest.container)
	}

	if failed {
		return fmt.Errorf("one or more copy operations failed")
	}
	return nil
}

// pushToContainer streams a local file/directory as tar to a container.
func pushToContainer(ctx context.Context, rpc *orchestrator.RPC, endpoint, srcPath string, srcInfo os.FileInfo, destPath string, overwrite bool) error {
	stream, err := rpc.Copy(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("failed to open copy stream: %w", err)
	}

	if err := stream.Send(&convoypb.CopyRequest{
		Payload: &convoypb.CopyRequest_Start{
			Start: &convoypb.CopyStart{
				Direction: convoypb.CopyStart_TO_AGENT,
				Path:      destPath,
				Overwrite: overwrite,
			},
		},
	}); err != nil {
		return fmt.Errorf("failed to send start message: %w", err)
	}

	pr, pw := io.Pipe()

	go func() {
		tw := tar.NewWriter(pw)
		var tarErr error

		if srcInfo.IsDir() {
			tarErr = filepath.Walk(srcPath, func(path string, info os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}

				relPath, err := filepath.Rel(srcPath, path)
				if err != nil {
					return err
				}

				if relPath == "." {
					return nil
				}

				return addFileToTar(tw, path, relPath, info)
			})
		} else {
			tarErr = addFileToTar(tw, srcPath, filepath.Base(srcPath), srcInfo)
		}

		_ = tw.Close()
		if tarErr != nil {
			_ = pw.CloseWithError(tarErr)
		} else {
			_ = pw.Close()
		}
	}()

	buf := make([]byte, 32*1024)
	for {
		n, readErr := pr.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			if err := stream.Send(&convoypb.CopyRequest{
				Payload: &convoypb.CopyRequest_Chunk{
					Chunk: &convoypb.CopyChunk{
						Data: chunk,
						Eof:  false,
					},
				},
			}); err != nil {
				return fmt.Errorf("failed to send data chunk: %w", err)
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("tar read error: %w", readErr)
		}
	}

	if err := stream.Send(&convoypb.CopyRequest{
		Payload: &convoypb.CopyRequest_Chunk{
			Chunk: &convoypb.CopyChunk{
				Data: nil,
				Eof:  true,
			},
		},
	}); err != nil {
		return fmt.Errorf("failed to send EOF: %w", err)
	}

	if err := stream.CloseSend(); err != nil {
		return fmt.Errorf("failed to close send: %w", err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("receive error: %w", err)
		}

		if result := resp.GetResult(); result != nil {
			if !result.GetSuccess() {
				return fmt.Errorf("copy failed: %s", result.GetMessage())
			}
			return nil
		}
	}

	return nil
}

// pullFromContainer pulls data from a container and extracts to local filesystem.
func pullFromContainer(ctx context.Context, rpc *orchestrator.RPC, endpoint, srcPath, destPath string, overwrite bool) error {
	stream, err := rpc.Copy(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("failed to open copy stream: %w", err)
	}

	if err := stream.Send(&convoypb.CopyRequest{
		Payload: &convoypb.CopyRequest_Start{
			Start: &convoypb.CopyStart{
				Direction: convoypb.CopyStart_FROM_AGENT,
				Path:      srcPath,
				Overwrite: overwrite,
			},
		},
	}); err != nil {
		return fmt.Errorf("failed to send start message: %w", err)
	}

	if err := stream.CloseSend(); err != nil {
		return fmt.Errorf("failed to close send: %w", err)
	}

	if err := os.MkdirAll(destPath, 0o755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	pr, pw := io.Pipe()
	extractDone := make(chan error, 1)

	go func() {
		extractDone <- extractTarFromReader(pr, destPath, overwrite)
	}()

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			_ = pw.CloseWithError(err)
			return fmt.Errorf("receive error: %w", err)
		}

		if chunk := resp.GetChunk(); chunk != nil {
			if len(chunk.GetData()) > 0 {
				if _, err := pw.Write(chunk.GetData()); err != nil {
					return fmt.Errorf("pipe write error: %w", err)
				}
			}
			if chunk.GetEof() {
				break
			}
		}

		if result := resp.GetResult(); result != nil {
			if !result.GetSuccess() {
				_ = pw.CloseWithError(fmt.Errorf("copy failed: %s", result.GetMessage()))
				return fmt.Errorf("copy failed: %s", result.GetMessage())
			}
		}
	}

	_ = pw.Close()

	return <-extractDone
}

// pullTarFromContainer pulls data from a container and returns the raw tar bytes.
func pullTarFromContainer(ctx context.Context, rpc *orchestrator.RPC, endpoint, srcPath string) ([]byte, error) {
	stream, err := rpc.Copy(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to open copy stream: %w", err)
	}

	if err := stream.Send(&convoypb.CopyRequest{
		Payload: &convoypb.CopyRequest_Start{
			Start: &convoypb.CopyStart{
				Direction: convoypb.CopyStart_FROM_AGENT,
				Path:      srcPath,
				Overwrite: false,
			},
		},
	}); err != nil {
		return nil, fmt.Errorf("failed to send start message: %w", err)
	}

	if err := stream.CloseSend(); err != nil {
		return nil, fmt.Errorf("failed to close send: %w", err)
	}

	var tarData []byte
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("receive error: %w", err)
		}

		if chunk := resp.GetChunk(); chunk != nil {
			if len(chunk.GetData()) > 0 {
				tarData = append(tarData, chunk.GetData()...)
			}
			if chunk.GetEof() {
				break
			}
		}

		if result := resp.GetResult(); result != nil {
			if !result.GetSuccess() {
				return nil, fmt.Errorf("copy failed: %s", result.GetMessage())
			}
		}
	}

	return tarData, nil
}

// pushTarToContainer sends pre-built tar data to a container.
func pushTarToContainer(ctx context.Context, rpc *orchestrator.RPC, endpoint string, tarData []byte, destPath string, overwrite bool) error {
	stream, err := rpc.Copy(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("failed to open copy stream: %w", err)
	}

	if err := stream.Send(&convoypb.CopyRequest{
		Payload: &convoypb.CopyRequest_Start{
			Start: &convoypb.CopyStart{
				Direction: convoypb.CopyStart_TO_AGENT,
				Path:      destPath,
				Overwrite: overwrite,
			},
		},
	}); err != nil {
		return fmt.Errorf("failed to send start message: %w", err)
	}

	chunkSize := 32 * 1024
	for i := 0; i < len(tarData); i += chunkSize {
		end := i + chunkSize
		if end > len(tarData) {
			end = len(tarData)
		}

		if err := stream.Send(&convoypb.CopyRequest{
			Payload: &convoypb.CopyRequest_Chunk{
				Chunk: &convoypb.CopyChunk{
					Data: tarData[i:end],
					Eof:  false,
				},
			},
		}); err != nil {
			return fmt.Errorf("failed to send data chunk: %w", err)
		}
	}

	if err := stream.Send(&convoypb.CopyRequest{
		Payload: &convoypb.CopyRequest_Chunk{
			Chunk: &convoypb.CopyChunk{
				Data: nil,
				Eof:  true,
			},
		},
	}); err != nil {
		return fmt.Errorf("failed to send EOF: %w", err)
	}

	if err := stream.CloseSend(); err != nil {
		return fmt.Errorf("failed to close send: %w", err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("receive error: %w", err)
		}

		if result := resp.GetResult(); result != nil {
			if !result.GetSuccess() {
				return fmt.Errorf("copy failed: %s", result.GetMessage())
			}
			return nil
		}
	}

	return nil
}

// extractTarToLocal extracts tar data to a local directory.
func extractTarToLocal(tarData []byte, destPath string, overwrite bool) error {
	if err := os.MkdirAll(destPath, 0o755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	reader := tar.NewReader(strings.NewReader(string(tarData)))
	return extractTarEntries(reader, destPath, overwrite)
}

// extractTarFromReader extracts tar data from a reader to a local directory.
func extractTarFromReader(r io.Reader, destPath string, overwrite bool) error {
	reader := tar.NewReader(r)
	return extractTarEntries(reader, destPath, overwrite)
}

// extractTarEntries extracts entries from a tar reader to a destination path.
func extractTarEntries(reader *tar.Reader, destPath string, overwrite bool) error {
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("tar read error: %w", err)
		}

		targetPath := filepath.Join(destPath, header.Name)

		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(destPath)) {
			return fmt.Errorf("invalid tar entry path: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}

			if _, err := io.Copy(file, reader); err != nil {
				_ = file.Close()
				return fmt.Errorf("failed to write file %s: %w", targetPath, err)
			}
			_ = file.Close()

		case tar.TypeSymlink:
			if overwrite {
				_ = os.Remove(targetPath)
			}
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				return fmt.Errorf("failed to create symlink %s: %w", targetPath, err)
			}
		}
	}
}

// addFileToTar adds a single file or directory to a tar writer.
func addFileToTar(tw *tar.Writer, srcPath, relPath string, info os.FileInfo) error {
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = relPath

	if info.Mode()&os.ModeSymlink != 0 {
		linkTarget, err := os.Readlink(srcPath)
		if err != nil {
			return err
		}
		header.Linkname = linkTarget
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil
	}

	file, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	_, err = io.Copy(tw, file)
	return err
}
