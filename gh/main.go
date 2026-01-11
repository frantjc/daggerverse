package main

import (
	"context"
	"fmt"

	"github.com/frantjc/daggerverse/gh/internal/dagger"
)

type Gh struct {
	Container *dagger.Container
}

const (
	group = "gh"
	user  = group
	home  = "/home/" + user
)

func New(githubToken *dagger.Secret) *Gh {
	return &Gh{
		Container: dag.Wolfi().
		Container(dagger.WolfiContainerOpts{
			Packages: []string{"gh"},
		}).
		WithEnvVariable("HOME", home).
		WithWorkdir("$HOME", dagger.ContainerWithWorkdirOpts{Expand: true}).
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
	repoArg := fmt.Sprintf("-R=%s", repo)

	release := m.Gh.Container.WithExec([]string{"gh", "release", repoArg, "create", tag, "--generate-notes", "--draft"})

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
				"gh", "release", repoArg, "upload", tag, file,
			})
	}

	if _, err := release.
		WithExec([]string{"gh", "release", repoArg, "edit", tag, "--latest", "--draft=false"}).
		Sync(ctx); err != nil {
		return err
	}

	return nil
}
