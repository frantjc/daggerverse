// A generated module for Dogger functions

package main

import (
	"context"
	"fmt"

	"github.com/frantjc/daggerverse/dogger/internal/dagger"
	"github.com/google/uuid"
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

func (m *Dogger) Service() *dagger.Service {
	return dag.Wolfi().
		Container().
		WithFile(
			"/usr/local/bin/dogger",
			dag.
				DoggerDev(dagger.DoggerDevOpts{
					Src: dag.CurrentModule().Source().Directory("internal/dogger"),
				}).
				Binary(),
		).
		WithExposedPort(m.Port).
		AsService(dagger.ContainerAsServiceOpts{
			ExperimentalPrivilegedNesting: true,
			Args:                          []string{"dogger", fmt.Sprintf("--addr=:%d", m.Port)},
		})
}

func (m *Dogger) DockerHost(alias string) string {
	return fmt.Sprintf("tcp://%s:%d", alias, m.Port)
}

func (m *Dogger) BoundTo(
	container *dagger.Container,
	// +default="dogger"
	alias string,
	// +optional
	installDockerCli bool,
) *dagger.Container {
	return container.
		WithServiceBinding(alias, m.Service()).
		WithEnvVariable("DOCKER_HOST", m.DockerHost(alias)).
		With(func(c *dagger.Container) *dagger.Container {
			if installDockerCli {
				return c.WithFile(
					"/usr/local/bin/docker",
					dag.Container().
						From("docker.io/library/docker:cli").
						File("/usr/local/bin/docker"),
				)
			}

			return c
		})
}

func (m *Dogger) Playground() *dagger.Container {
	return m.
		BoundTo(dag.Container().From("docker.io/library/docker:cli"), "dogger", false).
		Terminal(dagger.ContainerTerminalOpts{
			ExperimentalPrivilegedNesting: true,
		})
}

// +check
func (m *Dogger) TestStdout(ctx context.Context) error {
	expected := uuid.NewString()

	actual, err := m.BoundTo(
		dag.Container().From("alpine"),
		"dogger",
		true,
	).
		WithExec([]string{"docker", "run", "busybox", "echo", expected}).
		Stdout(ctx)
	if err != nil {
		return err
	} else if expected != actual {
		return fmt.Errorf("expected %s, got %s", expected, actual)
	}

	return nil
}
