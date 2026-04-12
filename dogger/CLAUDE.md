# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Dogger is a Docker API implementation built on top of Dagger, enabling Docker-in-Dagger workflows without requiring a full Docker daemon. It acts as a Docker-compatible API server that translates Docker API calls into Dagger operations.

## Architecture

### Two-Layer Structure

The project has a nested Dagger module structure:

1. **Outer module** (`/dogger`) - Dagger module that exposes the dogger service
   - `main.go` - Defines the `Dogger` Dagger module with functions like `Service()`, `BoundTo()`, `Playground()`
   - `dagger.json` - Declares dependency on `dogger-dev` (the inner module)

2. **Inner module** (`/internal/dogger`) - The actual dogger implementation
   - `internal/dogger/.dagger/main.go` - The `DoggerDev` Dagger module that builds the binary
   - `cmd/dogger/` - CLI entry point
   - `command/` - Cobra command implementation
   - `internal/backends/` - Docker API backend implementations (container, image, system)
   - `internal/handler/` - HTTP handler setup using Docker's server package
   - `internal/storage/` - Storage interfaces and SQL implementation for container/image metadata
   - `internal/dagutil/` - Dagger client utilities with telemetry integration

### Backend Architecture

Dogger implements Docker API routers using the `github.com/docker/docker` packages:

- **ContainerBackend** - Implements `containerrouter.Backend` interface
  - Translates Docker container operations to Dagger `Container` and `Service` APIs
  - Handles container lifecycle: create, start, stop, inspect, wait, attach
  - Uses OpenTelemetry spans to track container execution and route logs
  - Stores container metadata in an in-memory SQLite database

- **ImageBackend** - Implements `imagerouter.Backend` interface
  - Handles image pull operations by translating to `dag.Container().From()`
  - Most operations are unimplemented stubs

- **SystemBackend** - Implements `systemrouter.Backend` interface
  - Provides system-level Docker API compatibility

### Key Design Patterns

1. **Telemetry-based log routing**: Container logs are exported via OpenTelemetry and filtered by span lineage to route output to the correct client
2. **Service vs Command**: Containers with commands run through `AsService()` and track completion via channels; long-running containers use started services
3. **Name aliasing**: Containers get multiple names (full ID, short ID, user-provided name) stored in the database
4. **Privileged nesting**: Uses Dagger's `ExperimentalPrivilegedNesting` for nested container execution

## Development Commands

### Testing

Run tests:
```bash
cd internal/dogger && go test ./command/...
```

## Module Structure

- **internal/dogger/.dagger/** - Inner Dagger module (`DoggerDev`) that builds the binary
- **Outer root** - Wrapper Dagger module (`Dogger`) that exposes the service
- The outer module depends on the inner via `dagger.json` dependency `dogger-dev`

## Go Module Details

Two separate Go modules:
- `/go.mod` - Outer module (lightweight, just Dagger SDK)
- `/internal/dogger/go.mod` - Inner module with full dependencies (Docker libs, SQLite, OpenTelemetry)

The inner module requires CGO for SQLite (`mattn/go-sqlite3`), so builds use `gcc` and `Cgo: true`.

## Container Execution Model

When a container starts:
1. Storage creates a Container record with metadata (config, host config, platform)
2. A Dagger Container is built from the image with appropriate options (exposed ports, labels, user, workdir, annotations, entrypoint)
3. If the container has a command: it's run via `AsService()` with a goroutine tracking completion to a channel for `ContainerWait()`
4. If it's a long-running service: uses `AsService().Start()` and tracks the endpoint as hostname
5. OpenTelemetry spans created for the execution track logs, which are filtered by span lineage for `ContainerAttach()`

## Important Files

- `internal/dogger/command/dogger.go` - Main server logic, sets up HTTP server with Docker API handlers
- `internal/dogger/internal/backends/container.go` - Core container operation implementations
- `internal/dogger/internal/dagutil/client.go` - Dagger client creation with telemetry plumbing
- `internal/dogger/internal/handler/handler.go` - Wires up Docker API routers
