# go

A Dagger module for building, testing, and maintaining Go projects. It sets up a `golang` container for the Go version series from your `go.mod`, with shared module/build caches, and exposes functions for common Go workflows: `build`, `test`, `fmt`, `vet`, `staticcheck`, `vulncheck`, `tidy`, and `generate`. Tools such as `staticcheck` are built apart from your source, so they stay cached across edits.

## use

Build a binary for Linux/arm64:

```sh
dagger api call -m github.com/frantjc/daggerverse/go build --goos linux --goarch arm64 export --path ./myapp
```

`generate` (and the other commands) only mount the files gomod finds to be part of the Go module. If a `go:generate` tool reads other files, such as its own configuration, add them with the `include` setting (patterns are relative to the workspace root):

```toml
[modules.go.settings]
include = ["buf.yaml", "buf.gen.yaml", "buf.lock", "api/proto/**"]
```
