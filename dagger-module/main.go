package main

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/dagger/dagger/core/modules"
	"github.com/dagger/dagger/engine"
	"github.com/frantjc/daggerverse/dagger-module/internal/dagger"
)

type DaggerModule struct {
	Container *dagger.Container
}

func isGo(moduleConfig *modules.ModuleConfigWithUserFields) bool {
	return moduleConfig.SDK != nil && moduleConfig.SDK.Source == "go"
}

func New(
	ctx context.Context,
	// +optional
	// +defaultPath="."
	source *dagger.Directory,
	// +optional
	// +default="."
	path string,
) (*DaggerModule, error) {
	return &DaggerModule{
		Container: dag.Wolfi().
			Container().
			WithEnvVariable("_EXPERIMENTAL_DAGGER_CLI_BIN", "/usr/bin/dagger").
			WithFile("$_EXPERIMENTAL_DAGGER_CLI_BIN", dag.Dagger().Binary(), dagger.ContainerWithFileOpts{Expand: true}).
			WithWorkdir(filepath.Join("/src", path)).
			WithMountedDirectory("/src", source),
	}, nil
}

// +generate
func (m *DaggerModule) Bump(ctx context.Context) (*dagger.Changeset, error) {
	daggerJSON, err := m.Container.File(modules.Filename).Contents(ctx)
	if err != nil {
		return nil, err
	}

	version, err := dag.Version(ctx)
	if err != nil {
		return nil, err
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

	container := m.Container.
		WithFile(
			modules.Filename,
			dag.File(modules.Filename, string(updatedDaggerJSON)+"\n"),
		)

	for _, dependency := range moduleConfig.Dependencies {
		if strings.HasPrefix(dependency.Source, "github.com/dagger/dagger") {
			module, _, _ := strings.Cut(dependency.Source, "@")
			container = container.WithExec(
				[]string{"dagger", "update", fmt.Sprintf("%s@%s", module, version)},
				dagger.ContainerWithExecOpts{ExperimentalPrivilegedNesting: true},
			)
		}
	}

	if isGo(moduleConfig) {
		workdir, err := container.Workdir(ctx)
		if err != nil {
			return nil, err
		}

		goModulePath := filepath.Join(workdir, moduleConfig.Source)

		container = container.WithDirectory(
			goModulePath,
			container.Directory(goModulePath).
				WithChanges(
					dag.Go(dagger.GoOpts{
							Source: container.
								WithExec(
									[]string{"dagger", "develop"},
									dagger.ContainerWithExecOpts{ExperimentalPrivilegedNesting: true},
								).
								Directory(goModulePath),
						}).
							Tidy(),
				),
		)
	}

	return container.
		Directory("/src").
		Changes(m.Container.Directory("/src")), nil
}
