# Default recipe (runs when you type 'just')
default:
    @just --list

# Build the Go binary
compile:
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
