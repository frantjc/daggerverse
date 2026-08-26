package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/frantjc/daggerverse/go/internal/dagger"
	"golang.org/x/mod/modfile"
)

type Go struct {
	// +private
	Ctr *dagger.Container
}

func parseGoModFrom(ctx context.Context, goMod *dagger.File) (*modfile.File, error) {
	goModContents, err := goMod.Contents(ctx)
	if err != nil {
		return nil, err
	}

	return modfile.Parse("go.mod", []byte(goModContents), nil)
}

func New(
	ctx context.Context,
	workspace *dagger.Workspace,
	// +optional
	// +default="."
	path string,
	// +optional
	container *dagger.Container,
) (*Go, error) {
	src := workspace.Directory(path, dagger.WorkspaceDirectoryOpts{
		Include: []string{"**/go.mod", "**/go.sum"},
	})

	parsedGoMod, err := parseGoModFrom(ctx, src.File("go.mod"))
	if err != nil {
		return nil, err
	}

	if container == nil {
		container = dag.Container().From(fmt.Sprintf("docker.io/library/golang:%s", parsedGoMod.Go.Version))
	}

	return &Go{
		Ctr: container.
			With(func(r *dagger.Container) *dagger.Container {
				if goPath, err := r.EnvVariable(ctx, "GOPATH"); err != nil || goPath != "" {
					return r
				}

				return r.WithEnvVariable("GOPATH", "$HOME/go", dagger.ContainerWithEnvVariableOpts{
					Expand: true,
				})
			}).
			With(func(r *dagger.Container) *dagger.Container {
				if goBin, err := r.EnvVariable(ctx, "GOBIN"); err != nil || goBin != "" {
					return r
				}

				return r.WithEnvVariable("GOBIN", "$GOPATH/bin", dagger.ContainerWithEnvVariableOpts{
					Expand: true,
				})
			}).
			WithEnvVariable("PATH", "$GOBIN:$PATH", dagger.ContainerWithEnvVariableOpts{
				Expand: true,
			}).
			With(func(r *dagger.Container) *dagger.Container {
				if goModCache, err := r.EnvVariable(ctx, "GOMODCACHE"); err != nil || goModCache != "" {
					return r
				}

				return r.WithEnvVariable("GOMODCACHE", "$GOPATH/pkg/mod", dagger.ContainerWithEnvVariableOpts{
					Expand: true,
				})
			}).
			WithMountedCache("$GOMODCACHE", dag.CacheVolume("go-mod-cache"), dagger.ContainerWithMountedCacheOpts{
				Expand: true,
			}).
			With(func(r *dagger.Container) *dagger.Container {
				if goCache, err := r.EnvVariable(ctx, "GOCACHE"); err != nil || goCache != "" {
					return r
				}

				return r.WithEnvVariable("GOCACHE", "$GOPATH/build", dagger.ContainerWithEnvVariableOpts{
					Expand: true,
				})
			}).
			WithMountedCache("$GOCACHE", dag.CacheVolume("go-cache"), dagger.ContainerWithMountedCacheOpts{
				Expand: true,
			}).
			With(withParsedGoMod(parsedGoMod)).
			WithMountedDirectory(".", src).
			WithExec([]string{"go", "mod", "download"}).
			WithoutMount(".").
			WithoutWorkdir(),
	}, nil
}

func withParsedGoMod(parsedGoMod *modfile.File) func(c *dagger.Container) *dagger.Container {
	return func(c *dagger.Container) *dagger.Container {

		return c.WithWorkdir(filepath.Join("$GOPATH/src", parsedGoMod.Module.Mod.Path), dagger.ContainerWithWorkdirOpts{Expand: true})
	}
}

func (m *Go) Build(
	ctx context.Context,
	workspace *dagger.Workspace,
	// +optional
	// +default="."
	path string,
	// +optional
	// +default="./"
	pkg string,
	// +optional
	// +default="-s -w"
	ldflags string,
	// +optional
	cgo bool,
	// +optional
	goarch string,
	// +optional
	goos string,
) (*dagger.File, error) {
	container, err := m.Container(ctx, workspace, path)
	if err != nil {
		return nil, err
	}

	output := fmt.Sprintf("$GOBIN/%s", filepath.Base(pkg))
	if filepath.Ext(pkg) == ".go" {
		output = fmt.Sprintf("$GOBIN/%s", filepath.Base(filepath.Dir(pkg)))
	} else if pkg == "./" || pkg == "." {
		workdir, err := container.Workdir(ctx)
		if err != nil {
			return nil, err
		}
		output = fmt.Sprintf("$GOBIN/%s", filepath.Base(workdir))
	}

	return container.
		With(func(c *dagger.Container) *dagger.Container {
			if !cgo {
				return c.WithEnvVariable("CGO_ENABLED", "0")
			}
			return c
		}).
		With(func(c *dagger.Container) *dagger.Container {
			if goos != "" {
				c = c.WithEnvVariable("GOOS", goos)
			}
			if goarch != "" {
				c = c.WithEnvVariable("GOARCH", goarch)
			}
			return c
		}).
		WithExec([]string{"go", "build", "-trimpath", "-ldflags=" + ldflags, "-o", output, pkg}, dagger.ContainerWithExecOpts{Expand: true}).
		File(output, dagger.ContainerFileOpts{Expand: true}), nil
}

