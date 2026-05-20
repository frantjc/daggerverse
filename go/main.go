// A generated module for Go functions

package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/frantjc/daggerverse/go/internal/dagger"
	xstrings "github.com/frantjc/x/strings"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

type Go struct {
	Container *dagger.Container
}

func New(
	ctx context.Context,
	// +optional
	// +defaultPath="."
	source *dagger.Directory,
	// +optional
	version string,
	// +optional
	additionalWolfiPackages []string,
	// +optional
	container *dagger.Container,
) (*Go, error) {
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
		if version == "" {
			version = parsedGoMod.Go.Version
		}

		version = xstrings.EnsurePrefix(version, "v")
		majorMinor := semver.MajorMinor(version)
		if majorMinor == "" {
			majorMinor = strings.TrimPrefix(version, "v")
		} else {
			majorMinor = strings.TrimPrefix(majorMinor, "v")
		}

		container = dag.Wolfi().
			Container(dagger.WolfiContainerOpts{
				Packages: append([]string{"go-" + majorMinor}, additionalWolfiPackages...),
			}).
			WithEnvVariable("GOPATH", "/go").
			WithEnvVariable("GOBIN", "$GOPATH/bin", dagger.ContainerWithEnvVariableOpts{Expand: true}).
			WithEnvVariable("PATH", "$GOBIN:$PATH", dagger.ContainerWithEnvVariableOpts{Expand: true}).
			WithEnvVariable("GOMODCACHE", "$GOPATH/pkg/mod", dagger.ContainerWithEnvVariableOpts{Expand: true}).
			WithMountedCache("$GOMODCACHE", dag.CacheVolume("go-mod-cache"), dagger.ContainerWithMountedCacheOpts{Expand: true}).
			WithEnvVariable("GOCACHE", "$GOPATH/build", dagger.ContainerWithEnvVariableOpts{Expand: true}).
			WithMountedCache("$GOCACHE", dag.CacheVolume("go-cache"), dagger.ContainerWithMountedCacheOpts{Expand: true})
	} else if len(additionalWolfiPackages) != 0 {
		return nil, fmt.Errorf("cannot set both additionalWolfiPackages and container")
	}

	m := &Go{
		Container: container,
	}

	if source == nil {
		m.Container = m.Container.WithWorkdir("$GOPATH", dagger.ContainerWithWorkdirOpts{Expand: true})
		return m, nil
	}

	m.Container =  m.Container.
		WithWorkdir(filepath.Join("$GOPATH/src", parsedGoMod.Module.Mod.Path), dagger.ContainerWithWorkdirOpts{Expand: true}).
		WithMountedDirectory(".", source).
		WithExec([]string{"go", "mod", "download"})
	
	return m, nil
}

func (m *Go) WithServiceBinding(alias string, service *dagger.Service) *Go {
	return &Go{Container: m.Container.WithServiceBinding(alias, service)}
}

func (m *Go) WithFile(path string, source *dagger.File) *Go {
	return &Go{Container: m.Container.WithFile(path, source)}
}

func (m *Go) WithEnvVariable(name, value string) *Go {
	return &Go{Container: m.Container.WithEnvVariable(name, value)}
}

func (m *Go) WithExec(args []string) *Go {
	return &Go{Container: m.Container.WithExec(args)}
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
		WithExec([]string{"go", "build", "-trimpath", "-ldflags="+ldflags, "-o", output, pkg}, dagger.ContainerWithExecOpts{Expand: true}).
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
	args := []string{"go", "test"}
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

func (m *Go) UpgradeDirect(ctx context.Context) *dagger.Changeset {
	return m.Container.
		WithExec([]string{"go", "get", "-u", "./..."}).
		Directory(".").
		Changes(m.Container.Directory("."))
}

func (m *Go) UpgradeTransitive(ctx context.Context) *dagger.Changeset {
	return m.Container.
		WithExec([]string{"go", "get", "-ut", "./..."}).
		Directory(".").
		Changes(m.Container.Directory("."))
}
