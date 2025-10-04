// A generated module for Go functions

package main

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/frantjc/daggerverse/go/internal/dagger"
	xstrings "github.com/frantjc/x/strings"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

type Go struct {
	Container *dagger.Container
}

const (
	group = "go"
	user  = group
	home  = "/home/" + user
)

func New(
	ctx context.Context,
	// +optional
	module *dagger.Directory,
	// +optional
	goMod *dagger.File,
	// +optional
	version string,
) (*Go, error) {
	if module != nil {
		goMod = module.File("go.mod")
	}

	if goMod != nil {
		goModContents, err := goMod.Contents(ctx)
		if err != nil {
			return nil, err
		}

		parsedGoMod, err := modfile.Parse("go.mod", []byte(goModContents), nil)
		if err != nil {
			return nil, err
		}

		version = parsedGoMod.Go.Version
	}

	if version == "" {
		return nil, fmt.Errorf("one of module, go-mod, or version is required")
	}

	version = xstrings.EnsurePrefix(version, "v")
	majorMinor := semver.MajorMinor(version)
	if majorMinor == "" {
		majorMinor = strings.TrimPrefix(version, "v")
	} else {
		majorMinor = strings.TrimPrefix(majorMinor, "v")
	}

	m := &Go{
		Container: dag.Wolfi().
			Container(dagger.WolfiContainerOpts{
				Packages: []string{"go-" + majorMinor},
			}).
			WithEnvVariable("HOME", home).
			WithEnvVariable("GOPATH", "$HOME", dagger.ContainerWithEnvVariableOpts{Expand: true}).
			WithEnvVariable("GOBIN", "$GOPATH/bin", dagger.ContainerWithEnvVariableOpts{Expand: true}).
			WithEnvVariable("PATH", "$GOBIN:$PATH", dagger.ContainerWithEnvVariableOpts{Expand: true}).
			WithEnvVariable("GOMODCACHE", "$GOPATH/pkg/mod", dagger.ContainerWithEnvVariableOpts{Expand: true}).
			WithMountedCache("$GOMODCACHE", dag.CacheVolume("go-mod-cache"), dagger.ContainerWithMountedCacheOpts{Expand: true}),
	}

	if module == nil {
		return m, nil
	}

	return m.WithSource(ctx, module)
}

func (m *Go) WithSource(
	ctx context.Context,
	// +optional
	// +defaultPath="."
	source *dagger.Directory,
) (*Go, error) {
	goModContents, err := source.File("go.mod").Contents(ctx)
	if err != nil {
		return nil, err
	}

	parsedGoMod, err := modfile.Parse("go.mod", []byte(goModContents), nil)
	if err != nil {
		return nil, err
	}

	return &Go{
		Container: m.Container.
			WithWorkdir(path.Join("$GOPATH/src", parsedGoMod.Module.Mod.Path), dagger.ContainerWithWorkdirOpts{Expand: true}).
			WithMountedDirectory(".", source),
	}, nil
}

func (m *Go) Build(
	// +optional
	// +default="./"
	pkg string,
	// +optional
	// +default="-s -w"
	ldflags string,
) (*dagger.File, error) {
	outputPath := "$GOPATH/bin/output"

	return m.Container.
		WithEnvVariable("CGO_ENABLED", "0").
		WithEnvVariable("GOCACHE", "$GOPATH/build", dagger.ContainerWithEnvVariableOpts{Expand: true}).
		WithMountedCache("$GOCACHE", dag.CacheVolume("go-cache"), dagger.ContainerWithMountedCacheOpts{Expand: true}).
		WithExec([]string{"go", "build", "-trimpath", "-ldflags="+ldflags, "-o", outputPath, pkg}, dagger.ContainerWithExecOpts{Expand: true}).
		File(outputPath, dagger.ContainerFileOpts{Expand: true}), nil
}

func (m *Go) Test(
	// +optional
	// +default="./..."
	pkg string,
) (*dagger.Container, error) {
	return m.Container.
		WithExec([]string{"go", "test", "-race", "-cover", pkg}), nil
}
