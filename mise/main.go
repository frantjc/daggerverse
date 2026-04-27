package main

import (
	"context"
	"fmt"

	xslices "github.com/frantjc/x/slices"
	"github.com/logsquaredn/rubber-mm/.dagger/modules/mise/internal/dagger"
)

type Mise struct {
	Container *dagger.Container
}

func New(
	ctx context.Context,
	// +optional
	source *dagger.Directory,
	// +optional
	config *dagger.File,
	// +optional
	tools,
	// +optional
	additionalWolfiPackages []string,
	// +optional
	noEnv,
	// +optional
	noHooks bool,
) (*Mise, error) {
	if source == nil && config == nil {
		return nil, fmt.Errorf("one of source and config is required")
	} else if source == nil {
		source = dag.Directory().
			WithFile("mise.toml", config)
	}

	return &Mise{
		Container: dag.Wolfi().
			Container(dagger.WolfiContainerOpts{
				Packages: xslices.Unique(append(additionalWolfiPackages,
					"curl", // Or wget.
					"git",
				)), 
			}).
			WithFile("/usr/bin/mise-run", dag.HTTP("https://mise.run/"), dagger.ContainerWithFileOpts{
				Permissions: 0755,
			}).
			WithExec([]string{"mise-run"}).
			WithEnvVariable("PATH", "/root/.local/share/mise/shims:/root/.local/bin:$PATH", dagger.ContainerWithEnvVariableOpts{
				Expand: true,
			}).
			WithWorkdir("/src").
			With(func(r *dagger.Container) *dagger.Container {
				if noEnv {
					return r.WithEnvVariable("MISE_NO_ENV", "1")
				}
				return r
			}).
			With(func(r *dagger.Container) *dagger.Container {
				if noHooks {
					return r.WithEnvVariable("MISE_NO_HOOKS", "1")
				}
				return r
			}).
			WithDirectory(".", source).
			WithExec(append([]string{"mise", "--yes", "install"}, tools...)),
	}, nil
}
