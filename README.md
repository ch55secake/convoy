# convoy
🚢 Manage multiple containers and multiple tasks at once

Convoy is a Go-based CLI tool for orchestrating multiple Alpine Linux containers via Docker. It uses gRPC for communication, supervisord for process management, and round-robin load balancing to distribute tasks evenly.

## Features
- **Container Orchestration**: Create, manage, and stop multiple Alpine Linux containers.
- **Command Execution**: Send commands to individual containers or all at once via gRPC over TCP.
- **Load Balancing**: Evenly distribute tasks across containers using a round-robin algorithm.
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

### Available commands
- `convoy start <name>` – Creates (if needed) and starts a new container registered under the provided CLI name. Running the same name again reuses the existing container instead of spawning a duplicate.
- `convoy stop <name|id>` – Stops and removes the container identified by name or ID. Use `-a`/`--all` to stop and remove every tracked container.
- `convoy list` – Lists all containers managed by Convoy along with their CLI name, image, and agent endpoint.
- `convoy config` - Show, validate or initialize Convoy configuration.
- `convoy health` - Check if Convoy is running and healthy.  Use `-a`/`--all` to see the health status of every tracked container. 

## Image Setup
Convoy uses a custom Alpine Linux image with a pre-configured supervisor process to manage gRPC servers. To build the image, run:
```bash
make build-image
```
By default, the image is tagged as `convoy:latest`. The image currently only containers `opencode` and the packages to run it. 
However, the image being used can be changed in the configuration file.

## Contributing
Contributions welcome! See LICENSE for details.
