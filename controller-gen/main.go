package main

import (
	"context"
	"fmt"

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
) *ControllerGen {
	if container == nil {
		container = dag.Go(dagger.GoOpts{
			Workspace: workspace,
			Path:      path,
		}).
			Container().
			WithExec([]string{"go install", "sigs.k8s.io/controller-tools/cmd/controller-gen"})
	}
	return &ControllerGen{container}
}

// +generate
func (c *ControllerGen) Object(
	ctx context.Context,
	workspace *dagger.Workspace,
	// +optional
	// +default=["./..."]
	paths []string,
) *dagger.Changeset {
	args := []string{"controller-gen", "object"}
	for _, path := range paths {
		args = append(args, fmt.Sprintf("paths=%s", path))
	}
	src := workspace.Directory(".")
	return c.Container.
		WithMountedDirectory(".", src).
		WithExec(args).
		Directory(".").
		Changes(src)
}

// +generate
func (c *ControllerGen) RBAC(
	ctx context.Context,
	workspace *dagger.Workspace,
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
	args := []string{"controller-gen", "rbac", fmt.Sprintf("rbac:roleName=%s", roleName), fmt.Sprintf("output:rbac:artifacts:config=%s", output)}
	for _, path := range paths {
		args = append(args, fmt.Sprintf("paths=%s", path))
	}
	src := workspace.Directory(".")
	return c.Container.
		WithMountedDirectory(".", src).
		WithExec(args).
		Directory(".").
		Changes(src)
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
	src := workspace.Directory(".")
	return c.Container.
		WithMountedDirectory(".", src).
		WithExec(args).
		Directory(".").
		Changes(src)
}
