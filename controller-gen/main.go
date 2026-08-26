package main

import (
	"context"
	"fmt"

	"golang.org/x/mod/modfile"
	"github.com/frantjc/daggerverse/go/internal/dagger"
)

type ControllerGen struct {
	Container *dagger.Container
}

func New(
	ctx context.Context,
	workspace *dagger.Workspace,
	// +optional
	// +default="."
	path string,
	// +optional
	container *dagger.Container,
	// +optional
	// +default="v0.21.0"
	version string,
) (*ControllerGen, error) {
	if container == nil {
		container = dag.Go(workspace, dagger.GoOpts{
			Path:      path,
		}).
			Container()

		goMod := container.File("go.mod")

		goModContents, err := goMod.Contents(ctx)
		if err != nil {
			return nil, err
		}

		parsedGoMod, err := modfile.Parse("go.mod", []byte(goModContents), nil)
		if err != nil {
			return nil, err
		}

		for _, require := range parsedGoMod.Require {
			if require.Mod.Path == "sigs.k8s.io/controller-tools" {
				return &ControllerGen{
					Container: container.
						WithExec([]string{"go", "install", "sigs.k8s.io/controller-tools/cmd/controller-gen"}),
				}, nil
			}
		}

		return &ControllerGen{
			Container: container.
				WithExec([]string{"go", "install", fmt.Sprintf("sigs.k8s.io/controller-tools/cmd/controller-gen@%s", version)}),
		}, nil
	}

	return &ControllerGen{
		Container: container.
			WithWorkdir("/src").
			WithMountedDirectory(".", workspace.Directory(path)),
	}, nil
}

// +generate
func (c *ControllerGen) Object(
	ctx context.Context,
	// +optional
	// +default=["./..."]
	paths []string,
) *dagger.Changeset {
	args := []string{"controller-gen", "object"}
	for _, path := range paths {
		args = append(args, fmt.Sprintf("paths=%s", path))
	}
	return c.Container.
		WithExec(args).
		Directory(".").
		Changes(c.Container.Directory("."))
}

// +generate
func (c *ControllerGen) RBAC(
	ctx context.Context,
	// +optional
	// +default=["./..."]
	paths []string,
	// +optional
	// +default="manager"
	roleName string,
	// +optional
	// +default="config/rbac"
	output string,
) *dagger.Changeset {
	args := []string{"controller-gen", fmt.Sprintf("rbac:roleName=%s", roleName), fmt.Sprintf("output:rbac:artifacts:config=%s", output)}
	for _, path := range paths {
		args = append(args, fmt.Sprintf("paths=%s", path))
	}
	return c.Container.
		WithExec(args).
		Directory(".").
		Changes(c.Container.Directory("."))
}

// +generate
func (c *ControllerGen) CRD(
	ctx context.Context,
	workspace *dagger.Workspace,
	// +optional
	// +default=["./..."]
	paths []string,
	// +optional
	// +default="config/crd"
	output string,
) *dagger.Changeset {
	args := []string{"controller-gen", "crd", fmt.Sprintf("output:crd:artifacts:config=%s", output)}
	for _, path := range paths {
		args = append(args, fmt.Sprintf("paths=%s", path))
	}
	return c.Container.
		WithExec(args).
		Directory(".").
		Changes(c.Container.Directory("."))
}
