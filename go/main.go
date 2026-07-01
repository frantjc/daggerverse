// A generated module for Go functions

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
	Container *dagger.Container
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
	source := workspace.Directory(path)

	goMod := source.File("go.mod")

	goModContents, err := goMod.Contents(ctx)
	if err != nil {
		return nil, err
	}

	parsedGoMod, err := modfile.Parse("go.mod", []byte(goModContents), nil)
	if err != nil {
		return nil, err
	}

	if container == nil {
		container = dag.Container().From(fmt.Sprintf("docker.io/library/golang:%s", parsedGoMod.Go.Version))
	}

	withEnvVariableIfUnset := func(c *dagger.Container, name, value string) (*dagger.Container, error) {
		existing, err := c.EnvVariable(ctx, name)
		if err != nil {
			return nil, err
		}
		if existing == "" {
			return c.WithEnvVariable(name, value, dagger.ContainerWithEnvVariableOpts{
				Expand: true,
			}), nil
		}
		return c, nil
	}

	container, err = withEnvVariableIfUnset(container, "GOPATH", "$HOME/go")
	if err != nil {
		return nil, err
	}
	container, err = withEnvVariableIfUnset(container, "GOBIN", "$GOPATH/bin")
	if err != nil {
		return nil, err
	}
	container = container.WithEnvVariable("PATH", "$GOBIN:$PATH", dagger.ContainerWithEnvVariableOpts{
		Expand: true,
	})
	container, err = withEnvVariableIfUnset(container, "GOMODCACHE", "$GOPATH/pkg/mod")
	if err != nil {
		return nil, err
	}
	container = container.WithMountedCache("$GOMODCACHE", dag.CacheVolume("go-mod-cache"), dagger.ContainerWithMountedCacheOpts{Expand: true})
	container, err = withEnvVariableIfUnset(container, "GOCACHE", "$GOPATH/build")
	if err != nil {
		return nil, err
	}
	container = container.WithMountedCache("$GOCACHE", dag.CacheVolume("go-cache"), dagger.ContainerWithMountedCacheOpts{Expand: true})

	m := &Go{container}

	m.Container = m.Container.
		WithWorkdir(filepath.Join("$GOPATH/src", parsedGoMod.Module.Mod.Path), dagger.ContainerWithWorkdirOpts{Expand: true}).
		WithMountedDirectory(".", source).
		WithExec([]string{"go", "mod", "download"})

	return m, nil
}

func (m *Go) Build(
	ctx context.Context,
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
	output := fmt.Sprintf("$GOBIN/%s", filepath.Base(pkg))
	if filepath.Ext(pkg) == ".go" {
		output = fmt.Sprintf("$GOBIN/%s", filepath.Base(filepath.Dir(pkg)))
	} else if pkg == "./" || pkg == "." {
		workdir, err := m.Container.Workdir(ctx)
		if err != nil {
			return nil, err
		}
		output = fmt.Sprintf("$GOBIN/%s", filepath.Base(workdir))
	}

	return m.Container.
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
	_, err := m.Container.
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
	// +optional
	// +default="./..."
	pkg string,
) *dagger.Changeset {
	src := m.Container.
		Directory(".")
	return m.Container.
		WithExec([]string{"go", "fmt", pkg}).
		Directory(".").
		Changes(src)
}

// +check
func (m *Go) Vulncheck(
	ctx context.Context,
	// +optional
	// +default="latest"
	version string,
) error {
	_, err := m.Container.
		WithExec([]string{"go", "install", fmt.Sprintf("golang.org/x/vuln/cmd/govulncheck@%s", version)}).
		WithExec([]string{"govulncheck", "./..."}).
		Sync(ctx)
	return err
}

// +check
func (m *Go) Vet(ctx context.Context) error {
	_, err := m.Container.
		WithExec([]string{"go", "vet", "./..."}).
		Sync(ctx)
	return err
}

// +check
func (m *Go) Staticcheck(
	ctx context.Context,
	// +optional
	// +default="latest"
	version string,
) error {
	_, err := m.Container.
		WithExec([]string{"go", "install", fmt.Sprintf("honnef.co/go/tools/cmd/staticcheck@%s", version)}).
		WithExec([]string{"staticcheck", "./..."}).
		Sync(ctx)
	return err
}

// +generate
func (m *Go) Tidy(ctx context.Context) *dagger.Changeset {
	return m.Container.
		WithExec([]string{"go", "mod", "tidy"}).
		Directory(".").
		Changes(m.Container.Directory("."))
}
