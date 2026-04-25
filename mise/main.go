package main

import (
	"context"
	"fmt"

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
	tools []string,
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

	exec := []string{"mise", "--yes"}
	if noEnv {
		exec = append(exec, "--no-env")
	}
	if noHooks {
		exec = append(exec, "--no-hooks")
	}
	exec = append(exec, "install")
	exec = append(exec, tools...)

	return &Mise{
		Container: dag.Wolfi().
			Container(dagger.WolfiContainerOpts{
				Packages: []string{
					"curl", // or wget
					"git",
				}, 
			}).
			WithFile("/usr/bin/mise-run", dag.HTTP("https://mise.run/"), dagger.ContainerWithFileOpts{
				Permissions: 0755,
			}).
			WithExec([]string{"mise-run"}).
			WithEnvVariable("PATH", "/root/.local/share/mise/shims:/root/.local/bin:$PATH", dagger.ContainerWithEnvVariableOpts{
				Expand: true,
			}).
			WithWorkdir("/src").
			WithDirectory(".", source).
			WithExec(exec),
	}, nil
}
