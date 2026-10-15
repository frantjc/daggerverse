# go

A Dagger module for building, testing, and maintaining Go projects. It sets up a Wolfi-based container with the Go version from your `go.mod`, shared module/build caches, and exposes functions for common Go workflows: `build`, `test`, `fmt`, `vet`, `staticcheck`, `vulncheck`, `tidy`, and dependency upgrades.

## use

Build a binary for Linux/arm64:

```sh
dagger api call -m github.com/frantjc/daggerverse/go build --goos linux --goarch arm64 export --path ./myapp
```
