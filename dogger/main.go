// A generated module for Dogger functions

package main

import (
	"fmt"

	"github.com/frantjc/daggerverse/dogger/internal/dagger"
)

type Dogger struct {
	Port int
}

func New(
	// +default=1331
	port int,
) *Dogger {
	return &Dogger{
		Port: port,
	}
}

func (m *Dogger) AsService() *dagger.Service {
	return dag.Wolfi().
		Container().
		WithFile(
			"/usr/local/bin/dogger",
			dag.Go(dagger.GoOpts{
				Module: dag.CurrentModule().Source().Directory("internal/dogger"),
				AdditionalWolfiPackages: []string{"gcc"},
			}).
				Build(dagger.GoBuildOpts{
					Pkg: "./cmd/dogger",
					Cgo: true,
				}),
		).
		WithExposedPort(m.Port).
		AsService(dagger.ContainerAsServiceOpts{
			ExperimentalPrivilegedNesting: true,
			Args: []string{"dogger", fmt.Sprintf("--addr=:%d", m.Port)},
		})
}

func (m *Dogger) DockerHost(alias string) string {
	return fmt.Sprintf("%s:%d", alias, m.Port)
}

func (m *Dogger) BoundTo(
	container *dagger.Container,
	// +default="dogger"
	alias string,
	// +optional
	installDockerCli bool,
) *dagger.Container {
	return dag.
		Container().
		From("docker:cli").
		WithServiceBinding(alias, m.AsService()).
		WithEnvVariable("DOCKER_HOST", m.DockerHost(alias)).
		With(func(c *dagger.Container) *dagger.Container {
			if installDockerCli {
				return c.WithFile(
					"/usr/local/bin/docker",
					dag.Container().
						From("docker:cli").
						File("/usr/local/bin/docker"),
				)
			}

			return c
		}).
		Terminal(dagger.ContainerTerminalOpts{
			ExperimentalPrivilegedNesting: true,
		})
}

func (m *Dogger) Playground() *dagger.Container {
	return m.BoundTo(dag.Container().From("docker:cli"), "dogger", false)
}
