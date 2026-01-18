# convoy
🚢 Manage multiple containers and multiple tasks at once

Convoy is a Go-based CLI tool for orchestrating multiple Alpine Linux containers via Docker. It uses gRPC for communication, supervisord for process management, and round-robin load balancing to distribute tasks evenly.

## Features
- **Container Orchestration**: Create, manage, and stop multiple Alpine Linux containers.
- **Command Execution**: Send commands to individual containers or all at once via gRPC over TCP.
- **Load Balancing**: Evenly distribute tasks across containers using round-robin algorithm.
- **Individual Management**: Inspect, view logs, check stats (CPU/memory), restart, and access interactive shells on containers.
- **Process Management**: Supervisord handles gRPC servers in containers for reliability.
- **CLI**: Built with Cobra; alias `cvy` for convenience.

## Installation
Prerequisites: Go 1.21+, Docker, GitHub CLI (for remote setup).

```bash
go install ./cmd/convoy
```

## Usage
```bash
convoy --help
# or
cvy --help
```

### Commands
- `convoy start <name>` – Creates (if needed) and starts a new container registered under the provided CLI name. Running the same name again reuses the existing container instead of spawning a duplicate.
- `convoy stop <name|id>` – Stops and removes the container identified by name or ID. Use `-a`/`--all` to stop and remove every tracked container.
- `convoy list` – Lists all containers managed by Convoy along with their CLI name, image, and agent endpoint.

## Contributing
Contributions welcome! See LICENSE for details.
