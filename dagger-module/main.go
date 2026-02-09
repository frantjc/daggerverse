package main

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/dagger/dagger/core/modules"
	"github.com/dagger/dagger/engine"
	"github.com/frantjc/daggerverse/toolchains/dagger-module/internal/dagger"
)

type DaggerModule struct {
	// +private
	Source *dagger.Directory
	// +private
	Path string
}

func New(
	source *dagger.Directory,
	// +default="."
	path string,
) *DaggerModule {
	return &DaggerModule{
		Source: source,
		Path: path,
	}
}

func (m *DaggerModule) Bump(
	ctx context.Context,
	// +optional
	version string,
) (*dagger.Changeset, error) {
	daggerJSONPath := filepath.Join(m.Path, modules.Filename)

	daggerJSON, err := m.Source.File(daggerJSONPath).Contents(ctx)
	if err != nil {
		return nil, err
	}

	if version == "" {
		version, err = dag.Version(ctx)
		if err != nil {
			return nil, err
		}
	}

	engine.Version = version

	moduleConfig, err := modules.ParseModuleConfig([]byte(daggerJSON))
	if err != nil {
		return nil, err
	}

	moduleConfig.EngineVersion = version

	updatedDaggerJSON, err := json.MarshalIndent(moduleConfig, "", "  ")
	if err != nil {
		return nil, err
	}

	container := dag.Wolfi().Container()

	isGo := moduleConfig.SDK != nil && moduleConfig.SDK.Source == "go"

	if isGo {
		container = dag.Go(dagger.GoOpts{
			GoMod: m.Source.File(filepath.Join(m.Path, moduleConfig.Source, "go.mod")),
		}).
			Container()
	}

	container = container.
		WithEnvVariable("_EXPERIMENTAL_DAGGER_CLI_BIN", "/usr/bin/dagger").
		WithFile("$_EXPERIMENTAL_DAGGER_CLI_BIN", dag.Dagger().Binary(), dagger.ContainerWithFileOpts{Expand: true}).
		WithWorkdir(filepath.Join("/src", m.Path)).
		WithMountedDirectory("/src", m.Source).
		WithFile(
			modules.Filename,
			dag.File(modules.Filename, string(updatedDaggerJSON)+"\n"),
		)

	for _, dependency := range moduleConfig.Dependencies {
		if strings.HasPrefix(dependency.Source, "github.com/dagger/dagger") {
			module, _, _ := strings.Cut(dependency.Source, "@")
			container = container.WithExec(
				[]string{"dagger", "install", fmt.Sprintf("%s@%s", module, version)},
				dagger.ContainerWithExecOpts{ExperimentalPrivilegedNesting: true},
			)
		}
	}

	if isGo {
		container = container.
			WithExec(
				[]string{"dagger", "develop"},
				dagger.ContainerWithExecOpts{ExperimentalPrivilegedNesting: true},
			).
			WithWorkdir(filepath.Join("/src", m.Path, moduleConfig.Source)).
			WithExec([]string{"go", "mod", "tidy"})
	}

	return container.
		Directory("/src").
		Changes(m.Source), nil
}
