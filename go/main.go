// A generated module for Go functions

package main

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/frantjc/daggerverse/go/internal/dagger"
	"golang.org/x/mod/modfile"
)

type Go struct {
	Source *dagger.Directory
}

func New(
	// +optional
	// +defaultPath="."
	src *dagger.Directory,
) *Go {
	return &Go{
		Source: src,
	}
}

type GoBuild struct {
	Output *dagger.File
}

// Returns a container that echoes whatever string argument is provided
func (m *Go) Build(ctx context.Context, pkg string) (*GoBuild, error) {
	contents, err := m.Source.File("go.mod").Contents(ctx)
	if err != nil {
		return nil, err
	}

	gomod, err := modfile.Parse("go.mod", []byte(contents), nil)
	if err != nil {
		return nil, err
	}

	workdir := path.Join("$GOPATH/src", gomod.Module.Mod.Path)

	outputPath := "$GOPATH/bin/output"

	dotPatchIndex := strings.LastIndex(gomod.Go.Version, ".")
	if dotPatchIndex == -1 {
		return nil, fmt.Errorf("invalid go version %s", gomod.Go.Version)
	}

	return &GoBuild{
		Output: dag.Wolfi().
			Container(dagger.WolfiContainerOpts{
				Packages: []string{fmt.Sprintf("go-%s", gomod.Go.Version[:dotPatchIndex])},
			}).
			WithEnvVariable("CGO_ENABLED", "0").
			WithEnvVariable("GOMODCACHE", "$GOPATH/pkg/mod", dagger.ContainerWithEnvVariableOpts{Expand: true}).
			WithMountedCache("$GOMODCACHE", dag.CacheVolume("go-mod-cache"), dagger.ContainerWithMountedCacheOpts{Expand: true}).
			WithEnvVariable("GOCACHE", "$GOPATH/build", dagger.ContainerWithEnvVariableOpts{Expand: true}).
			WithMountedCache("$GOCACHE", dag.CacheVolume("go-cache"), dagger.ContainerWithMountedCacheOpts{Expand: true}).
			WithDirectory(workdir, m.Source, dagger.ContainerWithDirectoryOpts{Expand: true}).
			WithWorkdir(workdir, dagger.ContainerWithWorkdirOpts{Expand: true}).
			WithExec([]string{"go", "build", "-o", outputPath, pkg}, dagger.ContainerWithExecOpts{Expand: true}).
			File(outputPath, dagger.ContainerFileOpts{Expand: true}),
	}, nil
}