// +check
func (m *Go) Test(
	ctx context.Context,
	workspace *dagger.Workspace,
	// +optional
	// +default="."
	path string,
	// +optional
	// +default="./..."
	pkg string,
	// +optional
	race bool,
	// +optional
	cgo bool,
	// +optional
	tags []string,
) error {
	args := []string{"otelgotest"}
	if race {
		args = append(args, "-race")
	}
	if len(tags) > 0 {
		args = append(args, "-tags", strings.Join(tags, ","))
	}
	args = append(args, pkg)

	container, err := m.Container(ctx, workspace, path)
	if err != nil {
		return err
	}

	_, err = container.
		With(func(c *dagger.Container) *dagger.Container {
			if cgo || race {
				return c.WithEnvVariable("CGO_ENABLED", "1")
			}
			return c.WithEnvVariable("CGO_ENABLED", "0")
		}).
		WithExec([]string{"go", "install", "github.com/dagger/otel-go/cmd/otelgotest@main"}).
		WithExec(args, dagger.ContainerWithExecOpts{ExperimentalPrivilegedNesting: true}).
		Sync(ctx)
	return err
}

// +generate
func (m *Go) Fmt(
	ctx context.Context,
	workspace *dagger.Workspace,
	// +optional
	// +default="."
	path string,
	// +optional
	// +default="./..."
	pkg string,
) (*dagger.Changeset, error) {
	container, src, err := m.containerAndSrc(ctx, workspace, path)
	if err != nil {
		return nil, err
	}

	return container.
		WithExec([]string{"go", "fmt", pkg}).
		Directory(".").
		Changes(src), nil
}

// +check
func (m *Go) Vulncheck(
	ctx context.Context,
	workspace *dagger.Workspace,
	// +optional
	// +default="."
	path string,
	// +optional
	// +default="./..."
	pkg string,
	// +optional
	// +default="latest"
	version string,
) error {
	container, err := m.Container(ctx, workspace, path)
	if err != nil {
		return err
	}

	_, err = container.
		WithExec([]string{"go", "install", fmt.Sprintf("golang.org/x/vuln/cmd/govulncheck@%s", version)}).
		WithExec([]string{"govulncheck", pkg}).
		Sync(ctx)
	return err
}

// +check
func (m *Go) Vet(
	ctx context.Context,
	workspace *dagger.Workspace,
	// +optional
	// +default="."
	path string,
	// +optional
	// +default="./..."
	pkg string,
) error {
	container, err := m.Container(ctx, workspace, path)
	if err != nil {
		return err
	}

	_, err = container.
		WithExec([]string{"go", "vet", pkg}).
		Sync(ctx)
	return err
}

// +check
func (m *Go) Staticcheck(
	ctx context.Context,
	workspace *dagger.Workspace,
	// +optional
	// +default="."
	path string,
	// +optional
	// +default="./..."
	pkg string,
	// +optional
	// +default="latest"
	version string,
) error {
	container, err := m.Container(ctx, workspace, path)
	if err != nil {
		return err
	}

	_, err = container.
		WithExec([]string{"go", "install", fmt.Sprintf("honnef.co/go/tools/cmd/staticcheck@%s", version)}).
		WithExec([]string{"staticcheck", pkg}).
		Sync(ctx)
	return err
}

// +generate
func (m *Go) Tidy(
	ctx context.Context,
	workspace *dagger.Workspace,
	// +optional
	// +default="."
	path string,
) (*dagger.Changeset, error) {
	container, src, err := m.containerAndSrc(ctx, workspace, path)
	if err != nil {
		return nil, err
	}

	return container.
		WithExec([]string{"go", "mod", "tidy"}).
		Directory(".").
		Changes(src), nil
}

// +generate
func (m *Go) Generate(
	ctx context.Context,
	workspace *dagger.Workspace,
	// +optional
	// +default="."
	path string,
	// +optional
	// +default="./..."
	pkg string,
) (*dagger.Changeset, error) {
	container, src, err := m.containerAndSrc(ctx, workspace, path)
	if err != nil {
		return nil, err
	}

	return container.
		WithExec([]string{"go", "generate", pkg}, dagger.ContainerWithExecOpts{
			ExperimentalPrivilegedNesting: true,
		}).
		Directory(".").
		Changes(src), nil
}

func (m *Go) Container(
	ctx context.Context,
	workspace *dagger.Workspace,
	// +optional
	// +default="."
	path string,
) (*dagger.Container, error) {
	container, err := m.Container(ctx, workspace, path)
	return container, err
}

// +generate
func (m *Go) containerAndSrc(
	ctx context.Context,
	workspace *dagger.Workspace,
	path string,
) (*dagger.Container, *dagger.Directory, error) {
	src := workspace.Directory(path)

	parsedGoMod, err := parseGoModFrom(ctx, src.File("go.mod"))
	if err != nil {
		return nil, nil, err
	}

	return m.Ctr.
		With(withParsedGoMod(parsedGoMod)).
		WithMountedDirectory(".", src), src, nil
}
