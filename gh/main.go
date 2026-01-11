package main

import (
	"context"
	"fmt"
	"time"

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
}

func (m *Gh) Release() *Release {
	return &Release{Gh: m}
}

func (m *Release) Create(ctx context.Context,
	tag,
	repo string,
	// +optional
	assets []*dagger.File,
) error {
	arg := fmt.Sprintf("-R=%s", repo)

	release := m.Gh.Container.
		WithEnvVariable("BUST", time.Now().String()).
		WithExec([]string{"gh", "release", arg, "create", tag, "--generate-notes", "--draft"})

	for _, asset := range assets {
		file, err := asset.Name(ctx)
		if err != nil {
			return err
		}

		release = release.
			WithFile(
				file,
				asset,
			).
			WithExec([]string{
				"gh", "release", arg, "upload", tag, file,
			})
	}

	if _, err := release.
		WithExec([]string{"gh", "release", arg, "edit", tag, "--latest", "--draft=false"}).
		Sync(ctx); err != nil {
		return err
	}

	return nil
}
