package main

import (
	"context"
	"fmt"

	"github.com/frantjc/daggerverse/gh/internal/dagger"
)

type Gh struct {
	// +private
	GitHubToken *dagger.Secret
}

func New(githubToken *dagger.Secret) *Gh {
	return &Gh{GitHubToken: githubToken}
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

	release := dag.Wolfi().
		Container(dagger.WolfiContainerOpts{
			Packages: []string{"gh"},
		}).
		WithSecretVariable("GITHUB_TOKEN", m.Gh.GitHubToken).
		WithExec([]string{"gh", "release", repoArg, "create", tag, "--generate-notes", "--draft"})

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
