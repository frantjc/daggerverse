package main

import (
	"context"
	"fmt"

	"github.com/frantjc/daggerverse/gh/internal/dagger"
)

type Gh struct {
	Container *dagger.Container
}

func New(githubToken *dagger.Secret) *Gh {
	return &Gh{
		Container: dag.Wolfi().
		Container(dagger.WolfiContainerOpts{
			Packages: []string{"gh"},
		}).
		WithSecretVariable("GITHUB_TOKEN", githubToken),
	}
}

type Release struct {
	// +private
	Gh *Gh
	// +private
	Repo string
	// +private
	Tag string
}

func (m *Gh) Release(repo, tag string) *Release {
	return &Release{Gh: m, Repo: repo, Tag: tag}
}

func (m *Release) Create(ctx context.Context,
	// +optional
	title string,
	// +optional
	generateNotes,
	// +optional
	latest,
	// +optional
	prerelease,
	// +optional
	draft,
	// +optional
	verifyTag bool,
) error {
	args := []string{"create", m.Tag}

	if title != "" {
		args = append(args, fmt.Sprintf("--title=%s", title))
	}

	if generateNotes {
		args = append(args, "--generate-notes")
	}

	if latest {
		args = append(args, "--latest")
	}

	if prerelease {
		args = append(args, "--prerelease")
	}

	if draft {
		args = append(args, "--draft")
	}

	if verifyTag {
		args = append(args, "--verify-tag")
	}

	return m.run(ctx, nil, args...)
}

func (m *Release) Edit(ctx context.Context,
	// +optional
	title string,
	// +optional
	generateNotes,
	// +optional
	latest,
	// +optional
	prerelease,
	// +optional
	draft,
	// +optional
	verifyTag bool,
) error {
	args := []string{"edit", m.Tag}

	if title != "" {
		args = append(args, fmt.Sprintf("--title=%s", title))
	}

	if generateNotes {
		args = append(args, "--generate-notes")
	}

	if latest {
		args = append(args, "--latest")
	}

	if prerelease {
		args = append(args, "--prerelease")
	}

	if draft {
		args = append(args, "--draft")
	}

	if verifyTag {
		args = append(args, "--verify-tag")
	}

	return m.run(ctx, nil, args...)
}

func (m *Release) Upload(ctx context.Context,
	assets []*dagger.File,
	// +optional
	clobber bool,
) error {
	container := m.Gh.Container
	args := []string{"upload", m.Tag}

	if clobber {
		args = append(args, "--clobber")
	}

	for _, asset := range assets {
		name, err := asset.Name(ctx)
		if err != nil {
			return err
		}

		container = container.
			WithFile(
				name,
				asset,
			)

		args = append(args, name)
	}

	return m.run(ctx, container, args...)
}

func (m *Release) run(ctx context.Context, container *dagger.Container, args ...string) error {
	if container == nil {
		container = m.Gh.Container
	}
	if _, err := container.WithExec(append([]string{"gh", "release", fmt.Sprintf("--repo=%s", m.Repo)}, args...)).Sync(ctx); err != nil {
		return err
	}
	return nil
}
