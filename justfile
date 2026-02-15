# Default recipe (runs when you type 'just')
default:
    @just --list

# Build all binaries and image
build: build-cli build-agent build-image

# Build the CLI binary
build-cli:
    go build -o bin/convoy ./cmd/convoy

# Run tests
test:
    go test ./...

# Run linter
lint:
    golangci-lint run

# Clean build artifacts
clean:
    rm -rf bin/

# Build Docker image
build-image:
    docker build -f Dockerfile -t convoy:latest .

# Build the convoy agent binary
build-agent:
    go build -o bin/convoy-agent ./cmd/agent
