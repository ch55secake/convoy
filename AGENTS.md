# AGENTS.md - Convoy Project Guidelines

This document provides essential information for agentic coding assistants working on the Convoy project.

## Project Overview
Convoy is a Go-based CLI tool for orchestrating multiple Alpine Linux containers via Docker, using gRPC for communication and round-robin load balancing.

## Build Commands

### Build the Application
```bash
make compile 
# or
go build -o bin/convoy ./cmd/convoy
```

### Clean Build Artifacts
```bash
make clean
# or
rm -rf bin/
```

## Test Commands

### Run All Tests
```bash
make test
# or
go test ./...
```

### Run Tests for Specific Package
```bash
go test ./pkg/loadbalancer
go test ./internal/orchestrator
go test ./internal/app
```

### Run a Single Test Function
```bash
go test -run TestFunctionName ./path/to/package
# Example: go test -run TestRoundRobin ./pkg/loadbalancer
```

### Run Tests with Verbose Output
```bash
go test -v ./...
```

### Run Tests with Coverage
```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Linting and Code Quality

### Run Linter
```bash
make lint
# or
golangci-lint run
```

### Auto-fix Formatting Issues
```bash
gofmt -w .
goimports -w .
```

### Check for Security Issues
```bash
gosec ./...
```

## Code Style Guidelines

### Formatting
- Use `gofmt` for code formatting (simplified mode enabled)
- Use `goimports` for import organization and formatting
- Maximum line length: Follow standard Go conventions (typically 80-120 characters)

### Imports
- Group imports: standard library, third-party, local packages
- Use blank lines between import groups
- Example:
```go
import (
    "fmt"
    "sync"

    "github.com/spf13/cobra"
)
```

### Naming Conventions

#### Variables and Functions
- Use `camelCase` for unexported identifiers
- Use `PascalCase` for exported identifiers
- Functions: `DoSomething()`, `doSomething()`
- Variables: `userName`, `configFile`

#### Types and Structs
- Struct names: `UserConfig`, `LoadBalancer`
- Interface names: `Balancer`, `Executor`
- Method receivers: Use short, meaningful names (e.g., `rr` for RoundRobin)

#### Constants
- Use `PascalCase` for exported constants
- Use `camelCase` for unexported constants
- Group related constants together

### Error Handling
- Always check and handle errors appropriately
- Use `if err != nil` pattern consistently
- Return errors from functions that can fail
- Use error wrapping when appropriate: `fmt.Errorf("failed to connect: %w", err)`
- Avoid panic() except for truly exceptional circumstances

### Comments
- Document exported functions, types, and constants
- Use complete sentences starting with capital letters
- End sentences with periods
- Example: `// NewRoundRobin creates a new RoundRobin balancer`

### Package Organization
- `cmd/`: Main applications and CLI commands
- `pkg/`: Library code that can be used by external applications
- `internal/`: Private application and library code

### Interfaces
- Define interfaces close to their usage
- Keep interfaces small and focused
- Use interface{} sparingly; prefer specific types
- Example:
```go
type Balancer interface {
    Next() string
    AddServer(server string)
    RemoveServer(server string)
}
```

### Structs and Methods
- Group related methods together
- Use pointer receivers for methods that modify the receiver
- Use value receivers for methods that don't modify the receiver
- Example:
```go
func (rr *RoundRobin) AddServer(server string) {
    // modifies receiver, uses pointer
}

func (rr RoundRobin) Next() string {
    // doesn't modify receiver, uses value
}
```

### Synchronization
- Use `sync.Mutex` for protecting shared state
- Always use `defer mu.Unlock()` after `mu.Lock()`
- Prefer channels over mutexes when possible for goroutine communication
- Example:
```go
func (rr *RoundRobin) Next() string {
    rr.mu.Lock()
    defer rr.mu.Unlock()
    // ... implementation
}
```

### Testing
- Use `_test.go` suffix for test files
- Test functions start with `Test`
- Use table-driven tests for multiple test cases
- Example:
```go
func TestRoundRobin_Next(t *testing.T) {
    tests := []struct {
        name     string
        servers  []string
        expected string
    }{
        {"single server", []string{"server1"}, "server1"},
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            rr := NewRoundRobin()
            for _, server := range tt.servers {
                rr.AddServer(server)
            }
            result := rr.Next()
            if result != tt.expected {
                t.Errorf("expected %s, got %s", tt.expected, result)
            }
        })
    }
}
```

### Dependencies
- Keep dependencies minimal
- Use Go modules for dependency management
- Run `go mod tidy` regularly to clean up dependencies
- Current dependencies: `github.com/spf13/cobra`

### Git Workflow
- Use descriptive commit messages
- Follow conventional commit format when possible
- Example: `feat: add round-robin load balancer`

### Docker Integration
- Use Alpine Linux for containers (as per project goals)
- Ensure all container operations are properly handled
- Implement proper cleanup for container resources

### RPC/gRPC Guidelines
- Use clear, descriptive service and method names
- Handle connection errors gracefully
- Implement proper timeouts for RPC calls
- Use context for cancellation and timeouts

### Logging
- Use structured logging when implemented
- Log errors and important state changes
- Avoid logging sensitive information

### Configuration
- Use YAML for configuration files (planned)
- Validate configuration on startup
- Provide sensible defaults

## Development Workflow

1. **Before coding**: Run `make lint` to ensure clean starting point
2. **During development**: Run `make compile` frequently to catch compilation errors
3. **Before committing**: Run `make test && make lint` to ensure quality
4. **For new features**: Write tests first, then implement functionality
5. **For bug fixes**: Write test that reproduces the bug, then fix it

## Linter Configuration Details

The project uses golangci-lint with the following enabled linters:
- `govet`: Reports suspicious code constructs
- `errcheck`: Checks for unchecked errors
- `staticcheck`: Advanced static analysis
- `unused`: Checks for unused constants, variables, functions
- `ineffassign`: Detects ineffective assignments
- `gocritic`: Provides code improvements suggestions
- `gocyclo`: Checks cyclomatic complexity (min 15)
- `revive`: Configurable linter for style and conventions
- `dupl`: Detects code duplication (threshold 150)
- `gosec`: Security issues checker

## Performance Considerations
- Avoid unnecessary allocations in hot paths
- Use sync.Pool for frequently allocated objects
- Profile code with `go tool pprof` when optimizing
- Keep cyclomatic complexity under control

## Security Best Practices
- Validate all inputs from external sources
- Use gosec for automated security checks
- Avoid logging sensitive data
- Follow principle of least privilege
- Keep dependencies updated to avoid vulnerabilities

This document should be updated as the project evolves and new conventions are established.