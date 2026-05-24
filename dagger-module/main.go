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
	// +private
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
	container *dagger.Container,
) (*DaggerModule, error) {
	if container == nil {
		container = dag.Wolfi().Container()
	}

	version, err := dag.Version(ctx)
	if err != nil {
		return nil, err
	}

	arch, err := dag.Arch().Oci(ctx)
	if err != nil {
		return nil, err
	}

	daggerTgz := dag.HTTP(
		fmt.Sprintf(
			"https://github.com/dagger/dagger/releases/download/%s/dagger_%s_linux_%s.tar.gz",
			version, version, arch,
		),
	)

	return &DaggerModule{
		Container: container.
			WithEnvVariable("_EXPERIMENTAL_DAGGER_CLI_BIN", "/usr/local/bin/dagger").
			WithFile("$_EXPERIMENTAL_DAGGER_CLI_BIN", dag.Archive().Untar(daggerTgz).File("dagger"), dagger.ContainerWithFileOpts{
				Expand: true,
			}).
			WithWorkdir("/src").
			WithMountedDirectory(".", source),
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

	wd, err := m.Container.Workdir(ctx)
	if err != nil {
		return nil, err
	}

	for _, dependency := range append(moduleConfig.Toolchains, moduleConfig.Dependencies...) {
		if !strings.Contains(dependency.Source, "@") && !strings.HasPrefix(dependency.Source, "..") {
			nwd := filepath.Join(wd, dependency.Source)
			n := &DaggerModule{Container: container.WithWorkdir(nwd)}

			if _, err := n.Bump(ctx); err != nil {
				return nil, err
			}

			container = n.Container.WithWorkdir(wd)
		}

		if strings.HasPrefix(dependency.Source, "github.com/dagger/dagger") {
			module, _, _ := strings.Cut(dependency.Source, "@")
			container = container.
				WithExec(
					[]string{"dagger", "update", fmt.Sprintf("%s@%s", module, version)},
					dagger.ContainerWithExecOpts{
						ExperimentalPrivilegedNesting: true,
					},
				)
		}
	}

	if isGo(moduleConfig) {
		goModulePath := filepath.Join(wd, moduleConfig.Source)

		if exists, err := container.Exists(ctx, goModulePath); err != nil {
			return nil, err
		} else if exists {
			daggerDeveloped := container.
				WithExec(
					[]string{"dagger", "develop"},
					dagger.ContainerWithExecOpts{
						ExperimentalPrivilegedNesting: true,
					},
				).
				Directory(goModulePath)
			daggerGoModuleBumped := dag.Go(dagger.GoOpts{
				Source: daggerDeveloped,
			}).
				Container().
				// NB: This was for Dagger v0.19.11 < version < v0.20.5, which continued to use to SDK v0.19.11.
				// WithExec([]string{"go", "get", "-u", fmt.Sprintf("dagger.io/dagger@%s", version)}).
				WithExec([]string{"go", "get", "-u", fmt.Sprintf("github.com/dagger/dagger@%s", version)}).
				Directory(".")
			container = container.WithDirectory(
				goModulePath,
				container.
					Directory(goModulePath).
					WithChanges(
						dag.Go(dagger.GoOpts{
							Source: daggerGoModuleBumped,
						}).
							Tidy(),
					),
			)
		}
	}

	cs := container.
		Directory("/src").
		Changes(m.Container.Directory("/src"))

	m.Container = container

	return cs, nil
}
